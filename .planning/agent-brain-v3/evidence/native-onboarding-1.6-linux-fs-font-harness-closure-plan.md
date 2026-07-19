# Native 1.6 — Offline Closure Plan: vendored OFL fonts + Linux-native jsdom execution

- author: Kiro / Opus-4.8, wave w8:p1 — **ADVISORY ONLY.** No implementation.
- date: 2026-07-18T22:12:00Z
- mode: READ-ONLY. No tool install, no network fetch, no product/test/shared-planning/spec/task/git/index/ref edit; no credentials/env/DB/services. This is the only file created.

## Check-in / check-out
- CHECK-IN 2026-07-18T22:05:00Z — Kiro/Opus-4.8 w8:p1 — stream NATIVE-1.6-OFFLINE-CLOSURE-PLAN — READ-ONLY advisory.
- CHECK-OUT 2026-07-18T22:12:00Z — DONE. Advisory plan below. Kiro TL adjudicates; Agent-6 owns `apps/web/**` implementation; root integrates.

## Reference files (SHA-256 at authoring, HEAD `b6571299`)

| SHA-256 | Path |
|---|---|
| `2997ed83cedbd6ad46fe886e28269ec586ded5e193f698099f59f30683c03dd4` | `apps/web/app/layout.tsx` |
| `9535c99f99925d44ac4c5cb05e5ca97b83e64b3cdce12db50ef6c9c47af738cb` | `apps/web/app/globals.css` |
| `a1ffa217a53282f234bfe4dbf6047c6f0550c86c64045eb79b43bb57217ba08e` | `apps/web/vitest.config.ts` |
| `a0cfe07ad2058a6f97c622b41747d9d790fe3d851a1ee3476833cacedd02099f` | `apps/web/test/setup.ts` |
| `6a828894747e43afc1e7d702a2aa5e3c72d784dfdf023762a62776e08972e1d1` | `apps/web/package.json` |

## Root causes (both confirmed against source)

1. **Fonts need network at build.** `apps/web/app/layout.tsx` uses `next/font/google` for `Inter`, `Geist_Mono`, `Source_Serif_4`. `next/font/google` fetches font binaries from `fonts.gstatic.com` at build time; an offline `next build` (the 1.6 web-build harness) therefore fails or is non-hermetic. Prior online builds cached subset artifacts under `apps/web/.next/static/media/*.woff2` — these are **build artifacts (gitignored), not vendorable license-bearing source**.
2. **jsdom cannot execute on this filesystem.** `apps/web/vitest.config.ts` sets `environment: "jsdom"`. The repo lives on **9p/v9fs** (`stat -f` = `v9fs`; `/mnt/c`), where the vitest worker pool times out on worker startup ("Failed to start … worker / Timeout waiting for worker to respond", **0 tests executed** — established in `packetb-vendor-model-visibility-independent-review.md` and its steer). This is an I/O-locality blocker, not a test failure; it yields a false "compiled, not executed" result.

## Workstream A — vendor local OFL fonts (offline `next build`)

All three families are **SIL OFL-1.1** and may be vendored with their license:
- **Inter** — © The Inter Project Authors (github.com/rsms/inter), OFL-1.1, Reserved Font Name "Inter".
- **Geist Mono** — © Vercel (github.com/vercel/geist-font), OFL-1.1, Reserved Font Name "Geist"/"Geist Mono".
- **Source Serif 4** — © Adobe (github.com/adobe-fonts/source-serif), OFL-1.1, Reserved Font Name "Source".

**File ownership / placement** (Agent-6, `apps/web/**`):
```
apps/web/app/fonts/
  inter/         Inter-*.woff2 (or variable Inter.var.woff2) + OFL.txt
  geist-mono/    GeistMono-*.woff2                            + OFL.txt
  source-serif/  SourceSerif4-*.woff2 (normal+italic)         + OFL.txt
  README.md      source repo+version(tag/commit) per family, sha256 of each woff2, verbatim OFL copyright line
```

**License evidence (OFL-1.1 compliance):**
- Ship each family's upstream `OFL.txt` **verbatim** alongside its binaries (OFL §clause: license text must be bundled).
- Record in `README.md`: upstream repo + release tag/commit, the `Copyright … with Reserved Font Name "…"` line, and the sha256 of every vendored `.woff2`.
- We vendor **as-is** (no subsetting/renaming) → no OFL "Reserved Font Name" derivative concern and no "sold by itself" concern.

**Production edit (the only one required for offline build) — `apps/web/app/layout.tsx`:**
- Replace `import { Inter, Geist_Mono, Source_Serif_4 } from "next/font/google"` with `import localFont from "next/font/local"`.
- Define each family via `localFont({ src: [...vendored woff2...], variable: "--font-inter" | "--font-mono" | "--font-serif", display: "swap", style/weight per file })`, **preserving the exact CSS variable names** (`--font-inter`, `--font-mono`, `--font-serif`) and the existing `fallback` chains.
- Source Serif must keep `normal` + `italic` faces (layout comment: italic h1 accents).
- **`apps/web/app/globals.css` is NOT edited** — it composes `--font-sans` from `var(--font-inter)`; preserving the variable names keeps it working unchanged.

**Commands (offline; implementer runs — I did not run these):**
```sh
# from a working copy (see Workstream B for FS choice)
grep -rn "next/font/google" apps/web            # MUST become empty after migration (stop-condition)
sha256sum apps/web/app/fonts/**/**.woff2        # record into fonts/README.md
pnpm --filter @multica/web build                # offline next build; MUST succeed with no network
pnpm --filter @multica/web typecheck
```

