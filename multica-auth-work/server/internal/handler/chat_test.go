package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// withChatTestWorkspaceCtx injects the workspace+member context that the
// real chi middleware chain would normally set. SendChatMessage (and most
// other chat handlers) read workspace ID from ctxWorkspaceID; without this
// the test harness, which calls handlers directly, gets "invalid workspace
// id" on the parseUUIDOrBadRequest call inside SendChatMessage.
func withChatTestWorkspaceCtx(t *testing.T, req *http.Request) *http.Request {
	t.Helper()
	memberRow, err := testHandler.Queries.GetMemberByUserAndWorkspace(context.Background(), db.GetMemberByUserAndWorkspaceParams{
		UserID:      util.MustParseUUID(testUserID),
		WorkspaceID: util.MustParseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("load test member row: %v", err)
	}
	return req.WithContext(middleware.SetMemberContext(req.Context(), testWorkspaceID, memberRow))
}

func TestCreateChatSession_Routing(t *testing.T) {
	assertStoredSession := func(t *testing.T, response ChatSessionResponse, wantAgentID, wantTitle string) {
		t.Helper()
		t.Cleanup(func() {
			testPool.Exec(context.Background(), `DELETE FROM chat_session WHERE id = $1`, response.ID)
		})

		if response.AgentID != wantAgentID {
			t.Errorf("response agent_id = %s, want %s", response.AgentID, wantAgentID)
		}

		var workspaceID, agentID, creatorID, title, status string
		if err := testPool.QueryRow(context.Background(), `
			SELECT workspace_id, agent_id, creator_id, title, status
			FROM chat_session
			WHERE id = $1
		`, response.ID).Scan(&workspaceID, &agentID, &creatorID, &title, &status); err != nil {
			t.Fatalf("load persisted chat session %s: %v", response.ID, err)
		}
		if workspaceID != testWorkspaceID {
			t.Errorf("persisted workspace_id = %s, want %s", workspaceID, testWorkspaceID)
		}
		if agentID != wantAgentID {
			t.Errorf("persisted agent_id = %s, want %s", agentID, wantAgentID)
		}
		if creatorID != testUserID {
			t.Errorf("persisted creator_id = %s, want %s", creatorID, testUserID)
		}
		if title != wantTitle {
			t.Errorf("persisted title = %q, want %q", title, wantTitle)
		}
		if status != "active" {
			t.Errorf("persisted status = %q, want active", status)
		}
	}

	assertRoutingFailure := func(t *testing.T, title, wantError string) {
		t.Helper()
		req := newRequest(http.MethodPost, "/api/chat/sessions", map[string]any{
			"title": title,
		})
		req = withChatTestWorkspaceCtx(t, req)
		w := httptest.NewRecorder()

		testHandler.CreateChatSession(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusServiceUnavailable, w.Body.String())
		}
		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		if response["error"] != wantError {
			t.Errorf("error = %q, want %q", response["error"], wantError)
		}

		var sessionCount int
		if err := testPool.QueryRow(context.Background(), `
			SELECT count(*)
			FROM chat_session
			WHERE workspace_id = $1 AND title = $2
		`, testWorkspaceID, title).Scan(&sessionCount); err != nil {
			t.Fatalf("count chat sessions after routing failure: %v", err)
		}
		if sessionCount != 0 {
			t.Errorf("persisted sessions after routing failure = %d, want 0", sessionCount)
		}
	}

	t.Run("explicit agent_id literal @agent bypasses squad TL", func(t *testing.T) {
		leaderID := createHandlerTestAgent(t, "Chat routing direct-case workspace TL", nil)
		directAgentID := createHandlerTestAgent(t, "codex", nil)
		if directAgentID == leaderID {
			t.Fatalf("direct agent ID unexpectedly equals squad TL ID %s", leaderID)
		}
		createHandlerTestSquad(t, "Workspace Team", leaderID)

		const title = "literal @codex direct route"
		req := newRequest(http.MethodPost, "/api/chat/sessions", map[string]any{
			"agent_id": directAgentID,
			"title":    title,
		})
		req = withChatTestWorkspaceCtx(t, req)
		w := httptest.NewRecorder()

		testHandler.CreateChatSession(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var response ChatSessionResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.AgentID == leaderID {
			t.Errorf("explicit direct route selected squad TL %s", leaderID)
		}
		assertStoredSession(t, response, directAgentID, title)
	})

	t.Run("exactly one default squad routes to its TL", func(t *testing.T) {
		leaderID := createHandlerTestAgent(t, "Chat routing sole workspace TL", nil)
		createHandlerTestSquad(t, "Workspace Team", leaderID)

		const title = "Untargeted default TL route"
		req := newRequest(http.MethodPost, "/api/chat/sessions", map[string]any{
			"title": title,
		})
		req = withChatTestWorkspaceCtx(t, req)
		w := httptest.NewRecorder()

		testHandler.CreateChatSession(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var response ChatSessionResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		assertStoredSession(t, response, leaderID, title)
	})

	t.Run("missing default squad fails closed", func(t *testing.T) {
		assertRoutingFailure(t, "Untargeted missing default route", "default chat route is not configured")
	})

	t.Run("duplicate default squads fail closed", func(t *testing.T) {
		firstLeaderID := createHandlerTestAgent(t, "Chat routing duplicate TL one", nil)
		secondLeaderID := createHandlerTestAgent(t, "Chat routing duplicate TL two", nil)
		createHandlerTestSquad(t, "Workspace Team", firstLeaderID)
		createHandlerTestSquad(t, "Workspace Team", secondLeaderID)

		assertRoutingFailure(t, "Untargeted ambiguous default route", "default chat route is ambiguous")
	})
}

// TestSendChatMessage_LinksAttachments verifies that attachments uploaded
// against a chat_session (chat_message_id NULL) are back-filled with the
// message_id when SendChatMessage receives the matching attachment_ids.
func TestSendChatMessage_LinksAttachments(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	agentID := createHandlerTestAgent(t, "ChatSendAttachAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	// 1. Upload a file against the chat session.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "send-link.png")
	part.Write([]byte("\x89PNG\r\n\x1a\nbytes"))
	writer.WriteField("chat_session_id", sessionID)
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/api/upload-file", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-ID", testUserID)
	uploadReq.Header.Set("X-Workspace-ID", testWorkspaceID)

	uploadW := httptest.NewRecorder()
	testHandler.UploadFile(uploadW, uploadReq)
	if uploadW.Code != http.StatusOK {
		t.Fatalf("upload precondition: %d %s", uploadW.Code, uploadW.Body.String())
	}
	var uploadResp AttachmentResponse
	if err := json.Unmarshal(uploadW.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	attachmentID := uploadResp.ID
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, attachmentID)
	})

	// 2. Send a chat message that references the attachment.
	sendReq := newRequest("POST", "/api/chat-sessions/"+sessionID+"/messages", map[string]any{
		"content":        "look at this ![](" + uploadResp.URL + ")",
		"attachment_ids": []string{attachmentID},
	})
	sendReq = withURLParam(sendReq, "sessionId", sessionID)
	sendReq = withChatTestWorkspaceCtx(t, sendReq)
	sendW := httptest.NewRecorder()
	testHandler.SendChatMessage(sendW, sendReq)
	if sendW.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", sendW.Code, sendW.Body.String())
	}

	var sendResp SendChatMessageResponse
	if err := json.Unmarshal(sendW.Body.Bytes(), &sendResp); err != nil {
		t.Fatalf("decode send: %v", err)
	}
	if sendResp.MessageID == "" {
		t.Fatal("expected non-empty message_id in send response")
	}
	if sendResp.TaskID == "" {
		t.Fatal("expected non-empty task_id in send response")
	}

	var messageTaskID string
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT COALESCE(task_id::text, '') FROM chat_message WHERE id = $1`,
		sendResp.MessageID,
	).Scan(&messageTaskID); err != nil {
		t.Fatalf("query chat message task id: %v", err)
	}
	if messageTaskID != sendResp.TaskID {
		t.Fatalf("chat message task_id mismatch: want %s, got %s", sendResp.TaskID, messageTaskID)
	}

	// 3. Verify the attachment row now points at the new message.
	var dbMessageID *string
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT chat_message_id::text FROM attachment WHERE id = $1`,
		attachmentID,
	).Scan(&dbMessageID); err != nil {
		t.Fatalf("query attachment: %v", err)
	}
	if dbMessageID == nil {
		t.Fatal("chat_message_id is still NULL after send")
	}
	if *dbMessageID != sendResp.MessageID {
		t.Fatalf("chat_message_id mismatch: want %s, got %s", sendResp.MessageID, *dbMessageID)
	}
}

