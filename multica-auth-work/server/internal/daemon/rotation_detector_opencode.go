package daemon

import (
	"regexp"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// OpenCodeExhaustionDetector detects OpenCode quota exhaustion from pane text or
// HTTP status. Ported from the reference Python detector
// (AOP/control-plane/rotation/detector.py) per doc 36 §2.1.
//
// OpenCode is an open-source multi-provider coding agent runtime; GLM fleet
// agents that run through the OpenCode-compatible runtime reuse the same
// isolation preparer (see execenv/opencode_home.go). Because OpenCode fronts
// many providers, its exhaustion banner is provider-agnostic: rate/usage/token/
// quota limit phrases surfaced from the underlying provider. HTTP 429 is handled
// here too (detector.py detect_status_code).
//
// Implements rotation.ExhaustionDetector. Daemon-layer detector for OpenCode;
// the live screen-detection wiring also exists in the rotation package
// (rotation.matchesOpenCodeExhaustion). Swapping this struct into daemon.go is a
// daemon-owner step (this fatia does not touch daemon.go).
type OpenCodeExhaustionDetector struct {
	now func() time.Time
}

var _ rotation.ExhaustionDetector = (*OpenCodeExhaustionDetector)(nil)

// NewOpenCodeExhaustionDetector returns an OpenCode exhaustion detector using
// wall-clock time for reset parsing. Tests inject a fixed clock via the now
// field.
func NewOpenCodeExhaustionDetector() *OpenCodeExhaustionDetector {
	return &OpenCodeExhaustionDetector{now: time.Now}
}

// Detect implements rotation.ExhaustionDetector.
func (d *OpenCodeExhaustionDetector) Detect(vendor, screenText string, httpStatus int) rotation.DetectionResult {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return detectExhaustion(matchesOpenCodeScreenExhaustion, vendor, screenText, httpStatus, now)
}

// matchesOpenCodeScreenExhaustion reports whether screenText carries an OpenCode
// quota-exhaustion banner (provider passthrough or OpenCode-branded limit
// message). Requires BOTH a limit phrase AND a reset/retry indicator to avoid
// false positives on normal agent output.
func matchesOpenCodeScreenExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return opencodeScreenLimitPattern.MatchString(screenText) && opencodeScreenResetPattern.MatchString(screenText)
}

var (
	opencodeScreenLimitPattern = regexp.MustCompile(
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
	opencodeScreenResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
