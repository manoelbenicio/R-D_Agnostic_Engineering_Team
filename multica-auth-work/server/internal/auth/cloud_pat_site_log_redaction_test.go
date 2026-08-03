package auth

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestCloudPATNon200BodyDoesNotReachRawLogAttributes(t *testing.T) {
	const sentinel = "ORQ98_SENTINEL_CLOUD_PAT_BODY_91C4"
	fleetBody := `{"error":"invalid_request","access_token":"` + sentinel + `"}`

	verifier := NewCloudPATVerifier(CloudPATVerifierConfig{
		FleetBaseURL: "https://orq98-synthetic-fleet.invalid",
		HTTPClient: &http.Client{
			Transport: &staticRoundTripper{statusCode: http.StatusBadRequest, body: fleetBody},
		},
	})
	if verifier == nil {
		t.Fatal("NewCloudPATVerifier returned nil for the synthetic Fleet URL")
	}

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, err := verifier.Verify(context.Background(), "mcn_orq98_synthetic_not_real", nil)
	if err == nil {
		t.Fatal("expected verification to fail closed on a non-200 Fleet response")
	}
	if strings.Contains(logs.String(), sentinel) {
		t.Fatal("sentinel reached captured slog output or attributes")
	}
	if !strings.Contains(logs.String(), "cloud_pat: verify returned non-200") {
		t.Fatal("expected nominal discovery of the Cloud PAT non-200 log site")
	}
}
