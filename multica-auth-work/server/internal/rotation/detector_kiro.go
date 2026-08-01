package rotation

import "regexp"

// Reactive exhaustion detector for Kiro (Amazon Q Developer CLI fork).
//
// Ported from the reference Python detector (AOP control-plane/rotation/detector.py,
// DEFAULT_PATTERNS) per doc 36 §2.1. Kiro is an Amazon Q fork that surfaces quota
// limits both from its own credit/plan model ("Credits (N of M covered in plan)",
// "resets on YYYY-MM-DD" — see kiroUsageParser for the proactive side) and from the
// Claude/Bedrock runtime it wraps ("usage limit reached", "5-hour limit reached").
// This matcher covers both shapes.
//
// Detection requires BOTH a limit phrase AND a reset/retry indicator, mirroring the
// codex/antigravity matchers in detector.go, to avoid false positives on normal
// agent output. The reset-time itself is parsed by Detector.parseResetAt (clock
// shapes) and falls back to the 5h window when only a date/relative hint is present.
//
// Patterns are researched best-effort phrases (doc 36 §2.1: "confirmar contra a tela
// real no deploy"). Override at deploy time via the env pattern override mechanism.
// No credential material is inspected — only public vendor screen text and HTTP
// status, so no secret can leak into logs.

// matchesKiroExhaustion reports whether screenText carries a Kiro quota-exhaustion
// banner. It is the dedicated replacement for the prior kiro→claude pattern mapping.
func matchesKiroExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return kiroLimitPattern.MatchString(screenText) && kiroResetPattern.MatchString(screenText)
}

var (
	// kiroLimitPattern matches the exhaustion banner Kiro prints when the active
	// account's quota is hit. Covers Claude/Bedrock passthrough phrases plus
	// Kiro/Amazon-Q credit-model phrases.
	kiroLimitPattern = regexp.MustCompile(
		`(?i)\b(?:` +
			`usage\s+limit\s+reached|` +
			`5[\s-]?hour\s+limit\s+reached|` +
			`hit\s+your\s+limit\s+for\s+(?:claude|kiro)|` +
			`reached\s+(?:your\s+)?(?:kiro|amazon\s+q)\s+(?:usage\s+)?limit|` +
			`kiro\s+credits\s+(?:are\s+)?(?:exhausted|depleted)|` +
			`out\s+of\s+kiro\s+credits` +
			`)\b`,
	)
	// kiroResetPattern matches the reset/retry indicator that accompanies the
	// banner ("reset at 2pm", "resets on 2026-07-10", "try again", "resume").
	kiroResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|resume[s]?|retry|refreshes?)\b`,
	)
)
