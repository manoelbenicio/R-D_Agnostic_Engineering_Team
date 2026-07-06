agent: Codex#5.5#D
stream: G10-DEVOPS
phase: F0/F7 operational procedures
task: deliver rollback, sidecar-health, and kill-switch operational procedures with dry-run steps and LIVE F0 gate
priority: P0
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:30:21Z
finished_at: 2026-07-04T20:32:55Z
depends_on: F0 owner gate, F7 deploy/runbook approval, live sidecar smoke authorization
blockers: LIVE execution remains F0-gated; deploy_owner_approved=false
build_result: green - docs delivered; dry-run command references validated; git diff --check clean; no deploy or LIVE smoke executed
notes: Delivered rollback, sidecar-health, and kill-switch operational procedures under docs/deploy. LIVE steps remain F0-gated and require owner approval plus script execution gates.
ack: Codex#5.5#D @ 2026-07-04T20:30:21Z  status: ACKNOWLEDGED
herdr-comms-ack: Codex#5.5#D @ 2026-07-04T20:30:21Z  status: ACKNOWLEDGED

## Deliverables

- docs/deploy/rollback-operational-procedure.md
- docs/deploy/sidecar-health-operational-procedure.md
- docs/deploy/kill-switch-operational-procedure.md

## Verification

- `bash scripts/smoke/readyz-smoke.sh --dry-run --base-url http://127.0.0.1:43117` PASS
- `bash scripts/smoke/state-backend-smoke.sh --dry-run --base-url http://127.0.0.1:43117` PASS
- `bash scripts/smoke/event-stream-smoke.sh --dry-run --base-url http://127.0.0.1:43117` PASS
- `bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context` PASS
- `SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature gateway` PASS
- `SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature auto_redeem` PASS
- `SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature provider_bridge` PASS
- `SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature runtime_proxy` PASS
- `bash scripts/smoke/redaction-smoke.sh --dry-run` PASS
- `git diff --check -- docs/deploy/rollback-operational-procedure.md docs/deploy/sidecar-health-operational-procedure.md docs/deploy/kill-switch-operational-procedure.md .deploy-control/Codex-5.5-D__G10-DEVOPS__20260704T203021Z.md` PASS

## Artifact Hashes

- docs/deploy/rollback-operational-procedure.md: 6a0ce940df2f083303d1a000edba0f3b971bbacd2062effd9123f21498118d5d
- docs/deploy/sidecar-health-operational-procedure.md: 200ea35ccf4b94c99b755e2c34955b47aefdc6598c4c7c39f4464f856912a555
- docs/deploy/kill-switch-operational-procedure.md: 39af13b40c8080d7faef7f5ae4454b12e52df4c141e434e19fbee9941274b6c3
