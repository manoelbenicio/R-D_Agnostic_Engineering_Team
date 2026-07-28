package daemon

import (
	"encoding/json"
	"strings"
	"testing"
)

// ORQ-13 phase 1: the reasoning tier attached to usage rows must come from the
// agent record and never from the model id.

func TestUsageThinkingLevelFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task Task
		want string
	}{
		{
			name: "no agent yields not-declared",
			task: Task{ID: "t1"},
			want: "",
		},
		{
			name: "agent without tier yields not-declared",
			task: Task{ID: "t2", Agent: &AgentData{Name: "a"}},
			want: "",
		},
		{
			name: "agent tier is used verbatim",
			task: Task{ID: "t3", Agent: &AgentData{Name: "a", ThinkingLevel: "high"}},
			want: "high",
		},
		{
			name: "thinking is a tier like any other",
			task: Task{ID: "t4", Agent: &AgentData{Name: "a", ThinkingLevel: "thinking"}},
			want: "thinking",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := usageThinkingLevelFor(tc.task); got != tc.want {
				t.Fatalf("usageThinkingLevelFor() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestUsageThinkingLevelIgnoresModelSuffix is the guard that gives this change
// its point: a model id carrying `-high` or `-thinking` must NOT produce a tier
// on its own. Cost attribution has to come from declared config, because
// OmniRoute also serves ids with no tier suffix at all (e.g. auto/best-coding),
// so suffix inference would silently misattribute spend.
func TestUsageThinkingLevelIgnoresModelSuffix(t *testing.T) {
	t.Parallel()

	task := Task{ID: "t5", Agent: &AgentData{Name: "a", Model: "gemini-3.6-flash-high"}}
	if got := usageThinkingLevelFor(task); got != "" {
		t.Fatalf("tier was inferred from the model id: got %q, want %q", got, "")
	}
}

// TestTaskUsageEntryOmitsEmptyThinkingLevel locks backward compatibility: a
// backend older than ORQ-13 must receive byte-for-byte the payload it saw
// before, so the new field may not appear when it is empty.
func TestTaskUsageEntryOmitsEmptyThinkingLevel(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(TaskUsageEntry{
		Provider:     "anthropic",
		Model:        "claude-opus-4-6",
		InputTokens:  10,
		OutputTokens: 5,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "thinking_level") {
		t.Fatalf("empty tier leaked onto the wire: %s", raw)
	}
}

func TestTaskUsageEntryEmitsThinkingLevelWhenSet(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(TaskUsageEntry{
		Provider:      "anthropic",
		Model:         "claude-opus-4-6",
		InputTokens:   10,
		ThinkingLevel: "thinking",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"thinking_level":"thinking"`) {
		t.Fatalf("tier missing from payload: %s", raw)
	}
}

// TestTaskUsageEntryDecodesLegacyPayload proves the reverse direction: a payload
// produced before this change still decodes, with the tier absent.
func TestTaskUsageEntryDecodesLegacyPayload(t *testing.T) {
	t.Parallel()

	const legacy = `{"provider":"openai","model":"gpt-5.4","input_tokens":1,"output_tokens":2,"cache_read_tokens":3,"cache_write_tokens":4}`

	var entry TaskUsageEntry
	if err := json.Unmarshal([]byte(legacy), &entry); err != nil {
		t.Fatalf("unmarshal legacy payload: %v", err)
	}
	if entry.ThinkingLevel != "" {
		t.Fatalf("legacy payload produced a tier: %q", entry.ThinkingLevel)
	}
	if entry.CacheWriteTokens != 4 {
		t.Fatalf("legacy fields were not preserved: %+v", entry)
	}
}
