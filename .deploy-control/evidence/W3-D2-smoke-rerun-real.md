# W3-D2 smoke rerun

- timestamp_utc: 2026-07-06T00:04:31Z
- runner: `.deploy-control/evidence/D2-test-runner.sh`
- base_url: `http://127.0.0.1:43292`
- required_port: `43292`
- target: already-running local sidecar
- secrets_present: false

## Plan

C1 readiness, C2 state backend, C3 session replay, C4 event stream replay, C5 Smart Context measurement, C6 fail-closed isolation, then S1-S5 smoke sequence.

## Pre-run Check-in

- repository: `/mnt/c/VMs/Projects/Automonous_Agentic`
- branch: `main`
- head: `c35bd78`
- initial dirty state observed before starting runtime work:
  - `M CHECKIN_OUT.md`
  - `?? .codex/config.toml`
  - `?? .deploy-control/`
  - `?? scripts/smoke/`
- `prodex-sidecar/` does not exist in this repository and was not edited.

## Runtime Setup

- sidecar binary: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- binary mtime: `2026-07-05 20:04:27.647954400 -0300`
- sidecar bind: `127.0.0.1:43292`
- fake OpenAI-compatible upstream: `127.0.0.1:43290`
- prodex gateway listen: `127.0.0.1:43291`
- Postgres probe: real Docker Postgres `deploy-postgres-1`, via temporary `/tmp/d2-smoke-bin/psql` wrapper
- `GET /readyz` before the rerun returned HTTP 200 with `shared_state_backend=postgres`, `connection_status=ok`, and `runtime_proxy=pass`.
- secrets written to this report: false

## Harness Notes

- `.deploy-control/evidence/D2-test-runner.sh` was patched outside `prodex-sidecar/` to use `printf --` for report lines beginning with `-`.
- `scripts/smoke/event-stream-smoke.sh` was patched outside `prodex-sidecar/` to validate `redaction.secrets_present=false`, matching `docs/contracts/runtime-events.schema.json`.

## Raw Output

### preflight-healthz

- started_at: 2026-07-06T00:04:31Z
Command: `bash -c curl\ -fsS\ --max-time\ 8\ -H\ \"Authorization:\ Bearer\ \$\{L2_BEARER_TOKEN\}\"\ \"\$\{L2_BASE_URL%/\}/healthz\"`

- finished_at: 2026-07-06T00:04:31Z
- exit_code: 0

```text
{"contract_version":"rpp.l2.v1","sidecar":{"commit":"smoke","name":"prodex-sidecar","version":"0.1.0"},"status":"alive"}
```

### C1-readyz

- started_at: 2026-07-06T00:04:31Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/readyz-smoke.sh --execute --base-url http://127.0.0.1:43292 --timeout 8`

- finished_at: 2026-07-06T00:04:31Z
- exit_code: 0

```text
[readyz-smoke] PASS

```

### C2-state-backend

- started_at: 2026-07-06T00:04:31Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/state-backend-smoke.sh --execute --base-url http://127.0.0.1:43292 --timeout 8`

- finished_at: 2026-07-06T00:04:32Z
- exit_code: 0

```text
backend_type=postgres
[state-backend-smoke] PASS

```

### C3-session-start-stop

- started_at: 2026-07-06T00:04:32Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/session-start-stop-smoke.sh --execute --base-url http://127.0.0.1:43292 --session-id d2-20260706T000431Z-186332-c3-session`

- finished_at: 2026-07-06T00:04:32Z
- exit_code: 0

```text
[session-start-stop-smoke] PASS

```

### C4-event-stream

- started_at: 2026-07-06T00:04:32Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/event-stream-smoke.sh --execute --base-url http://127.0.0.1:43292 --session-id d2-20260706T000431Z-186332-c3-session --min-events 1`

- finished_at: 2026-07-06T00:04:33Z
- exit_code: 0

```text
validated_events=2
[event-stream-smoke] PASS

```

### C5-smart-context-measure

- started_at: 2026-07-06T00:04:33Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/smart-context-measure.sh --execute --base-url http://127.0.0.1:43292 --context-kib 64 --session-id d2-20260706T000431Z-186332-c5-smart-context --timeout 12`

- finished_at: 2026-07-06T00:04:33Z
- exit_code: 0

```text
{"CONTEXT_TOKENS_AFTER": 16384, "CONTEXT_TOKENS_BEFORE": 16384, "REQUEST_BYTES": 67209, "RESPONSE_BYTES": 569, "compression_ratio": 1.0, "fallback_triggered": false, "metric_sources": {"CONTEXT_TOKENS_AFTER": "inferred_no_response_token_metric", "CONTEXT_TOKENS_BEFORE": "payload.smart_context_probe.context_tokens_before_estimate", "fallback_triggered": "inferred_from_smart_context_mode"}, "response_summary": {"contract_version": "rpp.l2.v1", "event_stream_url_present": true, "router_owner": "rust_l2", "runtime_session_id_present": true, "smart_context_mode": "proxy_rewrite"}, "target": {"base_url": "http://127.0.0.1:43292", "path": "/v1/session/start", "session_id": "d2-20260706T000431Z-186332-c5-smart-context"}, "tokens_saved": 0}

```

### C6-profile-fail-closed

- started_at: 2026-07-06T00:04:33Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/profile-fail-closed-smoke.sh --execute --base-url http://127.0.0.1:43292 --invalid-profile-home /tmp/rpp-smoke-outside-managed-root`

- finished_at: 2026-07-06T00:04:33Z
- exit_code: 0

```text
[profile-fail-closed-smoke] PASS

```

### S1-readyz

- started_at: 2026-07-06T00:04:33Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/readyz-smoke.sh --execute --base-url http://127.0.0.1:43292 --timeout 8`

- finished_at: 2026-07-06T00:04:34Z
- exit_code: 0

```text
[readyz-smoke] PASS

```

### S2-policy-apply

- started_at: 2026-07-06T00:04:34Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/policy-apply-smoke.sh --execute --base-url http://127.0.0.1:43292 --tenant-id tenant-smoke`

- finished_at: 2026-07-06T00:04:34Z
- exit_code: 0

```text
[policy-apply-smoke] PASS

```

### S3-account-fail-closed

- started_at: 2026-07-06T00:04:37Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/profile-fail-closed-smoke.sh --execute --base-url http://127.0.0.1:43292 --invalid-profile-home /tmp/rpp-smoke-outside-managed-root`

- finished_at: 2026-07-06T00:04:38Z
- exit_code: 0

```text
[profile-fail-closed-smoke] PASS

```

### S4-session-start-stop

- started_at: 2026-07-06T00:04:38Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/session-start-stop-smoke.sh --execute --base-url http://127.0.0.1:43292 --session-id d2-20260706T000431Z-186332-s4-session`

- finished_at: 2026-07-06T00:04:38Z
- exit_code: 0

```text
[session-start-stop-smoke] PASS

```

### S5-kill-switch

- started_at: 2026-07-06T00:04:38Z
Command: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/smoke/kill-switch-smoke.sh --execute --base-url http://127.0.0.1:43292 --feature smart_context`

- finished_at: 2026-07-06T00:04:38Z
- exit_code: 0

```text
[kill-switch-smoke] PASS

```

## Result

PASS - all D2 smoke steps exited 0.
