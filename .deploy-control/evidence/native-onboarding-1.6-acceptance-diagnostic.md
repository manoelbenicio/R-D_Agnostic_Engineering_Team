# Diagnostic — native-runtimes-onboarding task 1.6 acceptance blockers

- Agent: Codex Agent-6 · Stream: NATIVE-ONBOARDING-1.6-ACCEPTANCE-DIAGNOSTIC
- UTC: 2026-07-18T20:58:55Z → 2026-07-18T21:20Z
- base_sha: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (working tree dirty; multi-agent WIP, none mine)
- Toolchain: node v24.17.0, pnpm 11.15.0, vitest 4.1.0, Next.js 16.2.x, jsdom 29.0.1
- Host fs: cwd + `node_modules` on **`/mnt/c` = 9p / v9fs (WSL2 DrvFs)** — high per-file I/O latency
- Mode: **DIAGNOSE-ONLY**. No product/test/spec/task edits, no installs, no network, no
  credentials, no git stage/commit/push. Transient throwaway vitest configs were created for
  reproduction and deleted in the same shell invocation (none committed or left on disk).
- **Does NOT claim task 1.6 acceptance.** Kiro TL/owner decide implementation.

---

## Blocker A — `next build` depends on `next/font/google` network fetch

### Facts (read-only)
- `apps/web/app/layout.tsx` loads 3 faces via `next/font/google`:
  - `Inter({ subsets:["latin"], variable:"--font-inter" })`
  - `Geist_Mono({ subsets:["latin"], variable:"--font-mono", fallback:[…] })`
  - `Source_Serif_4({ subsets:["latin"], style:["normal","italic"], variable:"--font-serif", fallback:[…] })`
- Variables are consumed downstream in static CSS: `apps/web/app/globals.css` composes
  `--font-sans` (Inter + per-`<html lang>` CJK chain) and `packages/ui/styles/tokens.css`
  maps `@theme inline` `--font-sans/-serif/-mono/-heading`. The **CJK tail is pure CSS**, not
  fetched.
- No local font assets exist (`find … -iname '*.woff*|*.ttf|*.otf'` → none under `apps/web/public`
  or `packages/ui`). No `@fontsource/*`, no `next/font/local` usage, no `@vercel/og` dep.
- `.next/server/**/next-font-manifest.json` exist (build manifests) but are **not** cached font
  binaries → a cold offline build still resolves/fetches from `fonts.gstatic.com`.

### Root cause
`next/font/google` fetches the font CSS + `.woff2` binaries from Google at **build time** and
self-hosts them into the build output. With the standing no-network boundary the build cannot
complete. `.next` warm-cache reuse is **not deterministic** (Next may revalidate the font fetch),
so it is not a reliable offline mode.

### Least-risk offline design (PROPOSAL — not implemented)
Switch the 3 faces to **`next/font/local`** with **vendored OFL `.woff2` subsets**, preserving the
exact CSS variable names so all downstream CSS/tokens and the color-parity gate are untouched.
- Licensing (all SIL Open Font License 1.1, embedding + redistribution allowed with license file):
  Inter (rsms), Geist Mono (Vercel), Source Serif 4 (Adobe). Ship each `OFL.txt` beside the fonts.
- Proposed files (owner/Kiro to create):
  - `apps/web/app/fonts/inter-latin.woff2`, `geist-mono-latin.woff2`,
    `source-serif-4-latin.woff2`, `source-serif-4-latin-italic.woff2` (+ `OFL.txt` per family).
    (Or shared under `packages/ui/styles/fonts/` to also serve the desktop renderer.)
  - Edit only `apps/web/app/layout.tsx`: replace the 3 `next/font/google` calls with
    `localFont({ src:[…], variable:"--font-inter|--font-mono|--font-serif", fallback:[…],
    display:"swap", declarations/adjustFontFallback for size-adjust })`. **No changes** to
    `globals.css` / `tokens.css` (variables preserved).
- Font-binary acquisition needs network **once** (owner/CI, e.g. google-webfonts-helper or the
  `@fontsource-variable/*` packages), after which every build is fully offline + deterministic.

### Risks (A)
- Manual `adjustFontFallback`/size-adjust metrics differ slightly from Google's auto values →
  minor CLS unless metrics are set; mitigated by keeping the existing `fallback` arrays.
- Serif must include the **italic** subset (used for onboarding `<em>` accents).
- Repo grows by a few hundred KB of `.woff2`; must include OFL license files.

### Acceptance gates (A)
1. `pnpm --filter @multica/web build` completes with network disabled (DNS blackhole / offline
   namespace) and **zero** requests to `fonts.gstatic.com` / `fonts.googleapis.com`.
2. `onboarding-auth-gate.test.ts` stays 3/3 (tokens unchanged).
3. `--font-inter` / `--font-mono` / `--font-serif` resolve at runtime; no FOUT/CLS regression.
4. No new runtime (client) network calls introduced.

### Interim (defers, does not fix offline)
Keep `next/font/google` and run `build` in the networked selfhost container (has egress at build).

---

## Blocker B — Vitest **jsdom worker startup** times out on `/mnt/c` (9p/DrvFs)

