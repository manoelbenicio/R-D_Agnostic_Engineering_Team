---
agent: Codex#5.5#B
stream: G5-F6-QA
phase: G5/F6
task: Smart Context rollout/fallback plan and QA conformance checklist
priority: P0
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:33:48Z
finished_at: 2026-07-04T20:37:00Z
depends_on: docs/prodex/* | docs/qa/smart-context-shadow-canary-plan.md | docs/qa/runtime-conformance-plan.md
blockers: none
build_result: green - Extended G5 and F6 QA docs and ran smoke dry-runs; no product code or deploy.
notes: DONE. G5 and F6 plan/criteria/dry-run delivered; live execution remains F0-GATED.
---

## Scope

- G5: extend `docs/qa/smart-context-shadow-canary-plan.md` with testable shadow to canary to live rollout, exact fallback, and acceptance criteria using prodex native knobs.
- F6: extend `docs/qa/runtime-conformance-plan.md` with C1-C6 QA conformance and PROD validation checklist.
- Mark live execution steps `F0-GATED`.
- Do not edit product code or deploy.

## Result

- G5 completed in `docs/qa/smart-context-shadow-canary-plan.md`: native prodex knobs, shadow/canary/live sequence, exact fallback probes, acceptance criteria, disable actions, and evidence package.
- F6 completed in `docs/qa/runtime-conformance-plan.md`: C1-C6 testable acceptance matrix, common harness requirements, F0-gated PROD validation checklist, pass/fail criteria, stop/rollback conditions, and evidence fields.
- Dry-run evidence recorded at `.deploy-control/evidence/g5-f6-qa-dry-run-20260704T203632Z.md`.

## Dry-Run Evidence

- `bash scripts/smoke/policy-apply-smoke.sh --dry-run`: green.
- `bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context`: green.
- `bash scripts/smoke/event-stream-smoke.sh --dry-run --min-events 1`: green.
- `bash scripts/smoke/redaction-smoke.sh --dry-run`: green.
