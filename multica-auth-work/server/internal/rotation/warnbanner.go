package rotation

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type WarningDetector struct {
	now func() time.Time
}

func NewWarningDetector() *WarningDetector {
	return &WarningDetector{now: time.Now}
}

func (d *WarningDetector) DetectWarning(vendor, screenText string) (approaching bool, percentLeft int, resetAt *time.Time) {
	text := strings.TrimSpace(screenText)
	if text == "" || reactiveLimitReachedPattern.MatchString(text) {
		return false, 0, nil
	}

	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "codex":
		approaching, percentLeft = detectCodexWarning(text)
	case "antigravity", "kiro":
		// Confirmar contra tela real antes de habilitar padrões. Estes
		// vendors ficam mapeados para evitar inventar strings não verificadas.
		return false, 0, nil
	default:
		return false, 0, nil
	}
	if !approaching {
		return false, 0, nil
	}
	return true, percentLeft, parseWarningResetAt(text, d.currentTime())
}

func (d *WarningDetector) currentTime() time.Time {
	if d != nil && d.now != nil {
		return d.now()
	}
	return time.Now()
}

func detectCodexWarning(text string) (bool, int) {
	if match := codexPercentWarningPattern.FindStringSubmatch(text); match != nil {
		percent, err := strconv.Atoi(match[1])
		if err != nil {
			return true, 0
		}
		return true, percent
	}
	if codexHeadsUpPattern.MatchString(text) && codexFiveHourLimitPattern.MatchString(text) {
		return true, 0
	}
	return false, 0
}

func parseWarningResetAt(text string, now time.Time) *time.Time {
	match := warningResetPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	hour, err := strconv.Atoi(match[1])
	if err != nil || hour < 0 || hour > 23 {
		return nil
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil || minute < 0 || minute > 59 {
		return nil
	}
	reset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if reset.Before(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return &reset
}

var (
	codexPercentWarningPattern  = regexp.MustCompile(`(?i)\bless\s+than\s+([0-9]{1,3})%\s+of\s+your\s+5h\s+limit\s+left\b`)
	codexHeadsUpPattern         = regexp.MustCompile(`(?i)\bheads\s+up\b`)
	codexFiveHourLimitPattern   = regexp.MustCompile(`(?i)\b5h\s+limit\b`)
	warningResetPattern         = regexp.MustCompile(`(?i)\bresets?\s+([0-9]{1,2}):([0-9]{2})\b`)
	reactiveLimitReachedPattern = regexp.MustCompile(
		`(?i)\b(?:usage\s+limit\s+reached|5-hour\s+limit\s+reached|you['’]?ve\s+hit\s+your\s+usage\s+limit|reached\s+the\s+quota\s+limit)\b`,
	)
)
