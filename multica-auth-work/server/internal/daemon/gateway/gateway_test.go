package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const syntheticCredentialValue = "fixture-credential-value"

type syntheticCredentialSource struct {
	calls atomic.Int64
	err   error
}

func (s *syntheticCredentialSource) WithCredential(ctx context.Context, ref brain.SecretFileRef, use func(string) error) error {
	s.calls.Add(1)
	if s.err != nil {
		return s.err
	}
	if ref.Path != "/synthetic/reference/only" {
		return errors.New("unexpected reference")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return use(syntheticCredentialValue)
}

func testCorrelation() brain.Correlation {
	return brain.Correlation{TaskID: "task-synthetic", SessionID: "session-synthetic", RequestID: "request-synthetic"}
}

func testGatewayConfig(t *testing.T, baseURL string) brain.GatewayConfig {
	t.Helper()
	secretRef, err := brain.NewSecretFileRef("/synthetic/reference/only")
	if err != nil {
		t.Fatalf("NewSecretFileRef: %v", err)
	}
	return brain.GatewayConfig{Required: true, BaseURL: baseURL, SecretFile: secretRef, Readiness: brain.StrictReadinessPolicy()}
}

func newTestClient(t *testing.T, baseURL string, source CredentialSource, timeout time.Duration) *Client {
	t.Helper()
	client, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, baseURL),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/health/ready"},
		Credential:     source,
		RequestTimeout: timeout,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestClientSeparatesProbesAndAuthenticatesModels(t *testing.T) {
	source := &syntheticCredentialSource{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get(HeaderTaskID) != "task-synthetic" || request.Header.Get(HeaderSessionID) != "session-synthetic" || request.Header.Get(HeaderRequestID) != "request-synthetic" {
			t.Error("correlation headers missing")
		}
		switch request.URL.Path {
		case "/health/live":
			if request.Header.Get("Authorization") != "" {
				t.Error("liveness probe unexpectedly authenticated")
			}
		case "/health/ready", "/v1/models":
			if request.Header.Get("Authorization") != "Bearer "+syntheticCredentialValue {
				t.Error("authenticated request missing synthetic authorization")
			}
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
		}
		w.Header().Set(HeaderOmniRouteRequestID, "omni-request-synthetic")
		w.Header().Set(HeaderRegistryVersion, "registry-v1")
		if request.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"synthetic-ok"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, source, time.Second)
	if _, err := client.CheckLiveness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("CheckLiveness: %v", err)
	}
	if _, err := client.CheckReadiness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	document, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if document.RegistryVersion != "registry-v1" || source.calls.Load() != 2 {
		t.Fatalf("unexpected authenticated calls/version: calls=%d version=%q", source.calls.Load(), document.RegistryVersion)
	}
	printed := client.String()
	if strings.Contains(printed, syntheticCredentialValue) || strings.Contains(printed, "/synthetic/reference/only") {
		t.Fatal("client string exposed credential material")
	}
}

func TestClientDeterministicErrorClassificationAndRedaction(t *testing.T) {
	source := &syntheticCredentialSource{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"fixture-credential-value must never escape"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, source, time.Second)
	_, err := client.CheckReadiness(context.Background(), testCorrelation())
	var gatewayErr *GatewayError
	if !errors.As(err, &gatewayErr) || gatewayErr.Class != ErrorRateLimited || !gatewayErr.Retryable || gatewayErr.RetryAfter != 3*time.Second {
		t.Fatalf("unexpected classified error: %#v", err)
	}
	if strings.Contains(err.Error(), syntheticCredentialValue) {
		t.Fatal("error exposed response content")
	}
}

func TestClientCancellationAndTimeout(t *testing.T) {
	source := &syntheticCredentialSource{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, source, 20*time.Millisecond)
	_, err := client.CheckReadiness(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorTimeout) {
		t.Fatalf("expected timeout, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.CheckReadiness(ctx, testCorrelation())
	if !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestClientRejectsUnsafeConfigurationAndCorrelation(t *testing.T) {
	source := &syntheticCredentialSource{}
	_, err := NewClient(ClientOptions{
		Gateway:    testGatewayConfig(t, "http://127.0.0.1:20128"),
		Endpoints:  EndpointSet{Liveness: "relative", Readiness: "/ready"},
		Credential: source,
	})
	if !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid configuration, got %v", err)
	}
	client := newTestClient(t, "http://127.0.0.1:1", source, time.Second)
	correlation := testCorrelation()
	correlation.RequestID = "unsafe\nheader"
	_, err = client.CheckLiveness(context.Background(), correlation)
	if !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid correlation, got %v", err)
	}
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	source := &syntheticCredentialSource{}
	redirectTargetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirectTargetCalled = true
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, source, time.Second)
	_, err := client.CheckReadiness(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorProtocol) || redirectTargetCalled {
		t.Fatalf("redirect was not rejected: err=%v target_called=%t", err, redirectTargetCalled)
	}
}

func boolPointer(value bool) *bool { return &value }
