// Single source of truth for every contract this module binds to.
//
// Git-free by owner ruling. Every external contract is resolved with ordinary
// filesystem reads of the consolidated source tree and anchored by a pinned
// digest, never by shelling out to Git. The integrity property that matters is
// preserved: a contract this suite depends on cannot be edited without changing
// its hash, and a hash mismatch is a hard failure. Both test files import from
// here so they provably consume the same pinned contracts.
//
// Reads only repository files. No source-home path, credential, account
// identity, environment value or secret is read or exported.

import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

/**
 * Repository root, resolved without Git by walking up for the two markers that
 * identify this repository. A missing root is a hard failure, never a guess.
 */
function resolveRepoRoot(start) {
  let candidate = start;
  for (;;) {
    if (existsSync(join(candidate, "openspec")) && existsSync(join(candidate, "multica-auth-work"))) {
      return candidate;
    }
    const parent = resolve(candidate, "..");
    if (parent === candidate) {
      throw new Error(`cannot resolve repository root above ${start}: no openspec/ + multica-auth-work/ marker pair`);
    }
    candidate = parent;
  }
}

export const REPO_ROOT = resolveRepoRoot(here);

export function sha256Hex(text) {
  return createHash("sha256").update(text).digest("hex");
}

function readJson(name) {
  return JSON.parse(readFileSync(join(here, name), "utf8"));
}

/** The fixture under validation is read from disk: it is the artifact being checked. */
export function loadSuite() {
  return readJson("evaluations.json");
}

export function loadSchema() {
  return readJson("evaluation.schema.json");
}

/** Pinned Git-free authority: clause text, digest form and canonical vocabulary. */
export function loadAuthority() {
  const authority = readJson("authority-fixture.json");
  const required = [
    "fixture_version",
    "provenance",
    "raw64_digest_clause",
    "digest_form",
    "canonical_digest_fields",
    "forbidden_digest_fields",
    "canonical_document_discriminator",
    "canonical_spec",
  ];
  const missing = required.filter((key) => authority[key] === undefined);
  if (missing.length > 0) throw new Error(`authority-fixture.json is missing ${missing.join(", ")}`);
  if (!/^[0-9a-f]{40}$/.test(authority.provenance.authority_commit)) {
    throw new Error("authority provenance commit must be a full lowercase object id");
  }
  if (!/^[0-9a-f]{64}$/.test(authority.provenance.authority_document_sha256)) {
    throw new Error("authority provenance document sha256 must be raw lowercase 64-hex");
  }
  return authority;
}

export const AUTHORITY = Object.freeze(loadAuthority());

/** A repository file, read relative to the resolved root. Missing is a hard failure. */
export function readSource(relativePath) {
  const absolute = join(REPO_ROOT, relativePath);
  if (!existsSync(absolute)) throw new Error(`required contract file is absent: ${relativePath}`);
  return readFileSync(absolute, "utf8");
}

/**
 * Repository-relative paths of `.go` files under `relativeDir` containing the
 * fixed string, skipping any path under `excludes`. Empty array means absent.
 * Replaces the previous `git grep`.
 *
 * Excludes exist so generated storage code cannot be mistaken for a
 * hand-written adapter: a generated sqlc column named `runtime_binding` proves
 * the schema landed, not that an adapter exists.
 */
export function sourceFilesContaining(needle, relativeDir, excludes = []) {
  const rootDir = join(REPO_ROOT, relativeDir);
  if (!existsSync(rootDir)) throw new Error(`source pathspec is absent: ${relativeDir}`);
  const excluded = excludes.map((value) => value.replace(/\/+$/, ""));
  const isExcluded = (repoRelative) =>
    excluded.some((prefix) => repoRelative === prefix || repoRelative.startsWith(`${prefix}/`));
  const hits = [];
  const walk = (dir) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = join(dir, entry.name);
      const repoRelative = relative(REPO_ROOT, full).split(sep).join("/");
      if (isExcluded(repoRelative)) continue;
      if (entry.isDirectory()) {
        if (entry.name === "node_modules" || entry.name === ".git") continue;
        walk(full);
      } else if (entry.isFile() && entry.name.endsWith(".go")) {
        if (readFileSync(full, "utf8").includes(needle)) hits.push(repoRelative);
      }
    }
  };
  walk(rootDir);
  return hits.sort();
}

/**
 * The canonical spec, read from disk and verified against the digest the fixture
 * pins. A working-tree edit changes the hash and fails here, which is the same
 * guarantee the previous immutable-object read provided.
 */
export function frozenSpec(suite) {
  const text = readSource(suite.canonical_spec.path);
  const digest = sha256Hex(text);
  if (digest !== suite.canonical_spec.digest) {
    throw new Error(
      `canonical spec ${suite.canonical_spec.path} hashes to ${digest}, fixture pins ${suite.canonical_spec.digest}; ` +
        "re-review every scenario before repinning",
    );
  }
  return { path: suite.canonical_spec.path, text, digest, source: "filesystem" };
}

/**
 * The governing raw-64 digest clause. Pinned verbatim in authority-fixture.json
 * because the authority document is absent from some worktrees. When the
 * document IS present under the resolved root, its hash is verified and the
 * clause is relocated inside it, upgrading the pin to a real read.
 */
export function authorityClause() {
  const { raw64_digest_clause: pinned, provenance } = AUTHORITY;
  const absolute = join(REPO_ROOT, provenance.authority_path);
  if (!existsSync(absolute)) {
    return { clause: pinned, verifiedAgainstDocument: false, reason: "authority document absent from this tree" };
  }
  const document = readFileSync(absolute, "utf8");
  const digest = sha256Hex(document);
  if (digest !== provenance.authority_document_sha256) {
    throw new Error(
      `authority document ${provenance.authority_path} hashes to ${digest}, fixture pins ` +
        `${provenance.authority_document_sha256}; the pinned clause can no longer be trusted`,
    );
  }
  if (!document.includes(pinned)) {
    throw new Error(`authority document ${provenance.authority_path} no longer contains the pinned clause verbatim`);
  }
  return { clause: pinned, verifiedAgainstDocument: true, reason: "verified against on-disk authority document" };
}

/** Per-requirement bodies of the canonical spec, so a check can prove its clause exists. */
export function requirementBodies(specText) {
  const parts = specText.split(/^### Requirement: (REQ-\d{2})[^\n]*$/m);
  const bodies = new Map();
  for (let index = 1; index < parts.length; index += 2) bodies.set(parts[index], parts[index + 1]);
  return bodies;
}

/** Literal prohibition clause lines inside a requirement body. */
export function prohibitionClauses(body) {
  return body
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => /MUST NOT|MUST be forbidden/.test(line));
}
