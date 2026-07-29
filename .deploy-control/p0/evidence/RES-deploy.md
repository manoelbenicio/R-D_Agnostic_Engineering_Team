# RES deploy — STANDBY deploy-prep (fixed artifact → orq1, launcher preserved). NOTHING deployed.

- agent: Opus48#D · lane: deploy-prep · task: RES-DEPLOY-PREP · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__RES-DEPLOY-PREP__20260723T021416Z.json`
- as-of (UTC): 2026-07-23T02:14Z · target: orq1 `/tmp/multica-auth`
- **STATUS: STANDBY / DRAFT. No build, copy, restart, `at`, SSH, or observation performed. No secrets.**
- canonical runbook (single source of truth — reused verbatim, not duplicated):
  `.deploy-control/p0/evidence/FIX-deploy-prep.md` (§1 preserve launcher, §2 stage+checksum+rollback,
  §3 restart via `at` + health gate, §4 one-clean-run watcher, §5 non-claims).

## 1. Candidate fixed artifact (read-only provenance)
Built by another lane (not this lane); recorded for the §2 checksum step:
| Path | sha256 | type | mtime |
|---|---|---|---|
| `/tmp/multica-auth-fixed` | `b554ee52802db7f9b73caef3e2c15cbd9720853db4a86c2fcc662570d935b7b4` | ELF 64-bit LSB executable | 2026-07-23T00:13 |
| `/tmp/multica-auth` (prior) | `bd5c38481868c46fc6a1130ffe61cf5ea0df2e49d909fab664ff274e99cacd61` | ELF 64-bit LSB executable | 2026-07-22T22:02 |

The fixed candidate to stage is `/tmp/multica-auth-fixed` (digest `b554ee52…`). This digest MUST be
re-verified (a) after transfer to `orq1:/tmp/multica-auth.new` and (b) against the build host's record
before the swap. Note: the candidate must still pass the §14 backend validation (Go 1.26.1
`go test ./internal/daemon` + vet) as its "fix built + verified" gate — provenance of that run is
owned by the build/verify lane, not asserted here.

## 2. Current gate state (read-only `control.json` @02:14Z)
- `implementation_authorized = true` (offline engineering).
- **`deploy_authorized` = unset → NO deploy authorization.**
- `live_runs_authorized` = unset; all live-run families `authorized:false`.
- No authorized orq1 (`100.118.244.61`) access path granted to this lane.
⇒ Standby holds. Do not stage/swap/restart until deploy authorization + authorized orq1 access.

## 3. Standby posture (ready-to-execute on authorization; no action now)
On ALL gates GREEN — (a) fixed build verified, (b) `deploy_authorized=true`, (c) authorized orq1
operator session, (d) OmniRoute health OK — execute the canonical `FIX-deploy-prep.md` steps in order:
1. §1 capture-then-preserve the live launcher (systemd `ExecStart`/wrapper) + runtimes **`claude-code`**
   and **`claude_code_kimi_2.7_Code`** + `AGENT_BRAIN_*` gateway env (values never printed).
2. §2 `scp /tmp/multica-auth-fixed → orq1:/tmp/multica-auth.new`; verify sha256 `b554ee52…`; keep
   `/tmp/multica-auth.prev` rollback copy.
3. §3 `at`-scheduled restart script: graceful stop → atomic swap into the live path → relaunch with the
   PRESERVED launcher/env → health gate (`:20128` + `:8080`); rollback path documented.
4. §4 arm the one-clean-run read-only watcher (claim/start/CLI/gateway/persist/terminal; ≥5s poll,
   30-min cap, exit on first terminal; never claim/enqueue/rerun/restart).

## 4. Non-claims / scope
- Nothing executed: no build, no scp/ssh, no swap, no `at`, no restart, no observation, no inference,
  no secret read/print. Standby only.
- Deploy gated on `deploy_authorized` + authorized orq1 access (both absent now).
- OmniRoute internals never probed (readiness only). No product/OpenSpec edit; no commit.
- No divergent second runbook created — `FIX-deploy-prep.md` remains the single authoritative procedure.
