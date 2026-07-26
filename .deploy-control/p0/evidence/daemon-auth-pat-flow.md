# Daemon auth — temporary PAT flow validation (independent)

- agent: Codex56#B · pane: w7:p4 · task: P0-DAEMON-AUTH-AND-UI-DELIVERY
- mode: independent READ-ONLY static/contract validation; **no token value was read, printed, or hashed by this validation**
- repo HEAD: `a6d5098`; backend `127.0.0.1:8080` health = connection refused (no running stack)

## Result

STATIC/CONTRACT VALIDATION: **PASS** for the supported temporary-PAT flow, secure `MULTICA_TOKEN`
injection, and daemon register/heartbeat auth. LIVE end-to-end confirmation is **BLOCKED_EXTERNAL** (no
running backend/daemon/DB in this environment; `live_runs=false`).

## Flow (verified in source)

1. **POST /api/tokens** — `cmd/server/router.go:671` `r.Route("/api/tokens")` → `r.Post("/", h.CreatePersonalAccessToken)`
   under the authenticated user group. Handler `internal/handler/personal_access_token.go:60`:
   - requires an authenticated user (`requireUserID`);
   - mints `mul_`+40hex via `auth.GeneratePATToken()`;
   - **persists only `auth.HashToken(rawToken)`** + a 12-char prefix (raw token never stored);
   - sets expiry (`expires_in_days`, default issuance window 90d; renew threshold 7d / extension 90d);
   - returns the raw token **once** in `CreatePATResponse.Token` at `201 Created` — correct show-once PAT pattern.
   - CLI path: `cmd/multica/cmd_auth.go:322` `client.PostJSON("/api/tokens", ...)` (`multica login`).
2. **Secure injection via MULTICA_TOKEN** — `internal/daemon/daemon.go:3517` builds the agent child env with
   `"MULTICA_TOKEN": agentToken`, where `agentToken = taskScopedAuthToken(task)` — a **task-scoped** token
   bound to (agent, task), NOT the daemon's own credential (explicit comment forbids that fallback; refusing
   to start the agent if the task token is invalid). Least-privilege injection. Sibling env is task metadata
   only (server URL, workspace/agent/task ids, slot) — no secrets besides the scoped token.
3. **Daemon register/heartbeat** — routes `/api/daemon/register` and `/api/daemon/heartbeat`
   (`internal/handler/daemon.go`, `processHeartbeat`), guarded by `middleware/daemon_auth.go DaemonAuth`:
   - `mdt_` daemon-token path (cache → DB `GetDaemonTokenByHash`), `mcn_` cloud-PAT path, and `mul_` PAT
     fallback (`GetPersonalAccessTokenByHash`);
   - tokens are **hashed before any lookup** (`auth.HashToken`); logs contain only path/error, **never the
     raw token**; strips client-supplied `X-Actor-Source`; **fail-closed** on missing/invalid/misformatted
     Authorization (401) and on unconfigured cloud PAT.
   - PAT renewal supported in-place (`RenewCurrentPersonalAccessToken`, `token_renewal_test.go`).

## Security properties confirmed (no weakening)

- Raw PAT stored only as a hash; returned once at creation.
- `MULTICA_TOKEN` is task-scoped, never the daemon credential.
- DaemonAuth is fail-closed and does not log token values.
- No secret value was surfaced by this validation.

## Blocker / fix (report to Kiro + Principal)

- **Blocker:** live end-to-end validation (real `POST /api/tokens` → `MULTICA_TOKEN` injection → daemon
  `register`/`heartbeat` 200) cannot run here — backend `:8080` is not listening and cannot be started in
  this lane (no deploy/Docker; requires Postgres/Redis; `live_runs=false`).
- **Owner:** Kiro / Principal Orchestrator (provision the running backend+daemon+DB in the non-prod stack).
- **Next action:** with the stack up, run `multica login` (issues the temporary PAT via POST /api/tokens),
  start the daemon (token injected as `MULTICA_TOKEN`), and confirm `POST /api/daemon/register` +
  `POST /api/daemon/heartbeat` return 200 under `DaemonAuth`. **Never print the token**; assert only status
  codes, daemon_id/workspace scope, and heartbeat ack. No inference / no OmniRoute probing required for this
  control-plane check.

## Non-claims

- Live register/heartbeat NOT executed (backend down). No product code modified. No token read/printed.
- No OpenSpec checkbox closed; Principal adjudicates the live confirmation.
