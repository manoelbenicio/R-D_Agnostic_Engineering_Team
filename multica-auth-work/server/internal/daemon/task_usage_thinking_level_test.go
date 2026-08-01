package daemon

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUsageThinkingLevelForUsesClaimedAgentSnapshot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		task Task
		want string
	}{
		{name: "no agent", task: Task{ID: "t1"}, want: ""},
		{name: "no tier", task: Task{ID: "t2", Agent: &AgentData{Name: "a"}}, want: ""},
		{name: "high", task: Task{ID: "t3", Agent: &AgentData{Name: "a", ThinkingLevel: "high"}}, want: "high"},
		{name: "model suffix ignored", task: Task{ID: "t4", Agent: &AgentData{Name: "a", Model: "gpt-5.4-high"}}, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := usageThinkingLevelFor(tc.task); got != tc.want {
				t.Fatalf("usageThinkingLevelFor() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTaskUsageEntryThinkingLevelWireCompatibility(t *testing.T) {
	t.Parallel()
	legacy, err := json.Marshal(TaskUsageEntry{Provider: "openai", Model: "gpt-5.4", InputTokens: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacy), "thinking_level") {
		t.Fatalf("empty tier leaked onto legacy wire shape: %s", legacy)
	}

	tiered, err := json.Marshal(TaskUsageEntry{Provider: "openai", Model: "gpt-5.4", InputTokens: 1, ThinkingLevel: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tiered), `"thinking_level":"high"`) {
		t.Fatalf("tier missing from payload: %s", tiered)
	}
}
