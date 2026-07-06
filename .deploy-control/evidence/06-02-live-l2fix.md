# 06-02 Live L2FIX Evidence

status: RED
timestamp_utc: 2026-07-05T03:22:48Z
agent: Codex
plan: PLAN-06-02-LIVE-L2FIX

## User Instruction

User reported that the L2 failure was corrected by adding these variables to
`multica-auth-work/.env`:

- `MULTICA_PRODEX_ENABLED=1`
- `MULTICA_PRODEX_PATH=/home/dataops-lab/runtime/prodex-src/target/release/prodex`
- `MULTICA_PRODEX_VERSION=v0.246.0`
- `MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144`

Requested actions:

1. Kill previous server PID `1016130`.
2. Reload the complete `.env`.
3. Run `./multica-server` again.
4. Validate that sidecar comes up on `43117` and rerun live tests.

## Execution

- PID `1016130` was already gone.
- Loaded `multica-auth-work/.env` from `multica-auth-work/server/`.
- Started `./multica-server`.
- A server launched from another terminal is currently listening:
  - PID `1023245`
  - command: `./multica-server`
  - env source shown by parent shell: `source .../multica-auth-work/.env`

## Backend Readiness

- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- Checks: `db=ok`, `migrations=ok`

## L2 Endpoint

- `http://127.0.0.1:43117/healthz`: connection refused
- `http://127.0.0.1:43117/readyz`: connection refused
- `ss -ltnp`: no listener on `43117`
- `prodex info`: `0 runtime`, `No active prodex runtime detected`

## Live Smoke Results

- C5 policy apply: RED
  - `curl: (7) Failed to connect to 127.0.0.1 port 43117`
  - JSON validation failed because no response body was returned.
- C6 kill switch: RED
  - `curl: (7) Failed to connect to 127.0.0.1 port 43117`
  - JSON validation failed because no response body was returned.
- C6 session start/stop: RED
  - `curl: (7) Failed to connect to 127.0.0.1 port 43117`
  - JSON validation failed because no response body was returned.
- MCP event stream: RED
  - `curl: (7) Failed to connect to 127.0.0.1 port 43117`
  - `expected at least 1 event(s), got 0`

## Code Finding

The loaded Prodex variables satisfy the Prodex launch metadata checks, but the
L2 sidecar path is gated elsewhere:

- `multica-auth-work/server/internal/daemon/prodex.go:54-80`
  - L2 runtime is enabled only when `MULTICA_L2_ENABLED` is truthy.
  - It requires `MULTICA_L2_BASE_URL` and `MULTICA_L2_BEARER_TOKEN`.
- `multica-auth-work/server/internal/daemon/l2_runtime.go:125-151`
  - Sidecar launch requires non-empty `MULTICA_L2_SIDECAR_ARGS`.
- `multica-auth-work/server/internal/daemon/daemon.go:789`
  - `startL2Runtime` is invoked in the daemon run path, not observed in the
    `multica-server` HTTP server path.

Current `.env` has `MULTICA_PRODEX_*` but no `MULTICA_L2_*` entries.

## Conclusion

The L2FIX run remains RED. The Multica backend is healthy on `8080`, but the
`rpp.l2.v1` sidecar is still not reachable on `127.0.0.1:43117`.

