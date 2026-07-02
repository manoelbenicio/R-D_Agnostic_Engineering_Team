package rotation

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Detector struct {
	now func() time.Time
}

var _ ExhaustionDetector = (*Detector)(nil)

func NewExhaustionDetector() *Detector {
	return &Detector{now: time.Now}
}

func (d *Detector) Detect(vendor, screenText string, httpStatus int) DetectionResult {
	if httpStatus == 429 {
		return DetectionResult{
			Exhausted: true,
			Signal:    SignalHTTP429,
			ResetAt:   d.parseResetAt(screenText),
		}
	}
	if httpStatus == 503 || highTrafficPattern.MatchString(screenText) {
		return DetectionResult{}
	}

	if !matchesVendorExhaustion(vendor, screenText) {
		return DetectionResult{}
	}
	return DetectionResult{
		Exhausted: true,
		Signal:    SignalScreen,
		ResetAt:   d.parseResetAt(screenText),
	}
}

func (d *Detector) parseResetAt(text string) *time.Time {
	now := time.Now()
	if d != nil && d.now != nil {
		now = d.now()
	}
	return parseResetAt(text, now)
}

func matchesVendorExhaustion(vendor, screenText string) bool {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "codex":
		return codexUsageLimitPattern.MatchString(screenText) && codexTryAgainPattern.MatchString(screenText)
	case "antigravity":
		return antigravityQuotaPattern.MatchString(screenText) && antigravityResumePattern.MatchString(screenText)
	case "kiro", "opus":
		return claudeLimitPattern.MatchString(screenText) && resetPattern.MatchString(screenText)
	default:
		return false
	}
}

func parseResetAt(text string, now time.Time) *time.Time {
	match := resetAtPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}

	hour, err := strconv.Atoi(match[1])
	if err != nil || hour < 1 || hour > 12 {
		return nil
	}
	minute := 0
	if match[2] != "" {
		minute, err = strconv.Atoi(match[2])
		if err != nil || minute < 0 || minute > 59 {
			return nil
		}
	}

	meridiem := strings.ToLower(strings.ReplaceAll(match[3], ".", ""))
	switch meridiem {
	case "am":
		if hour == 12 {
			hour = 0
		}
	case "pm":
		if hour != 12 {
			hour += 12
		}
	default:
		return nil
	}

	reset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if reset.Before(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return &reset
}

var (
	highTrafficPattern       = regexp.MustCompile(`(?i)\bhigh traffic\b`)
	codexUsageLimitPattern   = regexp.MustCompile(`(?i)\byou['’]ve hit your usage limit\b`)
	codexTryAgainPattern     = regexp.MustCompile(`(?i)\btry again\s+(?:at|in)\b`)
	antigravityQuotaPattern  = regexp.MustCompile(`(?i)\breached the quota limit for this model\b`)
	antigravityResumePattern = regexp.MustCompile(`(?i)\bresume using this model at\b`)
	claudeLimitPattern       = regexp.MustCompile(`(?i)\b(?:usage limit reached|5-hour limit reached|hit your limit for claude)\b`)
	resetPattern             = regexp.MustCompile(`(?i)\breset\b`)
	resetAtPattern           = regexp.MustCompile(`(?i)\b(?:reset|resume(?:\s+using\s+this\s+model)?|try\s+again)\b[^\n.]{0,120}?\bat\s+([0-9]{1,2})(?::([0-9]{2}))?\s*([ap]\.?m\.?)\b`)
)
