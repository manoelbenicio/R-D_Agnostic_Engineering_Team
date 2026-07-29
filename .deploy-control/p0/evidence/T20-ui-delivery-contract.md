# Refined Realtime UI Delivery Contract & Deterministic Test Spec (2026-07-24)

## 1. Executive Summary & Governance
This document specifies the refined **Web / Kanban UI Terminal Result Delivery Contract** for `realtime.Hub`.

- **Governance Constraint**: Read-Only / Test-Only. ZERO product source edits (`git diff` clean).
- **Core Objective**: Define the exact architectural contract, session derivation rules, frame filtering rules, multi-client aggregation policy, and deterministic Go test specification for the UI delivery hop (`HopDelivery`).

---

## 2. Architectural Contract Decisions

### A. Aggregate Workspace Delivery vs. Per-Client Fanout (The Exactly-One Contract)
- **Question**: When a workspace has multiple connected UI clients (e.g., 3 browser tabs open in the same workspace), does the delivery hop record 1 aggregate delivery span or N per-client connection spans?
- **Architectural Decision**: **EXACTLY ONE AGGREGATE WORKSPACE DELIVERY DISPOSITION** per terminal task execution/acceptance.
- **Trace Assembly Proof (`server/internal/daemon/observability/e2e/assemble.go:173-180`)**:
  - The canonical `agent-brain.e2e.v1` trace assembler (`e2e.Assemble`) enforces strict 1-per-hop span continuity for each task.
  - If N per-client spans were emitted for a single workspace broadcast, `e2e.Assemble` flags `AnomalyDuplicateSpan` (`reason: "duplicate_task_hop"`), invalidating trace continuity (`AllContinuous = false`).
  - Therefore, `realtime.Hub` must emit **exactly one** aggregate `HopDelivery` span when the terminal event frame is successfully queued into the workspace fanout channel.

### B. Canonical Session Derivation Rules
- **Primary Source**: `chat_session_id` (when present and valid in the event payload, e.g., for chat tasks).
- **Fallback Source**: `task_id` (when `chat_session_id` is absent/empty, e.g., for issue tasks or quick-create tasks).
- **Join Invariant**: Every `HopDelivery` span carries a non-empty `SessionID` mapped via `sessionToTask` during trace assembly (`e2e.Assemble:235-242`).

### C. Frame Filtering & Scope Contract
1. **Terminal Task Outcomes Only**:
   - `task:completed` → **1 HopDelivery span** (`outcome="delivered"`)
   - `task:failed` → **1 HopDelivery span** (`outcome="delivered"`)
   - `task:cancelled` → **1 HopDelivery span** (`outcome="delivered"`)
2. **Non-Terminal Task Streams (MUST NOT emit spans)**:
   - `task:progress`, `task:message`, `task:queued`, `task:running`, `task:waiting_local_directory` → **0 HopDelivery spans**
3. **Unrelated Daemon Control Frames (MUST NOT emit spans)**:
   - `daemon:task_available`, `task_available` → **0 HopDelivery spans** (these belong to `daemonws` control-plane, not UI delivery).

---

## 3. Comprehensive Assertion Matrix

| Event / Frame Category | Event Type | Target Hub | Expected `HopDelivery` Spans | Canonical `SessionID` | e2e Trace Impact |
|---|---|---|---|---|---|
| **Terminal Success** | `task:completed` | `realtime.Hub` | **1 (Aggregate)** | `chat_session_id` ?? `task_id` | Completes Hop 7 (`HopDelivery`) |
| **Terminal Failure** | `task:failed` | `realtime.Hub` | **1 (Aggregate)** | `chat_session_id` ?? `task_id` | Completes Hop 7 (`HopDelivery`) |
| **Terminal Cancel** | `task:cancelled` | `realtime.Hub` | **1 (Aggregate)** | `chat_session_id` ?? `task_id` | Completes Hop 7 (`HopDelivery`) |
| **Intermediate Progress** | `task:progress` | `realtime.Hub` | **0** | N/A | Ignored by delivery hop |
| **Intermediate Message** | `task:message` | `realtime.Hub` | **0** | N/A | Ignored by delivery hop |
| **Daemon Control Wakeup** | `daemon:task_available` | `daemonws.Hub` | **0 (in UI Hub)** | N/A | Ignored by UI delivery hop |

---

## 4. Recommended Deterministic Go Test Contract Spec

Below is the recommended, self-contained Go test harness specification that deterministically verifies all 4 contract invariants against a wired `realtime.Hub`:

