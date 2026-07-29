package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/multica-ai/multica/server/internal/auth"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// createHandlerTestChatSession seeds a chat_session row owned by testUserID
// targeting the given agent and returns the session UUID. Cleanup runs after
// the test. Used by attachment / chat tests that need an existing session.
func createHandlerTestChatSession(t *testing.T, agentID string) string {
	t.Helper()

	var sessionID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO chat_session (workspace_id, agent_id, creator_id, title, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id
	`, testWorkspaceID, agentID, testUserID, "Handler Test Chat Session").Scan(&sessionID); err != nil {
		t.Fatalf("failed to create handler test chat session: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM chat_session WHERE id = $1`, sessionID)
	})
	return sessionID
}

// mockStorage is a tiny in-memory Storage stand-in. Upload records the bytes
// keyed by the storage key so GetReader can round-trip them in tests; KeyFromURL
// strips the synthetic CDN host so consumers can pass either the URL or the
// raw key.
type mockStorage struct {
	mu                  sync.Mutex
	files               map[string][]byte
	presignCalls        []string
	presignDispositions []string
}

func (m *mockStorage) Upload(_ context.Context, key string, data []byte, _ string, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[key] = append([]byte(nil), data...)
	return fmt.Sprintf("https://cdn.example.com/%s", key), nil
}

func (m *mockStorage) Delete(_ context.Context, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, key)
}
func (m *mockStorage) DeleteKeys(_ context.Context, _ []string) {}
func (m *mockStorage) KeyFromURL(rawURL string) string {
	for _, prefix := range []string{
		"https://cdn.example.com/",
		"http://rustfs:9000/test-bucket/",
		"https://s3.example.com/test-bucket/",
	} {
		if strings.HasPrefix(rawURL, prefix) {
			return strings.TrimPrefix(rawURL, prefix)
		}
	}
	return rawURL
}
func (m *mockStorage) CdnDomain() string { return "cdn.example.com" }

// mockStorageNoCdn is a mockStorage variant that returns an empty CdnDomain
// to simulate a private S3 / R2 / MinIO deployment where the operator has
// NOT configured a public-facing CDN domain. buildMarkdownURL must not
// persist `a.Url` for this shape — it would write a private bucket URL
// into markdown that no client can load.
type mockStorageNoCdn struct{ mockStorage }

func (m *mockStorageNoCdn) CdnDomain() string { return "" }
func (m *mockStorage) GetReader(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if data, ok := m.files[key]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, fmt.Errorf("mockStorage GetReader: key not found: %q", key)
}
func (m *mockStorage) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presignCalls = append(m.presignCalls, key)
	return "https://signed.example.com/" + key + "?X-Amz-Signature=mock", nil
}
func (m *mockStorage) PresignGetWithContentDisposition(_ context.Context, key string, _ time.Duration, contentDisposition string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presignCalls = append(m.presignCalls, key)
	m.presignDispositions = append(m.presignDispositions, contentDisposition)
	u := url.URL{
		Scheme: "https",
		Host:   "signed.example.com",
		Path:   "/" + key,
	}
	q := u.Query()
	q.Set("X-Amz-Signature", "mock")
	if contentDisposition != "" {
		q.Set("response-content-disposition", contentDisposition)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
func (m *mockStorage) put(key string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[key] = append([]byte(nil), data...)
}

func TestUploadFileForeignWorkspace(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("hello world"))
	writer.Close()

	foreignWorkspaceID := "00000000-0000-0000-0000-000000000099"
	req := httptest.NewRequest("POST", "/api/upload-file", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", foreignWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("UploadFile with foreign workspace: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestUploadFileResolvesWorkspaceViaSlugHeader is a regression test for the
// v2 workspace URL refactor (#1141). The frontend switched from sending
// X-Workspace-ID (UUID) to X-Workspace-Slug. For endpoints that sit outside
// the workspace middleware — like /api/upload-file — the handler-side
// resolver must accept the slug and translate it to a UUID, otherwise the
// handler silently falls through to the "no workspace context" branch and
// skips creating the DB attachment record. Files end up in S3 with no row
// in the attachment table, invisible to the UI.
func TestUploadFileResolvesWorkspaceViaSlugHeader(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "slug-upload.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("hello via slug"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload-file", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", testUserID)
	// Intentionally NOT setting X-Workspace-ID — post-v2 clients only send slug.
	req.Header.Set("X-Workspace-Slug", handlerTestWorkspaceSlug)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadFile with slug header: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The workspace-aware branch returns the full AttachmentResponse (with
	// id, workspace_id, uploader, etc.). The no-workspace-context branch
	// returns only {filename, link}. Distinguish by checking the shape.
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, w.Body.String())
	}
	if _, ok := resp["id"]; !ok {
		t.Fatalf("expected attachment response with 'id' field (DB row created); got fallback link-only response: %s", w.Body.String())
	}
	if gotWs, _ := resp["workspace_id"].(string); gotWs != testWorkspaceID {
		t.Fatalf("attachment workspace_id mismatch: want %s, got %v", testWorkspaceID, resp["workspace_id"])
	}

	// Verify the row actually exists in the database.
	var count int
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM attachment WHERE workspace_id = $1 AND filename = $2`,
		testWorkspaceID,
		"slug-upload.txt",
	).Scan(&count); err != nil {
		t.Fatalf("query attachment count: %v", err)
	}
	if count != 1 {
		t.Fatalf("attachment row count: want 1, got %d", count)
	}

	// Clean up so reruns don't accumulate rows.
	if _, err := testPool.Exec(
		context.Background(),
		`DELETE FROM attachment WHERE workspace_id = $1 AND filename = $2`,
		testWorkspaceID,
		"slug-upload.txt",
	); err != nil {
		t.Fatalf("cleanup attachment: %v", err)
	}
}

// TestUploadFileResolvesWorkspaceViaIDHeaderStill confirms the legacy path
// (CLI / daemon clients sending X-Workspace-ID as a UUID) still works after
// the refactor. Prevents a regression in the CLI/daemon compat branch.
func TestUploadFileResolvesWorkspaceViaIDHeaderStill(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "uuid-upload.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("hello via uuid"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload-file", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadFile with UUID header: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Clean up.
	if _, err := testPool.Exec(
		context.Background(),
		`DELETE FROM attachment WHERE workspace_id = $1 AND filename = $2`,
		testWorkspaceID,
		"uuid-upload.txt",
	); err != nil {
		t.Fatalf("cleanup attachment: %v", err)
	}
}

// TestUploadFile_AttachesToChatSession verifies that a multipart upload with
// a chat_session_id form field creates an attachment row linked to that chat
// session (chat_message_id remains NULL — it is back-filled on send).
func TestUploadFile_AttachesToChatSession(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	agentID := createHandlerTestAgent(t, "ChatUploadAgent", []byte("[]"))
	sessionID := createHandlerTestChatSession(t, agentID)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "chat-upload.png")
	if err != nil {
		t.Fatal(err)
	}
	// Minimal PNG signature so content-type sniffs as image/png.
	part.Write([]byte("\x89PNG\r\n\x1a\nrest-of-bytes"))
	if err := writer.WriteField("chat_session_id", sessionID); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload-file", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadFile with chat_session_id: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AttachmentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, w.Body.String())
	}
	if resp.ChatSessionID == nil || *resp.ChatSessionID != sessionID {
		t.Fatalf("chat_session_id in response: want %s, got %v", sessionID, resp.ChatSessionID)
	}
	if resp.ChatMessageID != nil {
		t.Fatalf("chat_message_id should be NULL before send, got %v", resp.ChatMessageID)
	}
	if resp.IssueID != nil || resp.CommentID != nil {
		t.Fatalf("issue_id/comment_id should be NULL for chat-only upload: %+v", resp)
	}
	if resp.URL == "" {
		t.Fatal("expected non-empty url")
	}

	// Verify the DB row directly.
	var dbSession, dbMessage *string
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT chat_session_id::text, chat_message_id::text FROM attachment WHERE id = $1`,
		resp.ID,
	).Scan(&dbSession, &dbMessage); err != nil {
		t.Fatalf("query attachment row: %v", err)
	}
	if dbSession == nil || *dbSession != sessionID {
		t.Fatalf("DB chat_session_id mismatch: want %s, got %v", sessionID, dbSession)
	}
	if dbMessage != nil {
		t.Fatalf("DB chat_message_id should be NULL, got %v", dbMessage)
	}

	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, resp.ID)
	})
}

