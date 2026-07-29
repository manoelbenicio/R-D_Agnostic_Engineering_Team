# FIX deploy-prep — DRAFT runbook (NO deploy executed) + one-clean-run watcher

- agent: Opus48#D · lane: deploy-prep · task: FIX-DEPLOY-PREP · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__FIX-DEPLOY-PREP__20260723T000913Z.json`
- as-of (UTC): 2026-07-23T00:10Z · target: orq1 `/tmp/multica-auth`
- **STATUS: DRAFT ONLY. Nothing executed** — no build, no copy, no restart, no `at` scheduled, no SSH, no observation, no secret read. Per instruction "No deploy until fix built" + standing global prohibition (no deploy/restart/Docker/systemd/inference/secret/commit).

## 0. Hard gates before ANY step below runs (operator/Principal, not this lane)
1. **Fix built + verified**: the authoritative binary is rebuilt from the fixed source and passes the
   §14 backend validation (Go 1.26.1 `go test ./internal/daemon` + `go vet` on brain/gateway/runtimeenv/e2e/pkg/agent). Do not stage an unbuilt/unverified binary.
2. **Deploy authorization**: explicit owner/Principal authorization to deploy to orq1 (this session's control-plane has no deploy authorization; `live_runs.*=false`).
3. **Authorized orq1 access**: a sanctioned operator session on orq1 (no ad-hoc SSH from this lane; no keys/secrets in evidence).
4. **OmniRoute readiness**: `curl -fsS http://127.0.0.1:20128/health` OK on orq1; secret file present at `/etc/agent-brain/secrets/omniroute-inference-key` (owner-provisioned; never printed).

## 1. Capture-then-preserve the current daemon launcher (read-only, on orq1, by operator)
The relaunch MUST reuse the existing launcher verbatim — do not invent it. Before touching anything,
snapshot (no secret values):
```bash
# identify the running daemon + its exact argv and env KEYS (values redacted)
ps -o pid,cmd -C multica-auth 2>/dev/null || pgrep -af multica-auth
# capture the launcher invocation (systemd unit OR the wrapper/launch script OR the at/nohup line)
#   e.g. systemctl cat multica-auth   (if unit-managed)  -> save ExecStart verbatim
#   else: copy the current launch wrapper script path used to start the daemon
# capture ONLY the env KEY NAMES actually set (never values):
tr '\0' '\n' < /proc/$(pgrep -n multica-auth)/environ | cut -d= -f1 | sort -u
```
Preserve exactly (these must be present after restart):
- agent runtimes registered: **`claude-code`** and **`claude_code_kimi_2.7_Code`** (the Kimi-2.7-Code CLI runtime) — same executable paths/args as currently registered.
- gateway env (values owner-managed, never printed): `AGENT_BRAIN_DEVELOPMENT_ENABLED`, `AGENT_BRAIN_CONTROL_URL`, `AGENT_BRAIN_GATEWAY_REQUIRED`, `AGENT_BRAIN_GATEWAY_BASE_URL`, `AGENT_BRAIN_GATEWAY_SECRET_FILE` (→ `/etc/agent-brain/secrets/omniroute-inference-key`), `AGENT_BRAIN_GATEWAY_READINESS_POLICY`, `AGENT_BRAIN_TASK_CAPACITY_TIER`, `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED` (Prodex stays default-OFF, never simultaneous with OmniRoute).

## 2. Stage the fixed binary to orq1:/tmp/multica-auth (after gate §0.1–0.3)
```bash
# from the authorized build host (NOT from this lane); checksum before/after transfer
sha256sum ./multica-auth                      # record digest D_build
scp ./multica-auth orq1:/tmp/multica-auth.new # staged path; do NOT overwrite live binary yet
ssh orq1 'sha256sum /tmp/multica-auth.new'    # MUST equal D_build
ssh orq1 'chmod 0755 /tmp/multica-auth.new'
```
Keep the current live binary for rollback: `ssh orq1 'cp -a <current_binary_path> /tmp/multica-auth.prev'`.

## 3. Draft restart via `at` (scheduled, preserving launcher; NOT scheduled now)
Write a restart script that swaps the binary and relaunches with the PRESERVED launcher (§1), then
schedule it with `at`. Draft only — do not run `at` until §0 gates are GREEN.
```bash
cat > /tmp/multica-restart.sh <<'EOS'
#!/bin/sh
set -eu
LIVE="<current_binary_path>"          # from §1 capture
LAUNCH="<preserved_launcher_cmd>"     # from §1: systemd ExecStart OR wrapper script, verbatim
# 1) graceful stop of the current daemon (unit-managed preferred; else the project's stop hook)
#    systemctl stop multica-auth   ||  <project stop command>   (NO kill -9 unless hung)
# 2) atomic binary swap (rollback copy already at /tmp/multica-auth.prev)
cp -a /tmp/multica-auth.new "$LIVE"
# 3) relaunch with the identical preserved launcher + gateway env (claude-code + claude_code_kimi_2.7_Code)
#    systemctl start multica-auth   ||  exec $LAUNCH
# 4) health gate
curl -fsS http://127.0.0.1:20128/health >/dev/null
curl -fsS http://127.0.0.1:8080/health  >/dev/null
EOS
chmod +x /tmp/multica-restart.sh
# schedule (example; time chosen by operator at authorization):
#   echo '/tmp/multica-restart.sh >> /tmp/multica-restart.log 2>&1' | at -m <HH:MM>
# verify queue:  atq        cancel:  atrm <job>
```
**Rollback:** re-run swap with `/tmp/multica-auth.prev` + relaunch with the same preserved launcher; confirm both health endpoints.

## 4. One-clean-run watcher (read-only, single pass, NO rerun hammering)
Arms on deploy; begins observation only when Kiro deploys AND authorized orq1 access exists (§0.3).
Observes ONE run's lifecycle across the hops and stops at terminal — it never re-enqueues, retriggers,
or re-runs the task, and polls at a bounded low cadence (no hammering):
- **claim → start → CLI → gateway → persist** for the target task, via a read-only view (task_runs /
  daemon logs / task status), matching the OBS hops: queue(dequeue)→admission→cli→route(OmniRoute
  readiness/telemetry only, never provider internals)→persist(terminal/result metadata).
- Single-pass, bounded, idempotent:
```bash
# poll at >=5s, once per interval, until a terminal status; hard cap; then STOP (one clean run).
deadline=$(( $(date +%s) + 1800 ))            # 30-min cap
while [ "$(date +%s)" -lt "$deadline" ]; do
  st=$(<read-only task status for 5f0f1a99-... , e.g. one SELECT status FROM agent_task_queue WHERE id=...>)
  printf '%s %s\n' "$(date -u +%FT%TZ)" "$st"  # append to evidence; metadata only
  case "$st" in completed|failed|cancelled) break;; esac
  sleep 5
done
```
- **No rerun hammering guarantee:** read-only status polling only; never calls claim/enqueue/rerun;
  single watcher instance; exits on first terminal state; fixed ≥5s interval + 30-min cap. Does not
  restart the task or the daemon.
- Capture into this evidence (metadata only, no content/secrets): claim ts, start ts, CLI launch +
  exit-code class, gateway route/model + readiness (pseudonymous telemetry only), persisted terminal
  status + byte/token counts + result_id — mirroring the frozen `agent-brain.e2e.v1` hop schema.

## 5. Non-claims / limitations
- Nothing executed: no build, no scp/ssh, no binary swap, no `at` job, no daemon restart, no task
  observation, no inference, no secret read/print. This is a **draft runbook + armed watcher plan**.
- Live observation and any deploy step require §0 gates (fix built+verified, deploy authorization,
  authorized orq1 access). Absent those, this lane does not touch orq1.
- OmniRoute internals (provider/account/credential/session/rotation/model-mapping) are never
  probed/tested — readiness/telemetry consumed only.
- No product/OpenSpec edit; no commit. `<placeholders>` (current binary path, launcher cmd, read-only
  status query) are filled by the operator from the §1 read-only capture on orq1 — not invented here.
