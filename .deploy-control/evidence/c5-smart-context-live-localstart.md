# C5 Smart Context Live Localstart Evidence

status: RED
timestamp_utc: 2026-07-05T03:18:02Z
agent: Codex
plan: PLAN-06-02-LIVE-LOCALSTART

## Setup

- User-provided binaries used from `multica-auth-work/server/`:
  - `./multica-migrate`
  - `./multica-server`
- Migration command that succeeded:
  - `DATABASE_URL='postgres://aop_dev:***@localhost:5432/multica?sslmode=disable' ./multica-migrate up`
- Server command used after migration:
  - `. ../.env` loaded into environment
  - `MULTICA_PRODEX_ENABLED=1 ./multica-server`
- Detached server PID after final restart: `1016130`.

## Backend Readiness

- `http://127.0.0.1:8080/healthz`: HTTP 200
- `http://127.0.0.1:8080/readyz`: HTTP 200
- Response checks reported `db=ok` and `migrations=ok`.

## L2 Contract Endpoint

- `http://127.0.0.1:43117/healthz`: connection refused
- `http://127.0.0.1:43117/readyz`: connection refused

## Smoke Result

Command:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=dummy \
  bash scripts/smoke/policy-apply-smoke.sh --execute --base-url http://127.0.0.1:43117
```

Result:

- `curl: (7) Failed to connect to 127.0.0.1 port 43117`
- Python validation then failed with `JSONDecodeError` because no response body
  was returned.

## Conclusion

C5 live remains RED because the L2 `rpp.l2.v1` endpoint required for policy
application is not reachable on `127.0.0.1:43117`.

