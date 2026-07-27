package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/util"
)

// DaemonLedgerSummaryPayload is the transport contract reserved for ORQ-41
// Wave W4. The route is owned by W3 and durable persistence is owned by the
// serial LANE-DB; this file must not acknowledge a summary until both exist.
type DaemonLedgerSummaryPayload struct {
	TaskID         string `json:"task_id"`
	EverHadToolUse bool   `json:"ever_had_tool_use"`
	EverDefinite   bool   `json:"ever_definite"`
	EverAmbiguous  bool   `json:"ever_ambiguous"`
	EverSaturated  bool   `json:"ever_saturated"`
}

// PutDaemonLedgerSummary validates the reserved daemon transport contract and
// then fails closed. The durable ledger summary store (LANE-DB migration
// `task_ledger_summary`) and its generated upsert do not exist in this lane, so
// answering 200 would falsely claim durability: nothing is written here.
//
// Two requirements this handler cannot enforce alone, for whoever registers the
// route in W3 (`cmd/server/router.go` is owned by W3, not W4):
//  1. mount it inside the authenticated daemon group, never on a public route —
//     there is no authentication check in this function;
//  2. validate that {taskId} belongs to the calling daemon — this function
//     validates UUID *shape*, not ownership, so without that check one daemon
//     could report a summary for another daemon's task.
func (h *Handler) PutDaemonLedgerSummary(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	var payload DaemonLedgerSummaryPayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	if payload.TaskID == "" {
		payload.TaskID = taskID
	} else if !strings.EqualFold(payload.TaskID, taskID) {
		writeError(w, http.StatusBadRequest, "task_id mismatch between url and payload")
		return
	}

	if _, err := util.ParseUUID(payload.TaskID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid task_id uuid format")
		return
	}

	writeError(w, http.StatusServiceUnavailable, "ledger summary persistence is not configured; fail closed")
}
