package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ORQ-41 acceptance coverage: Kanban metadata must never buy paid execution.
//
// The live regression that motivated this contract (2026-07-29T15:33Z) was a
// status-only transition on an issue whose agent assignee was left over from a
// previous run: it created a paid task even though nobody asked for new work.
// ORQ-68 closed the terminal (done) case; these tests pin the general rule for
// every status and both endpoints:
//
//   - a field-only update (status / title / description / priority) on an issue
//     with a historical (already-dispatched) agent or squad assignee enqueues
//     ZERO tasks;
//   - an explicit assignee transition to an executable agent / squad leader
//     enqueues EXACTLY ONE task, even when that assignee already ran on the
//     issue;
//   - the first activation of a never-dispatched backlog assignment still
//     dispatches (documented serial sub-task promotion).

func executionIntentTestName(t *testing.T) string {
	t.Helper()
	return strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
}

// seedHistoricalTask writes a completed task for the (issue, agent) pair. That
// row is what makes an assignee "historical/stale": execution was already
// bought once. Terminal status keeps it out of active-queue assertions.
func seedHistoricalTask(t *testing.T, issueID, agentID string) string {
	t.Helper()
	var taskID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id, status, priority,
			dispatched_at, started_at, completed_at
		)
		VALUES ($1, $2, $3, 'completed', 0, now(), now(), now())
		RETURNING id
	`, agentID, handlerTestRuntimeID(t), issueID).Scan(&taskID); err != nil {
		t.Fatalf("seed historical task: %v", err)
	}
	return taskID
}

func assignIssueDirectly(t *testing.T, issueID, assigneeType, assigneeID string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `
		UPDATE issue SET assignee_type = $2, assignee_id = $3 WHERE id = $1
	`, issueID, assigneeType, assigneeID); err != nil {
		t.Fatalf("assign issue directly: %v", err)
	}
}

func setIssueStatusDirectly(t *testing.T, issueID, status string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`UPDATE issue SET status = $2 WHERE id = $1`, issueID, status); err != nil {
		t.Fatalf("set issue status directly: %v", err)
	}
}

// countActiveQueueTasks mirrors the ORQ-41 acceptance query:
// SELECT count(*) FROM agent_task_queue
// WHERE status IN ('queued','dispatched','running','waiting_local_directory').
func countActiveQueueTasks(t *testing.T, issueID string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(), `
		SELECT count(*) FROM agent_task_queue
		WHERE issue_id = $1
		  AND status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
	`, issueID).Scan(&n); err != nil {
		t.Fatalf("count active queue tasks: %v", err)
	}
	return n
}

func createExecutionIntentIssue(t *testing.T, title, status string) string {
	t.Helper()
	issueID := createTestIssue(t, title, status, "high")
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
		testPool.Exec(context.Background(), `DELETE FROM comment WHERE issue_id = $1`, issueID)
		testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
	})
	return issueID
}

func applyIssueUpdate(t *testing.T, endpoint, issueID string, updates map[string]any) {
	t.Helper()
	var req *http.Request
	if endpoint == "single" {
		req = newRequest(http.MethodPut, "/api/issues/"+issueID, updates)
		req = withURLParam(req, "id", issueID)
	} else {
		req = newRequest(http.MethodPatch, "/api/issues/batch", map[string]any{
			"issue_ids": []string{issueID},
			"updates":   updates,
		})
	}
	w := httptest.NewRecorder()
	if endpoint == "single" {
		testHandler.UpdateIssue(w, req)
	} else {
		testHandler.BatchUpdateIssues(w, req)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("%s update %v: expected 200, got %d: %s", endpoint, updates, w.Code, w.Body.String())
	}
}

// TestFieldOnlyUpdateWithHistoricalAgentAssigneeNeverDispatches is the direct
// regression for the observed production defect, generalised past `done`:
// status-only transitions with a stale agent assignee must create no task, for
// nonterminal (in_progress / in_review / todo) as well as terminal statuses.
func TestFieldOnlyUpdateWithHistoricalAgentAssigneeNeverDispatches(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		for _, tc := range []struct {
			from string
			to   string
		}{
			{from: "backlog", to: "in_progress"},
			{from: "backlog", to: "in_review"},
			{from: "backlog", to: "todo"},
			{from: "todo", to: "in_progress"},
			{from: "in_progress", to: "in_review"},
			{from: "in_review", to: "done"},
			{from: "done", to: "in_progress"},
		} {
			t.Run(fmt.Sprintf("%s/%s-to-%s", endpoint, tc.from, tc.to), func(t *testing.T) {
				name := "ORQ41 stale agent " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createExecutionIntentIssue(t, name, tc.from)
				assignIssueDirectly(t, issueID, "agent", agentID)
				seedHistoricalTask(t, issueID, agentID)

				if got := countActiveQueueTasks(t, issueID); got != 0 {
					t.Fatalf("active queue before update = %d, want 0", got)
				}

				applyIssueUpdate(t, endpoint, issueID, map[string]any{"status": tc.to})

				if got := countActiveQueueTasks(t, issueID); got != 0 {
					t.Fatalf("status-only %s→%s with historical assignee enqueued %d active task(s), want 0",
						tc.from, tc.to, got)
				}
			})
		}
	}
}

// TestFieldOnlyUpdateWithHistoricalSquadAssigneeNeverDispatches is the squad
// mirror: a stale squad assignment must not re-dispatch its leader on a
// status-only edit.
func TestFieldOnlyUpdateWithHistoricalSquadAssigneeNeverDispatches(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		for _, tc := range []struct{ from, to string }{
			{from: "backlog", to: "in_progress"},
			{from: "backlog", to: "todo"},
			{from: "in_progress", to: "in_review"},
			{from: "in_review", to: "done"},
		} {
			t.Run(fmt.Sprintf("%s/%s-to-%s", endpoint, tc.from, tc.to), func(t *testing.T) {
				name := "ORQ41 stale squad " + executionIntentTestName(t)
				leaderID := createHandlerTestAgent(t, name+" leader", nil)
				squadID := createHandlerTestSquad(t, name, leaderID)
				issueID := createExecutionIntentIssue(t, name, tc.from)
				assignIssueDirectly(t, issueID, "squad", squadID)
				seedHistoricalTask(t, issueID, leaderID)

				if got := countActiveQueueTasks(t, issueID); got != 0 {
					t.Fatalf("active queue before update = %d, want 0", got)
				}

				applyIssueUpdate(t, endpoint, issueID, map[string]any{"status": tc.to})

				if got := countActiveQueueTasks(t, issueID); got != 0 {
					t.Fatalf("status-only %s→%s with historical squad leader enqueued %d active task(s), want 0",
						tc.from, tc.to, got)
				}
			})
		}
	}
}

// TestNonStatusFieldUpdateNeverDispatches covers the pure-metadata edits:
// title, description and priority changes are never execution intent, not even
// on a backlog issue whose assignee has never run.
func TestNonStatusFieldUpdateNeverDispatches(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		for field, updates := range map[string]map[string]any{
			"title":       {"title": "ORQ41 retitled"},
			"description": {"description": "ORQ41 evidence appended"},
			"priority":    {"priority": "low"},
		} {
			t.Run(endpoint+"/"+field, func(t *testing.T) {
				name := "ORQ41 metadata " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createExecutionIntentIssue(t, name, "backlog")
				assignIssueDirectly(t, issueID, "agent", agentID)

				applyIssueUpdate(t, endpoint, issueID, updates)

				if got := countActiveQueueTasks(t, issueID); got != 0 {
					t.Fatalf("%s-only update enqueued %d active task(s), want 0", field, got)
				}
			})
		}
	}
}

// TestExplicitAssigneeTransitionDispatchesExactlyOneTask is the positive half
// of the contract: explicit execution intent still works, and it works even
// when the target agent already has task history on the issue (a re-assignment
// is a deliberate request, unlike a status flip).
func TestExplicitAssigneeTransitionDispatchesExactlyOneTask(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		for _, withHistory := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/history=%t", endpoint, withHistory), func(t *testing.T) {
				name := "ORQ41 assign " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createExecutionIntentIssue(t, name, "in_progress")
				if withHistory {
					seedHistoricalTask(t, issueID, agentID)
				}

				applyIssueUpdate(t, endpoint, issueID, map[string]any{
					"assignee_type": "agent",
					"assignee_id":   agentID,
				})

				var queued int
				if err := testPool.QueryRow(context.Background(), `
					SELECT count(*) FROM agent_task_queue
					WHERE issue_id = $1 AND agent_id = $2 AND status = 'queued'
				`, issueID, agentID).Scan(&queued); err != nil {
					t.Fatalf("count queued tasks: %v", err)
				}
				if queued != 1 {
					t.Fatalf("explicit assignee transition enqueued %d task(s), want exactly 1", queued)
				}
			})
		}
	}
}

// TestExplicitSquadAssigneeTransitionDispatchesExactlyOneTask mirrors the
// positive case for squads.
func TestExplicitSquadAssigneeTransitionDispatchesExactlyOneTask(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		t.Run(endpoint, func(t *testing.T) {
			name := "ORQ41 squad assign " + executionIntentTestName(t)
			leaderID := createHandlerTestAgent(t, name+" leader", nil)
			squadID := createHandlerTestSquad(t, name, leaderID)
			issueID := createExecutionIntentIssue(t, name, "in_progress")

			applyIssueUpdate(t, endpoint, issueID, map[string]any{
				"assignee_type": "squad",
				"assignee_id":   squadID,
			})

			var queued int
			if err := testPool.QueryRow(context.Background(), `
				SELECT count(*) FROM agent_task_queue
				WHERE issue_id = $1 AND agent_id = $2 AND status = 'queued'
			`, issueID, leaderID).Scan(&queued); err != nil {
				t.Fatalf("count queued leader tasks: %v", err)
			}
			if queued != 1 {
				t.Fatalf("explicit squad assignment enqueued %d task(s), want exactly 1", queued)
			}
		})
	}
}

// TestFirstBacklogActivationStillDispatches protects the documented serial
// sub-task workflow: a child created in backlog and promoted when its turn
// comes is genuine execution intent, because its assignee has never been
// dispatched for that issue.
func TestFirstBacklogActivationStillDispatches(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, endpoint := range []string{"single", "batch"} {
		t.Run(endpoint+"/agent", func(t *testing.T) {
			name := "ORQ41 first activation " + executionIntentTestName(t)
			agentID := createHandlerTestAgent(t, name, nil)
			issueID := createExecutionIntentIssue(t, name, "backlog")
			assignIssueDirectly(t, issueID, "agent", agentID)

			applyIssueUpdate(t, endpoint, issueID, map[string]any{"status": "todo"})

			var queued int
			if err := testPool.QueryRow(context.Background(), `
				SELECT count(*) FROM agent_task_queue
				WHERE issue_id = $1 AND agent_id = $2 AND status = 'queued'
			`, issueID, agentID).Scan(&queued); err != nil {
				t.Fatalf("count queued tasks: %v", err)
			}
			if queued != 1 {
				t.Fatalf("first backlog activation enqueued %d task(s), want exactly 1", queued)
			}
		})

		t.Run(endpoint+"/squad", func(t *testing.T) {
			name := "ORQ41 first activation squad " + executionIntentTestName(t)
			leaderID := createHandlerTestAgent(t, name+" leader", nil)
			squadID := createHandlerTestSquad(t, name, leaderID)
			issueID := createExecutionIntentIssue(t, name, "backlog")
			assignIssueDirectly(t, issueID, "squad", squadID)

			applyIssueUpdate(t, endpoint, issueID, map[string]any{"status": "todo"})

			var queued int
			if err := testPool.QueryRow(context.Background(), `
				SELECT count(*) FROM agent_task_queue
				WHERE issue_id = $1 AND agent_id = $2 AND status = 'queued'
			`, issueID, leaderID).Scan(&queued); err != nil {
				t.Fatalf("count queued leader tasks: %v", err)
			}
			if queued != 1 {
				t.Fatalf("first backlog activation (squad) enqueued %d task(s), want exactly 1", queued)
			}
		})
	}
}

// TestRepeatedBacklogRoundTripDispatchesOnlyOnce is the loop the defect enabled:
// backlog→todo→backlog→todo on an already-dispatched assignment must not keep
// buying runs. The first activation dispatches; every later flip is bookkeeping.
func TestRepeatedBacklogRoundTripDispatchesOnlyOnce(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	name := "ORQ41 round trip " + executionIntentTestName(t)
	agentID := createHandlerTestAgent(t, name, nil)
	issueID := createExecutionIntentIssue(t, name, "backlog")
	assignIssueDirectly(t, issueID, "agent", agentID)

	applyIssueUpdate(t, "single", issueID, map[string]any{"status": "todo"})

	// Terminalize the first run so the pending-task dedup is not what makes
	// the second flip a no-op — the execution-intent gate must be.
	if _, err := testPool.Exec(context.Background(), `
		UPDATE agent_task_queue SET status = 'completed', started_at = now(), completed_at = now()
		WHERE issue_id = $1
	`, issueID); err != nil {
		t.Fatalf("terminalize first task: %v", err)
	}

	setIssueStatusDirectly(t, issueID, "backlog")
	applyIssueUpdate(t, "single", issueID, map[string]any{"status": "todo"})

	var total int
	if err := testPool.QueryRow(context.Background(), `
		SELECT count(*) FROM agent_task_queue WHERE issue_id = $1 AND agent_id = $2
	`, issueID, agentID).Scan(&total); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if total != 1 {
		t.Fatalf("backlog round trip created %d task(s) in total, want 1 (first activation only)", total)
	}
	if got := countActiveQueueTasks(t, issueID); got != 0 {
		t.Fatalf("active queue after round trip = %d, want 0", got)
	}
}
