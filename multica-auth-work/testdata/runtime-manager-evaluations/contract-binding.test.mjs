// Binds the SPE-7 evaluation fixtures to contracts that really exist in this
// repository, resolved with ordinary filesystem reads and anchored by pinned
// digests.
//
// Git-free by owner ruling: consolidated tests must not shell out to Git. The
// integrity property is preserved by hashing instead of by object immutability.
// The canonical spec is read from disk and must hash to `canonical_spec.digest`;
// the raw-64 digest clause comes from the pinned `authority-fixture.json` and is
// re-verified against the authority document wherever that document is present.
// A missing contract file is a hard failure, never a skip, so a green run cannot
// mean "the binding was not checked".
//
// The Runtime Manager adapters this suite describes do not exist yet. This file
// therefore binds what is real and machine-checks the gap for what is not, so
// the pending lists shrink on their own as implementation lands. Generated sqlc
// storage code is excluded from the adapter scan: a generated column named
// `runtime_binding` proves the C2 schema landed, not that an adapter exists.

import assert from "node:assert/strict";
import test from "node:test";

import {
  AUTHORITY,
  authorityClause,
  frozenSpec,
  loadSuite,
  readSource,
  requirementBodies,
  sourceFilesContaining,
} from "./frozen-contracts.mjs";

const suite = loadSuite();
const spec = frozenSpec(suite);
const EVENTS_PATH = "multica-auth-work/server/pkg/protocol/events.go";

// Extraction is anchored on the declaration, not on the expected shape, so a
// malformed real event is caught instead of filtered out.
function declaredEvents(source) {
  return [...source.matchAll(/^\s*Event[A-Za-z0-9_]*\s*=\s*"([^"]*)"/gm)].map(([, value]) => value);
}

function fixtureEvents() {
  return [...new Set(suite.scenarios.flatMap(({ expected }) => expected.events ?? []))].sort();
}

test("canonical spec resolves on disk and its provenance pins are well formed", () => {
  assert.match(suite.frozen_commit, /^[0-9a-f]{40}$/, "frozen_commit must be a full lowercase object id");
  assert.ok(spec.text.length > 0, `canonical spec ${suite.canonical_spec.path} is empty`);
  assert.equal(spec.source, "filesystem", "the canonical spec must be read without Git");
});

test("fixture requirement refs exist in the canonical OpenSpec requirement set", () => {
  const declared = new Set(requirementBodies(spec.text).keys());
  assert.equal(
    declared.size,
    suite.canonical_spec.declared_requirements,
    `canonical spec declares ${declared.size} requirements, fixture pins ${suite.canonical_spec.declared_requirements}`,
  );

  const referenced = new Set(suite.scenarios.flatMap(({ requirement_refs }) => requirement_refs));
  assert.ok(referenced.size > 0, "suite must reference at least one requirement");
  const unknown = [...referenced].filter((id) => !declared.has(id)).sort();
  assert.deepEqual(unknown, [], "fixtures reference requirements the canonical spec does not declare");
});

test("canonical spec digest matches the pinned digest", () => {
  // frozenSpec throws on mismatch; assert explicitly so the contract is visible.
  assert.equal(spec.digest, suite.canonical_spec.digest, "canonical spec does not hash to canonical_spec.digest");
  assert.match(suite.canonical_spec.digest, /^[0-9a-f]{64}$/, "canonical digest must be raw lowercase 64-hex");
});

