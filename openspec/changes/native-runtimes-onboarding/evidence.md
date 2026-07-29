# OpenSpec Evidence — Native Runtimes Onboarding (v6.2)

> **Date (UTC):** 2026-07-29T17:45:29Z  
> **Mandate Reference:** `[GTL-DOCUMENTATION-STEWARD-V6.2-20260729]` (`415cc8b2-063c-4182-a935-dd64907b6921`)

## 1. Owner Ruling on NVIDIA NIM (SUPERSEDED)
- Verified canonical production source tip `15626386da2725af8e8d4ac611754cffe359fe31` (and baseline `112e8dada455b4e7a3400e63728e00e6e3a0aa27`).
- Zero `nim.go` or `nim_home.go` files exist in production source.
- NVIDIA NIM was deliberately removed from canonical source per owner ruling. All obsolete NIM implementation and deployment claims/tasks are **SUPERSEDED**, not pending executable work. Any NIM revival requires a new owner-approved proposal.

## 2. Native Cline 3.x Runtime Status
- Canonical backend source contains `server/pkg/agent/cline.go`, config probe (`MULTICA_CLINE_PATH`/`cline`), `agent.go` `New()` cases, `SupportedTypes`, and `cmd/server/router.go` `POST /auth/login`.
- Live CLI version verified: `cline 3.0.46`.
- Live runtime execution and canary remain pending the ORQ-66 durable daemon deployment (ORQ2 daemon remains old `88ca` until ORQ-66). No frontend or live Cline acceptance is claimed prematurely.

## 3. Onboarding & Design System Acceptance
- **Onboarding Frontend Auth & Marketing Removal**: Accepted via ORQ-51.
- **Design System Color & UI Parity**: Accepted via ORQ-52.
- **Backend Auth (`POST /auth/login`)**: Simple username/password authentication implemented behind `AuthProvider` interface (Firebase-ready).
