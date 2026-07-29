package runtimeenv

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// TestOpenAICompatibleAdapterReadyForOmniRouteChat proves the accepted Cline
// (OpenAI-compatible) frontend resolves to a ready adapter that speaks OpenAI
// Chat Completions to the gateway (W1 D1). This asserts only CLI/protocol
// intent; credentials, rotation and fallback remain OmniRoute-owned and are
// never handled here. It is the positive complement to the native
// Kimi/NIM/Antigravity fail-closed guard in model_test.go / isolation_g4_test.go.
func TestOpenAICompatibleAdapterReadyForOmniRouteChat(t *testing.T) {
	contract, err := CredentiallessAdapterContract(brain.CLIOpenAICompatible)
	if err != nil {
		t.Fatalf("accepted OpenAI-compatible adapter returned error: %v", err)
	}
	if contract.State != AdapterReady {
		t.Fatalf("adapter state = %q, want %q", contract.State, AdapterReady)
	}
	if contract.Protocol != brain.ProtocolOpenAIChat {
		t.Fatalf("adapter protocol = %q, want %q", contract.Protocol, brain.ProtocolOpenAIChat)
	}
	if contract.Gate != "" {
		t.Fatalf("ready adapter must not carry a fail-closed gate, got %q", contract.Gate)
	}
	if len(contract.FallbackFrontends) != 0 {
		t.Fatalf("accepted Cline adapter must not advertise fallback frontends, got %v", contract.FallbackFrontends)
	}
}
