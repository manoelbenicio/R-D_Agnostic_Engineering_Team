package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// withCapturedLogs swaps the default slog logger for one that writes to buf,
// then restores it on cleanup. Returns the buffer so tests can inspect what
// RequestLogger emitted.
//
// Uses a shared mutex because t.Parallel tests would otherwise race on the
// global slog.Default — tests in this file intentionally do NOT run in
// parallel for that reason.
var defaultLoggerMu sync.Mutex

func withCapturedLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	defaultLoggerMu.Lock()
	buf := &bytes.Buffer{}
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() {
		slog.SetDefault(orig)
		defaultLoggerMu.Unlock()
	})
	return buf
}

func runRequestLogger(t *testing.T, status int, body string) *bytes.Buffer {
	t.Helper()
	logs := withCapturedLogs(t)
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/daemon/heartbeat", nil).
		WithContext(context.Background())
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return logs
}

// requireLogLevel asserts that the captured output contains exactly the
// expected slog level prefix and not any of the disallowed ones.
func requireLogLevel(t *testing.T, logs *bytes.Buffer, want string, disallowed ...string) {
	t.Helper()
	out := logs.String()
	if !strings.Contains(out, "level="+want) {
		t.Fatalf("expected level=%s in logs, got:\n%s", want, out)
	}
	for _, dis := range disallowed {
		if strings.Contains(out, "level="+dis) {
			t.Fatalf("did not expect level=%s in logs, got:\n%s", dis, out)
		}
	}
}

func TestRequestLogger_RuntimeNotFound404DowngradesToInfo(t *testing.T) {
	// The whole reason this middleware change exists: a flood of WRN lines
	// after a runtime is deleted (issue #2391). The daemon catches the same
	// body and self-heals, so the line is signal-not-noise.
	logs := runRequestLogger(t, http.StatusNotFound, `{"error":"runtime not found"}`)
	requireLogLevel(t, logs, "INFO", "WARN", "ERROR")
}

func TestRequestLogger_TaskNotFound404DowngradesToInfo(t *testing.T) {
	logs := runRequestLogger(t, http.StatusNotFound, `{"error":"task not found"}`)
	requireLogLevel(t, logs, "INFO", "WARN", "ERROR")
}

func TestRequestLogger_GenericNotFound404KeepsWarn(t *testing.T) {
	// A 404 with an unfamiliar body is still a real 404 — most likely a
	// daemon hitting a wrong path, which is what Warn is for. We do NOT
	// want to downgrade these blindly.
	logs := runRequestLogger(t, http.StatusNotFound, `{"error":"not found"}`)
	requireLogLevel(t, logs, "WARN", "INFO", "ERROR")
}

func TestRequestLogger_400StaysWarn(t *testing.T) {
	logs := runRequestLogger(t, http.StatusBadRequest, `{"error":"bad input"}`)
	requireLogLevel(t, logs, "WARN", "INFO", "ERROR")
}

func TestRequestLogger_500StaysError(t *testing.T) {
	logs := runRequestLogger(t, http.StatusInternalServerError, `{"error":"boom"}`)
	requireLogLevel(t, logs, "ERROR", "WARN", "INFO")
}

func TestRequestLogger_200StaysInfo(t *testing.T) {
	logs := runRequestLogger(t, http.StatusOK, `{"ok":true}`)
	requireLogLevel(t, logs, "INFO", "WARN", "ERROR")
}

func TestRequestLogger_HealthEndpointIsSkipped(t *testing.T) {
	logs := withCapturedLogs(t)
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if logs.Len() != 0 {
		t.Fatalf("/health should not be logged, got:\n%s", logs.String())
	}
}

