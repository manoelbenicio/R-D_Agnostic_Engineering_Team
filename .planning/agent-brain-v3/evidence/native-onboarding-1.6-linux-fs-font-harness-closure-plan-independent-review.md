# Independent review — native-onboarding 1.6 Linux-FS + font harness closure plan

- Reviewer: Kiro/Opus-4.8, pane **w7:p2** — **distinct pane from the plan author (w8:p1)**. Independent, read-only.
- Under review: `native-onboarding-1.6-linux-fs-font-harness-closure-plan.md` (author w8:p1).
- **Advisory only.** No network/install/build/test; no product/shared-planning/spec/task/git/index/ref/credential/env/DB/service mutation. This is the only file created.

## Check-IN / Check-OUT
- **Check-IN** 2026-07-18T22:40:00Z — read-only verification vs current `apps/web` sources + package config + existing 1.6 diagnostics.
- **Check-OUT** 2026-07-18T22:56:00Z — DONE. Verdict below. Kiro TL adjudicates.

## Provenance — plan's reference hashes re-verified against current bytes (HEAD `b6571299`)

| File | Plan SHA-256 | Re-verified | Match |
|---|---|---|---|
| `apps/web/app/layout.tsx` | `2997ed83…03dd4` | `2997ed83…03dd4` | ✅ |
| `apps/web/app/globals.css` | `9535c99f…738cb` | `9535c99f…738cb` | ✅ |
| `apps/web/vitest.config.ts` | `a1ffa217…ba08e` | `a1ffa217…ba08e` | ✅ |
| `apps/web/test/setup.ts` | `a0cfe07a…d2099f` | `a0cfe07a…d2099f` | ✅ |
| `apps/web/package.json` | `6a828894…2e1d1` | `6a828894…2e1d1` | ✅ |

No drift. The plan reviews the current state.

## Claim-by-claim verification (local evidence)

| Plan claim | Verified against | Result |
|---|---|---|
| layout.tsx uses `next/font/google` for Inter/Geist_Mono/Source_Serif_4 | layout.tsx:3, 24-52 | **CONFIRMED** |
| CSS variables `--font-inter`/`--font-mono`/`--font-serif` | layout.tsx (`variable:` each) + className | **CONFIRMED** |
| Source Serif needs normal+italic | layout.tsx `style: ["normal","italic"]` | **CONFIRMED** |
| globals.css composes `--font-sans` from `var(--font-inter)`; needs no edit | globals.css `:root` **and** `html[lang|="ja"]` both use `var(--font-inter)` | **CONFIRMED** (two sites, not one — preserving the var keeps both) |
| Repo on 9p; jsdom worker-start timeout, 0 tests | `stat -f`=v9fs; 1.6 acceptance-diagnostic Blocker B (fixed ~60s startup timeout across all pool/config/280s-timeout knobs; node-env controls pass) | **CONFIRMED + strongly corroborated** |
| `.next/static/media/*.woff2` are gitignored build artifacts, not vendorable | present on disk; diagnostic agrees ("not cached font binaries / not deterministic") | **CONFIRMED** |
| pnpm assumptions (`--filter`, `--frozen-lockfile`, `catalog:`) | root `packageManager: pnpm@10.28.2`; `pnpm-lock.yaml`+`pnpm-workspace.yaml`; `catalog:`/`workspace:*` in deps | **CONFIRMED** |
| `test:onboarding-auth`/`validate:onboarding-auth` use `--maxWorkers=1`; suites exist | package.json scripts; `test/onboarding-auth-gate.test.ts` + `app/(auth)/login/page.test.tsx` present | **CONFIRMED** |
| Fonts (Inter/Geist Mono/Source Serif 4) are SIL OFL-1.1 w/ named copyrights + Reserved Font Names | **no OFL.txt / font source / license file in repo** | **NOT locally verifiable — see Flags** |

## Technical feasibility

- **Workstream A (vendor OFL + `next/font/local`): FEASIBLE.** Families, variable names, fallback chains, and the
  normal+italic serif requirement are all verified in source; globals.css is safe to leave untouched. The migration
  matches the prior Agent-6 diagnostic's own proposal.
  - **Completeness gap 1 (minor):** `subsets: ["latin"]` is a `next/font/google`-only option; `next/font/local`
    does **not** accept `subsets`. The migration must **drop** `subsets` — the plan does not call this out.
  - **Completeness gap 2 (minor):** the plan does not mention `adjustFontFallback`/size-adjust metrics. The 1.6
    diagnostic explicitly flags CLS risk when manual local-font metrics differ from Google's auto values (Risk A).
    Preserving the `fallback` arrays (plan does) mitigates but does not fully replace size-adjust; worth an explicit
    note for the implementer.
  - Nit: Inter has **no** `fallback` array in source (only geistMono/sourceSerif do); the plan's "preserve existing
    fallback chains" is accurate only for two of three families — Inter relies on the globals.css `--font-sans` chain.
