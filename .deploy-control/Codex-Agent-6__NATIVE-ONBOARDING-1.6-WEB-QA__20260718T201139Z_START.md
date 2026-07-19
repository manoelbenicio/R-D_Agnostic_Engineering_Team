agent: Codex Agent-6
stream: NATIVE-ONBOARDING-1.6-WEB-QA
phase: Wave-1
task: 1.6
priority: P1
status: BLOCKED
progress: 85
eta: needs orchestrator decision (networked build env)
started_at: 2026-07-18T20:11:39Z
finished_at: 2026-07-18T20:50:00Z
depends_on: none (frontend-only; disjoint from Agent-5 task 1.5 and Codex/root CREDISO-4.4)
blockers: >
  (1) `next build` requires network — apps/web/app/layout.tsx uses next/font/google
  (Inter, Geist_Mono, Source_Serif_4); interrupted + NOT run under the standing OFFLINE
  boundary → NON-CLAIM. (2) Full unscoped `vitest run` (validate:onboarding-auth step 1)
  deadlocks the jsdom worker pool on the /mnt/c Windows mount ("Timeout waiting for worker
  to respond" @120.07s, reproduced twice). Both are environment/policy limits, NOT code
  defects. Recommend Kiro run build + full suite in the networked selfhost container.
build_result: >
  green for ALL task-1.6 in-scope deliverables via the REAL repo harness, offline:
  vitest i18n parity 158/158 PASS (19.96s); vitest color-parity gate 3/3 PASS (13.55s);
  @multica/web typecheck PASS (5:39.84); @multica/views typecheck PASS (4:22.70).
  next build = NON-CLAIM (no-network: next/font/google). full web vitest suite = PENDING
  (jsdom worker deadlock on /mnt/c). No product source edited (deliverables pre-existing
  from Codex-56-B and verified intact). tasks.md checkbox NOT self-accepted.
ack: Codex-Agent-6 @ 2026-07-18T20:11:39Z  status: ACKNOWLEDGED
base_sha: b6571299b00c8e388abefe7ef9dcbcf8ac715d7f (working tree dirty; multi-agent WIP present)
plan_ref: openspec/changes/native-runtimes-onboarding/tasks.md task 1.6; design.md Wave-1 Agent-6
files_locked:
  - multica-auth-work/packages/ui/styles/tokens.css
  - multica-auth-work/packages/ui/styles/base.css
  - multica-auth-work/apps/web/app/globals.css
  - multica-auth-work/packages/views/locales/en/**
  - multica-auth-work/packages/views/locales/ja/**
  - multica-auth-work/packages/views/locales/ko/**
  - multica-auth-work/packages/views/locales/zh-Hans/**
  - multica-auth-work/packages/views/locales/index.ts
  - multica-auth-work/packages/views/locales/parity.test.ts
  - multica-auth-work/apps/web/test/onboarding-auth-gate.test.ts
  - multica-auth-work/apps/web/package.json  # validate:onboarding-auth harness script only
  - .deploy-control/evidence/native-onboarding-1.6-web-qa.md
  - .deploy-control/Codex-Agent-6__NATIVE-ONBOARDING-1.6-WEB-QA__20260718T201139Z_START.md
scope: >
  Task 1.6 (Agent-6) frontend-only: (a) design-token / Kanban-Agents color parity
  audit + gate; (b) i18n cleanup (orphan code/resend/verify/download keys) and
  locale structural parity; (c) offline web build/test harness QA with reproducible
  evidence.
excluded_files (owned by others; NOT touched):
  - marketing/landing/sponsors removal — owned by Agent-5 task 1.5
  - multica-auth-work/apps/web/app/(auth)/**, (landing)/**, features/landing/**, content/use-cases/** — Agent-5
  - multica-auth-work/packages/views/auth/** (incl. login-page.tsx, auth-locale-parity.test.ts) — Agent-5
  - multica-auth-work/packages/core/auth/**, packages/core/api/client.ts — Agent-5
  - openspec/changes/native-runtimes-onboarding/tasks.md — Agent-5 (checkbox NOT self-accepted here)
  - packages/core/runtimes/models.ts, packages/core/types/agent.ts, packages/views/agents/components/{model-picker,model-dropdown,runtime-picker}* — other WIP
  - multica-auth-work/apps/mobile/** — excluded per Agent-5 note
  - multica-auth-work/server/** (Go) — Codex/root + backend agents
notes: >
  Cross-checked Agent-5 check-in Codex-Agent-5__NATIVE-ONBOARDING-1.5-WEB__20260718T200604Z:
  Agent-5 explicitly excludes 1.6 design tokens, i18n cleanup, web harness/build, QA (mine).
  No file overlap with Agent-5's files_locked or Codex/root CREDISO-4.4 files_locked.
  Pre-check-in activity was read-only inspection of OpenSpec change (proposal/design/tasks +
  onboarding/agent-runtimes/model-discovery specs), ledger, and current diff. No product or
  evidence edits predate this check-in. Acceptance requires reproducible offline
  typecheck/tests/lint/build where feasible; checkbox in tasks.md left to Kiro/orchestrator.