// TestUploadFile_RejectsForeignChatSession verifies a chat_session in another
// workspace (or owned by another user) is rejected with 403/404, preventing
// cross-tenant attachment binding.
func TestUploadFile_RejectsForeignChatSession(t *testing.T) {
	origStorage := testHandler.Storage
	testHandler.Storage = &mockStorage{}
	defer func() { testHandler.Storage = origStorage }()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "evil.txt")
	part.Write([]byte("payload"))
	// Random non-existent UUID.
	writer.WriteField("chat_session_id", "00000000-0000-0000-0000-0000deadbeef")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload-file", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
		t.Fatalf("UploadFile with unknown chat_session_id: expected 4xx, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetAttachmentContent tests (preview proxy)
// ---------------------------------------------------------------------------

// seedPreviewAttachment inserts an attachment row + writes the bytes into the
// active mockStorage. Returns the new attachment id. Caller is responsible for
// installing the mockStorage on testHandler before calling.
func seedPreviewAttachment(t *testing.T, store *mockStorage, key, filename, contentType string, body []byte) string {
	t.Helper()
	// Register the body so GetReader can find it via KeyFromURL → key.
	url, err := store.Upload(context.Background(), key, body, contentType, filename)
	if err != nil {
		t.Fatalf("seed Upload: %v", err)
	}

	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO attachment (workspace_id, uploader_type, uploader_id, filename, url, content_type, size_bytes)
		VALUES ($1, 'member', $2, $3, $4, $5, $6)
		RETURNING id::text
	`, testWorkspaceID, testUserID, filename, url, contentType, len(body)).Scan(&id); err != nil {
		t.Fatalf("seed attachment row: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, id)
	})
	return id
}

func seedAttachmentURL(t *testing.T, rawURL, filename, contentType string, sizeBytes int64) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO attachment (workspace_id, uploader_type, uploader_id, filename, url, content_type, size_bytes)
		VALUES ($1, 'member', $2, $3, $4, $5, $6)
		RETURNING id::text
	`, testWorkspaceID, testUserID, filename, rawURL, contentType, sizeBytes).Scan(&id); err != nil {
		t.Fatalf("seed attachment row: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM attachment WHERE id = $1`, id)
	})
	return id
}

func newPreviewRequest(t *testing.T, attachmentID, workspaceID string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/attachments/"+attachmentID+"/content", nil)
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", workspaceID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", attachmentID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req, httptest.NewRecorder()
}

func newDownloadRequest(t *testing.T, attachmentID, workspaceID string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/attachments/"+attachmentID+"/download", nil)
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", workspaceID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", attachmentID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req, httptest.NewRecorder()
}

func newDownloadRouter() http.Handler {
	// Mirrors the production router after MUL-3130: the download
	// route is registered under Auth-only with no
	// RequireWorkspaceMember wrapper. The handler self-resolves the
	// workspace from the attachment row and enforces membership
	// internally, so a native browser <img>/<video> resource load
	// with no X-Workspace-* headers is the supported call shape.
	r := chi.NewRouter()
	r.Get("/api/attachments/{id}/download", testHandler.DownloadAttachment)
	return r
}

