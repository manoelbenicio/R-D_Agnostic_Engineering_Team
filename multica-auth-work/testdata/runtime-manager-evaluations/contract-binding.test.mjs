// Binds the SPE-7 evaluation fixtures to contracts that really exist in this
// repository. Nothing here is mocked, stubbed or inferred: every expectation is
// recomputed from a real file on disk, and a missing input is a hard failure
// rather than a skip, so a green run cannot mean "the binding was not checked".
//
// Three real contracts are bound:
//   1. the frozen OpenSpec requirement set (requirement_refs must exist);
//   2. the product event-name contract in pkg/protocol/events.go (lexical form);
//   3. the absence of runtime-manager symbols in the Go source, which is what
//      makes the rest of the suite fixture-only today.
//
// The Runtime Manager adapters this suite describes do not exist yet. This file
// therefore binds what is real and *machine-checks the gap* for what is not, so
// the pending lists shrink on their own as implementation lands.

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFile, readdir, stat } from "node:fs/promises";
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

// Deterministic root discovery: the first ancestor holding both the OpenSpec
// tree and the product workspace. No environment variable, no cwd dependency.
async function repositoryRoot() {
  let candidate = here;
  for (let depth = 0; depth <= 8; depth += 1) {
    const hasOpenSpec = await exists(join(candidate, "openspec"));
    const hasWorkspace = await exists(join(candidate, "multica-auth-work"));
    if (hasOpenSpec && hasWorkspace) return candidate;
    const parent = resolve(candidate, "..");
    if (parent === candidate) break;
    candidate = parent;
  }
  throw new Error("repository root not found: expected an ancestor containing openspec/ and multica-auth-work/");
}

const root = await repositoryRoot();
const suite = JSON.parse(await readFile(join(here, "evaluations.json"), "utf8"));

async function readRequired(relativePath, why) {
  const absolute = join(root, relativePath);
  assert.ok(await exists(absolute), `${why}: missing real contract file ${relativePath}`);
  return readFile(absolute, "utf8");
}

async function goSourceFiles(relativeDir) {
  const absolute = join(root, relativeDir);
  const skip = new Set(["vendor", "node_modules", "testdata", ".git"]);
  const found = [];
  async function walk(dir) {
    for (const entry of await readdir(dir, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        if (skip.has(entry.name)) continue;
        await walk(join(dir, entry.name));
      } else if (entry.isFile() && entry.name.endsWith(".go")) {
        found.push(join(dir, entry.name));
      }
    }
  }
  assert.ok(await exists(absolute), `missing real source tree ${relativeDir}`);
  await walk(absolute);
  assert.ok(found.length > 0, `no Go sources found under ${relativeDir}`);
  return found;
}

test("fixture requirement refs exist in the real frozen OpenSpec requirement set", async () => {
  const specPath = suite.canonical_spec.path;
  const spec = await readRequired(specPath, "requirement binding");

  const declared = new Set(
    [...spec.matchAll(/^### Requirement: (REQ-\d{2})\b/gm)].map(([, id]) => id),
  );
  assert.equal(
    declared.size,
    suite.canonical_spec.declared_requirements,
    `${specPath} declares ${declared.size} requirements, fixture pins ${suite.canonical_spec.declared_requirements}`,
  );

  const referenced = new Set(suite.scenarios.flatMap(({ requirement_refs }) => requirement_refs));
  assert.ok(referenced.size > 0, "suite must reference at least one requirement");
  const unknown = [...referenced].filter((id) => !declared.has(id)).sort();
  assert.deepEqual(unknown, [], "fixtures reference requirements that the canonical spec does not declare");
});

test("frozen spec digest still matches the pinned canonical digest", async () => {
  const spec = await readRequired(suite.canonical_spec.path, "spec freeze binding");
  const actual = createHash("sha256").update(spec).digest("hex");
  assert.equal(
    actual,
    suite.canonical_spec.digest,
    "the canonical spec changed; re-review every scenario, then repin canonical_spec.digest",
  );
});

test("product event-name contract is subject:verb and fixtures obey it", async () => {
  const eventsPath = "multica-auth-work/server/pkg/protocol/events.go";
  const source = await readRequired(eventsPath, "event-name binding");

  // Extraction is anchored on the declaration, not on the expected shape, so a
  // malformed real event is caught instead of filtered out.
  const declared = [...source.matchAll(/^\s*Event[A-Za-z0-9_]*\s*=\s*"([^"]*)"/gm)].map(([, value]) => value);
  assert.ok(declared.length > 0, `no event constants extracted from ${eventsPath}`);

  // Shape derived from the real registry, not assumed: subjects are
  // [a-z][a-z0-9_]*, verbs additionally allow '-' (inbox:batch-read,
  // inbox:batch-archived). Widen this only against real declarations.
  const shape = /^[a-z][a-z0-9_]*:[a-z][a-z0-9_-]*$/;
  const malformed = declared.filter((value) => !shape.test(value)).sort();
  assert.deepEqual(malformed, [], `${eventsPath} defines events outside the subject:verb contract`);

  const fixtureEvents = [
    ...new Set(suite.scenarios.flatMap(({ expected }) => expected.events ?? [])),
  ].sort();
  assert.ok(fixtureEvents.length > 0, "golden scenarios must emit events");
  const offending = fixtureEvents.filter((value) => !shape.test(value));
  assert.deepEqual(offending, [], "fixture events must follow the product subject:verb contract");
});

test("pending unregistered events equal the real gap against the event registry", async () => {
  const source = await readRequired(
    "multica-auth-work/server/pkg/protocol/events.go",
    "event-registry binding",
  );
  const registered = new Set(
    [...source.matchAll(/^\s*Event[A-Za-z0-9_]*\s*=\s*"([^"]*)"/gm)].map(([, value]) => value),
  );

  const fixtureEvents = [
    ...new Set(suite.scenarios.flatMap(({ expected }) => expected.events ?? [])),
  ];
  const actualGap = fixtureEvents.filter((value) => !registered.has(value)).sort();

  assert.deepEqual(
    actualGap,
    [...suite.pending_contracts.unregistered_events].sort(),
    "pending_contracts.unregistered_events drifted: remove events now present in the registry, add newly unbacked ones",
  );

  const wronglyPending = suite.pending_contracts.unregistered_events.filter((value) => registered.has(value)).sort();
  assert.deepEqual(wronglyPending, [], "these events are already registered and must leave pending_contracts");
});

test("pending absent symbols equal the real gap against the Go source tree", async () => {
  const files = await goSourceFiles("multica-auth-work/server");
  const sources = await Promise.all(files.map((file) => readFile(file, "utf8")));
  const corpus = sources.join("\n");

  const claimed = [...suite.pending_contracts.absent_source_symbols].sort();
  const stillAbsent = claimed.filter((symbol) => !corpus.includes(symbol)).sort();

  assert.deepEqual(
    stillAbsent,
    claimed,
    "a symbol listed as absent now exists in the Go source: bind those scenarios to the real adapter and drop the entry",
  );
});
