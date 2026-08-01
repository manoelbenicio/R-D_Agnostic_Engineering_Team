package daemon

import (
	"regexp"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// KiroExhaustionDetector detects Kiro (Amazon Q Developer CLI fork) quota
// exhaustion from pane text or HTTP status. Ported from the reference Python
// detector (AOP/control-plane/rotation/detector.py) per doc 36 §2.1.
//
// Kiro surfaces quota limits both from its own credit/plan model ("Credits (N of
// M covered in plan)", "resets on YYYY-MM-DD" — see rotation.kiroUsageParser for
// the proactive side) and from the Claude/Bedrock runtime it wraps ("usage limit
// reached", "5-hour limit reached"). This detector covers both shapes.
//
// Implements rotation.ExhaustionDetector. It is the daemon-layer detector the
// credential-isolation fix requires for Kiro; the live screen-detection wiring
// also exists in the rotation package (rotation.matchesKiroExhaustion) so the
// current daemon.go rotationDetector (rotation.NewExhaustionDetector()) already
// handles Kiro on screen. Swapping this struct into daemon.go is a daemon-owner
// step (this fatia does not touch daemon.go).
type KiroExhaustionDetector struct {
	now func() time.Time
}

var _ rotation.ExhaustionDetector = (*KiroExhaustionDetector)(nil)

// NewKiroExhaustionDetector returns a Kiro exhaustion detector using wall-clock
// time for reset parsing. Tests inject a fixed clock via the now field.
func NewKiroExhaustionDetector() *KiroExhaustionDetector {
	return &KiroExhaustionDetector{now: time.Now}
}

// Detect implements rotation.ExhaustionDetector.
func (d *KiroExhaustionDetector) Detect(vendor, screenText string, httpStatus int) rotation.DetectionResult {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return detectExhaustion(matchesKiroScreenExhaustion, vendor, screenText, httpStatus, now)
}

// matchesKiroScreenExhaustion reports whether screenText carries a Kiro
// quota-exhaustion banner. Requires BOTH a limit phrase AND a reset/retry
// indicator to avoid false positives on normal agent output.
func matchesKiroScreenExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	return kiroScreenLimitPattern.MatchString(screenText) && kiroScreenResetPattern.MatchString(screenText)
}

var (
	kiroScreenLimitPattern = regexp.MustCompile(
		`(?i)\b(?:` +
			`usage\s+limit\s+reached|` +
			`5[\s-]?hour\s+limit\s+reached|` +
			`hit\s+your\s+limit\s+for\s+(?:claude|kiro)|` +
			`reached\s+(?:your\s+)?(?:kiro|amazon\s+q)\s+(?:usage\s+)?limit|` +
			`kiro\s+credits\s+(?:are\s+)?(?:exhausted|depleted)|` +
			`out\s+of\s+kiro\s+credits` +
			`)\b`,
	)
	kiroScreenResetPattern = regexp.MustCompile(
		`(?i)\b(?:resets?|try\s+again|resume[s]?|retry|refreshes?)\b`,
	)
)