// TestSendChatMessage_LinksUnattachedAttachments verifies the new compose
// path: upload creates a workspace-scoped unattached attachment, and chat send
// binds it to both the session and the user message.
func TestSendChatMessage_LinksUnattachedAttachments(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	agentID := createHandlerTestAgent(t, "ChatSendUnattachedAttachAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "send-unattached.png")
	part.Write([]byte("\x89PNG\r\n\x1a\nbytes"))
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/api/upload-file", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-ID", testUserID)
	uploadReq.Header.Set("X-Workspace-ID", testWorkspaceID)

	uploadW := httptest.NewRecorder()
	testHandler.UploadFile(uploadW, uploadReq)
	if uploadW.Code != http.StatusOK {
		t.Fatalf("upload precondition: %d %s", uploadW.Code, uploadW.Body.String())
	}
	var uploadResp AttachmentResponse
	if err := json.Unmarshal(uploadW.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	attachmentID := uploadResp.ID
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, attachmentID)
	})
	if uploadResp.ChatSessionID != nil {
		t.Fatalf("pre-send chat_session_id should be nil, got %v", *uploadResp.ChatSessionID)
	}
	if uploadResp.ChatMessageID != nil {
		t.Fatalf("pre-send chat_message_id should be nil, got %v", *uploadResp.ChatMessageID)
	}

	sendReq := newRequest("POST", "/api/chat-sessions/"+sessionID+"/messages", map[string]any{
		"content":        "look at this ![](" + uploadResp.MarkdownURL + ")",
		"attachment_ids": []string{attachmentID},
	})
	sendReq = withURLParam(sendReq, "sessionId", sessionID)
	sendReq = withChatTestWorkspaceCtx(t, sendReq)
	sendW := httptest.NewRecorder()
	testHandler.SendChatMessage(sendW, sendReq)
	if sendW.Code != http.StatusCreated {
		t.Fatalf("SendChatMessage: expected 201, got %d: %s", sendW.Code, sendW.Body.String())
	}

	var sendResp SendChatMessageResponse
	if err := json.Unmarshal(sendW.Body.Bytes(), &sendResp); err != nil {
		t.Fatalf("decode send: %v", err)
	}

	var dbSessionID, dbMessageID *string
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT chat_session_id::text, chat_message_id::text FROM attachment WHERE id = $1`,
		attachmentID,
	).Scan(&dbSessionID, &dbMessageID); err != nil {
		t.Fatalf("query attachment: %v", err)
	}
	if dbSessionID == nil || *dbSessionID != sessionID {
		t.Fatalf("chat_session_id mismatch: want %s, got %v", sessionID, dbSessionID)
	}
	if dbMessageID == nil || *dbMessageID != sendResp.MessageID {
		t.Fatalf("chat_message_id mismatch: want %s, got %v", sendResp.MessageID, dbMessageID)
	}
}

// TestUpdateChatSession_RenamesTitle confirms PATCH writes the new title,
// returns the updated row, and the server-side row reflects it.
func TestUpdateChatSession_RenamesTitle(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatRenameAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	req := newRequest("PATCH", "/api/chat/sessions/"+sessionID, map[string]any{
		"title": "  Renamed Session  ",
	})
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.UpdateChatSession(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChatSession: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ChatSessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if resp.Title != "Renamed Session" {
		t.Fatalf("response title: want %q, got %q", "Renamed Session", resp.Title)
	}

	var dbTitle string
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT title FROM chat_session WHERE id = $1`,
		sessionID,
	).Scan(&dbTitle); err != nil {
		t.Fatalf("query chat_session: %v", err)
	}
	if dbTitle != "Renamed Session" {
		t.Fatalf("db title: want %q, got %q", "Renamed Session", dbTitle)
	}
}

