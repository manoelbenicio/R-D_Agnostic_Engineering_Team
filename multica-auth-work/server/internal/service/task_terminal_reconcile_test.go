package service

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/multica-ai/multica/server/internal/events"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// mockReconcileDBTX tracks calls to RefreshAgentStatusFromTasks.
type mockReconcileDBTX struct {
	task           db.AgentTaskQueue
	reconcileCount int
}

func (m *mockReconcileDBTX) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m *mockReconcileDBTX) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (m *mockReconcileDBTX) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	if strings.Contains(sql, "RefreshAgentStatusFromTasks") {
		m.reconcileCount++
		return &mockAgentRow{agent: db.Agent{ID: m.task.AgentID, Status: "idle"}}
	}
	if strings.Contains(sql, "SET status =") {
		return &mockRow{err: pgx.ErrNoRows}
	}
	return &mockRow{task: &m.task}
}

type mockAgentRow struct {
	agent db.Agent
}

func (r *mockAgentRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		if a, ok := dest[0].(*db.Agent); ok {
			*a = r.agent
		}
	}
	return nil
}

func TestTerminalTransitions_ReconcileAgentStatusWhenAlreadyFinalized(t *testing.T) {
	taskID := testUUID(10)
	agentID := testUUID(20)

	mock := &mockReconcileDBTX{
		task: db.AgentTaskQueue{
			ID:      taskID,
			AgentID: agentID,
			Status:  "failed",
		},
	}
	svc := &TaskService{
		Queries: db.New(mock),
		Bus:     events.New(),
	}

	ctx := context.Background()

	// 1. FailTask on already finalized task
	mock.reconcileCount = 0
	_, err := svc.FailTask(ctx, taskID, "crashed", "", "", "")
	if err != nil {
		t.Fatalf("FailTask failed: %v", err)
	}
	if mock.reconcileCount == 0 {
		t.Error("expected FailTask on already-finalized task to trigger ReconcileAgentStatus")
	}

	// 2. CompleteTask on already finalized task
	mock.reconcileCount = 0
	_, err = svc.CompleteTask(ctx, taskID, nil, "", "")
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}
	if mock.reconcileCount == 0 {
		t.Error("expected CompleteTask on already-finalized task to trigger ReconcileAgentStatus")
	}

	// 3. CancelTaskWithResult on already finalized task
	mock.reconcileCount = 0
	_, err = svc.CancelTaskWithResult(ctx, taskID)
	if err != nil {
		t.Fatalf("CancelTaskWithResult failed: %v", err)
	}
	if mock.reconcileCount == 0 {
		t.Error("expected CancelTaskWithResult on already-finalized task to trigger ReconcileAgentStatus")
	}
}
