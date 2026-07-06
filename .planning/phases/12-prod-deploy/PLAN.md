# PLAN — Phase 12: PROD Deploy + Live Test

phase: 12-prod-deploy
milestone: v2.1
status: BLOCKED (waiting Phase 11 GATE)
depends_on: Phase 11 GATE P11 verde

## Objective

Deploy prodex sidecar to PROD and validate with real provider-backed session. F7 AUTORIZADO pelo dono.

## Pre-Conditions (from Phase 11)

- [ ] All not_validated cells resolved (verified or not_applicable)
- [ ] Smart Context proven per-vendor (tokens_saved>0)
- [ ] Kill-switch + rollback already proven (v2.0 D3)
- [ ] Readyz-falsification proven (HTTP 503 when PG down)

## Task Breakdown

### 12.1 Pre-Deploy Checklist

Verify all gates green:
- P0 Foundation ✅
- P1 Contract ✅
- P2 Fork-map ✅
- P3 Integration ✅
- P4 State/Security ✅
- P5 Vendor Matrix ✅ (after P11)
- P6 QA/Conformance ✅
- P7 Deploy (local) ✅
- P11 Vendor Validation ⏳

### 12.2 Execute Prod-Rollout Runbook

Follow docs/deploy/prod-rollout-runbook.md step by step:
1. Pre-flight checks (env vars, binary pin, Postgres UP)
2. Deploy sidecar binary to PROD host
3. Start sidecar with gateway
4. Verify /readyz returns 200
5. Verify /healthz returns ok
6. Run smoke battery

### 12.3 PROD Session

Start real provider-backed session:
1. POST /v1/session/start with real provider credentials
2. POST request to runtime_endpoint
3. Capture response with Smart Context metrics
4. Verify tokens_saved>0 in PROD

### 12.4 PROD Kill-Switch

POST /v1/killswitch/apply in PROD → verify request routing stops.
POST /v1/killswitch/remove → verify routing resumes.

### 12.5 PROD Rollback

Execute rollback procedure → verify sidecar returns to raw codex mode.

### 12.6 PROD Logs Scrubbed

grep for secrets/tokens in PROD logs → verify 0 matches.

### 12.7 GATE P12

PROD session evidence + kill-switch + rollback + scrubbed logs. Commit + push.

## Staffing

- **Codex#D** (or first free agent) executes sequentially
- **TL** validates evidence

## Evidence

- .deploy-control/evidence/V2-prod-deploy.md
- .deploy-control/evidence/V2-prod-session.md
- .deploy-control/evidence/V2-prod-killswitch.md
- .deploy-control/evidence/V2-prod-rollback.md
- .deploy-control/evidence/V2-prod-logs-scrubbed.md
