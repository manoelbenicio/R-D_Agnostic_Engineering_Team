package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestOpenCodeExhaustionDetectorMatcher covers the OpenCode screen matcher
// directly: provider-agnostic rate/usage/token/quota banners.
func TestOpenCodeExhaustionDetectorMatcher(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{name: "usage limit reached with reset", screenText: "Usage limit reached. Your limit will reset at 4:05 PM.", want: true},
		{name: "token limit reached with try again", screenText: "Token limit reached. Please try again later.", want: true},
		{name: "exceeded quota with retry", screenText: "Exceeded your quota. Retry in 5 minutes.", want: true},
		{name: "rate limit reached with cooldown", screenText: "rate limit reached. Please wait for cooldown to reset.", want: true},
		{name: "opencode branded limit with reset", screenText: "You've hit your OpenCode limit. Resets at 3:36 PM.", want: true},
		{name: "limit phrase without reset indicator", screenText: "rate limit exceeded", want: false},
		{name: "reset indicator without limit phrase", screenText: "Please try again later.", want: false},
		{name: "normal agent output", screenText: "Running the test suite for the rotation package.", want: false},
		{name: "bare token mention is not exhaustion", screenText: "Counting token usage for the response.", want: false},
		{name: "empty screen", screenText: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesOpenCodeScreenExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesOpenCodeScreenExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestOpenCodeExhaustionDetectorViaDetect exercises the rotation.ExhaustionDetector
// contract: screen signal + reset parsing, 429, 503, high-traffic.
func TestOpenCodeExhaustionDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	d := &OpenCodeExhaustionDetector{now: func() time.Time { return now }}

	t.Run("implements ExhaustionDetector", func(t *testing.T) {
		var _ rotation.ExhaustionDetector = d
	})

	t.Run("clock reset parsed", func(t *testing.T) {
		got := d.Detect("opencode", "Usage limit reached. Your limit will reset at 4:05 PM.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 16 || got.ResetAt.Minute() != 5 {
			t.Fatalf("Detect = %+v, want exhausted screen 16:05", got)
		}
	})

	t.Run("relative reset falls back to nil", func(t *testing.T) {
		got := d.Detect("opencode", "Token limit reached. Please try again in 10 minutes.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt != nil {
			t.Fatalf("Detect = %+v, want exhausted screen nil-reset", got)
		}
	})

	t.Run("http 429 triggers", func(t *testing.T) {
		got := d.Detect("opencode", "", 429)
		if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
			t.Fatalf("Detect = %+v, want exhausted http429", got)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := d.Detect("opencode", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted", got)
		}
	})

	t.Run("http 503 does not rotate", func(t *testing.T) {
		got := d.Detect("opencode", "Usage limit reached. Your limit will reset at 4:05 PM.", 503)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted for 503", got)
		}
	})
}
