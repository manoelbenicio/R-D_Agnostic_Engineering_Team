package rotation

import (
	"testing"
	"time"
)

// TestClineMatcherDetectsExhaustion covers the Cline matcher directly: provider
// passthrough rate/usage/quota banners plus Cline-branded limit messages, each
// paired with a reset/retry indicator.
func TestClineMatcherDetectsExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{
			name:       "rate limit exceeded with try again",
			screenText: "API request failed: rate limit exceeded. Please try again later.",
			want:       true,
		},
		{
			name:       "usage limit reached with reset",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			want:       true,
		},
		{
			name:       "cline branded limit with retry",
			screenText: "You've hit your Cline limit. Retry after cooldown.",
			want:       true,
		},
		{
			name:       "quota exceeded with try again",
			screenText: "Quota exceeded. Please try again in 30 minutes.",
			want:       true,
		},
		{
			name:       "too many requests with back off",
			screenText: "Too many requests. Please back off and retry.",
			want:       true,
		},
		{
			name:       "limit phrase without reset indicator",
			screenText: "rate limit exceeded",
			want:       false,
		},
		{
			name:       "reset indicator without limit phrase",
			screenText: "Please try again later.",
			want:       false,
		},
		{
			name:       "normal agent output",
			screenText: "Adding tests for the new rotation module.",
			want:       false,
		},
		{
			name:       "bare rate limit mention is not exhaustion",
			screenText: "The provider rate limit is 100 requests per minute.",
			want:       false,
		},
		{
			name:       "empty screen",
			screenText: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesClineExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesClineExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestClineDetectorViaDetect confirms the Cline matcher is wired into Detector.Detect
// (screen signal + reset-time parsing) and that 429/high-traffic behave per spec.
func TestClineDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	detector := &Detector{now: func() time.Time { return now }}

	t.Run("clock reset parsed", func(t *testing.T) {
		got := detector.Detect("cline", "Usage limit reached. Your limit will reset at 2pm.", 0)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalScreen {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
		}
		if got.ResetAt == nil || got.ResetAt.Hour() != 14 || got.ResetAt.Minute() != 0 {
			t.Fatalf("ResetAt = %v, want 14:00", got.ResetAt)
		}
	})

	t.Run("relative reset falls back to nil", func(t *testing.T) {
		got := detector.Detect("cline", "rate limit exceeded. Please try again in 30 minutes.", 0)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalScreen {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
		}
		if got.ResetAt != nil {
			t.Fatalf("ResetAt = %v, want nil for relative reset hint", got.ResetAt)
		}
	})

	t.Run("http 429 still triggers", func(t *testing.T) {
		got := detector.Detect("cline", "", 429)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalHTTP429 {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalHTTP429)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := detector.Detect("cline", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect exhausted = true, want false: %+v", got)
		}
	})
}
