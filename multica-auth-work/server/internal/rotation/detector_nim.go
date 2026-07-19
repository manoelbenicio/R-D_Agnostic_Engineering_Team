package rotation

import "regexp"

// Reactive exhaustion detector for NVIDIA NIM.
//
// NIM is an OpenAI-compatible inference API
// (https://integrate.api.nvidia.com/v1). The native nim runtime (see
// server/pkg/agent/nim.go, Agent-1 / task 1.1) talks to it over HTTP, so the
// reactive exhaustion path is the OpenAI-compatible 429 / rate-limit / quota
// banner the gateway surfaces. HTTP 429 is handled upstream in Detector.Detect
// (before this matcher runs) and already triggers exhaustion for every vendor;
// this matcher adds on-screen detection for the cases where the gateway wraps the
// 429 in a JSON error body that lands in the agent pane. Transient 503/"high
// traffic" is likewise handled upstream and does NOT rotate.
//
// Detection requires BOTH a limit phrase AND a reset/retry indicator (mirroring
// the codex/antigravity/kiro/cline/opencode matchers) to avoid false positives on
// normal agent output.
//
// Patterns are researched best-effort OpenAI-compatible/NVIDIA phrases
// (doc 36 §2.1: "confirmar contra a tela real no deploy"). No credential material
// is inspected — only public vendor screen text and HTTP status, so no secret can
// leak into logs.

// matchesNimExhaustion reports whether screenText carries a NIM quota-exhaustion
// banner (OpenAI-compatible rate/usage/quota limit or NVIDIA/NIM-branded message).
func matchesNimExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return nimLimitPattern.MatchString(screenText) && nimResetPattern.MatchString(screenText)
}

var (
	// nimLimitPattern matches the exhaustion banner NIM surfaces when the active
	// account's quota is hit (rate/usage/quota limit, too-many-requests, plus
	// NVIDIA/NIM-branded and credits-exhausted messages).
	nimLimitPattern = regexp.MustCompile(
		`(?i)\b(?:` +
			`usage\s+limit\s+reached|` +
			`rate\s+limit\s+(?:exceeded|reached)|` +
			`quota\s+(?:exceeded|limit\s+reached)|` +
			`hit\s+your\s+(?:nim|nvidia)\s+limit|` +
			`request\s+limit\s+(?:exceeded|reached)|` +
			`too\s+many\s+requests|` +
			`monthly\s+(?:usage\s+)?limit\s+(?:reached|exceeded)|` +
			`credits?\s+(?:exhausted|depleted|insufficient)` +
			`)\b`,
	)
	// nimResetPattern matches the reset/retry indicator that accompanies the
	// banner ("reset at", "try again", "retry", "cooldown", "back off").
	nimResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
