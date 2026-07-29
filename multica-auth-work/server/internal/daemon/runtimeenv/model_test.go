package runtimeenv

import (
	"errors"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestGatewayModelPolicyAcceptsApprovedOmniRouteIDWithoutNativeDiscovery(t *testing.T) {
	model := brain.RouteModel("agy/claude-opus-4-6-thinking")
	policy, err := NewGatewayModelPolicy([]ApprovedGatewayModel{{
		Model: model, Protocol: brain.ProtocolAnthropicMessages,
		CLIs: []brain.CLIKind{brain.CLIClaudeCode}, ThinkingLevels: []string{"high"},
	}})
	if err != nil {
		t.Fatalf("NewGatewayModelPolicy returned error: %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIClaudeCode, model, "high"); err != nil {
		t.Fatalf("approved gateway selection rejected: %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIClaudeCode, model, "unapproved"); !errors.Is(err, ErrThinkingNotApproved) {
		t.Fatalf("thinking error = %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIClaudeCode, brain.RouteModel("agy/other"), ""); !errors.Is(err, ErrModelNotApproved) {
		t.Fatalf("unapproved model error = %v", err)
	}
}

func TestCredentialBearingNativeAdaptersFailClosed(t *testing.T) {
	tests := []struct {
		cli  brain.CLIKind
		gate AdapterGate
	}{
		// CLIOpenAICompatible (Cline) is now an ACCEPTED gateway adapter (W1 D1)
		// and is asserted ready in adapter_test.go; it is intentionally no longer
		// in this fail-closed table. The remaining native adapters stay fail-closed.
		{brain.CLIKimi, GateNativeKimiUnaccepted},
		{brain.CLINIM, GateNativeNIMUnaccepted},
		{brain.CLIAntigravity, GateNativeAntigravityUnaccepted},
	}
	for _, test := range tests {
		t.Run(string(test.cli), func(t *testing.T) {
			contract, err := CredentiallessAdapterContract(test.cli)
			if !errors.Is(err, ErrAdapterFailClosed) {
				t.Fatalf("adapter error = %v", err)
			}
			if contract.State != AdapterFailClosed || contract.Gate != test.gate {
				t.Fatalf("adapter contract = %+v", contract)
			}
		})
	}
}

func TestNativeFallbackIsNeverAutomatic(t *testing.T) {
	contract, _ := CredentiallessAdapterContract(brain.CLIAntigravity)
	if len(contract.FallbackFrontends) != 2 || contract.State != AdapterFailClosed {
		t.Fatalf("Antigravity stub contract = %+v", contract)
	}
	policy, err := NewGatewayModelPolicy([]ApprovedGatewayModel{{
		Model:    brain.RouteModel("agy/claude-opus-4-6-thinking"),
		Protocol: brain.ProtocolAnthropicMessages, CLIs: []brain.CLIKind{brain.CLIClaudeCode},
	}})
	if err != nil {
		t.Fatalf("NewGatewayModelPolicy returned error: %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIAntigravity, brain.RouteModel("agy/claude-opus-4-6-thinking"), ""); !errors.Is(err, ErrAdapterFailClosed) {
		t.Fatalf("native Antigravity validation error = %v", err)
	}
}

// TestGatewayModelPolicyAcceptsClineGLMOverChat proves the accepted Cline
// (OpenAI-compatible) frontend validates the A3-FROZEN GLM RouteModel
// cp/cline-pass/glm-5.2 over OpenAI Chat Completions, and fail-closes on the
// wrong CLI, an unapproved thinking level, and an unknown model. The exact
// GLM id is consumed from the A3 route freeze; no id is invented here.
func TestGatewayModelPolicyAcceptsClineGLMOverChat(t *testing.T) {
	model := brain.RouteModel("cp/cline-pass/glm-5.2")
	policy, err := NewGatewayModelPolicy([]ApprovedGatewayModel{{
		Model: model, Protocol: brain.ProtocolOpenAIChat,
		CLIs: []brain.CLIKind{brain.CLIOpenAICompatible},
	}})
	if err != nil {
		t.Fatalf("NewGatewayModelPolicy returned error: %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIOpenAICompatible, model, ""); err != nil {
		t.Fatalf("approved Cline GLM selection rejected: %v", err)
	}
	if err := policy.ValidateSelection(brain.CLIClaudeCode, model, ""); !errors.Is(err, ErrCLIModelNotApproved) {
		t.Fatalf("wrong-CLI selection error = %v, want ErrCLIModelNotApproved", err)
	}
	if err := policy.ValidateSelection(brain.CLIOpenAICompatible, model, "high"); !errors.Is(err, ErrThinkingNotApproved) {
		t.Fatalf("unapproved thinking error = %v, want ErrThinkingNotApproved", err)
	}
	if err := policy.ValidateSelection(brain.CLIOpenAICompatible, brain.RouteModel("cp/cline-pass/other"), ""); !errors.Is(err, ErrModelNotApproved) {
		t.Fatalf("unknown model error = %v, want ErrModelNotApproved", err)
	}
}

// TestGatewayModelPolicyClineKimiHeldPendingExactID documents that the
// Cline -> Kimi-K2.7 (task 5.7) model-selection assertion is HELD: the exact
// OmniRoute Kimi RouteModel is UNDECLARED per the A3 route freeze (BLK-KIMI).
// It must not be guessed or silently defaulted; this test unblocks with one
// edit once OmniRoute publishes the exact versioned Kimi id.
func TestGatewayModelPolicyClineKimiHeldPendingExactID(t *testing.T) {
	t.Skip("BLK-KIMI: exact OmniRoute Cline->Kimi-K2.7 RouteModel is UNDECLARED (A3 route freeze); not invented here")
}