### Reproductions available (this session, transient configs, all deleted)
| # | Config (real repo files) | Result | Wall |
|---|---|---|---|
| baseline | web/views default (jsdom, threads) — scoped 2 files | `[vitest-pool-runner]: Timeout waiting for worker to respond` @120.07s, "no tests / 2 errors" | 2:14 |
| baseline | same, `--pool=forks` | same @120.07s | 2:20 |
| control | `@multica/core` (env=node) `utils.test.ts` | **14 tests PASS** | 34.6s |
| control | views parity+auth-parity, **env=node**, `setupFiles:[]` | **158 tests PASS** | 39.1s |
| control | web `onboarding-auth-gate.test.ts`, **env=node** | **3 tests PASS** | 42.0s |
| B2 | ONE jsdom render test, scoped include, forks+singleFork+isolate:false | `Failed to start forks worker … Timeout waiting for worker to respond` @60.07s, `transform 0ms` | ~1:17 |
| B3 | ONE jsdom render test, **threads+singleThread**, testTimeout/hookTimeout/teardownTimeout=**280000** | same @**60.07s** | ~1:00 |
| B4 | ONE jsdom render test, forks+singleFork, `deps.optimizer.web.include:["jsdom"]` | same @60.07s | ~1:00 |

### Root cause
The pool **worker never finishes startup** (`transform 0ms`, `environment 0ms`) before vitest's
internal "worker respond" ceiling. Evidence the cause is **jsdom environment load over 9p**:
- Node-environment workers start fine on the same mount (controls: core/views/web all pass).
- jsdom-environment workers fail at a **fixed ~60s** regardless of `pool` (forks/threads),
  `singleFork/singleThread`, `isolate`, `fileParallelism`, `deps.optimizer`, or
  `testTimeout/hookTimeout/teardownTimeout` (280s had **no** effect → the ceiling is not those knobs).
- cwd + `node_modules` (incl. jsdom 29's large file tree) sit on `9p/v9fs`, whose per-file `open`
  latency is ~1–2 orders of magnitude worse than ext4; jsdom's many-file `require` graph exceeds
  the worker-respond budget. On a Linux-native fs this init is ~1–2s.
Conclusion: **no pure vitest-config/pool knob available in 4.1.0 fixes this**; it is an I/O-locality
problem, not a code defect.

### Deterministic repo-native fix (PROPOSAL — not implemented), ranked
1. **Run the harness where code AND `node_modules` are on a Linux-native fs (not 9p).** Root-cause
   fix, zero code change, executes the FULL `validate:onboarding-auth` with real DOM assertions.
   The project already ships the container topology.
   - Proposed command (owner/Kiro), node_modules installed on the container/overlay fs:
     `docker compose -f multica-auth-work/docker-compose.selfhost.yml run --rm --no-deps \
        web sh -lc "pnpm install --frozen-lockfile && pnpm --filter @multica/web validate:onboarding-auth"`
     (or a plain `node:24` container with the repo bind-mounted and `pnpm install` targeting an
     overlay `node_modules`). Merely `cd`-ing to `~` will NOT help while `node_modules` remains on 9p.
   - Acceptance gate: `Test Files > 0`, `Tests > 0`, all passed, including jsdom render tests
     (`app/(auth)/login/page.test.tsx`, `app/auth/callback/page.test.tsx`,
     `components/pageview-tracker.test.tsx`); `typecheck` + `build` green (build per Blocker A).
2. **Swap the DOM env `jsdom → happy-dom`** (far lighter file tree; likely starts under the ceiling
   on 9p). Requires adding a devDependency (network — currently blocked) and validating
   `@testing-library/jest-dom` + component compatibility. Medium risk; owner decision.
3. **Environment split (partial mitigation)** via `test.projects` (vitest 4; `environmentMatchGlobs`
   is deprecated): run pure-fs/logic tests under `environment:node` (proven to pass on 9p) and only
   render tests under jsdom. This lets the majority run on 9p but the true DOM tests **still cannot
   start** → does **not** satisfy "full suite with non-zero DOM assertions". Useful only as a fast
   local pre-commit subset, not as the acceptance harness.

### Risks (B)
- Option 1: needs a Linux-native install (network once for `pnpm install`); container parity with
  local Next/vitest versions must match the lockfile (it does — `--frozen-lockfile`).
- Option 2: happy-dom API gaps vs jsdom could cause false pass/fail; must re-run full suite to confirm.
- Option 3: masks the problem; risk of "green locally, untested DOM paths".

---

## Summary for Kiro TL/owner (no acceptance claimed)
- **A**: deterministic offline fix = vendor OFL `.woff2` + `next/font/local`, variables preserved
  (tokens/gate untouched); one-time networked font acquisition; else build in networked container.
- **B**: not a code defect and not fixable via vitest pool/config on `/mnt/c` (9p) — proven across
  pool type, singleFork/Thread, isolate, deps.optimizer, and 280s timeouts (all fail at a fixed
  ~60s during jsdom worker startup). Deterministic fix = run the existing `validate:onboarding-auth`
  harness with code + `node_modules` on a Linux-native fs (selfhost container). happy-dom is a
  candidate but needs a dependency add + compat validation.
- In-scope task-1.6 deliverables remain verified offline (see
  `.deploy-control/evidence/native-onboarding-1.6-web-qa.md`): i18n parity 158/158, color gate 3/3,
  web+views typecheck green. The two items above are environment/policy-bound, decided by Kiro.
