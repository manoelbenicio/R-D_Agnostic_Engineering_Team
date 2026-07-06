# 06-02 Live Localstart Summary

status: RED
timestamp_utc: 2026-07-05T03:18:02Z
agent: Codex

## What Ran

Followed the user-provided local binary flow:

1. Stopped further Go/download/compile work.
2. Ran the user-provided `multica-auth-work/server/multica-migrate` binary.
3. Loaded the corrected environment and started
   `MULTICA_PRODEX_ENABLED=1 ./multica-server`.
4. Re-ran live C5/C6/MCP smokes against `http://127.0.0.1:43117`.

## Migration

The direct command without environment initially failed because
`multica-migrate` does not load `.env` natively and defaulted to `user=multica`.

The explicit command succeeded:

```sh
DATABASE_URL='postgres://aop_dev:***@localhost:5432/multica?sslmode=disable' ./multica-migrate up
```

Result: all migrations were already applied and the command ended with `Done.`

## Server

Final server startup:

- PID: `1016130`
- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- DB check: `ok`
- Migrations check: `ok`

## L2 Sidecar

The L2 contract port did not come up:

- `http://127.0.0.1:43117/healthz`: connection refused
- `http://127.0.0.1:43117/readyz`: connection refused

The active `.env` has database settings but no `MULTICA_L2*` sidecar settings.

## Smoke Status

- C5 policy apply: RED, connection refused on `43117`.
- C6 kill switch: RED, connection refused on `43117`.
- C6 session start/stop: RED, connection refused on `43117`.
- MCP/event stream: RED, connection refused on `43117`; expected at least 1
  event, got 0.

## Evidence

- `.deploy-control/evidence/c5-smart-context-live-localstart.md`
- `.deploy-control/evidence/c6-isolation-live-localstart.md`
- `.deploy-control/evidence/mcp-conformance-live-localstart.md`

## Final State

The Multica backend is healthy on `8080`, but 06-02 live conformance remains
RED because the `rpp.l2.v1` sidecar endpoint on `43117` is not reachable.

