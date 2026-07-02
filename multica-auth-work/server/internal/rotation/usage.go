package rotation

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultUsageThreshold = 0.10

type QuotaModel string

const (
	QuotaTime5h    QuotaModel = "time_5h"
	QuotaCredits   QuotaModel = "credits"
	QuotaPerModel  QuotaModel = "per_model"
	QuotaRateLimit QuotaModel = "rate_limit"
)

type UsageSample struct {
	Vendor           string
	Model            string
	PercentRemaining float64
	Approaching      bool
	ResetAt          *time.Time
	QuotaModel       QuotaModel
	Raw              string
}

type UsageParser interface {
	Parse(text string) []UsageSample
}

type UsageDetector struct {
	threshold float64
	parsers   map[string]UsageParser
}

func NewUsageDetector(threshold float64) *UsageDetector {
	if threshold <= 0 || threshold >= 1 {
		threshold = defaultUsageThreshold
	}
	return &UsageDetector{
		threshold: threshold,
		parsers: map[string]UsageParser{
			"codex":       codexUsageParser{now: time.Now},
			"kiro":        kiroUsageParser{},
			"antigravity": antigravityUsageParser{now: time.Now},
			"cline":       clineUsageParser{},
		},
	}
}

func (d *UsageDetector) Detect(vendor, text string) []UsageSample {
	if d == nil {
		d = NewUsageDetector(0)
	}
	parser, ok := d.parsers[strings.ToLower(strings.TrimSpace(vendor))]
	if !ok || parser == nil {
		return nil
	}
	threshold := d.threshold
	if threshold <= 0 || threshold >= 1 {
		threshold = defaultUsageThreshold
	}

	samples := parser.Parse(text)
	for i := range samples {
		if samples[i].QuotaModel == QuotaRateLimit {
			samples[i].Approaching = false
			continue
		}
		samples[i].Approaching = samples[i].PercentRemaining <= threshold*100
	}
	return samples
}

type codexUsageParser struct {
	now func() time.Time
}

func (p codexUsageParser) Parse(text string) []UsageSample {
	now := time.Now
	if p.now != nil {
		now = p.now
	}

	var samples []UsageSample
	for _, match := range codexPassiveUsagePattern.FindAllStringSubmatch(text, -1) {
		percent, ok := parseFloat(match[1])
		if !ok {
			continue
		}
		samples = append(samples, UsageSample{
			Vendor:           "codex",
			PercentRemaining: percent,
			QuotaModel:       QuotaTime5h,
			Raw:              strings.TrimSpace(match[0]),
		})
	}

	for _, line := range lines(text) {
		match := codexStatusUsagePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		percent, ok := parseFloat(match[1])
		if !ok {
			continue
		}
		samples = append(samples, UsageSample{
			Vendor:           "codex",
			PercentRemaining: percent,
			ResetAt:          parseTimeOfDayReset(line, now()),
			QuotaModel:       QuotaTime5h,
			Raw:              strings.TrimSpace(line),
		})
	}
	return samples
}

type kiroUsageParser struct{}

func (kiroUsageParser) Parse(text string) []UsageSample {
	match := kiroCreditsPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	used, ok := parseFloat(match[1])
	if !ok {
		return nil
	}
	covered, ok := parseFloat(match[2])
	if !ok || covered <= 0 {
		return nil
	}

	return []UsageSample{{
		Vendor:           "kiro",
		PercentRemaining: clampPercent(100 * (covered - used) / covered),
		ResetAt:          parseDateReset(text),
		QuotaModel:       QuotaCredits,
		Raw:              strings.TrimSpace(match[0]),
	}}
}

type antigravityUsageParser struct {
	now func() time.Time
}

func (p antigravityUsageParser) Parse(text string) []UsageSample {
	now := time.Now
	if p.now != nil {
		now = p.now
	}

	var samples []UsageSample
	for _, line := range lines(text) {
		modelMatch := antigravityModelPattern.FindStringSubmatch(line)
		if modelMatch == nil {
			continue
		}

		percent, ok := antigravityRemainingPercent(line)
		if !ok {
			continue
		}
		samples = append(samples, UsageSample{
			Vendor:           "antigravity",
			Model:            modelMatch[1] + " (" + modelMatch[2] + ")",
			PercentRemaining: percent,
			ResetAt:          parseRefreshDuration(line, now()),
			QuotaModel:       QuotaPerModel,
			Raw:              strings.TrimSpace(line),
		})
	}
	return samples
}

