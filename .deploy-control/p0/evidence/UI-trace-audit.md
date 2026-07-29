# UI Terminal/Status Validation & 8-Hop/9-ID Trace Audit

**Lane**: `UI/trace-audit` (`wK:p2`)  
**Agent**: `Agy-UI-Trace`  
**Task ID**: `UI-TRACE-AUDIT`  
**Timestamp**: 2026-07-23T00:09:00Z  
**Repository HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`  
**Contract Version**: `agent-brain.e2e.v1`  
**Posture**: Read-only audit & OpenSpec traceability for UI terminal delivery, status presentation, and 8-hop / 9-ID trace correlation. Zero product edits, zero secrets, zero live inference.  

---

## 1. Executive Summary & Audit Purpose

This audit establishes the frontend UI validation plan and 8-hop/9-ID trace correlation traceability for the **Main Brain / OmniRoute** single live acceptance run:
1. **UI Terminal & Status Validation**: Verifies the UI components responsible for rendering Kanban task status transitions (`queued` -> `running` -> `completed`/`failed`), streaming execution logs via WebSockets, and presenting terminal results.
2. **8-Hop / 9-ID Trace Correlation**: Verifies end-to-end propagation from HTTP ingress (`HopIngress`) to WebSocket UI delivery (`HopDelivery`) and trace assembly (`HopTrace`), enforcing the `SecretsPresent == false` invariant.
3. **OpenSpec Traceability**: Maps OpenSpec tasks (specifically `build-omniroute-agent-brain` 4.1, 6.2, 6.5 and `chat-orchestration-standard` 1.3, 2.3) to concrete evidence files.

---

## 2. UI Terminal & Status Component Architecture

| Component File | UI Purpose & Responsibilities | Backend Data Source | Telemetry Hop / Correlation |
|---|---|---|---|
| `packages/views/issues/components/board-view.tsx` | Renders Kanban board columns and task status cards. Reflects real-time state changes. | `GET /api/issues` & WS events | Hop 7 (`delivery`) via WS updates |
| `packages/views/issues/components/execution-log-section.tsx` | Streams live process execution logs (stdout/stderr) and displays completion state. | WS log topic & `GET /api/tasks/{taskId}/logs` | Hop 4 (`cli`) & Hop 6 (`persist`) |
| `packages/views/issues/components/issue-detail.tsx` | Displays full issue metadata, assigned agent, task lifecycle state, and persisted result. | `GET /api/issues/{id}` & `GET /api/tasks/{id}` | Hop 6 (`persist`) & Hop 7 (`delivery`) |
| `apps/mobile/app/(app)/[workspace]/issue/[id]/runs.tsx` | Mobile agent run sheet displaying active and past execution runs. | `GET /api/issues/{id}/tasks` | Hop 6 (`persist`) & Hop 7 (`delivery`) |
| `server/internal/daemonws/obs_delivery.go` | Emits Hop 7 (`delivery`) span upon WebSocket message dispatch (delivers status to UI). | `DeliveryRecorder.EmitDelivery` | Hop 7 (`delivery_id`, `session_id`) |

---

## 3. Eight-Hop Trace & Nine Safe Identifiers Audit (`agent-brain.e2e.v1`)

### 3.1 Nine Safe Correlation Identifiers

All 9 identifiers are strictly metadata-only, matching `e2e.safeID` (`[A-Za-z0-9-._:]`). No credentials, bearer tokens, prompts, or account identities exist within correlation payloads:

| # | Safe Identifier Key | Field | Emitting Hop | UI Relevance |
|---|---|---|---|---|
| 1 | `request_id` | `RequestID` | Hop 1 (`ingress`) | Correlates initial HTTP trigger request |
| 2 | `queue_msg_id` | `QueueMsgID` | Hop 2 (`queue`) | Correlates DB queue message |
| 3 | `task_id` | `TaskID` | Hops 1, 2, 3, 6 | Key join ID for issue task details & logs |
| 4 | `session_id` | `SessionID` | Hops 3, 7 | WS session ID for UI client connection |
| 5 | `launch_id` | `LaunchID` | Hops 3, 4 | Process launch identifier |
| 6 | `proc_id` | `ProcID` | Hop 4 (`cli`) | Process execution ID |
| 7 | `omni_request_id` | `OmniRequestID` | Hop 5 (`route`) | OmniRoute gateway request ID |
| 8 | `result_id` | `ResultID` | Hop 6 (`persist`) | Persisted terminal result ID rendered in UI |
| 9 | `delivery_id` | `DeliveryID` | Hop 7 (`delivery`) | WS delivery payload tracking ID |

### 3.2 End-to-End Hop Flow & Fail-Closed Behavior

```
[User Action in UI]
       │
       ▼
