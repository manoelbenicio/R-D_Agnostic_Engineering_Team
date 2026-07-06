# Check-in: Codex-5.5-D — P12-PROD-DEPLOY (AGENTIC REAL SESSION)

```yaml
agent: Antigravity (TL orchestrator, Gemini 3.5 Flash)
phase: 12-prod-deploy
milestone: v2.1
status: BLOCKED (12.3 — auth scope mismatch)
started: 2026-07-06T01:51Z
blocker: |
  Codex OAuth token (auth_mode=chatgpt) lacks scope api.responses.write.
  Gateway reaches api.openai.com (REAL upstream, NOT fake) but gets 401 on /v1/responses
  and insufficient_quota on /v1/chat/completions.
  Codex uses ChatGPT subscription billing, NOT API credits.
  The prodex gateway cannot forward using Codex's OAuth token as a standard API key.
  NEED: either (a) a real OPENAI_API_KEY with API credits, or (b) a way to run
  codex CLI itself through the sidecar using its own auth flow.
plan_ref: .planning/phases/12-prod-deploy/PLAN.md
method_ref: .planning/phases/12-prod-deploy/AGENTIC-REAL-SESSION.md
evidence_contract: .planning/EVIDENCE_CONTRACT.md
host: manoelneto-laptop (WSL, fleet run host)
files_locked:
  - .deploy-control/evidence/P12-*
  - .deploy-control/kill-switch/
```

## Task Progress

- [x] 12.0 Check-in (THIS FILE)
- [ ] 12.1 Bring up PROD stack (PG + kill-switch store)
- [ ] 12.2 Deploy prodex L2 PINNED (v0.246.0, NOT smoke)
- [ ] 12.3 AGENTIC real session (per-vendor, real provider, real model)
- [ ] 12.4 Kill-switch LIVE
- [ ] 12.5 Rollback LIVE
- [ ] 12.6 Logs scrubbed
- [ ] 12.7 GATE P12 + backfill P11
