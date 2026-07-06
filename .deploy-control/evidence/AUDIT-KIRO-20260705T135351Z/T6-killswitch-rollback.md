# T6 kill-switch + rollback LIVE evidence

- Agent: Codex#5.5#D
- Start timestamp: 2026-07-05T13:59:34Z
- Evidence directory: .deploy-control/evidence/AUDIT-KIRO-20260705T135351Z/
- F0 override: not set by this run
- DEPLOY_OWNER_APPROVED initial state: <unset>

## Commands executed

```bash
MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117 &
scripts/smoke/p7-kill-switch-exercise.sh
SMOKE_ALLOW_EXECUTE=1 bash scripts/smoke/rollback-smoke.sh --execute
kill ${sidecar_pid}
```

## prodex-sidecar

- Start timestamp: 2026-07-05T13:59:35Z
- Command: `MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117`
- PID: 5
- Startup status: exited early
- Exit code: 101

## p7-kill-switch-exercise raw output

- Start timestamp: 2026-07-05T13:59:37Z
- Command: `scripts/smoke/p7-kill-switch-exercise.sh`
- End timestamp: 2026-07-05T13:59:37Z
- Exit code: 1

```text
Traceback (most recent call last):
  File "<stdin>", line 2, in <module>
  File "/usr/lib/python3.12/socket.py", line 233, in __init__
    _socket.socket.__init__(self, family, type, proto, fileno)
PermissionError: [Errno 1] Operation not permitted
```

## rollback-smoke --execute raw output

- Start timestamp: 2026-07-05T13:59:37Z
- Command: `env SMOKE_ALLOW_EXECUTE=1 bash scripts/smoke/rollback-smoke.sh --execute`
- End timestamp: 2026-07-05T13:59:37Z
- Exit code: 1

```text
[rollback-smoke] ERROR: execute blocked: DEPLOY_OWNER_APPROVED is not true
```

## prodex-sidecar shutdown

- Stop timestamp: 2026-07-05T13:59:37Z
- Stop action: already stopped

### prodex-sidecar raw output

```text

thread 'main' (5) panicked at src/main.rs:553:37:
failed to bind sidecar: Os { code: 1, kind: PermissionDenied, message: "Operation not permitted" }
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

## Summary

- End timestamp: 2026-07-05T13:59:38Z
- p7-kill-switch-exercise exit code: 1
- rollback-smoke --execute exit code: 1
- Evidence file: .deploy-control/evidence/AUDIT-KIRO-20260705T135351Z/T6-killswitch-rollback.md

---

# T6 rerun after sandbox socket denial

- Rerun start timestamp: 2026-07-05T14:02:50Z
- Reason: first attempt could not bind/open sockets under sandbox (`Operation not permitted`).
- Capture note: the rerun wrapper emitted `/bin/bash: line 16: Operation: command not found` while writing this reason line because backticks were interpreted by the shell; the target command outputs below were still captured in full.
- F0 override: not set by this run
- DEPLOY_OWNER_APPROVED initial state: <unset>

## Rerun commands executed

```bash
MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117 &
scripts/smoke/p7-kill-switch-exercise.sh
SMOKE_ALLOW_EXECUTE=1 bash scripts/smoke/rollback-smoke.sh --execute
kill ${sidecar_pid}
```

## Rerun prodex-sidecar

- Start timestamp: 2026-07-05T14:02:51Z
- Command: `MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117`
- PID: 49431
- Startup status: running after 2s

## Rerun p7-kill-switch-exercise raw output

- Start timestamp: 2026-07-05T14:02:53Z
- Command: `scripts/smoke/p7-kill-switch-exercise.sh`
- End timestamp: 2026-07-05T14:02:54Z
- Exit code: 0

```text
[p7-kill-switch-exercise] PASS tenant/provider/profile scopes; smart_context/gateway/auto_redeem features; disable and resume behavior
```

## Rerun rollback-smoke --execute raw output

- Start timestamp: 2026-07-05T14:02:54Z
- Command: `env SMOKE_ALLOW_EXECUTE=1 bash scripts/smoke/rollback-smoke.sh --execute`
- End timestamp: 2026-07-05T14:02:54Z
- Exit code: 1

```text
[rollback-smoke] ERROR: execute blocked: DEPLOY_OWNER_APPROVED is not true
```

## Rerun prodex-sidecar shutdown

- Stop timestamp: 2026-07-05T14:02:55Z
- Stop action: SIGTERM sent
- Exit code after stop: 143

### Rerun prodex-sidecar raw output

```text
prodex-sidecar listening on 127.0.0.1:43117
```

## Rerun summary

- Rerun end timestamp: 2026-07-05T14:02:55Z
- p7-kill-switch-exercise exit code: 0
- rollback-smoke --execute exit code: 1
- Evidence file: .deploy-control/evidence/AUDIT-KIRO-20260705T135351Z/T6-killswitch-rollback.md
