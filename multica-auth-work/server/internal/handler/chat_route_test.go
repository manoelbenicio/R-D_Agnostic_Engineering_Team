package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// Chat escape-hatch routing tests (ORQ-54).
//
// Contract under test, end to end through SendChatMessage:
//   - untargeted message   → the session's own agent (the default Squad TL for
//     sessions created without agent_id)
//   - `@agent` mention     → that agent runs the turn, no TL hop
//   - unusable mention     → falls back to the session agent, send still 201
//   - the chat_session row never changes, so the hatch is per message
//
// The pure parsing rule is covered without a database in
// internal/util/mention_chat_route_test.go.

// sendChatMessageForRouting posts content to the session as the test user and
// returns the decoded send response.
func sendChatMessageForRouting(t *testing.T, sessionID, content string) SendChatMessageResponse {
	t.Helper()

	req := newRequest("POST", "/api/chat-sessions/"+sessionID+"/messages", map[string]any{
		"content": content,
	})
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.SendChatMessage(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp SendChatMessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	return resp
}

// taskAgentID reads the agent the queued task was actually assigned to — the
// authoritative routing outcome, independent of what the response reports.
func taskAgentID(t *testing.T, taskID string) string {
	t.Helper()

	var agentID string
	if err := testPool.QueryRow(context.Background(),
		`SELECT agent_id FROM agent_task_queue WHERE id = $1`, taskID).Scan(&agentID); err != nil {
		t.Fatalf("load task agent_id: %v", err)
	}
	return agentID
}

func chatSessionAgentID(t *testing.T, sessionID string) string {
	t.Helper()

	var agentID string
	if err := testPool.QueryRow(context.Background(),
		`SELECT agent_id FROM chat_session WHERE id = $1`, sessionID).Scan(&agentID); err != nil {
		t.Fatalf("load chat session agent_id: %v", err)
	}
	return agentID
}

// createDefaultSquadTLSession builds the real untargeted-chat shape: a default
// squad whose leader is the TL, and a chat session created with no agent_id so
// CreateChatSession routes it to that leader. Returns the TL agent id and the
// session id.
func createDefaultSquadTLSession(t *testing.T, tlName string) (string, string) {
	t.Helper()

	tlAgentID := createHandlerTestAgent(t, tlName, []byte("[]"))

	// One default squad per workspace for the duration of the test; the
	// fixture workspace is shared, so restore nothing and simply scope the
	// squad row to this test.
	if _, err := testPool.Exec(context.Background(),
		`DELETE FROM squad WHERE workspace_id = $1`, testWorkspaceID); err != nil {
		t.Fatalf("clear squads: %v", err)
	}
	squad, err := testHandler.Queries.CreateSquad(context.Background(), db.CreateSquadParams{
		WorkspaceID: util.MustParseUUID(testWorkspaceID),
		Name:        "Workspace Team " + t.Name(),
		LeaderID:    util.MustParseUUID(tlAgentID),
		CreatorID:   util.MustParseUUID(testUserID),
	})
	if err != nil {
		t.Fatalf("create default squad: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM squad WHERE id = $1`, uuidToString(squad.ID))
	})

	createReq := newRequest("POST", "/api/chat-sessions", map[string]any{"title": "Untargeted chat"})
	createReq.Header.Set("X-User-ID", testUserID)
	createReq.Header.Set("X-Workspace-ID", testWorkspaceID)
	createReq = withChatTestWorkspaceCtx(t, createReq)
	createW := httptest.NewRecorder()
	testHandler.CreateChatSession(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("CreateChatSession untargeted: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}
	var session ChatSessionResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.AgentID != tlAgentID {
		t.Fatalf("precondition: untargeted session should route to TL %s, got %s", tlAgentID, session.AgentID)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM chat_session WHERE id = $1`, session.ID)
	})

	return tlAgentID, session.ID
}

func agentMention(name, agentID string) string {
	return "[@" + name + "](mention://agent/" + agentID + ")"
}

// TestSendChatMessage_DirectAgentMentionRoutesPastTL is the escape hatch: the
// session belongs to the default squad's TL, but a message addressed to another
// agent must run on that agent — and the session must stay on the TL so the
// next untargeted message goes back to it.
func TestSendChatMessage_DirectAgentMentionRoutesPastTL(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	tlAgentID, sessionID := createDefaultSquadTLSession(t, "EscapeHatchTL")
	codexAgentID := createHandlerTestAgent(t, "EscapeHatchCodex", []byte("[]"))

	// 1. Untargeted message → TL.
	untargeted := sendChatMessageForRouting(t, sessionID, "please summarize the deploy state")
	if got := taskAgentID(t, untargeted.TaskID); got != tlAgentID {
		t.Fatalf("untargeted message: expected TL %s to run the turn, got %s", tlAgentID, got)
	}
	if untargeted.AgentID != tlAgentID {
		t.Fatalf("untargeted message: response agent_id = %s, want TL %s", untargeted.AgentID, tlAgentID)
	}

	// 2. Direct @agent mention → mentioned agent, no TL interception.
	direct := sendChatMessageForRouting(t, sessionID,
		agentMention("Codex", codexAgentID)+" take this one directly")
	if got := taskAgentID(t, direct.TaskID); got != codexAgentID {
		t.Fatalf("direct mention: expected mentioned agent %s to run the turn, got %s", codexAgentID, got)
	}
	if direct.AgentID != codexAgentID {
		t.Fatalf("direct mention: response agent_id = %s, want %s", direct.AgentID, codexAgentID)
	}

	// 3. The session itself is untouched: the hatch is per message.
	if got := chatSessionAgentID(t, sessionID); got != tlAgentID {
		t.Fatalf("session agent should stay on the TL %s, got %s", tlAgentID, got)
	}
	followUp := sendChatMessageForRouting(t, sessionID, "and what about the rollback tag?")
	if got := taskAgentID(t, followUp.TaskID); got != tlAgentID {
		t.Fatalf("follow-up untargeted message: expected TL %s, got %s", tlAgentID, got)
	}
}

// TestSendChatMessage_MentionShapesThatDoNotReroute pins which mention shapes
// are NOT routing targets. Squad mentions matter most: routing a squad mention
// to the squad would re-introduce the TL hop this hatch exists to bypass, and
// silently sending the turn to a squad leader the user did not address would be
// worse than ignoring the markup.
func TestSendChatMessage_MentionShapesThatDoNotReroute(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	tlAgentID, sessionID := createDefaultSquadTLSession(t, "MentionShapesTL")
	otherAgentID := createHandlerTestAgent(t, "MentionShapesOther", []byte("[]"))

	var squadID string
	if err := testPool.QueryRow(context.Background(), `
		SELECT id FROM squad WHERE workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&squadID); err != nil {
		t.Fatalf("load squad id: %v", err)
	}

	cases := []struct {
		name    string
		content string
	}{
		{"member mention", "[@Handler Test User](mention://member/" + testUserID + ") please confirm"},
		{"squad mention", "[@Workspace Team](mention://squad/" + squadID + ") pick this up"},
		{"issue mention", "context is [ORQ-54](mention://issue/" + squadID + ")"},
		{"plain text at-name", "@" + "MentionShapesOther can you take this?"},
		{"malformed agent id", "[@Broken](mention://agent/not-a-uuid) hello"},
		{"unknown agent id", "[@Ghost](mention://agent/11111111-1111-1111-1111-111111111111) hello"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := sendChatMessageForRouting(t, sessionID, tc.content)
			if got := taskAgentID(t, resp.TaskID); got != tlAgentID {
				t.Fatalf("%s: expected fallback to session agent %s, got %s", tc.name, tlAgentID, got)
			}
		})
	}

	// Sanity: the same session DOES re-route for a real agent mention, so the
	// cases above prove the shape rule and not a dead code path.
	direct := sendChatMessageForRouting(t, sessionID, agentMention("Other", otherAgentID)+" ping")
	if got := taskAgentID(t, direct.TaskID); got != otherAgentID {
		t.Fatalf("control case: expected %s, got %s", otherAgentID, got)
	}
}

// TestSendChatMessage_UnusableMentionTargetFallsBackToSessionAgent covers the
// targets that exist but must not receive the turn. Each one falls back to the
// session agent with a 201 rather than failing a message the user already sent.
func TestSendChatMessage_UnusableMentionTargetFallsBackToSessionAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	tlAgentID, sessionID := createDefaultSquadTLSession(t, "UnusableTargetTL")

	archivedID := createHandlerTestAgent(t, "UnusableArchived", []byte("[]"))
	if _, err := testPool.Exec(ctx, `UPDATE agent SET archived_at = now() WHERE id = $1`, archivedID); err != nil {
		t.Fatalf("archive agent: %v", err)
	}

	// An agent in a different workspace must never be reachable from this
	// session, even with a valid mention link — workspace isolation.
	var foreignWorkspaceID, foreignRuntimeID, foreignAgentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, description, issue_prefix)
		VALUES ('ORQ54 Foreign', 'orq54-foreign-ws', 'foreign workspace', 'FOR')
		RETURNING id
	`).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("create foreign workspace: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, foreignWorkspaceID) })
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status,
			device_info, metadata, owner_id, last_seen_at
		)
		VALUES ($1, NULL, 'ORQ54 Foreign Runtime', 'cloud', 'orq54_foreign', 'online', 'foreign', '{}'::jsonb, $2, now())
		RETURNING id
	`, foreignWorkspaceID, testUserID).Scan(&foreignRuntimeID); err != nil {
		t.Fatalf("create foreign runtime: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, 'ORQ54 Foreign Agent', '', 'cloud', '{}'::jsonb, $2, 'workspace', 1, $3)
		RETURNING id
	`, foreignWorkspaceID, foreignRuntimeID, testUserID).Scan(&foreignAgentID); err != nil {
		t.Fatalf("create foreign agent: %v", err)
	}

	// A target whose runtime is offline: the task would be queued and never
	// claimed, so the turn must go to the session agent instead of hanging.
	var offlineRuntimeID, offlineAgentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_runtime (
			workspace_id, daemon_id, name, runtime_mode, provider, status,
			device_info, metadata, owner_id, last_seen_at
		)
		VALUES ($1, NULL, 'ORQ54 Offline Runtime', 'cloud', 'orq54_offline', 'offline', 'offline', '{}'::jsonb, $2, now())
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&offlineRuntimeID); err != nil {
		t.Fatalf("create offline runtime: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent_runtime WHERE id = $1`, offlineRuntimeID) })
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id
		)
		VALUES ($1, 'ORQ54 Offline Agent', '', 'cloud', '{}'::jsonb, $2, 'workspace', 1, $3)
		RETURNING id
	`, testWorkspaceID, offlineRuntimeID, testUserID).Scan(&offlineAgentID); err != nil {
		t.Fatalf("create offline agent: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent WHERE id = $1`, offlineAgentID) })

	cases := []struct {
		name    string
		agentID string
	}{
		{"archived agent", archivedID},
		{"agent in another workspace", foreignAgentID},
		{"agent whose runtime is offline", offlineAgentID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := sendChatMessageForRouting(t, sessionID, agentMention("Target", tc.agentID)+" please run")
			if got := taskAgentID(t, resp.TaskID); got != tlAgentID {
				t.Fatalf("%s: expected fallback to session agent %s, got %s", tc.name, tlAgentID, got)
			}
			if resp.AgentID != tlAgentID {
				t.Fatalf("%s: response agent_id = %s, want %s", tc.name, resp.AgentID, tlAgentID)
			}
		})
	}
}

