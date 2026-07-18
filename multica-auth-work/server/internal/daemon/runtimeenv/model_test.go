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
		{brain.CLIOpenAICompatible, GateOpenAICompatibleUnaccepted},
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
