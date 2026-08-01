// SPE-7 hardening checks. Each one is bound to text that is really committed in
// the canonical spec, and every expectation is recomputed rather than restated,
// so the check keeps biting when either side moves.
//
// Covers the five reinforcement areas: exclusivity race, rollback, digest,
// pathlessness, and the exact-eight prohibition coverage of the forbidden set.
//
// Reads only committed repository files. No source-home path, credential,
// account identity, environment value or secret is imported or asserted here.

import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

async function exists(path) {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

async function repositoryRoot() {
  let candidate = here;
  for (let depth = 0; depth <= 8; depth += 1) {
    if ((await exists(join(candidate, "openspec"))) && (await exists(join(candidate, "multica-auth-work")))) {
      return candidate;
    }
    const parent = resolve(candidate, "..");
    if (parent === candidate) break;
    candidate = parent;
  }
  throw new Error("repository root not found: expected an ancestor containing openspec/ and multica-auth-work/");
}

const root = await repositoryRoot();
const suite = JSON.parse(await readFile(join(here, "evaluations.json"), "utf8"));

const specPath = join(root, suite.canonical_spec.path);
assert.ok(await exists(specPath), `missing canonical spec ${suite.canonical_spec.path}`);
const specText = await readFile(specPath, "utf8");

// Split the spec into per-requirement bodies so a check can prove the clause it
// relies on is really present instead of assuming it.
const requirementBodies = (() => {
  const parts = specText.split(/^### Requirement: (REQ-\d{2})[^\n]*$/m);
  const bodies = new Map();
  for (let index = 1; index < parts.length; index += 2) bodies.set(parts[index], parts[index + 1]);
  return bodies;
})();
assert.equal(requirementBodies.size, suite.canonical_spec.declared_requirements, "requirement split disagrees with the pinned count");

const scenarioById = new Map(suite.scenarios.map((item) => [item.id, item]));
function scenario(id) {
  const found = scenarioById.get(id);
  assert.ok(found, `missing scenario ${id}`);
  return found;
}
function requirement(id) {
  const body = requirementBodies.get(id);
  assert.ok(body, `canonical spec does not declare ${id}`);
  return body;
}

function stringValues(value, out = []) {
  if (typeof value === "string") out.push(value);
  else if (Array.isArray(value)) for (const item of value) stringValues(item, out);
  else if (value && typeof value === "object") for (const item of Object.values(value)) stringValues(item, out);
  return out;
}

test("forbidden set covers exactly eight prohibition-bearing requirements", () => {
  const prohibitionRequirements = [...requirementBodies.entries()]
    .filter(([, body]) => /MUST NOT|MUST be forbidden/.test(body))
    .map(([id]) => id);
  assert.ok(prohibitionRequirements.length >= 8, "canonical spec must declare prohibitions");

  const forbidden = suite.scenarios.filter(({ classification }) => classification === "forbidden");
  assert.equal(forbidden.length, 9, "forbidden scenario count changed; re-derive the prohibition mapping");

  // Nine scenarios collapse onto eight prohibition clauses because copy, move
  // and delete are three refusals of the single REQ-07 source-data prohibition.
  const covered = [
    ...new Set(
      forbidden.flatMap(({ requirement_refs }) => requirement_refs.filter((ref) => prohibitionRequirements.includes(ref))),
    ),
  ].sort();
  assert.equal(covered.length, 8, `forbidden cases must bind exactly eight prohibitions, got ${covered.join(",")}`);

  for (const item of forbidden) {
    const bound = item.requirement_refs.filter((ref) => prohibitionRequirements.includes(ref));
    assert.ok(bound.length > 0, `${item.id} binds no prohibition requirement`);
  }
});

test("exclusivity race refuses the second claimant without touching the first", () => {
  const req = requirement("REQ-08");
  assert.match(req, /#### Scenario: Concurrent assignment/, "REQ-08 must declare the concurrent-assignment scenario");

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
  const req = requirement("REQ-12");
  assert.match(req, /rollback/i, "REQ-12 must govern rollback");

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
  const req = requirement("REQ-12");
  assert.match(req, /#### Scenario: Digest drift/, "REQ-12 must declare the digest-drift scenario");

  const shape = /^[0-9a-f]{64}$/;
  const digestKeys = ["active_digest", "capability_digest", "effective_configuration_digest"];
  const pairs = [];
  for (const item of suite.scenarios) {
    const carriers = [item.state_before, item.expected.state_after, item.state_before?.parent_snapshot];
    for (const side of carriers) {
      if (!side) continue;
      for (const key of digestKeys) {
        if (side[key] === undefined) continue;
        assert.match(side[key], shape, `${item.id}.${key} must be raw lowercase 64-hex`);
      }
      // Where a snapshot carries both spellings they must describe one digest.
      if (side.active_digest && side.effective_configuration_digest) {
        assert.equal(
          side.effective_configuration_digest,
          side.active_digest,
          `${item.id} snapshot disagrees with itself on the effective digest`,
        );
      }
      if (side.active_digest && side.active_version_id) pairs.push([side.active_version_id, side.active_digest]);
      if (side.effective_configuration_digest && side.configuration_version_id) {
        pairs.push([side.configuration_version_id, side.effective_configuration_digest]);
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
  assert.notEqual(hot.expected.state_after.active_digest, hot.state_before.active_digest, "a new version must drift the digest");

  // The inherited parent snapshot must agree with the mapping hot-apply set, or
  // a subagent could run under a digest that never existed.
  const child = scenario("golden-subagent-inheritance");
  const snapshot = child.state_before.parent_snapshot;
  assert.equal(
    byVersion.get(snapshot.configuration_version_id),
    snapshot.effective_configuration_digest,
    "inherited snapshot digest disagrees with the version/digest mapping of the suite",
  );
});

test("no value in the suite carries a filesystem path", () => {
  const req = requirement("REQ-20");
  assert.match(req, /raw paths?/i, "REQ-20 must forbid raw paths");

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
