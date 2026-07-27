package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The (workspace_id, name) unique constraint from migration 046 is the real
// guard against two agents answering to the same name inside one workspace.
// CreateAgent already translated its violation into 409; UpdateAgent did not,
// so a rename onto a taken name answered 500 and echoed the driver message.
// These tests pin both endpoints to the same contract.
//
// Scope note: the constraint is exact-match, so "Name" and "name " remain
// distinct rows. Closing that needs a functional unique index on
// lower(btrim(name)), which is a migration and therefore a LANE-DB follow-up,
// deliberately not attempted here.
func TestAgentDuplicateName_CreateAndUpdateReturnConflict(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	runtimeID := createClaudeProviderRuntime(t)
	t.Cleanup(func() {
		testPool.Exec(ctx,
			`DELETE FROM agent WHERE workspace_id = $1 AND name LIKE 'dupname-test-%'`,
			testWorkspaceID,
		)
	})

	create := func(t *testing.T, name string) (string, int, string) {
		t.Helper()
		body := map[string]any{
			"name":                 name,
			"runtime_id":           runtimeID,
			"visibility":           "private",
			"max_concurrent_tasks": 1,
		}
		w := httptest.NewRecorder()
		testHandler.CreateAgent(w, newRequest(http.MethodPost, "/api/agents", body))
		id := ""
		if w.Code == http.StatusCreated {
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatalf("decode created agent: %v", err)
			}
			id = created.ID
		}
		return id, w.Code, w.Body.String()
	}

	rename := func(t *testing.T, id, name string) (int, string) {
		t.Helper()
		req := withURLParam(
			newRequest(http.MethodPatch, "/api/agents/"+id, map[string]any{"name": name}),
			"id", id,
		)
		w := httptest.NewRecorder()
		testHandler.UpdateAgent(w, req)
		return w.Code, w.Body.String()
	}

	firstID, code, body := create(t, "dupname-test-first")
	if code != http.StatusCreated {
		t.Fatalf("first create must succeed, got %d: %s", code, body)
	}
	secondID, code, body := create(t, "dupname-test-second")
	if code != http.StatusCreated {
		t.Fatalf("second create must succeed, got %d: %s", code, body)
	}

	t.Run("create with a taken name conflicts", func(t *testing.T) {
		_, code, body := create(t, "dupname-test-first")
		if code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", code, body)
		}
		assertContentFreeConflict(t, body)
	})

	t.Run("rename onto a taken name conflicts instead of 500", func(t *testing.T) {
		code, body := rename(t, secondID, "dupname-test-first")
		if code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", code, body)
		}
		assertContentFreeConflict(t, body)
		if strings.Contains(body, firstID) {
			t.Errorf("conflict body must not disclose the colliding agent id: %s", body)
		}
	})

	t.Run("renaming an agent to its own name is idempotent", func(t *testing.T) {
		if code, body := rename(t, secondID, "dupname-test-second"); code != http.StatusOK {
			t.Fatalf("self-rename must succeed, got %d: %s", code, body)
		}
	})

	t.Run("rename to a free name succeeds", func(t *testing.T) {
		if code, body := rename(t, secondID, "dupname-test-renamed"); code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", code, body)
		}
	})

	t.Run("name freed by a rename becomes available again", func(t *testing.T) {
		if _, code, body := create(t, "dupname-test-second"); code != http.StatusCreated {
			t.Fatalf("expected 201 after the name was freed, got %d: %s", code, body)
		}
	})
}

// The conflict response must state the field and the scope without leaking
// driver text, constraint names or SQL.
func assertContentFreeConflict(t *testing.T, body string) {
	t.Helper()
	lowered := strings.ToLower(body)
	for _, leak := range []string{"sqlstate", "23505", "agent_workspace_name_unique", "duplicate key", "pq:", "insert into", "update agent set"} {
		if strings.Contains(lowered, leak) {
			t.Errorf("conflict body leaks %q: %s", leak, body)
		}
	}
	if !strings.Contains(lowered, "already exists in this workspace") {
		t.Errorf("conflict body must name the scope, got: %s", body)
	}
}
