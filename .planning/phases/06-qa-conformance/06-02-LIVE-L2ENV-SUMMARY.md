# 06-02 Live L2ENV Summary

status: RED
timestamp_utc: 2026-07-05T03:26:52Z
agent: Codex

## Summary

After `MULTICA_L2_ENABLED`, `MULTICA_L2_BASE_URL`, and
`MULTICA_L2_BEARER_TOKEN` were added to `multica-auth-work/.env`, Codex
reloaded the env, restarted the backend, and probed the daemon path.

## Result

- Backend is green on `8080`.
- The daemon reaches L2 startup when given temporary CLI auth.
- L2 startup fails with:

```text
l2 runtime enabled but MULTICA_L2_SIDECAR_ARGS is required
```

## Final State

No service is listening on `127.0.0.1:43117`, so live C5/C6/MCP cannot turn
green yet.

The remaining blocker is explicit and code-confirmed:

- set `MULTICA_L2_SIDECAR_ARGS` to a command supported by the configured
  `MULTICA_PRODEX_PATH` that actually starts an `rpp.l2.v1` sidecar on
  `127.0.0.1:43117`; or
- provide the compiled/forked sidecar binary/command that implements the
  contract.

Evidence:

- `.deploy-control/evidence/06-02-live-l2env.md`

