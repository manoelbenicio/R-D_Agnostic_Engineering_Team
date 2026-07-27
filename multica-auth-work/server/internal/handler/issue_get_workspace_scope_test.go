package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	workspacemiddleware "github.com/multica-ai/multica/server/internal/middleware"
)

type scopedIssueFixture struct {
	ID         string
	Identifier string
	Number     int32
	Title      string
}

func createScopedIssueFixture(t *testing.T, workspaceID, prefix, title, description string) scopedIssueFixture {
	t.Helper()
	var issue scopedIssueFixture
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO issue (
			workspace_id, title, description, status, priority,
			creator_type, creator_id, position, number
		)
		VALUES (
			$1, $2, $3, 'todo', 'medium',
			'member', $4, 0,
			(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1)
		)
		RETURNING id, number
	`, workspaceID, title, description, testUserID).Scan(&issue.ID, &issue.Number); err != nil {
		t.Fatalf("create scoped issue: %v", err)
	}
	issue.Identifier = fmt.Sprintf("%s-%d", prefix, issue.Number)
	issue.Title = title
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issue.ID)
	})
	return issue
}

func issueScopeRequest(issueRef, workspaceID string) *http.Request {
	path := "/api/issues/" + url.PathEscape(issueRef)
	if workspaceID != "" {
		path += "?workspace_id=" + url.QueryEscape(workspaceID)
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-ID", testUserID)
	return withURLParam(req, "id", issueRef)
}

func assertIssueError(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) map[string]json.RawMessage {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, status, recorder.Body.String())
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v; body=%q", err, recorder.Body.String())
	}
	var gotMessage string
	if err := json.Unmarshal(payload["error"], &gotMessage); err != nil || gotMessage != message {
		t.Fatalf("error = %q (%v), want %q", gotMessage, err, message)
	}
	for _, key := range []string{"id", "identifier", "number", "title", "description", "workspace_id"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("error response leaked %q: %s", key, recorder.Body.String())
		}
	}
	return payload
}

func assertHeadersDoNotContain(t *testing.T, headers http.Header, forbidden ...string) {
	t.Helper()
	for name, values := range headers {
		for _, value := range values {
			for _, token := range forbidden {
				if token != "" && strings.Contains(value, token) {
					t.Fatalf("header %s leaked %q in %q", name, token, value)
				}
			}
		}
	}
}

func assertIssueIdentity(t *testing.T, recorder *httptest.ResponseRecorder, want scopedIssueFixture) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var got IssueResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}
	if got.ID != want.ID || got.Identifier != want.Identifier || got.Number != want.Number {
		t.Fatalf("identity = (%q,%q,%d), want (%q,%q,%d)", got.ID, got.Identifier, got.Number, want.ID, want.Identifier, want.Number)
	}
}

func TestGetIssueWorkspaceFailClosed(t *testing.T) {
	local := createScopedIssueFixture(t, testWorkspaceID, "HAN", "ORQ-38 local issue", "local scope sentinel")

	foreignSlug := fmt.Sprintf("orq38-foreign-%d", time.Now().UnixNano())
	var foreignWorkspaceID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO workspace (name, slug, description, issue_prefix)
		VALUES ('ORQ-38 Foreign', $1, 'cross-tenant fixture', 'FRG')
		RETURNING id
	`, foreignSlug).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("create foreign workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE id = $1`, foreignWorkspaceID)
	})
	if _, err := testPool.Exec(context.Background(), `
		INSERT INTO member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')
	`, foreignWorkspaceID, testUserID); err != nil {
		t.Fatalf("add foreign membership: %v", err)
	}
	foreign := createScopedIssueFixture(t, foreignWorkspaceID, "FRG", "ORQ-38 foreign issue", "FOREIGN-NON-LEAK-SENTINEL")

	wrapped := workspacemiddleware.RequireWorkspaceMember(testHandler.Queries)(http.HandlerFunc(testHandler.GetIssue))
	run := func(issueRef, workspaceID string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		wrapped.ServeHTTP(recorder, issueScopeRequest(issueRef, workspaceID))
		return recorder
	}

	t.Run("S01 missing workspace returns 400", func(t *testing.T) {
		assertIssueError(t, run(local.ID, ""), http.StatusBadRequest, "workspace_id or workspace_slug is required")
	})

	t.Run("S02 invalid workspace returns 400", func(t *testing.T) {
		assertIssueError(t, run(local.ID, "not-a-uuid"), http.StatusBadRequest, "invalid workspace_id")
	})

	var foreignNotFoundBody string
	var foreignNotFoundHeaders http.Header
	t.Run("S03 foreign issue under local scope returns non-leaking 404", func(t *testing.T) {
		recorder := run(foreign.ID, testWorkspaceID)
		assertIssueError(t, recorder, http.StatusNotFound, "issue not found")
		foreignNotFoundBody = strings.TrimSpace(recorder.Body.String())
		foreignNotFoundHeaders = recorder.Header().Clone()
		if strings.Contains(recorder.Body.String(), foreign.Title) || strings.Contains(recorder.Body.String(), "FOREIGN-NON-LEAK-SENTINEL") {
			t.Fatalf("foreign response leaked issue data: %s", recorder.Body.String())
		}
		assertHeadersDoNotContain(t, recorder.Header(), foreign.ID, foreign.Identifier, foreign.Title, foreignWorkspaceID, "FOREIGN-NON-LEAK-SENTINEL")
	})

	t.Run("S04 foreign issue succeeds in its own scope", func(t *testing.T) {
		assertIssueIdentity(t, run(foreign.ID, foreignWorkspaceID), foreign)
	})

	t.Run("S05 unknown UUID has same shape as foreign issue", func(t *testing.T) {
		const unknownID = "00000000-0000-0000-0000-000000000038"
		recorder := run(unknownID, testWorkspaceID)
		assertIssueError(t, recorder, http.StatusNotFound, "issue not found")
		if got := strings.TrimSpace(recorder.Body.String()); got != foreignNotFoundBody {
			t.Fatalf("unknown body %q differs from foreign body %q", got, foreignNotFoundBody)
		}
		assertHeadersDoNotContain(t, recorder.Header(), unknownID, foreign.ID, foreign.Identifier, foreign.Title, foreignWorkspaceID, "FOREIGN-NON-LEAK-SENTINEL")
		if !reflect.DeepEqual(recorder.Header(), foreignNotFoundHeaders) {
			t.Fatalf("unknown headers %#v differ from foreign headers %#v", recorder.Header(), foreignNotFoundHeaders)
		}
	})

	t.Run("S06 local UUID succeeds", func(t *testing.T) {
		assertIssueIdentity(t, run(local.ID, testWorkspaceID), local)
	})

	t.Run("S07 identifier without workspace returns 400", func(t *testing.T) {
		assertIssueError(t, run(local.Identifier, ""), http.StatusBadRequest, "workspace_id or workspace_slug is required")
	})

	t.Run("S08 identifier succeeds in workspace", func(t *testing.T) {
		assertIssueIdentity(t, run(local.Identifier, testWorkspaceID), local)
	})
}
