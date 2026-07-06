# MCP Conformance Live Localstart Evidence

status: RED
timestamp_utc: 2026-07-05T03:18:02Z
agent: Codex
plan: PLAN-06-02-LIVE-LOCALSTART

## Setup

After migration and server startup:

- `multica-server` detached PID: `1016130`
- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- `http://127.0.0.1:43117/healthz`: connection refused
- `http://127.0.0.1:43117/readyz`: connection refused

## Event Stream Smoke

Command:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=dummy \
  bash scripts/smoke/event-stream-smoke.sh --execute \
  --base-url http://127.0.0.1:43117 --min-events 1
```

Result:

- `curl: (7) Failed to connect to 127.0.0.1 port 43117`
- `expected at least 1 event(s), got 0`

## Conclusion

MCP/event-stream live conformance remains RED because
`/v1/events/stream` cannot be reached while `127.0.0.1:43117` has no listener.

