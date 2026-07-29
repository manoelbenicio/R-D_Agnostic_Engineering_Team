# Tier-20 trace observability wiring — defect + plan + progress 2026-07-23T22:48Z
## Defect (confirmed)
e2e.NewRecorder(sink) is wired live ONLY in brain/admission_observability.go. All other hop owners use NewRecorder(nil)/discardSink no-op (e.g. daemonws/obs_delivery.go). => only admission (3/8) emits; 4/9 IDs present.
## Foundational sink (DONE, tested)
e2e.BoundedSink: fixed-capacity ring buffer, metadata-only (Emit validates + leak-scans before Record), concurrency-safe, Stats(total/retained/dropped); bounded memory, no busy loop. Tests PASS (capacity/eviction-order/clamp/concurrent).
## Per-hop injection plan (6 unwired) + REAL id sources (no synthesis)
- ingress (W6 backend HTTP handler): IDs request_id+task_id. Inject recorder at chat/issue send handler.
- queue (W7 backend enqueue): queue_msg_id (durable agent_task_queue row id) + task_id, at service/task enqueue.
- cli (W3 daemon launch): launch_id (AdmissionLaunchID) + proc_id (actual agent CLI subprocess PID at cmd.Start). Requires surfacing the spawned PID from the agent runner.
- route (W2 daemon): request_id + omni_request_id = sanitized gateway ProbeResult.RequestID (HeaderOmniRouteRequestID) — currently captured then DISCARDED in ReadinessChecker; must surface consume-only (no OmniRoute internals).
- persist (W7 backend): task_id + result_id (persisted result/message row id).
- delivery (W6 backend WS daemonws): session_id + delivery_id (WS delivery attempt id).
- Cross-process correlation: propagate via existing X-AB-* headers (contract.go) backend<->daemon.
## Status: foundational bounded sink complete+tested. Remaining: inject shared sink at 6 hop owners with real IDs + correlation propagation + instrumented Tier-20 rerun + independent 20x8hop/9ID verify. Large multi-layer cross-process change; in progress. 6.3 OPEN; tier50/100 BLOCKED. No synthesized IDs.
