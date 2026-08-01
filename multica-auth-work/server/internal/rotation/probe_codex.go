package rotation

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CodexUsage struct {
	FiveHourPercentLeft float64
	FiveHourResetAt     *time.Time
	WeeklyPercentLeft   float64
	WeeklyResetAt       *time.Time
	ResetsAvailable     int
	Account             string
	Raw                 string
}

func ParseCodexUsage(panelText string, now time.Time) CodexUsage {
	var usage CodexUsage
	recognized := false

	for _, line := range strings.Split(strings.ReplaceAll(panelText, "\r\n", "\n"), "\n") {
		if match := codexProbeFiveHourPattern.FindStringSubmatch(line); match != nil {
			if percent, ok := parseCodexProbeFloat(match[1]); ok {
				usage.FiveHourPercentLeft = percent
				usage.FiveHourResetAt = parseCodexProbeTimeOfDay(match[2], match[3], now)
				recognized = true
			}
			continue
		}

		if match := codexProbeWeeklyPattern.FindStringSubmatch(line); match != nil {
			if percent, ok := parseCodexProbeFloat(match[1]); ok {
				usage.WeeklyPercentLeft = percent
				usage.WeeklyResetAt = parseCodexProbeMonthDayTime(match[4], match[5], match[2], match[3], now)
				recognized = true
			}
			continue
		}

		if match := codexProbeAccountPattern.FindStringSubmatch(line); match != nil {
			usage.Account = match[1]
			recognized = true
		}
	}

	if match := codexProbeResetsPattern.FindStringSubmatch(panelText); match != nil {
		if resets, err := strconv.Atoi(match[1]); err == nil {
			usage.ResetsAvailable = resets
			recognized = true
		}
	}

	if recognized {
		usage.Raw = panelText
	}
	return usage
}

func parseCodexProbeFloat(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseCodexProbeTimeOfDay(hourRaw, minuteRaw string, now time.Time) *time.Time {
	hour, minute, ok := parseCodexProbeClock(hourRaw, minuteRaw)
	if !ok {
		return nil
	}
	reset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if reset.Before(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return &reset
}

func parseCodexProbeMonthDayTime(dayRaw, monthRaw, hourRaw, minuteRaw string, now time.Time) *time.Time {
	day, err := strconv.Atoi(dayRaw)
	if err != nil || day < 1 || day > 31 {
		return nil
	}
	month, ok := codexProbeMonths[strings.ToLower(monthRaw)]
	if !ok {
		return nil
	}
	hour, minute, ok := parseCodexProbeClock(hourRaw, minuteRaw)
	if !ok {
		return nil
	}

	reset := time.Date(now.Year(), month, day, hour, minute, 0, 0, now.Location())
	if reset.Month() != month || reset.Day() != day {
		return nil
	}
	if reset.Before(now) {
		reset = time.Date(now.Year()+1, month, day, hour, minute, 0, 0, now.Location())
		if reset.Month() != month || reset.Day() != day {
			return nil
		}
	}
	return &reset
}

func parseCodexProbeClock(hourRaw, minuteRaw string) (int, int, bool) {
	hour, err := strconv.Atoi(hourRaw)
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, false
	}
	minute, err := strconv.Atoi(minuteRaw)
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

var (
	codexProbeFiveHourPattern = regexp.MustCompile(`(?i)^\s*5h\s+limit:\s*(?:\[[^\]]*\]\s*)?([0-9]+(?:\.[0-9]+)?)\s*%\s+left\s*\(\s*resets\s+([0-9]{1,2}):([0-9]{2})\s*\)\s*$`)
	codexProbeWeeklyPattern   = regexp.MustCompile(`(?i)^\s*weekly\s+limit:\s*(?:\[[^\]]*\]\s*)?([0-9]+(?:\.[0-9]+)?)\s*%\s+left\s*\(\s*resets\s+([0-9]{1,2}):([0-9]{2})\s+on\s+([0-9]{1,2})\s+([a-z]{3})\s*\)\s*$`)
	codexProbeResetsPattern   = regexp.MustCompile(`(?i)\byou\s+have\s+([0-9]+)\s+usage\s+limit\s+resets\s+available\b`)
	codexProbeAccountPattern  = regexp.MustCompile(`(?i)^\s*account:\s*([a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,})\s*\([^)]*\)\s*$`)
	codexProbeMonths          = map[string]time.Month{
		"jan": time.January,
		"feb": time.February,
		"mar": time.March,
		"apr": time.April,
		"may": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October,
		"nov": time.November,
		"dec": time.December,
	}
)
