# C4 Fail-Closed Profile Switch Evidence

- status: BLOCKED
- captured_at_utc: 2026-07-05T03:25:00Z
- executor: Codex#5.5#A
- scope: localhost live run

## Probe

Command shape:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/profile-fail-closed-smoke.sh --execute
```

Observed result:

```text
curl: (7) Failed to connect to 127.0.0.1 port 43117: Couldn't connect to server
missing HTTP status
result=FAIL
```

Rerun after `.env` prodex variable correction produced the same result: connection refused on `127.0.0.1:43117`.

Rerun after `.env` L2 variable correction produced the same result: connection refused on `127.0.0.1:43117`.

## Auth/Secret Handling

- No HTTP 401 was observed.
- No bearer token or database password was recorded in this evidence.

## Conclusion

C4 is not green. The invalid-profile fail-closed behavior cannot be proven until `/v1/accounts/register` is reachable on the L2 sidecar.
