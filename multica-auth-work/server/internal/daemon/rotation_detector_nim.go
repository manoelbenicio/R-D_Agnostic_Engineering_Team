package daemon

import (
	"regexp"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// NIMExhaustionDetector detects NVIDIA NIM quota exhaustion from the native
// HTTP backend's error text or an explicit HTTP status.
type NIMExhaustionDetector struct {
	now func() time.Time
}

var _ rotation.ExhaustionDetector = (*NIMExhaustionDetector)(nil)

func NewNIMExhaustionDetector() *NIMExhaustionDetector {
	return &NIMExhaustionDetector{now: time.Now}
}

func (d *NIMExhaustionDetector) Detect(vendor, screenText string, httpStatus int) rotation.DetectionResult {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return detectExhaustion(matchesNIMScreenExhaustion, vendor, screenText, httpStatus, now)
}

func matchesNIMScreenExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	if nimScreenHTTP429Pattern.MatchString(screenText) {
		return true
	}
	return nimScreenLimitPattern.MatchString(screenText) && nimScreenResetPattern.MatchString(screenText)
}

var (
	nimScreenHTTP429Pattern = regexp.MustCompile(`(?i)\bNIM\s+API\s+returned\s+429\b`)
	nimScreenLimitPattern   = regexp.MustCompile(`(?i)\b(?:rate\s+limit\s+(?:exceeded|reached)|quota\s+(?:exceeded|depleted)|resource\s+exhausted|too\s+many\s+requests|credit(?:s|\s+balance)?\s+(?:exhausted|depleted)|usage\s+limit\s+reached)\b`)
	nimScreenResetPattern   = regexp.MustCompile(`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|cooldown|back[\s-]?off|after)\b`)
)
