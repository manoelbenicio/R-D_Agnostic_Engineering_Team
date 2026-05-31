## Why

The repository lives on a Windows mount (`/mnt/c/...`) but is operated from two
environments at once — Windows and WSL/Linux — against a single shared
`node_modules`. `npm install`/`npm ci` resolves platform-specific optional
dependencies (e.g. `@rollup/rollup-win32-x64-msvc` vs
`@rollup/rollup-linux-x64-gnu`); whichever OS installed last leaves the other
unable to build (`Cannot find module @rollup/rollup-linux-x64-gnu`). The same
shared-mount root cause also defeats Vite's native file watcher (HMR misses
edits, surfacing stale compile errors). These were patched reactively this
session (a `--no-save` rollup install, `usePolling`), but those patches are not
durable — the rollup binary vanishes on the next `npm ci`. This change replaces
the band-aids with a durable, documented, enforced fix.

## What Changes

- Establish and **enforce a single canonical dev environment** for install/build/
  dev-server (WSL/Linux), so cross-platform `node_modules` corruption cannot recur.
- Replace the temporary `npm install --no-save @rollup/rollup-linux-x64-gnu`
  band-aid with a durable resolution (clean platform-correct install verified by
  `npm ci`).
- Keep Vite's `optimizeDeps.entries` scoped to the real SPA entry (`index.html`)
  so stray `*.html` files across the repo never break dependency scanning.
- Keep watcher reliability for the mounted-filesystem case, but as a documented,
  justified setting rather than an ad-hoc patch.
- Add a **preflight doctor check** that fails fast with a clear message when the
  installed platform binaries do not match the host (turns a cryptic rollup stack
  trace into an actionable instruction).
- Document the workflow in the repo so any agent/developer follows the same path.
- **Zero-placeholder/mockup audit**: produce evidence that no placeholder or mock
  data ships in the production bundle, and record any out-of-scope findings.

## Capabilities

### New Capabilities
- `dev-environment`: the contract for a reliable, reproducible local development
  and build environment — canonical environment, dependency-install integrity
  across platforms, dev-server file-watching behavior, dependency-scan scope, a
  preflight environment doctor, and the production-bundle no-mock guarantee.

### Modified Capabilities
<!-- None. No existing capability's spec-level requirements change. -->

## Impact

- **Build tooling**: `vite.config.ts` (`optimizeDeps.entries`, `server.watch`),
  `package.json` scripts (preflight doctor, optional `engines`/OS guard),
  `scripts/` (new doctor script).
- **Dependencies**: removes reliance on the transient `--no-save` rollup binary;
  relies on a clean, platform-correct `npm ci`.
- **Docs**: a new `docs/dev-environment.md` describing the canonical workflow.
- **No application/runtime code changes**: this change does not touch `src/`
  feature code. The audit inspects `src/` but only reports; any fixes to
  product code are out of scope and tracked separately.
- **CI**: verifies a clean `npm ci` + `npm run build` succeed on Linux.
