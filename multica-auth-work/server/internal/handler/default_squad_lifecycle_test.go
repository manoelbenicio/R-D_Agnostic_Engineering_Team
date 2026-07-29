package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// lifecycleRequest builds a handler-level request for an arbitrary workspace.
// newRequest is hardcoded to the shared fixture workspace, and this test needs
// the workspace it just created through CreateWorkspace.
func lifecycleRequest(method, path, workspaceID string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", testUserID)
	if workspaceID != "" {
		req.Header.Set("X-Workspace-ID", workspaceID)
	}
	return req
}

// withLifecycleWorkspaceCtx injects the workspace+member context that the chi
// middleware chain would set in production. CreateChatSession reads the
// workspace from the context, not from the header.
func withLifecycleWorkspaceCtx(t *testing.T, req *http.Request, workspaceID string) *http.Request {
	t.Helper()
	memberRow, err := testHandler.Queries.GetMemberByUserAndWorkspace(context.Background(), db.GetMemberByUserAndWorkspaceParams{
		UserID:      util.MustParseUUID(testUserID),
		WorkspaceID: util.MustParseUUID(workspaceID),
	})
	if err != nil {
		t.Fatalf("load member row for workspace %s: %v", workspaceID, err)
	}
	return req.WithContext(middleware.SetMemberContext(req.Context(), workspaceID, memberRow))
}

