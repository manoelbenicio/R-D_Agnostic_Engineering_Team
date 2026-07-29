# Design — Agentic Execution Plan (Native Runtimes Onboarding v6.2)

## Roles
- **GTL / Kiro (orchestrator)** — Coordinates waves, reviews check-ins, runs container build/test integration, validates deliverables, grants DONE.
- **Coder Agents** — Produce code in assigned disjunct scope.

## Check-in Protocol
Each agent writes in `.deploy-control/`:
- START check-in (scope, files, dependencies, risks).
- DONE check-in (modifications, files, container build/test evidence).

## File Ownership
- Backend files: `server/pkg/agent/cline.go` (Cline ACP backend).
- Shared files (`internal/daemon/config.go`, `pkg/agent/agent.go`, `cmd/server/router.go`) — managed sequentially during integration.

## Status & Lineage Truth (2026-07-29)

### NVIDIA NIM (SUPERSEDED)
Per owner ruling, NVIDIA NIM was deliberately removed from canonical source (verified zero `nim.go`/`nim_home.go` at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`). All obsolete NIM implementation/deploy claims and tasks are SUPERSEDED. Any NIM revival requires a new owner-approved proposal.

### Cline 3.x Runtime
Canonical backend source contains `cline.go`, config probe (`MULTICA_CLINE_PATH`/`cline`), `New()` cases, `SupportedTypes`, and `POST /auth/login`. Live CLI is `cline 3.0.46`. Live runtime execution and canary remain pending the ORQ-66 durable daemon deployment (ORQ2 daemon remains old `88ca` until ORQ-66). No frontend or live Cline acceptance is claimed prematurely.

### Onboarding & Design System Parity
- **Onboarding Frontend & Sponsor Removal**: Accepted via ORQ-51.
- **Design System Color/UI Parity**: Accepted via ORQ-52.
- **Backend Auth (`POST /auth/login`)**: Simple username/password login implemented behind `AuthProvider` interface (Firebase-ready).

## Decisions & Risk Governance
- **Cline ACP Transport**: Uses `cline --acp` over stdin/stdout (ACP JSON-RPC 2.0). The `--json` flag belongs to a separate headless prompt-output mode and is filtered out to prevent breaking the ACP handshake.
- **NIM Removal**: Sealed as SUPERSEDED. No pending NIM tasks remain executable.
