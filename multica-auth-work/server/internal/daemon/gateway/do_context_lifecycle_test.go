package gateway

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func realServerClient(t *testing.T, baseURL string, timeout time.Duration) *Client {
	t.Helper()
	c, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, baseURL),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:     &syntheticCredentialSource{},
		RequestTimeout: timeout, // HTTPClient nil -> real defaultHTTPClient
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestClientDoDoesNotCancelBeforeLargeBodyRead proves do() hands the request
// context off to the body: an 83KB body served slowly in flushed chunks is read
// in full by the caller AFTER do() returns, without a "context canceled" abort.
// A real server (not an injected transport) is required because only a real
// http.Response body honors request-context cancellation.
func TestClientDoDoesNotCancelBeforeLargeBodyRead(t *testing.T) {
	const size = 83 * 1024
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		chunk := bytes.Repeat([]byte("a"), 4096)
		for written := 0; written < size; {
			n := size - written
			if n > len(chunk) {
				n = len(chunk)
			}
			if _, err := w.Write(chunk[:n]); err != nil {
				return
			}
			written += n
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(2 * time.Millisecond) // stream slowly across do()'s return
		}
	}))
	defer srv.Close()

	c := realServerClient(t, srv.URL, 30*time.Second)
	resp, err := c.do(context.Background(), operationModels, http.MethodGet, "/v1/models", testCorrelation(), true)
	if err != nil {
		t.Fatalf("do returned error before body read: %v", err)
	}
	if _, ok := resp.Body.(*cancelOnCloseBody); !ok {
		t.Fatalf("success body must be wrapped for cancel-on-close, got %T", resp.Body)
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("body read aborted (context canceled before full read?): %v", readErr)
	}
	if len(body) != size {
		t.Fatalf("short read: got %d bytes, want %d", len(body), size)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close after full read: %v", err)
	}
}

// TestClientDoNonSuccessCancelsAndClosesImmediately proves the error path does
// NOT hand off the context: a non-2xx status returns a classified error with a
// nil response (do closed the body and cancelled the request context via the
// handedOff=false deferred cancel).
func TestClientDoNonSuccessCancelsAndClosesImmediately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer srv.Close()

	c := realServerClient(t, srv.URL, 30*time.Second)
	resp, err := c.do(context.Background(), operationModels, http.MethodGet, "/v1/models", testCorrelation(), true)
	if resp != nil {
		t.Fatalf("non-2xx must return a nil response (body closed in do), got %+v", resp)
	}
	if !IsErrorClass(err, ErrorOverloaded) {
		t.Fatalf("503 must classify ErrorOverloaded, got %v", err)
	}
}

// TestClientDoBodyCloseIsIdempotent proves closing the wrapped success body more
// than once is safe: no double-close panic and no error, and the underlying
// cancel is idempotent (no context leak).
func TestClientDoBodyCloseIsIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","registry_version":"v1","data":[]}`))
	}))
	defer srv.Close()

	c := realServerClient(t, srv.URL, 30*time.Second)
	resp, err := c.do(context.Background(), operationModels, http.MethodGet, "/v1/models", testCorrelation(), true)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	// Second close must not panic and must remain error-free (idempotent).
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("second close must be safe/idempotent, got: %v", err)
	}
}
