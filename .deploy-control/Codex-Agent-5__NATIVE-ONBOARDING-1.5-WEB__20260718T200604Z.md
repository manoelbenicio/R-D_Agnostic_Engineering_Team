agent: Codex Agent-5
stream: NATIVE-ONBOARDING-1.5-WEB
phase: Wave-1
priority: P1
status: IN_PROGRESS
progress: 0
started_at: 2026-07-18T20:06:04Z
finished_at:
files_locked:
  - multica-auth-work/apps/web/app/(auth)/**
  - multica-auth-work/apps/web/app/(landing)/**
  - multica-auth-work/apps/web/features/landing/**
  - multica-auth-work/apps/web/content/use-cases/**
  - multica-auth-work/packages/views/auth/**
  - multica-auth-work/packages/core/auth/**
  - multica-auth-work/packages/core/api/client.ts
  - openspec/changes/native-runtimes-onboarding/tasks.md
  - .deploy-control/evidence/native-onboarding-1.5-web.md
  - .deploy-control/Codex-Agent-5__NATIVE-ONBOARDING-1.5-WEB__20260718T200604Z.md
depends_on: native-runtimes-onboarding task 1.7 backend contract present on disk
plan_ref: .planning/PLAN_RUNTIMES_ONBOARDING_AGENTIC.md task 1.5; openspec/changes/native-runtimes-onboarding/tasks.md task 1.5
build_result:
notes: >
  Golden-Rule ownership claim for Agent-5's web onboarding slice only:
  marketing/landing/sponsors removal and username/password auth service/UI.
  Preserve Google OAuth, CLI callback, and desktop handoff. Agent-6 task 1.6
  design tokens, i18n cleanup, broad web harness/build, and QA remain excluded.
  Existing mobile changes are excluded. Pre-check-in activity was read-only
  inspection requested by the assignment; no product or evidence edits predate
  this check-in. .planning ledger remains read-only under Golden Rule 12.
