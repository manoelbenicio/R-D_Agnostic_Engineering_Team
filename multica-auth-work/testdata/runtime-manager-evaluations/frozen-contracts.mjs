// Single source of truth for every contract this module binds to.
//
// All external contracts are read as immutable Git objects at pinned revisions,
// never from the mutable working tree or another checkout. A working-tree read
// would let an uncommitted edit silently satisfy a gate; a Git-object read
// cannot. Both test files import from here so they provably consume the same
// frozen blob rather than two independently resolved copies.
//
// Reads only committed repository objects. No source-home path, credential,
// account identity, environment value or secret is read or exported.

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

export const REPO_ROOT = execFileSync("git", ["rev-parse", "--show-toplevel"], {
  cwd: here,
  encoding: "utf8",
}).trim();

// The authority baseline is pinned by commit and path. It is read as a Git
// object precisely so the producer branch never has to merge it: K1 handles
// ancestry later, and this gate stays honest in the meantime.
export const AUTHORITY = Object.freeze({
  commit: "c35c200be62d63cb654638d5b142b2a8105c1484",
  path: ".deploy-control/p0/evidence/spe4-spe5-unified-authority-baseline.md",
});

function git(args) {
  return execFileSync("git", args, {
    cwd: REPO_ROOT,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
    stdio: ["ignore", "pipe", "pipe"],
  });
}

/** Blob content at an exact revision. Throws when the object is missing. */
export function showBlob(rev, path) {
  try {
    return git(["show", `${rev}:${path}`]);
  } catch (cause) {
    throw new Error(`missing Git object ${rev}:${path}`, { cause });
  }
}

/** Resolve a revision to its full object id, proving it exists. */
export function resolveCommit(rev) {
  try {
    return git(["rev-parse", "--verify", `${rev}^{commit}`]).trim();
  } catch (cause) {
    throw new Error(`unresolvable revision ${rev}`, { cause });
  }
}

/**
 * Files under pathspec that contain the fixed string, searched in the committed
 * tree of `rev`. Empty array means absent. git grep exits 1 on no match, which
 * is not an error here.
 */
export function grepFilesAtRev(rev, needle, pathspec) {
  try {
    return git(["grep", "-lF", "-e", needle, rev, "--", pathspec])
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
  } catch (error) {
    if (error.status === 1) return [];
    throw error;
  }
}

export function sha256Hex(text) {
  return createHash("sha256").update(text).digest("hex");
}

/** The fixture under validation is deliberately read from disk: it is the artifact being checked. */
export function loadSuite() {
  return JSON.parse(readFileSync(join(here, "evaluations.json"), "utf8"));
}

export function loadSchema() {
  return JSON.parse(readFileSync(join(here, "evaluation.schema.json"), "utf8"));
}

/** The frozen canonical spec blob, addressed by the fixture's own pins. */
export function frozenSpec(suite) {
  const commit = resolveCommit(suite.frozen_commit);
  const text = showBlob(suite.frozen_commit, suite.canonical_spec.path);
  return { commit, text, digest: sha256Hex(text) };
}

/** Per-requirement bodies of the frozen spec, so a check can prove its clause exists. */
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

/** The exact commit whose tree the source-gap checks inspect. */
export const COMMITTED_HEAD = resolveCommit("HEAD");
