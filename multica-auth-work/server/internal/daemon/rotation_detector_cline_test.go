package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestClineExhaustionDetectorMatcher covers the Cline screen matcher directly:
// provider passthrough rate/usage/quota banners plus Cline-branded messages.
func TestClineExhaustionDetectorMatcher(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{name: "rate limit exceeded with try again", screenText: "API request failed: rate limit exceeded. Please try again later.", want: true},
		{name: "usage limit reached with reset", screenText: "Usage limit reached. Your limit will reset at 2pm.", want: true},
		{name: "cline branded limit with retry", screenText: "You've hit your Cline limit. Retry after cooldown.", want: true},
		{name: "quota exceeded with try again", screenText: "Quota exceeded. Please try again in 30 minutes.", want: true},
		{name: "too many requests with back off", screenText: "Too many requests. Please back off and retry.", want: true},
		{name: "limit phrase without reset indicator", screenText: "rate limit exceeded", want: false},
		{name: "reset indicator without limit phrase", screenText: "Please try again later.", want: false},
		{name: "normal agent output", screenText: "Adding tests for the new rotation module.", want: false},
		{name: "bare rate limit mention is not exhaustion", screenText: "The provider rate limit is 100 requests per minute.", want: false},
		{name: "empty screen", screenText: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesClineScreenExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesClineScreenExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestClineExhaustionDetectorViaDetect exercises the rotation.ExhaustionDetector
// contract: screen signal + reset parsing, 429, 503, high-traffic.
func TestClineExhaustionDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	d := &ClineExhaustionDetector{now: func() time.Time { return now }}

	t.Run("implements ExhaustionDetector", func(t *testing.T) {
		var _ rotation.ExhaustionDetector = d
	})

	t.Run("clock reset parsed", func(t *testing.T) {
		got := d.Detect("cline", "Usage limit reached. Your limit will reset at 2pm.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt == nil || got.ResetAt.Hour() != 14 {
			t.Fatalf("Detect = %+v, want exhausted screen 14:00", got)
		}
	})

	t.Run("relative reset falls back to nil", func(t *testing.T) {
		got := d.Detect("cline", "rate limit exceeded. Please try again in 30 minutes.", 0)
		if !got.Exhausted || got.Signal != rotation.SignalScreen || got.ResetAt != nil {
			t.Fatalf("Detect = %+v, want exhausted screen nil-reset", got)
		}
	})

	t.Run("http 429 triggers", func(t *testing.T) {
		got := d.Detect("cline", "", 429)
		if !got.Exhausted || got.Signal != rotation.SignalHTTP429 {
			t.Fatalf("Detect = %+v, want exhausted http429", got)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := d.Detect("cline", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted", got)
		}
	})

	t.Run("http 503 does not rotate", func(t *testing.T) {
		got := d.Detect("cline", "Usage limit reached. Your limit will reset at 2pm.", 503)
		if got.Exhausted {
			t.Fatalf("Detect = %+v, want not exhausted for 503", got)
		}
	})
}
