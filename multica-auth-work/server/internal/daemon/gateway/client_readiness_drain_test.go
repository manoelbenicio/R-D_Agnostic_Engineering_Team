package gateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// Regression for the readiness-probe drain fix: under devModelsCompat the
// readiness endpoint (/v1/models) returns OmniRoute's full native catalog, which
// must be FULLY drained so the connection is reusable by the following
// FetchModels. Flag-off retains the bounded drain.

type countingReadCloser struct {
	r      *strings.Reader
	read   int
	closed bool
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}
func (c *countingReadCloser) Close() error { c.closed = true; return nil }

func readinessDrainClient(t *testing.T, maxBody int64, body *countingReadCloser) *Client {
	t.Helper()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       body,
		}, nil
	})
	c, err := NewClient(ClientOptions{
		Gateway:         testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:       EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:      &syntheticCredentialSource{},
		HTTPClient:      &http.Client{Transport: transport},
		RequestTimeout:  time.Second,
		MaxResponseBody: maxBody,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestReadinessProbeCompatFullyDrainsBody(t *testing.T) {
	t.Setenv(envDevModelsCompat, "1")
	// Body larger than the configured max: bounded drain would reject it, but
	// the compat path must fully drain it to io.Discard.
	payload := `{"object":"list","data":[` + strings.Repeat(`{"id":"m"},`, 400) + `{"id":"claude_code_kimi_2.7_Code"}]}`
	if int64(len(payload)) <= 1024 {
		t.Fatalf("payload not large enough: %d", len(payload))
	}
	body := &countingReadCloser{r: strings.NewReader(payload)}
	client := readinessDrainClient(t, 1024, body)

	res, err := client.CheckReadiness(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("compat readiness should succeed by fully draining: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if body.read != len(payload) {
		t.Fatalf("body not fully drained: read=%d want=%d", body.read, len(payload))
	}
	if !body.closed {
		t.Fatal("body not closed")
	}
}

func TestReadinessProbeCompatThenFetchModelsSucceeds(t *testing.T) {
	t.Setenv(envDevModelsCompat, "1")
	native := `{"object":"list","data":[{"id":"gpt-4o"},{"id":"claude_code_kimi_2.7_Code"}]}`
	// Two sequential calls (readiness then models) served by the same client.
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return syntheticResponse(req, http.StatusOK, "application/json", native), nil
	})
	client, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:     &syntheticCredentialSource{},
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CheckReadiness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("readiness: %v", err)
	}
	doc, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels after readiness: %v", err)
	}
	snap, err := buildSnapshot(doc, time.Now())
	if err != nil {
		t.Fatalf("projected doc rejected: %v", err)
	}
	if !snap.Models[brain.RouteModel("claude_code_kimi_2.7_Code")].Available {
		t.Fatal("approved model not available after readiness+fetch")
	}
}

func TestReadinessProbeFlagOffRetainsBoundedDrain(t *testing.T) {
	t.Setenv(envDevModelsCompat, "") // flag off
	// Body larger than the configured max: bounded drain must reject it
	// (unchanged pre-fix behavior).
	payload := strings.Repeat("x", 2000)
	body := &countingReadCloser{r: strings.NewReader(payload)}
	client := readinessDrainClient(t, 1024, body)
	if _, err := client.CheckReadiness(context.Background(), testCorrelation()); !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("flag-off oversize readiness should be bounded/rejected, got %v", err)
	}
	// Small body under the bound still succeeds flag-off.
	small := &countingReadCloser{r: strings.NewReader(`{"ok":true}`)}
	client2 := readinessDrainClient(t, 1024, small)
	if _, err := client2.CheckReadiness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("flag-off small readiness should succeed: %v", err)
	}
}

var _ io.Reader = (*countingReadCloser)(nil)
