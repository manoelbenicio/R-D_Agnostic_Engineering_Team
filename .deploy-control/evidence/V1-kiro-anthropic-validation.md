# V1.1 Kiro/Anthropic smart_context validation

- timestamp_utc: 2026-07-06T00:56:05Z
- milestone: v2.1 P11 vendor validation
- agent: Codex#C
- task: V1.1 Kiro/Anthropic smart_context via proxy
- sidecar: `http://127.0.0.1:43292`
- secrets_present: false

## Check-in before

- repository: `/mnt/c/VMs/Projects/Automonous_Agentic`
- branch: `main`
- head: `c35bd78`
- initial dirty state before this task:
  - `M CHECKIN_OUT.md`
  - `?? .codex/config.toml`
  - `?? .deploy-control/`
  - `?? scripts/smoke/`
- `prodex-sidecar/` was not edited.

## Preflight

Observed listeners:

```text
127.0.0.1:43290 fake upstream
127.0.0.1:43291 prodex gateway
127.0.0.1:43292 prodex-sidecar
```

`GET /readyz` returned HTTP 200 with:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "ready",
  "checks": [
    {"name": "shared_state_backend", "status": "pass", "details": {"backend_type": "postgres", "connection_status": "ok"}},
    {"name": "kill_switch", "status": "pass"},
    {"name": "runtime_proxy", "status": "pass", "details": {"listen_addr": "127.0.0.1:43291"}},
    {"name": "event_stream", "status": "pass"}
  ]
}
```

## Step 1 - session start

Requested operation:

```text
POST /v1/session/start
```

Request body shape:

```json
{
  "contract_version": "rpp.l2.v1",
  "tenant_id": "v21-kiro-test",
  "provider": "anthropic",
  "requested_provider": "anthropic",
  "session_id": "v21-kiro-anthropic-1783299359"
}
```

Notes:

- `provider=anthropic` was included as requested.
- `requested_provider=anthropic` was also included because this sidecar build uses `requested_provider` for provider selection.
- `contract_version` was included because this sidecar rejects POST bodies without it.

Observed response summary:

```json
{
  "http_status": 200,
  "extracted_session_id": "v21-kiro-anthropic-1783299359",
  "runtime_endpoint_present": true,
  "router_owner": "rust_l2",
  "runtime_session_id_present": true,
  "smart_context_mode": "proxy_rewrite"
}
```

## Step 2 - Messages API proxy

Requested operation:

```text
POST /v1/runtime/proxy?session_id=v21-kiro-anthropic-1783299359
```

Request body shape:

```json
{
  "model": "claude-sonnet-4-20250514",
  "messages": [
    {
      "role": "user",
      "content": "<16KiB repeated text>"
    }
  ],
  "max_tokens": 1
}
```

Payload size:

```text
content_bytes=16384
```

Observed proxy response summary:

```json
{
  "http_status": 200,
  "gateway_status": 404,
  "router_owner": "rust_l2",
  "smart_context": {
    "mode": "proxy_rewrite",
    "gateway_addr": "127.0.0.1:43291",
    "input_tokens_before_estimate": 4119,
    "input_tokens_after_observed_or_estimate": 8,
    "input_token_reduction_percent": 99,
    "measurement_source": "local_estimate"
  },
  "tokens_saved": 4111
}
```

Computed metric:

```text
tokens_saved = input_tokens_before_estimate - input_tokens_after_observed_or_estimate
tokens_saved = 4119 - 8 = 4111
```

## Event stream confirmation

`GET /v1/events/stream?session_id=v21-kiro-anthropic-1783299359` returned:

```json
[
  {
    "event_type": "session_started",
    "provider": "anthropic",
    "profile_id": "kiro-anthropic-profile-A",
    "redaction_secrets_present": false
  },
  {
    "event_type": "route_decision",
    "provider": "anthropic",
    "profile_id": "kiro-anthropic-profile-A",
    "message": "smart_context tokens before=4119 after=8 reduction_percent=99",
    "redaction_secrets_present": false
  }
]
```

## Responses API fallback check

Responses API shape was not executed because the Messages API run produced `tokens_saved=4111`, satisfying the requested validation gate (`tokens_saved > 0`).

## Verdict

KIRO validated for the requested smart_context proxy gate:

- PASS: `POST /v1/session/start` returned HTTP 200.
- PASS: extracted session id: `v21-kiro-anthropic-1783299359`.
- PASS: `POST /v1/runtime/proxy` accepted a Messages API-shaped body with 16KiB content.
- PASS: smart_context returned positive savings: `tokens_saved=4111`.
- PASS: event stream confirmed `provider=anthropic` and redaction clean.
- NOTE: inner `gateway_status=404`, so this run validates smart_context metric/rewrite behavior through the proxy, not a successful Anthropic provider completion.

No files under `prodex-sidecar/` were edited.
