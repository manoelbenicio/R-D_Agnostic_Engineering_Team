package rotation

import (
	"bytes"
	"log"
	"log/slog"
	"testing"
	"time"
)

func TestDetectDiscoverySessionStatusExhaustedOrExpired(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	for _, status := range []string{"expired", "EXHAUSTED"} {
		t.Run(status, func(t *testing.T) {
			got := DetectDiscoverySession("codex", DiscoverySession{
				Provider:  "codex",
				Status:    status,
				ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			}, now)

			assertDiscoveryExhausted(t, got)
		})
	}
}

func TestDetectDiscoverySessionExpiresAtBoundary(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		status    string
		expiresAt time.Time
		want      bool
	}{
		{name: "past", status: "active", expiresAt: now.Add(-time.Nanosecond), want: true},
		{name: "equal", status: "active", expiresAt: now, want: true},
		{name: "future expiring session", status: "expiring", expiresAt: now.Add(time.Nanosecond), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectDiscoverySession("codex", DiscoverySession{
				Provider:  "codex",
				Status:    tt.status,
				ExpiresAt: tt.expiresAt.Format(time.RFC3339Nano),
			}, now)
			if got.Exhausted != tt.want {
				t.Fatalf("Exhausted = %v, want %v: %+v", got.Exhausted, tt.want, got)
			}
		})
	}
}

func TestDetectDiscoverySessionMissingOrMalformedExpiryPreservesFallback(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	for _, expiresAt := range []string{"", "not-a-timestamp", "2026-07-18"} {
		t.Run(expiresAt, func(t *testing.T) {
			got := DetectDiscoverySession("kiro", DiscoverySession{
				Provider:  "kiro_cli",
				Status:    "active",
				ExpiresAt: expiresAt,
			}, now)
			if got.Exhausted || got.Signal != "" || got.ResetAt != nil {
				t.Fatalf("DetectDiscoverySession = %+v, want empty fallback result", got)
			}
		})
	}
}

func TestDetectDiscoverySessionProviderBoundaries(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	expired := DiscoverySession{Provider: "codex", Status: "expired"}

	if got := DetectDiscoverySession("kiro", expired, now); got.Exhausted {
		t.Fatalf("cross-provider observation exhausted kiro: %+v", got)
	}
	if got := DetectDiscoverySession("", expired, now); got.Exhausted {
		t.Fatalf("missing requested provider exhausted a pool: %+v", got)
	}
	if got := DetectDiscoverySession("kiro", DiscoverySession{Provider: "", Status: "expired"}, now); got.Exhausted {
		t.Fatalf("missing discovery provider exhausted kiro: %+v", got)
	}
	assertDiscoveryExhausted(t, DetectDiscoverySession("kiro", DiscoverySession{
		Provider: "kiro_cli",
		Status:   "expired",
	}, now))
}

func TestDetectDiscoverySessionDoesNotLogSecrets(t *testing.T) {
	var legacyLog bytes.Buffer
	previousLegacyWriter := log.Writer()
	log.SetOutput(&legacyLog)
	t.Cleanup(func() { log.SetOutput(previousLegacyWriter) })

	var structuredLog bytes.Buffer
	previousStructuredLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&structuredLog, nil)))
	t.Cleanup(func() { slog.SetDefault(previousStructuredLogger) })

	secretSentinel := "token-secret-must-not-appear"
	got := DetectDiscoverySession("codex", DiscoverySession{
		Provider:  "codex",
		Status:    "active",
		ExpiresAt: secretSentinel,
	}, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))

	if got.Exhausted {
		t.Fatalf("malformed secret-like timestamp exhausted session: %+v", got)
	}
	if legacyLog.Len() != 0 || structuredLog.Len() != 0 {
		t.Fatalf("discovery detection logged input: legacy=%q structured=%q", legacyLog.String(), structuredLog.String())
	}
}

func assertDiscoveryExhausted(t *testing.T, got DetectionResult) {
	t.Helper()
	if !got.Exhausted {
		t.Fatalf("Exhausted = false, want true: %+v", got)
	}
	if got.Signal != SignalDiscovery {
		t.Fatalf("Signal = %q, want %q", got.Signal, SignalDiscovery)
	}
	if got.ResetAt != nil {
		t.Fatalf("ResetAt = %v, want nil", got.ResetAt)
	}
}
