// SPE-7 hardening checks. Each is bound to clause text that is really committed
// in the frozen canonical spec, read as an immutable Git object through the same
// accessor `contract-binding.test.mjs` uses, so both files provably consume the
// same frozen blob rather than two independently resolved copies.
//
// Covers exclusivity race, rollback, digest, pathlessness, and the prohibition
// coverage of the forbidden set.
//
// The SPE-18 capacity-envelope terminology (3 Kiro + 5 Codex) is reserved to
// that lane, where its frozen-ceiling reading is BLOCKED, and is not reused
// here: it has no bearing on prohibition coverage. No count invariant is
// asserted over prohibition requirements either. The auditable property is that
// every forbidden case binds at least one explicit prohibition clause, not that
// the clause set totals any particular number.
//
// Reads only committed repository objects. No source-home path, credential,
// account identity, environment value or secret is read or asserted.

import assert from "node:assert/strict";
import test from "node:test";

import { frozenSpec, loadSuite, prohibitionClauses, requirementBodies } from "./frozen-contracts.mjs";

const suite = loadSuite();
const spec = frozenSpec(suite);
const bodies = requirementBodies(spec.text);

assert.equal(
  bodies.size,
  suite.canonical_spec.declared_requirements,
  "requirement split of the frozen blob disagrees with the pinned count",
);

const FORBIDDEN_OPERATIONS = Object.freeze([
  "expose_raw_path",
  "share_account_home",
  "copy_source_credentials",
  "move_source_credentials",
  "delete_source_credentials",
  "configure_static_slot_allowlist",
  "fallback_across_transport_binding",
  "create_custom_infrastructure",
  "perform_sharepoint_work",
]);

const scenarioById = new Map(suite.scenarios.map((item) => [item.id, item]));
function scenario(id) {
  const found = scenarioById.get(id);
  assert.ok(found, `missing scenario ${id}`);
  return found;
}
function requirement(id) {
  const body = bodies.get(id);
  assert.ok(body, `frozen spec does not declare ${id}`);
  return body;
}
function stringValues(value, out = []) {
  if (typeof value === "string") out.push(value);
  else if (Array.isArray(value)) for (const item of value) stringValues(item, out);
  else if (value && typeof value === "object") for (const item of Object.values(value)) stringValues(item, out);
  return out;
}

test("forbidden set covers exactly the nine declared forbidden operations", () => {
  const forbidden = suite.scenarios.filter(({ classification }) => classification === "forbidden");
  assert.deepEqual(
    forbidden.map(({ operation }) => operation),
    [...FORBIDDEN_OPERATIONS],
    "forbidden operation coverage must stay exact and ordered",
  );
  assert.equal(new Set(forbidden.map(({ operation }) => operation)).size, FORBIDDEN_OPERATIONS.length, "operations must be unique");
});

test("every forbidden case binds an explicit prohibition clause of the frozen spec", () => {
  for (const item of suite.scenarios.filter(({ classification }) => classification === "forbidden")) {
    const clauses = item.requirement_refs.flatMap((ref) => prohibitionClauses(requirement(ref)).map((line) => [ref, line]));
    assert.ok(
      clauses.length > 0,
      `${item.id} references ${item.requirement_refs.join(",")}, none of which states a MUST NOT or MUST-be-forbidden clause`,
    );
    for (const [ref, line] of clauses) {
      assert.match(line, /MUST NOT|MUST be forbidden/, `${item.id} clause from ${ref} is not a prohibition`);
    }
  }
});

