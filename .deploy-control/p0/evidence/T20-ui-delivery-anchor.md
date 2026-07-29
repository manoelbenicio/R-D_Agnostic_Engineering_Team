# Real Terminal-Result Delivery Path to Web / Kanban UI — Audit & Delivery Anchor Evidence (2026-07-24)

## 1. Executive Summary
This audit traces the actual, end-to-end persisted terminal-result delivery path from task execution completion to the Web / Kanban UI. 

**Crucial Distinction**: Daemon `task-available` notification (`daemonws/hub.go`) is a internal control-plane wakeup for daemon task-claiming, **NOT** the user-facing UI terminal result delivery path. The actual Web / Kanban UI terminal result delivery path flows through the HTTP API completion handler -> `TaskService` DB persistence & span emission -> `events.Bus` pub/sub -> `realtime.Hub` workspace WebSocket broadcast -> Web client `useRealtimeSync` cache invalidation and UI re-render.

---

## 2. End-to-End Persisted Terminal-Result Delivery Flow

```
[Agent Execution / Daemon]
       │
       ▼  POST /api/daemon/tasks/{taskId}/complete (or /fail)
[server/internal/handler/daemon.go:1972 (CompleteTask) / L2142 (FailTask)]
       │
       ▼  Calls TaskService
[server/internal/service/task.go:1279 (CompleteTask) / L1466 (FailTask)]
       │
       ├─► DB Tx: CompleteAgentTask / FailAgentTask (Persists outcome)
       ├─► OBS-7: emitPersistSpan (Terminal persistence span)
       │
       ▼  Calls s.broadcastTaskEvent(ctx, protocol.EventTaskCompleted, task)
[server/internal/service/task.go:1445 / L1593 / L2118 (broadcastTaskEvent)]
       │
       ▼  Publishes to events.Bus
[server/internal/events/bus.go:61 (Publish)]
       │
       ▼  Listened by bus.SubscribeAll in server listeners
[server/cmd/server/listeners.go:151 (registerListeners)]
       │
       ▼  Calls b.BroadcastToWorkspace(workspaceID, jsonPayload)
[server/internal/realtime/hub.go:267 (Hub) / L331 (fanoutAll / broadcast)]
       │
       ▼  WebSocket Frame over TLS/TCP to Web Client
[packages/core/realtime/use-realtime-sync.ts:485 (task:* handler) / L974 (task:completed)]
       │
       ▼  Invalidates React Query caches (agentTaskSnapshot, agentTasks, issues/tasks)
[Web UI / Kanban Board UI Updates & Renders Final Terminal Result]
```

---

## 3. Exact File & Line Evidence

### A. Terminal Outcome Ingress & Persistence (Server API & TaskService)
- **`server/internal/handler/daemon.go:1972`** (`CompleteTask`): Receives daemon completion payload (`output`, `session_id`, `pr_url`) and delegates to `TaskService`.
- **`server/internal/handler/daemon.go:2142`** (`FailTask`): Receives daemon failure payload (`error`, `session_id`, `failure_reason`) and delegates to `TaskService`.
- **`server/internal/service/task.go:1279`** (`CompleteTask`): Executes DB transaction `CompleteAgentTask`, writes assistant chat/comment records, emits OBS-7 persistence span (`emitPersistSpan` at line 1350), and triggers event broadcast.
- **`server/internal/service/task.go:1466`** (`FailTask`): Executes DB transaction `FailAgentTask`, records failure reason, emits failure comment/chat message, and triggers event broadcast.

### B. Domain Event Generation & Bus Publishing
- **`server/internal/service/task.go:1445`**: `s.broadcastTaskEvent(ctx, protocol.EventTaskCompleted, task)` called on completion.
- **`server/internal/service/task.go:1593`**: `s.broadcastTaskEvent(ctx, protocol.EventTaskFailed, task)` called on failure.
- **`server/internal/service/task.go:2118-2139`** (`broadcastTaskEvent`): Builds event payload carrying `task_id`, `agent_id`, `issue_id`, `status`, and optional `chat_session_id`, then calls `s.Bus.Publish(events.Event{Type: "task:completed" | "task:failed", WorkspaceID: workspaceID, Payload: payload})`.
- **`server/internal/events/bus.go:61-88`** (`Publish`): Synchronously dispatches event to all registered listeners and global subscribers.

### C. Server-Side Realtime Hub Broadcast to Web UI
- **`server/cmd/server/listeners.go:151-193`** (`registerListeners`): Subscribes to `events.Bus` via `SubscribeAll`. Filters out personal events and routes all workspace events (including `task:completed` and `task:failed`) to `b.BroadcastToWorkspace(e.WorkspaceID, data)`.
- **`server/internal/realtime/hub.go:267-335`** (`Hub`): Manages Web UI client WebSocket connections in scope rooms (workspace room `workspace:{workspaceID}`). `BroadcastToWorkspace` queues JSON frame into client send channels.

### D. Web UI Client Receiver & Kanban Invalidation
- **`packages/core/realtime/use-realtime-sync.ts:485-520`**: Global prefix handler for `task:` events. Upon receiving `task:completed` or `task:failed`, invalidates:
  - `agentTaskSnapshotKeys.list(wsId)` (Agent presence state)
  - `agentTasksKeys.all(wsId)` (Recent work list)
  - `["issues", "tasks"]` (Kanban card task execution log)
  - `["issues", "usage"]` (Issue token usage summary)
- **`packages/core/realtime/use-realtime-sync.ts:974-1007`**: Specific handlers for `task:completed` and `task:failed` for chat/task view clearing and cross-session aggregate invalidation.

---

## 4. Single Delivery Disposition Emission Anchor Analysis

### Location of UI Delivery Anchor
To capture **exactly one delivery disposition** for the Web/Kanban UI terminal result delivery hop:
- **Target File**: `server/internal/realtime/hub.go` (Web UI WebSocket Hub)
- **Delivery Point**: Inside `realtime.Hub` when a `task:completed` or `task:failed` frame is pushed onto a connected Web client's WebSocket channel (`sendToClient` / `writePump` in `server/internal/realtime/hub.go`).
- **Carrier Propagation**:
  - The event payload from `broadcastTaskEvent` (`server/internal/service/task.go:2123-2131`) contains `task_id` and optional `chat_session_id`.
  - The Web client WebSocket connection in `realtime.Hub` holds client `userID` and `workspaceID`.
  - A `delivery_id` is generated per frame delivery attempt.
  - The delivery disposition (`delivered`, `dropped`, `backpressure`) can be emitted via `e2e.Recorder` at the socket write success/drop point in `realtime.Hub`.

---

## 5. Audit Verification & Verdict
- **Status**: AUDIT COMPLETE (Read-only, 0 source modifications, 0 deployments).
- **Target Evidence File**: `.deploy-control/p0/evidence/T20-ui-delivery-anchor.md`
- **Result**: Clear distinction established between control-plane daemon wakeup (`daemonws`) and user-facing Web/Kanban UI terminal result delivery (`realtime.Hub`). Exact file:line anchor identified.
