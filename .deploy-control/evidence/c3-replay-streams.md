# C3 Replay Streams Evidence

- status: BLOCKED
- captured_at_utc: 2026-07-05T03:25:00Z
- executor: Codex#5.5#A
- scope: localhost live run

## Probe

Command shape:

```sh
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 L2_BEARER_TOKEN=<redacted> \
bash scripts/smoke/event-stream-smoke.sh --execute
```

Observed result:

```text
curl: (7) Failed to connect to 127.0.0.1 port 43117: Couldn't connect to server
expected at least 1 event(s), got 0
result=FAIL
```

Rerun after `.env` prodex variable correction produced the same result: connection refused on `127.0.0.1:43117`.

Rerun after `.env` L2 variable correction produced the same result: connection refused on `127.0.0.1:43117`.

## Coverage Impact

- Compact replay: not proven.
- SSE/runtime event stream: not proven.
- WebSocket stream: not proven.

## Conclusion

C3 is not green. Stream integrity cannot be tested while the sidecar event endpoint is unreachable.
