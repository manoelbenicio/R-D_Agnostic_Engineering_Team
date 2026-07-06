# Smart Context Measurement Harness

This harness measures the request/response byte footprint and Smart Context token delta for the L2 `StartSession` path.

Script:

```bash
bash scripts/smoke/smart-context-measure.sh --execute
```

Default target:

```text
http://127.0.0.1:43117/v1/session/start
```

## Safety Gates

The harness follows the smoke-script execution gates used by the rest of `scripts/smoke`:

- dry-run is the default;
- execution requires `SMOKE_ALLOW_EXECUTE=1`;
- if `SMOKE_TARGET_ENV=prod`, execution also requires `DEPLOY_OWNER_APPROVED=true`;
- the bearer token is read from `L2_BEARER_TOKEN`, or from the env var named by `L2_BEARER_TOKEN_ENV`;
- only loopback URLs are accepted.

Example local execution:

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=local \
L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/smart-context-measure.sh --execute --context-kib 512
```

Do not paste the raw request or raw response into evidence. The harness captures the response in a temporary file only long enough to compute metrics, then prints a scrubbed JSON summary.

## Payload Shape

The script sends a valid `rpp.l2.v1` `StartSession` request and adds a `smart_context_probe` object containing a large synthetic context block. Current sidecars that ignore unknown fields still accept the request; Smart Context-aware sidecars may use the probe to return native token counters.

Configurable inputs:

| Setting | Default | Purpose |
| --- | --- | --- |
| `L2_BASE_URL` / `--base-url` | `http://127.0.0.1:43117` | Sidecar base URL |
| `SMOKE_CONTEXT_KIB` / `--context-kib` | `256` | Synthetic context size |
| `SMOKE_SESSION_ID` / `--session-id` | `session-smart-context-measure` | Session identifier |
| `SMOKE_TIMEOUT_SECONDS` / `--timeout` | `12` | Curl timeout |

## Output Metrics

The harness prints one JSON object with these fields:

| Metric | Definition |
| --- | --- |
| `REQUEST_BYTES` | Exact byte length of the JSON body sent to `/v1/session/start`. |
| `RESPONSE_BYTES` | Exact byte length of the captured JSON response body. |
| `CONTEXT_TOKENS_BEFORE` | Native response counter when present; otherwise the local estimate from the synthetic context block, using `ceil(utf8_bytes / 4)`. |
| `CONTEXT_TOKENS_AFTER` | Native response counter when present; otherwise inferred equal to `CONTEXT_TOKENS_BEFORE` for exact/shadow/no-metric responses. |
| `compression_ratio` | `CONTEXT_TOKENS_AFTER / CONTEXT_TOKENS_BEFORE`; lower is better. |
| `tokens_saved` | `max(CONTEXT_TOKENS_BEFORE - CONTEXT_TOKENS_AFTER, 0)`. |
| `fallback_triggered` | Native fallback boolean when present; otherwise inferred true when `smart_context_mode == "exact"`, false otherwise. |

`metric_sources` in the output records whether each token/fallback value came from the sidecar response or from a local inference. A green live Smart Context measurement should prefer native response fields for both token counters.

## Accepted Response Token Fields

The parser recognizes the following response paths:

- before: `context_tokens_before`, `smart_context.context_tokens_before`, `smart_context.tokens_before`, `smart_context.estimated_input_tokens_before`, `metrics.context_tokens_before`, `metrics.smart_context_tokens_before`, `usage.context_tokens_before`;
- after: `context_tokens_after`, `smart_context.context_tokens_after`, `smart_context.tokens_after`, `smart_context.estimated_input_tokens_after`, `smart_context.compressed_tokens`, `metrics.context_tokens_after`, `metrics.smart_context_tokens_after`, `usage.context_tokens_after`;
- fallback: `fallback_triggered`, `smart_context.fallback_triggered`, `smart_context.exact_fallback_triggered`, `metrics.fallback_triggered`.

If a sidecar exposes different field names, extend the parser before treating its results as native Smart Context evidence.

## Pass Criteria

For harness health:

- `/v1/session/start` returns JSON;
- `contract_version == "rpp.l2.v1"`;
- `router_owner == "rust_l2"`;
- `runtime_session_id` is present;
- metrics JSON is emitted without raw payload or raw response content.

For Smart Context efficacy:

- token counters are native response fields, not local inferences;
- `compression_ratio < 1.0` for active rewrite;
- `tokens_saved > 0` for active rewrite;
- `fallback_triggered == false` for a normal rewrite run;
- exact fallback runs report `fallback_triggered == true` and `compression_ratio == 1.0`.