func TestRequestLogger_BodyStillReachesClient(t *testing.T) {
	// The body capture is implemented via Tee, which must mirror writes
	// rather than swallow them. Regress-protect: assert the response writer
	// still gets the full body.
	rec := httptest.NewRecorder()
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"runtime not found"}`))
	}))
	_ = withCapturedLogs(t)
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/daemon/heartbeat", nil))
	if got := rec.Body.String(); got != `{"error":"runtime not found"}` {
		t.Fatalf("response body lost or mutated: got %q", got)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestRequestLogger_LargeBodyBeyondCaptureLimit(t *testing.T) {
	// If the soft-404 marker only appears beyond the capture limit we
	// intentionally keep Warn — capturing arbitrary-size bodies is the
	// memory blowup we are guarding against. This test pins that
	// trade-off.
	prefix := strings.Repeat("x", softNotFoundBodyCaptureLimit+8)
	logs := runRequestLogger(t, http.StatusNotFound, prefix+`{"error":"runtime not found"}`)
	requireLogLevel(t, logs, "WARN", "INFO", "ERROR")
}

func TestRedactWebhookPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"/api/webhooks/autopilots/awt_secret", "/api/webhooks/autopilots/[redacted]"},
		{"/api/webhooks/autopilots/awt_secret/", "/api/webhooks/autopilots/[redacted]/"},
		{"/api/webhooks/autopilots/", "/api/webhooks/autopilots/"},
		{"/api/webhooks/github", "/api/webhooks/github"},
		{"/api/runtimes/abc", "/api/runtimes/abc"},
		{"/", "/"},
	}
	for _, tc := range cases {
		if got := redactWebhookPath(tc.in); got != tc.want {
			t.Errorf("redactWebhookPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRequestLogger_RedactsWebhookTokenInPath(t *testing.T) {
	logs := withCapturedLogs(t)
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/autopilots/awt_supersecret", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	out := logs.String()
	if strings.Contains(out, "awt_supersecret") {
		t.Fatalf("token leaked into logs:\n%s", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatalf("expected [redacted] in logs:\n%s", out)
	}
}

func TestRequestLogger_IncludesWebhookTriggerIDFromContext(t *testing.T) {
	// Exercise the real production flow: the webhook handler resolves the
	// trigger, then calls SetWebhookTriggerID(r, ...) which mutates *r in
	// place. After the handler returns, the wrapping RequestLogger
	// middleware reads the stashed ID off the (now-updated) request
	// context. If SetWebhookTriggerID didn't mutate in place, the
	// middleware would see the old context and the trigger ID would
	// silently drop from the audit line.
	logs := withCapturedLogs(t)
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetWebhookTriggerID(r, "trigger-abc")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/autopilots/awt_supersecret", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	out := logs.String()
	if !strings.Contains(out, "webhook_trigger_id=trigger-abc") {
		t.Fatalf("expected webhook_trigger_id in logs, got:\n%s", out)
	}
	if strings.Contains(out, "awt_supersecret") {
		t.Fatalf("token leaked into logs:\n%s", out)
	}
}

func TestIsSoftNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		body string
		want bool
	}{
		{`{"error":"runtime not found"}`, true},
		{`{"error":"task not found"}`, true},
		{`{"error":"Runtime Not Found"}`, true},
		{`{"error":"not found"}`, false},
		{`{"error":"workspace not found"}`, false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isSoftNotFound([]byte(tc.body)); got != tc.want {
			t.Errorf("isSoftNotFound(%q) = %v, want %v", tc.body, got, tc.want)
		}
	}
}

// --- F5 OBS-2 ingress span wiring tests (metadata-only, fail-closed) ---

func TestEmitIngressSpanRecordsMetadataOnlyWithBothIDs(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/v1/tasks"}
	// Raw path contains a token-like segment that MUST NOT reach the span.
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks/awt_secret-token-xyz", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withIngressHolder(req.Context()))
	SetIngressPrincipalClass(req, "user")
	SetIngressTaskID(req.Context(), "task-abc")

	emitIngressSpan(req, http.StatusAccepted, 8*time.Millisecond)

	if sink.Len() != 1 {
		t.Fatalf("expected 1 ingress span, got %d", sink.Len())
	}
	spans := sink.Spans()
	sp := spans[0]
	if err := sp.Validate(); err != nil {
		t.Fatalf("ingress span invalid: %v", err)
	}
	if report := e2e.ScanSpans(spans); !report.Clean {
		t.Fatalf("metadata leak scan not clean: %+v", report.Findings)
	}
	if sp.Hop != e2e.HopIngress {
		t.Fatalf("hop = %q, want ingress", sp.Hop)
	}
	if sp.Correlation.RequestID != e2e.CanonicalRequestID("task-abc") || sp.Correlation.TaskID != "task-abc" {
		t.Fatalf("correlation = %+v (request_id must be canonical from task_id)", sp.Correlation)
	}
	if sp.Labels["method"] != "POST" || sp.Labels["route_template"] != "/v1/tasks" || sp.Labels["principal_class"] != "user" {
		t.Fatalf("labels = %+v", sp.Labels)
	}
	if sp.HTTPStatus != http.StatusAccepted {
		t.Fatalf("status = %d", sp.HTTPStatus)
	}
	for k, v := range sp.Labels {
		if strings.Contains(v, "awt_secret-token-xyz") || strings.Contains(v, "/v1/tasks/awt") {
			t.Fatalf("raw path leaked into label %q=%q", k, v)
		}
	}
	if strings.Contains(sp.Outcome, "awt_") {
		t.Fatalf("raw content leaked into outcome %q", sp.Outcome)
	}
}

func TestEmitIngressSpanFailsClosedOnMissingIDs(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	// Holder present but no task_id -> no span (fail closed).
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks", nil)
	req = req.WithContext(withIngressHolder(req.Context()))
	emitIngressSpan(req, http.StatusOK, time.Millisecond)
	if sink.Len() != 0 {
		t.Fatalf("expected no span without task_id, got %d", sink.Len())
	}

	// No holder at all (non-request context) -> no span, no panic.
	bare := httptest.NewRequest(http.MethodGet, "/v1/tasks", nil)
	emitIngressSpan(bare, http.StatusOK, time.Millisecond)
	if sink.Len() != 0 {
		t.Fatalf("expected no span without holder, got %d", sink.Len())
	}
}

func TestEmitIngressSpanNoopWhenRecorderUnset(t *testing.T) {
	SetIngressRecorder(nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks", nil)
	req = req.WithContext(withIngressHolder(req.Context()))
	SetIngressTaskID(req.Context(), "task-1")
	// Must not panic and must be a no-op with no recorder installed.
	emitIngressSpan(req, http.StatusOK, time.Millisecond)
}

func TestIngressRouteTemplateNeverRawPath(t *testing.T) {
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/v1/tasks/{id}"}
	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/awt_secret", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	if got := ingressRouteTemplate(req); got != "/v1/tasks/{id}" {
		t.Fatalf("route template = %q, want bounded template not raw path", got)
	}
	// No chi context -> empty (omit label), never the raw path.
	bare := httptest.NewRequest(http.MethodGet, "/v1/tasks/awt_secret", nil)
	if got := ingressRouteTemplate(bare); got != "" {
		t.Fatalf("expected empty template without chi ctx, got %q", got)
	}
}

func TestIngressOutcomeBoundedCodes(t *testing.T) {
	cases := map[int]string{
		http.StatusAccepted: "accepted", http.StatusFound: "redirect",
		http.StatusNotFound: "client_error", http.StatusInternalServerError: "server_error",
		http.StatusContinue: "unknown",
	}
	for status, want := range cases {
		if got := ingressOutcome(status); got != want {
			t.Fatalf("ingressOutcome(%d) = %q, want %q", status, got, want)
		}
	}
}
