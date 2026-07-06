# Check-in: Codex-5.5-D — P12-PROD-DEPLOY (AGENTIC REAL SESSION)

```yaml
agent: Antigravity (TL orchestrator, Gemini 3.5 Flash)
phase: 12-prod-deploy
milestone: v2.1
status: BLOCKED (12.3 — fork (a) failed, need real API key)
started: 2026-07-06T01:51Z
blocker: |
  FORK (a) INVESTIGATED AND FAILED in <5min. Root cause (JWT decoded):
  - scp = ['openid','profile','email','offline_access'] — NO api.responses.write
  - chatgpt_plan_type = free — no API credits
  - aud = api.openai.com/v1 (correct endpoint) but scopes are OpenID-only (login)
  - Gateway reaches api.openai.com REAL but gets 401 (scope) / insufficient_quota (billing)
  CONCLUSION: the Codex OAuth token cannot be used for API calls. Need a real OPENAI_API_KEY (sk-...).
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
