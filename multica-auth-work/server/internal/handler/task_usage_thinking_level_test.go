package handler

import (
	"encoding/json"
	"testing"
)

// ORQ-13 phase 1: thinking_level is nullable. "Not declared" must reach the
// database as NULL, never as the empty string, so a pricing lookup can tell a
// missing tier apart from a real one.

func TestThinkingLevelText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        string
		wantValid bool
		wantStr   string
	}{
		{name: "empty becomes NULL", in: "", wantValid: false, wantStr: ""},
		{name: "whitespace only becomes NULL", in: "   ", wantValid: false, wantStr: ""},
		{name: "tab and newline become NULL", in: "\t\n", wantValid: false, wantStr: ""},
		{name: "value is stored", in: "high", wantValid: true, wantStr: "high"},
		{name: "value is trimmed", in: "  thinking  ", wantValid: true, wantStr: "thinking"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := thinkingLevelText(tc.in)
			if got.Valid != tc.wantValid {
				t.Fatalf("thinkingLevelText(%q).Valid = %v, want %v", tc.in, got.Valid, tc.wantValid)
			}
			if got.String != tc.wantStr {
				t.Fatalf("thinkingLevelText(%q).String = %q, want %q", tc.in, got.String, tc.wantStr)
			}
		})
	}
}

// TestThinkingLevelTextNeverStoresEmptyString is the invariant that keeps a
// blank tier from being charged as the model's base tier: when Valid is false
// the driver writes NULL, so String must not carry a value either.
func TestThinkingLevelTextNeverStoresEmptyString(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", " ", "\t", "\n", "  \t "} {
		got := thinkingLevelText(in)
		if got.Valid {
			t.Fatalf("blank input %q produced a non-NULL tier", in)
		}
		if got.String != "" {
			t.Fatalf("blank input %q produced String %q, want empty", in, got.String)
		}
	}
}

func TestTaskUsagePayloadDecodesThinkingLevel(t *testing.T) {
	t.Parallel()

	const body = `{"provider":"google","model":"gemini-3.1-pro","input_tokens":7,"thinking_level":"high"}`

	var payload TaskUsagePayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ThinkingLevel != "high" {
		t.Fatalf("ThinkingLevel = %q, want %q", payload.ThinkingLevel, "high")
	}
	if got := thinkingLevelText(payload.ThinkingLevel); !got.Valid || got.String != "high" {
		t.Fatalf("mapped tier = %+v, want valid high", got)
	}
}

// TestTaskUsagePayloadLegacyDaemonYieldsNull covers the mixed-version window:
// a daemon older than ORQ-13 omits the field entirely and its rows must land as
// NULL rather than as a fabricated base tier.
func TestTaskUsagePayloadLegacyDaemonYieldsNull(t *testing.T) {
	t.Parallel()

	const legacy = `{"provider":"anthropic","model":"claude-sonnet-4-6","input_tokens":1,"output_tokens":2}`

	var payload TaskUsagePayload
	if err := json.Unmarshal([]byte(legacy), &payload); err != nil {
		t.Fatalf("unmarshal legacy payload: %v", err)
	}
	if payload.ThinkingLevel != "" {
		t.Fatalf("legacy payload produced a tier: %q", payload.ThinkingLevel)
	}
	if got := thinkingLevelText(payload.ThinkingLevel); got.Valid {
		t.Fatal("legacy payload would have written a non-NULL tier")
	}
}
