# T20 deploy — STANDBY tier-20 deploy-prep (fixed artifact → orq1). NOTHING deployed.

- agent: Opus48#D · lane: tier20-deploy · task: T20-DEPLOY-PREP · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__T20-DEPLOY-PREP__20260723T210918Z.json`
- as-of (UTC): 2026-07-23T21:09Z · target: orq1 `/tmp/multica-auth`
- **STATUS: STANDBY / DRAFT. No build, copy, restart, `at`, SSH, env-set, or observation performed. No secrets.**
- canonical runbook (single source, reused — not duplicated): `.deploy-control/p0/evidence/FIX-deploy-prep.md`
  (§1 preserve launcher, §2 stage+checksum+rollback, §3 restart via `at` + health gate, §4 watcher).
  Prior standby: `.deploy-control/p0/evidence/RES-deploy.md`.

## 1. Candidate fixed artifact (read-only provenance — REBUILT since RES-deploy)
| Path | sha256 (current) | type | mtime |
|---|---|---|---|
| `/tmp/multica-auth-fixed` | `1233b118431946ec56de7590d7dde5cd2544f69a84e735fc9dfa6d27dc072219` | ELF 64-bit LSB executable | 2026-07-23T02:20 |
Stage this digest; re-verify after transfer to `orq1:/tmp/multica-auth.new` and against the build
host's record before swap. The fixed build must still pass its §14 backend verify gate (owned by the
build/verify lane; not asserted here). (Supersedes the earlier `b554ee52…`/`bd5c3848…` candidates.)

## 2. Tier-20 + capacity-gate env (to set on the preserved launcher; NOT set now)
Add to the preserved daemon env (§1 of the canonical runbook), values non-secret:
- `AGENT_BRAIN_TASK_CAPACITY_TIER=20`  (schema tier; `brain.EnvTaskCapacityTier`; config default is already `20`).
- capacity-gate / fail-closed admission env (must accompany tier-20 so admission stays gated):
  - `AGENT_BRAIN_GATEWAY_REQUIRED=true`
  - `AGENT_BRAIN_GATEWAY_READINESS_POLICY=strict`
  - `AGENT_BRAIN_GATEWAY_BASE_URL` (host default `http://127.0.0.1:20128`)
  - `AGENT_BRAIN_GATEWAY_SECRET_FILE=/etc/agent-brain/secrets/omniroute-inference-key` (owner-provisioned; never printed)
  - `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED=false` (Prodex default-OFF, never simultaneous with OmniRoute).

**Critical (do not over-claim):** `AGENT_BRAIN_TASK_CAPACITY_TIER=20` sets the tier **schema** only. The
daemon's `effectiveTaskAdmissionLimit` (config.go:664) **fail-closes to the development admission limit
(`agentBrainDevelopmentMaxTasks`) regardless of tier=20**. Real 20-concurrency admission is a separate
ACCEPTANCE gate (OpenSpec 9.1 run + 9.2 enable, entry-gated on G4-OBS PASS) and is **not** enabled by
this env. `capacity_tier_authorized` is currently unset.

## 3. Current gate state (read-only `control.json` @21:09Z)
- `implementation_authorized=true`; **`deploy_authorized`=unset (NO deploy)**; `live_runs_authorized`=unset;
  **`capacity_tier_authorized`=unset (tier-20 admission NOT accepted)**; no authorized orq1 access path.
⇒ Standby holds on: (a) fixed build verified, (b) `deploy_authorized=true`, (c) authorized orq1 session,
(d) OmniRoute health OK, and — for actually RUNNING at tier 20 — (e) G4-OBS PASS + 9.1/9.2 capacity acceptance.

## 4. Standby execution order (on authorization; nothing now)
1. Canonical §1 capture-then-preserve launcher (runtimes `claude-code` + `claude_code_kimi_2.7_Code`, `AGENT_BRAIN_*` env).
2. Add the §2 tier-20 + capacity-gate env to the preserved launcher env.
3. Canonical §2 stage `/tmp/multica-auth-fixed` (sha256 `1233b118…`) → `orq1:/tmp/multica-auth.new`, verify, `.prev` rollback.
4. Canonical §3 `at`-scheduled restart with preserved launcher + tier-20 env → health gate (`:20128`+`:8080`).
5. Canonical §4 arm the one-clean-run read-only watcher (no rerun/hammering).
6. Tier-20 concurrency remains fail-closed at the dev limit until the 9.1/9.2 capacity-acceptance gate is separately authorized.

## 5. Non-claims / scope
- Nothing executed: no build, no scp/ssh, no env-set on orq1, no swap, no `at`, no restart, no observation, no inference, no secret read.
- Setting `AGENT_BRAIN_TASK_CAPACITY_TIER=20` does NOT enable 20-task capacity (fail-closed to dev limit until 9.1/9.2 acceptance).
- OmniRoute internals never probed (readiness only). No product/OpenSpec edit; no commit. Single canonical runbook preserved (no divergent copy).
