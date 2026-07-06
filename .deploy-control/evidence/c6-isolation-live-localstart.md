# C6 Isolation Live Localstart Evidence

status: RED
timestamp_utc: 2026-07-05T03:18:02Z
agent: Codex
plan: PLAN-06-02-LIVE-LOCALSTART

## Setup

Backend and DB readiness were validated after running the user-provided
`multica-migrate` and `multica-server` binaries.

- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- `127.0.0.1:43117`: no listener / connection refused

## Smoke Results

Kill switch smoke:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=dummy \
  bash scripts/smoke/kill-switch-smoke.sh --execute \
  --base-url http://127.0.0.1:43117 --feature smart_context
```

Result:

- `curl: (7) Failed to connect to 127.0.0.1 port 43117`
- JSON validation failed because no response body was returned.

Session start/stop smoke:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=dummy \
  bash scripts/smoke/session-start-stop-smoke.sh --execute \
  --base-url http://127.0.0.1:43117
```

Result:

- `curl: (7) Failed to connect to 127.0.0.1 port 43117`
- JSON validation failed because no response body was returned.

## Conclusion

C6 live remains RED because session isolation and kill-switch behavior cannot be
validated without the reachable L2 sidecar contract endpoint.

