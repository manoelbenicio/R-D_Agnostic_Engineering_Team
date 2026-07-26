# Kiro Supervisor — Correlation & Terminal/UI Evidence Collection Plan (Clean Live Task)

**Lane**: `KIRO-TRACE-UI-AUDIT` (`wK:p2`)  
**Agent**: `Agy-KiroTrace`  
**Task ID**: `KIRO-TRACE-UI-AUDIT`  
**Timestamp**: 2026-07-23T00:09:50Z  
**Repository HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`  
**Contract Version**: `agent-brain.e2e.v1`  
**Posture**: Read-only audit & evidence collection plan for clean live task execution. Consumes safe correlation IDs ONLY. Zero secrets, zero product edits, zero unauthorized inference.  

---

## 1. Executive Audit Purpose & Safety Constraints

This document defines the Kiro Supervisor evidence collection protocol for the upcoming single clean live acceptance run:
- **Safe Correlation Identifiers Only**: All trace logging, telemetry collection, and UI verification consume **only safe metadata IDs** (`request_id`, `queue_msg_id`, `task_id`, `session_id`, `launch_id`, `proc_id`, `omni_request_id`, `result_id`, `delivery_id`).
- **Secrets-Free Invariant**: `SecretsPresent == false` is enforced across all 8 hops and UI presentation layers. Prompts, raw stderr/stdout containing user content, bearer tokens, and credentials are **strictly excluded**.
- **No Product Edits**: Product source code in `multica-auth-work/server/...` and `packages/...` remains **100% read-only**.

---

## 2. Safe Identifier Mapping Matrix

| Safe Identifier Key | Field Name | Emitting Hop Layer | Safe Value Charset / Bounds | UI & Telemetry Role |
|---|---|---|---|---|
| 1. `request_id` | `RequestID` | Hop 1 (`ingress`) | `[A-Za-z0-9-._:]` (max 128) | Correlates HTTP trigger request |
| 2. `queue_msg_id` | `QueueMsgID` | Hop 2 (`queue`) | `[A-Za-z0-9-._:]` (max 128) | Correlates DB queue message |
| 3. `task_id` | `TaskID` | Hops 1, 2, 3, 6 | UUID format | Primary key joining task details & UI logs |
| 4. `session_id` | `SessionID` | Hops 3, 7 | `[A-Za-z0-9-._:]` (max 128) | WS session ID for UI client subscription |
| 5. `launch_id` | `LaunchID` | Hops 3, 4 | `[A-Za-z0-9-._:]` (max 128) | Process launch correlation |
| 6. `proc_id` | `ProcID` | Hop 4 (`cli`) | `[A-Za-z0-9-._:]` (max 128) | Subprocess execution ID |
| 7. `omni_request_id` | `OmniRequestID` | Hop 5 (`route`) | `[A-Za-z0-9-._:]` (max 128) | OmniRoute gateway request ID |
| 8. `result_id` | `ResultID` | Hop 6 (`persist`) | `[A-Za-z0-9-._:]` (max 128) | Terminal result ID rendered in UI |
| 9. `delivery_id` | `DeliveryID` | Hop 7 (`delivery`) | `[A-Za-z0-9-._:]` (max 128) | WS delivery tracking token |

---

## 3. Eight-Hop Trace Sequence Audit (`agent-brain.e2e.v1`)

```
[UI Trigger / API Call]
          │
          ▼
   1. HopIngress (W6)    ───►  Carries [request_id, task_id]
          │
          ▼
   2. HopQueue (W7)      ───►  Carries [queue_msg_id, task_id]
          │
          ▼
   3. HopAdmission (W1)  ───►  Carries [task_id, session_id, launch_id]
          │
          ▼
   4. HopCLI (W3)        ───►  Carries [launch_id, proc_id] (Shape tokens only)
          │
          ▼
   5. HopRoute (W2)      ───►  Carries [request_id, omni_request_id] (Sanitized Telemetry)
          │
          ▼
   6. HopPersist (W7)    ───►  Carries [task_id, result_id]
          │
          ▼
   7. HopDelivery (W6)   ───►  Carries [session_id, delivery_id]
          │
          ▼
   8. HopTrace (W5)      ───►  Synthesizes 8-hop continuous trace (AllContinuous = true)
```

---

## 4. UI Terminal & Status Presentation Audit

1. **Kanban Card Status Transition (`board-view.tsx`)**:
   - Card reflects real-time status updates received via WebSocket: `queued` -> `running` -> `completed` (or `failed`).
2. **Real-time Execution Log Stream (`execution-log-section.tsx`)**:
   - Live execution logs stream via WebSocket and `GET /api/tasks/{taskId}/logs`.
   - Log lines contain process status messages; sensitive environment variables and credentials are scrubbed by `safeAgentArgvForLog`.
3. **Persisted Terminal Result Display (`issue-detail.tsx`)**:
   - Displays final task result (`ResultID`), execution duration, and exit status code.
4. **WebSocket Delivery Metadata (`obs_delivery.go`)**:
   - Emits Hop 7 metadata span with safe `session_id` and `delivery_id`. Zero payload body text transported in span.

---

## 5. Protocol for Clean Live Acceptance Task Execution

When the Principal Orchestrator authorizes the single live run (`control.json.live_runs.*.authorized = true`):
1. **Pre-Launch**: Record target `task_id` and reserved `run_id` from `control.json`.
2. **Execution Monitoring**: Observe dispatches across Hops 1 through 7 without logging prompt text or secret headers.
3. **UI Verification**:
   - Verify Kanban card moves from `queued` to `running` to `completed`.
   - Verify terminal log section displays clean execution output.
   - Verify `issue-detail.tsx` renders the terminal result.
4. **Trace Synthesis**: Run `e2e.Assemble` over recorded spans to confirm `AllContinuous == true` across all 8 hops.

---

## 6. Non-Claims & Compliance Summary

- **Product Source Code**: Zero product source code files were edited by this audit (`read-only` posture).
- **Secrets & Credentials**: Zero secret values, bearer tokens, or API keys were read, logged, or printed (`SecretsPresent == false`).
- **Inference & Execution**: Zero model inference requests were issued (`live_runs.*.authorized = false`).
- **Deploy & Infrastructure**: No deploy, container restart, Docker, or systemd commands were executed.
