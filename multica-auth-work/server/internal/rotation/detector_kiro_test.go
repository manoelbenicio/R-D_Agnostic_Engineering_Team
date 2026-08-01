package rotation

import (
	"testing"
	"time"
)

// TestKiroMatcherDetectsExhaustion covers the dedicated Kiro matcher directly:
// Claude/Bedrock passthrough phrases plus Kiro/Amazon-Q credit-model phrases,
// each paired with a reset/retry indicator.
func TestKiroMatcherDetectsExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		screenText string
		want       bool
	}{
		{
			name:       "claude passthrough usage limit with reset",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			want:       true,
		},
		{
			name:       "kiro usage limit with date reset",
			screenText: "You've reached your Kiro usage limit. Credits reset on 2026-07-10.",
			want:       true,
		},
		{
			name:       "kiro credits exhausted with try again",
			screenText: "Out of Kiro credits. Try again after the plan refreshes.",
			want:       true,
		},
		{
			name:       "5h limit reached with reset",
			screenText: "5-hour limit reached. Resets 6am.",
			want:       true,
		},
		{
			name:       "limit reached without reset indicator",
			screenText: "5-hour limit reached.",
			want:       false,
		},
		{
			name:       "reset indicator without limit phrase",
			screenText: "Your limit will reset at 2pm.",
			want:       false,
		},
		{
			name:       "normal agent output",
			screenText: "Refactoring the auth module and adding tests.",
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
			got := matchesKiroExhaustion(tt.screenText)
			if got != tt.want {
				t.Fatalf("matchesKiroExhaustion(%q) = %v, want %v", tt.screenText, got, tt.want)
			}
		})
	}
}

// TestKiroDetectorViaDetect confirms the Kiro matcher is wired into Detector.Detect
// (screen signal + reset-time parsing) and stays backward-compatible with the prior
// kiro→claude mapping exercised by detector_test.go.
func TestKiroDetectorViaDetect(t *testing.T) {
	now := time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	detector := &Detector{now: func() time.Time { return now }}

	t.Run("clock reset parsed", func(t *testing.T) {
		got := detector.Detect("kiro", "Usage limit reached. Your limit will reset at 2pm.", 0)
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

	t.Run("date reset falls back to nil (5h window)", func(t *testing.T) {
		got := detector.Detect("kiro", "You've reached your Kiro usage limit. Credits reset on 2026-07-10.", 0)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalScreen {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
		}
		if got.ResetAt != nil {
			t.Fatalf("ResetAt = %v, want nil for non-clock reset hint", got.ResetAt)
		}
	})

	t.Run("http 429 still triggers", func(t *testing.T) {
		got := detector.Detect("kiro", "", 429)
		if !got.Exhausted {
			t.Fatal("Detect exhausted = false, want true")
		}
		if got.Signal != SignalHTTP429 {
			t.Fatalf("Signal = %q, want %q", got.Signal, SignalHTTP429)
		}
	})

	t.Run("high traffic does not rotate", func(t *testing.T) {
		got := detector.Detect("kiro", "We are experiencing high traffic. Please try again later.", 0)
		if got.Exhausted {
			t.Fatalf("Detect exhausted = true, want false: %+v", got)
		}
	})
}
