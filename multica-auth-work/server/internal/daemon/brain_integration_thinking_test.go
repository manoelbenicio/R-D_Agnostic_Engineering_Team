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
	// Whitespace-only and padded values are misconfigurations, not defaults:
	// only "" selects the runtime default, so each of these must fail closed.
	for _, level := range []string{"ultra", "max", "MEDIUM", "medium ", " ", "\t", "\n", " medium"} {
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
	// Only the exact empty string is the default.
	if err := r.validateThinking(plan, ""); err != nil {
		t.Errorf("empty level must be admitted, got %v", err)
	}
}

func TestValidateThinking_ModelReasoningGateFollowsCapabilityAuthority(t *testing.T) {
	r := &agentBrainRuntime{}

	// Enriched schema (flag unset): the capability bit is an observation, so a
	// model without reasoning must refuse a level.
	t.Setenv("OMNIROUTE_DEV_MODELS_COMPAT", "")
	if !gatewayReasoningCapabilityAuthoritative() {
		t.Fatal("capability must be authoritative when the compat flag is unset")
	}
	plan := thinkingPlan(t, brain.CLICodex, "gpt-5.5", brain.ProtocolOpenAIResponses, false)
	if err := r.validateThinking(plan, "high"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("observed Reasoning=false must refuse a level, got %v", err)
	}
	if err := r.validateThinking(plan, ""); err != nil {
		t.Fatalf("empty level must remain admissible, got %v", err)
	}

	// Compat projection (the deployment that runs today): every row is
	// hardcoded Reasoning=false, so the bit carries no information. The level
	// must still be admitted when the provider allowlist accepts it, otherwise
	// production keeps rejecting every configured level.
	t.Setenv("OMNIROUTE_DEV_MODELS_COMPAT", "1")
	if gatewayReasoningCapabilityAuthoritative() {
		t.Fatal("capability must not be authoritative under the compat projection")
	}
	if err := r.validateThinking(plan, "high"); err != nil {
		t.Fatalf("compat projection must not veto an allowlisted level, got %v", err)
	}
	// The provider gate is still live under compat: an unknown level and an
	// unapproved CLI both fail closed.
	if err := r.validateThinking(plan, "ultra"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("unknown level must fail closed even under compat, got %v", err)
	}
	agy := thinkingPlan(t, brain.CLIAntigravity, "gemini-3.6-flash", brain.ProtocolAntigravity, true)
	if err := r.validateThinking(agy, "high"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("antigravity must fail closed even under compat, got %v", err)
	}
}

func TestValidateThinking_FailsClosedForUnapprovedCLIs(t *testing.T) {
	r := &agentBrainRuntime{}
	for _, kind := range []brain.CLIKind{brain.CLIAntigravity, brain.CLIKimi, brain.CLINIM, brain.CLIKind("made-up")} {
		if _, err := gatewayThinkingLevelsFor(kind); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
			t.Errorf("%s must have no gateway thinking allowlist, got %v", kind, err)
		}
	}
	// Antigravity end to end. Note the distinction: the `agy` CLI itself accepts
	// --effort low|medium|high, but the product never forwards a level to it, so
	// the gateway must refuse one instead of pretending it will be honoured.
	plan := thinkingPlan(t, brain.CLIAntigravity, "gemini-3.6-flash", brain.ProtocolAntigravity, true)
	if err := r.validateThinking(plan, "high"); !errors.Is(err, runtimeenv.ErrThinkingNotApproved) {
		t.Fatalf("antigravity must refuse a thinking level, got %v", err)
	}
}

// thinkingTokenUniverse is every effort token used by any provider enum in
// pkg/agent today, plus a few plausible-but-absent tokens. The comparison below
// is exact in both directions over this set, so adding "ultra" to claude's enum
// in pkg/agent, or dropping "max", breaks this test.
//
// Honest limitation: a token outside this list would escape the check. Making it
// exhaustive requires pkg/agent to export its enum (an immutable accessor), which
// belongs to another owner; it is requested as a handoff and deliberately not
// taken here.
var thinkingTokenUniverse = []string{
	"none", "minimal", "low", "medium", "high", "xhigh", "max",
	"ultra", "xlow", "default", "off", "on", "medium ", "MEDIUM", "",
}

func TestGatewayThinkingLevelsMatchProviderEnums(t *testing.T) {
	for provider, levels := range gatewayApprovedThinkingLevels {
		if len(levels) == 0 {
			t.Errorf("provider %q has an empty allowlist; remove the entry instead", provider)
		}
		mine := map[string]bool{}
		for _, level := range levels {
			mine[level] = true
		}
		for _, token := range thinkingTokenUniverse {
			// pkg/agent always accepts "" as "runtime default"; the gateway
			// allowlist deliberately never contains it.
			if token == "" {
				if mine[token] {
					t.Errorf("provider %q must not list the empty token", provider)
				}
				continue
			}
			known := agent.IsKnownThinkingValue(provider, token)
			if known != mine[token] {
				t.Errorf("provider %q token %q: pkg/agent knows=%v, gateway allowlist has=%v — the two drifted",
					provider, token, known, mine[token])
			}
		}
	}
	if _, ok := gatewayApprovedThinkingLevels["antigravity"]; ok {
		t.Error("antigravity must stay absent: agy accepts --effort, but the product never passes a level to it, so advertising one here would be a lie")
	}
	// pkg/agent owns the product-side contract and has no antigravity reasoning
	// entry. If that changes — i.e. the runtime is wired under its own approval —
	// this fails and the gateway allowlist must be revisited deliberately.
	for _, token := range thinkingTokenUniverse {
		if token == "" {
			continue
		}
		if agent.IsKnownThinkingValue("antigravity", token) {
			t.Errorf("pkg/agent now accepts antigravity level %q; the gateway allowlist must be revisited", token)
		}
	}
}
