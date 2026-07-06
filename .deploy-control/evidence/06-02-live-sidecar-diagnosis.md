# 06-02 Live Sidecar Diagnosis

status: RED
timestamp_utc: 2026-07-05T03:06:37Z
agent: Codex
plan: PLAN-06-02-LIVE-DIAG
depends_on: PLAN-03-01, PLAN-06-02-LIVE

## Objective

Diagnose why the real live C5/C6/MCP rerun for 06-02 still fails after the
reported 03-01 completion and alleged L2 sidecar startup.

## Host Observations

- `curl` probes from the prior live rerun to `http://127.0.0.1:43117/healthz`
  and `http://127.0.0.1:43117/readyz` failed with connection refused.
- `ss -ltnp` still shows no listener on `127.0.0.1:43117`.
- `env | rg '^(MULTICA_L2|MULTICA_PRODEX|PRODEX_HOME|SMOKE_)'` returned no
  matching runtime configuration in this shell.
- `docker ps --format ... | rg -i 'prodex|l2|sidecar|rpp|codex'` returned no
  matching sidecar container.
- `ps -ef | rg 'prodex|l2|sidecar|43117|rpp\.l2'` found no persistent L2 sidecar
  process; the only hit was the transient help command run by this diagnostic.

## Prodex Observations

- `prodex --version`: `prodex 0.246.0`.
- `prodex info` reports:
  - `Profiles: 0`
  - `Providers: none`
  - `Runtime policy: disabled`
  - `Prodex processes: Yes (1 total, 0 runtime)`
  - `Recent load: No active prodex runtime detected`
- `prodex gateway --help` describes an OpenAI-compatible gateway. It does not
  expose the required `rpp.l2.v1` contract endpoints used by 06-02:
  `/readyz`, `/v1/session/start`, `/v1/session/stop`, and
  `/v1/events/stream`.

## Code Findings

- `multica-auth-work/server/internal/daemon/prodex.go:54-80` only enables L2
  when `MULTICA_L2_ENABLED` is truthy and validates
  `MULTICA_L2_BASE_URL` plus `MULTICA_L2_BEARER_TOKEN`.
- `multica-auth-work/server/internal/daemon/prodex.go:14-33` only enables
  Prodex launch when `MULTICA_PRODEX_ENABLED` is truthy and requires
  `MULTICA_PRODEX_VERSION` and `MULTICA_PRODEX_COMMIT`.
- `multica-auth-work/server/internal/daemon/l2_runtime.go:125-151` launches a
  sidecar only if `MULTICA_L2_SIDECAR_ARGS` parses to at least one argument.
  Without it, `Start()` returns:
  `l2 runtime enabled but MULTICA_L2_SIDECAR_ARGS is required`.
- `multica-auth-work/server/internal/daemon/l2_runtime.go:201-230` starts
  `prodex` with those sidecar args and discards sidecar stdout/stderr.
- `multica-auth-work/server/internal/l2runtime/client.go:177-198` implements a
  loopback-only client, not a server.
- `multica-auth-work/server/internal/l2runtime/client.go:232-255`,
  `370-391`, and `532-538` confirm the expected live endpoints are client
  calls to `/healthz`, `/readyz`, `/v1/session/start`, `/v1/session/stop`, and
  `/v1/events/stream`.

## Root Cause

03-01 appears to have delivered daemon lifecycle/client integration, but there
is no verifiable live `rpp.l2.v1` sidecar process or endpoint in this host
namespace. The current environment also lacks the variables required for the
daemon path to launch a sidecar.

## Conclusion

The 06-02 live C5/C6/MCP result remains RED. The blocker is external to the
06-02 smoke scripts: no reachable L2 sidecar endpoint implementing the
`rpp.l2.v1` contract is running at the tested endpoint, and no alternate
endpoint/token is discoverable from the environment, process table, Docker, or
03-01 artifacts.