type clineUsageParser struct{}

func (clineUsageParser) Parse(text string) []UsageSample {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	// ClinePass is a rate-limit product without a published fixed percentage
	// ceiling. Proactive percent detection is intentionally disabled here; the
	// reactive 429 path in detector.go remains the correct signal.
	return []UsageSample{{
		Vendor:     "cline",
		QuotaModel: QuotaRateLimit,
		Raw:        strings.TrimSpace(text),
	}}
}

func antigravityRemainingPercent(line string) (float64, bool) {
	if antigravityQuotaAvailablePattern.MatchString(line) {
		return 100, true
	}
	if match := antigravityRemainingPattern.FindStringSubmatch(line); match != nil {
		return parseFloat(match[1])
	}
	match := antigravityBarePercentPattern.FindStringSubmatch(line)
	if match == nil {
		return 0, false
	}
	return parseFloat(match[1])
}

func parseTimeOfDayReset(text string, now time.Time) *time.Time {
	match := timeOfDayResetPattern.FindStringSubmatch(text)
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

func parseDateReset(text string) *time.Time {
	match := dateResetPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	reset, err := time.Parse("2006-01-02", match[1])
	if err != nil {
		return nil
	}
	return &reset
}

func parseRefreshDuration(text string, now time.Time) *time.Time {
	match := refreshDurationPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	hours, err := strconv.Atoi(match[1])
	if err != nil || hours < 0 {
		return nil
	}
	minutes, err := strconv.Atoi(match[2])
	if err != nil || minutes < 0 {
		return nil
	}
	reset := now.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)
	return &reset
}

func parseFloat(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func clampPercent(percent float64) float64 {
	switch {
	case percent < 0:
		return 0
	case percent > 100:
		return 100
	default:
		return percent
	}
}

func lines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

var (
	codexPassiveUsagePattern = regexp.MustCompile(`(?i)\bless\s+than\s+([0-9]+(?:\.[0-9]+)?)%\s+of\s+your\s+5h\s+limit\s+left\b`)
	codexStatusUsagePattern  = regexp.MustCompile(`(?i)\b5h\s+limit:\s*(?:\[[^\]]*\]\s*)?([0-9]+(?:\.[0-9]+)?)%\s+left\b`)
	kiroCreditsPattern       = regexp.MustCompile(`(?i)\bCredits\s*\(\s*([0-9]+(?:\.[0-9]+)?)\s+of\s+([0-9]+(?:\.[0-9]+)?)\s+covered\s+in\s+plan\s*\)`)
	dateResetPattern         = regexp.MustCompile(`(?i)\bresets\s+on\s+([0-9]{4}-[0-9]{2}-[0-9]{2})\b`)
	timeOfDayResetPattern    = regexp.MustCompile(`(?i)\bresets?\s+([0-9]{1,2}):([0-9]{2})\b`)
	antigravityModelPattern  = regexp.MustCompile(
		`^\s*(Gemini 3\.5 Flash|Gemini 3\.1 Pro|Claude Sonnet 4\.6|Claude Opus 4\.6|GPT-OSS 120B)\s+\((Medium|High|Low|Thinking)\)(?:\s|$)`,
	)
	antigravityQuotaAvailablePattern = regexp.MustCompile(`(?i)\bQuota\s+available\b`)
	antigravityRemainingPattern      = regexp.MustCompile(`(?i)\b([0-9]+(?:\.[0-9]+)?)%\s+remaining\b`)
	antigravityBarePercentPattern    = regexp.MustCompile(`\b([0-9]+(?:\.[0-9]+)?)%\b`)
	refreshDurationPattern           = regexp.MustCompile(`(?i)\bRefreshes\s+in\s+([0-9]+)h\s+([0-9]+)m\b`)
)
