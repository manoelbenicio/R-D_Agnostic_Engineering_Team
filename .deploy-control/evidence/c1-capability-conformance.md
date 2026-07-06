# C1 Capability Conformance Evidence

- status: BLOCKED
- captured_at_utc: 2026-07-05T03:25:00Z
- executor: Codex#5.5#A
- scope: localhost live run

## Setup

- `multica-auth-work/server/multica-migrate up`: PASS against local PostgreSQL `multica` database.
- `MULTICA_PRODEX_ENABLED=1 ./multica-server &`: backend process stayed running.
- Rerun after operator added `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, and `MULTICA_PRODEX_COMMIT` to `.env`: backend still stayed running.
- Rerun after operator added `MULTICA_L2_ENABLED`, `MULTICA_L2_BASE_URL`, and `MULTICA_L2_BEARER_TOKEN` to `.env`: backend still stayed running.
- `curl http://127.0.0.1:8080/health`: HTTP 200.
- `curl http://127.0.0.1:43117/readyz`: connection refused.

## Probe

Command shape:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/readyz-smoke.sh --execute
```

Observed result:

```text
curl: (7) Failed to connect to 127.0.0.1 port 43117: Couldn't connect to server
result=FAIL
```

## Conclusion

C1 is not green. The backend is healthy, but the L2 sidecar endpoint required for behavior-based capability conformance is not listening on `127.0.0.1:43117`.
