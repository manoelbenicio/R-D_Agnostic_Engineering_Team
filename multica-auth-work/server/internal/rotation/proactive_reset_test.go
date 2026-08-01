package rotation

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeResetClaimer is a test-only ResetClaimer (no real CLI/network).
type fakeResetClaimer struct {
	claimed bool
	err     error
	calls   int
	lastAcc Account
}

func (f *fakeResetClaimer) ClaimReset(_ context.Context, acc Account) (bool, error) {
	f.calls++
	f.lastAcc = acc
	return f.claimed, f.err
}

func TestEvaluateProactiveEmptyTextIsNotApproaching(t *testing.T) {
	sig := EvaluateProactive("codex", "", time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	if sig.Approaching {
		t.Fatalf("Approaching = true, want false for empty text: %+v", sig)
	}
	if sig.Source != "" {
		t.Fatalf("Source = %q, want empty", sig.Source)
	}
	if sig.ResetsAvailable != 0 {
		t.Fatalf("ResetsAvailable = %d, want 0", sig.ResetsAvailable)
	}
}

func TestEvaluateProactiveCodexUsagePanelApproaching(t *testing.T) {
	// Codex /usage panel: 5h limit at 5% left (<= 10% threshold) with 2 resets
	// available, resetting at 14:30. The "account" line makes ParseCodexUsage
	// mark the panel as recognized (Raw != "").
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	panel := "5h limit: 5% left (resets 14:30)\n" +
		"weekly limit: 80% left (resets 17:00 on 7 Jul)\n" +
		"you have 2 usage limit resets available\n" +
		"account: dev@multica.ai (codex)"

	sig := EvaluateProactive("codex", panel, now)
	if !sig.Approaching {
		t.Fatalf("Approaching = false, want true: %+v", sig)
	}
	if sig.Source != proactiveSourceCodexUsage {
		t.Fatalf("Source = %q, want %q", sig.Source, proactiveSourceCodexUsage)
	}
	if sig.ResetsAvailable != 2 {
		t.Fatalf("ResetsAvailable = %d, want 2", sig.ResetsAvailable)
	}
}

func TestEvaluateProactiveCodexUsagePanelWeeklyApproaching(t *testing.T) {
	// 5h window healthy (60% left) but weekly limit at 8% left (<= 10%).
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	panel := "5h limit: 60% left (resets 14:30)\n" +
		"weekly limit: 8% left (resets 17:00 on 7 Jul)\n" +
		"account: dev@multica.ai (codex)"

	sig := EvaluateProactive("codex", panel, now)
	if !sig.Approaching {
		t.Fatalf("Approaching = false, want true on weekly limit: %+v", sig)
	}
	if sig.Source != proactiveSourceCodexUsage {
		t.Fatalf("Source = %q, want %q", sig.Source, proactiveSourceCodexUsage)
	}
}

func TestEvaluateProactiveCodexUsagePanelNotApproachingStillReportsResets(t *testing.T) {
	// Healthy panel (50% / 70% left) — not approaching, but resets still parsed.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	panel := "5h limit: 50% left (resets 14:30)\n" +
		"weekly limit: 70% left (resets 17:00 on 7 Jul)\n" +
		"you have 1 usage limit resets available\n" +
		"account: dev@multica.ai (codex)"

	sig := EvaluateProactive("codex", panel, now)
	if sig.Approaching {
		t.Fatalf("Approaching = true, want false on healthy panel: %+v", sig)
	}
	if sig.ResetsAvailable != 1 {
		t.Fatalf("ResetsAvailable = %d, want 1 (parsed even when not approaching)", sig.ResetsAvailable)
	}
}

func TestEvaluateProactiveUsagePanelApproaching(t *testing.T) {
	// Codex status-line usage: "5h limit: 4% left" — UsageDetector flags
	// Approaching (PercentRemaining <= 10). No "account:" line so ParseCodexUsage
	// does not recognize a full panel and EvaluateProactive falls through to the
	// usage detector.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	sig := EvaluateProactive("codex", "Status: 5h limit: 4% left", now)
	if !sig.Approaching {
		t.Fatalf("Approaching = false, want true: %+v", sig)
	}
	if sig.Source != proactiveSourceUsagePanel {
		t.Fatalf("Source = %q, want %q", sig.Source, proactiveSourceUsagePanel)
	}
}

func TestEvaluateProactiveWarnBannerApproaching(t *testing.T) {
	// Heads-up banner: "less than 5% of your 5h limit left" with no usage panel
	// lines and no account line → codex probe + usage detector miss, warning
	// banner fires.
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	sig := EvaluateProactive("codex", "Heads up — less than 5% of your 5h limit left (resets 14:30)", now)
	if !sig.Approaching {
		t.Fatalf("Approaching = false, want true: %+v", sig)
	}
	if sig.Source != proactiveSourceWarnBanner {
		t.Fatalf("Source = %q, want %q", sig.Source, proactiveSourceWarnBanner)
	}
}

func TestEvaluateProactiveUnknownVendorIsNotApproaching(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	sig := EvaluateProactive("unknownvendor", "some random text with no quota signal", now)
	if sig.Approaching {
		t.Fatalf("Approaching = true, want false for unknown vendor: %+v", sig)
	}
}

func TestDecideProactiveNotApproachingIsNone(t *testing.T) {
	got := DecideProactive(ProactiveSignal{Approaching: false}, NoopResetClaimer{}, Account{AccountID: "a1"}, context.Background())
	if got != DecisionNone {
		t.Fatalf("decision = %s, want NONE", got)
	}
}

func TestDecideProactiveApproachingWithoutResetsRotates(t *testing.T) {
	got := DecideProactive(ProactiveSignal{Approaching: true, ResetsAvailable: 0}, NoopResetClaimer{}, Account{AccountID: "a1"}, context.Background())
	if got != DecisionRotate {
		t.Fatalf("decision = %s, want ROTATE", got)
	}
}

func TestDecideProactiveApproachingWithResetsNoopRotates(t *testing.T) {
	// Approaching + 2 resets available + NoopResetClaimer → claim returns
	// (false, nil) → ROTATE. A real claimer would KEEP; the no-op forces ROTATE
	// until a headless claim mechanism is confirmed (gated, design.md §7 (2)).
	got := DecideProactive(ProactiveSignal{Approaching: true, ResetsAvailable: 2}, NoopResetClaimer{}, Account{AccountID: "a1"}, context.Background())
	if got != DecisionRotate {
		t.Fatalf("decision = %s, want ROTATE (no-op claim must fall back)", got)
	}
}

func TestDecideProactiveClaimKeepsAccount(t *testing.T) {
	// Fake claimer returns (true, nil) → KEEP. Proves the KEEP path works and
	// would be taken once a real headless claimer is wired in.
	claimer := &fakeResetClaimer{claimed: true}
	acc := Account{AccountID: "a1", Vendor: "codex"}
	got := DecideProactive(ProactiveSignal{Approaching: true, ResetsAvailable: 1}, claimer, acc, context.Background())
	if got != DecisionKeep {
		t.Fatalf("decision = %s, want KEEP", got)
	}
	if claimer.calls != 1 {
		t.Fatalf("claimer calls = %d, want 1", claimer.calls)
	}
	if claimer.lastAcc.AccountID != "a1" {
		t.Fatalf("claimer lastAcc.AccountID = %q, want a1", claimer.lastAcc.AccountID)
	}
}

func TestDecideProactiveClaimErrorRotates(t *testing.T) {
	// Claimer returns an error → cannot claim → ROTATE (do not keep on error).
	claimer := &fakeResetClaimer{claimed: true, err: errors.New("rpc unavailable")}
	got := DecideProactive(ProactiveSignal{Approaching: true, ResetsAvailable: 3}, claimer, Account{AccountID: "a1"}, context.Background())
	if got != DecisionRotate {
		t.Fatalf("decision = %s, want ROTATE on claim error", got)
	}
}

func TestDecideProactiveNilClaimerRotates(t *testing.T) {
	// Defensive: nil claimer must not panic; treat as cannot claim → ROTATE.
	got := DecideProactive(ProactiveSignal{Approaching: true, ResetsAvailable: 1}, nil, Account{AccountID: "a1"}, context.Background())
	if got != DecisionRotate {
		t.Fatalf("decision = %s, want ROTATE for nil claimer", got)
	}
}

func TestNoopResetClaimerClaimResetReturnsFalseNil(t *testing.T) {
	// GATED contract: NoopResetClaimer.ClaimReset always returns (false, nil).
	claimed, err := NoopResetClaimer{}.ClaimReset(context.Background(), Account{AccountID: "a1"})
	if claimed {
		t.Fatalf("claimed = true, want false (no-op is gated)")
	}
	if err != nil {
		t.Fatalf("err = %v, want nil (no-op never errors)", err)
	}
}

func TestDecisionString(t *testing.T) {
	cases := map[Decision]string{
		DecisionNone:   "NONE",
		DecisionKeep:   "KEEP",
		DecisionRotate: "ROTATE",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Fatalf("Decision(%d).String() = %q, want %q", d, got, want)
		}
	}
}
