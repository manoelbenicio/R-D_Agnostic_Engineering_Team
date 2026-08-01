package rotation

import "regexp"

// Reactive exhaustion detector for Cline (Cline CLI 2.0).
//
// Ported from the reference Python detector (AOP control-plane/rotation/detector.py,
// DEFAULT_PATTERNS) per doc 36 §2.1. Cline is a multi-provider agentic CLI (OpenRouter,
// Anthropic, OpenAI, …) and a rate-limit product without a published fixed percentage
// ceiling — the proactive clineUsageParser intentionally disables percent detection
// and defers to "the reactive 429 path in detector.go" (see usage.go). This matcher
// adds the reactive SCREEN-text path: when Cline surfaces a provider quota banner in
// the pane, it fires so the rotation service can switch accounts.
//
// HTTP 429 is handled upstream in Detector.Detect (before this matcher runs) and
// already triggers exhaustion for every vendor; this matcher adds on-screen detection.
// Transient 503/"high traffic" is likewise handled upstream and does NOT rotate.
//
// Detection requires BOTH a limit phrase AND a reset/retry indicator (mirroring the
// codex/antigravity/kiro matchers) to avoid false positives on normal agent output.
//
// Patterns are researched best-effort phrases (doc 36 §2.1: "confirmar contra a tela
// real no deploy"). No credential material is inspected — only public vendor screen
// text and HTTP status, so no secret can leak into logs.

// matchesClineExhaustion reports whether screenText carries a Cline quota-exhaustion
// banner (provider passthrough or Cline-branded limit message).
func matchesClineExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return clineLimitPattern.MatchString(screenText) && clineResetPattern.MatchString(screenText)
}

var (
	// clineLimitPattern matches the exhaustion banner Cline surfaces when the active
	// provider/account quota is hit (rate limit, usage limit, quota, too-many-requests).
	clineLimitPattern = regexp.MustCompile(
		`(?i)\b(?:` +
			`usage\s+limit\s+reached|` +
			`rate\s+limit\s+(?:exceeded|reached)|` +
			`quota\s+(?:exceeded|limit\s+reached)|` +
			`hit\s+your\s+(?:cline\s+)?limit|` +
			`request\s+limit\s+(?:exceeded|reached)|` +
			`too\s+many\s+requests|` +
			`monthly\s+(?:usage\s+)?limit\s+(?:reached|exceeded)` +
			`)\b`,
	)
	// clineResetPattern matches the reset/retry indicator that accompanies the banner
	// ("reset at", "try again", "retry", "cooldown", "back off").
	clineResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
