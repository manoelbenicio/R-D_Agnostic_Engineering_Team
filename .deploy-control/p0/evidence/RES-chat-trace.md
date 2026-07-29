# Post-Deploy Chat Smokes & 8-Hop/9-ID Trace Standby Protocol

**Lane**: `chat-smokes-trace` (`wK:p2`)  
**Agent**: `Agy-ChatTrace`  
**Task ID**: `CHAT-SMOKES-TRACE`  
**Timestamp**: 2026-07-23T02:14:15Z  
**Repository HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`  
**Contract Version**: `agent-brain.e2e.v1`  
**Posture**: Standby evidence protocol for post-deploy chat smokes (untargeted -> Kiro-TL, direct -> Codex escape hatch), 8-hop / 9-ID trace correlation, and UI status presentation. Zero secrets, zero product edits, zero unauthorized inference (`live_runs = false`).  

---

## 1. Standby Posture & Precondition Gates

This protocol governs post-deploy chat smokes and telemetry correlation once the execution environment is fully provisioned:

| Precondition Gate | Status | Impact / Requirement |
|---|---|---|
| **1. Deployed Live Database** | **Gated / Standby** | Requires live PostgreSQL (`127.0.0.1:5432`) to be reachable for handler execution. |
| **2. OmniRoute Live Run Token** | **Gated / Standby** | Requires `control.json` `live_runs.*.authorized = true` issued by Principal (`w5:p9`). |
| **3. D-V3-25(B) Security Stop** | **Gated / Active** | Requires confirmation of key revocation before live provider execution. |

---

## 2. Post-Deploy Chat Smokes Verification Protocol

### 2.1 Smoke 1: Untargeted Chat -> Kiro-TL (`req.AgentID == ""`)
- **Action**: Post `POST /api/chat-sessions` with empty `agent_id` (`{ "title": "Untargeted Chat" }`).
- **Backend Flow (`chat.go`)**:
  1. `CreateChatSession` checks `req.AgentID == ""`.
  2. Invokes `Queries.ListSquads` for the workspace.
  3. Resolves default squad leader `squads[0].LeaderID` (Kiro-TL).
  4. Sets session `agentID = squads[0].LeaderID`.
- **Expected Result**: Response HTTP `201 Created`, `AgentID == squadTLID`.

### 2.2 Smoke 2: Direct `@agent` -> Codex Escape Hatch (`req.AgentID != ""`)
- **Action**: Post `POST /api/chat-sessions` with explicit `agent_id` (`{ "agent_id": "<codex_agent_id>", "title": "Direct Codex Chat" }`).
- **Backend Flow (`chat.go`)**:
  1. `CreateChatSession` parses explicit `agent_id`.
  2. Validates agent workspace scope via `GetAgentInWorkspace`.
  3. Evaluates private agent access gate via `canAccessPrivateAgent`.
  4. Routes session directly to specified Codex agent.
- **Expected Result**: Response HTTP `201 Created`, `AgentID == codexAgentID`.

---

## 3. Eight-Hop Trace & Nine Safe Identifiers Correlation (`agent-brain.e2e.v1`)

### 3.1 Nine Safe Correlation Identifiers

All 9 identifiers are strictly metadata-only, matching `e2e.safeID` (`[A-Za-z0-9-._:]`). No credentials, bearer tokens, prompts, or account identities exist within correlation payloads:

| # | Safe Identifier Key | Field | Emitting Hop Layer | UI & Telemetry Role |
|---|---|---|---|---|
| 1 | `request_id` | `RequestID` | Hop 1 (`ingress`) | Correlates initial HTTP chat / task request |
| 2 | `queue_msg_id` | `QueueMsgID` | Hop 2 (`queue`) | Correlates DB queue message |
| 3 | `task_id` | `TaskID` | Hops 1, 2, 3, 6 | Key join ID for issue task details & logs |
| 4 | `session_id` | `SessionID` | Hops 3, 7 | WS session ID for UI client subscription |
| 5 | `launch_id` | `LaunchID` | Hops 3, 4 | Process launch identifier |
| 6 | `proc_id` | `ProcID` | Hop 4 (`cli`) | Subprocess execution ID |
| 7 | `omni_request_id` | `OmniRequestID` | Hop 5 (`route`) | OmniRoute gateway request ID |
| 8 | `result_id` | `ResultID` | Hop 6 (`persist`) | Terminal result ID rendered in UI |
| 9 | `delivery_id` | `DeliveryID` | Hop 7 (`delivery`) | WS delivery tracking token |

### 3.2 End-to-End Hop Sequence

```
[Chat Trigger / User Action]
            │
            ▼
     1. HopIngress (W6)    ───► [request_id, task_id]
            │
            ▼
     2. HopQueue (W7)      ───► [queue_msg_id, task_id]
            │
            ▼
     3. HopAdmission (W1)  ───► [task_id, session_id, launch_id]
            │
            ▼
     4. HopCLI (W3)        ───► [launch_id, proc_id]
            │
            ▼
     5. HopRoute (W2)      ───► [request_id, omni_request_id]
            │
            ▼
     6. HopPersist (W7)    ───► [task_id, result_id]
            │
            ▼
     7. HopDelivery (W6)   ───► [session_id, delivery_id]
            │
            ▼
     8. HopTrace (W5)      ───► Synthesizes 8-hop continuous trace (AllContinuous = true)
```

---

## 4. UI Status Presentation Validation Protocol

1. **Kanban Card Status Transition (`board-view.tsx`)**:
   - Card reflects real-time status updates received via WebSocket: `queued` -> `running` -> `completed` / `failed`.
2. **Real-time Execution Log Stream (`execution-log-section.tsx`)**:
   - Live execution logs stream via WebSocket and `GET /api/tasks/{taskId}/logs`.
   - Log lines contain process status messages; sensitive environment variables and credentials are scrubbed by `safeAgentArgvForLog`.
3. **Persisted Terminal Result Display (`issue-detail.tsx`)**:
   - Displays final task result (`ResultID`), execution duration, and exit status code.
4. **WebSocket Delivery Metadata (`obs_delivery.go`)**:
   - Emits Hop 7 metadata span with safe `session_id` and `delivery_id`. Zero payload body text transported in span.

---

## 5. Non-Claims & Compliance Summary

- **Product Source Code**: Zero product source code files were edited by this audit (`read-only` posture).
- **Secrets & Credentials**: Zero secret values, bearer tokens, or API keys were read, logged, or printed (`SecretsPresent == false`).
- **Inference & Execution**: Zero model inference requests were issued (`live_runs.*.authorized = false`).
- **Deploy & Infrastructure**: No deploy, container restart, Docker, or systemd commands were executed.
