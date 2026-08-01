package rotation

import (
	"context"
	"strings"
	"time"
)

// proactiveApproachingPercent is the percent-remaining ceiling at which a vendor
// account is considered "approaching" its hard-stop. It mirrors the usage
// detector's default threshold (defaultUsageThreshold = 0.10 → 10%).
const proactiveApproachingPercent = 10.0

// Sources recorded on ProactiveSignal.Source to identify which read-only
// detector fired. Deterministic and ordered: the Codex usage panel is checked
// first (it is the richest source and the only one that carries
// ResetsAvailable), then the multi-vendor usage panel, then the warning banner.
const (
	proactiveSourceCodexUsage = "codex_usage"
	proactiveSourceUsagePanel = "usage_panel"
	proactiveSourceWarnBanner = "warn_banner"
)

// ProactiveSignal is the outcome of reading a vendor's quota surface
// (banner / usage panel / Codex probe) to decide whether the account is
// approaching its hard-stop before any request fails. See design.md §7 (1).
type ProactiveSignal struct {
	// Approaching is true when any read-only detector reports the account is
	// near its quota limit.
	Approaching bool
	// Source identifies which detector set Approaching (one of the
	// proactiveSource* constants). Empty when Approaching is false.
	Source string
	// ResetsAvailable is the number of "usage limit resets available" reported
	// by the Codex /usage panel (you have N usage limit resets available).
	// Only the Codex probe parses this today; other vendors report 0.
	ResetsAvailable int
}

// EvaluateProactive inspects the vendor screen/panel text with the read-only
// WarningDetector, UsageDetector and ParseCodexUsage helpers and reports
// whether the account is approaching its quota hard-stop. It invents nothing:
// all parsing is delegated to the existing detectors. The decision is
// deterministic given (vendor, screenText, now).
//
// Detection order (first hit wins, richest source first):
//  1. ParseCodexUsage — also extracts ResetsAvailable.
//  2. UsageDetector.Detect — multi-vendor panel parser.
//  3. WarningDetector.DetectWarning — heads-up / "less than N% left" banner.
func EvaluateProactive(vendor, screenText string, now time.Time) ProactiveSignal {
	sig := ProactiveSignal{}
	text := strings.TrimSpace(screenText)
	if text == "" {
		return sig
	}

	// 1. Codex /usage panel — richest source; carries ResetsAvailable.
	codex := ParseCodexUsage(screenText, now)
	if codex.Raw != "" {
		sig.ResetsAvailable = codex.ResetsAvailable
		if codex.FiveHourPercentLeft > 0 && codex.FiveHourPercentLeft <= proactiveApproachingPercent {
			sig.Approaching = true
			sig.Source = proactiveSourceCodexUsage
			return sig
		}
		if codex.WeeklyPercentLeft > 0 && codex.WeeklyPercentLeft <= proactiveApproachingPercent {
			sig.Approaching = true
			sig.Source = proactiveSourceCodexUsage
			return sig
		}
	}

	// 2. Warning banner (heads-up / "less than N% of your 5h limit left").
	// Checked before the generic usage panel because the banner is the
	// dedicated proactive-signal surface (design.md §7): "Heads up — less than
	// N% ... left" is a banner, not a panel line, and must surface as
	// warn_banner. (The usage detector also matches the "less than N% of your
	// 5h limit left" substring, so checking the banner first wins for that
	// text; genuine panel lines like "5h limit: N% left" do not match the
	// banner and fall through to the usage detector below.)
	if approaching, _, _ := NewWarningDetector().DetectWarning(vendor, screenText); approaching {
		sig.Approaching = true
		sig.Source = proactiveSourceWarnBanner
		return sig
	}

	// 3. Usage detector (multi-vendor panel parser). Threshold 0 selects the
	// documented default (10% remaining).
	for _, sample := range NewUsageDetector(0).Detect(vendor, screenText) {
		if sample.Approaching {
			sig.Approaching = true
			sig.Source = proactiveSourceUsagePanel
			return sig
		}
	}

	return sig
}

// Decision is the proactive-rotation verdict for one account.
type Decision int

const (
	// DecisionNone means no proactive action is needed (account not approaching).
	DecisionNone Decision = iota
	// DecisionKeep means the account should be kept: a reset was claimed and the
	// quota window refreshed, so rotation is unnecessary (zero context switch).
	DecisionKeep
	// DecisionRotate means the account is approaching its hard-stop and no reset
	// could be claimed, so the router should fall back to the next account.
	DecisionRotate
)

// String renders the decision for logs and test diagnostics.
func (d Decision) String() string {
	switch d {
	case DecisionKeep:
		return "KEEP"
	case DecisionRotate:
		return "ROTATE"
	default:
		return "NONE"
	}
}

// ResetClaimer claims a vendor "usage limit reset" for an approaching account
// so it can be kept instead of rotated (design.md §7 (2)). The only contract is
// that a successful claim returns (true, nil): the account's quota window has
// been refreshed and the caller may continue using it.
//
// GATING NOTE: the Codex "/usage" screen is TUI-only; a headless claim
// mechanism (e.g. an app-server RPC) is NOT confirmed against the binary yet —
// "CONFIRMAR CONTRA BINÁRIO". Until a headless path is verified, the router
// uses NoopResetClaimer (below) which never claims, forcing ROTATE.
type ResetClaimer interface {
	ClaimReset(ctx context.Context, acc Account) (bool, error)
}

// NoopResetClaimer is the GATED default ResetClaimer. ClaimReset always returns
// (false, nil): it never claims a reset.
//
// Rationale (design.md §7 (2), [DEP]): "/usage is TUI-only; headless claim
// mechanism CONFIRMAR CONTRA BINÁRIO (app-server RPC?)". Because the headless
// claim path is unconfirmed, inventing a claim command here would be unsafe.
// With the no-op, DecideProactive treats "resets available but not claimable"
// as ROTATE — rotation becomes the fallback, exactly as designed. A real
// claimer is deferred until the headless mechanism is confirmed.
type NoopResetClaimer struct{}

// ClaimReset implements ResetClaimer. GATED: always returns (false, nil).
func (NoopResetClaimer) ClaimReset(_ context.Context, _ Account) (bool, error) {
	return false, nil
}

// DecideProactive turns a ProactiveSignal into a concrete Decision, consulting
// the ResetClaimer first when resets are available (design.md §7 (2)):
//
//   - Not approaching                       → NONE
//   - Approaching with ResetsAvailable > 0  → try claimer.ClaimReset first:
//        claimed (true, nil)                → KEEP  (zero context switch)
//        not claimed (false/err)            → ROTATE (fallback)
//   - Approaching with no resets available  → ROTATE
//
// A nil claimer is treated as "cannot claim" → ROTATE, so callers never panic.
// With NoopResetClaimer the claim path always yields ROTATE until a headless
// claim mechanism is confirmed against the binary.
func DecideProactive(sig ProactiveSignal, claimer ResetClaimer, acc Account, ctx context.Context) Decision {
	if !sig.Approaching {
		return DecisionNone
	}
	if sig.ResetsAvailable > 0 {
		if claimer == nil {
			return DecisionRotate
		}
		claimed, err := claimer.ClaimReset(ctx, acc)
		if err != nil || !claimed {
			return DecisionRotate
		}
		return DecisionKeep
	}
	return DecisionRotate
}
