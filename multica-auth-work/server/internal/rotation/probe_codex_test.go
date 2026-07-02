package rotation

import (
	"math"
	"testing"
	"time"
)

func TestParseCodexUsageRealPanel(t *testing.T) {
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	panel := `Account:              operator@example.com (Plus)
5h limit:             [████░] 96% left (resets 12:51)
Weekly limit:         [████] 98% left (resets 16:32 on 8 Jul)
Context window:       91% left (33.4K used / 258K)
You have 2 usage limit resets available.`

	got := ParseCodexUsage(panel, now)

	assertCodexUsageFloat(t, got.FiveHourPercentLeft, 96)
	assertCodexUsageTime(t, got.FiveHourResetAt, time.Date(2026, 7, 2, 12, 51, 0, 0, time.UTC))
	assertCodexUsageFloat(t, got.WeeklyPercentLeft, 98)
	assertCodexUsageTime(t, got.WeeklyResetAt, time.Date(2026, 7, 8, 16, 32, 0, 0, time.UTC))
	if got.ResetsAvailable != 2 {
		t.Fatalf("ResetsAvailable = %d, want 2", got.ResetsAvailable)
	}
	if got.Account != "operator@example.com" {
		t.Fatalf("Account = %q, want parsed email", got.Account)
	}
	if got.Raw != panel {
		t.Fatal("Raw does not preserve recognized panel text")
	}
}

func TestParseCodexUsageFiveHourResetTomorrow(t *testing.T) {
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)

	got := ParseCodexUsage("5h limit: 5% left (resets 08:00)", now)

	assertCodexUsageFloat(t, got.FiveHourPercentLeft, 5)
	assertCodexUsageTime(t, got.FiveHourResetAt, time.Date(2026, 7, 3, 8, 0, 0, 0, time.UTC))
}

func TestParseCodexUsageWeeklyResetNextYear(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)

	got := ParseCodexUsage("Weekly limit: [████] 98% left (resets 16:32 on 8 Jul)", now)

	assertCodexUsageFloat(t, got.WeeklyPercentLeft, 98)
	assertCodexUsageTime(t, got.WeeklyResetAt, time.Date(2027, 7, 8, 16, 32, 0, 0, time.UTC))
}

func TestParseCodexUsageMenuResetsLine(t *testing.T) {
	panel := "Redeem usage limit reset  You have 3 usage limit resets available."

	got := ParseCodexUsage(panel, time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC))

	if got.ResetsAvailable != 3 {
		t.Fatalf("ResetsAvailable = %d, want 3", got.ResetsAvailable)
	}
	if got.Raw != panel {
		t.Fatal("Raw does not preserve recognized menu text")
	}
}

func TestParseCodexUsageMissingResetsLineDefaultsZero(t *testing.T) {
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)

	got := ParseCodexUsage("5h limit: [████░] 96% left (resets 12:51)", now)

	if got.ResetsAvailable != 0 {
		t.Fatalf("ResetsAvailable = %d, want 0", got.ResetsAvailable)
	}
	assertCodexUsageFloat(t, got.FiveHourPercentLeft, 96)
}

func TestParseCodexUsageIgnoresContextWindow(t *testing.T) {
	got := ParseCodexUsage("Context window:       91% left (33.4K used / 258K)", time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC))

	if got != (CodexUsage{}) {
		t.Fatalf("got %+v, want zero-value usage", got)
	}
}

func TestParseCodexUsageEmptyOrUnrelatedText(t *testing.T) {
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	for _, text := range []string{"", "not a codex usage panel"} {
		t.Run(text, func(t *testing.T) {
			got := ParseCodexUsage(text, now)
			if got != (CodexUsage{}) {
				t.Fatalf("got %+v, want zero-value usage", got)
			}
		})
	}
}

func TestParseCodexUsageIsCaseAndSpacingTolerant(t *testing.T) {
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	panel := "  weekly   limit:   [░░]  12.5%   LEFT   ( resets 07:05 on 9 JUL )\nYOU HAVE 4 USAGE LIMIT RESETS AVAILABLE"

	got := ParseCodexUsage(panel, now)

	assertCodexUsageFloat(t, got.WeeklyPercentLeft, 12.5)
	assertCodexUsageTime(t, got.WeeklyResetAt, time.Date(2026, 7, 9, 7, 5, 0, 0, time.UTC))
	if got.ResetsAvailable != 4 {
		t.Fatalf("ResetsAvailable = %d, want 4", got.ResetsAvailable)
	}
}

func assertCodexUsageFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func assertCodexUsageTime(t *testing.T, got *time.Time, want time.Time) {
	t.Helper()
	if got == nil {
		t.Fatalf("got nil time, want %v", want)
	}
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
