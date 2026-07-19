# Credential Isolation Session API Audit

## Objective
Determine the current Go/server equivalents and implementation status for tasks 2.1, 2.2, and 2.3 of the `agent-credential-isolation` spec. 
- 2.1: `GET /auth/sessions` (account listing by provider)
- 2.2: `POST /auth/login` (with account `config_dir`)
- 2.3: `DELETE /auth/sessions/:id` (revocation)

## Audit Findings & Evidence

### 1. Route Registration & Handler Paths
**Finding: MISSING**
- A comprehensive `grep_search` across `server/cmd/server/router.go` (the central router) and `server/internal/handler/*` reveals that these endpoints **do not exist** in the Go backend.
- The existing `/auth/login` and `/auth/logout` endpoints (handled by `h.Login` and `h.Logout` in `server/internal/handler/auth.go`) are strictly for Multica user authentication (email/password and Google OAuth), not for agent credential/provider sessions.
- There are no endpoints mapping to `GET /auth/sessions` or `DELETE /auth/sessions/:id`. The word "session" in the router is only used for chat sessions (`/api/chat/sessions`), cloud billing checkout sessions, and task pinning.

### 2. Request/Response Shapes & Semantics
**Finding: MISSING**
- Because the routes are completely absent, there are no Go structs defining the request/response shapes.
- Provider/account/config_dir ownership, deletion/revocation semantics, and authorization/tenant boundaries for these specific credential tasks have not been implemented in the Go Core.

### 3. Tests & Executable Evidence
**Finding: MISSING**
- No existing named tests related to provider session listing, agent credential login, or revocation exist within the `server/internal/handler/` test suite.

## Discrepancies versus Spec & Honest Blocker
**Blocker:** The design artifacts (`proposal.md` and `design.md`) are **stale**. They assume that the API contract for credential isolation (`GET /auth/sessions`, `POST /auth/login`, `DELETE /auth/sessions/:id`) is an "AS-IS" port of the legacy Python CAO backend (`infra/cao/auth_routes.py`). However, our audit conclusively proves that this contract has not been ported to the new Go Core server. The required foundation for tasks 2.1, 2.2, and 2.3 simply does not exist.

## Task Grading
- **Task 2.1:** `GET /auth/sessions` lista contas por provedor. → **MISSING**
- **Task 2.2:** `POST /auth/login` com config_dir da conta. → **MISSING**
- **Task 2.3:** `DELETE /auth/sessions/:id` revoga a conta. → **MISSING**

## Explicit Non-Claims
- I do not infer what the new REST resource paths should be named or how they should be designed in Go.
- I do not fabricate Go handler code to fill this gap.
- No OpenSpec checkboxes were marked as ACCEPT or completed. The tasks remain marked as `done: false`.
