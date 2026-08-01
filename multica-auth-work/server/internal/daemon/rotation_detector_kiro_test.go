package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestKiroExhaustionDetectorMatcher covers the Kiro screen matcher directly:
// Claude/Bedrock passthrough phrases plus Kiro/Amazon-Q credit-model phrases.
func TestKiroExhaustionDetectorMatcher(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{name: "claude passthrough usage limit with reset", screenText: "Usage limit reached. Your limit will reset at 2pm.", want: true},
		{name: "kiro usage limit with date reset", screenText: "You've reached your Kiro usage limit. Credits reset on 2026-07-10.", want: true},
		{name: "kiro credits exhausted with try again", screenText: "Out of Kiro credits. Try again after the plan refreshes.", want: true},
		{name: "5h limit reached with reset", screenText: "5-hour limit reached. Resets 6am.", want: true},
		{name: "limit reached without reset indicator", screenText: "5-hour limit reached.", want: false},
		{name: "reset indicator without limit phrase", screenText: "Your limit will reset at 2pm.", want: false},
		{name: "normal agent output", screenText: "Refactoring the auth module and adding tests.", want: false},
		{name: "empty screen", screenText: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesKiroScreenExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesKiroScreenExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestKiroExhaustionDetectorViaDetect exercises the rotation.ExhaustionDetector
// contract: screen signal + reset parsing, 429, 503, high-traffic.
func TestKiroExhaustionDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	d := &KiroExhaustionDetector{now: func() time.Time { return now }}

	t.Run("implements ExhaustionDetector", func(t *testing.T) {
		var _ rotation.ExhaustionDetector = d
	})

	t.Run("clock reset parsed", func(t *testing.T) {
		got := d.Detect("kiro", "Usage limit reached. Your limit will reset at 2pm.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 14 {
			t.Fatalf("Detect = %+v, want exhausted screen 14:00", got)
		}
	})

	t.Run("date reset falls back to nil", func(t *testing.T) {
		got := d.Detect("kiro", "You've reached your Kiro usage limit. Credits reset on 2026-07-10.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt != nil {
			t.Fatalf("Detect = %+v, want exhausted screen nil-reset", got)
		}
	})

	t.Run("http 429 triggers", func(t *testing.T) {
		got := d.Detect("kiro", "", 429)
		if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
			t.Fatalf("Detect = %+v, want exhausted http429", got)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := d.Detect("kiro", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted", got)
		}
	})

	t.Run("http 503 does not rotate", func(t *testing.T) {
		got := d.Detect("kiro", "Usage limit reached. Your limit will reset at 2pm.", 503)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted for 503", got)
		}
	})
}
