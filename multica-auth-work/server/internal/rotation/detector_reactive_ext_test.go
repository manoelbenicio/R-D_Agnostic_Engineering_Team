package rotation

import (
	"testing"
	"time"
)

func TestReactiveClassifiesCodexHardStopWithResetToday(t *testing.T) {
	now := time.Date(2026, 7, 2, 14, 0, 0, 0, time.UTC)

	got := ClassifyCodexReactive("Codex message usage limit reached\nPlease wait until 15:06", now)

	if !got.Exhausted {
		t.Fatal("Exhausted = false, want true")
	}
	if got.Kind != "hard_stop" {
		t.Fatalf("Kind = %q, want hard_stop", got.Kind)
	}
	if got.ResetAt == nil {
		t.Fatal("ResetAt = nil, want parsed time")
	}
	want := time.Date(2026, 7, 2, 15, 6, 0, 0, time.UTC)
	if !got.ResetAt.Equal(want) {
		t.Fatalf("ResetAt = %s, want %s", got.ResetAt.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	if got.SuspectFalsePositive {
		t.Fatal("SuspectFalsePositive = true, want false")
	}
}

func TestReactiveClassifiesApproachingAsNotExhausted(t *testing.T) {
	got := ClassifyCodexReactive("less than 10% of your 5h limit left", time.Date(2026, 7, 2, 14, 0, 0, 0, time.UTC))

	if got.Exhausted {
		t.Fatal("Exhausted = true, want false")
	}
	if got.Kind != "approaching" {
		t.Fatalf("Kind = %q, want approaching", got.Kind)
	}
	if got.ResetAt != nil {
		t.Fatalf("ResetAt = %s, want nil", got.ResetAt.Format(time.RFC3339))
	}
	if got.SuspectFalsePositive {
		t.Fatal("SuspectFalsePositive = true, want false")
	}
}

func TestReactiveFlagsHardStopWithRemaining5hQuotaAsSuspect(t *testing.T) {
	got := ClassifyCodexReactive("usage limit reached\n5h limit: 83% left", time.Date(2026, 7, 2, 14, 0, 0, 0, time.UTC))

	if !got.Exhausted {
		t.Fatal("Exhausted = false, want true")
	}
	if got.Kind != "hard_stop" {
		t.Fatalf("Kind = %q, want hard_stop", got.Kind)
	}
	if !got.SuspectFalsePositive {
		t.Fatal("SuspectFalsePositive = false, want true")
	}
}

func TestReactiveRollsResetAtToTomorrow(t *testing.T) {
	now := time.Date(2026, 7, 2, 23, 50, 0, 0, time.UTC)

	got := ClassifyCodexReactive("usage limit reached\nPlease wait until 00:30", now)

	if !got.Exhausted {
		t.Fatal("Exhausted = false, want true")
	}
	if got.ResetAt == nil {
		t.Fatal("ResetAt = nil, want parsed time")
	}
	want := time.Date(2026, 7, 3, 0, 30, 0, 0, time.UTC)
	if !got.ResetAt.Equal(want) {
		t.Fatalf("ResetAt = %s, want %s", got.ResetAt.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestReactiveClassifiesNormalTextAsNone(t *testing.T) {
	got := ClassifyCodexReactive("Refactoring the auth module...", time.Date(2026, 7, 2, 14, 0, 0, 0, time.UTC))

	if got.Exhausted {
		t.Fatal("Exhausted = true, want false")
	}
	if got.Kind != "none" {
		t.Fatalf("Kind = %q, want none", got.Kind)
	}
	if got.ResetAt != nil {
		t.Fatalf("ResetAt = %s, want nil", got.ResetAt.Format(time.RFC3339))
	}
	if got.SuspectFalsePositive {
		t.Fatal("SuspectFalsePositive = true, want false")
	}
}
