# W3-D3 Kill Switch + Rollback Retest

agent: Codex#5.5#B
stream: W3-D3-KILLSWITCH-ROLLBACK
timestamp_utc: 2026-07-06T00:15:15Z
status: PARTIAL

## Scope

Retest against the already-running sidecar:

```text
http://127.0.0.1:43292
```

Bearer token was used for local auth but is intentionally omitted from evidence.
No `prodex-sidecar/` files were edited.

## Kill Switch Apply

```text
POST /v1/killswitch/apply
HTTP 200
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"next_request","request_id":"req-d3-kill-on"}
```

Verdict: `applied=true` confirmed.

## Kill Switch Status

```text
GET /v1/killswitch/status?tenant_id=tenant-d3&feature=smart_context&provider=codex&profile_id=codex-d3-main&session_id=w3-d3-killswitch-rollback
HTTP 200
{"active":true,"contract_version":"rpp.l2.v1","feature":"smart_context","profile_id":"codex-d3-main","provider":"codex","session_id":"w3-d3-killswitch-rollback","tenant_id":"tenant-d3"}
```

Verdict: `active=true` confirmed.

## Session Start Under Kill Switch

Session was started after applying the session-scoped kill switch so runtime proxy had an active session.

```text
POST /v1/session/start
HTTP 200
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43292/v1/events/stream?session_id=w3-d3-killswitch-rollback","gateway":{"listen_addr":"127.0.0.1:43291","smart_context_enabled":true},"request_id":"req-d3-start","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:43292/v1/runtime/proxy?session_id=w3-d3-killswitch-rollback","runtime_log_ref":"prodex-gateway://127.0.0.1:43291","runtime_session_id":"rt-1783296842640045620","smart_context_mode":"exact"}
```

Verdict: session creation reflects the kill switch with `smart_context_mode="exact"`.

## Runtime Proxy With Kill Switch Active

```text
POST /v1/runtime/proxy?session_id=w3-d3-killswitch-rollback
HTTP 200
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-d3-proxy-kill-active","router_owner":"rust_l2","runtime_request_id":"rt-d3-kill-active","runtime_session_id":"rt-1783296842640045620","session_id":"w3-d3-killswitch-rollback","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":77,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":35,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

Verdict: `runtime/proxy` did **not** reflect the active Smart Context kill switch in `smart_context.mode`; it remained `proxy_rewrite`. This is a behavioral gap in the proxy response surface, even though `session/start` reflected `exact`.

## Rollback Apply

Rollback used `state="enabled"` for the existing API; this is the active-false equivalent because the sidecar removes the stored disabled switch.

```text
POST /v1/killswitch/apply
HTTP 200
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"next_request","request_id":"req-d3-kill-off"}
```

Verdict: rollback apply confirmed.

## Rollback Status

```text
GET /v1/killswitch/status?tenant_id=tenant-d3&feature=smart_context&provider=codex&profile_id=codex-d3-main&session_id=w3-d3-killswitch-rollback
HTTP 200
{"active":false,"contract_version":"rpp.l2.v1","feature":"smart_context","profile_id":"codex-d3-main","provider":"codex","session_id":"w3-d3-killswitch-rollback","tenant_id":"tenant-d3"}
```

Verdict: `active=false` confirmed after rollback.

## Session Start After Rollback

```text
POST /v1/session/start
HTTP 200
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43292/v1/events/stream?session_id=w3-d3-killswitch-rollback","gateway":{"listen_addr":"127.0.0.1:43291","smart_context_enabled":true},"request_id":"req-d3-start-after-rollback","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:43292/v1/runtime/proxy?session_id=w3-d3-killswitch-rollback","runtime_log_ref":"prodex-gateway://127.0.0.1:43291","runtime_session_id":"rt-1783296842711593100","smart_context_mode":"proxy_rewrite"}
```

Verdict: session creation returned to normal `smart_context_mode="proxy_rewrite"` after rollback.

## Runtime Proxy After Rollback

```text
POST /v1/runtime/proxy?session_id=w3-d3-killswitch-rollback
HTTP 200
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-d3-proxy-after-rollback","router_owner":"rust_l2","runtime_request_id":"rt-d3-after-rollback","runtime_session_id":"rt-1783296842711593100","session_id":"w3-d3-killswitch-rollback","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":77,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":35,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

Verdict: `runtime/proxy` stayed `proxy_rewrite`, same as during kill-switch-active state.

## Readyz Falsification: Postgres Up

```text
HTTP/1.1 200 OK
Server: tiny-http (Rust)
Date: Mon, 06 Jul 2026 00:14:29 GMT
Content-Type: application/json
Content-Length: 385

{"checks":[{"details":{"backend_type":"postgres","configured":true,"connection_status":"ok","probe":"SELECT 1"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"details":{"listen_addr":"127.0.0.1:43291","pid":188565},"name":"runtime_proxy","status":"pass"},{"name":"event_stream","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

## Readyz Falsification: Postgres Stopped

Action:

```text
docker stop deploy-postgres-1
```

Output:

```text
deploy-postgres-1
```

`/readyz` while Postgres was stopped:

```text
HTTP/1.1 503 Service Unavailable
Server: tiny-http (Rust)
Date: Mon, 06 Jul 2026 00:14:50 GMT
Content-Type: application/json
Content-Length: 350

{"checks":[{"details":{"connection_status":"error","error":"connect_failed"},"name":"shared_state_backend","status":"fail"},{"name":"kill_switch","status":"pass"},{"details":{"listen_addr":"127.0.0.1:43291","pid":188565},"name":"runtime_proxy","status":"pass"},{"name":"event_stream","status":"pass"}],"contract_version":"rpp.l2.v1","status":"error"}
```

Verdict: PASS. `/readyz` did not falsify readiness; it returned `503 Service Unavailable` and `status="error"` when Postgres was unavailable.

## Postgres Restored

Action:

```text
docker start deploy-postgres-1
```

Output:

```text
deploy-postgres-1
```

`/readyz` after restore:

```text
HTTP/1.1 200 OK
Server: tiny-http (Rust)
Date: Mon, 06 Jul 2026 00:15:15 GMT
Content-Type: application/json
Content-Length: 385

{"checks":[{"details":{"backend_type":"postgres","configured":true,"connection_status":"ok","probe":"SELECT 1"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"details":{"listen_addr":"127.0.0.1:43291","pid":188565},"name":"runtime_proxy","status":"pass"},{"name":"event_stream","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

## Summary

- Kill switch apply: PASS (`applied=true`).
- Kill switch status active: PASS (`active=true`).
- Session start under kill switch: PASS (`smart_context_mode="exact"`).
- Runtime proxy mode under kill switch: GAP (`smart_context.mode` stayed `proxy_rewrite`).
- Rollback apply/status: PASS (`active=false` after rollback).
- Session start after rollback: PASS (`smart_context_mode="proxy_rewrite"`).
- Readyz falsification: PASS (`503` when Postgres stopped, `200` after restore).
