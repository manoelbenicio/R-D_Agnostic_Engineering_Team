package daemon

import (
	"regexp"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// NimExhaustionDetector detects NVIDIA NIM quota exhaustion from pane text or
// HTTP status. NIM is an OpenAI-compatible inference API
// (https://integrate.api.nvidia.com/v1); the native nim runtime (Agent-1,
// server/pkg/agent/nim.go) drives it over HTTP. This detector adds the reactive
// SCREEN-text path: when NIM surfaces a quota banner (OpenAI-compatible 429
// wrapped in a JSON error body that reaches the agent pane), it fires so the
// rotation service can switch accounts. HTTP 429 is handled here too
// (detector.py detect_status_code).
//
// Implements rotation.ExhaustionDetector. Daemon-layer detector for NIM; the
// live screen-detection wiring also exists in the rotation package
// (rotation.matchesNimExhaustion). Swapping this struct into daemon.go is a
// daemon-owner step (this fatia does not touch daemon.go).
type NimExhaustionDetector struct {
	now func() time.Time
}

var _ rotation.ExhaustionDetector = (*NimExhaustionDetector)(nil)

// NewNimExhaustionDetector returns a NIM exhaustion detector using wall-clock
// time for reset parsing. Tests inject a fixed clock via the now field.
func NewNimExhaustionDetector() *NimExhaustionDetector {
	return &NimExhaustionDetector{now: time.Now}
}

// Detect implements rotation.ExhaustionDetector.
func (d *NimExhaustionDetector) Detect(vendor, screenText string, httpStatus int) rotation.DetectionResult {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return detectExhaustion(matchesNimScreenExhaustion, vendor, screenText, httpStatus, now)
}

// matchesNimScreenExhaustion reports whether screenText carries a NIM
// quota-exhaustion banner (OpenAI-compatible rate/usage/quota limit or
// NVIDIA/NIM-branded message). Requires BOTH a limit phrase AND a reset/retry
// indicator to avoid false positives on normal agent output.
func matchesNimScreenExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return nimScreenLimitPattern.MatchString(screenText) && nimScreenResetPattern.MatchString(screenText)
}

var (
	nimScreenLimitPattern = regexp.MustCompile(
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
	nimScreenResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|resume[s]?|cooldown|back[\s-]?off)\b`,
	)
)
