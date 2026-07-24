package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// largeModelsBody builds an ~83KB OpenAI-basic /v1/models payload including the
// approved route, matching the real OmniRoute readiness catalog size.
func largeModelsBody() string {
	var b strings.Builder
	b.WriteString(`{"object":"list","data":[`)
	for i := 0; i < 2600; i++ {
		fmt.Fprintf(&b, `{"id":"model-placeholder-%04d"},`, i)
	}
	b.WriteString(`{"id":"claude_code_kimi_2.7_Code"}]}`)
	return b.String()
}

// TestDoReadsFullBodyWithoutPrematureCancel is a regression for the readiness
// instability root cause: Client.do() must NOT cancel the request context when
// it returns the response, because callers read the (large) body afterwards.
// The server writes the body slowly/chunked so the read spans time; under the
// premature-cancel bug this fails with "context canceled".
func TestDoReadsFullBodyWithoutPrematureCancel(t *testing.T) {
	t.Setenv(envDevModelsCompat, "1")
	payload := largeModelsBody()
	if len(payload) < 80000 {
		t.Fatalf("payload too small: %d", len(payload))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		// Write in chunks with small delays so the body read spans real time
		// and clearly occurs after do() has returned the response.
		const chunk = 8192
		for off := 0; off < len(payload); off += chunk {
			end := off + chunk
			if end > len(payload) {
				end = len(payload)
			}
			_, _ = w.Write([]byte(payload[off:end]))
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(3 * time.Millisecond)
		}
	}))
	defer srv.Close()

	c, err := NewClient(ClientOptions{
		Gateway:         testGatewayConfig(t, srv.URL),
		Endpoints:       EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:      &syntheticCredentialSource{},
		RequestTimeout:  10 * time.Second,
		MaxResponseBody: 10 << 20,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// FetchModels reads the full body via readBounded — the exact path that
	// failed with "context canceled" before the fix.
	for i := 0; i < 5; i++ {
		doc, err := c.FetchModels(context.Background(), testCorrelation())
		if err != nil {
			t.Fatalf("FetchModels attempt %d: unexpected error (premature cancel regression?): %v", i, err)
		}
		found := false
		for _, m := range doc.Models {
			if m.ID == "claude_code_kimi_2.7_Code" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("attempt %d: approved model missing from fully-read body", i)
		}
	}
}

// TestDoErrorPathCancelsContext verifies a non-2xx response returns an error and
// does not leak the request context (cancel fires on the error path).
func TestDoErrorPathCancelsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c, err := NewClient(ClientOptions{
		Gateway:         testGatewayConfig(t, srv.URL),
		Endpoints:       EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:      &syntheticCredentialSource{},
		RequestTimeout:  5 * time.Second,
		MaxResponseBody: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.FetchModels(context.Background(), testCorrelation()); err == nil {
		t.Fatal("expected error on 500 status")
	}
}