// TestUpdateChatSession_RejectsBlank refuses an empty/whitespace title with 400.
// (Untitled is a render-side fallback, not a stored value.)
func TestUpdateChatSession_RejectsBlank(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatRenameBlankAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	req := newRequest("PATCH", "/api/chat/sessions/"+sessionID, map[string]any{
		"title": "   ",
	})
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.UpdateChatSession(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateChatSession blank: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSendChatMessage_InvalidAttachmentIDs rejects malformed UUIDs in
// attachment_ids with 400 before any side effects (no message row created).
func TestSendChatMessage_InvalidAttachmentIDs(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatBadAttachAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	req := newRequest("POST", "/api/chat-sessions/"+sessionID+"/messages", map[string]any{
		"content":        "hi",
		"attachment_ids": []string{"not-a-uuid"},
	})
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.SendChatMessage(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("SendChatMessage with bad attachment id: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Confirm no message row was created.
	var count int
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM chat_message WHERE chat_session_id = $1`,
		sessionID,
	).Scan(&count); err != nil {
		t.Fatalf("count chat_message: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 chat_message rows after rejected send, got %d", count)
	}
}

func fetchChatMessagesPageForTest(t *testing.T, sessionID string, params url.Values) ChatMessagesPageResponse {
	t.Helper()
	target := "/api/chat/sessions/" + sessionID + "/messages/page"
	if encoded := params.Encode(); encoded != "" {
		target += "?" + encoded
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-User-ID", testUserID)
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.ListChatMessagesPage(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListChatMessagesPage: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var page ChatMessagesPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page messages: %v", err)
	}
	return page
}

func TestListChatMessagesPage_UsesCursorWithoutChangingLegacyList(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatCursorPaginationAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	for i, content := range []string{"oldest", "middle", "newest"} {
		_, err := testPool.Exec(
			context.Background(),
			`INSERT INTO chat_message (chat_session_id, role, content, created_at)
			 VALUES ($1, 'user', $2, timestamp '2026-01-01 00:00:00' + ($3::int * interval '1 second'))`,
			sessionID,
			content,
			i,
		)
		if err != nil {
			t.Fatalf("insert chat message %d: %v", i, err)
		}
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+sessionID+"/messages", nil)
	legacyReq.Header.Set("X-User-ID", testUserID)
	legacyReq = withURLParam(legacyReq, "sessionId", sessionID)
	legacyReq = withChatTestWorkspaceCtx(t, legacyReq)
	legacyW := httptest.NewRecorder()
	testHandler.ListChatMessages(legacyW, legacyReq)
	if legacyW.Code != http.StatusOK {
		t.Fatalf("ListChatMessages: expected 200, got %d: %s", legacyW.Code, legacyW.Body.String())
	}
	var legacy []ChatMessageResponse
	if err := json.Unmarshal(legacyW.Body.Bytes(), &legacy); err != nil {
		t.Fatalf("decode legacy messages: %v", err)
	}
	if len(legacy) != 3 || legacy[0].Content != "oldest" || legacy[2].Content != "newest" {
		t.Fatalf("legacy messages = %#v", legacy)
	}

	latest := fetchChatMessagesPageForTest(t, sessionID, url.Values{"limit": {"2"}})
	if latest.Limit != 2 || !latest.HasMore || latest.NextCursor == nil {
		t.Fatalf("latest page metadata = %#v", latest)
	}
	if len(latest.Messages) != 2 || latest.Messages[0].Content != "middle" || latest.Messages[1].Content != "newest" {
		t.Fatalf("latest page messages = %#v", latest)
	}

	older := fetchChatMessagesPageForTest(t, sessionID, url.Values{
		"limit":             {"2"},
		"before_created_at": {latest.NextCursor.CreatedAt},
		"before_id":         {latest.NextCursor.ID},
	})
	if older.HasMore || older.NextCursor != nil {
		t.Fatalf("older page metadata = %#v", older)
	}
	if len(older.Messages) != 1 || older.Messages[0].Content != "oldest" {
		t.Fatalf("older page messages = %#v", older)
	}
}

func TestListChatMessagesPage_CursorTieBreaksSameTimestampWithoutDupesOrGaps(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatCursorTieBreakAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	contents := []string{"a", "b", "c", "d", "e"}
	for _, content := range contents {
		_, err := testPool.Exec(
			context.Background(),
			`INSERT INTO chat_message (chat_session_id, role, content, created_at)
			 VALUES ($1, 'user', $2, timestamp '2026-01-01 00:00:00')`,
			sessionID,
			content,
		)
		if err != nil {
			t.Fatalf("insert chat message %q: %v", content, err)
		}
	}

	seen := map[string]bool{}
	var ordered []string
	params := url.Values{"limit": {"2"}}
	for {
		page := fetchChatMessagesPageForTest(t, sessionID, params)
		for _, msg := range page.Messages {
			if seen[msg.ID] {
				t.Fatalf("duplicate message id %s across cursor pages", msg.ID)
			}
			seen[msg.ID] = true
			ordered = append(ordered, msg.Content)
		}
		if !page.HasMore {
			if page.NextCursor != nil {
				t.Fatalf("terminal page has next cursor: %#v", page.NextCursor)
			}
			break
		}
		if page.NextCursor == nil {
			t.Fatalf("has_more page missing next cursor: %#v", page)
		}
		params = url.Values{
			"limit":             {"2"},
			"before_created_at": {page.NextCursor.CreatedAt},
			"before_id":         {page.NextCursor.ID},
		}
	}

	if len(ordered) != len(contents) {
		t.Fatalf("expected %d messages across pages, got %d: %v", len(contents), len(ordered), ordered)
	}
	// Pages are newest-window first and chronological within each page. With all
	// timestamps equal, the id tie-break must still produce a deterministic,
	// gap-free traversal.
	for _, content := range contents {
		found := false
		for _, got := range ordered {
			if got == content {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing content %q across cursor pages: %v", content, ordered)
		}
	}
}

func TestListChatMessagesPage_RejectsInvalidLimit(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ChatPaginationBadLimitAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+sessionID+"/messages/page?limit=0", nil)
	req.Header.Set("X-User-ID", testUserID)
	req = withURLParam(req, "sessionId", sessionID)
	req = withChatTestWorkspaceCtx(t, req)
	w := httptest.NewRecorder()
	testHandler.ListChatMessagesPage(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ListChatMessagesPage invalid limit: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
