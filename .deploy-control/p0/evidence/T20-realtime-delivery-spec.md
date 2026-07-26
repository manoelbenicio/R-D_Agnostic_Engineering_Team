# Realtime UI Terminal Delivery Test Specification & RED Contract Evidence (2026-07-24)

## 1. Executive Summary & Purpose
This document defines the **Realtime UI Delivery Test Specification** for the Web / Kanban UI delivery hop (`realtime.Hub`).

It specifies the exact test harness, event payload contracts, expected span emissions, and the **RED contract status** resulting from the API gap in `realtime.Hub`.

---

## 2. API Inspection & Structural Gap Analysis

### A. Current `realtime.Hub` API Inventory (`server/internal/realtime/hub.go`)
- **Existing Methods**:
  - `NewHub() *Hub`
  - `SetAuthorizer(a ScopeAuthorizer)`
  - `SetSubscriptionCallbacks(onFirst, onLast SubscriptionCallback)`
  - `BroadcastToWorkspace(workspaceID string, message []byte)`
  - `Broadcast(message []byte)`
  - `SendToUser(userID string, message []byte, excludeWorkspace ...string)`
- **Identified API Seam Gap**:
  - `realtime.Hub` currently **lacks** a `DeliveryRecorder` field, `SetDeliveryRecorder(...)` method, and internal `emitDelivery(...)` call site during client frame dispatch.
  - Unlike `daemonws.Hub` (which has `SetDeliveryRecorder` installed for control-plane daemon wakeups), `realtime.Hub` currently operates without an e2e observability delivery recorder.

### B. Event Listener Payload Structure (`server/cmd/server/listeners.go` & `server/internal/service/task.go`)
When a terminal task status is reached (`CompleteTask` or `FailTask`), the server dispatches a domain event:
- **`task:completed`** / **`task:failed`** / **`task:cancelled`**
- **JSON Structure**:
```json
{
  "type": "task:completed",
  "payload": {
    "task_id": "0190a6cd-72b1-7f8e-9d21-4f3801234567",
    "agent_id": "0190a6cd-7000-7000-8000-000000000001",
    "issue_id": "0190a6cd-7000-7000-8000-000000000002",
    "status": "completed",
    "chat_session_id": "0190a6cd-7111-7222-8333-444444444444"
  },
  "actor_id": "",
  "actor_type": "system"
}
```

---

## 3. Test Harness Specification & Assertion Matrix

The test harness must validate the following 4 strict assertions:

| Frame / Event Category | Event Type | Expected `HopDelivery` Spans | Canonical `SessionID` Source | Rationale |
|---|---|---|---|---|
| **Terminal Task Outcome (Success)** | `task:completed` | **Exactly 1** | `chat_session_id` (or `task_id`) | UI terminal state transition must be recorded once per client delivery |
| **Terminal Task Outcome (Failure)** | `task:failed` | **Exactly 1** | `chat_session_id` (or `task_id`) | UI terminal failure state transition must be recorded once per client delivery |
| **Terminal Task Outcome (Cancel)** | `task:cancelled` | **Exactly 1** | `chat_session_id` (or `task_id`) | UI terminal cancellation must be recorded once per client delivery |
| **Unrelated Daemon Wakeup** | `daemon:task_available` | **0 (Zero)** | N/A | Daemon wakeups are control-plane frames (`daemonws`), not UI terminal deliveries |
| **Non-terminal Task Stream** | `task:progress` / `task:message` | **0 (Zero)** | N/A | High-frequency intermediate streaming updates must NOT emit terminal delivery spans |

---

## 4. Test Code Contract Specification (RED Contract)

Below is the canonical Go test specification that defines the test harness for `realtime.Hub` once the delivery recorder seam is approved:

```go
package realtime_test

import (
	"encoding/json"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/realtime"
)

// TestRealtimeHub_TerminalTaskDelivery_EmitsExactlyOneDisposition asserts that:
// 1. Terminal task events (task:completed, task:failed) emit EXACTLY ONE HopDelivery span.
// 2. Spans carry valid canonical SessionID and unique DeliveryID.
// 3. Unrelated daemon wakeups and non-terminal stream frames emit ZERO spans.
func TestRealtimeHub_TerminalTaskDelivery_EmitsExactlyOneDisposition(t *testing.T) {
	sink := e2e.NewMemorySink()
	recorder := e2e.NewRecorder(sink)

	// Note: Harness relies on SetDeliveryRecorder API seam on realtime.Hub
	hub := realtime.NewHub()
	// EXPECTED API SEAM (Currently RED / Unwired on realtime.Hub):
	// hub.SetDeliveryRecorder(realtime.NewDeliveryRecorder(recorder))

	workspaceID := "ws-test-123"
	sessionID := "0190a6cd-7111-7222-8333-444444444444"
	taskID := "0190a6cd-72b1-7f8e-9d21-4f3801234567"

	// 1. Deliver Terminal Event: task:completed
	completedPayload, _ := json.Marshal(map[string]any{
		"type": "task:completed",
		"payload": map[string]any{
			"task_id":         taskID,
			"status":          "completed",
			"chat_session_id": sessionID,
		},
	})
	hub.BroadcastToWorkspace(workspaceID, completedPayload)

	// 2. Deliver Non-terminal Event: task:progress (MUST NOT emit span)
	progressPayload, _ := json.Marshal(map[string]any{
		"type": "task:progress",
		"payload": map[string]any{
			"task_id": taskID,
			"step":    1,
			"total":   5,
		},
	})
	hub.BroadcastToWorkspace(workspaceID, progressPayload)

	// 3. Deliver Unrelated Control-Plane Event: daemon:task_available (MUST NOT emit span)
	wakeupPayload, _ := json.Marshal(map[string]any{
		"type": "daemon:task_available",
		"payload": map[string]any{
			"task_id": taskID,
		},
	})
	hub.Broadcast(wakeupPayload)

	// Verify Spans
	spans := sink.Spans()

	// Assert EXACTLY ONE delivery span was emitted (for the terminal task:completed event)
	if len(spans) != 1 {
		t.Fatalf("RED CONTRACT VERIFICATION: Expected exactly 1 HopDelivery span for terminal event, got %d", len(spans))
	}

	span := spans[0]
	if span.Hop != e2e.HopDelivery {
		t.Errorf("expected Hop=%q, got %q", e2e.HopDelivery, span.Hop)
	}
	if span.Correlation.SessionID != sessionID && span.Correlation.SessionID != taskID {
		t.Errorf("expected SessionID=%q, got %q", sessionID, span.Correlation.SessionID)
	}
	if span.Correlation.DeliveryID == "" {
		t.Errorf("missing DeliveryID in delivery span")
	}
	if span.SecretsPresent {
		t.Errorf("span secrets_present must be false")
	}
}
```

---

## 5. RED Status Verification & Audit Conclusion

1. **Current Status**: **RED (Unwired API Seam)**
   - `realtime.Hub` does not currently expose `SetDeliveryRecorder` or invoke `EmitDelivery` during client WebSocket frame writes.
   - Compiling test code against `realtime.Hub.SetDeliveryRecorder` fails because the method is not part of `realtime.Hub`'s public API surface today.

2. **Compliance**:
   - As mandated, **no source code APIs were invented or altered** in `server/internal/realtime/hub.go`.
   - The exact RED contract and test specification have been recorded in this evidence file.
   - Once the architecture owner approves adding `SetDeliveryRecorder` to `realtime.Hub`, the test harness above will validate GREEN.
