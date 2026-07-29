package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// mockRow implements pgx.Row, returning either a scanned task or pgx.ErrNoRows.
type mockRow struct {
	task *db.AgentTaskQueue
	err  error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	t := r.task
	ptrs := []any{
		&t.ID, &t.AgentID, &t.IssueID, &t.Status, &t.Priority,
		&t.DispatchedAt, &t.StartedAt, &t.CompletedAt, &t.Result,
		&t.Error, &t.CreatedAt, &t.Context, &t.RuntimeID,
		&t.SessionID, &t.WorkDir, &t.TriggerCommentID,
		&t.ChatSessionID, &t.AutopilotRunID,
	}
	for i, p := range ptrs {
		if i >= len(dest) {
			break
		}
		// Copy value from source to dest by assigning through the pointer.
		switch d := dest[i].(type) {
		case *pgtype.UUID:
			*d = *(p.(*pgtype.UUID))
		case *string:
			*d = *(p.(*string))
		case *int32:
			*d = *(p.(*int32))
		case *pgtype.Timestamptz:
			*d = *(p.(*pgtype.Timestamptz))
		case *[]byte:
			*d = *(p.(*[]byte))
		case *pgtype.Text:
			*d = *(p.(*pgtype.Text))
		}
	}
	return nil
}

// mockDBTX routes QueryRow calls: complete/fail queries return ErrNoRows,
// getAgentTask returns the stored task.
type mockDBTX struct {
	task db.AgentTaskQueue
}

func (m *mockDBTX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDBTX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (m *mockDBTX) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	// CompleteAgentTask and FailAgentTask SQL contain "SET status ="
	if strings.Contains(sql, "SET status =") {
		return &mockRow{err: pgx.ErrNoRows}
	}
	// GetAgentTask — return the existing task
	return &mockRow{task: &m.task}
}

func testUUID(b byte) pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	u.Bytes[0] = b
	return u
}

func TestCompleteTask_AlreadyFinalized(t *testing.T) {
	taskID := testUUID(1)
	agentID := testUUID(2)

	tests := []struct {
		name   string
		status string
	}{
		{"already completed", "completed"},
		{"already cancelled", "cancelled"},
		{"already failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDBTX{task: db.AgentTaskQueue{
				ID:      taskID,
				AgentID: agentID,
				Status:  tt.status,
			}}
			svc := &TaskService{
				Queries: db.New(mock),
				Bus:     events.New(),
			}

			got, err := svc.CompleteTask(context.Background(), taskID, nil, "", "")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == nil {
				t.Fatal("expected task, got nil")
			}
			if got.Status != tt.status {
				t.Errorf("expected status %q, got %q", tt.status, got.Status)
			}
			if got.ID != taskID {
				t.Error("returned task ID doesn't match")
			}
		})
	}
}

func TestFailTask_AlreadyFinalized(t *testing.T) {
	taskID := testUUID(1)
	agentID := testUUID(2)

	tests := []struct {
		name   string
		status string
	}{
		{"already completed", "completed"},
		{"already cancelled", "cancelled"},
		{"already failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDBTX{task: db.AgentTaskQueue{
				ID:      taskID,
				AgentID: agentID,
				Status:  tt.status,
			}}
			svc := &TaskService{
				Queries: db.New(mock),
				Bus:     events.New(),
			}

			got, err := svc.FailTask(context.Background(), taskID, "agent crashed", "", "", "")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == nil {
				t.Fatal("expected task, got nil")
			}
			if got.Status != tt.status {
				t.Errorf("expected status %q, got %q", tt.status, got.Status)
			}
			if got.ID != taskID {
				t.Error("returned task ID doesn't match")
			}
		})
	}
}

