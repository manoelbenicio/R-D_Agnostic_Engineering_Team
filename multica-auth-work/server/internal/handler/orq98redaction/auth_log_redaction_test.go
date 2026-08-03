package orq98redaction

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/handler"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func captureRawSlog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &output
}

func TestGoogleLoginNonOKTokenBodyDoesNotReachLogAttributes(t *testing.T) {
	const sentinel = "ORQ98_SENTINEL_GOOGLE_TOKEN_BODY_7D3A"

	previousTransport := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body: io.NopCloser(strings.NewReader(
				`{"error":"invalid_request","api_key":"` + sentinel + `"}`,
			)),
			Header:  make(http.Header),
			Request: req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	t.Setenv("GOOGLE_CLIENT_ID", "orq98-synthetic-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "orq98-synthetic-client-secret")
	logs := captureRawSlog(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", strings.NewReader(`{"code":"orq98-synthetic-code"}`))
	recorder := httptest.NewRecorder()
	new(handler.Handler).GoogleLogin(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("GoogleLogin status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if strings.Contains(recorder.Body.String(), sentinel) {
		t.Fatal("sentinel reached the HTTP response")
	}
	if strings.Contains(logs.String(), sentinel) {
		t.Fatal("sentinel reached captured slog output or attributes")
	}
	if !strings.Contains(logs.String(), "google oauth token exchange returned error") {
		t.Fatal("expected nominal discovery of the Google token failure log site")
	}
}

func TestGoogleLoginNonOKSafeDiagnosticRemainsDiscoverable(t *testing.T) {
	const safeReason = "orq98 synthetic upstream rejection"

	previousTransport := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     http.StatusText(http.StatusTooManyRequests),
			Body:       io.NopCloser(strings.NewReader(`{"error":"rate_limited","reason":"` + safeReason + `"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	t.Setenv("GOOGLE_CLIENT_ID", "orq98-synthetic-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "orq98-synthetic-client-secret")
	logs := captureRawSlog(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", strings.NewReader(`{"code":"orq98-synthetic-code"}`))
	recorder := httptest.NewRecorder()
	new(handler.Handler).GoogleLogin(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("GoogleLogin status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(logs.String(), safeReason) {
		t.Fatal("safe diagnostic context was not discoverable at the Google token log site")
	}
}
