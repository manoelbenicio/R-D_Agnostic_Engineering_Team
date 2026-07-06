# 06-02 Live L2FIX Summary

status: RED
timestamp_utc: 2026-07-05T03:22:48Z
agent: Codex

## Summary

After the user added `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, and
`MULTICA_PRODEX_COMMIT` to `multica-auth-work/.env`, the server was restarted
with the complete env loaded.

## Result

- Backend `8080` is healthy:
  - `/healthz`: HTTP 200
  - `/readyz`: HTTP 200
  - `db=ok`, `migrations=ok`
- L2 `43117` is not healthy:
  - `/healthz`: connection refused
  - `/readyz`: connection refused
  - no listener on `43117`
- `prodex info` still reports `0 runtime`.

## Live Tests

The C5/C6/MCP live smoke rerun still fails because the sidecar endpoint is not
reachable:

- policy apply: failed connection refused
- kill switch: failed connection refused
- session start/stop: failed connection refused
- event stream: failed connection refused, got 0 events

## Root Cause From Code

`MULTICA_PRODEX_*` variables are necessary for Prodex metadata/config, but they
are not sufficient to start the L2 sidecar from the observed server run.

The code gates L2 sidecar startup behind:

- `MULTICA_L2_ENABLED`
- `MULTICA_L2_BASE_URL`
- `MULTICA_L2_BEARER_TOKEN`
- `MULTICA_L2_SIDECAR_ARGS`
- daemon runtime path (`startL2Runtime`), not plain HTTP server readiness

## Evidence

- `.deploy-control/evidence/06-02-live-l2fix.md`

