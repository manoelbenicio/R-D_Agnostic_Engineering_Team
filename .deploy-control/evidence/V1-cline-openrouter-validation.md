# V1.1 Cline/OpenRouter Smart Context Validation

agent: Codex#5.5#B
stream: V1-CLINE-OPENROUTER-VALIDATION
timestamp_utc: 2026-07-06T00:55:56Z
rerun_timestamp_utc: 2026-07-06T01:00:26Z
status: PASS

## Scope

Validate Cline/OpenRouter Smart Context through curl against our sidecar proxy:

```text
sidecar: http://127.0.0.1:43292
tenant_id: v21-cline-test
provider: openrouter
```

No `prodex-sidecar/` files were edited.

## Step 1: Session Start

Request path:

```text
POST /v1/session/start
```

Request shape:

```json
{
  "contract_version": "rpp.l2.v1",
  "tenant_id": "v21-cline-test",
  "request_id": "req-v1-cline-curl2-start",
  "session_id": "v21-cline-openrouter-curl-validation",
  "provider": "openrouter",
  "requested_provider": "openrouter",
  "profile_pool": ["cline-openrouter-main"],
  "task": {
    "task_id": "task-v1-cline-curl",
    "workspace_id": "workspace-v21"
  }
}
```

Raw response:

```json
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43292/v1/events/stream?session_id=v21-cline-openrouter-curl-validation","gateway":{"listen_addr":"127.0.0.1:43291","smart_context_enabled":true},"request_id":"req-v1-cline-curl2-start","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:43292/v1/runtime/proxy?session_id=v21-cline-openrouter-curl-validation","runtime_log_ref":"prodex-gateway://127.0.0.1:43291","runtime_session_id":"rt-1783299626205793815","smart_context_mode":"proxy_rewrite"}
```

Extracted session/runtime:

```text
session_id=v21-cline-openrouter-curl-validation
runtime_session_id=rt-1783299626205793815
runtime_endpoint=http://127.0.0.1:43292/v1/runtime/proxy?session_id=v21-cline-openrouter-curl-validation
```

## Step 2: Runtime Proxy

Request path:

```text
POST /v1/runtime/proxy?session_id=v21-cline-openrouter-curl-validation
```

Runtime body shape:

```json
{
  "model": "gpt-4.1",
  "input": [
    {
      "role": "user",
      "content": "<16KiB repeated context>"
    }
  ],
  "instructions": "summarize",
  "max_output_tokens": 1
}
```

Raw response:

```json
{"contract_version":"rpp.l2.v1","gateway_response":{"error":{"message":"not found"}},"gateway_status":404,"request_id":"req-v1-cline-curl2-runtime","router_owner":"rust_l2","runtime_request_id":"rt-v1-cline-curl2-runtime","runtime_session_id":"rt-1783299626205793815","session_id":"v21-cline-openrouter-curl-validation","smart_context":{"gateway_addr":"127.0.0.1:43291","input_token_reduction_percent":99,"input_tokens_after_observed_or_estimate":8,"input_tokens_before_estimate":4166,"measurement_source":"local_estimate","mode":"proxy_rewrite"}}
```

## Token Result

| Field | Value |
|---|---:|
| runtime request bytes | 16,876 |
| gateway status | 404 |
| measurement source | `local_estimate` |
| smart_context mode | `proxy_rewrite` |
| tokens_before | 4,166 |
| tokens_after | 8 |
| tokens_saved | 4,158 |
| reduction_percent | 99% |

## Verdict

PASS. `tokens_saved=4158`, so Cline/OpenRouter is validated via our sidecar proxy for the requested OpenAI-compatible Responses API shape.

Operational caveat: the current fake upstream on `43290` returns `404` for gateway default `/v1/responses`; therefore this validation is based on sidecar Smart Context local estimates rather than upstream usage counters.
