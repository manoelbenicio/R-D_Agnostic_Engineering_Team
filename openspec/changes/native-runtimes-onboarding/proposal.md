# Proposal — Native Runtimes (Cline, NIM Superseded) + Model Discovery Fix + Onboarding Rework

## Why
Multica requires native runtime integration for Cline 3.x, reliable UI model discovery, and an onboarding redesign replacing sponsor landing pages and email verification codes with a clean, app-styled login interface.

- **Cline**: Backend native Cline 3.x via `cline --acp`, exchanging ACP JSON-RPC 2.0 messages over stdin/stdout. The `--json` flag belongs to a separate headless prompt-output mode and is incompatible with the ACP handshake when combined with `--acp`.
- **NVIDIA NIM (SUPERSEDED)**: Deliberately removed from canonical source (zero `nim.go`/`nim_home.go` at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`). All obsolete NIM implementation and deployment claims/tasks are SUPERSEDED per owner ruling. Any NIM revival requires a new owner-approved proposal.
- **Model Discovery**: Asynchronous enumeration flow (`agy models` ~20s) fixed with timeouts, caching, and error surfacing so the UI populates reliably.
- **Onboarding**: Sponsor landing pages and email verification code flow removed. Simple username/password login implemented in the design-system style (Firebase-ready).

## What Changes
- **ADDED** native `cline` runtime backend (`cline.go`, config probe, `New`/`SupportedTypes`, `POST /auth/login`).
- **SUPERSEDED** native `nim` runtime (deliberately removed from canonical source; zero `nim.go`/`nim_home.go` at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`).
- **MODIFIED** model discovery: timeout, caching, and error surfacing so UI populates reliably.
- **MODIFIED** onboarding/auth: sponsor landing pages and email-code verification removed; app-styled login implemented.

## Impact
- Code: `server/pkg/agent/cline.go`, `internal/daemon/config.go`, `pkg/agent/agent.go`, `cmd/server/router.go`, `apps/web/app/(auth)`.
- Execs: Expert coder agents for implementation; GTL / Kiro for verification and Kanban control.

## Non-goals
- Operating NIM via opencode or custom wrappers (explicitly rejected by owner decision).
- Claiming frontend or live Cline acceptance before ORQ-66 durable daemon deployment (live CLI is `cline 3.0.46`, but ORQ2 daemon remains old `88ca` until ORQ-66).

## Implementation Status — 2026-07-29 (v6.2 Reconciliation)
- **NIM Removal**: Verified zero `nim.go` or `nim_home.go` files at production SHA `15626386da2725af8e8d4ac611754cffe359fe31`. All NIM tasks marked SUPERSEDED.
- **Cline Backend**: Canonical source contains `cline.go`, config probe, `New`/`SupportedTypes`, and `POST /auth/login`. Live CLI `cline 3.0.46` verified. Pending ORQ-66 durable daemon deployment for live runtime/canary.
- **Onboarding Frontend & Marketing Removal**: Completed and accepted under ORQ-51.
- **Design System Parity**: Completed and accepted under ORQ-52.
