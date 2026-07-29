# Canonical-correlation join diagnosis (source-verified) 2026-07-24T00:07Z
## What joins vs breaks
- task_id/session_id: JOIN OK. safeCorrelationID(prefix,v)="prefix-"+hex(sha256(v)[:8]) == admissionSafeID — identical algorithm/input, so admission & daemon newCorrelation produce equal task_id/session_id.
- request_id: BREAKS. daemon newCorrelation regenerates RequestID="request-"+digest(task.ID)+"-"+seq (independent, sequence-based) instead of propagating the backend INGRESS request_id. Route assembler expects ingress request_id -> route hop cannot join ingress.
- launch_id: BREAKS. AdmissionLaunchID = admissionSafeID("launch", TaskID+":"+SessionID+":"+RequestID). Admission builds it with RequestID="" (AdmissionCorrelation sets no RequestID) while CLI/daemon path has a populated RequestID -> launch_id recomputed to DIFFERENT values across hops.
## Confirmed Principal blockers
1. Recorders unwired at startup (d.cliObs, middleware.SetIngressRecorder, TaskService.Obs, Hub.SetDeliveryRecorder, route/OTLP) = no-op; only brain admission recorder assigned.
2. No canonical carrier: request_id not propagated ingress->daemon; launch_id derived from inconsistent RequestID; queue/persist use raw task UUID.
3. daemonws delivery emits every frame w/ random connection session_id (not the one terminal-result delivery; can't resolve to admission session_id) -> dup/orphan.
4. CLI proc_id synthetic (safeCorrelationID) not real PID; launch_id synthetic not AdmissionLaunchID.
## Required (fleet + Kiro serial): ONE canonical deterministic correlation carrier propagated cross-process (ingress request_id -> queue -> daemon -> route/CLI; admission-consistent launch_id); real startup wiring of all 6 hop recorders; real child PID via agent.Session.ProcessID; actual terminal-result delivery anchor w/ admission session carrier; correct OTLP shape (event.name=api_request, resource attrs, fail-close, dedup). Acceptance: exactly one of each hop, AllContinuous, real IDs, no synthetic fixtures.

## Additional source-proven blockers (2026-07-24T00:09Z)
(A) INGRESS unemittable: SetIngressTaskID has ZERO production callers (tests only) -> RequestLogger cannot emit a valid ingress span even with a recorder installed. FIX: wire a real accepted control endpoint that knows the task_id and sets the canonical task/request carrier (SetIngressTaskID at the actual accepted send/dispatch handler). 
(B) QUEUE double-emit: TaskService emits HopQueue at BOTH enqueue (notifyTaskAvailable->emitQueueEnqueued) AND dispatch (captureTaskDispatched->emitQueueDequeued). Assemble permits exactly ONE span/task/hop -> 2nd = duplicate_task_hop. FIX: emit ONE completed queue lifecycle span for accepted tasks (prefer dequeue carrying enqueue+dequeue counters) OR a formally-approved contract/assembler revision — not two. Add exact per-hop cardinality tests at real call sites.
