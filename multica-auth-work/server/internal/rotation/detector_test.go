package rotation

import (
	"testing"
	"time"
)

func TestDetectorDetectsVendorExhaustion(t *testing.T) {
	now := time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	detector := &Detector{now: func() time.Time { return now }}

	tests := []struct {
		name       string
		vendor     string
		screenText string
		wantHour   int
		wantMinute int
	}{
		{
			name:       "codex usage limit with reset time",
			vendor:     "codex",
			screenText: "You've hit your usage limit. Please try again at 3:51 PM.",
			wantHour:   15,
			wantMinute: 51,
		},
		{
			name:       "antigravity model quota with reset time",
			vendor:     "antigravity",
			screenText: "You have reached the quota limit for this model. You can resume using this model at 3:36 PM.",
			wantHour:   15,
			wantMinute: 36,
		},
		{
			name:       "kiro usage limit with reset time",
			vendor:     "kiro",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			wantHour:   14,
			wantMinute: 0,
		},
		{
			name:       "opus claude limit with reset time",
			vendor:     "opus",
			screenText: "You've hit your limit for Claude. Usage will reset at 4:05 PM.",
			wantHour:   16,
			wantMinute: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.Detect(tt.vendor, tt.screenText, 0)
			if !got.Exhausted {
				t.Fatal("Detect exhausted = false, want true")
			}
			if got.Signal != SignalScreen {
				t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
			}
			if got.ResetAt == nil {
				t.Fatal("ResetAt = nil, want parsed time")
			}
			if got.ResetAt.Hour() != tt.wantHour || got.ResetAt.Minute() != tt.wantMinute {
				t.Fatalf("ResetAt = %s, want %02d:%02d", got.ResetAt.Format(time.RFC3339), tt.wantHour, tt.wantMinute)
			}
		})
	}
}

func TestDetectorResetTimeOptional(t *testing.T) {
	detector := &Detector{now: func() time.Time {
		return time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	}}

	got := detector.Detect("codex", "You've hit your usage limit. Please try again in 42 minutes.", 0)
	if !got.Exhausted {
		t.Fatal("Detect exhausted = false, want true")
	}
	if got.Signal != SignalScreen {
		t.Fatalf("Signal = %q, want %q", got.Signal, SignalScreen)
	}
	if got.ResetAt != nil {
		t.Fatalf("ResetAt = %s, want nil for relative time", got.ResetAt.Format(time.RFC3339))
	}
}

func TestDetectorHTTP429TriggersExhaustion(t *testing.T) {
	detector := NewExhaustionDetector()

	got := detector.Detect("unknown", "", 429)
	if !got.Exhausted {
		t.Fatal("Detect exhausted = false, want true")
	}
	if got.Signal != SignalHTTP429 {
		t.Fatalf("Signal = %q, want %q", got.Signal, SignalHTTP429)
	}
}

func TestDetectorTransientVendorTrafficDoesNotTrigger(t *testing.T) {
	detector := NewExhaustionDetector()

	tests := []struct {
		name       string
		vendor     string
		screenText string
		httpStatus int
	}{
		{
			name:       "http 503",
			vendor:     "codex",
			screenText: "You've hit your usage limit. Please try again at 3:51 PM.",
			httpStatus: 503,
		},
		{
			name:       "high traffic screen",
			vendor:     "antigravity",
			screenText: "We are experiencing high traffic. Please try again later.",
			httpStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.Detect(tt.vendor, tt.screenText, tt.httpStatus)
			if got.Exhausted {
				t.Fatalf("Detect exhausted = true, want false: %+v", got)
			}
			if got.Signal != "" {
				t.Fatalf("Signal = %q, want empty", got.Signal)
			}
			if got.ResetAt != nil {
				t.Fatalf("ResetAt = %s, want nil", got.ResetAt.Format(time.RFC3339))
			}
		})
	}
}

func TestDetectorUnknownVendorDoesNotTriggerScreenDetection(t *testing.T) {
	detector := NewExhaustionDetector()

	got := detector.Detect("unknown", "Usage limit reached. Your limit will reset at 2pm.", 0)
	if got.Exhausted {
		t.Fatalf("Detect exhausted = true, want false: %+v", got)
	}
}

func TestDetectorRequiresVendorSpecificPhrases(t *testing.T) {
	detector := NewExhaustionDetector()

	tests := []struct {
		name       string
		vendor     string
		screenText string
	}{
		{
			name:       "codex missing try again",
			vendor:     "codex",
			screenText: "You've hit your usage limit.",
		},
		{
			name:       "antigravity missing resume phrase",
			vendor:     "antigravity",
			screenText: "Reached the quota limit for this model.",
		},
		{
			name:       "kiro missing reset",
			vendor:     "kiro",
			screenText: "5-hour limit reached.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.Detect(tt.vendor, tt.screenText, 0)
			if got.Exhausted {
				t.Fatalf("Detect exhausted = true, want false: %+v", got)
			}
		})
	}
}
