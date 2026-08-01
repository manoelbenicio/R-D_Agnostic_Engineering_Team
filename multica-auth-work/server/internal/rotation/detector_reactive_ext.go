package rotation

import (
	"regexp"
	"strconv"
	"time"
)

type ReactiveClassification struct {
	Exhausted            bool
	Kind                 string // "hard_stop" | "approaching" | "none"
	ResetAt              *time.Time
	SuspectFalsePositive bool
}

func ClassifyCodexReactive(screenText string, now time.Time) ReactiveClassification {
	if codexReactiveHardStopPattern.MatchString(screenText) {
		return ReactiveClassification{
			Exhausted:            true,
			Kind:                 "hard_stop",
			ResetAt:              parseCodexReactiveResetAt(screenText, now),
			SuspectFalsePositive: hasCodexReactiveRemaining5hQuota(screenText),
		}
	}

	if codexReactiveApproachingPattern.MatchString(screenText) {
		return ReactiveClassification{
			Kind: "approaching",
		}
	}

	return ReactiveClassification{Kind: "none"}
}

func parseCodexReactiveResetAt(screenText string, now time.Time) *time.Time {
	match := codexReactiveWaitUntilPattern.FindStringSubmatch(screenText)
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

func hasCodexReactiveRemaining5hQuota(screenText string) bool {
	match := codexReactiveRemaining5hPattern.FindStringSubmatch(screenText)
	if match == nil {
		return false
	}

	percentLeft, err := strconv.Atoi(match[1])
	if err != nil {
		return false
	}

	// A hard-stop banner is suspect when the same screen shows the real status
	// format "5h limit: N% left" with nonzero N. That remaining 5h quota
	// contradicts definitive 5h exhaustion, so callers should cross-check /status.
	return percentLeft > 0 && percentLeft <= 100
}

var (
	codexReactiveHardStopPattern    = regexp.MustCompile(`(?i)\b(?:codex\s+message\s+)?usage\s+limit\s+reached\b`)
	codexReactiveApproachingPattern  = regexp.MustCompile(`(?i)\bless\s+than\s+\d+%\s+of\s+your\s+5h\s+limit\s+left\b`)
	codexReactiveWaitUntilPattern    = regexp.MustCompile(`(?i)\bplease\s+wait\s+until\s+([0-9]{1,2}):([0-9]{2})\b`)
	codexReactiveRemaining5hPattern  = regexp.MustCompile(`(?i)\b5h\s+limit:\s*([0-9]{1,3})%\s+left\b`)
)
