// Binds the SPE-7 evaluation fixtures to contracts that really exist in this
// repository, read as immutable Git objects at pinned revisions.
//
// Nothing here is mocked, stubbed or inferred, and nothing is read from the
// mutable working tree or another checkout: the canonical spec comes from
// `suite.frozen_commit:suite.canonical_spec.path`, the digest contract from a
// pinned authority commit, and the source/event gaps from the tree of the exact
// committed HEAD. A missing object is a hard failure, never a skip, so a green
// run cannot mean "the binding was not checked".
//
// The Runtime Manager adapters this suite describes do not exist yet. This file
// therefore binds what is real and machine-checks the gap for what is not, so
// the pending lists shrink on their own as implementation lands.

import assert from "node:assert/strict";
import test from "node:test";

import {
  AUTHORITY,
  COMMITTED_HEAD,
  frozenSpec,
  grepFilesAtRev,
  loadSuite,
  requirementBodies,
  resolveCommit,
  sha256Hex,
  showBlob,
} from "./frozen-contracts.mjs";

const suite = loadSuite();
const spec = frozenSpec(suite);
const EVENTS_PATH = "multica-auth-work/server/pkg/protocol/events.go";
const SERVER_PATHSPEC = "multica-auth-work/server";

// Extraction is anchored on the declaration, not on the expected shape, so a
// malformed real event is caught instead of filtered out.
function declaredEvents(source) {
  return [...source.matchAll(/^\s*Event[A-Za-z0-9_]*\s*=\s*"([^"]*)"/gm)].map(([, value]) => value);
}

function fixtureEvents() {
  return [...new Set(suite.scenarios.flatMap(({ expected }) => expected.events ?? []))].sort();
}

test("frozen commit and canonical spec resolve as Git objects", () => {
  assert.equal(spec.commit, resolveCommit(suite.frozen_commit), "frozen_commit must resolve to itself");
  assert.match(suite.frozen_commit, /^[0-9a-f]{40}$/, "frozen_commit must be a full lowercase object id");
  assert.ok(spec.text.length > 0, `frozen spec blob ${suite.frozen_commit}:${suite.canonical_spec.path} is empty`);
});

test("fixture requirement refs exist in the frozen OpenSpec requirement set", () => {
  const declared = new Set(requirementBodies(spec.text).keys());
  assert.equal(
    declared.size,
    suite.canonical_spec.declared_requirements,
    `frozen spec declares ${declared.size} requirements, fixture pins ${suite.canonical_spec.declared_requirements}`,
  );

  const referenced = new Set(suite.scenarios.flatMap(({ requirement_refs }) => requirement_refs));
  assert.ok(referenced.size > 0, "suite must reference at least one requirement");
  const unknown = [...referenced].filter((id) => !declared.has(id)).sort();
  assert.deepEqual(unknown, [], "fixtures reference requirements the frozen spec does not declare");
});

test("frozen spec blob digest matches the pinned canonical digest", () => {
  assert.equal(
    spec.digest,
    suite.canonical_spec.digest,
    "the frozen spec blob does not hash to canonical_spec.digest; repin only after re-reviewing every scenario",
  );
  assert.match(suite.canonical_spec.digest, /^[0-9a-f]{64}$/, "canonical digest must be raw lowercase 64-hex");
});

test("raw 64-hex digest form is bound to the pinned authority clause", () => {
  const authorityCommit = resolveCommit(AUTHORITY.commit);
  assert.equal(authorityCommit, AUTHORITY.commit, "authority commit must resolve to its pinned id");

  const baseline = showBlob(AUTHORITY.commit, AUTHORITY.path);
  const clause = baseline
    .split("\n")
    .find((line) => /effective configuration digest/i.test(line) && /raw lowercase 64 hexadecimal/i.test(line));
  assert.ok(clause, `${AUTHORITY.commit}:${AUTHORITY.path} no longer states the raw-64 digest clause`);
  assert.match(clause, /`sha256:` prefix is forbidden/, "authority clause must forbid the sha256: prefix");

  // Enforce that clause against every digest the fixture carries.
  const digestKeys = ["active_digest", "capability_digest", "effective_configuration_digest"];
  const observed = [];
  for (const item of suite.scenarios) {
    for (const side of [item.state_before, item.expected.state_after, item.state_before?.parent_snapshot]) {
      if (!side) continue;
      for (const key of digestKeys) {
        if (side[key] !== undefined) observed.push([`${item.id}.${key}`, side[key]]);
      }
    }
  }
  observed.push(["canonical_spec.digest", suite.canonical_spec.digest]);
  assert.ok(observed.length > 1, "suite must carry digests to bind");
  for (const [where, value] of observed) {
    assert.match(value, /^[0-9a-f]{64}$/, `${where} violates the pinned raw-64 clause`);
  }
});

test("event-name contract is read from the committed HEAD tree and fixtures obey it", () => {
  const source = showBlob(COMMITTED_HEAD, EVENTS_PATH);
  const declared = declaredEvents(source);
  assert.ok(declared.length > 0, `no event constants extracted from ${COMMITTED_HEAD}:${EVENTS_PATH}`);

  // Shape derived from the real registry, not assumed: subjects are
  // [a-z][a-z0-9_]*, verbs additionally allow '-' (inbox:batch-read).
  const shape = /^[a-z][a-z0-9_]*:[a-z][a-z0-9_-]*$/;
  const malformed = declared.filter((value) => !shape.test(value)).sort();
  assert.deepEqual(malformed, [], `${EVENTS_PATH} defines events outside the subject:verb contract`);

  const events = fixtureEvents();
  assert.ok(events.length > 0, "golden scenarios must emit events");
  assert.deepEqual(events.filter((value) => !shape.test(value)), [], "fixture events must follow the subject:verb contract");
});

test("pending unregistered events equal the gap in the committed HEAD tree", () => {
  const registered = new Set(declaredEvents(showBlob(COMMITTED_HEAD, EVENTS_PATH)));
  const actualGap = fixtureEvents().filter((value) => !registered.has(value)).sort();

  assert.deepEqual(
    actualGap,
    [...suite.pending_contracts.unregistered_events].sort(),
    "pending_contracts.unregistered_events drifted from the committed tree: remove registered events, add newly unbacked ones",
  );
  assert.deepEqual(
    suite.pending_contracts.unregistered_events.filter((value) => registered.has(value)).sort(),
    [],
    "these events are already registered and must leave pending_contracts",
  );
});

test("pending absent symbols equal the gap in the committed HEAD tree", () => {
  const claimed = [...suite.pending_contracts.absent_source_symbols].sort();
  assert.ok(claimed.length > 0, "pending_contracts must declare the symbols it claims are absent");

  const present = claimed.filter((symbol) => grepFilesAtRev(COMMITTED_HEAD, symbol, SERVER_PATHSPEC).length > 0).sort();
  assert.deepEqual(
    present,
    [],
    `these symbols now exist in ${COMMITTED_HEAD}:${SERVER_PATHSPEC}: bind those scenarios to the real adapter and drop the entries`,
  );
});
