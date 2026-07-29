package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// errorReadCloser deterministically fails on Read (a connection-level fault)
// while reporting a healthy 200 status, so the body-drain classification can be
// exercised without a live network.
type errorReadCloser struct{ closed bool }

func (e *errorReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("synthetic connection reset")
}
func (e *errorReadCloser) Close() error { e.closed = true; return nil }

var _ io.ReadCloser = (*errorReadCloser)(nil)

func fetchModelsClientWithResponse(t *testing.T, resp *http.Response) *Client {
	t.Helper()
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) { return resp, nil })
	c, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:     &syntheticCredentialSource{},
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestFetchModelsBodyReadFailureIsRetryableTransport: a connection-level read
// failure AFTER a healthy 200 is a retryable transport fault, not a protocol
// violation.
func TestFetchModelsBodyReadFailureIsRetryableTransport(t *testing.T) {
	body := &errorReadCloser{}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       body,
	}
	c := fetchModelsClientWithResponse(t, resp)

	_, err := c.FetchModels(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorTransport) {
		t.Fatalf("body read failure after 200 must be ErrorTransport, got %v", err)
	}
	var ge *GatewayError
	if !errors.As(err, &ge) || !ge.Retryable {
		t.Fatalf("transport read failure must be Retryable, got %+v", err)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}

// TestFetchModelsInvalidJSONIsProtocol: a well-read but structurally invalid
// body is a protocol fault (not transport).
func TestFetchModelsInvalidJSONIsProtocol(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"object":"list","data":[`)), // truncated / invalid JSON
	}
	c := fetchModelsClientWithResponse(t, resp)

	_, err := c.FetchModels(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("invalid JSON must be ErrorProtocol, got %v", err)
	}
}

// TestFetchModelsRegistryVersionMismatchIsProtocol: a valid body whose
// registry_version disagrees with the X-OmniRoute-Registry-Version header is a
// protocol fault.
func TestFetchModelsRegistryVersionMismatchIsProtocol(t *testing.T) {
	header := http.Header{"Content-Type": {"application/json"}}
	header.Set(HeaderRegistryVersion, "header-v2") // Set canonicalizes to match Client's Get()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(`{"object":"list","registry_version":"body-v1","data":[]}`)),
	}
	c := fetchModelsClientWithResponse(t, resp)

	_, err := c.FetchModels(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("registry version mismatch must be ErrorProtocol, got %v", err)
	}
}
