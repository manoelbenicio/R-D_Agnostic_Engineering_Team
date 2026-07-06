# PLAN 06-01 SUMMARY - QA Conformance C1-C4

- phase: 06-qa-conformance
- plan: 01
- status: BLOCKED
- executed_by: Codex#5.5#A takeover
- finished_at_utc: 2026-07-05T03:25:00Z
- requirements: REQ-13, REQ-14, REQ-15

## Result

06-01 is not green. The local database migration completed with the provided `multica-migrate` binary and the backend is healthy on `127.0.0.1:8080`, but the L2 sidecar is not listening on `127.0.0.1:43117`.

The test was rerun after `.env` was corrected with `MULTICA_PRODEX_PATH`, `MULTICA_PRODEX_VERSION`, and `MULTICA_PRODEX_COMMIT`. The backend still started successfully, but no `43117` listener appeared and the C1-C4 smokes still failed with connection refused.

The test was rerun again after `.env` was corrected with `MULTICA_L2_ENABLED`, `MULTICA_L2_BASE_URL`, and `MULTICA_L2_BEARER_TOKEN`. The backend still started successfully, but no `43117` listener appeared and the C1-C4 smokes still failed with connection refused.

## Evidence

- `.deploy-control/evidence/c1-capability-conformance.md`
- `.deploy-control/evidence/c2-replay-sessions.md`
- `.deploy-control/evidence/c3-replay-streams.md`
- `.deploy-control/evidence/c4-fail-closed.md`

## Commands Executed

- `cd multica-auth-work/server && ./multica-migrate up`
- `cd multica-auth-work/server && MULTICA_PRODEX_ENABLED=1 ./multica-server &`
- `curl http://127.0.0.1:8080/health` -> HTTP 200
- `curl http://127.0.0.1:43117/readyz` -> connection refused
- `scripts/smoke/readyz-smoke.sh --execute` -> FAIL, sidecar connection refused
- `scripts/smoke/policy-apply-smoke.sh --execute` -> FAIL, sidecar connection refused
- `scripts/smoke/session-start-stop-smoke.sh --execute` -> FAIL, sidecar connection refused
- `scripts/smoke/event-stream-smoke.sh --execute` -> FAIL, sidecar connection refused
- `scripts/smoke/profile-fail-closed-smoke.sh --execute` -> FAIL, sidecar connection refused

## Verification State

- [ ] C1 green (behavior, not label) - BLOCKED
- [ ] C2 green (replay passes) - BLOCKED
- [ ] C3 green (streams intact) - BLOCKED
- [ ] C4 green (fail-closed proven) - BLOCKED
- [x] Evidence scrubbed of secrets

## Blocker

`multica-server` does not start or expose the L2 sidecar on `127.0.0.1:43117` in this run. No HTTP 401/auth failure was observed.
