# PLAN — Phase 12: PROD Deploy + Live Test  (milestone v2.1)

phase: 12-prod-deploy
milestone: v2.1
status: IN_PROGRESS  (OWNER OVERRIDE 2026-07-05: deploy NOW — TOP PRIORITY; do NOT wait P11 gate)
depends_on: none (P11 real per-vendor numbers are CAPTURED IN PROD via this deploy)

## Owner directive
Deploy EVERYTHING to PRODUCTION now, run ONE real provider-backed live session, prove kill-switch +
rollback LIVE. The PROD live session supersedes the local P11 estimate: real providers = gateway 200,
so real tokens_saved is measured HERE (also closes the OpenCode/GLM5.2 + Kiro/Opus4.8 vendor gaps).
F7 AUTHORIZED. MANDATORY: create a Golden-Rule check-in BEFORE touching anything.

## Task Breakdown (evidence-gated; each task writes raw evidence)

### 12.0 Check-in (Golden Rule — BEFORE any command)
- [ ] Create .deploy-control/Codex-5.5-D__P12-PROD-DEPLOY__<UTC>.md (files_locked, status IN_PROGRESS)

### 12.1 Bring up PROD stack (infra prereqs found live)
- [ ] Start Postgres + Redis via docker-compose (deploy/) — Postgres was NOT running
- [ ] Create kill-switch store (was missing)
- [ ] Apply reversible migrations (up)

### 12.2 Deploy prodex L2 (runbook docs/deploy/prod-rollout-runbook.md)
- [ ] Pinned binary (v0.246.0 / 7750da9b), env wired (MULTICA_PRODEX_* + MULTICA_L2_*)
- [ ] Start sidecar+gateway; /readyz 200 (REAL PG probe), /healthz ok

### 12.3 REAL provider-backed session (this is the real P11 measurement)
- [ ] POST /v1/session/start with REAL provider creds; POST body to runtime_endpoint /v1/runtime/proxy
- [ ] Gateway returns 200 (NOT 404); measurement_source = REAL (not local_estimate)
- [ ] Capture tokens_saved REAL per real vendor(s) in use: OpenAI/Codex, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8
- [ ] router_owner=rust_l2

### 12.4 Kill-switch LIVE in PROD
- [ ] apply (tenant/provider/profile) -> routing stops; remove -> resumes

### 12.5 Rollback LIVE in PROD
- [ ] rollback-to-raw-codex -> service recovers to raw codex

### 12.6 Logs scrubbed in PROD path
- [ ] grep secrets/tokens in PROD logs -> 0 matches

### 12.7 GATE P12
- [ ] PROD up + real session (real tokens_saved) + kill-switch + rollback + scrubbed logs; SUMMARY + commit+push

## Staffing
- Codex#5.5#D executes (check-in FIRST). TL validates (re-runs). Only one owner on hotspot.

## Evidence
- .deploy-control/evidence/P12-prod-deploy-live.md · P12-prod-session-real.md · P12-killswitch-prod.md
  · P12-rollback-prod.md · P12-logs-scrubbed-prod.md
