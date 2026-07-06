agent: Codex#5.5#D
stream: P12-121-122-PROD-DEPLOY
phase: P12
priority: P0
status: IN_PROGRESS
progress: 0
started_at: 2026-07-06T03:14:16Z
finished_at:
files_locked:
  - .deploy-control/Codex-5.5-D__P12-121-122-PROD-DEPLOY__20260706T031416Z.md
  - .deploy-control/evidence/P12-121-122-prod-deploy.md
  - .deploy-control/evidence/P12-121-122-*.log
  - deploy/**
  - scripts/deploy/**
  - multica-auth-work/docker-compose*.yml
  - multica-auth-work/**/migrations/**
  - .deploy-control/kill-switch/**
depends_on: P12 PREREQUISITES (real PROD host, pinned v0.246.0 binary, real non-fake provider infra); tasks 12.1/12.2
plan_ref: .planning/phases/12-prod-deploy/PLAN.md
build_result:
notes: Starting P12 tasks 12.1 and 12.2 only: bring up PROD stack prerequisites and deploy pinned prodex L2 if real PROD prerequisites exist. Evidence must satisfy .planning/EVIDENCE_CONTRACT.md; no fake API keys, no localhost-as-PROD evidence.
