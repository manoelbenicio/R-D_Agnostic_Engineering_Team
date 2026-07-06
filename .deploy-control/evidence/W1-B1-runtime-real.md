# W1-B1 - Runtime REAL adapter behind rpp.l2.v1

Agent: Codex#5.5#B  
Timestamp UTC: 2026-07-05T22:56:07Z  
Check-in: `.deploy-control/Codex-5.5-B__W1-B1-RUNTIME-REAL__20260705T224233Z.md`  
Scope: `multica-auth-work/prodex-sidecar/**`

## Result

Status: GREEN.

Implemented a real sidecar adapter in `multica-auth-work/prodex-sidecar/src/main.rs`:

- `POST /v1/session/start` now ensures a real `prodex gateway` subprocess is running with `--smart-context`.
- The response returns `router_owner: "rust_l2"`, `runtime_endpoint`, `event_stream_url`, `runtime_log_ref`, and `contract_version: "rpp.l2.v1"`.
- Added `POST /v1/runtime/proxy?session_id=...` as the rpp.l2.v1 facade that forwards the client body through the prodex gateway and records Smart Context token metrics.
- `GET /readyz` now probes both the gateway subprocess/port and Postgres readiness via `PRODEX_PG_URL` using `psql -Atqc 'SELECT 1'`.
- Event stream emits NDJSON with `occurred_at`, `event_type`, route decision data, and token metric summary in the schema-compatible `message`.

## Code evidence

```text
prodex --version
prodex 0.246.0
```

Relevant source locations from `rg`:

```text
src/main.rs:188  handle_readyz()
src/main.rs:247  reads PRODEX_PG_URL
src/main.rs:255  executes SELECT 1 via psql
src/main.rs:582  session_start ensures gateway
src/main.rs:597  returns /v1/runtime/proxy runtime_endpoint
src/main.rs:737  handle_runtime_proxy()
src/main.rs:820  emits route_decision
src/main.rs:855  returns input_token_reduction_percent
src/main.rs:960  ensure_gateway_running()
src/main.rs:981  injects --smart-context
src/main.rs:1017 reads MULTICA_L2_SIDECAR_ARGS when provided
src/main.rs:1126 routes POST /v1/runtime/proxy
```

Note: `multica-auth-work/prodex-sidecar/` is currently untracked in this checkout (`git ls-files ... | wc -l => 0`), so `git diff` does not show these source changes even though the files exist on disk.

## Local smoke

No provider call was made. A fake loopback upstream was used behind `prodex gateway` to verify the adapter path.

Commands run:

```text
python3 -c '<fake HTTP upstream on 127.0.0.1:43290>'

MULTICA_L2_BEARER_TOKEN=audit-test-token \
PRODEX_GATEWAY_UPSTREAM_BASE_URL=http://127.0.0.1:43290 \
PRODEX_GATEWAY_LISTEN=127.0.0.1:43291 \
multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43292
```

`session_start`:

```text
HTTP/1.1 200 OK
{
  "contract_version":"rpp.l2.v1",
  "router_owner":"rust_l2",
  "runtime_endpoint":"http://127.0.0.1:43292/v1/runtime/proxy?session_id=session-b1",
  "runtime_log_ref":"prodex-gateway://127.0.0.1:43291",
  "smart_context_mode":"proxy_rewrite",
  "gateway":{"listen_addr":"127.0.0.1:43291","smart_context_enabled":true}
}
```

`readyz` with no `PRODEX_PG_URL` intentionally fails closed while gateway passes:

```text
HTTP/1.1 503 Service Unavailable
{
  "contract_version":"rpp.l2.v1",
  "status":"error",
  "checks":[
    {"name":"shared_state_backend","status":"fail","details":{"connection_status":"error","error":"missing_config"}},
    {"name":"runtime_proxy","status":"pass","details":{"listen_addr":"127.0.0.1:43291","pid":95814}}
  ]
}
```

Runtime proxy through `prodex gateway`:

```text
HTTP/1.1 200 OK
{
  "contract_version":"rpp.l2.v1",
  "router_owner":"rust_l2",
  "gateway_status":200,
  "smart_context":{
    "mode":"proxy_rewrite",
    "gateway_addr":"127.0.0.1:43291",
    "input_tokens_before_estimate":25,
    "input_tokens_after_observed_or_estimate":6,
    "input_token_reduction_percent":76,
    "measurement_source":"gateway_usage"
  }
}
```

Event stream:

```text
HTTP/1.1 200 OK
Content-Type: application/x-ndjson

{"contract_version":"rpp.l2.v1","event_type":"session_started","occurred_at":"2026-07-05T22:53:47.156734033+00:00","producer":{"component":"event_stream","plane":"rust_l2"},"session_id":"session-b1","tenant_id":"tenant-b1",...}
{"contract_version":"rpp.l2.v1","event_type":"route_decision","message":"smart_context tokens before=25 after=6 reduction_percent=76","occurred_at":"2026-07-05T22:55:30.436932005+00:00","producer":{"component":"runtime_proxy","plane":"rust_l2"},"runtime_request_id":"rt-req-b1","route_decision":{"committed":true,"decision_phase":"pre_commit","reason":"prodex_gateway_smart_context","selected_profile_id":"profile-b1","selected_provider":"openai"},...}
```

Cleanup verified:

```text
pgrep -af '[p]rodex gateway.*43291|[p]rodex.*--listen 127.0.0.1:43291'
# exit 1, no temporary gateway left running
```

## Build and test

```text
cargo fmt --check
# exit 0

cargo test
running 4 tests
test tests::parse_http_response_extracts_status_and_body ... ok
test tests::split_args_preserves_quoted_values ... ok
test tests::scrub_error_category_never_returns_raw_error ... ok
test tests::token_reduction_percent_handles_growth_and_reduction ... ok
test result: ok. 4 passed; 0 failed

cargo build --release
Finished `release` profile [optimized] target(s) in 4.79s
```

## Caveats

- The smoke used a fake upstream to avoid provider calls and secrets in evidence. The adapter path still launched real `prodex 0.246.0 gateway --smart-context`.
- No live `PRODEX_PG_URL` was provided in this shell. `/readyz` therefore demonstrated fail-closed Postgres readiness (`missing_config`) while proving the gateway subprocess check passes.
- Token reduction in the smoke is measured from local preflight estimate versus gateway `usage.input_tokens`. With a real upstream/provider, this records prodex gateway usage when returned; otherwise it falls back to local estimate.