func testCloudFrontSigner(t *testing.T) *auth.CloudFrontSigner {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CloudFront test key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	t.Setenv("CLOUDFRONT_KEY_PAIR_ID", "KTEST")
	t.Setenv("CLOUDFRONT_DOMAIN", "static.example.test")
	t.Setenv("COOKIE_DOMAIN", ".example.test")
	t.Setenv("CLOUDFRONT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(pemBytes))
	t.Setenv("CLOUDFRONT_PRIVATE_KEY_SECRET", "")
	signer := auth.NewCloudFrontSignerFromEnv()
	if signer == nil {
		t.Fatal("expected CloudFront signer")
	}
	return signer
}

func TestAttachmentToResponse_NonCloudFrontUsesDownloadEndpoint(t *testing.T) {
	origSigner := testHandler.CFSigner
	testHandler.CFSigner = nil
	t.Cleanup(func() { testHandler.CFSigner = origSigner })

	id := seedAttachmentURL(t, "http://rustfs:9000/test-bucket/private.txt", "private.txt", "text/plain", 5)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	if resp.URL != "http://rustfs:9000/test-bucket/private.txt" {
		t.Fatalf("stored url changed: %q", resp.URL)
	}
	if resp.DownloadURL != "/api/attachments/"+id+"/download" {
		t.Fatalf("download_url = %q, want unified endpoint", resp.DownloadURL)
	}
}

func TestDownloadAttachment_CloudFrontRedirectSignsAttachmentDisposition(t *testing.T) {
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = &mockStorage{}
	testHandler.cfg.AttachmentDownloadMode = "cloudfront"
	testHandler.CFSigner = testCloudFrontSigner(t)
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	id := seedAttachmentURL(t, "https://static.example.test/downloads/cloudfront.md", "cloud front.md", "text/markdown", 10)

	req, w := newDownloadRequest(t, id, testWorkspaceID)
	testHandler.DownloadAttachment(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if got := parsed.Query().Get("response-content-disposition"); got != `attachment; filename="cloud front.md"` {
		t.Fatalf("response-content-disposition = %q", got)
	}
	if got := parsed.Query().Get("Key-Pair-Id"); got != "KTEST" {
		t.Fatalf("Key-Pair-Id = %q", got)
	}
}

func TestDownloadAttachment_BareNavigationWithWorkspaceSlugQueryPassesMiddleware(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "proxy"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	key := "downloads/bare-nav.txt"
	body := []byte("download body")
	store.put(key, body)
	id := seedAttachmentURL(t, "https://s3.example.com/test-bucket/"+key, "bare-nav.txt", "text/plain", int64(len(body)))

	req := httptest.NewRequest("GET", "/api/attachments/"+id+"/download?workspace_slug="+url.QueryEscape(handlerTestWorkspaceSlug), nil)
	req.Header.Set("X-User-ID", testUserID)
	w := httptest.NewRecorder()

	newDownloadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}
	if req.Header.Get("X-Workspace-ID") != "" || req.Header.Get("X-Workspace-Slug") != "" {
		t.Fatalf("bare navigation test must not set custom workspace headers")
	}
}

// TestDownloadAttachment_BareNavigationServesMemberWithoutWorkspaceHeaders
// is the regression test for MUL-3130: a markdown image rendered as
// `<img src="/api/attachments/<id>/download">` produces a native browser
// resource load that cannot attach X-Workspace-Slug / X-Workspace-ID
// headers. After the fix the handler self-resolves the workspace from
// the attachment row, so a bare URL succeeds for a workspace member.
func TestDownloadAttachment_BareNavigationServesMemberWithoutWorkspaceHeaders(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "proxy"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	key := "downloads/bare-nav.txt"
	body := []byte("download body")
	store.put(key, body)
	id := seedAttachmentURL(t, "https://s3.example.com/test-bucket/"+key, "bare-nav.txt", "text/plain", int64(len(body)))

	// Bare URL — no workspace_slug / workspace_id query, no
	// X-Workspace-* headers. This is what a browser <img> tag emits
	// when the markdown stores `/api/attachments/<id>/download`.
	req := httptest.NewRequest("GET", "/api/attachments/"+id+"/download", nil)
	req.Header.Set("X-User-ID", testUserID)
	w := httptest.NewRecorder()

	newDownloadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}
	if req.Header.Get("X-Workspace-ID") != "" || req.Header.Get("X-Workspace-Slug") != "" {
		t.Fatalf("bare navigation test must not set custom workspace headers")
	}
}

// TestDownloadAttachment_BareNavigationDeniesNonMemberWith404 covers the
// IDOR boundary: a stray attachment ID belonging to a workspace the
// requester is NOT a member of must return 404, not 200 (would leak
// bytes) and not 403 (would confirm the ID exists). Mirrors
// ServeLocalUpload's deny shape.
func TestDownloadAttachment_BareNavigationDeniesNonMemberWith404(t *testing.T) {
	if testPool == nil {
		t.Skip("test database not available")
	}
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "proxy"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	// Seed an attachment that lives in a workspace testUserID is NOT
	// a member of. The workspace row has to exist so the FK on
	// attachment.workspace_id resolves; we tear both down on
	// cleanup.
	ctx := context.Background()
	var foreignWorkspaceID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, description, issue_prefix)
		VALUES ('Bare-Nav Foreign', 'bare-nav-foreign', '', 'BNF')
		RETURNING id::text
	`).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("seed foreign workspace: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM workspace WHERE id = $1`, foreignWorkspaceID) })

	key := "downloads/bare-nav-foreign.txt"
	store.put(key, []byte("foreign-body"))
	var id string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO attachment (workspace_id, uploader_type, uploader_id, filename, url, content_type, size_bytes)
		VALUES ($1, 'member', $2, $3, $4, $5, $6)
		RETURNING id::text
	`, foreignWorkspaceID, testUserID, "foreign.txt", "https://s3.example.com/test-bucket/"+key, "text/plain", 12).Scan(&id); err != nil {
		t.Fatalf("seed foreign attachment: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM attachment WHERE id = $1`, id) })

	req := httptest.NewRequest("GET", "/api/attachments/"+id+"/download", nil)
	req.Header.Set("X-User-ID", testUserID)
	w := httptest.NewRecorder()

	newDownloadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for non-member; body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "foreign-body") {
		t.Fatalf("response body leaked file contents: %q", w.Body.String())
	}
}

func TestDownloadAttachment_AutoInternalEndpointProxies(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "auto"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	key := "downloads/proxy-private.txt"
	body := []byte("private object")
	store.put(key, body)
	id := seedAttachmentURL(t, "http://rustfs:9000/test-bucket/"+key, "report.txt", "text/plain", int64(len(body)))

	req, w := newDownloadRequest(t, id, testWorkspaceID)
	testHandler.DownloadAttachment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}
	if got := w.Header().Get("Location"); got != "" {
		t.Fatalf("Location should be empty for proxy download, got %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="report.txt"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if len(store.presignCalls) != 0 {
		t.Fatalf("internal endpoint should not presign, calls=%v", store.presignCalls)
	}
}

