# PROD Rollout Runbook - prodex AS-IS under Multica

Status: PRE-DEPLOY REQUIRED

## 1. Approval Gate

Before running any command that changes PROD behavior, Opus 4.8 must record:

```text
deploy_owner_approved
```

in:

```text
.deploy-control/evidence/status-board.md
```

Required approval fields:

- owner;
- timestamp;
- artifact hash;
- prodex version/commit;
- rollback command;
- kill switch command;
- accepted risk.

## 2. Pre-Checks

1. Confirm current daemon state.
2. Confirm no active critical incident.
3. Confirm Postgres reachable.
4. Confirm no shared SQLite configured.
5. Confirm prodex pin/integrity.
6. Confirm profile homes exist.
7. Confirm permissions for credential files.
8. Confirm logs directory writable.
9. Confirm scrubber enabled.
10. Confirm kill switch available.
11. Confirm raw codex rollback path preserved.

## 3. Initial Runtime Settings

Recommended initial PROD mode:

```text
Smart Context: shadow
Auto redeem: off
Gateway: only approved provider paths
Runtime events: on
Log format: json if supported
Redaction: on
```

If owner explicitly approves canary:

```text
PRODEX_SMART_CONTEXT_CANARY_PERCENT=1
```

Increase only after evidence is clean.

## 4. Deploy Steps

1. Record pre-deploy status.
2. Apply configuration.
3. Start/restart sidecar path.
4. Run `healthz`.
5. Run `readyz`.
6. Apply policy.
7. Register approved account refs.
8. Start one controlled real session.
9. Verify event stream.
10. Verify no secrets in logs.
11. Verify kill switch can disable Smart Context for next request.
12. Keep observation window open.

## 5. Success Criteria

All must pass:

- real session launches via prodex;
- correct profile isolation;
- event stream emits `session_started`;
- `route_selected` event present;
- no raw secret appears;
- kill switch test passed;
- rollback command ready;
- Go daemon stays healthy;
- sidecar stays ready.

## 6. Immediate Rollback Triggers

Rollback immediately if:

- raw secret appears in log/evidence;
- profile switch is not fail-closed;
- session affinity breaks continuation;
- Smart Context causes protocol/tool-call failure without exact fallback;
- sidecar not ready;
- Go daemon unhealthy;
- Postgres unavailable for required state;
- kill switch cannot be applied.

## 7. Observation Window

Record:

- session id;
- profile id;
- provider;
- model;
- Smart Context mode;
- token before/after if available;
- event ids;
- errors;
- rollback readiness.

## 8. Post-Deploy Evidence

Store scrubbed evidence under:

```text
.deploy-control/evidence/
```

Required:

- precheck result;
- health/readiness result;
- session smoke result;
- event stream snippet;
- redaction smoke result;
- kill switch smoke result.

