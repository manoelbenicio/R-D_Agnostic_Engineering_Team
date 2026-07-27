package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestPutDaemonLedgerSummary_Valid(t *testing.T) {
	h := &Handler{}

	taskID := uuid.New().String()
	payload := DaemonLedgerSummaryPayload{
		TaskID:         taskID,
		EverHadToolUse: true,
		EverDefinite:   false,
		EverAmbiguous:  false,
		EverSaturated:  false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/daemon/tasks/"+taskID+"/ledger-summary", bytes.NewReader(body))
	r := chi.NewRouter()
	r.Put("/api/daemon/tasks/{taskId}/ledger-summary", h.PutDaemonLedgerSummary)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp DaemonLedgerSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "recorded" {
		t.Errorf("expected status 'recorded', got '%s'", resp.Status)
	}
	if resp.TaskID != taskID {
		t.Errorf("expected task_id '%s', got '%s'", taskID, resp.TaskID)
	}
}

func TestPutDaemonLedgerSummary_InvalidUUID(t *testing.T) {
	h := &Handler{}

	payload := DaemonLedgerSummaryPayload{
		TaskID:         "invalid-uuid",
		EverHadToolUse: false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/daemon/tasks/invalid-uuid/ledger-summary", bytes.NewReader(body))
	r := chi.NewRouter()
	r.Put("/api/daemon/tasks/{taskId}/ledger-summary", h.PutDaemonLedgerSummary)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
	}
}

func TestPutDaemonLedgerSummary_IDMismatch(t *testing.T) {
	h := &Handler{}

	taskID1 := uuid.New().String()
	taskID2 := uuid.New().String()

	payload := DaemonLedgerSummaryPayload{
		TaskID:         taskID2,
		EverHadToolUse: false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/daemon/tasks/"+taskID1+"/ledger-summary", bytes.NewReader(body))
	r := chi.NewRouter()
	r.Put("/api/daemon/tasks/{taskId}/ledger-summary", h.PutDaemonLedgerSummary)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request on mismatch, got %d", w.Code)
	}
}
