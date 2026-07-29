package auth

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/pkg/redact"
)

// This test exercises the real CloudPATVerifier.Verify/fetch path against a
// synthetic non-200 Fleet response containing fake access_token/api-key
// sentinels in the JSON body, using a custom http.RoundTripper — no
// httptest.Server, no listener, no real network socket of any kind. It
// verifies that the production slog.Warn("cloud_pat: verify returned
// non-200", "status", ..., "body", snippet) call, once routed through the
// production redact.SanitizeSlogAttr ReplaceAttr hook, never surfaces the
// sentinels in captured log output while the status code and a truncated,
// still-useful body excerpt remain visible for diagnostics.
//
// t.Parallel is deliberately not used: this test mutates the process-global
// slog default (via slog.SetDefault) for the duration of the call to Verify,
// and must not race with any other test doing the same.

// staticRoundTripper is a minimal http.RoundTripper that returns a
// pre-built *http.Response for any request, without touching the network,
// a listener, or any external process. It exists purely to let this test
// drive CloudPATVerifier.fetch's real HTTP call path deterministically.
type staticRoundTripper struct {
	statusCode int
	body       string
}

func (t *staticRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Status:     http.StatusText(t.statusCode),
		Body:       io.NopCloser(bytes.NewReader([]byte(t.body))),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// captureSlogDefault temporarily replaces the process-global slog default
// with a JSON handler wired to the same production ReplaceAttr hook
// (redact.SanitizeSlogAttr) used by internal/logger.Init/NewLogger, writing
// into buf. It returns a restore function that must be deferred immediately
// so the prior global default is always put back, even if the test fails.
func captureSlogDefault(buf *bytes.Buffer) (restore func()) {
	prior := slog.Default()
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: redact.SanitizeSlogAttr,
	})
	slog.SetDefault(slog.New(handler))
	return func() {
		slog.SetDefault(prior)
	}
}

func TestCloudPATVerifyNon200BodyRedactsAccessTokenAndAPIKeySentinels(t *testing.T) {
	const (
		accessTokenSentinel = "synthetic-access-token-sentinel-0011223344"
		apiKeySentinel      = "synthetic-api-key-sentinel-5566778899"
	)
	fleetBody := `{"error":"invalid_request","access_token":"` + accessTokenSentinel +
		`","api_key":"` + apiKeySentinel + `","reason":"malformed token"}`

	verifier := NewCloudPATVerifier(CloudPATVerifierConfig{
		FleetBaseURL: "https://synthetic-fleet.invalid",
		HTTPClient: &http.Client{
			Transport: &staticRoundTripper{statusCode: http.StatusBadRequest, body: fleetBody},
		},
		// Redis intentionally nil: caching is irrelevant to this log-safety
		// test and nil-Redis is already the documented "disable cache" mode.
	})
	if verifier == nil {
		t.Fatal("NewCloudPATVerifier returned nil for a non-empty FleetBaseURL")
	}

	var logBuf bytes.Buffer
	restore := captureSlogDefault(&logBuf)
	defer restore()

	_, err := verifier.Verify(context.Background(), "mcn_synthetic-token-not-real", nil)
	if err == nil {
		t.Fatal("expected Verify to fail closed on a non-200 Fleet response")
	}

	got := logBuf.String()
	if strings.Contains(got, accessTokenSentinel) {
		t.Fatalf("access_token sentinel leaked into captured log output: %q", got)
	}
	if strings.Contains(got, apiKeySentinel) {
		t.Fatalf("api_key sentinel leaked into captured log output: %q", got)
	}

	// Status/body context must remain useful for diagnostics even though
	// the secret-shaped fields are redacted.
	if !strings.Contains(got, "cloud_pat: verify returned non-200") {
		t.Fatalf("expected the production log message to be present: %q", got)
	}
	if !strings.Contains(got, "400") {
		t.Fatalf("expected the non-200 status code (400) to remain visible: %q", got)
	}
	if !strings.Contains(got, "malformed token") {
		t.Fatalf("expected the non-secret diagnostic reason to remain visible: %q", got)
	}
	if !strings.Contains(got, "[REDACTED") {
		t.Fatalf("expected a redaction placeholder to be present: %q", got)
	}
}

func TestCloudPATVerifyNon200BodyRedactsBearerTokenSentinel(t *testing.T) {
	const bearerSentinel = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJTWU5USEVUSUMifQ.synthetic"
	fleetBody := `{"error":"unauthorized","message":"Authorization: Bearer ` + bearerSentinel + `"}`

	verifier := NewCloudPATVerifier(CloudPATVerifierConfig{
		FleetBaseURL: "https://synthetic-fleet.invalid",
		HTTPClient: &http.Client{
			Transport: &staticRoundTripper{statusCode: http.StatusInternalServerError, body: fleetBody},
		},
	})
	if verifier == nil {
		t.Fatal("NewCloudPATVerifier returned nil for a non-empty FleetBaseURL")
	}

	var logBuf bytes.Buffer
	restore := captureSlogDefault(&logBuf)
	defer restore()

	_, err := verifier.Verify(context.Background(), "mcn_synthetic-token-not-real", nil)
	if err == nil {
		t.Fatal("expected Verify to fail closed on a non-200 Fleet response")
	}

	got := logBuf.String()
	if strings.Contains(got, bearerSentinel) {
		t.Fatalf("bearer token sentinel leaked into captured log output: %q", got)
	}
	if !strings.Contains(got, "500") {
		t.Fatalf("expected the non-200 status code (500) to remain visible: %q", got)
	}
}

func TestCloudPATVerifySafeNon200BodyIsPreservedForDiagnostics(t *testing.T) {
	// A realistic Fleet error body with no secret-shaped field at all must
	// pass through unredacted, proving this test's redaction assertions are
	// not vacuously true (i.e. the mechanism doesn't just blank everything).
	fleetBody := `{"error":"rate_limited","reason":"too many verify requests"}`

	verifier := NewCloudPATVerifier(CloudPATVerifierConfig{
		FleetBaseURL: "https://synthetic-fleet.invalid",
		HTTPClient: &http.Client{
			Transport: &staticRoundTripper{statusCode: http.StatusTooManyRequests, body: fleetBody},
		},
	})
	if verifier == nil {
		t.Fatal("NewCloudPATVerifier returned nil for a non-empty FleetBaseURL")
	}

	var logBuf bytes.Buffer
	restore := captureSlogDefault(&logBuf)
	defer restore()

	_, err := verifier.Verify(context.Background(), "mcn_synthetic-token-not-real", nil)
	if err == nil {
		t.Fatal("expected Verify to fail closed on a non-200 Fleet response")
	}

	got := logBuf.String()
	if !strings.Contains(got, "too many verify requests") {
		t.Fatalf("safe, non-sensitive diagnostic body content was altered/dropped: %q", got)
	}
	if !strings.Contains(got, "429") {
		t.Fatalf("expected the non-200 status code (429) to remain visible: %q", got)
	}
}
