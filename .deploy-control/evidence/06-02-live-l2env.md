# 06-02 Live L2ENV Evidence

status: RED
timestamp_utc: 2026-07-05T03:26:52Z
agent: Codex
plan: PLAN-06-02-LIVE-L2ENV

## Environment Confirmed

`multica-auth-work/.env` now contains:

- `MULTICA_PRODEX_ENABLED=1`
- `MULTICA_PRODEX_PATH=/home/dataops-lab/runtime/prodex-src/target/release/prodex`
- `MULTICA_PRODEX_VERSION=v0.246.0`
- `MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144`
- `PRODEX_HOME=/home/dataops-lab/runtime/prodex-home`
- `MULTICA_L2_ENABLED=1`
- `MULTICA_L2_BASE_URL=http://127.0.0.1:43117`
- `MULTICA_L2_BEARER_TOKEN=<redacted>`

## Backend Restart

Started `multica-auth-work/server/multica-server` with the complete env loaded.

- PID: `1027903`
- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- Checks: `db=ok`, `migrations=ok`

## Daemon L2 Probe

No compiled `multica` CLI binary exists in the workspace. To test the daemon
path without mutating the real user config, Codex used an isolated temporary
`HOME=/tmp/codex-multica-daemon-home` with a minimal local CLI config.

Command shape:

```sh
HOME=/tmp/codex-multica-daemon-home \
PATH=/home/dataops-lab/.cache/codex-go/go/bin:$PATH \
go run ./cmd/multica daemon start --foreground --server-url http://127.0.0.1:8080
```

Result:

```text
authenticated
l2 runtime enabled but MULTICA_L2_SIDECAR_ARGS is required
exit status 1
```

This proves the daemon reached `startL2Runtime`, but the sidecar launch gate
failed before any process could bind `43117`.

## L2 Endpoint

- `127.0.0.1:43117`: no listener observed.
- `prodex --help` exposes `gateway` as an OpenAI-compatible gateway, but no
  documented `rpp.l2.v1` sidecar command.
- Searching the available Prodex source did not find an existing
  `rpp.l2.v1` HTTP server surface for `/healthz`, `/readyz`,
  `/v1/session/start`, `/v1/session/stop`, or `/v1/events/stream`.

## Conclusion

The run remains RED. `MULTICA_L2_ENABLED`, `MULTICA_L2_BASE_URL`, and
`MULTICA_L2_BEARER_TOKEN` are necessary but still not sufficient for the
current Go daemon path. The remaining required variable is
`MULTICA_L2_SIDECAR_ARGS`, and the available Prodex binary/source does not
advertise a ready `rpp.l2.v1` sidecar command to use as those args.

