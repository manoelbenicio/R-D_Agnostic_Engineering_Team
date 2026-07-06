# W1-D1 Revised - readyz REAL spec + test plan

Agent: Codex#5.5#D  
Timestamp: 2026-07-05T22:32:56Z  
Scope: READ ONLY on `multica-auth-work/prodex-sidecar/`; evidence-only write.

## Current `readyz` behavior

Read-only source inspected:

- `multica-auth-work/prodex-sidecar/src/main.rs:148-158`
- `multica-auth-work/prodex-sidecar/src/main.rs:503-512`

Current handler:

- `GET /readyz` routes to `handle_readyz()`.
- `handle_readyz()` calls `json_response(...)`, which emits HTTP 200.
- Top-level `contract_version` is hardcoded to `CONTRACT_VERSION`.
- Top-level `status` is hardcoded to `"ready"`.
- `checks[0].name` is hardcoded to `"shared_state_backend"`.
- `checks[0].status` is hardcoded to `"pass"`.
- `checks[0].details.backend_type` is hardcoded to `"postgres"`.
- `checks[0].details.connection_status` is hardcoded to `"ok"`.
- `checks[1]` is hardcoded to `{"name": "kill_switch", "status": "pass"}`.
- `checks[2]` is hardcoded to `{"name": "runtime_proxy", "status": "pass"}`.

Falsification finding:

- No Postgres URL is read.
- No Redis URL is read.
- No Postgres connection is opened.
- No `SELECT 1` is executed.
- No Redis connection is opened.
- No Redis `PING` is executed.
- Therefore Postgres can be down and `/readyz` still reports HTTP 200 + `"status": "ready"` + Postgres `"pass"`.

## Required spec for real readiness

Configuration:

- `PRODEX_PG_URL`: required for shared state readiness.
- `PRODEX_REDIS_URL`: optional. If absent or empty, Redis is not checked and must be reported as not configured/skipped, not as pass.

Probe behavior:

- On every `GET /readyz`, probe Postgres using `PRODEX_PG_URL`.
- Postgres probe must execute `SELECT 1` and verify success.
- If `PRODEX_PG_URL` is missing, invalid, connection fails, times out, or `SELECT 1` fails, the Postgres check must be `fail`.
- If `PRODEX_REDIS_URL` is present and non-empty, probe Redis with `PING`.
- If Redis URL is configured but connection fails, times out, or `PING` does not return success, the Redis check must be `fail`.
- Keep probe timeouts bounded so `/readyz` cannot hang deployment checks.

Response semantics:

- Ready only when Postgres passes and Redis either passes or is not configured.
- If Postgres fails, `/readyz` must not pass: return HTTP 503 and top-level `"status": "error"`.
- If Postgres passes but configured Redis fails, `/readyz` must not pass: return HTTP 503 and top-level `"status": "degraded"`.
- If both Postgres and configured Redis fail, return HTTP 503 and top-level `"status": "error"`.
- Preserve existing contract metadata and non-storage checks, but do not hardcode storage dependency success.

Recommended JSON shape:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "ready|degraded|error",
  "checks": [
    {
      "name": "shared_state_backend",
      "status": "pass|fail",
      "details": {
        "backend_type": "postgres",
        "configured": true,
        "probe": "SELECT 1",
        "connection_status": "ok|error",
        "error": "scrubbed error category, no credentials"
      }
    },
    {
      "name": "redis",
      "status": "pass|fail|skip",
      "details": {
        "configured": true,
        "probe": "PING",
        "connection_status": "ok|error|not_configured",
        "error": "scrubbed error category, no credentials"
      }
    },
    {
      "name": "kill_switch",
      "status": "pass"
    },
    {
      "name": "runtime_proxy",
      "status": "pass"
    }
  ]
}
```

Security requirements:

- Never include full `PRODEX_PG_URL`, `PRODEX_REDIS_URL`, usernames, passwords, tokens, hosts with credentials, or raw driver errors that may contain credentials in logs or responses.
- Error details should be categorical, for example `missing_config`, `connect_failed`, `timeout`, `query_failed`, or `ping_failed`.

## Unit test plan for Codex#5.5#B implementation

Unit tests should isolate readiness decision logic from real network dependencies by injecting probe results.

- `readyz_all_dependencies_ok_returns_ready`: PG probe pass, Redis configured pass -> HTTP 200, top-level `ready`, PG `pass`, Redis `pass`.
- `readyz_postgres_down_returns_error`: PG probe fail, Redis absent or pass -> HTTP 503, top-level `error`, PG `fail`; must not contain any hardcoded PG pass.
- `readyz_redis_down_returns_degraded`: PG probe pass, Redis configured fail -> HTTP 503, top-level `degraded`, Redis `fail`.
- `readyz_both_down_returns_error`: PG probe fail, Redis configured fail -> HTTP 503, top-level `error`, both checks `fail`.
- `readyz_redis_unconfigured_is_skipped`: PG probe pass, `PRODEX_REDIS_URL` empty/unset -> HTTP 200, top-level `ready`, Redis `skip` or omitted by explicit design.
- `readyz_postgres_url_missing_returns_error`: `PRODEX_PG_URL` empty/unset -> HTTP 503, top-level `error`, PG `fail`.
- `readyz_errors_are_scrubbed`: simulated driver error containing credentials -> response does not include credentials or full URLs.

## Runtime falsification test plan

T1 - Postgres down:

- Start sidecar with valid `PRODEX_PG_URL` and Redis either absent or healthy.
- Stop or block Postgres.
- Call `GET /readyz`.
- Expected: HTTP 503; top-level `status` is `error`; `shared_state_backend.status` is `fail`; response must not report Postgres `connection_status: "ok"`.

T2 - Redis down:

- Start sidecar with valid `PRODEX_PG_URL` and configured `PRODEX_REDIS_URL`.
- Keep Postgres healthy.
- Stop or block Redis.
- Call `GET /readyz`.
- Expected: HTTP 503; top-level `status` is `degraded`; Redis check is `fail`; Postgres check remains `pass`.

T3 - Postgres and Redis both down:

- Start sidecar with both URLs configured.
- Stop or block both services.
- Call `GET /readyz`.
- Expected: HTTP 503; top-level `status` is `error`; PG and Redis checks are both `fail`.

T4 - Postgres and Redis both up:

- Start sidecar with both URLs configured and services healthy.
- Call `GET /readyz`.
- Expected: HTTP 200; top-level `status` is `ready`; PG `SELECT 1` check is `pass`; Redis `PING` check is `pass`.

Evidence required after implementation:

- Command lines used to start sidecar and run each test, with secrets scrubbed.
- HTTP status and response body for T1-T4.
- `cargo test` result for readiness unit tests.
- `cargo build --release` result from `multica-auth-work/prodex-sidecar`.
