package daemon

import (
	"regexp"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// ClineExhaustionDetector detects Cline (Cline CLI 2.0) quota exhaustion from
// pane text or HTTP status. Ported from the reference Python detector
// (AOP/control-plane/rotation/detector.py) per doc 36 §2.1.
//
// Cline is a multi-provider agentic CLI (OpenRouter, Anthropic, OpenAI, …) and a
// rate-limit product without a published fixed percentage ceiling — the proactive
// rotation.clineUsageParser intentionally disables percent detection and defers
// to the reactive 429/screen path (see rotation/usage.go). This detector adds
// the reactive SCREEN-text path: when Cline surfaces a provider quota banner in
// the pane, it fires so the rotation service can switch accounts. HTTP 429 is
// handled here too (detector.py detect_status_code).
//
// Implements rotation.ExhaustionDetector. Daemon-layer detector for Cline; the
// live screen-detection wiring also exists in the rotation package
// (rotation.matchesClineExhaustion). Swapping this struct into daemon.go is a
// daemon-owner step (this fatia does not touch daemon.go).
type ClineExhaustionDetector struct {
	now func() time.Time
}

var _ rotation.ExhaustionDetector = (*ClineExhaustionDetector)(nil)

// NewClineExhaustionDetector returns a Cline exhaustion detector using
// wall-clock time for reset parsing. Tests inject a fixed clock via the now
// field.
func NewClineExhaustionDetector() *ClineExhaustionDetector {
	return &ClineExhaustionDetector{now: time.Now}
}

// Detect implements rotation.ExhaustionDetector.
func (d *ClineExhaustionDetector) Detect(vendor, screenText string, httpStatus int) rotation.DetectionResult {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return detectExhaustion(matchesClineScreenExhaustion, vendor, screenText, httpStatus, now)
}

// matchesClineScreenExhaustion reports whether screenText carries a Cline
// quota-exhaustion banner (provider passthrough or Cline-branded limit message).
// Requires BOTH a limit phrase AND a reset/retry indicator to avoid false
// positives on normal agent output.
func matchesClineScreenExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return clineScreenLimitPattern.MatchString(screenText) && clineScreenResetPattern.MatchString(screenText)
}

var (
	clineScreenLimitPattern = regexp.MustCompile(
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
	clineScreenResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