- **Workstream B (ext4 jsdom run): FEASIBLE and root-cause-correct.** The plan's `rsync --exclude node_modules` +
  `pnpm install --offline` on ext4 correctly puts **node_modules on native fs**, directly addressing the
  diagnostic's caveat that "merely `cd`-ing to `~` will NOT help while node_modules remains on 9p." Sound.
  - **Primary feasibility risk:** the offline-only path (`pnpm install --offline --frozen-lockfile`) is load-bearing
    and depends on a **pre-populated pnpm store** that matches the lockfile. Note a version discrepancy: root pins
    `pnpm@10.28.2` while the 1.6 diagnostic ran `pnpm 11.15.0` — corepack/store mismatch could break a strict
    offline frozen install. The plan gates this with STOP condition 3 (correct), but it is the main execution risk.
  - The diagnostic's alternative (selfhost Docker container with `pnpm install --frozen-lockfile`) is an equally
    valid native-fs target and does not require a warm offline store; the plan could note it as a fallback.
- **Expected nonzero gates: SOUND.** The plan refuses fabricated aggregate totals and requires actual per-package
  nonzero+green, plus the onboarding-auth suites executing >0. Its cited anchors (models 14 / dropdown 6 / picker 3)
  come from Packet-B evidence; the diagnostic offers independent nonzero anchors (i18n parity 158, onboarding-gate 3
  under env=node). Both are real; the "capture actual counts" discipline is correct.

## Governance

- **License (BLOCKING for the font commit, external verification required):** the OFL-1.1 status, copyright lines,
  and Reserved Font Names for all three families are **stated as fact in the plan but cannot be verified from local
  evidence** — no OFL.txt or font source exists in the repo. The plan's **Stop condition 1** correctly gates
  vendoring on obtaining/verifying each `OFL.txt` offline, which is the right control. Recommendation: treat the
  license assertions as **unverified pending upstream OFL.txt + copyright-line capture** at implementation; the
  reviewer flags this as the item requiring external (network/upstream) verification.
- **Network/authority separation: clean.** The plan authorizes no edit/commit/push, forbids install/fetch in-lane,
  and correctly isolates the one-time font acquisition and (if the store is cold) any pnpm fetch as separately
  authorized networked steps. `apps/web/**` ownership = Agent-6; author is advisory; Kiro TL adjudicates; root
  integrates. Atomic boundaries (Commit A1 = `fonts/**` + layout.tsx; globals.css excluded and verified safe;
  `.next/**` excluded; A5 auth removal excluded) are coherent and non-overlapping.
- **Provenance chaining (minor):** the nonzero anchors depend on other evidence artifacts (Packet-B, vendor-model
  visibility); their acceptance is assumed, not re-verified here.

## Verdict (advisory)

- **Plan is technically sound and faithful to current source and the existing 1.6 diagnostics.** Both root causes
  (offline font fetch; 9p jsdom worker-startup timeout) are independently confirmed; the two workstreams and atomic
  ownership are correct.
- **Conditions before implementation/acceptance:** (1) obtain + verify each family's OFL.txt/copyright upstream
  (external — license claim not locally provable); (2) drop `subsets` and consider `adjustFontFallback` metrics in
  the localFont migration; (3) confirm the offline pnpm store matches the lockfile under the pinned pnpm version, or
  use the container native-fs path; (4) capture actual per-package nonzero counts (no fabricated totals).
- **Feasibility: PASS.** **Governance: PASS with the license external-verification flag as the one blocking gate.**
- Reviewer authorizes nothing; sets no checkbox. Kiro TL adjudicates; Agent-6 implements under `apps/web/**`; root integrates.

## Non-claims
- Read-only. No network/install/build/test run; no product/shared/spec/task/git/index/ref edit; no credentials/env/DB/service.
  Font-family/CSS/migration/FS/package claims verified from local source at HEAD `b6571299`; OFL license claims are
  explicitly **not** locally verified and are flagged for external verification. Only this file was created.
