---
phase: 06-qa-conformance
plan: 06
status: DONE
finished_at_utc: 2026-07-05T06:27:39Z
requirements: [REQ-18, REQ-26]
artifacts:
  - .deploy-control/evidence/herdr-smoke.md
  - .deploy-control/evidence/mcp-conformance.md
---

# PLAN 06-06 SUMMARY - P6 Tasks 6.7 and 6.8

## Result

Tasks 6.7 and 6.8 are complete with scrubbed evidence.

## 6.7 Herdr Coordination

Live Herdr commands passed:

- `herdr agent list`: 9 agents discovered.
- `herdr pane list`: 10 panes discovered.
- `herdr agent send Gemini#PRO#31 ...`: `type=ok`.
- `herdr notification show ...`: notification shown.
- `herdr wait agent-status w3:pN --status idle --timeout 1000`: event received.

Evidence: `.deploy-control/evidence/herdr-smoke.md`.

## 6.8 MCP Conformance

Local L2/MCP smokes passed against the real
`multica-auth-work/prodex-sidecar/target/release/prodex-sidecar` process on an
ephemeral loopback port:

- `readyz-smoke.sh`: PASS.
- `policy-apply-smoke.sh`: PASS.
- `session-start-stop-smoke.sh`: PASS.
- `event-stream-smoke.sh`: PASS, `validated_events=2`.

Focused Go tests passed:

- `./internal/l2runtime`.
- `./internal/daemon`.
- `./cmd/multica`.
- `./internal/daemon/execenv`.

Rust provider conformance in pinned prodex source passed:

- `provider_conformance`: 13 passed.
- `provider_conformance_v1`: 14 passed.

Evidence: `.deploy-control/evidence/mcp-conformance.md`.

## Files Changed

- `.deploy-control/evidence/herdr-smoke.md`
- `.deploy-control/evidence/mcp-conformance.md`
- `.planning/phases/06-qa-conformance/06-06-PLAN.md`
- `.planning/phases/06-qa-conformance/06-06-SUMMARY.md`
- `multica-auth-work/server/internal/daemon/execenv/cursor_mcp.go`
- `openspec/changes/rotation-parity-polyglot/tasks.md`

## Scope Guard

Only tasks 6.7 and 6.8 were marked complete. S1-S5 and P6 gate task 6.9 remain
open.
