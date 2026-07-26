# T20-live-correlation-audit — canonical correlation across the 7 hops (READ-ONLY)

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-LIVE-CORRELATION-AUDIT` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/T20-live-correlation-audit.md` · MODE: **READ-ONLY** (no source edits, no deploy/inference/secret).
- paths below are under `multica-auth-work/server/`.

> VERDICT: With every sink wired, `e2e.Assemble` would still **NOT** reach `AllContinuous`. Task-anchored
> hops (queue, admission, persist) join by `task_id`, but the four via-joined/absent hops break:
> **ingress never stamps `task_id`** (absent), **CLI `launch_id` ≠ admission `launch_id`** (different
> derivation), **route `request_id` is a different namespace and has no production emitter**, and
> **delivery `session_id` is the WS transport id, not the agent session**. The passing wiring tests only
> succeed because they **hand-align** the IDs (fixture alignment), which production does not.

## 1. Assemble join contract (source: `observability/e2e/assemble.go`)

Direct (anchored) by `task_id`: ingress, queue, admission, persist. Via-joined: **cli → `launch_id`
→ (admission) → task**; **route → `request_id` → (ingress) → task**; **delivery → `session_id` →
(admission) → task**. Continuous requires all 7 present for the task; unresolved via-keys → orphans.

## 2. Per-hop live values (production source)

| Hop | Emit call site (file:line) | Join key(s) | Production value | Sink today |
|---|---|---|---|---|
| ingress | `middleware/request_logger.go:143` `emitIngressSpan`; fail-closed on empty task_id `:296–303` | task_id (+request_id) | request_id = `chimw.GetReqID` (`:139`); **task_id = `ingressTaskIDFromContext` — no production caller of `SetIngressTaskID` (`:236`)** → empty → span never emitted | `ingressRecorder` nil (`:217`, no prod `SetIngressRecorder`) |
| queue | `service/task.go:186` `emitQueueEnqueued` / `:199` dequeued | task_id | `queue_msg_id = task_id = util.UUIDToString(task.ID)` (row id) `:190–194` | `TaskService.Obs` (`task.go:52`) nil in prod |
| admission | `daemon.go:4003` `agentBrainOBS.Emit`; span built `brain/admission_observability.go:60–66` | task_id (+session_id, launch_id) | corr = `plan.Task.Request.Correlation` (`daemon.go:3998–4000`, plan!=nil). task_id=`corr.TaskID`; session_id=`corr.SessionID`; **launch_id=`AdmissionLaunchID` = `admissionSafeID("launch", TaskID+":"+SessionID+":"+RequestID)`** (`admission_observability.go` AdmissionLaunchID + `:128` sha256[:8]) | `daemon.go:283` `daemonAdmissionSink(logger)` — logs; JSONL export **only if `AGENT_BRAIN_E2E_EXPORT_FILE` set** |
| cli | `daemon.go:3778` `EmitCLI(d.cliObs, …)` | launch_id → task | corr = `plan.Task.Request.Correlation` (`:3756`). task_id=`corr.TaskID`; **launch_id=`safeCorrelationID("launch", corr.RequestID)`** (`:3779`; fn `brain_integration.go:531` sha256[:8] of **RequestID only**) | `d.cliObs` nil (`:3767`) |
| route (OTLP) | `gateway/obs_span.go:37–40` `EmitProviderSpan` | request_id → task | span.request_id = `record.RequestID`; omni_request_id = `Telemetry.RequestID`. **No production caller of `EmitProviderSpan`** (only tests) | recorder nil (no prod caller) |
| persist | `service/task.go:227` `emitPersistSpan` (called `:1350`) | task_id | task_id = `util.UUIDToString(task.ID)`; result_id = `"result-"+id` (`:234–240`) | `TaskService.Obs` nil in prod |
| delivery | `daemonws/hub.go:415/449` `h.emitDelivery(sessionID,…)`; span `:259–266` | session_id → task | **session_id = the WS connection id `nextWSSessionID()` = `"wssess-N"`** (`:111`, `:304`) — NOT the agent/chat session | `delivRecorder` nil (no prod `SetDeliveryRecorder`) |

## 3. Mismatches proven (file:line)

- **M1 — ingress absent (task_id never stamped).** `emitIngressSpan` fail-closes when
  `ingressTaskIDFromContext` is empty (`request_logger.go:296–303`); `SetIngressTaskID`
  (`request_logger.go:236`) has **no production caller** (grep: only defined; call sites are tests).
  ⇒ ingress span never emitted ⇒ `requestToTask` resolver empty ⇒ **route can never join** even if wired.
- **M2 — launch_id mismatch (CLI ≠ admission).** admission: `admissionSafeID("launch",
  TaskID+":"+SessionID+":"+RequestID)` (`admission_observability.go` AdmissionLaunchID, hash `:128`).
  CLI: `safeCorrelationID("launch", corr.RequestID)` (`daemon.go:3779`, hash `brain_integration.go:531`).
  Same `corr`, but **different hashed input** (full triple vs `RequestID` only) ⇒ different `launch_id`
  ⇒ `Assemble.assignVia(HopCLI, launchToTask, LaunchID)` → **`launch_id_unresolved` orphan**.
- **M3 — route request_id namespace + no emitter.** `Assemble` joins route by `request_id`, expecting the
  **ingress** request_id. But route carries `record.RequestID` while the OmniRoute id goes to
  `omni_request_id` (`obs_span.go:38–39`); the ingress request_id (`chimw.GetReqID`) is a **different
  namespace** and is **not propagated** to the gateway, and there is **no production `EmitProviderSpan`
  call**. ⇒ route absent/orphan.
