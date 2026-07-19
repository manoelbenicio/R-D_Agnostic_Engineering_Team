package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestNimExhaustionDetectorMatcher covers the NIM screen matcher directly:
// OpenAI-compatible rate/usage/quota banners (NIM is OpenAI-compatible), NVIDIA/NIM
// branded messages, and credits-exhausted, each paired with a reset/retry indicator.
func TestNimExhaustionDetectorMatcher(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{name: "rate limit exceeded with try again", screenText: "request failed: rate limit exceeded. Please try again later.", want: true},
		{name: "usage limit reached with reset", screenText: "Usage limit reached. Your limit will reset at 2pm.", want: true},
		{name: "nim branded limit with retry", screenText: "You've hit your NIM limit. Retry after cooldown.", want: true},
		{name: "nvidia branded limit with retry", screenText: "You've hit your NVIDIA limit. Please retry in a moment.", want: true},
		{name: "quota exceeded with try again", screenText: "Quota exceeded. Please try again in 30 minutes.", want: true},
		{name: "too many requests with back off", screenText: "Too many requests. Please back off and retry.", want: true},
		{name: "credits exhausted with reset", screenText: "Credits exhausted. Your credits will reset at 12am.", want: true},
		{name: "limit phrase without reset indicator", screenText: "rate limit exceeded", want: false},
		{name: "reset indicator without limit phrase", screenText: "Please try again later.", want: false},
		{name: "normal agent output", screenText: "Adding tests for the NIM rotation module.", want: false},
		{name: "bare rate limit mention is not exhaustion", screenText: "The endpoint rate limit is 100 requests per minute.", want: false},
		{name: "empty screen", screenText: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNimScreenExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesNimScreenExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestNimExhaustionDetectorViaDetect exercises the rotation.ExhaustionDetector
// contract: screen signal + reset parsing, 429, 503, high-traffic. Uses the
// NimExhaustionDetector struct directly (via the shared detectExhaustion flow),
// so it stays green without the daemon.go / detector.go wiring (Kiro, Wave 2).
func TestNimExhaustionDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	d := &NimExhaustionDetector{now: func() time.Time { return now }}

	t.Run("implements ExhaustionDetector", func(t *testing.T) {
		var _ rotation.ExhaustionDetector = d
	})

	t.Run("clock reset parsed", func(t *testing.T) {
		got := d.Detect("nim", "Usage limit reached. Your limit will reset at 2pm.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 14 {
			t.Fatalf("Detect = %+v, want exhausted screen 14:00", got)
		}
	})

	t.Run("relative reset falls back to nil", func(t *testing.T) {
		got := d.Detect("nim", "rate limit exceeded. Please try again in 30 minutes.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt != nil {
			t.Fatalf("Detect = %+v, want exhausted screen nil-reset", got)
		}
	})

	t.Run("http 429 triggers", func(t *testing.T) {
		got := d.Detect("nim", "", 429)
		if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
			t.Fatalf("Detect = %+v, want exhausted http429", got)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := d.Detect("nim", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted", got)
		}
	})

	t.Run("http 503 does not rotate", func(t *testing.T) {
		got := d.Detect("nim", "Usage limit reached. Your limit will reset at 2pm.", 503)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted for 503", got)
		}
	})

	t.Run("new constructor returns working detector", func(t *testing.T) {
		nd := NewNimExhaustionDetector()
		got := nd.Detect("nim", "", 429)
		if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
			t.Fatalf("NewNimExhaustionDetector Detect = %+v, want exhausted http429", got)
		}
	})
}
