# DIAG Smart Context Compaction

agent: Codex#5.5#B
stream: DIAG-SMART-CONTEXT-COMPACTION
timestamp_utc: 2026-07-05T23:46:19Z
status: PASS

## Root Cause

Smart Context was measured previously with a generic Chat-style body:

```json
{"model":"fake-model","messages":[...]}
```

That body does not exercise the relevant Responses API rewrite path. The gateway help states `--smart-context` applies to `/v1/responses` and `/v1/chat/completions`, and the source path verified is:

- `prodex-context`
- `prodex-app/src/runtime_proxy/smart_context/body.rs`
- `prodex-runtime-proxy/src/smart_context/*`

The sidecar already defaults runtime proxy traffic to `/v1/responses` when `gateway_path` is omitted, so no sidecar path change is required.

## Environment

- fake upstream: `127.0.0.1:43290`
- sidecar: `127.0.0.1:43292`
- sidecar gateway: `127.0.0.1:43291`
- direct gateway without Smart Context: `127.0.0.1:43294`
- direct gateway with Smart Context: `127.0.0.1:43303`
- prodex: `bin/prodex`, version from `prodex info`: `0.246.0`
- profile state from `prodex info`: `Profiles: 0`, `Active profile: -`

## Direct Gateway Measurement

Payload shape:

```json
{
  "model": "gpt-4.1",
  "instructions": "system prompt: preserve identifiers exactly and answer ok",
  "input": [
    {"role": "user", "content": "<32KiB repeated block>"},
    {"role": "user", "content": "<same 32KiB repeated block>"}
  ],
  "max_output_tokens": 1
}
```

Client body bytes: `66124`

Fake upstream log:

```text
UPSTREAM_POST path=/v1/responses len=66124
UPSTREAM_POST path=/v1/responses len=33172
```

Result:

- without `--smart-context`: upstream received `66124` bytes
- with `--smart-context`: upstream received `33172` bytes
- body bytes saved: `32952`
- byte reduction: `49.8%`

## Sidecar Runtime Proxy Measurement

Request used `/v1/runtime/proxy?session_id=diag-session2` and omitted `gateway_path`, so sidecar used its default `/v1/responses`.

Sidecar response counters:

```json
{
  "gateway_status": 200,
  "request_bytes_to_sidecar": 66305,
  "smart_context": {
    "gateway_addr": "127.0.0.1:43291",
    "input_token_reduction_percent": 99,
    "input_tokens_after_observed_or_estimate": 12,
    "input_tokens_before_estimate": 16531,
    "measurement_source": "gateway_usage",
    "mode": "proxy_rewrite"
  },
  "tokens_saved": 16519
}
```

Fake upstream log:

```text
UPSTREAM_POST path=/v1/responses len=33172
UPSTREAM_POST path=/v1/responses len=33147
```

Result:

- sidecar default `/v1/responses` does compact when the body is valid Responses API shape
- `tokens_saved=16519`
- no Rust sidecar code change required

## Conclusion

The zero-savings finding was caused by the measurement payload shape, not by the sidecar path. The sidecar already sends `/v1/responses` by default, and `prodex gateway --smart-context` compacted the correct Responses API body both directly and through `/v1/runtime/proxy`.
