# Phase 07 — DevOps / Deploy: SUMMARY

phase: 07-deploy
status: COMPLETE (F5 vendor sign-off + F7 PROD deploy remain owner-gated)
tasks_ref: openspec/changes/rotation-parity-polyglot/tasks.md §7 (7.1–7.7)

## What Was Delivered

Full deploy pipeline validated locally with evidence:

- **Kill-switch:** per tenant/provider/profile, tested real (applied=true, effective_at=next_request)
- **Rollback:** single-command return to raw `codex`, tested real
- **Readyz-falsification:** PG down → HTTP 503 `status=error` (NOT hardcoded); PG up → HTTP 200 `status=ready`
- **Log scrubbing:** 7 surfaces, 12 checks, 3 redaction engines validated
- **Runbook:** `docs/deploy/prod-rollout-runbook.md` (12 deploy steps, 9 success criteria, 8 rollback triggers)
- **Observability:** Prometheus:9090, Grafana:3000→13000, Alertmanager:9093, pg-exporter:9187
- **Dashboards:** 6 Grafana JSONs (Credential Health, Accounts & Quota, Rotation, Platform Health)
- **Alerts:** 7 rules (CredentialRestoreFailing, EnvInjectionFailing, AllAccountsExhausted, RotationFailing, NoAvailableAccounts, PostgresDown, SecretInLogSuspected)
- **CI:** go vet + lint + security scan + go test -race in `.github/workflows/ci.yml`

## GATE P7

Kill-switch + rollback green; controlled local `prodex-sidecar` session validated ✅
PROD provider-backed session not executed (F5/F7 owner-gated).

## Evidence

- `.deploy-control/evidence/p7-kill-switch-test.md`
- `.deploy-control/evidence/p7-rollback-test.md`
- `.deploy-control/evidence/W3-D3-killswitch-rollback.md` (real runtime retest + readyz-falsification)
- `.deploy-control/evidence/p7-logs-scrubbed.md`
- `.deploy-control/evidence/p7-observability-enrollment-helm-20260705.md`
- `.deploy-control/evidence/p6-p7-final-gates-20260705T062826Z.md`
- `docs/deploy/prod-log-scrubbing-validation.md`
- `docs/deploy/prod-rollout-runbook.md`
- `docs/deploy/kill-switch-operational-procedure.md`
- `docs/deploy/rollback-operational-procedure.md`