// TestDefaultSquadLifecycle_FirstAgentEnablesUntargetedChat walks the real
// product lifecycle end to end through the handlers, with nothing about the
// squad pre-seeded:
//
//	create workspace -> first agent -> default squad -> chat with no agent_id
//
// It is the flow onboarding actually performs, and the one that stayed broken
// after CreateWorkspace stopped writing a leaderless squad: nothing created the
// squad later, so untargeted chat answered 400 forever. TestCreateChatSession_
// Routing cannot catch that because it creates the squad itself.
func TestDefaultSquadLifecycle_FirstAgentEnablesUntargetedChat(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const slug = "handler-tests-default-squad-lifecycle"

	// Workspace deletion cascades to member/agent/squad/chat_session rows.
	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1`, slug)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1`, slug)
		_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET onboarded_at = NULL WHERE id = $1`, testUserID)
	})

	// --- Step 1: create the workspace through the handler. -------------------
	wsRec := httptest.NewRecorder()
	testHandler.CreateWorkspace(wsRec, lifecycleRequest(http.MethodPost, "/api/workspaces", "", map[string]any{
		"name": "Default Squad Lifecycle",
		"slug": slug,
	}))
	if wsRec.Code != http.StatusCreated {
		t.Fatalf("CreateWorkspace: expected 201, got %d: %s", wsRec.Code, wsRec.Body.String())
	}
	var wsResp map[string]any
	if err := json.NewDecoder(wsRec.Body).Decode(&wsResp); err != nil {
		t.Fatalf("decode workspace response: %v", err)
	}
	workspaceID, _ := wsResp["id"].(string)
	if workspaceID == "" {
		t.Fatalf("CreateWorkspace returned no id: %v", wsResp)
	}

	// A fresh workspace has no squad: squad.leader_id is NOT NULL and no agent
	// exists yet to be the leader.
	if got := countSquads(t, workspaceID); got != 0 {
		t.Fatalf("new workspace: expected 0 squads, got %d", got)
	}

	// --- Step 2: untargeted chat is not routable yet. ------------------------
	preRec := httptest.NewRecorder()
	preReq := withLifecycleWorkspaceCtx(t, lifecycleRequest(http.MethodPost, "/api/chat-sessions", workspaceID, map[string]any{
		"title": "Too early",
	}), workspaceID)
	testHandler.CreateChatSession(preRec, preReq)
	if preRec.Code != http.StatusBadRequest {
		t.Fatalf("CreateChatSession before any agent: expected 400, got %d: %s", preRec.Code, preRec.Body.String())
	}

	// --- Step 3: a runtime registers for the workspace. ----------------------
	// Runtimes are created by the daemon, not by an HTTP handler, so this is
	// seeded directly — it is a precondition of agent creation, not the
	// behaviour under test.
	var runtimeID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, owner_id, last_seen_at
		)
		VALUES ($1, NULL, $2, 'cloud', $3, 'offline', $4, '{}'::jsonb, $5, now())
		RETURNING id
	`, workspaceID, "Lifecycle Runtime", "handler_test_runtime", "Lifecycle runtime", testUserID).Scan(&runtimeID); err != nil {
		t.Fatalf("seed runtime: %v", err)
	}

	// --- Step 4: create the first agent through the handler. -----------------
	leaderID := createLifecycleAgent(t, workspaceID, runtimeID, "Lifecycle Leader")

	// --- Step 5: the default squad now exists, led by that agent. ------------
	var (
		squadID    string
		squadName  string
		gotLeader  string
		squadCount int
	)
	if err := testPool.QueryRow(ctx, `
		SELECT id, name, leader_id FROM squad WHERE workspace_id = $1
	`, workspaceID).Scan(&squadID, &squadName, &gotLeader); err != nil {
		t.Fatalf("first agent did not materialize the default squad: %v", err)
	}
	if squadName != defaultSquadName {
		t.Fatalf("default squad name: expected %q, got %q", defaultSquadName, squadName)
	}
	if gotLeader != leaderID {
		t.Fatalf("default squad leader: expected the first agent %s, got %s", leaderID, gotLeader)
	}

	// The leader is also a squad member with role "leader", and the human who
	// created the agent joined as a plain member — same shape the user-driven
	// CreateSquad handler produces.
	var leaderMembers, humanMembers int
	if err := testPool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE member_type = 'agent'  AND member_id = $2 AND role = 'leader'),
			count(*) FILTER (WHERE member_type = 'member' AND member_id = $3)
		FROM squad_member WHERE squad_id = $1
	`, squadID, leaderID, testUserID).Scan(&leaderMembers, &humanMembers); err != nil {
		t.Fatalf("load squad members: %v", err)
	}
	if leaderMembers != 1 {
		t.Fatalf("expected the leader agent to be a squad member with role leader, got %d rows", leaderMembers)
	}
	if humanMembers != 1 {
		t.Fatalf("expected the creating member to be a squad member, got %d rows", humanMembers)
	}

	// --- Step 6: untargeted chat now routes to the squad leader. -------------
	chatRec := httptest.NewRecorder()
	chatReq := withLifecycleWorkspaceCtx(t, lifecycleRequest(http.MethodPost, "/api/chat-sessions", workspaceID, map[string]any{
		"title": "Default routed chat",
	}), workspaceID)
	testHandler.CreateChatSession(chatRec, chatReq)
	if chatRec.Code != http.StatusCreated {
		t.Fatalf("CreateChatSession without agent_id: expected 201, got %d: %s", chatRec.Code, chatRec.Body.String())
	}
	var chatResp ChatSessionResponse
	if err := json.NewDecoder(chatRec.Body).Decode(&chatResp); err != nil {
		t.Fatalf("decode chat session response: %v", err)
	}
	if chatResp.AgentID != leaderID {
		t.Fatalf("untargeted chat routed to %s, expected the default squad leader %s", chatResp.AgentID, leaderID)
	}

	// --- Step 7: a later agent must not add or hijack the squad. -------------
	secondID := createLifecycleAgent(t, workspaceID, runtimeID, "Lifecycle Second Agent")
	if secondID == leaderID {
		t.Fatalf("second agent reused the first agent id %s", secondID)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM squad WHERE workspace_id = $1`, workspaceID).Scan(&squadCount); err != nil {
		t.Fatalf("count squads after second agent: %v", err)
	}
	if squadCount != 1 {
		t.Fatalf("expected exactly 1 squad after the second agent, got %d", squadCount)
	}
	if err := testPool.QueryRow(ctx, `SELECT leader_id FROM squad WHERE id = $1`, squadID).Scan(&gotLeader); err != nil {
		t.Fatalf("reload squad leader: %v", err)
	}
	if gotLeader != leaderID {
		t.Fatalf("second agent changed the squad leader to %s, expected %s to stay", gotLeader, leaderID)
	}
}

// createLifecycleAgent drives the real CreateAgent handler for the given
// workspace and returns the new agent id.
func createLifecycleAgent(t *testing.T, workspaceID, runtimeID, name string) string {
	t.Helper()

	rec := httptest.NewRecorder()
	testHandler.CreateAgent(rec, lifecycleRequest(http.MethodPost, "/api/agents", workspaceID, map[string]any{
		"name":                 name,
		"description":          "lifecycle probe",
		"runtime_id":           runtimeID,
		"visibility":           "workspace",
		"max_concurrent_tasks": 1,
	}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateAgent %q: expected 201, got %d: %s", name, rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode agent response: %v", err)
	}
	agentID, _ := resp["id"].(string)
	if agentID == "" {
		t.Fatalf("CreateAgent %q returned no id: %v", name, resp)
	}
	return agentID
}

func countSquads(t *testing.T, workspaceID string) int {
	t.Helper()

	var count int
	if err := testPool.QueryRow(context.Background(), `SELECT count(*) FROM squad WHERE workspace_id = $1`, workspaceID).Scan(&count); err != nil {
		t.Fatalf("count squads: %v", err)
	}
	return count
}