// TestClaimTask_MentionRoutedChatTurnDoesNotResumeSessionOwnersCLISession
// covers the read half of the resume-pointer contract. chat_session.session_id
// is the session owner's pointer, and the resume guard only compared runtimes —
// so once a second agent can run a turn in the same chat, two agents sharing
// one runtime would hand each other their CLI sessions. The owner resumes; a
// mention-routed agent must not inherit the owner's session or work dir.
func TestClaimTask_MentionRoutedChatTurnDoesNotResumeSessionOwnersCLISession(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	ownerAgentID, runtimeID, daemonID := createRuntimeGuardAgent(t, ctx)

	// A second agent on the SAME runtime — the only shape where the old
	// runtime-equality guard would have leaked the session pointer.
	var mentionedAgentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks
		)
		VALUES ($1, $2, 'local', '{}'::jsonb, $3, 'workspace', 3)
		RETURNING id
	`, testWorkspaceID, "Escape Hatch Sibling "+t.Name(), runtimeID).Scan(&mentionedAgentID); err != nil {
		t.Fatalf("setup: create sibling agent: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent WHERE id = $1`, mentionedAgentID) })

	var sessionID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO chat_session (
			workspace_id, agent_id, creator_id, title,
			session_id, work_dir, runtime_id
		)
		VALUES ($1, $2, $3, 'escape hatch resume guard', 'owner-cli-session', '/tmp/owner-workdir', $4)
		RETURNING id
	`, testWorkspaceID, ownerAgentID, testUserID, runtimeID).Scan(&sessionID); err != nil {
		t.Fatalf("setup: create chat session: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM chat_session WHERE id = $1`, sessionID) })

	// Turn routed to the mentioned agent: same runtime, different agent.
	if _, err := testPool.Exec(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, chat_session_id, status, priority)
		VALUES ($1, $2, $3, 'queued', 0)
	`, mentionedAgentID, runtimeID, sessionID); err != nil {
		t.Fatalf("setup: queue mention-routed chat task: %v", err)
	}

	task := claimTaskForRuntimeGuard(t, runtimeID, daemonID)
	if task.PriorSessionID != "" {
		t.Fatalf("mention-routed turn must not resume the session owner's CLI session, got %q", task.PriorSessionID)
	}
	if task.PriorWorkDir != "" {
		t.Fatalf("mention-routed turn must not inherit the session owner's work dir, got %q", task.PriorWorkDir)
	}
	if _, err := testPool.Exec(ctx, `
		UPDATE agent_task_queue SET status = 'completed', completed_at = now()
		WHERE chat_session_id = $1 AND status IN ('dispatched', 'running')
	`, sessionID); err != nil {
		t.Fatalf("setup: complete mention-routed task: %v", err)
	}

	// Control: the session's own agent still resumes, so the guard narrowed
	// exactly one case instead of disabling chat resume.
	if _, err := testPool.Exec(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, chat_session_id, status, priority)
		VALUES ($1, $2, $3, 'queued', 0)
	`, ownerAgentID, runtimeID, sessionID); err != nil {
		t.Fatalf("setup: queue owner chat task: %v", err)
	}
	task = claimTaskForRuntimeGuard(t, runtimeID, daemonID)
	if task.PriorSessionID != "owner-cli-session" {
		t.Fatalf("session owner should still resume: got PriorSessionID=%q", task.PriorSessionID)
	}
	if task.PriorWorkDir != "/tmp/owner-workdir" {
		t.Fatalf("session owner should still resume its work dir: got %q", task.PriorWorkDir)
	}
}

