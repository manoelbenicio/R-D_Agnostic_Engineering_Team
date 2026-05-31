#!/usr/bin/env node
/**
 * Preflight dev-environment doctor.
 *
 * Root cause this guards against: running the Node toolchain against a
 * `node_modules` that was installed on a different OS than the current host
 * leaves the platform-specific native binaries (rollup, esbuild) mismatched,
 * producing a cryptic `Cannot find module @rollup/rollup-<platform>` deep in a
 * build. This check turns that into a fast, actionable failure.
 *
 * It only asserts that the binary for the CURRENT host platform resolves, so it
 * passes on any single canonical environment (Windows, Linux, or macOS) and only
 * fails when the tree was populated by a different OS.
 *
 * Invoked automatically via the `predev` / `prebuild` npm hooks.
 */
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);

// rollup names its native packages @rollup/rollup-<platform>-<arch>[-libc].
// We don't hardcode the full matrix; instead we ask rollup itself to load,
// which triggers its own native-binary resolution and throws the precise
// missing-package name when the tree is cross-platform.
function check() {
  try {
    require('rollup');
    return null;
  } catch (err) {
    return err && err.message ? err.message : String(err);
  }
}

const failure = check();
if (failure) {
  const platform = `${process.platform}-${process.arch}`;
  console.error('\n[dev-env doctor] Native dependencies do not match this host.');
  console.error(`[dev-env doctor] Host: ${platform}`);
  console.error(`[dev-env doctor] Detail: ${failure.split('\n')[0]}`);
  console.error('\n[dev-env doctor] This usually means node_modules was installed on a');
  console.error('[dev-env doctor] different OS than the one you are running now.');
  console.error('[dev-env doctor] Fix (run from the canonical environment for this repo):');
  console.error('[dev-env doctor]   - Windows PowerShell:  rmdir /s /q node_modules && npm ci');
  console.error('[dev-env doctor]   - or:                  Remove-Item -Recurse -Force node_modules; npm ci');
  console.error('[dev-env doctor] See docs/dev-environment.md.\n');
  process.exit(1);
}
