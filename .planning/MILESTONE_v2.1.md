# MILESTONE v2.1 — Full Vendor Validation + PROD Deploy

> Created by Kiro (Principal) 2026-07-05 per owner directive. Full GSD, fully agentic, max parallel.
> Predecessor: v2.0 (Rotation-Parity Polyglot) — remediation complete (commit 6ba9a70), Smart Context REAL proven.

## Why
v2.0 delivered the real prodex L2 runtime with proven Smart Context (tokens_saved 4,139/16,476/65,827).
Two items remained: (F5) 8 vendor-capability cells `not_validated`, and (F7) PROD deploy not executed.
Owner directive: **NO owner-gate, NO later** — validate ALL capabilities for real and deploy+test ALL
features TODAY. `not_validated` is a gap to CLOSE by behavioral validation, not to accept.

## Scope (Definition of Done)
- **ZERO `not_validated` cells** — every capability VERIFIED by behavior with container/live evidence
  (per qa-conformance C1: validate by behavior, not label).
- **All features deployed to PROD and tested** with a real provider-backed session + kill-switch/rollback live.
- Full GSD trail (PLAN + SUMMARY + evidence) per phase; committed to origin/main.

## New Requirements
- **REQ-42** Every vendor capability in the matrix SHALL be `verified` by empirical behavior through the
  real prodex runtime (`/v1/runtime/proxy`) with captured evidence; `not_validated` is not an acceptable
  end state. OpenCode (archived) SHALL be migrated to Crush OR documented superseded.
- **REQ-43** The system SHALL be deployed to PROD and validated with a real provider-backed session;
  kill-switch + rollback proven live; logs scrubbed.

## Roadmap (phases)
- **P11 — Vendor Capability Live Validation** (dep: v2.0 runtime) — close all 8 not_validated cells.
- **P12 — PROD Deploy + Live Test** (dep: P11 green) — deploy + real session + kill-switch/rollback live.

## Parallelism / staffing (6 elite Codex, effective 4)
- P11: Codex-A→OpenAI/Codex, Codex-C→Kiro, Codex-D→Antigravity, Codex-E→Cline(+OpenCode/Crush) — PARALLEL.
- Codex-B = sole Rust-hotspot owner (fixes if a capability fails validation).
- P12: 1 agent executes deploy once P11 green.
- Evidence-gated: TL re-runs; no [x] without disk evidence. Fully agentic (agents execute, TL orchestrates).
