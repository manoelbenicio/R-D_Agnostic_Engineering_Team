# G2A Brain Core Evidence

Status: complete. Scope: OpenSpec tasks 3.1-3.5. The neutral execution path remains unwired from the active daemon.

## EV-G2A-01 — Neutral coordinator and runtime boundary

- Added a neutral coordinator, task-executor adapter, and immutable runtime registry around the frozen G1 lifecycle interfaces.
- Runtime selection is keyed by `CLIKind`; duplicate and missing runtime registrations fail deterministically.
- No active daemon entry point imports or invokes the new coordinator.

## EV-G2A-02 — Neutral task contract

- The neutral task reuses the frozen `CLIKind`, `RouteModel`, `RouterOwner`, correlation, and route-policy identifiers.
- Added an explicit approved-route policy carrying policy identity, revision, protocol, and approval state.
- Added opaque lifecycle references for workspace, repositories, worktree, context, skills, recovery, watchdog, stream, and terminal policies. They carry identifiers only, not task content or credentials.

## EV-G2A-03 — Compatibility and measurable legacy use

- Added legacy task and config translations into the neutral task contract.
- Legacy use records fixed-cardinality surface, alias, and outcome counters for translated, shadowed, and rejected paths.
- Measurements do not retain config values, task tokens, prompts, or credential material.

## EV-G2A-04 — Gateway-required fail-closed admission

- Added explicit admission and readiness states for gateway unavailable, authentication failure, capability rejection, and route-policy rejection.
- Gateway-required tasks fail closed to measurable terminal task statuses before executor invocation.
- Legacy non-gateway tasks retain the compatibility admission path without enabling the new execution path.

## EV-G2A-05 — Existing lifecycle preservation

- The lifecycle adapter preserves the existing workspace/repository/worktree, cancellation, watchdog, context/skills, stream batching, recovery, cleanup, and terminal-result responsibilities behind a neutral interface.
- Cancellation propagates to execution while terminal-result publication receives a non-cancelled derived context.
- Provider credential logic remains outside the neutral package.

## Verification

- Focused package tests: pass.
- Focused package vet: pass.
- Repository-wide Go suite: pass.
- Formatting and whitespace validation: pass.
