package metrics

import (
	"testing"
	"time"
)

func TestPriceForModelAliasAnthropicFableAndOpus48(t *testing.T) {
	cases := []struct {
		model string
		want  ModelPrice
	}{
		{
			model: "claude-fable-5",
			want:  ModelPrice{Provider: "anthropic", Model: "claude-fable-5", InputPerM: 10, CacheReadPerM: 1, CacheWritePerM: 12.5, OutputPerM: 50},
		},
		{
			model: "anthropic/claude-fable-5",
			want:  ModelPrice{Provider: "anthropic", Model: "claude-fable-5", InputPerM: 10, CacheReadPerM: 1, CacheWritePerM: 12.5, OutputPerM: 50},
		},
		{
			model: "claude-opus-4-8",
			want:  ModelPrice{Provider: "anthropic", Model: "claude-opus-4.8", InputPerM: 5, CacheReadPerM: 0.5, CacheWritePerM: 6.25, OutputPerM: 25},
		},
	}

	for _, tc := range cases {
		got, ok := PriceForModelAlias(tc.model)
		if !ok {
			t.Fatalf("PriceForModelAlias(%q) did not resolve", tc.model)
		}
		if got != tc.want {
			t.Fatalf("PriceForModelAlias(%q) = %+v, want %+v", tc.model, got, tc.want)
		}
	}
}

func TestResolveModelPriceByReasoningTierAndEffectivePeriod(t *testing.T) {
	effective := time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC)
	low, ok := ResolveModelPrice("gpt-5.4", "low", effective)
	if !ok {
		t.Fatal("low tier did not resolve")
	}
	high, ok := ResolveModelPrice("openai/gpt-5.4", "high", effective)
	if !ok {
		t.Fatal("high tier did not resolve")
	}
	if low.Version != CurrentPriceVersion || high.Version != CurrentPriceVersion {
		t.Fatalf("unexpected versions: low=%q high=%q", low.Version, high.Version)
	}
	if low.Model != high.Model || low.OutputPerM == high.OutputPerM {
		t.Fatalf("same model must have distinct tier prices: low=%+v high=%+v", low, high)
	}
	if gotLow, gotHigh := ComputeCostUSD(low.ModelPrice, 1_000_000, 1_000_000, 0, 0), ComputeCostUSD(high.ModelPrice, 1_000_000, 1_000_000, 0, 0); gotLow <= 0 || gotHigh <= gotLow {
		t.Fatalf("tier costs not ordered: low=%v high=%v", gotLow, gotHigh)
	}

	for _, tier := range []string{"", "medium", "xhigh"} {
		if _, ok := ResolveModelPrice("gpt-5.4", tier, effective); ok {
			t.Fatalf("unsupported tier %q unexpectedly resolved", tier)
		}
	}
	if _, ok := ResolveModelPrice("gpt-5.4", "high", currentPriceEffectiveFrom.Add(-time.Nanosecond)); ok {
		t.Fatal("quote resolved before effective_from")
	}
	if _, ok := ResolveModelPrice("gpt-5.4", "high", currentPriceEffectiveUntil); ok {
		t.Fatal("quote resolved at exclusive effective_until")
	}
}

func TestResolveModelPriceExplicitTierIndependentFallback(t *testing.T) {
	effective := time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC)
	price, ok := ResolveModelPrice("claude-opus-4-8", "unsupported-for-tiered-models", effective)
	if !ok {
		t.Fatal("tier-independent model did not use its explicit base-price policy")
	}
	if price.ThinkingLevel != "*" {
		t.Fatalf("fallback marker = %q, want *", price.ThinkingLevel)
	}
}