test("exclusivity race refuses the second claimant without touching the first", () => {
  assert.match(requirement("REQ-08"), /#### Scenario: Concurrent assignment/, "REQ-08 must declare the concurrent-assignment scenario");

  const race = scenario("forbidden-account-sharing");
  assert.ok(race.requirement_refs.includes("REQ-08"), "race case must bind REQ-08");
  assert.equal(race.expected.error_code, "exclusive_assignment_conflict");
  assert.equal(race.expected.mutation_count, 0);
  assert.deepEqual(race.expected.side_effects, []);
  // Contention is only real when a different binding claims the same home.
  assert.equal(race.request.home_ref, race.state_before.home_ref, "race must contend for the same home");
  assert.notEqual(
    race.request.target_binding_id,
    race.state_before.active_binding_id,
    "race must come from a binding other than the holder",
  );

  const attach = scenario("golden-exclusive-subscription-attach");
  assert.equal(attach.expected.state_after.active_assignments_for_home, 1, "the accept path must stay exclusive");
  assert.equal(attach.state_before.home_assigned, false, "attach may only take an unassigned home");
});

test("rollback activates prior immutable history under compare-and-swap", () => {
  assert.match(requirement("REQ-12"), /rollback/i, "REQ-12 must govern rollback");

  const rollback = scenario("golden-rollback");
  assert.equal(
    rollback.request.expected_active_version_id,
    rollback.state_before.active_version_id,
    "rollback must compare-and-swap on the observed active version",
  );
  assert.ok(rollback.state_before.history.includes(rollback.request.target_version_id), "target must already exist in history");
  assert.equal(rollback.expected.state_after.active_version_id, rollback.request.target_version_id);
  assert.deepEqual(rollback.expected.state_after.history, rollback.state_before.history);
  assert.equal(rollback.expected.state_after.history_mutated, false);
  assert.equal(
    rollback.expected.state_after.running_task_version_id,
    rollback.state_before.running_task_version_id,
    "a running task snapshot must survive rollback",
  );
  assert.equal(rollback.expected.state_after.next_claim_version_id, rollback.request.target_version_id);
  assert.ok(rollback.expected.created_resource_types.includes("activation_audit"), "rollback must be auditable");
  assert.equal(
    rollback.expected.created_resource_types.includes("runtime_configuration_version"),
    false,
    "rollback activates a prior version and must not mint a new one",
  );
  assert.ok(rollback.expected.events.includes("runtime_configuration:rolled_back"));
});

test("digests are raw 64-hex and map one-to-one with configuration versions", () => {
  assert.match(requirement("REQ-12"), /#### Scenario: Digest drift/, "REQ-12 must declare the digest-drift scenario");

  const shape = /^[0-9a-f]{64}$/;
  const digestKeys = ["capability_digest", "configuration_digest"];
  const pairs = [];
  for (const item of suite.scenarios) {
    for (const side of [item.state_before, item.expected.state_after, item.state_before?.parent_snapshot]) {
      if (!side) continue;
      for (const key of digestKeys) {
        if (side[key] === undefined) continue;
        assert.match(side[key], shape, `${item.id}.${key} must be raw lowercase 64-hex`);
      }
      // configuration_digest is the single canonical name for the immutable
      // configuration/version digest, so it pairs with whichever version id the
      // object carries: the active version, or a snapshot's pinned version.
      if (side.configuration_digest && side.active_version_id) {
        pairs.push([side.active_version_id, side.configuration_digest]);
      }
      if (side.configuration_digest && side.configuration_version_id) {
        pairs.push([side.configuration_version_id, side.configuration_digest]);
      }
    }
  }
  assert.ok(pairs.length >= 3, "suite must exercise at least three version/digest pairs");

  const byVersion = new Map();
  const byDigest = new Map();
  for (const [version, digest] of pairs) {
    if (byVersion.has(version)) assert.equal(byVersion.get(version), digest, `version ${version} carries two digests`);
    byVersion.set(version, digest);
    if (byDigest.has(digest)) assert.equal(byDigest.get(digest), version, `digest ${digest} covers two versions`);
    byDigest.set(digest, version);
  }

  const hot = scenario("golden-model-reasoning-hot-apply");
  assert.notEqual(hot.expected.state_after.active_version_id, hot.state_before.active_version_id, "hot-apply must move the version");
  assert.notEqual(hot.expected.state_after.configuration_digest, hot.state_before.configuration_digest, "a new version must drift the digest");

  // The inherited parent snapshot must agree with the mapping hot-apply set, or
  // a subagent could run under a digest that never existed.
  const snapshot = scenario("golden-subagent-inheritance").state_before.parent_snapshot;
  assert.equal(
    byVersion.get(snapshot.configuration_version_id),
    snapshot.configuration_digest,
    "inherited snapshot digest disagrees with the version/digest mapping of the suite",
  );
});

test("no value in the suite carries a filesystem path", () => {
  assert.match(requirement("REQ-20"), /raw paths?/i, "REQ-20 must forbid raw paths");

  const values = stringValues(suite);
  assert.ok(values.length > 0);
  const pathLike = [
    /^\//, // absolute posix
    /^~/, // home-relative
    /^[A-Za-z]:[\\/]/, // windows drive
    /\\\\/, // UNC
    /\/(home|root|Users|var|opt|tmp|etc|srv|proc)(\/|$)/, // embedded system root
  ];
  const offenders = values.filter((value) => value !== suite.canonical_spec.path && pathLike.some((rule) => rule.test(value)));
  assert.deepEqual(offenders, [], "fixture values must stay pathless; only opaque references may appear");

  for (const item of suite.scenarios) {
    for (const side of [item.state_before, item.request, item.expected.state_after]) {
      if (!side) continue;
      if (side.home_ref !== undefined) assert.match(side.home_ref, /^home_opaque_\d{2}$/, `${item.id} home_ref must be opaque`);
      if (side.subscription_ref !== undefined) {
        assert.match(side.subscription_ref, /^subscription_opaque_\d{2}$/, `${item.id} subscription_ref must be opaque`);
      }
    }
  }
});
