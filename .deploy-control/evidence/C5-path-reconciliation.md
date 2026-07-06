# C5 Path Reconciliation

agent: Codex#5.5#B
stream: C5-PATH-RECONCILIATION
timestamp_utc: 2026-07-06T00:20:04Z
status: PASS

## Root Cause

The previous `scripts/smoke/smart-context-measure.sh` measured Smart Context on:

```text
POST /v1/session/start
```

That endpoint is lifecycle/control-plane only. It creates the runtime session and returns the real runtime endpoint:

```text
runtime_endpoint=http://<sidecar>/v1/runtime/proxy?session_id=<session>
```

It does **not** proxy model traffic through the prodex gateway. Therefore measuring `tokens_saved` only from `/v1/session/start` was structurally wrong and could report `0` even when Smart Context works.

The real traffic path is:

```text
/v1/session/start        -> lifecycle/session creation
/v1/runtime/proxy        -> real provider/gateway traffic
gateway default path     -> /v1/responses
```

This matches `main.rs` behavior: session start returns `runtime_endpoint`, and runtime proxy defaults `gateway_path` to `/v1/responses` when omitted.

## Script Change

Updated:

```text
scripts/smoke/smart-context-measure.sh
```

Behavior after correction:

1. POST a small lifecycle payload to `/v1/session/start`.
2. Read `runtime_endpoint` from the start response.
3. POST a real runtime envelope to `/v1/runtime/proxy`.
4. Put a valid Responses API body in `body`:

```json
{
  "model": "gpt-4.1",
  "instructions": "system prompt: preserve identifiers exactly, keep JSON/control fields exact, and answer with a short confirmation.",
  "input": [
    {"role": "user", "content": "<large repeated context>"}
  ],
  "max_output_tokens": 1
}
```

5. Read metrics from the runtime proxy response:
   - `smart_context.input_tokens_before_estimate`
   - `smart_context.input_tokens_after_observed_or_estimate`
   - `smart_context.mode`

## Corrected Smoke Run

Command:

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=dev \
L2_BEARER_TOKEN=d2-smoke-token \
L2_BASE_URL=http://127.0.0.1:43292 \
bash scripts/smoke/smart-context-measure.sh \
  --execute \
  --base-url http://127.0.0.1:43292 \
  --context-kib 64 \
  --session-id c5-path-reconciliation-64k
```

Raw output:

```json
{"CONTEXT_TOKENS_AFTER": 8, "CONTEXT_TOKENS_BEFORE": 16659, "REQUEST_BYTES": 66875, "RESPONSE_BYTES": 574, "compression_ratio": 0.00048, "fallback_triggered": false, "metric_sources": {"CONTEXT_TOKENS_AFTER": "smart_context.input_tokens_after_observed_or_estimate", "CONTEXT_TOKENS_BEFORE": "smart_context.input_tokens_before_estimate", "fallback_triggered": "inferred_from_smart_context_mode"}, "response_summary": {"contract_version": "rpp.l2.v1", "event_stream_url_present": false, "router_owner": "rust_l2", "runtime_session_id_present": true, "smart_context_mode": "proxy_rewrite"}, "target": {"base_url": "http://127.0.0.1:43292", "path": "/v1/runtime/proxy?session_id=c5-path-reconciliation-64k", "session_id": "c5-path-reconciliation-64k"}, "tokens_saved": 16651}
```

## Result

Corrected smoke proves `tokens_saved>0` on the real runtime path:

| context_kib | target_path | tokens_before | tokens_after | tokens_saved | compression_ratio | smart_context_mode |
|---:|---|---:|---:|---:|---:|---|
| 64 | `/v1/runtime/proxy?session_id=c5-path-reconciliation-64k` | 16,659 | 8 | 16,651 | 0.00048 | `proxy_rewrite` |

Verdict: PASS. The C5 measurement path is now reconciled with the runtime architecture: `/v1/session/start` is lifecycle, `/v1/runtime/proxy` is real Smart Context traffic.
