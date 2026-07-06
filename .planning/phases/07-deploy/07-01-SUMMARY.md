---
phase: 07-deploy
plan: 01
status: DONE
progress: 100
started_at: 2026-07-05T05:00:00Z
finished_at: 2026-07-05T05:19:41Z
requirements: [REQ-19, REQ-20]
evidence:
  - .deploy-control/evidence/p7-kill-switch-test.md
  - .deploy-control/evidence/p7-rollback-test.md
---

# 07-01 Summary

P7 7.1 and 7.2 are complete.

## Completed

- Corrected and rebuilt `multica-auth-work/prodex-sidecar` so the local
  rpp.l2.v1 control surface includes health, readiness, policy apply, account
  registration, session start/stop, event stream, kill-switch apply, and
  kill-switch status.
- Added `scripts/smoke/p7-kill-switch-exercise.sh` to test tenant/provider/profile
  kill-switch behavior against a real local sidecar process.
- Added `scripts/deploy/rollback-to-raw-codex.sh` as the one-command raw Codex
  rollback action.
- Updated `scripts/smoke/rollback-smoke.sh --execute` to exercise that command
  in a temporary env-file harness and verify rollback plus backup restore.

## Verification

- `bash scripts/smoke/p7-kill-switch-exercise.sh`: PASS.
- `SMOKE_ALLOW_EXECUTE=1 DEPLOY_OWNER_APPROVED=true bash scripts/smoke/rollback-smoke.sh --execute`: PASS.
- `bash scripts/smoke/rollback-smoke.sh --dry-run`: PASS.
- `bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context`: PASS.
- `cargo build --release`: PASS.
- `cargo test`: PASS.
- `/home/dataops-lab/.cache/codex-go/go/bin/go test ./internal/l2runtime ./internal/daemon`: PASS.

## Remaining Scope

P7 tasks 7.3, 7.4a, 7.4e, 7.5, 7.6, and 7.7 remain open.