test("raw 64-hex digest form is bound to the pinned authority clause", () => {
  const { clause, verifiedAgainstDocument } = authorityClause();
  assert.match(clause, /raw lowercase 64 hexadecimal/i, "authority clause must state the raw-64 form");
  assert.match(clause, /`sha256:` prefix is forbidden/, "authority clause must forbid the sha256: prefix");
  assert.equal(AUTHORITY.digest_form.pattern, "^[0-9a-f]{64}$", "pinned digest pattern must match the clause");
  assert.equal(AUTHORITY.digest_form.forbidden_prefix, "sha256:", "pinned forbidden prefix must match the clause");

  // Reported, not asserted: the upgrade to a real document read happens only in
  // trees that carry the authority document. Both modes enforce the same clause.
  assert.equal(typeof verifiedAgainstDocument, "boolean");

  // Enforce that clause against every digest the fixture carries.
  const digestKeys = Object.keys(AUTHORITY.canonical_digest_fields);
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

test("forbidden digest and discriminator vocabulary appears nowhere in the fixture", () => {
  const serialized = JSON.stringify(suite);
  for (const forbidden of AUTHORITY.forbidden_digest_fields) {
    assert.equal(serialized.includes(`"${forbidden}"`), false, `${forbidden} is a forbidden alias and must not appear`);
  }
  for (const forbidden of AUTHORITY.canonical_document_discriminator.forbidden_fields) {
    assert.equal(serialized.includes(`"${forbidden}"`), false, `${forbidden} is forbidden everywhere`);
  }
});

test("every wire configuration document carries the canonical discriminator", () => {
  const { field, value } = AUTHORITY.canonical_document_discriminator;
  const documents = suite.scenarios
    .filter((item) => item.request && typeof item.request.configuration === "object" && item.request.configuration !== null)
    .map((item) => [item.id, item.request.configuration]);

  assert.ok(documents.length > 0, "suite must carry at least one wire configuration document to bind");
  for (const [id, document] of documents) {
    assert.equal(
      document[field],
      value,
      `${id}.request.configuration must declare the canonical discriminator ${field}="${value}"`,
    );
    for (const forbidden of AUTHORITY.canonical_document_discriminator.forbidden_fields) {
      assert.equal(document[forbidden], undefined, `${id}.request.configuration must not carry ${forbidden}`);
    }
  }
});

test("event-name contract is read from real source and fixtures obey it", () => {
  const declared = declaredEvents(readSource(EVENTS_PATH));
  assert.ok(declared.length > 0, `no event constants extracted from ${EVENTS_PATH}`);

  // Shape derived from the real registry, not assumed: subjects are
  // [a-z][a-z0-9_]*, verbs additionally allow '-' (inbox:batch-read).
  const shape = /^[a-z][a-z0-9_]*:[a-z][a-z0-9_-]*$/;
  const malformed = declared.filter((value) => !shape.test(value)).sort();
  assert.deepEqual(malformed, [], `${EVENTS_PATH} defines events outside the subject:verb contract`);

  const events = fixtureEvents();
  assert.ok(events.length > 0, "golden scenarios must emit events");
  assert.deepEqual(events.filter((value) => !shape.test(value)), [], "fixture events must follow the subject:verb contract");
});

test("pending unregistered events equal the real gap in the event registry", () => {
  const registered = new Set(declaredEvents(readSource(EVENTS_PATH)));
  const actualGap = fixtureEvents().filter((value) => !registered.has(value)).sort();

  assert.deepEqual(
    actualGap,
    [...suite.pending_contracts.unregistered_events].sort(),
    "pending_contracts.unregistered_events drifted from real source: remove registered events, add newly unbacked ones",
  );
  assert.deepEqual(
    suite.pending_contracts.unregistered_events.filter((value) => registered.has(value)).sort(),
    [],
    "these events are already registered and must leave pending_contracts",
  );
});

test("implemented symbols remain bound to hand-written adapters, excluding generated storage", () => {
  const { adapter_scan_root: root, adapter_scan_excludes: excludes } = suite.pending_contracts;
  const implemented = [...suite.pending_contracts.implemented_source_symbols].sort();
  assert.ok(implemented.length > 0, "implemented_source_symbols must bind the landed adapter surface");

  const missing = implemented
    .map((symbol) => [symbol, sourceFilesContaining(symbol, root, excludes)])
    .filter(([, hits]) => hits.length === 0)
    .map(([symbol]) => symbol);
  assert.deepEqual(missing, [], `implemented adapter symbols disappeared from hand-written source under ${root}`);
});

test("pending absent symbols equal the unresolved adapter gap, excluding generated storage", () => {
  const { adapter_scan_root: root, adapter_scan_excludes: excludes } = suite.pending_contracts;
  const claimed = [...suite.pending_contracts.absent_source_symbols].sort();

  const present = claimed
    .map((symbol) => [symbol, sourceFilesContaining(symbol, root, excludes)])
    .filter(([, hits]) => hits.length > 0)
    .map(([symbol, hits]) => `${symbol} -> ${hits.join(", ")}`)
    .sort();

  assert.deepEqual(
    present,
    [],
    `these symbols now have a hand-written adapter under ${root}: move them to implemented_source_symbols`,
  );
});

test("the generated-storage exclusion is real and not a blanket escape", () => {
  const { adapter_scan_root: root, adapter_scan_excludes: excludes } = suite.pending_contracts;
  assert.ok(excludes.length > 0, "the exclusion list must be declared");
  for (const excluded of excludes) {
    assert.ok(excluded.startsWith(`${root}/`), `${excluded} must live under the scan root`);
    assert.match(excluded, /generated/, "only generated code may be excluded from the adapter scan");
  }

  // Prove the scan still sees real code: a symbol every server tree defines must
  // be found, otherwise the exclusion or the walker silently matches nothing.
  assert.ok(
    sourceFilesContaining("package protocol", root, excludes).length > 0,
    "adapter scan found no real source; the exclusion or the walker is broken",
  );
});
