package daemon

import (
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/runtimeenv"
	"github.com/multica-ai/multica/server/pkg/agent"
)

// thinkingPlan builds the smallest plan validateThinking needs: the CLI kind,
// the route model, the policy protocol and the model capability. Everything else
// on the plan is irrelevant to admission of a reasoning level.
func thinkingPlan(t *testing.T, kind brain.CLIKind, model string, protocol brain.ProtocolFamily, reasoning bool) *agentBrainTaskPlan {
	t.Helper()
	routeModel, err := brain.ParseRouteModel(model)
	if err != nil {
		t.Fatalf("parse route model %q: %v", model, err)
	}
	plan := &agentBrainTaskPlan{}
	plan.Task.Request.CLIKind = kind
	plan.Task.Request.RouteModel = routeModel
	plan.Task.RoutePolicy.Protocol = protocol
	plan.Capability = brain.ModelCapability{RouteModel: routeModel, Protocol: protocol, Reasoning: reasoning}
	return plan
}

// The regression this file exists for: before the fix, any non-empty persisted
// thinking_level was rejected outright, so every agent with a configured level
// failed admission and the only dispatchable configuration was NULL.
func TestValidateThinking_AdmitsApprovedLevelsPerCLI(t *testing.T) {
	r := &agentBrainRuntime{}
	cases := []struct {
		name     string
		kind     brain.CLIKind
		model    string
		protocol brain.ProtocolFamily
		levels   []string
	}{
		{"claude-code", brain.CLIClaudeCode, "claude-opus-4-6", brain.ProtocolAnthropicMessages,
			[]string{"low", "medium", "high", "xhigh", "max"}},
		{"codex", brain.CLICodex, "gpt-5.5", brain.ProtocolOpenAIResponses,
			[]string{"none", "minimal", "low", "medium", "high", "xhigh"}},
		{"openai-compatible", brain.CLIOpenAICompatible, "gpt-5.5", brain.ProtocolOpenAIChat,
			[]string{"none", "low", "medium", "high", "xhigh"}},
	}
	for _, tc := range cases {
		for _, level := range tc.levels {
			plan := thinkingPlan(t, tc.kind, tc.model, tc.protocol, true)
			if err := r.validateThinking(plan, level); err != nil {
				t.Errorf("%s: level %q must be admitted, got %v", tc.name, level, err)
			}
		}
	}
}

// Kiro is a real gap, not an oversight in this test: it has no CLIKind in the
// frozen gateway identity set, so under gateway-required admission it cannot
// dispatch at all. Its reasoning support (`--effort`) lives on the native path,
// which validateThinking leaves untouched. The two assertions below pin both
// halves of that contract so a future CLIKind addition breaks loudly here.
func TestValidateThinking_KiroDispatchesOnlyOnNativePath(t *testing.T) {
	r := &agentBrainRuntime{}

	if _, err := brain.LegacyProviderCLIKind("kiro"); err == nil {
		t.Fatal("kiro gained a frozen gateway CLIKind; gatewayApprovedThinkingLevels must be extended")
	}
	if err := r.validateThinking(nil, "high"); err != nil {
		t.Fatalf("native path must accept the persisted level untouched, got %v", err)
	}
}

func TestValidateThinking_RejectsUnknownLevelBeforeEnqueue(t *testing.T) {
	r := &agentBrainRuntime{}
	for _, level := range []string{"ultra", "max", "MEDIUM", "medium "} {
		plan := thinkingPlan(t, brain.CLICodex, "gpt-5.5", brain.ProtocolOpenAIResponses, true)
		err := r.validateThinking(plan, level)
		if !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
			t.Errorf("codex level %q must be refused with ErrThinkingNotApproved, got %v", level, err)
		}
	}
	// "max" is valid for claude and invalid for codex; the allowlist is
	// per-provider, not global.
	plan := thinkingPlan(t, brain.CLIClaudeCode, "claude-opus-4-6", brain.ProtocolAnthropicMessages, true)
	if err := r.validateThinking(plan, "max"); err != nil {
		t.Errorf("claude must accept max, got %v", err)
	}
}

func TestValidateThinking_FailsClosedWithoutModelReasoning(t *testing.T) {
	r := &agentBrainRuntime{}
	plan := thinkingPlan(t, brain.CLICodex, "gpt-5.5", brain.ProtocolOpenAIResponses, false)
	if err := r.validateThinking(plan, "high"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("a model without reasoning capability must refuse a level, got %v", err)
	}
	// The same model with no level configured still admits normally, so the gate
	// costs nothing to non-reasoning workloads.
	if err := r.validateThinking(plan, ""); err != nil {
		t.Fatalf("empty level must remain admissible, got %v", err)
	}
}

func TestValidateThinking_FailsClosedForUnapprovedCLIs(t *testing.T) {
	r := &agentBrainRuntime{}
	for _, kind := range []brain.CLIKind{brain.CLIAntigravity, brain.CLIKimi, brain.CLINIM, brain.CLIKind("made-up")} {
		if _, err := gatewayThinkingLevelsFor(kind); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
			t.Errorf("%s must have no gateway thinking allowlist, got %v", kind, err)
		}
	}
	// Antigravity end to end: reasoning is model-embedded, never a level.
	plan := thinkingPlan(t, brain.CLIAntigravity, "gemini-3.6-flash", brain.ProtocolAntigravity, true)
	if err := r.validateThinking(plan, "high"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("antigravity must refuse a thinking level, got %v", err)
	}
}

// Guards against the allowlist drifting away from pkg/agent, which owns what a
// level means for each provider.
func TestGatewayThinkingLevelsMatchProviderEnums(t *testing.T) {
	for provider, levels := range gatewayApprovedThinkingLevels {
		if len(levels) == 0 {
			t.Errorf("provider %q has an empty allowlist; remove the entry instead", provider)
		}
		for _, level := range levels {
			if !agent.IsKnownThinkingValue(provider, level) {
				t.Errorf("provider %q level %q is not recognised by pkg/agent", provider, level)
			}
		}
	}
	if _, ok := gatewayApprovedThinkingLevels["antigravity"]; ok {
		t.Error("antigravity must stay absent: it has no effort flag")
	}
}
