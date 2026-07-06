# W2-A2 Smart Context REAL Metrics

Agent: Codex#5.5#A  
Date: 2026-07-05  
Target: local loopback sidecar on `http://127.0.0.1:43292`  
Check-in: `.deploy-control/Codex-5.5-A__W2-A2-SMART-CONTEXT-REAL__20260705T225752Z.md`

## Scope

Requested validation:

1. Use the A1 harness against the recompiled sidecar.
2. Run `scripts/smoke/smart-context-measure.sh` or direct curl against port `43292` with a `64KiB+` payload.
3. Verify Smart Context before/after token metrics.
4. Apply `smart_context` kill switch and confirm exact fallback mode.
5. Do not edit `prodex-sidecar/`.

`prodex-sidecar/` was not edited in this task.

## Setup

No sidecar was already listening on port `43292`, so a local fake upstream and the recompiled sidecar were started for this evidence run:

```text
fake upstream: 127.0.0.1:43290
gateway listen: 127.0.0.1:43291
sidecar bind: 127.0.0.1:43292
bearer token: <redacted test token>
```

The fake upstream returned a small OpenAI-compatible JSON body with `usage.input_tokens=64`. No real provider call was made.

## A1 Harness Result

Command shape:

```text
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/smart-context-measure.sh --execute \
  --base-url http://127.0.0.1:43292 \
  --context-kib 64 \
  --session-id session-w2-a2-harness
```

Result:

```json
{
  "REQUEST_BYTES": 67165,
  "RESPONSE_BYTES": 503,
  "CONTEXT_TOKENS_BEFORE": 16384,
  "CONTEXT_TOKENS_AFTER": 16384,
  "compression_ratio": 1.0,
  "tokens_saved": 0,
  "fallback_triggered": false,
  "response_summary": {
    "contract_version": "rpp.l2.v1",
    "router_owner": "rust_l2",
    "runtime_session_id_present": true,
    "event_stream_url_present": true,
    "smart_context_mode": "proxy_rewrite"
  },
  "metric_sources": {
    "CONTEXT_TOKENS_BEFORE": "payload.smart_context_probe.context_tokens_before_estimate",
    "CONTEXT_TOKENS_AFTER": "inferred_no_response_token_metric",
    "fallback_triggered": "inferred_from_smart_context_mode"
  }
}
```

Interpretation: the A1 harness successfully hit `StartSession` on port `43292` and observed `smart_context_mode=proxy_rewrite`. `StartSession` does not include native token counters, so the A1 harness metrics remain inference-backed for this endpoint.

## Runtime Proxy Metrics

Direct proxy request:

```text
POST /v1/runtime/proxy?session_id=session-w2-a2-harness
payload: 64KiB+ OpenAI-compatible request envelope
```

Scrubbed Smart Context metrics from the response:

```json
{
  "smart_context": {
    "mode": "proxy_rewrite",
    "input_tokens_before_estimate": 21889,
    "input_tokens_after_observed_or_estimate": 64,
    "input_token_reduction_percent": 99,
    "measurement_source": "gateway_usage"
  },
  "gateway_status": 200,
  "router_owner": "rust_l2"
}
```

Metric interpretation:

```text
input tokens before: 21889
input tokens after: 64
tokens saved: 21825
reduction: 99%
measurement source: gateway_usage
```

Field-name note: the response did not expose the exact requested field names `smart_context.input_tokens_before` and `smart_context.input_tokens_after`. It exposed `smart_context.input_tokens_before_estimate` and `smart_context.input_tokens_after_observed_or_estimate`, plus the requested `smart_context.input_token_reduction_percent`.

## Exact Fallback

Kill switch command shape:

```text
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/kill-switch-smoke.sh --execute \
  --base-url http://127.0.0.1:43292 \
  --feature smart_context
```

Result:

```text
[kill-switch-smoke] PASS
```

Status check with explicit session:

```json
{
  "active": true,
  "contract_version": "rpp.l2.v1",
  "feature": "smart_context",
  "tenant_id": "tenant-smoke",
  "provider": "codex",
  "profile_id": "codex-smoke-main",
  "session_id": "session-w2-a2-after-kill"
}
```

Repeat A1 harness after kill switch:

```json
{
  "REQUEST_BYTES": 67171,
  "RESPONSE_BYTES": 504,
  "CONTEXT_TOKENS_BEFORE": 16384,
  "CONTEXT_TOKENS_AFTER": 16384,
  "compression_ratio": 1.0,
  "tokens_saved": 0,
  "fallback_triggered": true,
  "response_summary": {
    "contract_version": "rpp.l2.v1",
    "router_owner": "rust_l2",
    "runtime_session_id_present": true,
    "event_stream_url_present": true,
    "smart_context_mode": "exact"
  },
  "metric_sources": {
    "CONTEXT_TOKENS_BEFORE": "payload.smart_context_probe.context_tokens_before_estimate",
    "CONTEXT_TOKENS_AFTER": "inferred_exact_pass_through",
    "fallback_triggered": "inferred_from_smart_context_mode"
  }
}
```

Control-plane result: `StartSession` mode changed from `proxy_rewrite` to `exact`, and the A1 harness inferred `fallback_triggered=true`.

Repeat runtime proxy after kill switch:

```json
{
  "smart_context": {
    "mode": "proxy_rewrite",
    "input_tokens_before_estimate": 31514,
    "input_tokens_after_observed_or_estimate": 64,
    "input_token_reduction_percent": 99,
    "measurement_source": "gateway_usage"
  },
  "gateway_status": 200,
  "router_owner": "rust_l2"
}
```

Data-plane gap: after the kill switch, `StartSession` reports `smart_context_mode=exact`, but `POST /v1/runtime/proxy` still reports `smart_context.mode=proxy_rewrite` and continues to return reduction metrics. This means exact fallback is visible at session start, but not consistently enforced/reported by the runtime proxy response in this build.

## Verdict

Partial / red:

- PASS: A1 harness runs against the recompiled sidecar on `43292`.
- PASS: direct runtime proxy with `64KiB+` payload returns real before/after token metrics from `gateway_usage`.
- PASS: `input_token_reduction_percent=99` observed.
- PASS: kill switch changes new `StartSession` from `proxy_rewrite` to `exact`.
- GAP: token fields are not named exactly as requested; before/after are exposed as estimate/observed-or-estimate names.
- GAP: runtime proxy response still reports `smart_context.mode=proxy_rewrite` after exact fallback is active at session start.

## Cleanup

The temporary sidecar process exited before the final field-name recheck. The fake upstream was stopped with `Ctrl-C`. No provider credentials, raw provider payloads, or bearer token values were written to evidence.
