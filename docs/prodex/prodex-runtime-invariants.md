# prodex Runtime Invariants

Status: PRE-DEPLOY REQUIRED

## Required Invariants

- Profile auth isolation is stronger than convenience.
- Shared Codex state remains upstream-compatible.
- Continuation affinity beats selection heuristics.
- Rotate only before commit.
- Request/stream commit path must not block on broad disk I/O.
- Runtime logs are diagnostics, not source of truth.
- Setup/repair must not mutate profile auth unexpectedly.
- Provider conformance must state lossless/degraded/rejected/unsupported.
- File/SQLite state is single-node only; use Postgres/Redis for shared state.
- Adaptive routing remains shadow unless explicit policy enables live behavior.

## Multica-Specific Additions

- Go is policy owner.
- Rust is runtime decision owner.
- Events do not re-decide committed requests.
- Secrets redaction is deployment blocking.
- Kill switch must override feature enablement.
