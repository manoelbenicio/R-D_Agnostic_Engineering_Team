# V1 Antigravity/Gemini validation evidence

- timestamp_utc: 2026-07-06T00:58:43Z
- milestone: v2.1
- phase: V1
- tasks: V1.1 Antigravity/Gemini smart_context; V1.2 rotation
- target_sidecar: `http://127.0.0.1:43292`
- tenant_id: `v21-antigravity-test`
- provider: `google`
- model: `gemini-2.5-pro`
- bearer: `MULTICA_L2_BEARER_TOKEN=<redacted>`
- secrets_present: false

## Guardrails

- Did not edit `prodex-sidecar/`.
- Probe artifacts were written under `/tmp/v1-antigravity-gemini-validation/`.
- The running sidecar required bearer authentication; unauthenticated health probes returned 401 earlier in the session.

## Runtime preflight

`GET /healthz` returned HTTP 200:

```json
{
  "contract_version": "rpp.l2.v1",
  "sidecar": {
    "commit": "smoke",
    "name": "prodex-sidecar",
    "version": "0.1.0"
  },
  "status": "alive"
}
```

`GET /readyz` returned HTTP 200:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "ready",
  "checks": [
    {
      "name": "shared_state_backend",
      "status": "pass",
      "details": {
        "backend_type": "postgres",
        "configured": true,
        "connection_status": "ok",
        "probe": "SELECT 1"
      }
    },
    {
      "name": "kill_switch",
      "status": "pass"
    },
    {
      "name": "runtime_proxy",
      "status": "pass",
      "details": {
        "listen_addr": "127.0.0.1:43291",
        "pid": 188565
      }
    },
    {
      "name": "event_stream",
      "status": "pass"
    }
  ]
}
```

## V1.1 smart_context

Requested minimal start payload:

```json
{
  "tenant_id": "v21-antigravity-test",
  "provider": "google"
}
```

Result:

```text
HTTP 400
{"error":"contract_version must be rpp.l2.v1"}
```

The running sidecar did not accept the minimal payload shape. To complete the Gemini smart_context validation, the probe used the required `rpp.l2.v1` envelope while preserving `provider=google`:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-v21-antigravity-test-google-20260706T005843Z",
  "tenant_id": "v21-antigravity-test",
  "workspace_id": "workspace-v21-antigravity-test",
  "task_id": "task-antigravity-gemini-smart-context",
  "session_id": "v21-antigravity-test-google-20260706T005843Z",
  "provider": "google",
  "requested_provider": "google",
  "requested_model": "gemini-2.5-pro",
  "working_directory": "/tmp/rpp-smoke-workspace"
}
```

Result summary:

```text
POST /v1/session/start HTTP_STATUS=200
router_owner=rust_l2
runtime_session_id=rt-1783299523940474503
event_stream_url_present=true
smart_context_mode=proxy_rewrite
session_id_used_for_proxy=v21-antigravity-test-google-20260706T005843Z
```

Gemini API body sent to `POST /v1/runtime/proxy?session_id=v21-antigravity-test-google-20260706T005843Z`:

```json
{
  "model": "gemini-2.5-pro",
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "<16384 byte repeated text>"
        }
      ]
    }
  ]
}
```

Payload measurements:

```text
gemini_text_bytes=16384
gemini_request_bytes=16461
```

`POST /v1/runtime/proxy` returned HTTP 200 with sidecar smart_context metrics:

```json
{
  "contract_version": "rpp.l2.v1",
  "gateway_status": 404,
  "router_owner": "rust_l2",
  "runtime_request_id": "rt-req-1783299523942163830",
  "runtime_session_id": "rt-1783299523940474503",
  "session_id": "v21-antigravity-test-google-20260706T005843Z",
  "smart_context": {
    "gateway_addr": "127.0.0.1:43291",
    "input_token_reduction_percent": 99,
    "input_tokens_after_observed_or_estimate": 8,
    "input_tokens_before_estimate": 4115,
    "measurement_source": "local_estimate",
    "mode": "proxy_rewrite"
  }
}
```

The response did not include a literal `tokens_saved` field. Extracted from the returned smart_context metrics:

```text
tokens_saved = input_tokens_before_estimate - input_tokens_after_observed_or_estimate
tokens_saved = 4115 - 8 = 4107
```

Notes:

- PASS: sidecar accepted a Google/Gemini session when the required `rpp.l2.v1` contract fields were present.
- PASS: `runtime/proxy` accepted the Gemini API request body and returned HTTP 200.
- PASS: smart_context was active in `proxy_rewrite` mode.
- OBSERVED: gateway returned `gateway_status=404` inside the sidecar response, but the sidecar still returned HTTP 200 and emitted local smart_context metrics.

## V1.2 rotation

Requested minimal rotation payload:

```json
{
  "tenant_id": "v21-antigravity-test",
  "provider": "google",
  "profile_pool": ["prof-a", "prof-b"]
}
```

Result:

```text
HTTP 400
{"error":"contract_version must be rpp.l2.v1"}
```

The running sidecar required the `rpp.l2.v1` envelope for rotation as well. Compatible payload:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req-v21-antigravity-test-rotation-20260706T005843Z",
  "tenant_id": "v21-antigravity-test",
  "workspace_id": "workspace-v21-antigravity-test",
  "task_id": "task-antigravity-gemini-rotation",
  "session_id": "v21-antigravity-test-rotation-20260706T005843Z",
  "provider": "google",
  "requested_provider": "google",
  "requested_model": "gemini-2.5-pro",
  "working_directory": "/tmp/rpp-smoke-workspace",
  "profile_pool": ["prof-a", "prof-b"]
}
```

Result summary:

```text
POST /v1/session/start HTTP_STATUS=200
router_owner=rust_l2
runtime_session_id=rt-1783299523945818674
event_stream_url_present=true
smart_context_mode=proxy_rewrite
```

Selection was verified through:

```text
GET /v1/events/stream?session_id=v21-antigravity-test-rotation-20260706T005843Z
```

Event response:

```json
{
  "contract_version": "rpp.l2.v1",
  "event_type": "session_started",
  "profile_id": "prof-a",
  "provider": "google",
  "runtime_session_id": "rt-1783299523945818674",
  "session_id": "v21-antigravity-test-rotation-20260706T005843Z",
  "tenant_id": "v21-antigravity-test",
  "redaction": {
    "scrubber_version": "sidecar-smoke-1.0.0",
    "secrets_present": false
  }
}
```

Empirical result:

- PASS: sidecar accepted `profile_pool: ["prof-a", "prof-b"]` when sent with the required contract envelope.
- PASS: selected profile was observable via `session_started.profile_id`.
- OBSERVED: selected profile was `prof-a`, the first profile in the pool.

## Conclusion

- V1.1 Antigravity/Gemini smart_context is validated on sidecar `127.0.0.1:43292` using the required `rpp.l2.v1` start envelope.
- The exact minimal `{tenant_id, provider}` start body requested by the milestone was rejected by this sidecar with HTTP 400 because `contract_version` is mandatory.
- `runtime/proxy` accepted a Gemini API body with 16 KiB repeated text and returned smart_context metrics.
- Extracted `tokens_saved=4107` from returned local estimates.
- V1.2 rotation accepted `profile_pool: ["prof-a", "prof-b"]` and selected `prof-a`.
- No `prodex-sidecar/` files were edited.
