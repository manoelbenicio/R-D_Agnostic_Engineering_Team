import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const [schema, suite] = await Promise.all([
  readJson("evaluation.schema.json"),
  readJson("evaluations.json"),
]);

async function readJson(name) {
  return JSON.parse(await readFile(join(here, name), "utf8"));
}

function resolveRef(root, ref) {
  assert.match(ref, /^#\//, `only local schema refs are supported: ${ref}`);
  return ref
    .slice(2)
    .split("/")
    .reduce((value, part) => value[part.replaceAll("~1", "/").replaceAll("~0", "~")], root);
}

function validate(value, rule, root = schema, location = "$") {
  const errors = [];
  if (rule.$ref) return validate(value, resolveRef(root, rule.$ref), root, location);

  if (rule.oneOf) {
    const branches = rule.oneOf.map((branch) => validate(value, branch, root, location));
    const passing = branches.filter((branch) => branch.length === 0);
    if (passing.length !== 1) errors.push(`${location}: expected exactly one oneOf branch, got ${passing.length}`);
    return errors;
  }

  if (Object.hasOwn(rule, "const") && !Object.is(value, rule.const)) {
    errors.push(`${location}: expected constant ${JSON.stringify(rule.const)}`);
  }
  if (rule.enum && !rule.enum.some((candidate) => Object.is(value, candidate))) {
    errors.push(`${location}: value is not in enum`);
  }

  const actualType = value === null ? "null" : Array.isArray(value) ? "array" : typeof value;
  if (rule.type && actualType !== rule.type) {
    errors.push(`${location}: expected ${rule.type}, got ${actualType}`);
    return errors;
  }

  if (actualType === "string") {
    if (rule.minLength !== undefined && value.length < rule.minLength) errors.push(`${location}: shorter than minLength`);
    if (rule.pattern && !new RegExp(rule.pattern).test(value)) errors.push(`${location}: does not match ${rule.pattern}`);
  }

  if (actualType === "array") {
    if (rule.minItems !== undefined && value.length < rule.minItems) errors.push(`${location}: fewer than minItems`);
    if (rule.maxItems !== undefined && value.length > rule.maxItems) errors.push(`${location}: more than maxItems`);
    if (rule.uniqueItems) {
      const serialized = value.map((item) => JSON.stringify(item));
      if (new Set(serialized).size !== serialized.length) errors.push(`${location}: duplicate array items`);
    }
    if (rule.items) value.forEach((item, index) => errors.push(...validate(item, rule.items, root, `${location}[${index}]`)));
  }

  if (actualType === "object") {
    const keys = Object.keys(value);
    if (rule.minProperties !== undefined && keys.length < rule.minProperties) errors.push(`${location}: fewer than minProperties`);
    for (const required of rule.required ?? []) {
      if (!Object.hasOwn(value, required)) errors.push(`${location}: missing required property ${required}`);
    }
    for (const [key, child] of Object.entries(value)) {
      if (rule.properties?.[key]) {
        errors.push(...validate(child, rule.properties[key], root, `${location}.${key}`));
      } else if (rule.additionalProperties === false) {
        errors.push(`${location}: unexpected property ${key}`);
      } else if (rule.additionalProperties && typeof rule.additionalProperties === "object") {
        errors.push(...validate(child, rule.additionalProperties, root, `${location}.${key}`));
      }
    }
  }

  return errors;
}

const byOperation = new Map(suite.scenarios.map((scenario) => [scenario.operation, scenario]));
const goldenOperations = [
  "reuse_runtime_session",
  "attach_exclusive_subscription",
  "hot_apply_model_reasoning",
  "rollback_runtime_configuration",
  "spawn_subagent",
];
const forbiddenOperations = [
  "expose_raw_path",
  "share_account_home",
  "copy_source_credentials",
  "move_source_credentials",
  "delete_source_credentials",
  "configure_static_slot_allowlist",
  "fallback_across_transport_binding",
  "create_custom_infrastructure",
  "perform_sharepoint_work",
];

function scenario(operation) {
  const found = byOperation.get(operation);
  assert.ok(found, `missing scenario for ${operation}`);
  return found;
}

function assertPreserved(scenarioValue, ...keys) {
  for (const key of keys) {
    assert.equal(
      scenarioValue.expected.preserved_ids[key],
      scenarioValue.state_before[key] ?? scenarioValue.request[key] ?? scenarioValue.state_before.parent_snapshot?.[key],
      `${scenarioValue.id} must preserve ${key}`,
    );
  }
}

test("evaluation suite conforms to its JSON Schema", () => {
  assert.deepEqual(validate(suite, schema), []);
});

test("suite has exact deterministic golden and forbidden coverage", () => {
  assert.equal(byOperation.size, suite.scenarios.length, "operations must be unique");
  assert.deepEqual(
    suite.scenarios.filter(({ classification }) => classification === "golden").map(({ operation }) => operation),
    goldenOperations,
  );
  assert.deepEqual(
    suite.scenarios.filter(({ classification }) => classification === "forbidden").map(({ operation }) => operation),
    forbiddenOperations,
  );
  for (const item of suite.scenarios) {
    assert.ok(item.id.startsWith(`${item.classification}-`), `${item.id} classification prefix mismatch`);
  }
});

test("session reuse and exclusive attach preserve existing identities", () => {
  const reuse = scenario("reuse_runtime_session");
  assertPreserved(reuse, "session_id", "agent_id", "runtime_id", "daemon_id");
  assert.deepEqual(reuse.expected.created_resource_types, ["runtime_binding"]);
  assert.equal(reuse.expected.state_after.home_assignment, null);
  assert.equal(reuse.expected.state_after.agent_recreated, false);
  assert.equal(reuse.expected.state_after.runtime_recreated, false);
  assert.equal(reuse.expected.state_after.daemon_recreated, false);

  const attach = scenario("attach_exclusive_subscription");
  assert.equal(attach.expected.preserved_ids.binding_id, attach.state_before.binding_id);
  assert.equal(attach.expected.state_after.home_ref, attach.request.home_ref);
  assert.equal(attach.expected.state_after.active_assignments_for_home, 1);
  assert.equal(attach.expected.state_after.binding_generation, attach.state_before.binding_generation + 1);
  for (const key of ["agent_recreated", "runtime_recreated", "session_recreated", "daemon_recreated"]) {
    assert.equal(attach.expected.state_after[key], false, `${key} must remain false`);
  }
});

test("model and reasoning hot-apply is acknowledged without rewriting pinned tasks", () => {
  const hot = scenario("hot_apply_model_reasoning");
  assert.equal(hot.request.expected_active_version_id, hot.state_before.active_version_id);
  assert.deepEqual(Object.values(hot.state_before.registered_apply_classes), ["hot", "hot", "hot"]);
  assert.equal(hot.expected.state_after.apply_class, "hot");
  assert.equal(hot.expected.state_after.daemon_acknowledged, true);
  assert.equal(hot.expected.state_after.process_restarted, false);
  assert.equal(hot.expected.state_after.running_task_version_id, hot.state_before.running_task_version_id);
  assert.notEqual(hot.expected.state_after.next_claim_version_id, hot.state_before.running_task_version_id);
});

test("rollback activates prior immutable history and leaves running snapshots pinned", () => {
  const rollback = scenario("rollback_runtime_configuration");
  assert.equal(rollback.request.expected_active_version_id, rollback.state_before.active_version_id);
  assert.ok(rollback.state_before.history.includes(rollback.request.target_version_id));
  assert.equal(rollback.expected.state_after.active_version_id, rollback.request.target_version_id);
  assert.deepEqual(rollback.expected.state_after.history, rollback.state_before.history);
  assert.equal(rollback.expected.state_after.history_mutated, false);
  assert.equal(rollback.expected.state_after.running_task_version_id, rollback.state_before.running_task_version_id);
});

test("subagent inherits snapshot and binding without persistent row creation", () => {
  const child = scenario("spawn_subagent");
  assert.deepEqual(child.request.task_overrides, {});
  assertPreserved(child, "session_id", "runtime_id", "agent_id", "binding_id", "home_ref");
  assert.equal(child.expected.state_after.snapshot_equal_to_parent, true);
  assert.equal(child.expected.state_after.persistent_runtime_rows_created, 0);
  assert.equal(child.expected.state_after.persistent_session_rows_created, 0);
  assert.equal(child.expected.state_after.persistent_home_assignments_created, 0);
  assert.equal(child.expected.state_after.active_subagents, child.state_before.active_subagents + 1);
  assert.ok(child.expected.state_after.active_subagents <= child.state_before.subagent_limit);
});

test("all forbidden actions fail closed with no mutation or side effect", () => {
  for (const operation of forbiddenOperations) {
    const blocked = scenario(operation);
    assert.equal(blocked.expected.decision, "reject", operation);
    assert.equal(blocked.expected.fail_closed, true, operation);
    assert.equal(blocked.expected.mutation_count, 0, operation);
    assert.deepEqual(blocked.expected.side_effects, [], operation);
  }
});

test("fixtures contain no literal source-home path or credential payload", () => {
  const serialized = JSON.stringify(suite);
  assert.doesNotMatch(serialized, /(?:^|["'])\/(?:home|root|Users|var|opt|tmp)\//);
  assert.doesNotMatch(serialized, /-----BEGIN [A-Z ]+PRIVATE KEY-----/);
  assert.doesNotMatch(serialized, /(?:api[_-]?key|access[_-]?token|refresh[_-]?token|password)\s*[=:]\s*[^,}\s]+/i);
});
