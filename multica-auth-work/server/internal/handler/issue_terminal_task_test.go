package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedAgentManagedIssueTask(t *testing.T) (issueID, agentID, taskID string) {
	t.Helper()
	ctx := context.Background()
	name := "Terminal " + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	agentID = createHandlerTestAgent(t, name, nil)
	issueID = createTestIssue(t, name, "in_progress", "urgent")

	if _, err := testPool.Exec(ctx, `
		UPDATE issue
		SET assignee_type = 'agent', assignee_id = $2
		WHERE id = $1
	`, issueID, agentID); err != nil {
		t.Fatalf("assign issue directly: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id, status, priority, dispatched_at, started_at
		)
		VALUES ($1, $2, $3, 'running', 0, now(), now())
		RETURNING id
	`, agentID, handlerTestRuntimeID(t), issueID).Scan(&taskID); err != nil {
		t.Fatalf("seed running issue task: %v", err)
	}
	if _, err := testPool.Exec(ctx, `UPDATE agent SET status = 'working' WHERE id = $1`, agentID); err != nil {
		t.Fatalf("seed working agent status: %v", err)
	}

	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
		testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
	})
	return issueID, agentID, taskID
}

func updateIssueStatusThroughHandler(t *testing.T, endpoint, issueID, status, actorAgentID, actorTaskID string) {
	t.Helper()
	var reqBody map[string]any
	var req *http.Request
	if endpoint == "single" {
		reqBody = map[string]any{"status": status}
		req = newRequest(http.MethodPut, "/api/issues/"+issueID, reqBody)
		req = withURLParam(req, "id", issueID)
	} else {
		reqBody = map[string]any{
			"issue_ids": []string{issueID},
			"updates":   map[string]any{"status": status},
		}
		req = newRequest(http.MethodPatch, "/api/issues/batch", reqBody)
	}
	if actorAgentID != "" {
		req.Header.Set("X-Agent-ID", actorAgentID)
		req.Header.Set("X-Task-ID", actorTaskID)
	}

	w := httptest.NewRecorder()
	if endpoint == "single" {
		testHandler.UpdateIssue(w, req)
	} else {
		testHandler.BatchUpdateIssues(w, req)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("%s status transition to %s: expected 200, got %d: %s", endpoint, status, w.Code, w.Body.String())
	}
}

func TestIssueStatusTransitionTaskCancellationSemantics(t *testing.T) {
	for _, endpoint := range []string{"single", "batch"} {
		for _, tc := range []struct {
			status         string
			wantTaskStatus string
			agentManaged   bool
		}{
			{status: "done", wantTaskStatus: "running", agentManaged: true},
			{status: "in_review", wantTaskStatus: "running", agentManaged: true},
			{status: "cancelled", wantTaskStatus: "cancelled", agentManaged: false},
		} {
			t.Run(fmt.Sprintf("%s/%s", endpoint, tc.status), func(t *testing.T) {
				issueID, agentID, taskID := seedAgentManagedIssueTask(t)
				actorAgentID, actorTaskID := "", ""
				if tc.agentManaged {
					actorAgentID, actorTaskID = agentID, taskID
				}

				updateIssueStatusThroughHandler(t, endpoint, issueID, tc.status, actorAgentID, actorTaskID)

				var gotStatus string
				if err := testPool.QueryRow(context.Background(), `
					SELECT status FROM agent_task_queue WHERE id = $1
				`, taskID).Scan(&gotStatus); err != nil {
					t.Fatalf("load task after %s transition: %v", tc.status, err)
				}
				if gotStatus != tc.wantTaskStatus {
					t.Fatalf("task status after %s transition = %q, want %q", tc.status, gotStatus, tc.wantTaskStatus)
				}
			})
		}
	}
}

func TestIssueAssignmentDispatchRespectsReviewAndTerminalStatuses(t *testing.T) {
	for _, endpoint := range []string{"single", "batch"} {
		for _, tc := range []struct {
			status    string
			wantTasks int
		}{
			{status: "in_review", wantTasks: 1},
			{status: "done", wantTasks: 0},
			{status: "cancelled", wantTasks: 0},
		} {
			t.Run(fmt.Sprintf("%s/%s", endpoint, tc.status), func(t *testing.T) {
				ctx := context.Background()
				name := "Review dispatch " + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createTestIssue(t, name, tc.status, "urgent")
				t.Cleanup(func() {
					testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
					testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
				})

				var req *http.Request
				updates := map[string]any{"assignee_type": "agent", "assignee_id": agentID}
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
					t.Fatalf("%s assign agent on %s issue: expected 200, got %d: %s", endpoint, tc.status, w.Code, w.Body.String())
				}

				var taskCount int
				if err := testPool.QueryRow(ctx, `
					SELECT count(*) FROM agent_task_queue
					WHERE issue_id = $1 AND agent_id = $2
				`, issueID, agentID).Scan(&taskCount); err != nil {
					t.Fatalf("count assignment tasks: %v", err)
				}
				if taskCount != tc.wantTasks {
					t.Fatalf("tasks dispatched after assigning agent to %s issue = %d, want %d", tc.status, taskCount, tc.wantTasks)
				}
			})
		}
	}
}

func createHandlerTestSquad(t *testing.T, name, leaderAgentID string) string {
	t.Helper()
	var squadID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO squad (workspace_id, name, description, leader_id, creator_id)
		VALUES ($1, $2, '', $3, $4)
		RETURNING id
	`, testWorkspaceID, name, leaderAgentID, testUserID).Scan(&squadID); err != nil {
		t.Fatalf("create handler test squad: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM squad WHERE id = $1`, squadID)
	})
	return squadID
}

