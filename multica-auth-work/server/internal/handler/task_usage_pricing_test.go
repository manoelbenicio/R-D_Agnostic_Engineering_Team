package handler

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestThinkingLevelText(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in    string
		valid bool
		value string
	}{
		{in: "", valid: false},
		{in: "  ", valid: false},
		{in: " HIGH ", valid: true, value: "high"},
	} {
		got := thinkingLevelText(tc.in)
		if got.Valid != tc.valid || got.String != tc.value {
			t.Fatalf("thinkingLevelText(%q) = %+v, want valid=%v value=%q", tc.in, got, tc.valid, tc.value)
		}
	}
}

func TestTaskUsageEffectiveAtPrefersFrozenStart(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	started := created.Add(time.Hour)
	task := db.AgentTaskQueue{
		CreatedAt: pgtype.Timestamptz{Time: created, Valid: true},
		StartedAt: pgtype.Timestamptz{Time: started, Valid: true},
	}
	if got := taskUsageEffectiveAt(task); !got.Equal(started) {
		t.Fatalf("effective time = %v, want %v", got, started)
	}
}

func TestTaskUsagePriceSnapshotTierAware(t *testing.T) {
	t.Parallel()
	effective := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	base := TaskUsagePayload{Model: "gpt-5.4", InputTokens: 1_000_000, OutputTokens: 1_000_000}

	low := base
	low.ThinkingLevel = "low"
	lowVersion, lowCost := taskUsagePriceSnapshot(low, effective)
	if !lowVersion.Valid || !lowCost.Valid || lowCost.Float64 <= 0 {
		t.Fatalf("low snapshot invalid: version=%+v cost=%+v", lowVersion, lowCost)
	}

	high := base
	high.ThinkingLevel = "high"
	highVersion, highCost := taskUsagePriceSnapshot(high, effective)
	if !highVersion.Valid || !highCost.Valid || highCost.Float64 <= lowCost.Float64 {
		t.Fatalf("high snapshot must exceed low: low=%+v high=%+v", lowCost, highCost)
	}
	if highVersion.String != lowVersion.String {
		t.Fatalf("tiers resolved different catalog versions: low=%q high=%q", lowVersion.String, highVersion.String)
	}

	for _, tier := range []string{"", "unsupported"} {
		missing := base
		missing.ThinkingLevel = tier
		version, cost := taskUsagePriceSnapshot(missing, effective)
		if version.Valid || cost.Valid {
			t.Fatalf("tier %q silently priced: version=%+v cost=%+v", tier, version, cost)
		}
	}
}
