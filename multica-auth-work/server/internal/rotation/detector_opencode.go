package rotation

import "regexp"

// Reactive exhaustion detector for OpenCode.
//
// Ported from the reference Python detector (AOP control-plane/rotation/detector.py,
// DEFAULT_PATTERNS) per doc 36 §2.1. OpenCode is an open-source multi-provider coding
// agent runtime; GLM fleet agents that run through the OpenCode-compatible runtime
// reuse the same isolation preparer (see execenv/opencode_home.go). Because OpenCode
// fronts many providers, its exhaustion banner is provider-agnostic: rate/usage/quota
// limit phrases surfaced from the underlying provider.
//
// HTTP 429 is handled upstream in Detector.Detect (before this matcher runs) and
// already triggers exhaustion for every vendor; this matcher adds on-screen detection.
// Transient 503/"high traffic" is likewise handled upstream and does NOT rotate.
//
// Detection requires BOTH a limit phrase AND a reset/retry indicator (mirroring the
// codex/antigravity/kiro/cline matchers) to avoid false positives on normal output.
//
// Patterns are researched best-effort phrases (doc 36 §2.1: "confirmar contra a tela
// real no deploy"). No credential material is inspected — only public vendor screen
// text and HTTP status, so no secret can leak into logs.

// matchesOpenCodeExhaustion reports whether screenText carries an OpenCode
// quota-exhaustion banner (provider passthrough or OpenCode-branded limit message).
func matchesOpenCodeExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return opencodeLimitPattern.MatchString(screenText) && opencodeResetPattern.MatchString(screenText)
}

var (
	// opencodeLimitPattern matches the exhaustion banner OpenCode surfaces when the
	// active provider/account quota is hit (rate/usage/token/quota limit, too-many).
	opencodeLimitPattern = regexp.MustCompile(
		`(?i)\b(?:` +
			`usage\s+limit\s+reached|` +
			`rate\s+limit\s+(?:exceeded|reached)|` +
			`quota\s+(?:exceeded|limit\s+reached)|` +
			`hit\s+your\s+(?:opencode\s+)?limit|` +
			`token\s+limit\s+reached|` +
			`exceeded\s+(?:your\s+)?quota|` +
			`request\s+limit\s+(?:exceeded|reached)|` +
			`too\s+many\s+requests` +
			`)\b`,
	)
	// opencodeResetPattern matches the reset/retry indicator that accompanies the
	// banner ("reset at", "try again", "retry", "cooldown", "back off").
	opencodeResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
