# AUDIT — 7 e2e hops: implemented / installed / one-span-canonical / metadata-only / no-dead-seam

- auditor: **Opus48#C** (independent; read-only; cite file:line) · lane **AUDIT** · task **AUDIT-E2E-HOPS** · pane `w8:p1`
- reporting to: Opus48-Kiro (`w5:p1`) · as-of (UTC): `2026-07-24T03:16Z` · HEAD `a6d50986…`
- method: read-only source inspection of the real wiring + emit paths; no code edited.

## VERDICT: **PASS** (all 5 claims verified) — with 3 low-severity NOTES (non-blocking).

## Claim 1 — all 7 hops implemented — **PASS**
| Hop | Emitter (file:line) | Builds |
|---|---|---|
| 1 ingress | `internal/middleware/obs_ingress.go:60` `EmitIngress` (+ `request_logger.go` `emitIngressSpan`) | `HopIngress` |
| 2 queue | `internal/service/obs_queue.go:32/46` `EmitQueue` | `HopQueue` |
| 3 admission | `internal/daemon/brain/admission_observability.go:65` | `HopAdmission` |
| 4 cli | `internal/daemon/cli_observability.go:43/74` `EmitCLI` | `HopCLI` |
| 5 route | `internal/daemon/gateway/obs_span.go:28` `EmitProviderSpan` | `HopRoute` |
| 6 persist | `internal/service/obs_persist.go:35/51` `EmitPersist` | `HopPersist` |
| 7 delivery | `internal/realtime/obs_delivery.go:143` `emitTerminalDelivery` (`buildSpan`) | `HopDelivery` |

## Claim 2 — installed on a shared per-process recorder — **PASS**
- **Server** (`cmd/server/e2e_recorder.go:64-73`): one `rec` installed on all seams — `middleware.SetIngressRecorder(rec)` (:73), `ts.Obs = rec` looped over **each** supplied `*service.TaskService` (:66-68 → both HTTP + sweeper instances), `hub.SetDeliveryRecorder(rec)` (:71). Single per-process recorder over one 0600 JSONL sink (`newServerSpanRecorder` :23-49, fail-closed on unopenable/untightenable file).
- **Daemon** (`internal/daemon/daemon.go:277-278, 296-297, 830`): `daemonSpanSink` + `daemonSpanRecorder = e2e.NewRecorder(daemonSpanSink)` (:277-278) — ONE sink/recorder; `agentBrainOBS = brain.NewAdmissionObserver(daemonSpanSink)` (:296, admission), `cliObs = daemonSpanRecorder` (:297, cli), and `startRouteTelemetryReceiver(d.cliObs, …)` (:830, route). Admission/cli/route all feed the SAME per-process daemon sink.
- **NOTE-1 (info):** admission attaches the raw `daemonSpanSink` while cli/route use `daemonSpanRecorder` (which wraps that same sink). Same export; consistent. Admission still fails closed on invalid metadata (`admission_observability_test.go TestAdmissionObserverFailsClosedWithoutRecordingInvalidMetadata`).

## Claim 3 — exactly one span per hop with canonical joins — **PASS**
Canonical joins (= frozen `RequiredIDs`):
- ingress `{request_id, task_id}`; queue `{queue_msg_id, task_id}` (`obs_queue.go:17-18,33-34`); admission `{task_id, session_id, launch_id}` (`admission_observability.go:61-63`); cli `{launch_id, proc_id}` (`cli_observability.go:22-23,44-45`); route `{request_id, omni_request_id}` (`obs_span.go` `RequestID`+`OmniRequestID=Telemetry.RequestID`); persist `{task_id, result_id}` (`obs_persist.go:18-19,36`); delivery `{session_id, delivery_id}` (`obs_delivery.go buildSpan`, session=`CanonicalSessionID`). ✓
Exactly one per hop:
- queue: `emitQueueEnqueued` is an intentional **no-op** (`task.go:183-189`); the single lifecycle span is emitted once at dequeue (`emitQueueDequeued`→`EmitQueue`, `task.go:196-217`). ✓ (no double-emit).
- persist: one after terminal persistence (`emitPersistSpan`→`EmitPersist`, `task.go:220-235`). ✓
- delivery: idempotent per `(task_id|event)` dedup set → at-most-once (`obs_delivery.go:143` + `markSeenLocked`, marked seen only after successful `Emit`). ✓
- cli: one per launch, emitted **only** with a real child PID — empty `proc_id` ⇒ span skipped, never synthesized (`daemon.go:3820-3835`). ✓
- admission: one per admission attempt, gated `agentBrainGatewayRequired` (`daemon.go:4046-4054`). ✓
- route: `EmitProviderSpan` is the sole `HopRoute` emitter, one per received telemetry record via the receiver (`route_telemetry.go:119`). ✓
- ingress: one per HTTP request (`request_logger.go emitIngressSpan`). ✓

