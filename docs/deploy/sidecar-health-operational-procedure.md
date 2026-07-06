# Sidecar Health Operational Procedure

Status: DRY-RUN READY - LIVE EXECUTION F0-GATED

This procedure validates L2 sidecar liveness/readiness without changing deploy state. LIVE validation is gated by owner approval and the smoke script execution gates.

References:

- `docs/contracts/l2-runtime-contract.md`
- `docs/deploy/l2-sidecar-deploy-plan.md`
- `docs/observability/l2-metrics-and-alerts.md`
- `scripts/smoke/readyz-smoke.sh`
- `scripts/smoke/state-backend-smoke.sh`
- `scripts/smoke/event-stream-smoke.sh`

## 1. Health Model

Operators must treat sidecar readiness as fail-closed. The runtime is not eligible for traffic unless:

- the sidecar answers on loopback only;
- bearer auth is required;
- `/readyz` returns `contract_version=rpp.l2.v1` and `status=ready`;
- all readiness checks pass;
- `shared_state_backend` is present and uses Postgres for shared/runtime state;
- event stream and audit ingestion are available;
- kill-switch store is reachable;
- redaction is enabled for logs, events, and evidence.

## 2. Dry-Run Procedure

Run from the repo root:

```bash
bash scripts/smoke/readyz-smoke.sh --dry-run --base-url http://127.0.0.1:43117
bash scripts/smoke/state-backend-smoke.sh --dry-run --base-url http://127.0.0.1:43117
bash scripts/smoke/event-stream-smoke.sh --dry-run --base-url http://127.0.0.1:43117
```

Dry-run pass criteria:

- each script prints a planned loopback request only;
- no script starts or restarts prodex;
- no bearer token is required in dry-run mode;
- readiness checks include `shared_state_backend`;
- state backend expectations reject SQLite for shared/runtime state.

Record dry-run evidence under `.deploy-control/evidence/` with command names, exit codes, and scrubbed summaries.

## 3. LIVE Procedure

Only run after F0/F7 owner approval is recorded.

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
bash scripts/smoke/readyz-smoke.sh --execute

SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
bash scripts/smoke/state-backend-smoke.sh --execute

SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
bash scripts/smoke/event-stream-smoke.sh --execute
```

LIVE pass criteria:

- all scripts exit 0;
- the base URL is loopback;
- no raw bearer token appears in logs, evidence, shell transcript, or screenshots;
- `/readyz` confirms `rpp.l2.v1`, `ready`, and all checks pass;
- shared state backend is Postgres, not SQLite;
- event stream emits scrubbed lifecycle/control evidence with `secrets_present=false`.

## 4. Failure Response

If sidecar health or readiness fails:

1. Keep or place runtime admission in fail-closed mode.
2. Do not start new prodex/L2-backed sessions.
3. Apply or confirm relevant kill switches for the affected scope.
4. Preserve scrubbed logs and event ids.
5. Escalate to owner and Opus 4.8 with the failing check name and impact.
6. Use `docs/deploy/rollback-operational-procedure.md` if traffic has already been routed to the affected runtime.

## 5. Evidence Fields

Minimum evidence:

```text
health_check_id
timestamp_utc
operator
base_url_loopback_confirmed
readyz_result
state_backend_result
event_stream_result
kill_switch_store_result
redaction_result
secrets_present
decision
remaining_risks
```
