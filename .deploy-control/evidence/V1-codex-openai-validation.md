# V1.1 Codex/OpenAI smart_context + V1.3 reset_claim evidence

- timestamp_utc: 2026-07-06T00:55:16Z
- milestone: v2.1
- phase: P11 vendor validation
- agent: Codex#A
- target_sidecar: `http://127.0.0.1:43292`
- target_gateway: `127.0.0.1:43291`
- target_fake_upstream: `127.0.0.1:43290`
- tenant_id: `v21-codex-test`
- provider: `openai`
- secrets_present: false

## Runtime processes

Observed before validation:

```text
python3 scripts/smoke/fake-upstream-logging.py --host 127.0.0.1 --port 43290
prodex gateway --listen 127.0.0.1:43291 --base-url http://127.0.0.1:43290 --smart-context
prodex-sidecar 127.0.0.1:43292
```

No `prodex-sidecar/` files were edited.

## Step 1: literal session/start command

Command requested:

```bash
curl -s -X POST http://127.0.0.1:43292/v1/session/start -H 'Content-Type: application/json' -d '{"tenant_id":"v21-codex-test","provider":"openai"}' > /tmp/v21-codex-session.json && cat /tmp/v21-codex-session.json
```

Output:

```json
{"error":"unauthorized"}
```

Result:

- The literal unauthenticated command did not return `runtime_endpoint`.
- A `session_id` could not be extracted from `/tmp/v21-codex-session.json`.
- Runtime proxy validation could not proceed from the literal response.

## Step 2/3: authenticated contract probe for smart_context

Because the sidecar requires bearer auth and `rpp.l2.v1` fields, a separate authenticated contract probe was run without printing the token.

Start payload summary:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_start_v21_codex_openai",
  "tenant_id": "v21-codex-test",
  "workspace_id": "workspace-v21-codex-test",
  "task_id": "task-vendor-validation-codex-openai",
  "session_id": "v21-codex-test-openai-session",
  "policy_id": "policy-smoke-shadow",
  "requested_provider": "openai",
  "requested_model": "gpt-4.1",
  "working_directory": "/tmp/rpp-smoke-workspace",
  "profile_pool": ["codex-smoke-main", "codex-smoke-backup"]
}
```

Start result:

```json
{
  "contract_version": "rpp.l2.v1",
  "event_stream_url": "http://127.0.0.1:43292/v1/events/stream?session_id=v21-codex-test-openai-session",
  "gateway": {
    "listen_addr": "127.0.0.1:43291",
    "smart_context_enabled": true
  },
  "request_id": "req_start_v21_codex_openai",
  "router_owner": "rust_l2",
  "runtime_endpoint": "http://127.0.0.1:43292/v1/runtime/proxy?session_id=v21-codex-test-openai-session",
  "runtime_log_ref": "prodex-gateway://127.0.0.1:43291",
  "runtime_session_id": "rt-1783299353014732457",
  "smart_context_mode": "proxy_rewrite"
}
```

Extracted `session_id` from `runtime_endpoint`:

```text
v21-codex-test-openai-session
```

Runtime proxy request:

```text
POST /v1/runtime/proxy?session_id=v21-codex-test-openai-session
content_bytes=16384
body_shape=Responses API
model=gpt-4.1
instructions=summarize
```

Runtime proxy result:

```json
{
  "contract_version": "rpp.l2.v1",
  "gateway_response": {
    "error": {
      "message": "not found"
    }
  },
  "gateway_status": 404,
  "router_owner": "rust_l2",
  "runtime_request_id": "rt-req-1783299353020505683",
  "runtime_session_id": "rt-1783299353014732457",
  "session_id": "v21-codex-test-openai-session",
  "smart_context": {
    "gateway_addr": "127.0.0.1:43291",
    "input_token_reduction_percent": 99,
    "input_tokens_after_observed_or_estimate": 8,
    "input_tokens_before_estimate": 4117,
    "measurement_source": "local_estimate",
    "mode": "proxy_rewrite"
  }
}
```

Notes:

- The fake upstream on `43290` returned `not found` for the Responses API path through the gateway.
- The sidecar still returned `smart_context` metrics for the proxy rewrite path.

## Step 4: extracted smart_context metrics

```text
input_tokens_before_estimate=4117
input_tokens_after_observed_or_estimate=8
tokens_saved=4109
input_token_reduction_percent=99
metric_source=local_estimate
```

Cleanup:

```text
POST /v1/session/stop HTTP_STATUS=200
stopped=true
```

## Step 5: reset_claim / redeem read-only grep

Command requested from this repository:

```bash
grep -r redeem prodex-sidecar/src/main.rs
```

Output:

```text
grep: prodex-sidecar/src/main.rs: No such file or directory
```

The current repository does not contain a local `prodex-sidecar/` directory. The running sidecar is external:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
```

Read-only grep against the external running sidecar source:

```bash
grep -r redeem /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/src/main.rs
```

Output:

```text
no matches
```

## Conclusion

- V1.1 smart_context is observable on the authenticated sidecar contract path: `mode=proxy_rewrite`, `tokens_saved=4109`, `metric_source=local_estimate`.
- The exact unauthenticated minimal `curl` requested returned `{"error":"unauthorized"}` and did not provide a `runtime_endpoint`.
- V1.3 reset_claim/redeem is not exposed in `prodex-sidecar/src/main.rs` by a `redeem` match.
- No `prodex-sidecar/` files were edited.
