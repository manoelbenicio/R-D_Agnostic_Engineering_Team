# V1.3 reset-claim Codex evidence

- timestamp_utc: 2026-07-06T00:50:03Z
- milestone: v2.1
- phase: V1
- task: V1.3 reset-claim Codex
- target_sidecar: `http://127.0.0.1:43292`
- tenant_id: `codex-redeem-test`
- secrets_present: false

## Check-in before

- `CHECKIN_OUT.md` received a CHECK-IN entry before this investigation.
- `prodex-sidecar/` was not edited. The current repository has no local `prodex-sidecar/` directory.
- The running sidecar process is external to this repository:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43292
```

## POST /v1/session/start

Unauthenticated probe, as first attempted:

```bash
curl -sS -i -X POST http://127.0.0.1:43292/v1/session/start \
  -H 'Content-Type: application/json' \
  --data '{"tenant_id":"codex-redeem-test"}'
```

Result:

```text
HTTP/1.1 401 Unauthorized
{"error":"unauthorized"}
```

Authenticated probe used the sidecar process bearer token without printing it. Payload included `tenant_id=codex-redeem-test` and the required `rpp.l2.v1` session fields:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_start_codex-redeem-test-session",
  "tenant_id": "codex-redeem-test",
  "workspace_id": "workspace-codex-redeem-test",
  "task_id": "task-reset-claim-probe",
  "session_id": "codex-redeem-test-session",
  "policy_id": "policy-smoke-shadow",
  "requested_provider": "codex",
  "requested_model": "gpt-5",
  "working_directory": "/tmp/rpp-smoke-workspace",
  "profile_pool": ["codex-smoke-main", "codex-smoke-backup"],
  "continuation": {
    "previous_response_id": null,
    "session_binding_hint": null
  }
}
```

Result summary:

```text
POST /v1/session/start HTTP_STATUS=200
router_owner=rust_l2
runtime_session_id=rt-1783299114888494826
event_stream_url_present=true
smart_context_mode=proxy_rewrite
```

Cleanup:

```text
POST /v1/session/stop HTTP_STATUS=200
stopped=true
```

## redeem/reset-claim endpoint investigation

Command:

```bash
grep -Rni "redeem\|reset.claim\|reset-claim\|reset_claim\|claim" \
  /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/src/main.rs
```

Result:

```text
no matches
```

Route table excerpt from `main.rs`:

```text
(Method::Get, "/healthz") => handle_healthz(),
(Method::Get, "/readyz") => handle_readyz(),
(Method::Get, p) if p.starts_with("/v1/killswitch/status") => handle_killswitch_status(p),
(Method::Get, p) if p.starts_with("/v1/events/stream") => handle_events_stream(p),
(Method::Post, p) if p.starts_with("/v1/runtime/proxy") => handle_runtime_proxy(req, p),
(Method::Post, "/v1/policy/apply") => post_json(req, handle_policy_apply),
(Method::Post, "/v1/accounts/register") => post_json(req, handle_accounts_register),
(Method::Post, "/v1/session/start") => post_json(req, handle_session_start),
(Method::Post, "/v1/session/stop") => post_json(req, handle_session_stop),
(Method::Post, "/v1/killswitch/apply") => post_json(req, handle_killswitch_apply),
_ => error_response(404, "not found"),
```

Authenticated HTTP probes:

```text
POST /v1/redeem HTTP_STATUS=404 {"error":"not found"}
POST /v1/session/redeem HTTP_STATUS=404 {"error":"not found"}
POST /v1/session/reset-claim HTTP_STATUS=404 {"error":"not found"}
POST /v1/reset-claim HTTP_STATUS=404 {"error":"not found"}
```

## Conclusion

- The sidecar adapter running on `127.0.0.1:43292` does not expose a redeem/reset-claim endpoint.
- The sidecar can start and stop a Codex session for `tenant_id=codex-redeem-test` when authenticated.
- Reset-claim/redeem is a `prodex run` feature, not a sidecar adapter feature.
- Validating actual reset-claim behavior requires a real PROD `prodex run` session with the relevant Codex account state.
- No `prodex-sidecar/` files were edited.