- **M4 — delivery session_id is the WS transport id.** delivery emits `nextWSSessionID()` = `"wssess-N"`
  (`hub.go:111`, `:304`, `:415`), not `corr.SessionID`. admission session_id = `corr.SessionID`
  (agent/chat session). ⇒ `Assemble.assignVia(HopDelivery, sessionToTask, SessionID)` →
  **`session_id_unresolved` orphan**.
- **M5 — queue hop double-emits per task (`duplicate_task_hop`).** The queue hop is emitted TWICE in one
  task lifecycle: `emitQueueEnqueued` at enqueue (`task.go:186`, outcome `enqueued`) AND
  `emitQueueDequeued` at claim/dequeue (`task.go:161→199`, outcome `dequeued`) — both `HopQueue` with the
  **same `task_id`**. `Assemble.place()` sees a second `HopQueue` for the task and records an
  **`AnomalyDuplicateSpan` / `duplicate_task_hop`** (`assemble.go` place()), which alone sets
  `AllContinuous=false`. The queue hop must emit **exactly one completed lifecycle span per task**.
- **task_id (the one that can join):** queue/persist use `util.UUIDToString(task.ID)`; admission/cli use
  `corr.TaskID`. These match **iff** `corr.TaskID == util.UUIDToString(task.ID)` (plan path). Verify;
  they appear equal, so {queue, admission, persist} would anchor to one task. **Caveat:** on the non-plan
  admission path (`daemon.go:3998` `AdmissionCorrelation` hashes task_id via `admissionSafeID("task",…)`)
  the admission task_id becomes `task-<hash>` and would **not** match the raw row id — a second task_id
  mismatch on the fail/no-plan path.

Net (even fully wired): joinable = {queue, admission, persist} by task_id (3/7); orphaned/absent =
ingress (M1), route (M1/M3), cli (M2), delivery (M4) ⇒ never `AllContinuous`.

## 4. Fixture alignment vs production values (the crux)

The passing tests **manufacture matching join keys** — they do not reflect production derivation:
- `e2ewiring/wiring_integration_test.go:73–115` sets `task/req/sess` to identical literals and, at
  `:88`, **reads admission's `launch_id` back** and feeds the SAME value into `EmitCLI` — so CLI's
  launch_id equals admission's by construction. Production instead **recomputes** CLI launch_id from a
  different input (M2), so they diverge.
- The same tests pass `RequestID: reqID` into `EmitProviderSpan` (route) equal to the ingress request_id,
  and `SessionID: sess` into delivery equal to admission's — both **hand-aligned**. Production route has
  no emitter/propagation (M3) and delivery uses `wssess-N` (M4).
⇒ Green wiring/anchor tests are **fixture alignment**, not proof of production correlation. This audit
distinguishes them: the emit *helpers* are correct; the **production value wiring** is not.

## 5. Minimal canonical propagation contract (specification only — source edits are out of this lane)

Adopt one canonical value per join key and propagate it via the frozen carrier
(`contract.go` `X-AB-*` headers); each item maps to the join it repairs:
1. **task_id = `util.UUIDToString(task.ID)` (row id) everywhere.** Guarantee `corr.TaskID == row id`
   (plan + non-plan admission paths); stamp ingress via `SetIngressTaskID(row id)` at the task-creating
   handler. Fixes M1 (+ anchors ingress/queue/admission/persist).
2. **launch_id: single derivation.** Replace `daemon.go:3779` `safeCorrelationID("launch",
   corr.RequestID)` with `brain.AdmissionLaunchID(corr)` so CLI and admission compute the identical key
   from the identical `corr` triple. Fixes M2.
3. **request_id: one canonical id, propagated ingress→gateway.** Carry the ingress request_id (or a
   single generated request_id) in `corr.RequestID` and into `EmitProviderSpan`'s `record.RequestID`
   (add the production route emit at the gateway terminal outcome). Fixes M3.
4. **session_id = agent/chat session (`corr.SessionID`) at delivery.** Map the WS connection → chat
   session and pass that as `emitDelivery`'s sessionID instead of `nextWSSessionID()`. Fixes M4.
5. **One shared exporter across processes.** Install a shared JSONLSink (+BoundedSink) recorder on all
   hops (ingress/queue/cli/route/persist/delivery) and route admission through the same export
   (`daemonAdmissionSink` already supports `AGENT_BRAIN_E2E_EXPORT_FILE`); the collector
   (`e2e.AssembleFromLogs`) then merges a consistent stream. (Wiring; P0-gated per D-V3-21.)

## 6. Non-claims / limitations
- Read-only audit; no source edited, no deploy/inference/secret. All findings cite file:line.
- `corr.TaskID == util.UUIDToString(task.ID)` is asserted as "verify; appears equal" — confirm at plan
  construction; the non-plan admission hashing path is a separate task_id mismatch (§3 caveat).
- Sink-wiring gaps (nil recorders) were separately evidenced in `T20-obs-wiring-inventory.md`; this audit
  focuses on VALUE correlation assuming sinks were wired.
- The canonical contract (§5) requires shared-source edits (W1-serial anchors) and is Priority-2/DEFERRED
  behind P0; owner: W1 + Principal (`w5:p9`). Not applied here.
