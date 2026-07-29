package metrics

import (
	"regexp"
	"strings"
	"time"
)

type ModelPrice struct {
	Provider       string
	Model          string
	InputPerM      float64
	CacheReadPerM  float64
	CacheWritePerM float64
	OutputPerM     float64
}

// EffectiveModelPrice is the immutable quote persisted with a usage row.
// EffectiveUntil is exclusive, so adjacent price versions cannot overlap.
type EffectiveModelPrice struct {
	ModelPrice
	ThinkingLevel  string
	Version        string
	EffectiveFrom  time.Time
	EffectiveUntil time.Time
}

const CurrentPriceVersion = "2026-07-01"

var (
	currentPriceEffectiveFrom  = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	currentPriceEffectiveUntil = time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
)

var modelPrices = map[string]ModelPrice{
	"openai:gpt-5.5":              {Provider: "openai", Model: "gpt-5.5", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 0.50, OutputPerM: 30.00},
	"openai:gpt-5.4":              {Provider: "openai", Model: "gpt-5.4", InputPerM: 2.50, CacheReadPerM: 0.25, CacheWritePerM: 0.25, OutputPerM: 15.00},
	"openai:gpt-5.4-mini":         {Provider: "openai", Model: "gpt-5.4-mini", InputPerM: 0.75, CacheReadPerM: 0.075, CacheWritePerM: 0.075, OutputPerM: 4.50},
	"openai:gpt-5.3-codex":        {Provider: "openai", Model: "gpt-5.3-codex", InputPerM: 1.75, CacheReadPerM: 0.175, CacheWritePerM: 0.175, OutputPerM: 14.00},
	"openai:gpt-5.2-codex":        {Provider: "openai", Model: "gpt-5.2-codex", InputPerM: 1.75, CacheReadPerM: 0.175, CacheWritePerM: 0.175, OutputPerM: 14.00},
	"anthropic:claude-fable-5":    {Provider: "anthropic", Model: "claude-fable-5", InputPerM: 10.00, CacheReadPerM: 1.00, CacheWritePerM: 12.50, OutputPerM: 50.00},
	"anthropic:claude-opus-4.8":   {Provider: "anthropic", Model: "claude-opus-4.8", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 6.25, OutputPerM: 25.00},
	"anthropic:claude-opus-4.7":   {Provider: "anthropic", Model: "claude-opus-4.7", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 6.25, OutputPerM: 25.00},
	"anthropic:claude-opus-4.6":   {Provider: "anthropic", Model: "claude-opus-4.6", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 6.25, OutputPerM: 25.00},
	"anthropic:claude-opus-4.5":   {Provider: "anthropic", Model: "claude-opus-4.5", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 6.25, OutputPerM: 25.00},
	"anthropic:claude-sonnet-4.6": {Provider: "anthropic", Model: "claude-sonnet-4.6", InputPerM: 3.00, CacheReadPerM: 0.30, CacheWritePerM: 3.75, OutputPerM: 15.00},
	"anthropic:claude-sonnet-4.5": {Provider: "anthropic", Model: "claude-sonnet-4.5", InputPerM: 3.00, CacheReadPerM: 0.30, CacheWritePerM: 3.75, OutputPerM: 15.00},
	"anthropic:claude-haiku-4.5":  {Provider: "anthropic", Model: "claude-haiku-4.5", InputPerM: 1.00, CacheReadPerM: 0.10, CacheWritePerM: 1.25, OutputPerM: 5.00},
	"deepseek:v4-pro":             {Provider: "deepseek", Model: "v4-pro", InputPerM: 1.74, CacheReadPerM: 0.0145, CacheWritePerM: 1.74, OutputPerM: 3.48},
	"deepseek:v4-flash":           {Provider: "deepseek", Model: "v4-flash", InputPerM: 0.56, CacheReadPerM: 0.0112, CacheWritePerM: 0.56, OutputPerM: 1.12},
	"minimax:m2.7":                {Provider: "minimax", Model: "m2.7", InputPerM: 0.30, CacheReadPerM: 0.06, CacheWritePerM: 0.375, OutputPerM: 1.20},
	"minimax:m2.7-highspeed":      {Provider: "minimax", Model: "m2.7-highspeed", InputPerM: 0.60, CacheReadPerM: 0.06, CacheWritePerM: 0.375, OutputPerM: 2.40},
	"google:gemini-3-flash":       {Provider: "google", Model: "gemini-3-flash", InputPerM: 0.50, CacheReadPerM: 0.05, CacheWritePerM: 0.50, OutputPerM: 3.00},
	"google:gemini-3.1-pro":       {Provider: "google", Model: "gemini-3.1-pro", InputPerM: 2.00, CacheReadPerM: 0.20, CacheWritePerM: 2.00, OutputPerM: 12.00},
	"google:gemini-2.5-pro":       {Provider: "google", Model: "gemini-2.5-pro", InputPerM: 1.25, CacheReadPerM: 0.31, CacheWritePerM: 1.25, OutputPerM: 10.00},
	"google:gemini-2.5-flash":     {Provider: "google", Model: "gemini-2.5-flash", InputPerM: 0.30, CacheReadPerM: 0.03, CacheWritePerM: 0.30, OutputPerM: 2.50},
}