## Workstream B — Linux-native (ext4) jsdom execution

**Principle:** the jsdom suites must run from an **ext4** checkout, not the 9p `/mnt/c` mount. Nothing in the code needs changing; this is an execution-environment procedure.

**Procedure (implementer; requires a pre-populated offline pnpm store — no install/fetch permitted in this lane):**
```sh
# 1. Materialize an ext4 working copy of the current tree (Linux home is ext4):
rsync -a --delete --exclude .next --exclude node_modules \
  /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/ ~/mca-ext4/multica-auth-work/
# 2. Offline install from the existing pnpm store (STOP if store absent):
cd ~/mca-ext4/multica-auth-work && pnpm install --offline --frozen-lockfile
# 3. Run the jsdom suites on ext4:
pnpm --filter @multica/web  test          # vitest run (jsdom)
pnpm --filter @multica/views test
pnpm --filter @multica/core test
pnpm --filter @multica/web  run test:onboarding-auth   # 1.6/1.5 boundary, --maxWorkers=1
```

**Expected nonzero tests (the pass/fail signal that distinguishes ext4 success from 9p false-green):**
- On 9p today: `Test Files no tests / Errors` (worker-start timeout) → **0 executed** = FAIL signal.
- On ext4: each targeted suite MUST report **>0 tests, 0 fail**. Known-nonzero anchors already recorded elsewhere: `packages/core/runtimes/models.test.tsx` = **14**, `packages/views/…/model-dropdown.test.tsx` = **6**, `…/inspector/model-picker.test.tsx` = **3** (from `vendor-model-visibility-ui.md`). The `@multica/web`, `@multica/views`, `@multica/core` aggregate counts must be captured from the actual ext4 run (do **not** assert a fabricated total; the acceptance assertion is "nonzero and green per package", plus the specific onboarding-auth suites executing >0).
- **Font migration does not change test counts** (jsdom does not rasterize fonts); Workstream A and B are independently verifiable.

**Race / concurrency considerations:**
- Run **`--maxWorkers=1` first** (the package's `test:onboarding-auth`/`validate:onboarding-auth` scripts already do) to obtain a deterministic nonzero baseline before enabling parallel workers; this isolates the FS-locality fix from any parallelism flakiness.
- `apps/web/test/setup.ts` stubs `ResizeObserver` and `localStorage` per worker — these are per-worker globals; parallel workers are safe, but a shared retained-catalog/session test (the vendor-model-visibility `QueryClient`-session sentinel) must not use `t.Parallel` across the same key.
- The ext4 copy must be a **quiescent snapshot** — do not run the suite while other agents are concurrently writing `/mnt/c`; re-`rsync` immediately before the run and record the source `git rev-parse HEAD` + dirty-hash so the ext4 result is attributable.
- No DB: the web/views/core jsdom suites are DB-free (unlike the Go `internal/handler` `TestMain`); the ext4 procedure needs no Postgres/network/service.

## Atomic commit boundaries

- **Commit A1 — vendored OFL fonts + local-font migration** (self-contained, offline-build-complete):
  `apps/web/app/fonts/**` (woff2 + per-family `OFL.txt` + `README.md`) **+** `apps/web/app/layout.tsx` (next/font/local). Gate: offline `pnpm --filter @multica/web build` + `typecheck` pass; `grep next/font/google apps/web` empty.
- **Commit A2 (optional, docs only)** — a short `apps/web/README` note documenting the ext4 jsdom execution procedure. Workstream B is otherwise a **process, not a code change** (no commit needed).
- **Exclusions (MUST NOT enter these commits):** A5's auth/landing/marketing removal; `apps/web/app/globals.css` (unchanged); `apps/web/.next/**` (gitignored build cache — never vendor the cached subsets); any mobile/desktop/docs font work; shared planning/OpenSpec/tasks.
- Ownership: `apps/web/**` = Agent-6 (native 1.6); disjoint from A5 (auth) per `tasks.md:10-11`. Confirm no active `IN_PROGRESS` lock on `apps/web/app/layout.tsx` before editing.

## Stop conditions

1. **License:** if any family's upstream is not verifiably OFL-1.1 (or its `OFL.txt`/copyright line cannot be obtained offline) → STOP; do not vendor.
2. **Residual network font:** if `grep -rn "next/font/google" apps/web` is non-empty after migration, or offline `next build` still performs any network fetch → STOP and resolve before claiming offline build.
3. **Offline install prerequisite:** if the pnpm store is not pre-populated for `pnpm install --offline` → STOP (installing/fetching is out of this lane's authority).
4. **FS diagnosis:** if the ext4 run still reports 0 tests, the blocker is not filesystem locality → STOP and re-diagnose (do not claim green).
5. **No fabricated counts:** report the ext4 run's actual per-package nonzero counts; never restate an assumed total as executed.
6. **Authority:** this plan authorizes no edit/commit/push. Kiro TL adjudicates; Agent-6 implements under its ownership; root integrates.

## Non-claims
- Created only this file. Installed nothing, fetched nothing, ran no build/test, edited no product/test/shared/spec/task/git/index/ref. No credentials/env/DB/services.
- I did **not** fetch or vendor any font binary (no network); the OFL sources/versions/placement are specified for an authorized offline implementer to execute.
- The `.next/static/media/*.woff2` cache observed on disk is a build artifact and is explicitly **not** proposed as the vendored source.
- Advisory only: this is a closure plan, not an acceptance, checkbox, or push authorization.
