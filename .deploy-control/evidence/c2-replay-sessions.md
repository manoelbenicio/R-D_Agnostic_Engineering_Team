# C2 Replay Sessions Evidence

- status: BLOCKED
- captured_at_utc: 2026-07-05T03:25:00Z
- executor: Codex#5.5#A
- scope: localhost live run

## Probe

Command shape:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/session-start-stop-smoke.sh --execute
```

Observed result:

```text
curl: (7) Failed to connect to 127.0.0.1 port 43117: Couldn't connect to server
result=FAIL
```

Rerun after `.env` prodex variable correction produced the same result: connection refused on `127.0.0.1:43117`.

Rerun after `.env` L2 variable correction produced the same result: connection refused on `127.0.0.1:43117`.

## Coverage Impact

- Long-session replay: not executed because StartSession endpoint is unreachable.
- Tool-call replay: not executed because runtime session start is unreachable.
- `previous_response_id` continuation: request shape exists in the smoke payload, but no runtime accepted it.

## Conclusion

C2 is not green. Replay cannot be proven until the L2 sidecar listens on `127.0.0.1:43117`.
