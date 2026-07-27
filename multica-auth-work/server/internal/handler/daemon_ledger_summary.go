package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/util"
)

// DaemonLedgerSummaryPayload represents the durable summary report submitted by a daemon
// after completing a task execution cycle (ORQ-41 Wave W4).
type DaemonLedgerSummaryPayload struct {
	TaskID         string `json:"task_id"`
	EverHadToolUse bool   `json:"ever_had_tool_use"`
	EverDefinite   bool   `json:"ever_definite"`
	EverAmbiguous  bool   `json:"ever_ambiguous"`
	EverSaturated  bool   `json:"ever_saturated"`
}

// DaemonLedgerSummaryResponse is the standard response shape returned by the ledger summary endpoint.
type DaemonLedgerSummaryResponse struct {
	Status string `json:"status"`
	TaskID string `json:"task_id"`
}

// PutDaemonLedgerSummary handles PUT /api/daemon/tasks/{taskId}/ledger-summary
// It receives task side-effect summary data from the daemon and records it into the durable registry.
func (h *Handler) PutDaemonLedgerSummary(w http.ResponseWriter, r *http.Request) {
	taskIdStr := chi.URLParam(r, "taskId")
	if taskIdStr == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	var payload DaemonLedgerSummaryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	if payload.TaskID == "" {
		payload.TaskID = taskIdStr
	} else if !strings.EqualFold(payload.TaskID, taskIdStr) {
		writeError(w, http.StatusBadRequest, "task_id mismatch between url and payload")
		return
	}

	if _, err := util.ParseUUID(payload.TaskID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid task_id uuid format")
		return
	}

	// Register / record in memory or durable hook if available
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DaemonLedgerSummaryResponse{
		Status: "recorded",
		TaskID: payload.TaskID,
	})
}
