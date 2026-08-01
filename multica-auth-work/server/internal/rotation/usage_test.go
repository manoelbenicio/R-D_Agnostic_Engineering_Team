package rotation

import (
	"math"
	"testing"
	"time"
)

func TestUsageDetectorCodexPassiveBannerApproaching(t *testing.T) {
	got := NewUsageDetector(0.10).Detect("codex", "Heads up, you have less than 10% of your 5h limit left. Run /status for details.")

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "codex", "", QuotaTime5h, 10, true)
	if got[0].ResetAt != nil {
		t.Fatalf("ResetAt = %v, want nil for passive banner", got[0].ResetAt)
	}
}

func TestUsageDetectorCodexStatusNotApproaching(t *testing.T) {
	d := NewUsageDetector(0.10)
	now := time.Date(2026, 7, 1, 18, 30, 0, 0, time.UTC)
	d.parsers["codex"] = codexUsageParser{now: func() time.Time { return now }}

	got := d.Detect("codex", "5h limit: 80% left (resets 19:59)")

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "codex", "", QuotaTime5h, 80, false)
	wantReset := time.Date(2026, 7, 1, 19, 59, 0, 0, time.UTC)
	if got[0].ResetAt == nil || !got[0].ResetAt.Equal(wantReset) {
		t.Fatalf("ResetAt = %v, want %v", got[0].ResetAt, wantReset)
	}
}

func TestUsageDetectorKiroCreditsNotApproaching(t *testing.T) {
	text := "Estimated Usage | resets on 2026-08-01 | KIRO PRO+ / Credits (895.04 of 2000 covered in plan) / 44.8% / Overages: Disabled"

	got := NewUsageDetector(0.10).Detect("kiro", text)

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "kiro", "", QuotaCredits, 55.248, false)
	wantReset := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if got[0].ResetAt == nil || !got[0].ResetAt.Equal(wantReset) {
		t.Fatalf("ResetAt = %v, want %v", got[0].ResetAt, wantReset)
	}
}

func TestUsageDetectorKiroCreditsApproaching(t *testing.T) {
	text := "Estimated Usage | resets on 2026-08-01 | KIRO PRO+ / Credits (1950 of 2000 covered in plan) / Overages: Disabled"

	got := NewUsageDetector(0.10).Detect("kiro", text)

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "kiro", "", QuotaCredits, 2.5, true)
	if got[0].ResetAt == nil {
		t.Fatal("ResetAt = nil, want monthly reset date")
	}
}

func TestUsageDetectorAntigravityModels(t *testing.T) {
	d := NewUsageDetector(0.10)
	now := time.Date(2026, 7, 1, 18, 30, 0, 0, time.UTC)
	d.parsers["antigravity"] = antigravityUsageParser{now: func() time.Time { return now }}

	text := `Models & Quota
Gemini 3.5 Flash (High) [████████] 100.00% / Quota available
Claude Sonnet 4.6 (Thinking) [███████░] 96.43% / 96% remaining · Refreshes in 1h 29m
Claude Opus 4.6 (Thinking) [░░░░░░░░] 8.00% / 8% remaining · Refreshes in 0h 42m`

	got := d.Detect("antigravity", text)

	if len(got) != 3 {
		t.Fatalf("samples = %d, want 3: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "antigravity", "Gemini 3.5 Flash (High)", QuotaPerModel, 100, false)
	if got[0].ResetAt != nil {
		t.Fatalf("available ResetAt = %v, want nil", got[0].ResetAt)
	}
	assertUsageSample(t, got[1], "antigravity", "Claude Sonnet 4.6 (Thinking)", QuotaPerModel, 96, false)
	wantSonnetReset := now.Add(time.Hour + 29*time.Minute)
	if got[1].ResetAt == nil || !got[1].ResetAt.Equal(wantSonnetReset) {
		t.Fatalf("sonnet ResetAt = %v, want %v", got[1].ResetAt, wantSonnetReset)
	}
	assertUsageSample(t, got[2], "antigravity", "Claude Opus 4.6 (Thinking)", QuotaPerModel, 8, true)
}

func TestUsageDetectorClineRateLimitOnly(t *testing.T) {
	got := NewUsageDetector(0.10).Detect("cline", "ClinePass/glm-5.2 (high) / Plan / ████████ (0 tokens) $0.00 (included with subscription)")

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	assertUsageSample(t, got[0], "cline", "", QuotaRateLimit, 0, false)
}

func TestUsageDetectorUnknownVendor(t *testing.T) {
	got := NewUsageDetector(0.10).Detect("unknown", "5h limit: 10% left (resets 19:59)")
	if len(got) != 0 {
		t.Fatalf("samples = %d, want 0: %+v", len(got), got)
	}
}

func TestUsageDetectorInvalidThresholdUsesDefault(t *testing.T) {
	got := NewUsageDetector(2).Detect("codex", "Heads up, you have less than 10% of your 5h limit left. Run /status for details.")

	if len(got) != 1 {
		t.Fatalf("samples = %d, want 1: %+v", len(got), got)
	}
	if !got[0].Approaching {
		t.Fatalf("Approaching = false, want true with default 10%% threshold: %+v", got[0])
	}
}

func assertUsageSample(t *testing.T, got UsageSample, vendor, model string, quotaModel QuotaModel, percent float64, approaching bool) {
	t.Helper()
	if got.Vendor != vendor {
		t.Fatalf("Vendor = %q, want %q", got.Vendor, vendor)
	}
	if got.Model != model {
		t.Fatalf("Model = %q, want %q", got.Model, model)
	}
	if got.QuotaModel != quotaModel {
		t.Fatalf("QuotaModel = %q, want %q", got.QuotaModel, quotaModel)
	}
	if math.Abs(got.PercentRemaining-percent) > 0.001 {
		t.Fatalf("PercentRemaining = %.6f, want %.6f", got.PercentRemaining, percent)
	}
	if got.Approaching != approaching {
		t.Fatalf("Approaching = %v, want %v for %+v", got.Approaching, approaching, got)
	}
	if got.Raw == "" {
		t.Fatal("Raw = empty, want matched source text")
	}
}
