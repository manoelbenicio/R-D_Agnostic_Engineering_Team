# P0 Fleet assignment matrix — saturation preflight

updated: 2026-07-21T22:27:00Z
phase: PREFLIGHT_AND_GAP_MATRIX
implementation_source_edits: BLOCKED until `FLEET_SATURATED=GREEN`

## Management boundary

- Principal-Orchestrator: owns coverage, assignment, arbitration, A10 traceability and final decisions; **no product code**.
- Opus48-Kiro: owns 10-minute supervision and bounded claim audit; **no product code**.
- `wA:p1`/`wA:p2`: inventory-only shell panes, no agent process; excluded from eligible-agent denominator.

## Assignments

| Worker | Pane | Bundle | Exact output lock | Covers | Required preflight |
|---|---|---|---|---|---|
| Opus48#A | `w6:p1` | A1+A2 Cline foundation/mapping | `handoffs/A1-A2-cline-foundation.md` | 5.6, 5.7 | git, python3, rg, Herdr, Go 1.26.1 |
| Opus48#B | `w6:p2` | W1+A9 integration/validation matrix | `handoffs/W1-A9-integration-validation.md` | 5.6–5.8, 8.1–8.2 | git, Python, rg, OpenSpec, Go 1.26.1, pnpm 10.28.2 |
| Codex56#A | `w7:p3` | A3 exact GLM/Kimi route freeze | `handoffs/A3-route-freeze.md` | 5.6, 5.7, 8.1 | git, Python, rg, Herdr; registry access or blocker |
| Codex56#B | `w7:p4` | A4 Opus48 Anthropic route | `handoffs/A4-opus48.md` | 5.8, 8.1, 8.2 | git, Python, rg, Go 1.26.1; registry access or blocker |
| Opus48#C | `w8:p1` | A5 Antigravity equivalence | `evidence/A5-antigravity-equivalence.md` | 5.8, 8.1, 8.2 | git, Python, rg, sha256sum |
| Opus48#D | `w8:p2` | A6 Main Brain lifecycle gap | `handoffs/A6-lifecycle-gap-matrix.md` | 8.2 | git, Python, rg, Go 1.26.1 |
| Agy-P0-A7 | `wB:p1` | A7 production-integrity frontend | `handoffs/A7-production-integrity-frontend.md` | P0 support | git, Python, rg, Node 22, pnpm 10.28.2 |
| Agy-P0-A8 | `wB:p2` | A8 production-integrity backend | `handoffs/A8-production-integrity-backend.md` | P0 support | git, Python, rg, Go 1.26.1, OpenSpec |

Paths are relative to `.deploy-control/p0/`. Every worker locks only its output artifact in this phase and reads product source read-only. This proves zero write overlap before source ownership is frozen.

## Toolchain

- Go: `/home/ec2-user/goroot/go/bin/go`, verified `go1.26.1 linux/amd64` (use absolute path; agent `$HOME` is credential-slot-specific).
- gofmt: `/home/ec2-user/goroot/go/bin/gofmt`.
- Node: `v22.23.1`.
- pnpm: run from `multica-auth-work/`; project pin resolves to `10.28.2`.
- OpenSpec: `1.4.1`.
- Docker: absent and not required for the preflight/gap-matrix phase. Any later check that truly requires a container is blocked until explicitly provisioned or replaced by an equivalent local command.

## Gate to source implementation

The Principal may freeze source locks only after all eight workers have: check-in, tool preflight in their artifact, Herdr `working|blocked`, and a first concrete finding. Kiro verifies the gate; Principal adjudicates. Principal/Kiro do not write source code.