func TestTaskFailureClassifiers(t *testing.T) {
	cases := []struct {
		reason       string
		wantType     string
		wantResumeOK bool
		wantRetry    bool
	}{
		{reason: "timeout", wantType: "timeout", wantResumeOK: true, wantRetry: true},
		{reason: "codex_semantic_inactivity", wantType: "timeout", wantResumeOK: false, wantRetry: true},
		{reason: "runtime_recovery", wantType: "runtime", wantResumeOK: true, wantRetry: true},
		{reason: "iteration_limit", wantType: "agent_output", wantResumeOK: false, wantRetry: false},
		{reason: "api_invalid_request", wantType: "agent_error", wantResumeOK: false, wantRetry: false},
		{reason: "agent_error", wantType: "agent_error", wantResumeOK: true, wantRetry: false},
	}

	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			if got := taskErrorType(tc.reason); got != tc.wantType {
				t.Fatalf("taskErrorType(%q) = %q, want %q", tc.reason, got, tc.wantType)
			}
			if got := !resumeUnsafeFailureReason(tc.reason); got != tc.wantResumeOK {
				t.Fatalf("resume-safe(%q) = %v, want %v", tc.reason, got, tc.wantResumeOK)
			}
			if got := retryableReasons[tc.reason]; got != tc.wantRetry {
				t.Fatalf("retryableReasons[%q] = %v, want %v", tc.reason, got, tc.wantRetry)
			}
		})
	}
}

// --- F6: persist-hop (OBS-7) span emitted only after terminal persistence ---

func TestEmitPersistSpanAfterPersistenceMetadataOnly(t *testing.T) {
	sink := e2e.NewMemorySink()
	svc := &TaskService{Obs: e2e.NewRecorder(sink)}
	id := testUUID(21)
	persisted := db.AgentTaskQueue{
		ID:     id,
		Status: "completed",
		Result: []byte(`{"output":"x"}`),
	}
	svc.emitPersistSpan(persisted, 7*time.Millisecond)
	if sink.Len() != 1 {
		t.Fatalf("sink len = %d, want 1", sink.Len())
	}
	sp := sink.Spans()[0]
	wantID := util.UUIDToString(id)
	if sp.Hop != e2e.HopPersist || sp.Correlation.TaskID != wantID || sp.Correlation.ResultID != wantID {
		t.Fatalf("unexpected persist span: hop=%q task=%q result=%q", sp.Hop, sp.Correlation.TaskID, sp.Correlation.ResultID)
	}
	if sp.Labels["terminal_status"] != "completed" {
		t.Fatalf("terminal_status label=%q, want completed", sp.Labels["terminal_status"])
	}
	if sp.Counters["byte_count"] != int64(len(persisted.Result)) {
		t.Fatalf("byte_count=%d, want %d", sp.Counters["byte_count"], len(persisted.Result))
	}
	if sp.Counters["persist_latency_ms"] != 7 {
		t.Fatalf("persist_latency_ms=%d, want 7", sp.Counters["persist_latency_ms"])
	}
	for k := range sp.Counters {
		switch k {
		case "persist_latency_ms", "byte_count", "token_count":
		default:
			t.Fatalf("unexpected counter key %q", k)
		}
	}
	if r := e2e.ScanSpans([]e2e.Span{sp}); !r.Clean {
		t.Fatalf("leak scan not clean: %+v", r.Findings)
	}
}

// TestEmitPersistSpanOrderingConsumesPersistedRow proves the persist span is
// derived from the PERSISTED row (byte_count == persisted Result length), so it
// can only carry post-persistence state; and that a nil Obs recorder is a no-op.
func TestEmitPersistSpanOrderingConsumesPersistedRow(t *testing.T) {
	sink := e2e.NewMemorySink()
	svc := &TaskService{Obs: e2e.NewRecorder(sink)}
	svc.emitPersistSpan(db.AgentTaskQueue{ID: testUUID(22), Status: "completed", Result: []byte("abcd")}, time.Millisecond)
	if sink.Len() != 1 || sink.Spans()[0].Counters["byte_count"] != 4 {
		t.Fatalf("persist span did not reflect persisted row bytes: %+v", sink.Spans())
	}
	// Nil Obs: no recorder, no panic, nothing emitted.
	(&TaskService{}).emitPersistSpan(db.AgentTaskQueue{ID: testUUID(23), Result: []byte("z")}, 0)
}
