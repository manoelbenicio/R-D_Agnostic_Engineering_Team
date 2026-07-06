# C5 Smoke Fix Reconciliation

agent: Codex#5.5#B
stream: C5-PATH-RECONCILIATION
timestamp_utc: 2026-07-06T00:21:11Z
status: PASS

## What Was Fixed

`scripts/smoke/smart-context-measure.sh` now measures the real Smart Context traffic path.

Before:

```text
POST /v1/session/start
```

This was incorrect for Smart Context measurement because `/v1/session/start` is lifecycle/control-plane only. It creates a session and returns a runtime endpoint URL. It does not proxy provider traffic through the prodex gateway.

After:

```text
POST /v1/session/start
POST /v1/runtime/proxy?session_id=<extracted session_id>
```

The script now:

1. Sends a minimum lifecycle body to `/v1/session/start`.
2. Extracts `session_id` from the returned `runtime_endpoint`.
3. Posts to `/v1/runtime/proxy?session_id=X`.
4. Uses a valid Responses API body inside the runtime envelope:

```json
{
  "model": "gpt-4.1",
  "input": [
    {"role": "user", "content": "CONTEXTO_GRANDE"}
  ],
  "instructions": "system prompt: preserve identifiers exactly, keep JSON/control fields exact, and answer with a short confirmation.",
  "max_output_tokens": 1
}
```

5. Reads metrics from the runtime proxy response:
   - `smart_context.input_tokens_before_estimate`
   - `smart_context.input_tokens_after_observed_or_estimate`

## Corrected Smoke Command

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=dev \
L2_BEARER_TOKEN=d2-smoke-token \
L2_BASE_URL=http://127.0.0.1:43292 \
bash scripts/smoke/smart-context-measure.sh \
  --execute \
  --base-url http://127.0.0.1:43292 \
  --context-kib 64 \
  --session-id c5-smoke-fix-64k
```

## Raw Output

```json
{"CONTEXT_TOKENS_AFTER": 8, "CONTEXT_TOKENS_BEFORE": 16659, "REQUEST_BYTES": 66845, "RESPONSE_BYTES": 544, "compression_ratio": 0.00048, "fallback_triggered": false, "metric_sources": {"CONTEXT_TOKENS_AFTER": "smart_context.input_tokens_after_observed_or_estimate", "CONTEXT_TOKENS_BEFORE": "smart_context.input_tokens_before_estimate", "fallback_triggered": "inferred_from_smart_context_mode"}, "response_summary": {"contract_version": "rpp.l2.v1", "event_stream_url_present": false, "router_owner": "rust_l2", "runtime_session_id_present": true, "smart_context_mode": "proxy_rewrite"}, "target": {"base_url": "http://127.0.0.1:43292", "path": "/v1/runtime/proxy?session_id=c5-smoke-fix-64k", "session_id": "c5-smoke-fix-64k"}, "tokens_saved": 16651}
```

## Result

| Field | Value |
|---|---:|
| target path | `/v1/runtime/proxy?session_id=c5-smoke-fix-64k` |
| request bytes | 66,845 |
| response bytes | 544 |
| tokens before | 16,659 |
| tokens after | 8 |
| tokens saved | 16,651 |
| compression ratio | 0.00048 |

Verdict: PASS. The corrected smoke now proves `tokens_saved>0` on the real runtime traffic path.

## Architecture Reconciliation

- `/v1/session/start` = lifecycle endpoint. It creates the runtime session and returns `runtime_endpoint`.
- `/v1/runtime/proxy` = real traffic endpoint. It forwards the Responses API body through the prodex gateway, where Smart Context can rewrite/measure.
- Gateway path default remains `/v1/responses` when `gateway_path` is omitted.
