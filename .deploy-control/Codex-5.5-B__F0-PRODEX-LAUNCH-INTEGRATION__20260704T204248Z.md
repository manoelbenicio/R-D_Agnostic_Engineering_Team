---
agent: Codex#5.5#B
stream: F0-PRODEX-LAUNCH-INTEGRATION
phase: F0-prep
task: prodex AS-IS pinned launch integration spec for Multica Go
priority: P0
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:42:48Z
finished_at: 2026-07-04T20:44:39Z
depends_on: docs/prodex/prodex-l2-facade.md | docs/prodex/prodex-pin-integrity.md | docs/deploy/*
blockers: none
build_result: green - Added documentation-only prodex AS-IS F0 launch integration spec; no deploy run.
notes: LIVE launch remains F0-GATED. Unconfirmed prodex AS-IS wiring marked a validar.
---

## Scope

- Add `docs/prodex/prodex-launch-integration.md`.
- Specify how Multica Go launches pinned prodex AS-IS in place of raw Codex.
- Cover launch command, env mapping, profile-pool wiring, ordered launch sequence, and rollback to raw Codex.
- Official prodex docs only; unconfirmed items marked `a validar`.

## Result

- Added `docs/prodex/prodex-launch-integration.md`.
- Covered pinned version/commit, launch commands, env mapping, ext4 filesystem requirements, profile-pool wiring, ordered F0 launch sequence, rollback to raw Codex, block conditions, and non-goals.
- Cross-referenced `docs/prodex/prodex-l2-facade.md`, `docs/prodex/prodex-pin-integrity.md`, `docs/deploy/l2-sidecar-deploy-plan.md`, `docs/deploy/prod-rollout-runbook.md`, and `docs/deploy/rollback-runbook.md`.
- No product code changed; no live launch or deploy executed.
