# Evidence — native-runtimes-onboarding task 1.6 (Agent-6) — Web QA

- Agent: Codex Agent-6
- Stream: NATIVE-ONBOARDING-1.6-WEB-QA
- UTC captured: 2026-07-18T20:11:39Z → ~2026-07-18T20:50Z
- base_sha: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` (working tree dirty; multi-agent WIP present, none of it mine)
- Toolchain: node v24.17.0, pnpm 11.15.0, vitest 4.1.0, tsc (workspace), Next.js 16.2.x
- Boundary honored: OFFLINE only — no network, no live providers, no credentials.
- Ownership: NO edits to any product source. Task 1.6 deliverables (color-parity
  gate, i18n cleanup, harness scripts) were already implemented by prior Codex-56-B
  (2026-07-14) and are verified INTACT here. No files of Agent-5 (task 1.5) or
  Codex/root (CREDISO-4.4) touched. tasks.md checkbox NOT self-accepted (deferred to Kiro).

## Scope recap (task 1.6)
(a) design-token / Kanban-Agents color parity · (b) i18n cleanup (orphan
code/resend/verify/download keys + locale structural parity) · (c) web build/test
harness QA with reproducible offline evidence.

---

## VERIFIED — real repository harness, offline (authoritative)

### Vitest — i18n parity (real repo test files)
Command (transient in-package override config: `environment:node`, `setupFiles:[]`,
created + deleted in one shell invocation; rationale below):
```
pnpm exec vitest run --config <tmp>  # locales/parity.test.ts + auth/auth-locale-parity.test.ts
```
Result: **2 files, 158 tests PASSED** — `auth/auth-locale-parity.test.ts` (1) +
`locales/parity.test.ts` (157). Duration 19.96s (wall 39.05s). Exit 0.
- Asserts: all 25 namespaces aligned across en/zh-Hans/ko/ja; per-namespace key
  parity both directions (plural-normalized); auth namespace has **no**
  `code|resend|verify|download` keys; password-login keys (`common.password`,
  `signin.submit`) present.

### Vitest — color-parity gate (real repo test file)
```
pnpm exec vitest run --config <tmp>  # apps/web/test/onboarding-auth-gate.test.ts
```
Result: **1 file, 3 tests PASSED**. Duration 13.55s (wall 42.00s). Exit 0.
- Asserts: removed marketing/email-code surfaces absent (`(landing)`,
  `features/landing`, `content/use-cases`, `public/usecases`); `/auth/login` present
  and no `sendCode|verifyCode` in api client; no `fumadocs|input-otp` dep; login
  locked to shared tokens (`bg-background`, `text-foreground`, `bg-card`,
  `border-border`, `text-muted-foreground`, `text-destructive`) with **no** arbitrary
  palette (`white/black/slate/gray/zinc/neutral`); root redirects `/login`; onboarding
  CTAs use `DESKTOP_RELEASES_URL` and avoid deleted `/download` route.

### Typecheck (tsc --noEmit)
- `@multica/web typecheck` → **PASS** (exit 0), wall 5:39.84 (user 21.15s). Covers web
  + imported `@multica/core`, `@multica/views`, `@multica/ui` type surface (incl. current WIP).
- `@multica/views tsc --noEmit` → **PASS** (exit 0), wall 4:22.70 (user 58.60s).

### Vitest feasibility control
- `@multica/core vitest run utils.test.ts` → **PASS** 2 files / 14 tests, 19.57s (wall 34.63s).
  Confirms vitest itself works on this mount (plain config, no jsdom).

---

## SUPPLEMENTARY — standalone /tmp scripts (NON-authoritative, not in repo)
Fast deterministic cross-checks (pure fs+JSON, no vite transform). Kept in /tmp only.
- `/tmp/i18n-verify.cjs packages/views/locales` → **PASS (0 failures)**, 2.43s. Same
  assertions as the vitest parity tests above.
- `/tmp/gate-verify.cjs <repo>` → **PASS (0 failures)**, 0.10s. Same assertions as the
  gate test above.
These corroborate the real vitest runs; they do NOT substitute for them.

---

## PENDING / BLOCKED / NON-CLAIM (exact evidence)

### next build — BLOCKED by no-network policy (NON-CLAIM)
- `apps/web/app/layout.tsx` imports `Inter, Geist_Mono, Source_Serif_4` from
  `next/font/google` → `next build` fetches Google Fonts at build time.
- Under the standing OFFLINE boundary the build was **interrupted and NOT run to
  completion**. No provably offline/cache-only build mode is guaranteed (next/font may
  revalidate). Left as **BLOCKED / NON-CLAIM**. Requires a networked or font-cached
  container to run — orchestrator (Kiro) decision.

### Full `validate:onboarding-auth` harness — cannot complete end-to-end on this mount
`validate:onboarding-auth = vitest run --maxWorkers=1 && pnpm typecheck && pnpm build`.
- Its unscoped `vitest run` (full web suite incl. jsdom component tests) **deadlocks**
  the vitest jsdom worker pool on the `/mnt/c` (Windows) mount:
  `[vitest-pool-runner]: Timeout waiting for worker to respond` @ 120.07s (reproduced
  twice: default config, and `--pool=forks`). Plain node-env config does NOT hang
  (the two pure-node parity tests ran fine), so this is a jsdom+mount interaction,
  not a code defect. Scoped, DOM-free tests in task 1.6 all pass (above).
  - Note: `--environment=node` alone breaks the package `test/setup.ts` (references
    `window`); hence the transient node-env override with `setupFiles:[]` for the two
    pure-fs parity tests. Timing of that failure: 82.28s, "2 failed (setup) / no tests".
- Its `pnpm build` step is BLOCKED per above.

---

## Conclusion (for Kiro/orchestrator — checkbox NOT self-accepted)
Task 1.6 in-scope deliverables are VERIFIED correct and intact via the real repository
harness, offline: color/design-token parity (gate 3/3), i18n cleanup + locale structural
parity (158/158), and web + views typecheck (both green). No product edits were needed
or made. The only unverified pieces are environment/policy-bound, not code: `next build`
(no-network: next/font/google) and the full unscoped web vitest suite (jsdom worker
deadlock on the Windows mount). Recommend Kiro run `next build` + full suite in the
networked selfhost container to close 3.1/2.5 before checking the box.