// createSiblingAgentOnRuntime adds a second agent bound to an existing runtime.
// Two agents on ONE runtime is the shape where the escape hatch's concurrency
// and session-pointer hazards are observable.
func createSiblingAgentOnRuntime(t *testing.T, ctx context.Context, runtimeID, name string) string {
	t.Helper()

	var agentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks
		)
		VALUES ($1, $2, 'local', '{}'::jsonb, $3, 'workspace', 3)
		RETURNING id
	`, testWorkspaceID, name+" "+t.Name(), runtimeID).Scan(&agentID); err != nil {
		t.Fatalf("setup: create sibling agent %q: %v", name, err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent WHERE id = $1`, agentID) })
	return agentID
}

func queueChatTask(t *testing.T, ctx context.Context, agentID, runtimeID, sessionID, status string) string {
	t.Helper()

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, chat_session_id, status, priority)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id
	`, agentID, runtimeID, sessionID, status).Scan(&taskID); err != nil {
		t.Fatalf("setup: queue chat task (%s): %v", status, err)
	}
	return taskID
}

// TestClaimAgentTask_OneInFlightTurnPerChatSessionAcrossAgents pins the
// serialization the escape hatch requires. ClaimAgentTask used to serialize
// chat turns per (agent, chat_session): fine while a session only ever ran on
// its own agent, but once a mention can route a turn elsewhere, two agents
// would claim the same session concurrently and interleave assistant messages
// and resume state in one transcript. Serialization is now per chat_session,
// whoever runs it — without over-serializing issue work, which stays per agent.
func TestClaimAgentTask_OneInFlightTurnPerChatSessionAcrossAgents(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	ownerAgentID, runtimeID, _ := createRuntimeGuardAgent(t, ctx)
	mentionedAgentID := createSiblingAgentOnRuntime(t, ctx, runtimeID, "Serialization Sibling")

	var sessionID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO chat_session (workspace_id, agent_id, creator_id, title)
		VALUES ($1, $2, $3, 'serialization guard chat')
		RETURNING id
	`, testWorkspaceID, ownerAgentID, testUserID).Scan(&sessionID); err != nil {
		t.Fatalf("setup: create chat session: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM chat_session WHERE id = $1`, sessionID) })

	// The mentioned agent's turn is already in flight.
	inFlight := queueChatTask(t, ctx, mentionedAgentID, runtimeID, sessionID, "running")

	// The session owner has a queued turn in the same session: it must NOT be
	// claimable while another agent is mid-turn there.
	queueChatTask(t, ctx, ownerAgentID, runtimeID, sessionID, "queued")
	if _, err := testHandler.Queries.ClaimAgentTask(ctx, util.MustParseUUID(ownerAgentID)); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("owner must not claim while another agent runs the same chat session, got err=%v", err)
	}

	// A different chat session is unaffected — the guard is per session, not a
	// global chat lock.
	var otherSessionID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO chat_session (workspace_id, agent_id, creator_id, title)
		VALUES ($1, $2, $3, 'serialization guard other chat')
		RETURNING id
	`, testWorkspaceID, ownerAgentID, testUserID).Scan(&otherSessionID); err != nil {
		t.Fatalf("setup: create second chat session: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM chat_session WHERE id = $1`, otherSessionID) })
	queueChatTask(t, ctx, ownerAgentID, runtimeID, otherSessionID, "queued")
	claimed, err := testHandler.Queries.ClaimAgentTask(ctx, util.MustParseUUID(ownerAgentID))
	if err != nil {
		t.Fatalf("owner should claim a task in a different chat session: %v", err)
	}
	if uuidToString(claimed.ChatSessionID) != otherSessionID {
		t.Fatalf("claimed the wrong session: got %s, want %s", uuidToString(claimed.ChatSessionID), otherSessionID)
	}

	// Once the in-flight turn finishes, the blocked turn becomes claimable.
	if _, err := testPool.Exec(ctx, `
		UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id IN ($1, $2)
	`, inFlight, uuidToString(claimed.ID)); err != nil {
		t.Fatalf("complete in-flight tasks: %v", err)
	}
	unblocked, err := testHandler.Queries.ClaimAgentTask(ctx, util.MustParseUUID(ownerAgentID))
	if err != nil {
		t.Fatalf("owner should claim after the other agent's turn completed: %v", err)
	}
	if uuidToString(unblocked.ChatSessionID) != sessionID {
		t.Fatalf("unblocked claim hit the wrong session: got %s, want %s", uuidToString(unblocked.ChatSessionID), sessionID)
	}
}