1. HopIngress (W6) ─────► 2. HopQueue (W7) ─────► 3. HopAdmission (W1)
 (POST /api/issues/rerun)   (DB Task Enqueue)       (Daemon Admission Check)
                                                           │
                                            ┌──────────────┴──────────────┐
                                     (Admission OK)               (Admission Failed)
                                            │                             │
                                            ▼                             ▼
                                    4. HopCLI (W3)                HopPersist (W7)
                                   (Process Launch)           (Persist Fail-Closed)
                                            │                             │
                                            ▼                             │
                                    5. HopRoute (W2)                      │
                                  (OmniRoute Request)                     │
                                            │                             │
                                            └──────────────┬──────────────┘
                                                           │
                                                           ▼
                                                    6. HopPersist (W7)
                                                  (Terminal Result Saved)
                                                           │
                                                           ▼
                                                    7. HopDelivery (W6)
                                                  (WS Status Broadcast)
                                                           │
                                                           ▼
                                                    [UI Update Rendered]
```

---

## 4. OpenSpec & Requirement Traceability Matrix

| OpenSpec Specification | Task ID | Description | Status | Evidence Artifact |
|---|---|---|---|---|
| `build-omniroute-agent-brain` | **4.1** | Preserve backend/web, projects, squads, Kanban issues/tasks, Postgres, terminal/session persistence | **CLOSED `[x]`** | `A7-production-integrity-frontend.md`, `A8-terminal-backend-evidence.md` |
| `build-omniroute-agent-brain` | **6.2** | Metadata-only correlation across 7 hops (`ingress` to `delivery`) + `HopTrace` | **CLOSED `[x]`** | `F4-hub-delivery-anchor.md`, `F8-post-t360-verification-report.md`, `TRACE-live.md` |
| `build-omniroute-agent-brain` | **6.5** | Single live acceptance run Kanban -> Main Brain -> OmniRoute -> Terminal UI | **OPEN `[ ]`** | `R1-glm-live-acceptance.md`, `R2-opus48-live-acceptance.md` (Gated on Principal live-run token) |
| `chat-orchestration-standard` | **1.3** | Untargeted chat -> Squad TL; `@agent` -> Direct escape hatch | **CLOSED `[x]`** | `CHECKOUT__Agy-C1__C1__C1-CHAT-BACKEND__20260722T122410Z.json` (`chat.go`) |
| `chat-orchestration-standard` | **2.3** | Check-ins DONE + evidence in `.deploy-control/` | **CLOSED `[x]`** | `.deploy-control/p0/checkins/` receipts |

---

## 5. Live Acceptance Run UI Validation Checklist

When the Principal Orchestrator authorizes the single live acceptance run (`live_runs.*.authorized = true`), the following UI terminal and status verifications will be performed:

1. **Kanban Card Status Transition**: Observe real-time status update from `queued` -> `running` -> `completed` (or `failed`) in `board-view.tsx`.
2. **Real-time Log Streaming**: Confirm `execution-log-section.tsx` receives and renders log output lines without truncation or buffer corruption.
3. **Terminal Result Persistence**: Verify `issue-detail.tsx` displays the terminal execution result (`ResultID`), matching the backend record in `task_result`.
4. **Fail-Closed Error Surface**: If execution fails closed (e.g. missing credentials), verify the UI displays a clean, non-secret error message without exposing environment or stack trace values.
5. **Zero Secret Leak Verification**: Scan browser network traffic and component state for any raw bearer token or credential string.

---

## 6. Non-Claims & Compliance Summary

- **Product Source Code**: Zero product source code files were edited by this audit (`read-only` posture).
- **Secrets & Credentials**: Zero secret values, bearer tokens, or API keys were read, logged, or printed.
- **Inference & Execution**: Zero model inference requests were issued (`live_runs.*.authorized = false`).
- **Deploy & Infrastructure**: No deploy, container restart, Docker, or systemd commands were executed.
