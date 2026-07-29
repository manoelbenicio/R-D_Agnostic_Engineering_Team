package main

import (
	"context"
	"sync"
	"testing"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/service"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestTerminalCompletionReconcilesFromOverlappingActiveTasks(t *testing.T) {
	if testPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()

	var runtimeID string
	if err := testPool.QueryRow(ctx, `
		SELECT id FROM agent_runtime WHERE workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("load integration runtime: %v", err)
	}

	var agentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id, status
		)
		VALUES ($1, 'Terminal reconcile overlap', '', 'cloud', '{}'::jsonb, $2, 'workspace', 3, $3, 'working')
		RETURNING id
	`, testWorkspaceID, runtimeID, testUserID).Scan(&agentID); err != nil {
		t.Fatalf("create overlap test agent: %v", err)
	}

	taskIDs := make([]string, 3)
	for i := range taskIDs {
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (
				agent_id, runtime_id, status, priority, dispatched_at, started_at
			)
			VALUES ($1, $2, 'running', 0, now(), now())
			RETURNING id
		`, agentID, runtimeID).Scan(&taskIDs[i]); err != nil {
			t.Fatalf("create overlapping task %d: %v", i, err)
		}
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE agent_id = $1`, agentID)
		testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentID)
	})

	taskService := service.NewTaskService(db.New(testPool), testPool, nil, events.New())

	if _, err := taskService.CompleteTask(ctx, parseUUID(taskIDs[0]), nil, "", ""); err != nil {
		t.Fatalf("complete first overlapping task: %v", err)
	}
	assertIntegrationAgentStatus(t, agentID, "working")

	// A duplicate terminal callback must still recompute from the database.
	// The two other running rows protect the agent from being released early.
	if _, err := taskService.CompleteTask(ctx, parseUUID(taskIDs[0]), nil, "", ""); err != nil {
		t.Fatalf("retry already-completed task: %v", err)
	}
	assertIntegrationAgentStatus(t, agentID, "working")

	// Finish the remaining tasks concurrently to exercise the terminal race.
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, taskID := range taskIDs[1:] {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := taskService.CompleteTask(ctx, parseUUID(id), nil, "", "")
			errCh <- err
		}(taskID)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("complete concurrent overlapping task: %v", err)
		}
	}

	assertIntegrationAgentStatus(t, agentID, "idle")
	var completed int
	if err := testPool.QueryRow(ctx, `
		SELECT count(*) FROM agent_task_queue
		WHERE agent_id = $1 AND status = 'completed'
	`, agentID).Scan(&completed); err != nil {
		t.Fatalf("count completed overlapping tasks: %v", err)
	}
	if completed != len(taskIDs) {
		t.Fatalf("completed overlapping tasks = %d, want %d", completed, len(taskIDs))
	}
}

func assertIntegrationAgentStatus(t *testing.T, agentID, want string) {
	t.Helper()
	var got string
	if err := testPool.QueryRow(context.Background(), `SELECT status FROM agent WHERE id = $1`, agentID).Scan(&got); err != nil {
		t.Fatalf("load agent status: %v", err)
	}
	if got != want {
		t.Fatalf("agent status = %q, want %q", got, want)
	}
}
