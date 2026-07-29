# KANBAN watch + DIRECT CODEX live acceptance — BLOCKED (no authorized live run; ORQ1 SSH unauthorized)

- agent: Opus48#D · lane: kanban-watcher · task: KANBAN-WATCH-5f0f1a99 · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__KANBAN-WATCH-5f0f1a99__20260722T215728Z.json`
- as-of (UTC): 2026-07-22T21:57Z · git HEAD `a6d5098`
- request: (1) execute live explicit Codex direct-chat acceptance vs the distinct Codex member + collect route/response-persistence evidence; (2) read-only watch task-runs for live task `5f0f1a99-8bc8-42ab-9cc0-a57ca7e9854a` on ORQ1 (`ssh 100.118.244.61`), capturing claim→launch→omniroute→completed + persisted terminal/result.
- verdict: **BLOCKED — not executed. No fake PASS produced.**

## 1. Why blocked (control-plane facts, verified — not assumed)
From `.deploy-control/p0/control.json` at this time:
- `live_runs_authorized = false` (top-level).
- `live_runs`: `cline_glm`, `cline_kimi`, `opus48`, `antigravity` — **every family `authorized: false`**.
- **No `codex` live-run family exists** in `live_runs`.
- No authorization for ORQ1 / `100.118.244.61` / SSH / task `5f0f1a99…` anywhere in `control.json` or `.deploy-control/`.

Standing hard gates (PROTOCOL / SIX_HOUR_CHECKIN_CONTRACT §8 / every dispatch this session):
- **"No inference while `live_runs.*=false`."** A live Codex direct-chat acceptance IS inference → prohibited.
- A live execution is allowed only with a `control.json` route-family token `live_run.authorized=true` + exact RouteModel + integrated provenance (PROTOCOL "Live-run token"). No such token exists for Codex.
- Global prohibitions: no deploy/restart/Docker/systemd; treat remote/production access as high-risk. **SSH into ORQ1 (`100.118.244.61`)** — a remote/shared orchestrator — is unauthorized remote access from this lane (no authorized credential path; not a local read).

## 2. What I did NOT do (fail-closed)
- Did not run any inference / live Codex chat acceptance.
- Did not SSH to `100.118.244.61` or any remote host; did not read/print/hash secrets or SSH keys.
- Did not fabricate a route/persistence PASS or synthesize a claim→launch→omniroute→completed trace. Without an authorized live run, that trace does not exist to capture; manufacturing one would be a fabricated acceptance (explicitly forbidden).

## 3. Blocker → owner → next action
- blocker: `Live Codex direct-chat acceptance + ORQ1 task watch require (a) an authorized codex live-run token in control.json and (b) explicit authorization + a read-only access path for ORQ1 (100.118.244.61); both absent (live_runs.*=false, no codex family, no ORQ1/SSH authorization).`
- owner: **Principal Orchestrator (`w5:p9`)** for live-run authorization; **OmniRoute/Product security** for the provider route + any ORQ1 access grant.
- next action (to unblock, in order):
  1. Principal adds a `codex` family to `control.json.live_runs` with `authorized=true`, the exact accepted Codex RouteModel, and integrated build/commit/config provenance (or flips the relevant existing family), and sets `live_runs_authorized=true` for that family.
  2. Provide an authorized, read-only observation path for the ORQ1 task (e.g. a sanctioned read-only DB/API/log view, or explicit SSH authorization with a scoped key) — no secret values in evidence.
  3. On both, this lane will: record the run ID before launch, observe claim→launch→omniroute(readiness/telemetry only, no provider internals)→completed, and capture the persisted terminal/result metadata (no content/secrets) into this file, then check out with reproducible provenance.

## 4. Non-claims / scope
- OmniRoute internals (provider/account/credential/session/rotation/model-mapping) are never probed/mapped/tested — readiness/pseudonymous telemetry only, and only under an authorized run.
- No product/source edit; no OpenSpec checkbox; no commit. Root disk at ~100% used (writes kept minimal, on `.deploy-control`).

## 5. Armed single-run watcher — status @2026-07-23T00:17Z (read-only gate check)
Requested to stay active and capture the next strict-ready window + Kiro single clean task
(claim/start/CLI/gateway/persist/terminal) without triggering reruns. Read-only `control.json` check:
- `strict_ready` = none · `window` = none → **no strict-ready window open**.
- `live_runs_authorized` unset; all families `authorized:false`; no `codex` family → **no authorized run**.
- No authorized orq1 (`100.118.244.61`) read path granted to this lane.
- A separate `LIVE-watch.md` lane exists → to avoid a **duplicate/hammering** watcher on the same task,
  this lane stays armed and defers unless assigned the primary observation.

**Posture:** ARMED, single-clean-run, read-only. It will capture exactly one run when ALL hold: (a) an
authorized read-only observation path to the task; (b) `strict_ready`/window open with an authorized
run; (c) this lane is the designated (non-duplicate) watcher. It performs status-only polling
(≥5s, 30-min cap, exit on first terminal), and NEVER claims/enqueues/reruns/restarts. Until the window
opens it takes no action (no speculative SSH, no rerun, no fabricated trace).
