package rotation

import (
	"testing"
	"time"
)

func TestWarningDetectorCodexPercentBanner(t *testing.T) {
	d := NewWarningDetector()
	d.now = func() time.Time {
		return time.Date(2026, 7, 1, 18, 30, 0, 0, time.UTC)
	}

	approaching, percentLeft, resetAt := d.DetectWarning("codex", "Heads up: less than 10% of your 5h limit left; resets 19:59.")
	if !approaching {
		t.Fatal("approaching = false, want true")
	}
	if percentLeft != 10 {
		t.Fatalf("percentLeft = %d, want 10", percentLeft)
	}
	wantReset := time.Date(2026, 7, 1, 19, 59, 0, 0, time.UTC)
	if resetAt == nil || !resetAt.Equal(wantReset) {
		t.Fatalf("resetAt = %v, want %v", resetAt, wantReset)
	}
}

func TestWarningDetectorNormalText(t *testing.T) {
	approaching, percentLeft, resetAt := NewWarningDetector().DetectWarning("codex", "Working on the task now.")
	if approaching || percentLeft != 0 || resetAt != nil {
		t.Fatalf("DetectWarning = %v,%d,%v; want false,0,nil", approaching, percentLeft, resetAt)
	}
}

func TestWarningDetectorIgnoresReactiveLimitReached(t *testing.T) {
	approaching, percentLeft, resetAt := NewWarningDetector().DetectWarning("codex", "usage limit reached; try again later")
	if approaching || percentLeft != 0 || resetAt != nil {
		t.Fatalf("DetectWarning = %v,%d,%v; want false,0,nil", approaching, percentLeft, resetAt)
	}
}

func TestWarningDetectorUnknownVendor(t *testing.T) {
	approaching, percentLeft, resetAt := NewWarningDetector().DetectWarning("unknown", "less than 10% of your 5h limit left; resets 19:59")
	if approaching || percentLeft != 0 || resetAt != nil {
		t.Fatalf("DetectWarning = %v,%d,%v; want false,0,nil", approaching, percentLeft, resetAt)
	}
}
