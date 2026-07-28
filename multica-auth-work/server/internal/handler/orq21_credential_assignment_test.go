package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func createORQ21Runtime(t *testing.T, provider string) string {
	t.Helper()
	ctx := context.Background()
	var runtimeID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider,
			status, device_info, metadata, last_seen_at, visibility, owner_id
		)
		VALUES ($1, NULL, $2, 'cloud', $3, 'online', 'orq21 test', '{}'::jsonb, now(), 'private', $4)
		RETURNING id
	`, testWorkspaceID, "ORQ21 "+uuid.NewString(), provider, testUserID).Scan(&runtimeID); err != nil {
		t.Fatalf("create ORQ21 runtime: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_runtime WHERE id=$1`, runtimeID)
	})
	return runtimeID
}

func claimORQ21Task(t *testing.T, runtimeID string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := newDaemonTokenRequest(
		http.MethodPost,
		"/api/daemon/runtimes/"+runtimeID+"/tasks/claim",
		nil,
		testWorkspaceID,
		"orq21-test-daemon",
	)
	req = withURLParam(req, "runtimeId", runtimeID)
	testHandler.ClaimTaskByRuntime(w, req)
	return w
}

func TestORQ21ClaimIncludesOnlyApprovedAssignmentMetadata(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Fatal("database-backed handler harness is required")
	}
	ctx := context.Background()
	runtimeID := createORQ21Runtime(t, "kiro")
	agentID, issueID := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "ORQ21 approved")
	taskID := createDispatchedClaimFixtureTask(t, ctx, agentID, runtimeID, issueID, "120 seconds", false)

	accountID := uuid.NewString()
	homeDir := "/private/orq21/slot-139/xdg-data"
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO accounts (account_id, vendor, tenant_id, priority, home_dir, config_dir, status)
		  VALUES ($1, 'kiro', $2, 10, $3, '/private/orq21/slot-139/xdg-config', 'available')`, []any{accountID, testWorkspaceID, homeDir}},
		{`INSERT INTO approved_accounts (tenant_id, account_id, allowed, worktype_scope)
		  VALUES ($1, $2, true, 'GENERAL')`, []any{testWorkspaceID, accountID}},
		{`INSERT INTO assignments (agent_id, account_id) VALUES ($1, $2)`, []any{agentID, accountID}},
	} {
		if _, err := testPool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed approved assignment metadata: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM accounts WHERE account_id=$1`, accountID)
	})

	w := claimORQ21Task(t, runtimeID)
	if w.Code != http.StatusOK {
		t.Fatalf("claim status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Task *struct {
			ID    string `json:"id"`
			Agent struct {
				CredentialAccountHome        string `json:"credential_account_home"`
				CredentialAssignmentRequired bool   `json:"credential_assignment_required"`
			} `json:"agent"`
		} `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode claim: %v", err)
	}
	if response.Task == nil || response.Task.ID != taskID {
		t.Fatalf("claimed task=%+v, want %s", response.Task, taskID)
	}
	if response.Task.Agent.CredentialAccountHome != homeDir || !response.Task.Agent.CredentialAssignmentRequired {
		t.Fatalf("assignment metadata=%+v", response.Task.Agent)
	}
	for _, forbidden := range []string{"secret_ref", "credential_id", "token_value", "password"} {
		if jsonContainsKey(w.Body.Bytes(), forbidden) {
			t.Fatalf("claim must not contain credential key %q", forbidden)
		}
	}
}

func TestORQ21ClaimCancelsWithoutApprovedAssignment(t *testing.T) {
	if testHandler == nil || testPool == nil {
		t.Fatal("database-backed handler harness is required")
	}
	ctx := context.Background()
	runtimeID := createORQ21Runtime(t, "antigravity")
	agentID, issueID := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "ORQ21 fail closed")
	taskID := createDispatchedClaimFixtureTask(t, ctx, agentID, runtimeID, issueID, "120 seconds", false)

	w := claimORQ21Task(t, runtimeID)
	if w.Code != http.StatusConflict {
		t.Fatalf("claim status=%d body=%s", w.Code, w.Body.String())
	}
	var status string
	if err := testPool.QueryRow(ctx, `SELECT status FROM agent_task_queue WHERE id=$1`, taskID).Scan(&status); err != nil {
		t.Fatalf("read cancelled task: %v", err)
	}
	if status != "cancelled" {
		t.Fatalf("task status=%q, want cancelled", status)
	}
}

func jsonContainsKey(raw []byte, target string) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	var walk func(any) bool
	walk = func(current any) bool {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				if key == target || walk(child) {
					return true
				}
			}
		case []any:
			for _, child := range typed {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	return walk(value)
}