var modelAliasRules = []struct {
	re       *regexp.Regexp
	priceKey string
}{
	{regexp.MustCompile(`(^|/|:)gpt-5[.-]5$|^gpt-5-5$`), "openai:gpt-5.5"},
	{regexp.MustCompile(`(^|/|:)gpt-5[.-]4($|-2026-03-05|-xhigh)`), "openai:gpt-5.4"},
	{regexp.MustCompile(`(^|/|:)gpt-5[.-]4-mini($|[^a-z0-9])`), "openai:gpt-5.4-mini"},
	{regexp.MustCompile(`(^|/|:)gpt-5[.-]3-codex$`), "openai:gpt-5.3-codex"},
	{regexp.MustCompile(`(^|/|:)gpt-5[.-]2-codex$`), "openai:gpt-5.2-codex"},
	{regexp.MustCompile(`claude-fable-5`), "anthropic:claude-fable-5"},
	{regexp.MustCompile(`claude-opus-4[-.]8`), "anthropic:claude-opus-4.8"},
	{regexp.MustCompile(`claude-opus-4[-.]7`), "anthropic:claude-opus-4.7"},
	{regexp.MustCompile(`claude-opus-4[-.]6`), "anthropic:claude-opus-4.6"},
	{regexp.MustCompile(`claude-opus-4[-.]5`), "anthropic:claude-opus-4.5"},
	{regexp.MustCompile(`claude-sonnet-4[-.]6|claude-4[-.]6-sonnet`), "anthropic:claude-sonnet-4.6"},
	{regexp.MustCompile(`claude-sonnet-4[-.]5|claude-4[-.]5-sonnet`), "anthropic:claude-sonnet-4.5"},
	{regexp.MustCompile(`claude-haiku-4[-.]5`), "anthropic:claude-haiku-4.5"},
	{regexp.MustCompile(`deepseek-v4-pro`), "deepseek:v4-pro"},
	{regexp.MustCompile(`deepseek-v4-flash|^deepseek-chat$|^deepseek-reasoner$`), "deepseek:v4-flash"},
	{regexp.MustCompile(`minimax-m2[.]7.*highspeed|highspeed.*minimax-m2[.]7`), "minimax:m2.7-highspeed"},
	{regexp.MustCompile(`minimax-m2[.]7`), "minimax:m2.7"},
	{regexp.MustCompile(`gemini-3-flash`), "google:gemini-3-flash"},
	{regexp.MustCompile(`gemini-3[.]1-pro`), "google:gemini-3.1-pro"},
	{regexp.MustCompile(`gemini-2[.]5-pro`), "google:gemini-2.5-pro"},
	{regexp.MustCompile(`gemini-2[.]5-flash`), "google:gemini-2.5-flash"},
}

