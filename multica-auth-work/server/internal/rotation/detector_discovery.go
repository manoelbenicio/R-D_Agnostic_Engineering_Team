package rotation

import (
	"strings"
	"time"
)

// SignalDiscovery identifies exhaustion derived from session-discovery
// metadata rather than terminal output or a provider HTTP response.
const SignalDiscovery ExhaustionSignal = "discovery"

// DiscoverySession is the non-secret subset of the session-discovery
// contract needed to determine whether a provider session is exhausted.
type DiscoverySession struct {
	Provider  string
	Status    string
	ExpiresAt string
}

// DetectDiscoverySession classifies one discovery observation without reading
// credentials or causing rotation. Explicit exhausted/expired status wins. An
// otherwise usable session is exhausted only when a valid expires_at is at or
// before now. Missing or malformed timestamps preserve the prior status-only
// fallback and do not create a false exhaustion signal.
//
// requestedProvider and session.Provider must identify the same provider
// family. This prevents an expired observation from one provider affecting a
// different provider's account pool.
func DetectDiscoverySession(requestedProvider string, session DiscoverySession, now time.Time) DetectionResult {
	if !sameDiscoveryProvider(requestedProvider, session.Provider) {
		return DetectionResult{}
	}

	switch strings.ToLower(strings.TrimSpace(session.Status)) {
	case "expired", "exhausted":
		return DetectionResult{Exhausted: true, Signal: SignalDiscovery}
	}

	expiresAt := strings.TrimSpace(session.ExpiresAt)
	if expiresAt == "" {
		return DetectionResult{}
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || expiry.After(now) {
		return DetectionResult{}
	}
	return DetectionResult{Exhausted: true, Signal: SignalDiscovery}
}

func sameDiscoveryProvider(left, right string) bool {
	left = canonicalDiscoveryProvider(left)
	right = canonicalDiscoveryProvider(right)
	return left != "" && left == right
}

func canonicalDiscoveryProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "kiro_cli":
		return "kiro"
	case "gemini_cli", "agy":
		return "antigravity"
	case "claude_code":
		return "claude"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}
