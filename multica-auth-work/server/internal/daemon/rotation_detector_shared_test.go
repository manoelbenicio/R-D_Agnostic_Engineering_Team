package daemon

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestDetectExhaustionShared covers the shared reactive flow: HTTP 429, 503,
// "high traffic" screen, vendor screen match (with reset parse), nil matcher,
// and non-matching screen.
func TestDetectExhaustionShared(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	matcher := func(s string) bool { return matchesKiroScreenExhaustion(s) } // reuse a real matcher

	tests := []struct {
		name       string
		matcher    vendorScreenMatcher
		vendor     string
		screenText string
		httpStatus int
		wantExh    bool
		wantSignal rotation.ExhaustionSignal
		wantReset  bool
	}{
		{
			name:    "http 429 triggers with clock reset",
			matcher: matcher, vendor: "kiro",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			httpStatus: 429, wantExh: true, wantSignal: rotation.SignalHTTP429, wantReset: true,
		},
		{
			name:    "http 429 with no reset hint yields nil reset",
			matcher: matcher, vendor: "kiro",
			screenText: "", httpStatus: 429,
			wantExh: true, wantSignal: rotation.SignalHTTP429, wantReset: false,
		},
		{
			name:    "http 503 does not rotate even with banner",
			matcher: matcher, vendor: "kiro",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			httpStatus: 503, wantExh: false, wantSignal: "", wantReset: false,
		},
		{
			name:    "high traffic screen does not rotate",
			matcher: matcher, vendor: "kiro",
			screenText: "We are experiencing high traffic. Please try again later.",
			httpStatus: 0, wantExh: false, wantSignal: "", wantReset: false,
		},
		{
			name:    "vendor screen match triggers screen signal with reset",
			matcher: matcher, vendor: "kiro",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			httpStatus: 0, wantExh: true, wantSignal: rotation.SignalScreen, wantReset: true,
		},
		{
			name:    "non-matching screen yields nothing",
			matcher: matcher, vendor: "kiro",
			screenText: "Refactoring the auth module.", httpStatus: 0,
			wantExh: false, wantSignal: "", wantReset: false,
		},
		{
			name:    "nil matcher yields nothing on screen",
			matcher: nil, vendor: "kiro",
			screenText: "Usage limit reached. Your limit will reset at 2pm.",
			httpStatus: 0, wantExh: false, wantSignal: "", wantReset: false,
		},
		{
			name:    "nil matcher still honors http 429",
			matcher: nil, vendor: "cline",
			screenText: "", httpStatus: 429,
			wantExh: true, wantSignal: rotation.SignalHTTP429, wantReset: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectExhaustion(tt.matcher, tt.vendor, tt.screenText, tt.httpStatus, now)
			if got.Exhausted != tt.wantExh {
				t.Fatalf("Exhausted = %v, want %v (%+v)", got.Exhausted, tt.wantExh, got)
			}
			if got.Signal != tt.wantSignal {
				t.Fatalf("Signal = %q, want %q", got.Signal, tt.wantSignal)
			}
			if tt.wantReset && got.ResetAt == nil {
				t.Fatalf("ResetAt = nil, want a parsed time")
			}
			if !tt.wantReset && got.ResetAt != nil {
				t.Fatalf("ResetAt = %v, want nil", got.ResetAt)
			}
		})
	}
}

// TestParseExhaustionResetAt covers clock-shaped reset parsing: AM/PM, minute
// optional, roll-to-tomorrow, and non-clock hints → nil.
func TestParseExhaustionResetAt(t *testing.T) {
	now := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		text   string
		wantH  int
		wantM  int
		wantOK bool
	}{
		{name: "pm with minutes", text: "Your limit will reset at 3:51 PM.", wantH: 15, wantM: 51, wantOK: true},
		{name: "pm without minutes", text: "resets at 2pm", wantH: 14, wantM: 0, wantOK: true},
		{name: "am midnight", text: "try again at 12am", wantH: 0, wantM: 0, wantOK: true},
		{name: "pm noon stays 12", text: "resume at 12pm", wantH: 12, wantM: 0, wantOK: true},
		{name: "relative hint -> nil", text: "try again in 30 minutes", wantOK: false},
		{name: "date hint -> nil", text: "resets on 2026-07-10", wantOK: false},
		{name: "no hint -> nil", text: "Refactoring the auth module.", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExhaustionResetAt(tt.text, now)
			if !tt.wantOK {
				if got != nil {
					t.Fatalf("ResetAt = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ResetAt = nil, want %02d:%02d", tt.wantH, tt.wantM)
			}
			if got.Hour() != tt.wantH || got.Minute() != tt.wantM {
				t.Fatalf("ResetAt = %s, want %02d:%02d", got.Format(time.RFC3339), tt.wantH, tt.wantM)
			}
		})
	}
}

// TestParseExhaustionResetAtRollsToTomorrow ensures a reset time earlier than
// now rolls forward one day (next occurrence after now, doc 36 §2.1).
func TestParseExhaustionResetAtRollsToTomorrow(t *testing.T) {
	now := time.Date(2026, 7, 6, 23, 50, 0, 0, time.UTC)
	got := parseExhaustionResetAt("Your limit will reset at 1:05 AM.", now)
	if got == nil {
		t.Fatal("ResetAt = nil, want parsed time")
	}
	want := time.Date(2026, 7, 7, 1, 5, 0, 0, now.Location())
	if !got.Equal(want) {
		t.Fatalf("ResetAt = %s, want %s (next day)", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