// TestClaimAgentTask_IssueWorkStaysParallelAcrossAgents guards the blast radius
// of the change above: issue tasks must still serialize per (issue, agent) so
// two agents can work one issue in parallel.
func TestClaimAgentTask_IssueWorkStaysParallelAcrossAgents(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	agentA, runtimeID, _ := createRuntimeGuardAgent(t, ctx)
	agentB := createSiblingAgentOnRuntime(t, ctx, runtimeID, "Issue Parallel Sibling")

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, description, status, priority, creator_id, creator_type, number)
		VALUES ($1, 'orq54 parallel issue', '', 'todo', 'medium', $2, 'member',
		        (SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1))
		RETURNING id
	`, testWorkspaceID, testUserID).Scan(&issueID); err != nil {
		t.Fatalf("setup: create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID) })

	for _, agentID := range []string{agentA, agentB} {
		if _, err := testPool.Exec(ctx, `
			INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority)
			VALUES ($1, $2, $3, 'queued', 0)
		`, agentID, runtimeID, issueID); err != nil {
			t.Fatalf("setup: queue issue task: %v", err)
		}
	}

	for _, agentID := range []string{agentA, agentB} {
		if _, err := testHandler.Queries.ClaimAgentTask(ctx, util.MustParseUUID(agentID)); err != nil {
			t.Fatalf("both agents must claim their own task on the same issue: %v", err)
		}
	}
}

// TestCompleteTask_MentionRoutedTurnKeepsOwnerSessionPointer covers the write
// half of the resume-pointer contract. Reading was already gated, but the
// mentioned agent's completion still wrote chat_session.session_id — so the
// owner's next turn resumed the mentioned agent's CLI session. The pointer is
// now explicitly (chat_session, agent): chat_session holds the owner's, and
// each agent's own continuity comes from its own task rows.
func TestCompleteTask_MentionRoutedTurnKeepsOwnerSessionPointer(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	ownerAgentID, runtimeID, daemonID := createRuntimeGuardAgent(t, ctx)
	mentionedAgentID := createSiblingAgentOnRuntime(t, ctx, runtimeID, "Pointer Sibling")

	var sessionID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO chat_session (
			workspace_id, agent_id, creator_id, title,
			session_id, work_dir, runtime_id
		)
		VALUES ($1, $2, $3, 'pointer guard chat', 'owner-cli-session', '/tmp/owner-workdir', $4)
		RETURNING id
	`, testWorkspaceID, ownerAgentID, testUserID, runtimeID).Scan(&sessionID); err != nil {
		t.Fatalf("setup: create chat session: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM chat_session WHERE id = $1`, sessionID) })

	// The mentioned agent runs a turn and reports its own CLI session.
	mentionedTaskID := queueChatTask(t, ctx, mentionedAgentID, runtimeID, sessionID, "running")
	if _, err := testHandler.TaskService.CompleteTask(ctx, util.MustParseUUID(mentionedTaskID),
		[]byte(`{"summary":"orq54 mention-routed reply"}`), "mentioned-cli-session", "/tmp/mentioned-workdir"); err != nil {
		t.Fatalf("complete mention-routed task: %v", err)
	}

	var pointerSession, pointerWorkDir string
	if err := testPool.QueryRow(ctx,
		`SELECT session_id, work_dir FROM chat_session WHERE id = $1`, sessionID).
		Scan(&pointerSession, &pointerWorkDir); err != nil {
		t.Fatalf("load chat session pointer: %v", err)
	}
	if pointerSession != "owner-cli-session" || pointerWorkDir != "/tmp/owner-workdir" {
		t.Fatalf("mention-routed completion overwrote the owner pointer: session_id=%q work_dir=%q",
			pointerSession, pointerWorkDir)
	}

	// The mentioned agent's session is still recorded on its own task row, so
	// its next turn in this chat resumes its own conversation.
	var taskSession string
	if err := testPool.QueryRow(ctx,
		`SELECT session_id FROM agent_task_queue WHERE id = $1`, mentionedTaskID).Scan(&taskSession); err != nil {
		t.Fatalf("load task session_id: %v", err)
	}
	if taskSession != "mentioned-cli-session" {
		t.Fatalf("mention-routed task should record its own session_id, got %q", taskSession)
	}

	queueChatTask(t, ctx, mentionedAgentID, runtimeID, sessionID, "queued")
	task := claimTaskForRuntimeGuard(t, runtimeID, daemonID)
	if task.PriorSessionID != "mentioned-cli-session" {
		t.Fatalf("mentioned agent should resume its own session, got %q", task.PriorSessionID)
	}
	if _, err := testPool.Exec(ctx, `
		UPDATE agent_task_queue SET status = 'completed', completed_at = now()
		WHERE chat_session_id = $1 AND status IN ('dispatched', 'running')
	`, sessionID); err != nil {
		t.Fatalf("complete claimed task: %v", err)
	}

	// And the owner still resumes its own, untouched session.
	queueChatTask(t, ctx, ownerAgentID, runtimeID, sessionID, "queued")
	task = claimTaskForRuntimeGuard(t, runtimeID, daemonID)
	if task.PriorSessionID != "owner-cli-session" {
		t.Fatalf("owner should resume owner-cli-session, got %q", task.PriorSessionID)
	}
	if task.PriorWorkDir != "/tmp/owner-workdir" {
		t.Fatalf("owner should resume its own work dir, got %q", task.PriorWorkDir)
	}
}