func TestIssueSquadAssignmentDispatchRespectsReviewAndTerminalStatuses(t *testing.T) {
	for _, endpoint := range []string{"single", "batch"} {
		for _, tc := range []struct {
			status    string
			wantTasks int
		}{
			{status: "in_review", wantTasks: 1},
			{status: "done", wantTasks: 0},
			{status: "cancelled", wantTasks: 0},
		} {
			t.Run(fmt.Sprintf("%s/%s", endpoint, tc.status), func(t *testing.T) {
				ctx := context.Background()
				name := "Squad review " + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
				leaderID := createHandlerTestAgent(t, name+" leader", nil)
				squadID := createHandlerTestSquad(t, name, leaderID)
				issueID := createTestIssue(t, name, tc.status, "urgent")
				t.Cleanup(func() {
					testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
					testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
				})

				updates := map[string]any{"assignee_type": "squad", "assignee_id": squadID}
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
					t.Fatalf("%s assign squad on %s issue: expected 200, got %d: %s", endpoint, tc.status, w.Code, w.Body.String())
				}

				var taskCount int
				if err := testPool.QueryRow(ctx, `
					SELECT count(*) FROM agent_task_queue
					WHERE issue_id = $1 AND agent_id = $2
				`, issueID, leaderID).Scan(&taskCount); err != nil {
					t.Fatalf("count squad leader assignment tasks: %v", err)
				}
				if taskCount != tc.wantTasks {
					t.Fatalf("squad leader tasks after assigning on %s issue = %d, want %d", tc.status, taskCount, tc.wantTasks)
				}
			})
		}
	}
}

func TestIssueCreateDispatchRespectsReviewAndTerminalStatuses(t *testing.T) {
	for _, assigneeType := range []string{"agent", "squad"} {
		for _, tc := range []struct {
			status    string
			wantTasks int
		}{
			{status: "in_review", wantTasks: 1},
			{status: "done", wantTasks: 0},
			{status: "cancelled", wantTasks: 0},
		} {
			t.Run(fmt.Sprintf("%s/%s", assigneeType, tc.status), func(t *testing.T) {
				ctx := context.Background()
				name := "Create review " + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
				leaderID := createHandlerTestAgent(t, name+" agent", nil)
				assigneeID := leaderID
				if assigneeType == "squad" {
					assigneeID = createHandlerTestSquad(t, name+" squad", leaderID)
				}

				w := httptest.NewRecorder()
				req := newRequest(http.MethodPost, "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
					"title":         name,
					"status":        tc.status,
					"priority":      "urgent",
					"assignee_type": assigneeType,
					"assignee_id":   assigneeID,
				})
				testHandler.CreateIssue(w, req)
				if w.Code != http.StatusCreated {
					t.Fatalf("create %s-assigned %s issue: expected 201, got %d: %s", assigneeType, tc.status, w.Code, w.Body.String())
				}

				var issueID string
				if err := testPool.QueryRow(ctx, `
					SELECT id FROM issue WHERE workspace_id = $1 AND title = $2
				`, testWorkspaceID, name).Scan(&issueID); err != nil {
					t.Fatalf("load created issue: %v", err)
				}
				t.Cleanup(func() {
					testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE issue_id = $1`, issueID)
					testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID)
				})

				var taskCount int
				if err := testPool.QueryRow(ctx, `
					SELECT count(*) FROM agent_task_queue
					WHERE issue_id = $1 AND agent_id = $2
				`, issueID, leaderID).Scan(&taskCount); err != nil {
					t.Fatalf("count create-dispatch tasks: %v", err)
				}
				if taskCount != tc.wantTasks {
					t.Fatalf("tasks after creating %s-assigned %s issue = %d, want %d", assigneeType, tc.status, taskCount, tc.wantTasks)
				}
			})
		}
	}
}