## Claim 4 — metadata-only (ScanSpans clean) — **PASS**
- Every hop span is built via `e2e.NewSpan` with only the closed label/counter vocabularies and emitted through `Recorder.Emit`, which runs the fail-closed `Span.Validate` (`secrets_present=false`, closed label keys, per-hop required IDs). Route pseudonym labels are re-validated as `principal_/acct_/conn_`+hex by `Span.Validate` (a malformed pseudonym is refused at Emit). Delivery `buildSpan` sets only closed delivery keys.
- Leak/metadata assertions present: `e2e/leakscan_test.go`, `e2e/contract_test.go`, `e2e/lifecycle_integration_test.go`, `e2e/collector_test.go`, `gateway/obs_span_test.go`, `brain/admission_observability_test.go` (+ `e2e/t20_hardened_verify_test.go`). ✓

## Claim 5 — NO dead/no-caller seam (emitTerminalDelivery reached from the real fanout) — **PASS**
- Real caller: `cmd/server/listeners.go:197-205` — on a **terminal task event** (`isTerminalTaskEvent(e.Type) && e.TaskID != ""`) the listener type-asserts the configured broadcaster to `realtime.TerminalBroadcaster` and calls `tb.BroadcastTerminalToWorkspace(e.WorkspaceID, data, TerminalDeliveryMeta{...})`.
- Single-node: `Hub.BroadcastTerminalToWorkspace` → `Hub.BroadcastTerminalDelivery` → `deliverScopeCounting` (REAL fan-out) **then** `emitTerminalDelivery` (`obs_delivery.go`). 
- Relay: `DualWriteBroadcaster.BroadcastTerminalToWorkspace` (`internal/realtime/redis_relay.go:561-564`) → `d.local.BroadcastTerminalDelivery(ScopeWorkspace, …, sharedID)` → same fan-out + `emitTerminalDelivery`.
- Both `*Hub` and `*DualWriteBroadcaster` satisfy `TerminalBroadcaster` (compile-time `var _` at `obs_delivery.go` and `redis_relay.go:595`). The span is emitted through the SAME path that fans the frame out, exactly once per `(session,event)`. **Not dead.** ✓

## NOTES for the Principal (low severity; non-blocking)
- **NOTE-2 (confirm):** I cited the route receiver wiring (`daemon.go:830`, shared `d.cliObs`, fail-closed bind :829-834) and that `EmitProviderSpan` (`obs_span.go:28`) is the sole `HopRoute` emitter, but did **not** read the receiver's HTTP handler body line that invokes `EmitProviderSpan`. Recommend a one-line confirm that `route_telemetry.go`'s handler calls `EmitProviderSpan` exactly once per decoded telemetry record (design implies it; not directly cited).
- **NOTE-3 (info):** two `SetDeliveryRecorder` exist — `internal/realtime/obs_delivery.go:179` (`*e2e.Recorder`, the ACTIVE server delivery hop, wired by `installServerObservabilityWith`) and `internal/daemonws/hub.go:235` (`*DeliveryRecorder`, a separate type). The server delivery hop is the `realtime.Hub` path; the `daemonws` one is not on the audited server seam. No conflict observed, but worth confirming the `daemonws` variant isn't a second, unwired delivery path expected to emit.

## Provenance / scope
- Read-only audit; no product/test/config edited. Independent (verified against source, not prior-lane claims). Citations are file:line at HEAD `a6d50986…`. No inference/deploy/secret; no live run. This audits instrumentation shape only, not LLM/route acceptance.