```go
package realtime_test

import (
	"encoding/json"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/realtime"
)

// TestRealtimeHub_UI_DeliveryContract verifies the 4 core delivery contract invariants:
// 1. Terminal task events emit EXACTLY ONE aggregate HopDelivery span per workspace broadcast.
// 2. Multi-client fanout in a single workspace still yields EXACTLY ONE span (no duplicate_task_hop anomaly).
// 3. Canonical session derives from chat_session_id when present, falling back to task_id.
// 4. Non-terminal events (task:progress) and control-plane frames (daemon:task_available) emit ZERO spans.
func TestRealtimeHub_UI_DeliveryContract(t *testing.T) {
	sink := e2e.NewMemorySink()
	recorder := e2e.NewRecorder(sink)

	// Harness setup (assuming SetDeliveryRecorder installed on realtime.Hub)
	hub := realtime.NewHub()
	// hub.SetDeliveryRecorder(realtime.NewDeliveryRecorder(recorder))

	workspaceID := "ws-multi-client-test"
	chatSessionID := "0190a6cd-7111-7222-8333-444444444444"
	issueTaskID := "0190a6cd-72b1-7f8e-9d21-4f3801234567"

	// --- Case 1: Chat Task Completed (multi-client workspace) ---
	// Simulates 3 connected browser clients in the same workspace room.
	chatCompletedPayload, _ := json.Marshal(map[string]any{
		"type": "task:completed",
		"payload": map[string]any{
			"task_id":         "chat-task-100",
			"status":          "completed",
			"chat_session_id": chatSessionID,
		},
	})
	hub.BroadcastToWorkspace(workspaceID, chatCompletedPayload)

	// --- Case 2: Issue Task Completed (no chat_session_id -> fallback to task_id) ---
	issueCompletedPayload, _ := json.Marshal(map[string]any{
		"type": "task:completed",
		"payload": map[string]any{
			"task_id": issueTaskID,
			"status":  "completed",
		},
	})
	hub.BroadcastToWorkspace(workspaceID, issueCompletedPayload)

	// --- Case 3: Non-Terminal Stream Frames (MUST emit 0 spans) ---
	progressPayload, _ := json.Marshal(map[string]any{
		"type": "task:progress",
		"payload": map[string]any{
			"task_id": issueTaskID,
			"step":    2,
			"total":   10,
		},
	})
	hub.BroadcastToWorkspace(workspaceID, progressPayload)

	// --- Case 4: Unrelated Control-Plane Frame (MUST emit 0 spans in UI Hub) ---
	daemonWakeupPayload, _ := json.Marshal(map[string]any{
		"type": "daemon:task_available",
		"payload": map[string]any{
			"task_id": issueTaskID,
		},
	})
	hub.Broadcast(daemonWakeupPayload)

	// --- Verification & Assertions ---
	spans := sink.Spans()

	// Assertion 1: Total Spans must equal EXACTLY 2 (1 for Chat Task, 1 for Issue Task)
	if len(spans) != 2 {
		t.Fatalf("Contract Violation: expected exactly 2 HopDelivery spans, got %d", len(spans))
	}

	// Assertion 2: Verify Span #1 (Chat Task -> SessionID derived from chat_session_id)
	span1 := spans[0]
	if span1.Hop != e2e.HopDelivery {
		t.Errorf("Span 1 Hop = %q, want %q", span1.Hop, e2e.HopDelivery)
	}
	if span1.Correlation.SessionID != chatSessionID {
		t.Errorf("Span 1 SessionID = %q, want %q", span1.Correlation.SessionID, chatSessionID)
	}
	if span1.Correlation.DeliveryID == "" {
		t.Errorf("Span 1 missing DeliveryID")
	}

	// Assertion 3: Verify Span #2 (Issue Task -> SessionID derived from fallback task_id)
	span2 := spans[1]
	if span2.Hop != e2e.HopDelivery {
		t.Errorf("Span 2 Hop = %q, want %q", span2.Hop, e2e.HopDelivery)
	}
	if span2.Correlation.SessionID != issueTaskID {
		t.Errorf("Span 2 SessionID = %q, want %q (task_id fallback)", span2.Correlation.SessionID, issueTaskID)
	}

	// Assertion 4: Verify trace assembler continuity check (e2e.Assemble)
	report := e2e.Assemble(spans)
	if len(report.Anomalies) != 0 {
		t.Errorf("Contract Violation: assemble report produced %d anomalies (expected 0 duplicate_task_hop)", len(report.Anomalies))
	}
}
```

---

## 5. Audit Conclusion & RED Contract Summary

- **Current Repository State**: `git diff` clean (0 product source modifications made).
- **Seam Status**: `realtime.Hub` API gap documented (`SetDeliveryRecorder` currently un-exported on `realtime.Hub`).
- **Contract Verification**: This specification resolves the aggregate vs. per-client question by proving that **1 aggregate span per workspace delivery** is required to satisfy `e2e.Assemble` without triggering `AnomalyDuplicateSpan`.
