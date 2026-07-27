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

// A well-formed report is still refused: the durable store does not exist in
// this lane, so the endpoint must not answer success and must not use the word
// "recorded". The assertion covers both the status code and the body claim.
func TestPutDaemonLedgerSummary_WellFormedPayloadFailsClosedWithoutClaimingPersistence(t *testing.T) {
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

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 Service Unavailable, got %d. Body: %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`"status":"recorded"`)) {
		t.Fatalf("fail-closed scaffold must not claim persistence: %s", w.Body.String())
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
