# RESEARCH — Phase 12: how to run the REAL deploy (execution-ready)

phase: 12-prod-deploy
author: Kiro/Principal
purpose: Remove all improvisation. Every 12.x task has concrete steps + the exact real-vs-fake checks.

## Reference
Follow docs/deploy/prod-rollout-runbook.md. This doc pins the concrete commands + gates per task.

## 12.1 — Bring up PROD stack
- Postgres + Redis via the deploy compose file (real service, not skipped).
- Create kill-switch store (was missing). Apply migrations up (reversible).
- GATE: PG reachable from the sidecar host; migration applied.

## 12.2 — Deploy pinned prodex L2
- Deploy the PINNED binary (confirm version+commit from PREREQUISITES; NOT "smoke"/0.1.0).
- Wire env: MULTICA_PRODEX_*, MULTICA_L2_* with REAL provider keys (from owner), real gateway token.
- Start sidecar + gateway on the REAL host.
- GATE: /readyz 200 with real PG probe (kill PG → expect 503, then restore); /healthz ok.

## 12.3 — REAL provider-backed session (the crux)
For EACH vendor the owner supplied keys for:
- POST /v1/session/start with that vendor's real provider creds → capture runtime_endpoint + runtime_session_id.
- POST a real payload to runtime_endpoint (/v1/runtime/proxy).
- CAPTURE and CHECK against EVIDENCE_CONTRACT:
  - gateway_status == 200
  - measurement_source == gateway_usage
  - gateway_response_model == a REAL model id (assert != "fake-upstream-logging")
  - usage is realistic (assert not input=8/output=1)
  - runtime_session_id and tokens_saved are DISTINCT per vendor
- Evidence: one file per vendor (P12-session-<vendor>-real.md) with the raw request/response provenance.

## 12.4 — Kill-switch LIVE
- Baseline: a request routes (observe). Apply kill-switch (tenant/provider/profile). Re-request →
  observe routing STOPS. Remove → observe routing RESUMES. Capture before/after, not prose.

## 12.5 — Rollback LIVE
- Execute the 1-command rollback (runbook §7). Re-request → observe service back to raw-codex. Capture.

## 12.6 — Logs scrubbed
- grep -RniE 'sk-|bearer|api[_-]?key|token=' <prod log path> → expect 0 real secrets. Show the command+result.

## 12.7 — GATE + backfill
- Write SUMMARY.md. Replace P11 local_estimate rows with the real 12.3 numbers. Commit + push (no build
  artifacts: ensure target/ is gitignored). Update STATE.md to reflect real completion.

## Failure handling
Any gate fail or any EVIDENCE_CONTRACT violation → mark evidence INVALID, revert task to BLOCKED,
escalate to Kiro. Never substitute a fake to make a gate go green.
