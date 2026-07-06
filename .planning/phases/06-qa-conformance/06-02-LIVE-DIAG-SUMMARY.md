# 06-02 Live Diagnosis Summary

status: RED
timestamp_utc: 2026-07-05T03:06:37Z
agent: Codex

## Summary

Continued the 06-02 live rerun investigation after the user reported that 03-01
completed and the L2 sidecar was up. The diagnosis did not find a live
`rpp.l2.v1` endpoint.

## Result

- No listener is present on `127.0.0.1:43117`.
- No `MULTICA_L2*` or `MULTICA_PRODEX*` runtime variables are set in the active
  shell.
- No Docker container or host process advertises an L2/rpp sidecar.
- `prodex info` shows no active runtime, no providers, no profiles, and runtime
  policy disabled.
- The 03-01 code path requires `MULTICA_L2_SIDECAR_ARGS`; without it, daemon
  sidecar launch cannot proceed.
- The code in scope implements an L2 client/lifecycle launcher, not the live
  `rpp.l2.v1` HTTP server itself.

## Evidence

Primary evidence is recorded in:

- `.deploy-control/evidence/06-02-live-sidecar-diagnosis.md`
- `.deploy-control/evidence/c5-smart-context-live.md`
- `.deploy-control/evidence/c6-isolation-live.md`
- `.deploy-control/evidence/mcp-conformance-live.md`

## Final State

06-02 live conformance remains blocked/red until a reachable L2 sidecar endpoint
and bearer token are provided or the sidecar is actually launched with the
required environment:

- `MULTICA_L2_ENABLED=1`
- `MULTICA_L2_BASE_URL=<loopback rpp.l2.v1 endpoint>`
- `MULTICA_L2_BEARER_TOKEN=<token>`
- `MULTICA_L2_SIDECAR_ARGS=<prodex args that start the rpp.l2.v1 server>`
- `MULTICA_PRODEX_ENABLED=1`
- `MULTICA_PRODEX_VERSION=<version>`
- `MULTICA_PRODEX_COMMIT=<commit>`