func TestDownloadAttachment_AutoPublicEndpointPresigns(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "auto"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	key := "downloads/public-private.txt"
	id := seedAttachmentURL(t, "https://s3.example.com/test-bucket/"+key, "public.txt", "text/plain", 10)

	req, w := newDownloadRequest(t, id, testWorkspaceID)
	testHandler.DownloadAttachment(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "X-Amz-Signature=mock") {
		t.Fatalf("Location = %q, want fake S3 signature", loc)
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if got := parsed.Query().Get("response-content-disposition"); got != `attachment; filename="public.txt"` {
		t.Fatalf("response-content-disposition = %q", got)
	}
	if len(store.presignCalls) != 1 || store.presignCalls[0] != key {
		t.Fatalf("presign calls = %v, want [%s]", store.presignCalls, key)
	}
	if len(store.presignDispositions) != 1 || store.presignDispositions[0] != `attachment; filename="public.txt"` {
		t.Fatalf("presign dispositions = %v", store.presignDispositions)
	}
}

func TestDownloadAttachment_ExplicitProxyStreamsPublicEndpoint(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	origCfg := testHandler.cfg
	origSigner := testHandler.CFSigner
	testHandler.Storage = store
	testHandler.cfg.AttachmentDownloadMode = "proxy"
	testHandler.CFSigner = nil
	t.Cleanup(func() {
		testHandler.Storage = origStorage
		testHandler.cfg = origCfg
		testHandler.CFSigner = origSigner
	})

	key := "downloads/forced-proxy.png"
	body := []byte("\x89PNG\r\n\x1a\nimage")
	store.put(key, body)
	id := seedAttachmentURL(t, "https://s3.example.com/test-bucket/"+key, "image.png", "image/png", int64(len(body)))

	req, w := newDownloadRequest(t, id, testWorkspaceID)
	testHandler.DownloadAttachment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.Bytes(); !bytes.Equal(got, body) {
		t.Fatalf("body mismatch: got %q want %q", got, body)
	}
	if got := w.Header().Get("Content-Disposition"); got != `inline; filename="image.png"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if len(store.presignCalls) != 0 {
		t.Fatalf("forced proxy should not presign, calls=%v", store.presignCalls)
	}
}

func TestShouldProxyAttachmentURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"http://rustfs:9000/test-bucket/file.txt", true},
		{"http://localhost:9000/test-bucket/file.txt", true},
		{"http://127.0.0.1:9000/test-bucket/file.txt", true},
		{"http://10.0.2.15/test-bucket/file.txt", true},
		{"https://minio.internal/test-bucket/file.txt", true},
		{"/uploads/workspaces/abc/file.txt", true},
		{"https://s3.example.com/test-bucket/file.txt", false},
		{"https://bucket.s3.us-east-1.amazonaws.com/file.txt", false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			if got := shouldProxyAttachmentURL(tc.raw); got != tc.want {
				t.Fatalf("shouldProxyAttachmentURL(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestGetAttachmentContent_HappyPath_Markdown(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	body := []byte("# heading\n\nbody text\n")
	id := seedPreviewAttachment(t, store, "preview-md-key.md", "preview.md", "text/markdown", body)

	req, w := newPreviewRequest(t, id, testWorkspaceID)
	testHandler.GetAttachmentContent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != string(body) {
		t.Errorf("body = %q, want %q", got, body)
	}
	if got := w.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", got)
	}
	if got := w.Header().Get("X-Original-Content-Type"); got != "text/markdown" {
		t.Errorf("X-Original-Content-Type = %q, want text/markdown", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

// Even when http.DetectContentType returned "text/plain" instead of "text/markdown"
// (a known sniffer quirk), the extension whitelist still grants access.
func TestGetAttachmentContent_AcceptsByExtensionWhenContentTypeIsGeneric(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	body := []byte("package main\n")
	id := seedPreviewAttachment(t, store, "main-go-key.go", "main.go", "application/octet-stream", body)

	req, w := newPreviewRequest(t, id, testWorkspaceID)
	testHandler.GetAttachmentContent(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAttachmentContent_Unsupported_PDF(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	id := seedPreviewAttachment(t, store, "pdf-key.pdf", "manual.pdf", "application/pdf", []byte("%PDF-1.4\n"))

	req, w := newPreviewRequest(t, id, testWorkspaceID)
	testHandler.GetAttachmentContent(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAttachmentContent_TooLarge(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	// One byte over the limit. Allocate ASCII so io.ReadAll has work to do.
	big := bytes.Repeat([]byte("a"), maxPreviewTextSize+1)
	id := seedPreviewAttachment(t, store, "huge-key.txt", "huge.txt", "text/plain", big)

	req, w := newPreviewRequest(t, id, testWorkspaceID)
	testHandler.GetAttachmentContent(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAttachmentContent_ForeignWorkspace(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	id := seedPreviewAttachment(t, store, "ws-mismatch.md", "note.md", "text/markdown", []byte("# secret\n"))

	// Same attachment id, but request comes in scoped to a different workspace.
	foreign := "00000000-0000-0000-0000-000000000099"
	req, w := newPreviewRequest(t, id, foreign)
	testHandler.GetAttachmentContent(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAttachmentContent_NotFound(t *testing.T) {
	store := &mockStorage{}
	origStorage := testHandler.Storage
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	req, w := newPreviewRequest(t, "00000000-0000-0000-0000-000000000abc", testWorkspaceID)
	testHandler.GetAttachmentContent(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// isTextPreviewable is the whitelist linkpin between the proxy and the
// client-side dispatcher. Regress against the most common content types so
// drifting one of the lists alone fails loud.
func TestIsTextPreviewable(t *testing.T) {
	t.Helper()
	cases := []struct {
		name        string
		contentType string
		filename    string
		want        bool
	}{
		{"markdown by ext", "application/octet-stream", "README.md", true},
		{"markdown by mime", "text/markdown", "README", true},
		{"plain text", "text/plain", "log.txt", true},
		{"json by mime", "application/json", "data.json", true},
		{"yaml by ext", "application/octet-stream", "config.yml", true},
		{"go source", "text/plain", "main.go", true},
		{"typescript", "application/octet-stream", "index.ts", true},
		{"html", "text/html", "page.html", true},
		{"dockerfile no ext", "application/octet-stream", "Dockerfile", true},

		{"pdf rejected", "application/pdf", "doc.pdf", false},
		{"png rejected", "image/png", "shot.png", false},
		{"video rejected", "video/mp4", "clip.mp4", false},
		{"binary fallthrough", "application/octet-stream", "blob.bin", false},
		{"docx rejected", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "report.docx", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTextPreviewable(tc.contentType, tc.filename); got != tc.want {
				t.Errorf("isTextPreviewable(%q, %q) = %v, want %v", tc.contentType, tc.filename, got, tc.want)
			}
		})
	}
}

// MUL-3192 — buildMarkdownURL must emit a durable, absolute-when-possible
// URL that loads natively in any client (web, desktop, mobile webview).
// `download_url` may be a short-lived signed URL and is unsafe to persist;
// `markdown_url` is the contract for "ok to embed in markdown body".
//
// Matrix:
//
//   - public CDN durable URL ............... reuse a.Url verbatim
//   - LocalStorage with PublicURL set ....... reuse a.Url (already absolute)
//   - CloudFront-signed mode ................ never reuse a.Url (raw S3),
//                                              prefer absolute API endpoint
//   - LocalStorage relative + PublicURL set . prefix to absolute API endpoint
//   - PublicURL unset ....................... fall back to site-relative
//                                              (web's Next rewrite handles it)
//   - signed URL (CloudFront-signed leaked
//     into a.Url somehow) ................... reject as durable, fall through
//                                              to API endpoint to avoid
//                                              re-opening MUL-3130

func TestBuildMarkdownURL_PublicCdnAbsoluteURLReusedVerbatim(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	origStorage := testHandler.Storage
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
		testHandler.Storage = origStorage
	})
	testHandler.cfg.PublicURL = "https://api.multica.test"
	testHandler.CFSigner = nil
	// mockStorage.CdnDomain() returns "cdn.example.com" — that's the
	// operator-set signal that the URL host serves content publicly
	// without per-request auth. Without this, the new gate routes
	// through the API endpoint to be safe.
	testHandler.Storage = &mockStorage{}

	id := seedAttachmentURL(t, "https://cdn.multica.test/uploads/abc.png", "abc.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	if resp.MarkdownURL != "https://cdn.multica.test/uploads/abc.png" {
		t.Fatalf("markdown_url = %q, want raw a.Url passthrough", resp.MarkdownURL)
	}
}

// MUL-3192 review must-fix 1 — `att.url` for a private S3 / R2 / MinIO
// bucket is absolute https + unsigned but is NOT publicly readable. The
// generic "absolute http(s) without signature" check would have wrongly
// persisted it; the gate now also requires `Storage.CdnDomain()` to be
// set so the operator has explicitly opted into "URLs from this storage
// load directly".
func TestBuildMarkdownURL_PrivateBucketWithoutCdnDomainRoutesThroughAPIEndpoint(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	origStorage := testHandler.Storage
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
		testHandler.Storage = origStorage
	})
	testHandler.cfg.PublicURL = "https://api.multica.test"
	testHandler.CFSigner = nil
	testHandler.Storage = &mockStorageNoCdn{}

	id := seedAttachmentURL(t, "https://prod.s3.amazonaws.com/key.png", "key.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	want := "https://api.multica.test/api/attachments/" + id + "/download"
	if resp.MarkdownURL != want {
		t.Fatalf("markdown_url = %q, want absolute API endpoint %q (private bucket without explicit CDN must not persist raw S3 URL)", resp.MarkdownURL, want)
	}
}

func TestBuildMarkdownURL_CloudFrontSignedModeNeverPersistsRawStorageURL(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
	})
	testHandler.cfg.PublicURL = "https://api.multica.test"
	testHandler.CFSigner = testCloudFrontSigner(t)

	// Raw S3 URL — private bucket, not loadable directly by clients.
	id := seedAttachmentURL(t, "https://prod.s3.amazonaws.com/key.png", "key.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	want := "https://api.multica.test/api/attachments/" + id + "/download"
	if resp.MarkdownURL != want {
		t.Fatalf("markdown_url = %q, want absolute API endpoint %q", resp.MarkdownURL, want)
	}
	// download_url is allowed to carry a TTL (CloudFront-signed); it's NOT
	// what the client persists, but it IS what the renderer uses for this
	// response. The two are intentionally distinct.
	if resp.DownloadURL == resp.MarkdownURL {
		t.Fatalf("download_url and markdown_url must differ in CloudFront-signed mode (got identical %q)", resp.DownloadURL)
	}
}

func TestBuildMarkdownURL_RelativeStorageURLPrefixedWithPublicURL(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
	})
	testHandler.cfg.PublicURL = "https://api.multica.test"
	testHandler.CFSigner = nil

	// LocalStorage without LOCAL_UPLOAD_BASE_URL stores a site-relative URL.
	id := seedAttachmentURL(t, "/uploads/abc.png", "abc.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	want := "https://api.multica.test/api/attachments/" + id + "/download"
	if resp.MarkdownURL != want {
		t.Fatalf("markdown_url = %q, want absolute API endpoint %q", resp.MarkdownURL, want)
	}
}

func TestBuildMarkdownURL_PublicURLUnsetFallsBackToSiteRelative(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
	})
	testHandler.cfg.PublicURL = ""
	testHandler.CFSigner = nil

	id := seedAttachmentURL(t, "/uploads/abc.png", "abc.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	want := "/api/attachments/" + id + "/download"
	if resp.MarkdownURL != want {
		t.Fatalf("markdown_url = %q, want site-relative fallback %q", resp.MarkdownURL, want)
	}
}

func TestBuildMarkdownURL_StripsTrailingSlashOnPublicURL(t *testing.T) {
	origPublic := testHandler.cfg.PublicURL
	origSigner := testHandler.CFSigner
	t.Cleanup(func() {
		testHandler.cfg.PublicURL = origPublic
		testHandler.CFSigner = origSigner
	})
	testHandler.cfg.PublicURL = "https://api.multica.test/"
	testHandler.CFSigner = nil

	id := seedAttachmentURL(t, "/uploads/abc.png", "abc.png", "image/png", 1)
	att, err := testHandler.Queries.GetAttachment(context.Background(), db.GetAttachmentParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(testWorkspaceID),
	})
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}

	resp := testHandler.attachmentToResponse(att)
	want := "https://api.multica.test/api/attachments/" + id + "/download"
	if resp.MarkdownURL != want {
		t.Fatalf("markdown_url = %q, want exactly one separator %q", resp.MarkdownURL, want)
	}
}

func TestIsDurablePublicURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"absolute https no signature", "https://cdn.multica.test/foo.png", true},
		{"absolute http no signature", "http://cdn.multica.test/foo.png", true},
		{"absolute with port + path", "https://cdn.example.test:8080/a/b/c.png", true},
		{"empty string", "", false},
		{"site-relative", "/uploads/abc.png", false},
		{"protocol-relative", "//cdn.example/foo.png", false},
		{"data URL", "data:image/png;base64,abc", false},
		{"blob URL", "blob:https://app/abc", false},
		{"unsupported scheme", "ftp://server/foo", false},
		{"cloudfront-signed Signature", "https://cdn.example/foo.png?Signature=abc&Key-Pair-Id=K1", false},
		{"cloudfront-signed Key-Pair-Id alone", "https://cdn.example/foo.png?Key-Pair-Id=K1", false},
		{"s3-presigned X-Amz-Signature", "https://bucket.s3/foo.png?X-Amz-Signature=abc", false},
		{"s3-presigned X-Amz-Expires alone", "https://bucket.s3/foo.png?X-Amz-Expires=900", false},
		{"plain Expires query", "https://cdn.example/foo.png?Expires=99", false},
		{"unrelated query", "https://cdn.example/foo.png?cache=1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDurablePublicURL(tc.url); got != tc.want {
				t.Errorf("isDurablePublicURL(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ORQ-26 upload contract (S1-S7)
//
// Three contract rules, all exercised against the real handler:
//   P1  insert failure  -> best-effort storage cleanup + 500
//   P2  contextless     -> id "" + url/download_url = storage link
//   P3  entity refs without a resolvable workspace -> 400 BEFORE the upload
// ---------------------------------------------------------------------------

// fileCount reports how many objects the mock currently holds. Used to assert
// that a rejection happened *before* anything was written to storage, and that
// a failed insert removed what it wrote.
func (m *mockStorage) fileCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

// mockStorageDeleteNoop keeps the object on Delete while still recording the
// attempt. Models a storage backend whose delete silently fails: cleanup is
// best-effort, so the request must still fail closed with 500.
type mockStorageDeleteNoop struct {
	mockStorage
	deleteCalls []string
}

func (m *mockStorageDeleteNoop) Delete(_ context.Context, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls = append(m.deleteCalls, key)
}

// failingCreateAttachmentDB wraps the real DBTX and fails ONLY the
// CreateAttachment statement. Everything else (membership lookup, actor
// resolution, cleanup queries) passes through to the real pool.
//
// Why not a BEFORE INSERT trigger on `attachment`: that mutates schema shared
// by the whole package, so any concurrently running test that inserts an
// attachment would fail, and a crashed test would leave the trigger installed.
// Why not a fully stubbed Queries: the membership check runs before the insert
// and would fail first, turning the expected 500 into a 403.
type failingCreateAttachmentDB struct {
	inner db.DBTX
	calls int
}

func (f *failingCreateAttachmentDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return f.inner.Exec(ctx, sql, args...)
}

func (f *failingCreateAttachmentDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return f.inner.Query(ctx, sql, args...)
}

func (f *failingCreateAttachmentDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if strings.Contains(sql, "-- name: CreateAttachment") {
		f.calls++
		return errRow{err: fmt.Errorf("orq26: forced attachment insert failure")}
	}
	return f.inner.QueryRow(ctx, sql, args...)
}

// errRow is a pgx.Row whose Scan always fails, which is how a sqlc `:one`
// query surfaces a database error.
type errRow struct{ err error }

func (e errRow) Scan(_ ...any) error { return e.err }

// withFailingAttachmentInsert points the handler at a DBTX that fails the
// attachment insert for the duration of the test, restoring the real Queries
// on cleanup. Hermetic: no DDL, no shared-table mutation, no leftover state.
func withFailingAttachmentInsert(t *testing.T) *failingCreateAttachmentDB {
	t.Helper()
	fake := &failingCreateAttachmentDB{inner: testPool}
	orig := testHandler.Queries
	testHandler.Queries = db.New(fake)
	t.Cleanup(func() { testHandler.Queries = orig })
	return fake
}

// uploadRequestBody builds a multipart body with the given filename, bytes and
// extra form fields.
func uploadRequestBody(t *testing.T, filename, content string, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write form field %s: %v", k, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

// S1 — contextless upload with no entity refs: id must be empty (no row
// exists, so /api/attachments/{id}/download would 404) and both url and
// download_url must carry the storage link.
func TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks(t *testing.T) {
	origStorage := testHandler.Storage
	store := &mockStorage{}
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	const filename = "contextless-upload.txt"
	body, contentType := uploadRequestBody(t, filename, "avatar bytes", nil)

	req := httptest.NewRequest("POST", "/api/upload-file", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-ID", testUserID)
	// No workspace slug/id header at all — this is the avatar / feedback shape.

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("contextless upload: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, w.Body.String())
	}
	if resp["id"] != "" {
		t.Fatalf("contextless upload must return an empty id (no attachment row exists), got %q", resp["id"])
	}
	if resp["url"] == "" {
		t.Fatal("contextless upload must return the storage link in url")
	}
	if resp["download_url"] != resp["url"] {
		t.Fatalf("download_url must equal the storage link: url=%q download_url=%q", resp["url"], resp["download_url"])
	}
	if resp["filename"] != filename {
		t.Fatalf("filename: want %q, got %q", filename, resp["filename"])
	}
	if store.fileCount() != 1 {
		t.Fatalf("expected exactly one stored object, got %d", store.fileCount())
	}

	var count int
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM attachment WHERE filename = $1`,
		filename,
	).Scan(&count); err != nil {
		t.Fatalf("query attachment count: %v", err)
	}
	if count != 0 {
		t.Fatalf("contextless upload must not create an attachment row, got %d", count)
	}
}

