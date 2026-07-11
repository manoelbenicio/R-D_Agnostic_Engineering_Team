package rotation

import (
	"testing"
	"time"
)

// TestOpenCodeMatcherDetectsExhaustion covers the OpenCode matcher directly:
// provider-agnostic rate/usage/token/quota banners paired with a reset/retry indicator.
func TestOpenCodeMatcherDetectsExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{
			name:       "usage limit reached with reset",
			screenText: "Usage limit reached. Your limit will reset at 4:05 PM.",
			want:       true,
		},
		{
			name:       "token limit reached with try again",
			screenText: "Token limit reached. Please try again later.",
			want:       true,
		},
		{
			name:       "exceeded quota with retry",
			screenText: "Exceeded your quota. Retry in 5 minutes.",
			want:       true,
		},
		{
			name:       "rate limit exceeded with cooldown",
			screenText: "rate limit reached. Please wait for cooldown to reset.",
			want:       true,
		},
		{
			name:       "opencode branded limit with reset",
			screenText: "You've hit your OpenCode limit. Resets at 3:36 PM.",
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
			screenText: "Running the test suite for the rotation package.",
			want:       false,
		},
		{
			name:       "bare token mention is not exhaustion",
			screenText: "Counting token usage for the response.",
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
			got := matchesOpenCodeExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesOpenCodeExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestOpenCodeDetectorViaDetect confirms the OpenCode matcher is wired into
// Detector.Detect (screen signal + reset-time parsing) and 429/high-traffic behave.
func TestOpenCodeDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	detector := &Detector{now: func() time.Time { return now }}

	t.Run("clock reset parsed", func(t *testing.T) {
		got := detector.Detect("opencode", "Usage limit reached. Your limit will reset at 4:05 PM.", 0)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalScreen {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
		}
		if got.ResetAt == nil || got.ResetAt.Hour() != 16 || got.ResetAt.Minute() != 5 {
			t.Fatalf("ResetAt = %v, want 16:05", got.ResetAt)
		}
	})

	t.Run("relative reset falls back to nil", func(t *testing.T) {
		got := detector.Detect("opencode", "Token limit reached. Please try again in 10 minutes.", 0)
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
		got := detector.Detect("opencode", "", 429)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalHTTP429 {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalHTTP429)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := detector.Detect("opencode", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect exhausted = true, want false: %+v", got)
		}
	})

	t.Run("http 503 does not rotate", func(t *testing.T) {
		got := detector.Detect("opencode", "Usage limit reached. Your limit will reset at 4:05 PM.", 503)
		if got.Exhausted {
			t.Fatalf("Detect exhausted = true, want false for 503: %+v", got)
		}
	})
}
