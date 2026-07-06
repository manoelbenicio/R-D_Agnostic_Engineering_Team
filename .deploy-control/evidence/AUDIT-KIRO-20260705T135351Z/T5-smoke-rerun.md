# T5 Smoke Re-run Evidence

- Timestamp UTC: 2026-07-05T14:16:06Z
- Sidecar command: MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117
- Base URL: http://127.0.0.1:43117
- Execute gate: SMOKE_ALLOW_EXECUTE=1
- Target env: SMOKE_TARGET_ENV=local
- DEPLOY_OWNER_APPROVED: not set
- F0 override: not set
- Note: smoke order matches the request; event-stream runs before redaction.

## Preflight

```text
State Recv-Q Send-Q Local Address:Port Peer Address:PortProcess
```

## Sidecar Startup

- PID: 61172

- Health check passed on attempt 1.

## readyz-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/readyz-smoke.sh --execute
```

Exit code: 0

Output:

```text
[readyz-smoke] PASS

```

## policy-apply-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/policy-apply-smoke.sh --execute
```

Exit code: 0

Output:

```text
[policy-apply-smoke] PASS

```

## profile-fail-closed-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/profile-fail-closed-smoke.sh --execute
```

Exit code: 0

Output:

```text
[profile-fail-closed-smoke] PASS

```

## session-start-stop-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/session-start-stop-smoke.sh --execute
```

Exit code: 0

Output:

```text
[session-start-stop-smoke] PASS

```

## kill-switch-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/kill-switch-smoke.sh --execute
```

Exit code: 0

Output:

```text
[kill-switch-smoke] PASS

```

## state-backend-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/state-backend-smoke.sh --execute
```

Exit code: 0

Output:

```text
backend_type=postgres
[state-backend-smoke] PASS

```

## event-stream-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/event-stream-smoke.sh --execute --min-events 1
```

Exit code: 0

Output:

```text
validated_events=2
[event-stream-smoke] PASS

```

## redaction-smoke

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=audit-test-token scripts/smoke/redaction-smoke.sh --execute
```

Exit code: 1

Output:

```text
[redaction-smoke] SKIP logs: path does not exist: /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/logs
[redaction-smoke] SKIP evidence: path does not exist: /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/evidence
[redaction-smoke] Fetching event stream from http://127.0.0.1:43117/v1/events/stream?session_id=session-smoke
no events received
[redaction-smoke] ERROR: redaction checks failed

```

## Sidecar Log

```text
prodex-sidecar listening on 127.0.0.1:43117

```

## Postflight

```text
sidecar_pid_61172_not_running
State Recv-Q Send-Q Local Address:Port Peer Address:PortProcess
```
