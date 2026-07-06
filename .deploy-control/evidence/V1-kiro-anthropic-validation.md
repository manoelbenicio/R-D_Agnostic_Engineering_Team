# V1.1 Kiro/Anthropic Smart Context Validation

- timestamp_utc: 2026-07-06T01:00:33Z
- milestone: v2.1 P11 vendor validation
- agent: Codex#C
- task: V1.1 Kiro/Anthropic smart_context via proxy
- sidecar: `http://127.0.0.1:43292`
- method: `curl`
- secrets_present: false

## Check-in Before

- repository: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team`
- branch: `main`
- head: `014d8e9`
- initial dirty state before this task:
  - `D .planning/phases/v2.1-roadmap/ROADMAP.md`
  - `M docs/vendors/vendor-capability-matrix.md`
  - `?? .deploy-control/Codex-5.5-B__V1-CLINE-OPENCODE-DISPOSITION__20260706T005919Z.md`
  - `?? .deploy-control/Codex-5.5-B__V1-CLINE-OPENROUTER-VALIDATION__20260706T005515Z.md`
  - `?? .deploy-control/Codex-5.5-B__V1-SMART-CONTEXT-PER-VENDOR__20260706T004950Z.md`
  - `?? .deploy-control/evidence/V1-cline-openrouter-validation.md`
  - `?? .deploy-control/evidence/V1-smart-context-per-vendor.md`
  - `?? RELATORIO_CHECKINS.md`
  - `?? TL_chat_history.md`
  - `?? bin/`
  - `?? dashboardatual.png`
  - `?? openspec/changes/rotation-router/`
  - `?? scripts/deploy/`
  - `?? start`
- `multica-auth-work/prodex-sidecar/` was not edited.

## Preflight

Observed listeners before the validation:

```text
127.0.0.1:43290 fake upstream
127.0.0.1:43291 prodex gateway
127.0.0.1:43292 prodex-sidecar
```

`GET /readyz` returned HTTP 200:

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

## Step 1 - Session Start

Command shape:

```text
curl -sS -H 'Authorization: Bearer <redacted>' \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/v1-kiro-start.json \
  http://127.0.0.1:43292/v1/session/start
```

Request body shape:

```json
{
  "contract_version": "rpp.l2.v1",
  "tenant_id": "v21-kiro-test",
  "provider": "anthropic",
  "requested_provider": "anthropic",
  "session_id": "v21-kiro-anthropic-curl-1783299615",
  "profile_pool": ["kiro-anthropic-profile-A", "kiro-anthropic-profile-B"]
}
```

Notes:

- `provider=anthropic` was included as requested.
- `requested_provider=anthropic` was also included because this sidecar build uses `requested_provider` for provider selection and event attribution.
- `contract_version=rpp.l2.v1` was included because this sidecar rejects POST bodies without it.
- `session_id` was supplied so `runtime/proxy?session_id=X` could be deterministic; it was also extracted back from `runtime_endpoint`.

Observed result:

```text
session_start_http=200
extracted_session_id=v21-kiro-anthropic-curl-1783299615
```

Response summary:

```json
{
  "contract_version": "rpp.l2.v1",
  "router_owner": "rust_l2",
  "runtime_endpoint_present": true,
  "runtime_session_id_present": true,
  "smart_context_mode": "proxy_rewrite"
}
```

## Step 2 - Messages API Proxy

Command shape:

```text
curl -sS -H 'Authorization: Bearer <redacted>' \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/v1-kiro-messages.json \
  'http://127.0.0.1:43292/v1/runtime/proxy?session_id=v21-kiro-anthropic-curl-1783299615'
```

Request body shape:

```json
{
  "model": "claude-sonnet-4-20250514",
  "messages": [
    {
      "role": "user",
      "content": "<BLOCO_16KIB>"
    }
  ],
  "max_tokens": 1
}
```

Payload size:

```text
block_bytes=16384
```

Observed result:

```text
messages_proxy_http=200
messages_tokens_saved=4111
responses_shape_tested=false
```

Response summary:

```json
{
  "contract_version": "rpp.l2.v1",
  "gateway_status": 404,
  "router_owner": "rust_l2",
  "smart_context": {
    "mode": "proxy_rewrite",
    "gateway_addr": "127.0.0.1:43291",
    "input_tokens_before_estimate": 4119,
    "input_tokens_after_observed_or_estimate": 8,
    "input_token_reduction_percent": 99,
    "measurement_source": "local_estimate"
  }
}
```

Computed metric:

```text
tokens_saved = input_tokens_before_estimate - input_tokens_after_observed_or_estimate
tokens_saved = 4119 - 8 = 4111
```

## Step 3 - Responses API Conditional

Responses API shape was not executed because the Messages API run produced `tokens_saved=4111`, satisfying the requested conditional gate.

If Messages had returned `tokens_saved=0`, the prepared fallback body shape was:

```json
{
  "model": "claude-sonnet-4-20250514",
  "input": [
    {
      "role": "user",
      "content": "<BLOCO_16KIB>"
    }
  ]
}
```

## Event Stream Confirmation

`GET /v1/events/stream?session_id=v21-kiro-anthropic-curl-1783299615` returned:

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

## Verdict

KIRO validated for the requested Smart Context proxy gate:

- PASS: `POST /v1/session/start` returned HTTP 200.
- PASS: session id extracted: `v21-kiro-anthropic-curl-1783299615`.
- PASS: `POST /v1/runtime/proxy` accepted a Messages API-shaped body with 16KiB repeated content.
- PASS: `tokens_saved=4111`, so the validation condition `tokens_saved > 0` is satisfied.
- PASS: event stream confirmed `provider=anthropic`, `profile_id=kiro-anthropic-profile-A`, and `redaction.secrets_present=false`.
- NOTE: inner `gateway_status=404`; this validates Smart Context metric/rewrite behavior through the proxy, not a successful Anthropic provider completion from the local fake upstream.

No files under `multica-auth-work/prodex-sidecar/` were edited.