// tierModelPrices lists only models whose unit price changes with reasoning
// effort. The absence of a wildcard is policy: missing or unsupported tiers
// are unpriced instead of silently falling back to the base model rate.
var tierModelPrices = map[string][]EffectiveModelPrice{
	"openai:gpt-5.4": {
		{
			ModelPrice:    ModelPrice{Provider: "openai", Model: "gpt-5.4", InputPerM: 2.50, CacheReadPerM: 0.25, CacheWritePerM: 0.25, OutputPerM: 15.00},
			ThinkingLevel: "low", Version: CurrentPriceVersion,
			EffectiveFrom: currentPriceEffectiveFrom, EffectiveUntil: currentPriceEffectiveUntil,
		},
		{
			ModelPrice:    ModelPrice{Provider: "openai", Model: "gpt-5.4", InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 0.50, OutputPerM: 30.00},
			ThinkingLevel: "high", Version: CurrentPriceVersion,
			EffectiveFrom: currentPriceEffectiveFrom, EffectiveUntil: currentPriceEffectiveUntil,
		},
	},
}

func priceKeyForModelAlias(model string) (string, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, rule := range modelAliasRules {
		if rule.re.MatchString(model) {
			return rule.priceKey, true
		}
	}
	return "", false
}

// ResolveModelPrice returns the authoritative quote for the model, declared
// reasoning tier, and usage instant. Tier-dependent models require an exact
// tier. Models absent from tierModelPrices are explicitly tier-independent and
// retain the legacy base-price fallback.
func ResolveModelPrice(model, thinkingLevel string, effectiveAt time.Time) (EffectiveModelPrice, bool) {
	key, ok := priceKeyForModelAlias(model)
	if !ok || effectiveAt.IsZero() {
		return EffectiveModelPrice{}, false
	}
	effectiveAt = effectiveAt.UTC()
	tier := strings.ToLower(strings.TrimSpace(thinkingLevel))
	if prices, tierDependent := tierModelPrices[key]; tierDependent {
		if tier == "" {
			return EffectiveModelPrice{}, false
		}
		for _, price := range prices {
			if price.ThinkingLevel == tier && !effectiveAt.Before(price.EffectiveFrom) && effectiveAt.Before(price.EffectiveUntil) {
				return price, true
			}
		}
		return EffectiveModelPrice{}, false
	}

	price, ok := modelPrices[key]
	if !ok || effectiveAt.Before(currentPriceEffectiveFrom) || !effectiveAt.Before(currentPriceEffectiveUntil) {
		return EffectiveModelPrice{}, false
	}
	return EffectiveModelPrice{
		ModelPrice:     price,
		ThinkingLevel:  "*",
		Version:        CurrentPriceVersion,
		EffectiveFrom:  currentPriceEffectiveFrom,
		EffectiveUntil: currentPriceEffectiveUntil,
	}, true
}

// PriceForModelAlias preserves the tier-independent lookup used by legacy
// dashboard metrics. New usage persistence must call ResolveModelPrice.
func PriceForModelAlias(model string) (ModelPrice, bool) {
	key, ok := priceKeyForModelAlias(model)
	if !ok {
		return ModelPrice{}, false
	}
	price, ok := modelPrices[key]
	return price, ok
}

func tokenCostUSD(tokens int64, pricePerM float64) float64 {
	if tokens <= 0 || pricePerM <= 0 {
		return 0
	}
	return float64(tokens) * pricePerM / 1_000_000
}

// ComputeCostUSD applies one resolved quote to all token counters.
func ComputeCostUSD(price ModelPrice, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64) float64 {
	return tokenCostUSD(inputTokens, price.InputPerM) +
		tokenCostUSD(outputTokens, price.OutputPerM) +
		tokenCostUSD(cacheReadTokens, price.CacheReadPerM) +
		tokenCostUSD(cacheWriteTokens, price.CacheWritePerM)
}
