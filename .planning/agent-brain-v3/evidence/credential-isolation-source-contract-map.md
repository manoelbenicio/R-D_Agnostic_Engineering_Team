# Credential Isolation Source Contract Map

## Objective
Map the authoritative `PROVIDERS` definitions from `infra/cao/auth_routes.py`, `resolveSessionEnv` from `src/canvas-reconciler/reconciler.ts`, and `session-discovery/session-store` contracts.

## Finding: Sources are Absent / Moved

Through rigorous verification across the current working tree and historical ledgers, we can precisely prove that these sources no longer exist in their requested paths due to a major architectural migration.

### 1. Evidence of Absence (Current State)
- **`infra/cao/auth_routes.py`**: Not found. The `infra/` directory no longer exists at the project root.
- **`src/canvas-reconciler/reconciler.ts`**: Not found. The `src/` directory no longer exists at the project root.
- **`session-discovery` / `session-store`**: Not found anywhere in the current project source code (`grep_search` across `multica-auth-work` returned zero matches for `resolveSessionEnv`, `session-discovery`, and `session-store`).

### 2. Evidence of Migration (Historical Ledgers & Context)
The absence of these files is not an anomaly, but the result of two major documented migrations:

#### A. The Go Core Migration (Replacing CAO)
According to `.planning/AGENT_LEDGER_S3.md` (Sprint 3 — Pré-Produção | GO Core Migration):
- At `2026-06-21T08:45:00Z`, `orquestrador_opus46` introduced `go-core-base-url.ts` and `go-core-client.ts`.
- At `2026-06-21T09:00:00Z`, `codex2` removed references to `CAO/:9889`, replacing them with `GO_CORE_BASE_URL`.
- **Conclusion**: The Python CAO backend (`infra/cao/*`), which contained `auth_routes.py`, was fully deprecated and replaced by the Go backend (`server/`).

#### B. The Turborepo / Workspace Refactor (Replacing `src/`)
According to the authoritative `AGENTS.md` (current):
- The project is now a "Go backend + monorepo frontend (pnpm workspaces + Turborepo) with shared packages."
- Code previously residing in a monolithic `src/` directory has been modularized:
  - `server/` - Go backend
  - `apps/web/` - Next.js frontend
  - `packages/core/` - Headless business logic (where stores now live)
- **Conclusion**: Legacy paths like `src/canvas-reconciler/reconciler.ts` and `src/api/session-discovery.ts` (recorded as active in `.planning/AGENT_LEDGER.md` on `2026-05-30`) were eliminated or heavily transformed during the Turborepo restructuring.

### 3. Gap / Blocker
**Blocker:** We cannot map the exact paths, line anchors, or source hashes for `auth_routes.py`, `reconciler.ts`, and `session-discovery.ts` because they no longer exist in the repository. 

**Non-claims:** 
- I am not inferring or fabricating what the current equivalents of these files are in the new `packages/core/` or `server/` architecture, as the objective strictly requested mapping the original legacy paths.
- No OpenSpec checkboxes have been marked as done for tasks 0.1, 0.2, or 0.3.
