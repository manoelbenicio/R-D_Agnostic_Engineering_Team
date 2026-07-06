# W2-B2 Real Events Evidence

timestamp_utc: 2026-07-05T23:21:39Z
agent: Codex#5.5#B
stream: W2-B2-REAL-EVENTS
sidecar: multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
base_url: http://127.0.0.1:43292
fake_upstream: http://127.0.0.1:43290
gateway: 127.0.0.1:43291
session_id: b2-real-events-20260705T2313Z

## Golden Rule Check-In

- Active check-in used: `.deploy-control/Codex-5.5-B__W2-B2-REAL-EVENTS__20260705T225809Z.md`
- The check-in already locked:
  - `multica-auth-work/prodex-sidecar/**`
  - `.deploy-control/Codex-5.5-B__W2-B2-REAL-EVENTS__20260705T225809Z.md`
  - `.deploy-control/evidence/W2-B2-real-events.md`

## Runtime Setup

- Existing 43292 listener was rejected as evidence because `/proc/<pid>/exe` pointed to `multica-auth-work/prodex-sidecar/target/release/prodex-sidecar (deleted)`.
- Restarted the sidecar with the current on-disk binary:
  - `MULTICA_L2_BEARER_TOKEN=<redacted-test-token> PRODEX_GATEWAY_UPSTREAM_BASE_URL=http://127.0.0.1:43290 PRODEX_GATEWAY_LISTEN=127.0.0.1:43291 multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43292`
- Verified current sidecar executable:
  - `/proc/119054/exe -> /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- Fake upstream was started on `127.0.0.1:43290` using Python `http.server`.
- `GET /healthz` returned `200 OK` with `contract_version=rpp.l2.v1`.

## Session Calls

All calls used `Authorization: Bearer <redacted-test-token>`.

### POST /v1/session/start

Result: `200 OK`

Key response fields:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-b2-start",
  "router_owner": "rust_l2",
  "runtime_session_id": "rt-1783293509024426025",
  "event_stream_url": "http://127.0.0.1:43292/v1/events/stream?session_id=b2-real-events-20260705T2313Z",
  "runtime_endpoint": "http://127.0.0.1:43292/v1/runtime/proxy?session_id=b2-real-events-20260705T2313Z",
  "smart_context_mode": "proxy_rewrite",
  "gateway": {
    "listen_addr": "127.0.0.1:43291",
    "smart_context_enabled": true
  }
}
```

### POST /v1/runtime/proxy

Result: `200 OK`

Key response fields:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-b2-proxy",
  "session_id": "b2-real-events-20260705T2313Z",
  "runtime_session_id": "rt-1783293509024426025",
  "runtime_request_id": "rt-req-b2-real",
  "router_owner": "rust_l2",
  "gateway_status": 200,
  "smart_context": {
    "mode": "proxy_rewrite",
    "gateway_addr": "127.0.0.1:43291",
    "input_tokens_before_estimate": 36,
    "input_tokens_after_observed_or_estimate": 64,
    "input_token_reduction_percent": -77,
    "measurement_source": "gateway_usage"
  }
}
```

### POST /v1/killswitch/apply

Result: `200 OK`

Key response fields:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-b2-kill",
  "applied": true,
  "effective_at": "next_request"
}
```

### POST /v1/session/stop

Result: `200 OK`

Key response fields:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-b2-stop",
  "stopped": true
}
```

## GET /v1/events/stream

Command summary:

- `GET http://127.0.0.1:43292/v1/events/stream?session_id=b2-real-events-20260705T2313Z`
- Result: `200 OK`
- Content-Type: `application/x-ndjson`
- Lines returned: 4

### Event 1: session_started

