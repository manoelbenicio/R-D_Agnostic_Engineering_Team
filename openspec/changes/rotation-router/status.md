# Status: rotation-router

> **Status:** SUPERSEDED
> **Current documentation successor:** `openspec/changes/document-multica-gateway-selection-lifecycle/`
> **Decision date:** 2026-07-04
> **Decision authority:** Product Owner (Manoel Benicio) + Codex R&D Engineering Team + Orchestration (Opus 4.8)
> **Historical successor:** `rotation-parity-polyglot` tree `60315448dc1929dd7c5bb95453637bda7232ab2d`
> at commit `52cdd877e1941872ac4651df5360ff885afd16ad`
> **Historical ADR:** blob `f0f2083116b8cc9025995514f5bdde8ec7d51748` at that commit

## Current-lineage correction (2026-08-02)

The historical successor and ADR named above are not present in the current tree; they
were removed by commit `9ab80a6f0d34d594193855cf1e659ee633a45ccc`. The accepted current
documentation lineage is `openspec/changes/document-multica-gateway-selection-lifecycle/`,
whose authority preserves the Multica implementation AS-IS and does not authorize runtime,
credential, production, or architecture changes. The 2026-07-04 rationale below is retained
as historical context and does not override that current authority.

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
| 2026-07-24 | historical rotation-parity-polyglot files removed from the current tree by `9ab80a6f0d34d594193855cf1e659ee633a45ccc` |
| 2026-08-02 | current documentation successor corrected to `document-multica-gateway-selection-lifecycle` |

## Related Files

- `openspec/changes/archive/2026-07-04-rotation-router/proposal.md` — archived original proposal
- `openspec/changes/archive/2026-07-04-rotation-router/design.md` — archived original design
- `openspec/changes/archive/2026-07-04-rotation-router/tasks.md` — archived original tasks
- `openspec/changes/document-multica-gateway-selection-lifecycle/` — accepted current documentation successor
- Historical rotation-parity-polyglot change tree: `60315448dc1929dd7c5bb95453637bda7232ab2d`
- Historical ADR blob: `f0f2083116b8cc9025995514f5bdde8ec7d51748`
