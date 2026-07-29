package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ORQ-41 acceptance coverage for the documentation-only comment mode.
//
// A documentation-only write must be durable evidence and nothing else: it is
// persisted, readable through GET /comments, leaves the issue row untouched, and
// suppresses EVERY execution trigger fail-closed — issue assignee, squad leader
// and @agent mentions alike — so evidence can be attached to a preserved,
// agent-assigned card without buying a run. Each case is paired with a control
// that omits the flag, proving the assertion would catch a regression.

func countAnyTasksForIssue(t *testing.T, issueID string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM agent_task_queue WHERE issue_id = $1`, issueID).Scan(&n); err != nil {
		t.Fatalf("count tasks for issue: %v", err)
	}
	return n
}

func listCommentsForTest(t *testing.T, issueID string) []CommentResponse {
	t.Helper()
	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/api/issues/"+issueID+"/comments", nil)
	r = withURLParam(r, "id", issueID)
	testHandler.ListComments(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("ListComments: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var comments []CommentResponse
	if err := json.NewDecoder(w.Body).Decode(&comments); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	return comments
}

type issueSnapshot struct {
	Title        string
	Description  string
	Status       string
	Priority     string
	AssigneeType string
	AssigneeID   string
}

func snapshotIssue(t *testing.T, issueID string) issueSnapshot {
	t.Helper()
	var s issueSnapshot
	if err := testPool.QueryRow(context.Background(), `
		SELECT title,
		       COALESCE(description, ''),
		       status,
		       priority,
		       COALESCE(assignee_type, ''),
		       COALESCE(assignee_id::text, '')
		FROM issue WHERE id = $1
	`, issueID).Scan(&s.Title, &s.Description, &s.Status, &s.Priority, &s.AssigneeType, &s.AssigneeID); err != nil {
		t.Fatalf("snapshot issue: %v", err)
	}
	return s
}

// documentationOnlyCase describes one trigger surface: how the issue is set up
// and what content would normally wake an agent.
type documentationOnlyCase struct {
	name string
	// setup returns the issue and the agent expected to be triggered when the
	// documentation-only flag is absent.
	setup func(t *testing.T) (issueID, expectAgentID, content string)
}

func documentationOnlyCases() []documentationOnlyCase {
	return []documentationOnlyCase{
		{
			name: "issue-agent-assignee",
			setup: func(t *testing.T) (string, string, string) {
				name := "ORQ41 doc-only assignee " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createCommentTriggerPreviewIssue(t, name, "agent", agentID)
				return issueID, agentID, "Evidence: attaching acceptance output."
			},
		},
		{
			name: "issue-squad-leader",
			setup: func(t *testing.T) (string, string, string) {
				name := "ORQ41 doc-only squad " + executionIntentTestName(t)
				leaderID := createHandlerTestAgent(t, name+" leader", nil)
				squadID := createHandlerTestSquad(t, name, leaderID)
				issueID := createCommentTriggerPreviewIssue(t, name, "squad", squadID)
				return issueID, leaderID, "Evidence: attaching acceptance output."
			},
		},
		{
			name: "agent-mention",
			setup: func(t *testing.T) (string, string, string) {
				name := "ORQ41 doc-only mention " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createCommentTriggerPreviewIssue(t, name, "", "")
				content := fmt.Sprintf("Evidence for [@Reviewer](mention://agent/%s) to read later.", agentID)
				return issueID, agentID, content
			},
		},
		{
			name: "preserved-done-issue-with-agent-assignee",
			setup: func(t *testing.T) (string, string, string) {
				name := "ORQ41 doc-only preserved " + executionIntentTestName(t)
				agentID := createHandlerTestAgent(t, name, nil)
				issueID := createCommentTriggerPreviewIssue(t, name, "agent", agentID)
				setIssueStatusDirectly(t, issueID, "done")
				seedHistoricalTask(t, issueID, agentID)
				return issueID, agentID, "Evidence appended to a preserved card; no rerun wanted."
			},
		},
	}
}

// TestDocumentationOnlyCommentSuppressesAllTriggers is the core fail-closed
// assertion across every trigger source.
func TestDocumentationOnlyCommentSuppressesAllTriggers(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, tc := range documentationOnlyCases() {
		t.Run(tc.name, func(t *testing.T) {
			issueID, expectAgentID, content := tc.setup(t)
			before := snapshotIssue(t, issueID)
			activeBefore := countActiveQueueTasks(t, issueID)
			if activeBefore != 0 {
				t.Fatalf("active queue before documentation-only comment = %d, want 0", activeBefore)
			}

			commentID := postCommentForTriggerPreviewTest(t, issueID, map[string]any{
				"content":            content,
				"documentation_only": true,
			})

			if got := countActiveQueueTasks(t, issueID); got != 0 {
				t.Fatalf("active queue after documentation-only comment = %d, want 0", got)
			}
			if got := countQueuedCommentTriggerTasks(t, issueID, expectAgentID); got != 0 {
				t.Fatalf("documentation-only comment enqueued %d task(s) for the expected agent, want 0", got)
			}

			// Additive recovery: the evidence is readable through GET.
			found := false
			for _, c := range listCommentsForTest(t, issueID) {
				if c.ID == commentID {
					found = true
					if c.Content != content {
						t.Fatalf("comment content via GET = %q, want %q", c.Content, content)
					}
				}
			}
			if !found {
				t.Fatalf("documentation-only comment %s not returned by GET /comments", commentID)
			}

			// Nothing about the issue itself may be rewritten.
			if after := snapshotIssue(t, issueID); after != before {
				t.Fatalf("issue row changed by documentation-only comment:\nbefore %+v\nafter  %+v", before, after)
			}
		})
	}
}

// TestCommentWithoutDocumentationOnlyStillTriggers is the control: the same
// fixtures DO dispatch when the flag is absent, so the suppression assertions
// above are meaningful rather than vacuous.
func TestCommentWithoutDocumentationOnlyStillTriggers(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	for _, tc := range documentationOnlyCases() {
		t.Run(tc.name, func(t *testing.T) {
			issueID, expectAgentID, content := tc.setup(t)

			postCommentForTriggerPreviewTest(t, issueID, map[string]any{"content": content})

			if got := countQueuedCommentTriggerTasks(t, issueID, expectAgentID); got != 1 {
				t.Fatalf("comment without documentation_only enqueued %d task(s), want 1", got)
			}
		})
	}
}

// TestDocumentationOnlyCommentEditSuppressesTriggers covers the edit path:
// saving an edit in documentation-only mode must not re-dispatch either.
func TestDocumentationOnlyCommentEditSuppressesTriggers(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	name := "ORQ41 doc-only edit " + executionIntentTestName(t)
	agentID := createHandlerTestAgent(t, name, nil)
	issueID := createCommentTriggerPreviewIssue(t, name, "agent", agentID)
	commentID := insertMemberRootCommentForTriggerPreviewTest(t, issueID, "initial evidence")

	updateCommentForTriggerPreviewTest(t, commentID, map[string]any{
		"content":            "initial evidence, corrected",
		"documentation_only": true,
	})

	if got := countAnyTasksForIssue(t, issueID); got != 0 {
		t.Fatalf("documentation-only comment edit created %d task(s), want 0", got)
	}
}

// TestDocumentationOnlyTriggerPreviewIsEmpty keeps preview honest: the preview
// endpoint must report exactly what the write will do (nothing), so a client
// never has to enumerate suppressed agents itself.
func TestDocumentationOnlyTriggerPreviewIsEmpty(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	name := "ORQ41 doc-only preview " + executionIntentTestName(t)
	agentID := createHandlerTestAgent(t, name, nil)
	issueID := createCommentTriggerPreviewIssue(t, name, "agent", agentID)
	content := fmt.Sprintf("ping [@Other](mention://agent/%s)", agentID)

	if resp := previewCommentTriggersForTest(t, issueID, map[string]any{"content": content}); len(resp.Agents) == 0 {
		t.Fatalf("control preview returned no agents; fixture cannot detect suppression")
	}

	resp := previewCommentTriggersForTest(t, issueID, map[string]any{
		"content":            content,
		"documentation_only": true,
	})
	if len(resp.Agents) != 0 {
		t.Fatalf("documentation-only preview returned %d agent(s), want 0", len(resp.Agents))
	}
}
