# Status: rotation-router

> **Status:** SUPERSEDED
> **Superseded by:** `openspec/changes/rotation-parity-polyglot/`
> **Decision date:** 2026-07-04
> **Decision authority:** Product Owner (Manoel Benicio) + Codex R&D Engineering Team + Orchestration (Opus 4.8)
> **ADR reference:** `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`

## Summary

The `rotation-router` change (policy-driven Go runtime router) is **SUPERSEDED**. Its runtime authority — selection, rotation, fallback, load-balancing, and proactive reset of requests in-flight — has been **absorbed by `prodex`/Rust L2** under the polyglot architecture (ADR-001).

## What Is Preserved (Go L4 Control Plane)

The following responsibilities **remain valid** in the Go L4 control plane (Multica):

- **Account Registry** — approved accounts per tenant (migration 124)
- **Policy definition** — RotationPolicy types (fallback/load-balancing/latency)
- **Observability** — KPI Savings, cost/volume/tokens/latency per account/vendor/task
- **Governance** — tenant-level account approval, audit trail

## What Is Superseded (Runtime → Rust L2)

The following responsibilities **transfer to prodex/Rust L2**:

- Request-in-flight selection and routing
- Fallback with retry/backoff
- Pre-commit rotation
- Session affinity / hard continuation binding
- Smart Context (shadow/canary/live)
- Reset-claim (`prodex redeem`)

## Reason

Per ADR-001 (Alternative A+D chosen):
- Hot path (proxy/Smart Context/gateway) requires GC-free runtime → Rust
- Go L4 is cold path (control plane); rewriting hot path in Go = risk + months
- `prodex` (Apache-2.0) already implements all runtime rotation features
- Polyglot architecture: Go decides desired state; Rust decides in-flight

## History

| Date | Event |
|---|---|
| 2026-06-XX | rotation-router proposed (Go-only design) |
| 2026-07-04 | ADR-001 accepted — polyglot architecture (Go L4 + Rust L2) |
| 2026-07-04 | rotation-router SUPERSEDED by rotation-parity-polyglot |

## Related Files

- `openspec/changes/rotation-router/proposal.md` — original proposal (has SUPERSEDED banner)
- `openspec/changes/rotation-router/design.md` — original design (reference/legacy)
- `openspec/changes/rotation-router/tasks.md` — original tasks (no longer active)
- `openspec/changes/rotation-parity-polyglot/` — successor change
- `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md` — architecture decision