```json
{
  "contract_version": "rpp.l2.v1",
  "event_id": "evt-1783293509024438787",
  "event_type": "session_started",
  "message": "session started with prodex gateway smart context",
  "occurred_at": "2026-07-05T23:18:29.024743904+00:00",
  "producer": {
    "component": "event_stream",
    "plane": "rust_l2"
  },
  "profile_id": "codex-b2-main",
  "provider": "codex",
  "redaction": {
    "scrubber_version": "sidecar-smoke-1.0.0",
    "secrets_present": false
  },
  "runtime_session_id": "rt-1783293509024426025",
  "session_id": "b2-real-events-20260705T2313Z",
  "severity": "info",
  "tenant_id": "tenant-b2"
}
```

Validation:

- `contract_version == rpp.l2.v1`
- `event_type == session_started`
- `redaction.secrets_present == false`
- Runtime session and provider/profile identifiers are present.

### Event 2: route_decision

```json
{
  "contract_version": "rpp.l2.v1",
  "event_id": "evt-1783293602105264311",
  "event_type": "route_decision",
  "message": "smart_context tokens before=36 after=64 reduction_percent=-77",
  "occurred_at": "2026-07-05T23:20:02.105271552+00:00",
  "producer": {
    "component": "runtime_proxy",
    "plane": "rust_l2"
  },
  "profile_id": "codex-b2-main",
  "provider": "codex",
  "redaction": {
    "scrubber_version": "sidecar-smoke-1.0.0",
    "secrets_present": false
  },
  "route_decision": {
    "committed": true,
    "decision_phase": "pre_commit",
    "reason": "prodex_gateway_smart_context",
    "selected_profile_id": "codex-b2-main",
    "selected_provider": "codex"
  },
  "runtime_request_id": "rt-req-b2-real",
  "runtime_session_id": "rt-1783293509024426025",
  "session_id": "b2-real-events-20260705T2313Z",
  "severity": "info",
  "tenant_id": "tenant-b2"
}
```

Validation:

- `contract_version == rpp.l2.v1`
- `event_type == route_decision`
- `redaction.secrets_present == false`
- `route_decision.decision_phase == pre_commit`
- `route_decision.committed == true`
- Smart Context metrics are present in the event message and the proxy response.

### Event 3: killswitch_toggled

```json
{
  "contract_version": "rpp.l2.v1",
  "event_id": "evt-1783293659400745696",
  "event_type": "killswitch_toggled",
  "kill_switch": {
    "enabled": false,
    "feature": "smart_context",
    "scope": "session"
  },
  "message": "kill switch smart_context disabled",
  "occurred_at": "2026-07-05T23:20:59.400748484+00:00",
  "producer": {
    "component": "event_stream",
    "plane": "rust_l2"
  },
  "redaction": {
    "scrubber_version": "sidecar-smoke-1.0.0",
    "secrets_present": false
  },
  "session_id": "b2-real-events-20260705T2313Z",
  "severity": "info",
  "tenant_id": "tenant-b2"
}
```

Validation:

- `contract_version == rpp.l2.v1`
- `event_type == killswitch_toggled`
- `redaction.secrets_present == false`
- `kill_switch.feature == smart_context`
- `kill_switch.enabled == false`
- `kill_switch.scope == session`

### Event 4: session_stopped

```json
{
  "contract_version": "rpp.l2.v1",
  "event_id": "evt-1783293669410189720",
  "event_type": "session_stopped",
  "message": "session stopped",
  "occurred_at": "2026-07-05T23:21:09.410192683+00:00",
  "producer": {
    "component": "event_stream",
    "plane": "rust_l2"
  },
  "redaction": {
    "scrubber_version": "sidecar-smoke-1.0.0",
    "secrets_present": false
  },
  "session_id": "b2-real-events-20260705T2313Z",
  "severity": "info",
  "tenant_id": "tenant-b2"
}
```

Validation:

- `contract_version == rpp.l2.v1`
- `event_type == session_stopped`
- `redaction.secrets_present == false`
- Session ID matches the started session.

## Conclusion

PASS. The rebuilt/current sidecar emitted real NDJSON events from `/v1/events/stream` for the requested session lifecycle:

- `session_started`
- `route_decision`
- `killswitch_toggled`
- `session_stopped`

Every returned event has `contract_version=rpp.l2.v1` and `redaction.secrets_present=false`.
