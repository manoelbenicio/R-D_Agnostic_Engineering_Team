## Context

The repo is checked out on a Windows mount (`/mnt/c/VMs/Projetos/Automonous_Agentic`)
and operated from both Windows and WSL/Linux against one shared `node_modules`.
Node's npm resolves **platform-specific optional dependencies** at install time
(rollup ships per-platform native binaries: `@rollup/rollup-win32-x64-msvc`,
`@rollup/rollup-linux-x64-gnu`, …). When `npm install` runs from Windows it
installs the win32 binaries; a subsequent build from WSL then fails with
`Cannot find module @rollup/rollup-linux-x64-gnu`. The reverse breaks Windows.

Two reactive patches were applied this session and are explicitly being replaced:
- `npm install --no-save @rollup/rollup-linux-x64-gnu@4.60.4` — **not durable**;
  removed by the next `npm ci`.
- `server.watch.usePolling` in `vite.config.ts` — correct for a mounted FS, but
  it was added ad-hoc without justification or documentation.

A scoped, durable fix was also applied and is retained: `optimizeDeps.entries:
['index.html']`, which stops Vite globbing stray repo `*.html` files during
dependency scanning.

**Verified facts (evidence gathered for this design):**
- `package-lock.json` already declares `@rollup/rollup-linux-x64-gnu@4.60.4` as
  an optional dependency. So a clean, platform-correct `npm ci` on Linux installs
  the right binary with no band-aid — the lockfile is already correct.
- Host of record for build/dev is `linux x64` (WSL).
- Production bundle is mock-free: MSW (`setupWorker`, `msw/browser`) appears in
  **zero** runtime `dist/assets/*.js` files; the only reference is in a
  `*.js.map` sourcemap. `import.meta.env.PROD` tree-shakes the dev-only import and
  `postbuild` strips `mockServiceWorker.js`.

## Goals / Non-Goals

**Goals:**
- Make the dev/build environment reproducible so cross-platform `node_modules`
  corruption cannot recur.
- Replace the transient rollup binary with a clean lockfile-driven install.
- Turn the cryptic rollup failure into an actionable, fail-fast preflight message.
- Keep watcher + optimizeDeps reliability, documented and justified.
- Prove, with evidence, that no placeholder/mock data ships in production.

**Non-Goals:**
- No changes to application/runtime feature code in `src/`.
- Not relocating the repo off `/mnt/c` (documented as the recommended long-term
  option, but not forced by this change).
- Not fixing out-of-scope product findings surfaced by the audit (logged only).

## Decisions

**D1 — Canonical environment = Windows (PowerShell) for the Node toolchain.**
Rationale (revised during execution): the repo is on `C:\` (NTFS), surfaced to
WSL as `/mnt/c` via DrvFs. Running `rm -rf node_modules` + `npm ci` from WSL on
that mount **failed three times** with `Input/output error` (Windows-locked
`.node`/`.exe` binaries WSL cannot unlink), `ENOENT` mid-write, and `EACCES
rmdir`. The DrvFs bridge is unreliable for `node_modules`-scale operations.
Running the toolchain natively on Windows (where the disk lives) eliminates both
the cross-platform binary mismatch and the bridge I/O failures. The CAO backend
stays in WSL/Docker and talks to the SPA over HTTP, so the split is fine.
Alternative (canonical = WSL on /mnt/c) **rejected by execution evidence**.
Alternative (move repo into WSL ext4) deferred — best performance but changes
Windows paths; documented as the long-term option.

**D2 — Durable rollup fix = clean `npm ci`, not `--no-save`.**
The lockfile already lists the Linux binary. The remediation is to remove the
mixed tree and reinstall from the lockfile on Linux. Alternative (vendoring the
binary, or committing `--no-save`) rejected: non-reproducible and lost on the
next `ci`. Alternative (`npm config omit=optional`) rejected: rollup *needs* the
native binary.

**D3 — Preflight "doctor" script run from a `predev`/`prebuild` hook.**
A tiny Node script checks that the rollup native binary matching
`process.platform/arch` is resolvable; if not, it exits non-zero with the exact
remediation (`rm -rf node_modules && npm ci` from WSL). Rationale: converts a
deep rollup stack trace into a one-line fix and catches the corruption the moment
it happens. Alternative (rely on humans reading docs) rejected: the failure is
cryptic and recurred multiple times this session.

**D4 — Keep `server.watch.usePolling`, documented.**
Native inotify does not fire reliably on `/mnt/c`. Polling is the standard,
recommended setting for mounted filesystems. Documented in `docs/dev-environment.md`
with the long-term alternative (move repo into the WSL filesystem for native
watching + faster IO).

**D5 — Keep `optimizeDeps.entries: ['index.html']`.**
Scopes dependency scanning to the real SPA entry; prevents stray repo `*.html`
from breaking the dev server. Already applied; formalized here.

**D6 — Audit methodology (zero-placeholder guarantee).**
Multi-sweep grep over `src/` for markers (`TODO|FIXME|placeholder|mock|stub|fake|
dummy|sample|hardcoded|lorem|coming soon|not implemented`), plus a production-bundle
check that no MSW/mock code reaches `dist/assets/*.js`. Distinguish legitimate
hits (HTML `placeholder=` attributes, test-only mocks under `__tests__`/`msw`)
from real ones. Findings recorded in this design (below).

## Audit findings (deep dive)

- **Production bundle: PASS.** No mock/placeholder runtime code ships. MSW only
  in a sourcemap; `strip-msw` removes the worker; PROD guard tree-shakes the import.
- **Legitimate, not placeholders:** input `placeholder=` attributes across
  settings/forms; `example.com` in doc comments and test fixtures; all `mock`/
  `stub`/`fake` hits confined to `__tests__/` and `api/__tests__/msw/`.
- **Out-of-scope real items (logged, NOT fixed here):**
  1. `src/finops/token-cost.ts` — "placeholder list prices" for new model families
     (codex-5.5, gemini-3.5-flash, opus-4) where providers have not published
     official rates. Surfaced to users via `CostConfidence: 'estimated'`. Belongs
     to a FinOps pricing change, not dev-env.
  2. `src/chat-view/ChatView.tsx:54` — `TODO task 12.5` local view-mode fallback
     instead of the settings store. Belongs to a chat-view change.
  These are flagged so they are tracked, not silently absorbed into this change.

## Risks / Trade-offs

- [Polling watcher uses more CPU than native inotify] → Accepted; standard for
  mounted FS. Documented escape hatch: move repo into WSL filesystem.
- [Developer still runs `npm install` from Windows out of habit] → D3 doctor
  fails fast at next `predev`/`prebuild` with the exact fix; documented in
  `docs/dev-environment.md`.
- [Doctor script false-positive on an unanticipated platform] → It only asserts
  the binary for the *current* `process.platform/arch` resolves; it does not
  pin a single OS, so CI on Linux and a dev on macOS both pass.

## Migration Plan

1. From WSL: `rm -rf node_modules && npm ci` (installs Linux binary from lockfile).
2. Add the doctor script + `predev`/`prebuild` hooks.
3. Add `docs/dev-environment.md` (canonical workflow).
4. Verify: `npm run build` and `npm test` green from a clean install.
Rollback: revert `vite.config.ts`/`package.json`/script changes; no data or
runtime impact.

## Open Questions

- Should we additionally relocate the repo into the WSL filesystem now (best IO +
  native watching), or defer? Deferred by default; documented as recommended.