// S2/S3/S4 — an entity ref without a resolvable workspace is rejected with a
// 4xx and, critically, before anything is written to storage.
func TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload(t *testing.T) {
	for _, field := range []string{"issue_id", "comment_id", "chat_session_id"} {
		t.Run(field, func(t *testing.T) {
			origStorage := testHandler.Storage
			store := &mockStorage{}
			testHandler.Storage = store
			defer func() { testHandler.Storage = origStorage }()

			body, contentType := uploadRequestBody(t, "orphan-ref.txt", "bytes", map[string]string{
				field: "11111111-1111-1111-1111-111111111111",
			})

			req := httptest.NewRequest("POST", "/api/upload-file", body)
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("X-User-ID", testUserID)
			// No workspace header: the ref cannot be gated, so it must 4xx.

			w := httptest.NewRecorder()
			testHandler.UploadFile(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s without workspace: expected 400, got %d: %s", field, w.Code, w.Body.String())
			}
			if store.fileCount() != 0 {
				t.Fatalf("%s must be rejected pre-upload, but %d object(s) were stored", field, store.fileCount())
			}
		})
	}
}

// S5 — a failed insert must delete the object it just wrote and return 500,
// never a 200 pointing at a row-less object.
func TestUploadFile_InsertFailureCleansUpAndReturns500(t *testing.T) {
	origStorage := testHandler.Storage
	store := &mockStorage{}
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	forceInsertFailure := withFailingAttachmentInsert(t)

	body, contentType := uploadRequestBody(t, "insert-failure.txt", "bytes", nil)
	req := httptest.NewRequest("POST", "/api/upload-file", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("insert failure: expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if forceInsertFailure.calls != 1 {
		t.Fatalf("expected exactly one CreateAttachment attempt, got %d", forceInsertFailure.calls)
	}
	if store.fileCount() != 0 {
		t.Fatalf("insert failure must clean up the uploaded object, %d left behind", store.fileCount())
	}

	var count int
	if err := testPool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM attachment WHERE filename = $1`,
		"insert-failure.txt",
	).Scan(&count); err != nil {
		t.Fatalf("query attachment count: %v", err)
	}
	if count != 0 {
		t.Fatalf("no attachment row should exist after a failed insert, got %d", count)
	}
}

// S6 — cleanup is best-effort: when the delete does not actually remove the
// object, the request must still fail closed with 500 and still have tried.
func TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop(t *testing.T) {
	origStorage := testHandler.Storage
	store := &mockStorageDeleteNoop{}
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	forceInsertFailure := withFailingAttachmentInsert(t)

	body, contentType := uploadRequestBody(t, "insert-failure-noop-cleanup.txt", "bytes", nil)
	req := httptest.NewRequest("POST", "/api/upload-file", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("insert failure with failing cleanup: expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if forceInsertFailure.calls != 1 {
		t.Fatalf("expected exactly one CreateAttachment attempt, got %d", forceInsertFailure.calls)
	}
	if len(store.deleteCalls) != 1 {
		t.Fatalf("expected exactly one best-effort delete attempt, got %d", len(store.deleteCalls))
	}
	if store.fileCount() != 1 {
		t.Fatalf("noop cleanup should leave the object in place (best-effort), got %d", store.fileCount())
	}
}

// S7 — every 200 from UploadFile carries the four fields the client contract
// requires, for both success shapes.
func TestUploadFile_SuccessShapesAlwaysCarryContractFields(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		workspaceID string
		wantEmptyID bool
	}{
		{name: "workspace shape", filename: "shape-workspace.txt", workspaceID: testWorkspaceID},
		{name: "contextless shape", filename: "shape-contextless.txt", wantEmptyID: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origStorage := testHandler.Storage
			testHandler.Storage = &mockStorage{}
			defer func() { testHandler.Storage = origStorage }()

			body, contentType := uploadRequestBody(t, tt.filename, "bytes", nil)
			req := httptest.NewRequest("POST", "/api/upload-file", body)
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("X-User-ID", testUserID)
			if tt.workspaceID != "" {
				req.Header.Set("X-Workspace-ID", tt.workspaceID)
			}

			w := httptest.NewRecorder()
			testHandler.UploadFile(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d: %s", tt.name, w.Code, w.Body.String())
			}
			t.Cleanup(func() {
				testPool.Exec(context.Background(), `DELETE FROM attachment WHERE filename = $1`, tt.filename)
			})

			var resp map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v; body: %s", err, w.Body.String())
			}
			for _, field := range []string{"id", "url", "download_url", "filename"} {
				if _, ok := resp[field]; !ok {
					t.Fatalf("%s: response is missing required field %q: %s", tt.name, field, w.Body.String())
				}
			}
			gotID, _ := resp["id"].(string)
			if tt.wantEmptyID && gotID != "" {
				t.Fatalf("%s: id must be empty, got %q", tt.name, gotID)
			}
			if !tt.wantEmptyID && gotID == "" {
				t.Fatalf("%s: id must be set when an attachment row exists", tt.name)
			}
			if url, _ := resp["url"].(string); url == "" {
				t.Fatalf("%s: url must be non-empty", tt.name)
			}
			if dl, _ := resp["download_url"].(string); dl == "" {
				t.Fatalf("%s: download_url must be non-empty", tt.name)
			}
		})
	}
}

// deleteSnapshot captures what a Delete call observed AT CALL TIME. Inspecting
// the stored context after cleanupOrphanObject returns is worthless: its
// `defer cancel()` has already fired, so Err() would report context.Canceled
// no matter how the context was built. Everything asserted about the cleanup
// context must therefore be sampled inside Delete.
type deleteSnapshot struct {
	key         string
	err         error
	deadline    time.Time
	hasDeadline bool
	// observedAt is sampled in the same instant as deadline, so the remaining
	// budget can be computed from two timestamps taken inside Delete. Measuring
	// against a timestamp captured before the call is wrong: the deadline is
	// created as time.Now()+5s *during* the call, so deadline minus a pre-call
	// instant is always slightly MORE than 5s.
	observedAt time.Time
	value      any
}

// mockStorageRecordingDelete records a snapshot per Delete call. mockStorage
// discards both the key and the context, which is exactly what F1/F2 need to
// observe.
type mockStorageRecordingDelete struct {
	mockStorage
	// uploadURL, when set, is returned by Upload instead of the CDN-shaped URL.
	uploadURL string
	// collapseKeyFromURL makes KeyFromURL behave like the S3Storage fallback
	// ("everything after the last /"), which is the shape F1 must never use.
	collapseKeyFromURL bool
	snapshots          []deleteSnapshot
}

type cleanupProbeKey struct{}

func (m *mockStorageRecordingDelete) Upload(ctx context.Context, key string, data []byte, contentType string, filename string) (string, error) {
	if _, err := m.mockStorage.Upload(ctx, key, data, contentType, filename); err != nil {
		return "", err
	}
	if m.uploadURL != "" {
		return m.uploadURL, nil
	}
	return fmt.Sprintf("https://cdn.example.com/%s", key), nil
}

func (m *mockStorageRecordingDelete) KeyFromURL(rawURL string) string {
	if m.collapseKeyFromURL {
		if i := strings.LastIndex(rawURL, "/"); i >= 0 {
			return rawURL[i+1:]
		}
		return rawURL
	}
	return m.mockStorage.KeyFromURL(rawURL)
}

func (m *mockStorageRecordingDelete) Delete(ctx context.Context, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	deadline, hasDeadline := ctx.Deadline()
	m.snapshots = append(m.snapshots, deleteSnapshot{
		key:         key,
		err:         ctx.Err(),
		deadline:    deadline,
		hasDeadline: hasDeadline,
		observedAt:  time.Now(),
		value:       ctx.Value(cleanupProbeKey{}),
	})
	delete(m.files, key)
}

func (m *mockStorageRecordingDelete) deleteSnapshots() []deleteSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]deleteSnapshot(nil), m.snapshots...)
}

// F1 (unit) — the orphan cleanup must delete the EXACT key it was given.
func TestCleanupOrphanObject_DeletesExactKeyNotDerivedFromURL(t *testing.T) {
	const key = "users/11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222.png"

	store := &mockStorageRecordingDelete{}
	store.put(key, []byte("orphan"))
	h := &Handler{Storage: store}

	h.cleanupOrphanObject(context.Background(), key)

	snaps := store.deleteSnapshots()
	if len(snaps) != 1 {
		t.Fatalf("expected exactly one Delete call, got %d", len(snaps))
	}
	if snaps[0].key != key {
		t.Fatalf("cleanup must delete the exact key\n got: %q\nwant: %q", snaps[0].key, key)
	}
	// Guard the specific regression: the last path element alone must never
	// reach the backend, because it addresses the bucket root.
	if last := key[strings.LastIndex(key, "/")+1:]; snaps[0].key == last {
		t.Fatalf("cleanup deleted the bare filename %q instead of the full key", last)
	}
	if store.fileCount() != 0 {
		t.Fatalf("orphan object still present, %d left", store.fileCount())
	}
}

// F2 (unit) — CreateAttachment most often fails BECAUSE the client
// disconnected and the request context was canceled. The cleanup must still
// run: cancellation is dropped, values survive, and its own deadline applies.
// Every context property is sampled INSIDE Delete, before `defer cancel()`.
func TestCleanupOrphanObject_RunsWithCanceledRequestContext(t *testing.T) {
	const key = "workspaces/33333333-3333-3333-3333-333333333333/44444444-4444-4444-4444-444444444444.pdf"

	store := &mockStorageRecordingDelete{}
	store.put(key, []byte("orphan"))
	h := &Handler{Storage: store}

	reqCtx, cancel := context.WithCancel(
		context.WithValue(context.Background(), cleanupProbeKey{}, "request-scoped"),
	)
	cancel() // the client is already gone before cleanup starts
	if reqCtx.Err() == nil {
		t.Fatal("test setup: request context should already be canceled")
	}

	h.cleanupOrphanObject(reqCtx, key)

	snaps := store.deleteSnapshots()
	if len(snaps) != 1 {
		t.Fatalf("cleanup must still call Delete with a canceled request context, got %d calls", len(snaps))
	}
	got := snaps[0]
	if got.key != key {
		t.Fatalf("cleanup deleted %q, want %q", got.key, key)
	}
	if got.err != nil {
		t.Fatalf("cleanup context must not inherit cancellation, Err() was %v at call time", got.err)
	}
	if got.value != "request-scoped" {
		t.Fatalf("cleanup context must preserve request values, got %v", got.value)
	}
	if !got.hasDeadline {
		t.Fatal("cleanup context must carry its own deadline")
	}
	// Both timestamps come from inside Delete, so the budget is measured
	// strictly after the deadline was created and cannot exceed it.
	remaining := got.deadline.Sub(got.observedAt)
	if remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("cleanup budget out of range: %v remaining at call time", remaining)
	}
	if store.fileCount() != 0 {
		t.Fatalf("orphan object still present after cleanup, %d left", store.fileCount())
	}
}

// F1 (wiring) — proves UploadFile itself deletes the ORIGINAL key. The storage
// fake returns an upload URL that its own KeyFromURL cannot parse and collapses
// to the bare filename, reproducing the S3Storage fallback. If the handler ever
// goes back to deleting by URL, the recorded key becomes the bare filename and
// this test fails.
func TestUploadFile_InsertFailureDeletesOriginalKeyWhenKeyFromURLCollapses(t *testing.T) {
	origStorage := testHandler.Storage
	store := &mockStorageRecordingDelete{
		uploadURL:          "https://unrecognised.example.net/some/other/prefix/collapsed.txt",
		collapseKeyFromURL: true,
	}
	testHandler.Storage = store
	defer func() { testHandler.Storage = origStorage }()

	forceInsertFailure := withFailingAttachmentInsert(t)

	body, contentType := uploadRequestBody(t, "collapse-cleanup.txt", "bytes", nil)
	req := httptest.NewRequest("POST", "/api/upload-file", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-ID", testUserID)
	req.Header.Set("X-Workspace-ID", testWorkspaceID)

	w := httptest.NewRecorder()
	testHandler.UploadFile(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("insert failure: expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if forceInsertFailure.calls != 1 {
		t.Fatalf("expected exactly one CreateAttachment attempt, got %d", forceInsertFailure.calls)
	}

	snaps := store.deleteSnapshots()
	if len(snaps) != 1 {
		t.Fatalf("expected exactly one cleanup Delete, got %d", len(snaps))
	}
	deleted := snaps[0].key
	// The handler builds workspaces/<workspace>/<uuid><ext>; assert the full
	// prefixed key, not the collapsed filename the URL would have produced.
	wantPrefix := "workspaces/" + testWorkspaceID + "/"
	if !strings.HasPrefix(deleted, wantPrefix) {
		t.Fatalf("cleanup must delete the original key\n got: %q\nwant prefix: %q", deleted, wantPrefix)
	}
	if !strings.HasSuffix(deleted, ".txt") {
		t.Fatalf("cleanup key lost the upload extension: %q", deleted)
	}
	if deleted == "collapsed.txt" || !strings.Contains(deleted, "/") {
		t.Fatalf("cleanup deleted a URL-derived bare filename %q", deleted)
	}
	if store.KeyFromURL(store.uploadURL) != "collapsed.txt" {
		t.Fatalf("test setup: KeyFromURL should collapse to the bare filename, got %q",
			store.KeyFromURL(store.uploadURL))
	}
	// And the cleanup context must be usable here too.
	if snaps[0].err != nil {
		t.Fatalf("cleanup context unusable in the handler path: %v", snaps[0].err)
	}
	if !snaps[0].hasDeadline {
		t.Fatal("cleanup context must carry its own deadline in the handler path")
	}
	if remaining := snaps[0].deadline.Sub(snaps[0].observedAt); remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("cleanup budget out of range in the handler path: %v", remaining)
	}
	if store.fileCount() != 0 {
		t.Fatalf("orphan object left behind, %d present", store.fileCount())
	}
}
