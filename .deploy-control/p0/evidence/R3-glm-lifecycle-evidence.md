# A9-R3-EVIDENCE — GLM Kanban→terminal 8.2 lifecycle evidence consumer (BLOCKED)

- agent: Opus48#D · lane: A9-R3-EVIDENCE · task: P0-R3-LIVE-EVIDENCE-CONSUMER
- pane: `w8:p2` (HERDR_ENV=1)
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-R3-LIVE-EVIDENCE-CONSUMER__20260722T000345Z.json`
- lock (only mutable file): `.deploy-control/p0/evidence/R3-glm-lifecycle-evidence.md`
- role: **consumer/interpreter only.** Map the Main-Brain-owned portion of task `8.2`
  (tools · reasoning · cancellation · usage · deterministic error / terminal result) from the
  **one** authorized `Codex56#A` `Cline → GLM-5.2` live run. **I do not initiate, rerun, or reserve
  that run.** No source/test/live action now.

## 0. Preflight
git 2.50.1 · python3 3.9.25 · rg 15.2.0. (No Go build/live action in this lane.)

## 1. Fail-closed safety net already in place (equivalent unit evidence)

The single live run is the *positive* proof; the R3 unit suite is its fail-closed complement,
already green (`R3-test-implementation.md`, HEAD `a6d5098` + W1 D1–D7):
- **tools/reasoning intent gating:** `runtimeenv` `GatewayModelPolicy.ValidateSelection` (thinking
  fail-closed) + `admitTask` `registry.ValidateCapability(Tools:true)` — unit-proven.
- **launch materialization:** `TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret` (+ child
  isolation) — controlled `CLINE_DATA_DIR`, injected `CLINE_OMNIROUTE_API_KEY`, sentinel-only carrier.
- **deterministic error/NVIDIA exclusion:** `TestBuildLaunchOpenAICompatibleRejectsNVIDIAOwnedRoute`,
  admission fail-closed classes.
- **cancel/usage/terminal:** reused equivalents — `TestAgentBrainCentralCapacityReconcilesOverloadAndCancellation`,
  `daemon_test.go::TestExecuteAndDrain_ContextCancelled_ReportsCancelled`, `TestMergeUsage`, `reportTaskResult` fail-closed.

These bound what the live run must merely *confirm on the wire* — they do not substitute for it, and it
does not duplicate them.

## 2. 8.2 interpretation plan — facts to extract from the ONE authorized GLM run (on availability)

Consume the **same** authorized run's sanitized terminal evidence (per PROTOCOL live-run token; the
`5.6`/`8.1`/`8.2` requirements reuse that single run). Map, do not re-execute:

| 8.2 axis | Fact to read from the run's sanitized evidence | Main-Brain boundary |
|---|---|---|
| tools | a real tool turn occurred over the OpenAI Chat Completions route (Brain passed intent; `ValidateCapability` admitted) | tool *conformance* is OmniRoute/provider; Brain owns intent+admission |
| reasoning | reasoning/thinking handled per policy (gateway slice: thinking fail-closed unless approved) | Brain owns the gate, not provider reasoning internals |
| cancellation | mid-run cancel → process terminated, slot/capacity released, `EventCancellation` emitted once, terminal status `cancelled` | Brain-owned process cancel + slot cleanup |
| usage | usage entries reported on the terminal path (`ReportTaskUsage`), non-content | Brain reports; token accounting values are OmniRoute-owned |
| deterministic error / terminal result | fail-closed terminal mapping (only `completed`=success), typed error class, result persisted + delivered to UI | Brain-owned terminal reporting |

Output on unblock: a short mapping of each axis → the exact sanitized artifact line/field + the R3 unit
test that is its fail-closed complement; declare `8.2 (Main-Brain portion) SATISFIED` only when the one
run's evidence covers the changed GLM route. No second run, no failure injection by Multica.

## 3. Ownership boundary (hard exclusions)
Not consumed/asserted here: authentication, credentials, account selection/rotation/quarantine, quota,
token/window limits, 401/403/429/5xx circuits, retry, provider/NVIDIA fallback — all OmniRoute-owned
(`8.5–8.7`, external). Prodex Phase-3/HOLD. No Kimi (BLK-KIMI). No live acceptance initiated by this lane.

## 4. Blocker record
- blocker: `cline_glm live-run token = false (control.json live_run.authorized=false); BLK-AVAIL (enriched OmniRoute /v1/models rows for cp/cline-pass/glm-5.2 not published) + BLK-25B (live-provider runs security-stopped until exposed key revoked)`
- owner: `OmniRoute architect / Product security + Principal Orchestrator`
- next action: `when the single Codex56#A GLM run is authorized and executed, consume THAT SAME sanitized run evidence and map the 8.2 axes per §2 (tools/reasoning/cancel/usage/deterministic-error/terminal); do not initiate, rerun, or reserve the run`
- also: BLK-KIMI holds the Kimi (5.7) equivalent; not in this lane until its RouteModel is frozen and its own single run is authorized.

Registered via `p0_control.py block`. No source, no test, no live action while BLOCKED.
