package rotation

import (
	"testing"
	"time"
)

func TestProactiveDetectorShouldRotateAtNinetyFivePercent(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	now := windowStart.Add(2 * time.Hour)

	got := NewProactiveDetector(0.95).ShouldRotate(Account{
		TokensUsed:   95,
		TokensPerWin: 100,
		WindowStart:  &windowStart,
	}, now)

	if !got.Exhausted {
		t.Fatal("ShouldRotate exhausted = false, want true")
	}
	if got.Signal != SignalLedger {
		t.Fatalf("Signal = %s, want %s", got.Signal, SignalLedger)
	}
	wantReset := windowStart.Add(5 * time.Hour)
	if got.ResetAt == nil || !got.ResetAt.Equal(wantReset) {
		t.Fatalf("ResetAt = %v, want %v", got.ResetAt, wantReset)
	}
}

func TestProactiveDetectorShouldNotRotateAtHalfWindow(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	got := NewProactiveDetector(0.95).ShouldRotate(Account{
		TokensUsed:   50,
		TokensPerWin: 100,
		WindowStart:  &windowStart,
	}, windowStart.Add(time.Hour))

	if got.Exhausted {
		t.Fatalf("ShouldRotate exhausted = true, want false: %+v", got)
	}
}

func TestProactiveDetectorShouldNotRotateAfterWindowExpired(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	got := NewProactiveDetector(0.95).ShouldRotate(Account{
		TokensUsed:   100,
		TokensPerWin: 100,
		WindowStart:  &windowStart,
	}, windowStart.Add(5*time.Hour))

	if got.Exhausted {
		t.Fatalf("ShouldRotate exhausted = true, want false after reset: %+v", got)
	}
}

func TestProactiveDetectorShouldNotRotateWithUnknownTokenWindow(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	got := NewProactiveDetector(0.95).ShouldRotate(Account{
		TokensUsed:   100,
		TokensPerWin: 0,
		WindowStart:  &windowStart,
	}, windowStart.Add(time.Hour))

	if got.Exhausted {
		t.Fatalf("ShouldRotate exhausted = true, want false for unknown window: %+v", got)
	}
}

func TestProactiveDetectorInvalidThresholdUsesDefault(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	detector := NewProactiveDetector(2)

	got := detector.ShouldRotate(Account{
		TokensUsed:   94,
		TokensPerWin: 100,
		WindowStart:  &windowStart,
	}, windowStart.Add(time.Hour))
	if got.Exhausted {
		t.Fatalf("94%% exhausted = true, want false with default threshold: %+v", got)
	}

	got = detector.ShouldRotate(Account{
		TokensUsed:   95,
		TokensPerWin: 100,
		WindowStart:  &windowStart,
	}, windowStart.Add(time.Hour))
	if !got.Exhausted {
		t.Fatalf("95%% exhausted = false, want true with default threshold: %+v", got)
	}
}

func TestAccountsNeedingProactiveRotationOrdersByPriority(t *testing.T) {
	windowStart := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	got := AccountsNeedingProactiveRotation([]Account{
		{AccountID: "fallback", Priority: 30, TokensUsed: 95, TokensPerWin: 100, WindowStart: &windowStart},
		{AccountID: "codex", Priority: 10, TokensUsed: 95, TokensPerWin: 100, WindowStart: &windowStart},
		{AccountID: "safe", Priority: 1, TokensUsed: 50, TokensPerWin: 100, WindowStart: &windowStart},
	}, windowStart.Add(time.Hour), NewProactiveDetector(0.95))

	if len(got) != 2 {
		t.Fatalf("accounts needing rotation = %d, want 2", len(got))
	}
	if got[0].AccountID != "codex" || got[1].AccountID != "fallback" {
		t.Fatalf("order = %s,%s; want codex,fallback", got[0].AccountID, got[1].AccountID)
	}
}
