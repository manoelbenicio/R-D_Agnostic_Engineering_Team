package runtimeenv

import (
	"errors"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestCodexConfigContractIsResponsesOnlyAndSecretFree(t *testing.T) {
	correlation := testCorrelation()
	model := brain.RouteModel("approved/codex-model")
	contract, err := NewCodexConfigContract("http://127.0.0.1:20128", model, correlation)
	if err != nil {
		t.Fatalf("NewCodexConfigContract returned error: %v", err)
	}
	raw := string(contract.Bytes())
	for _, required := range []string{
		`model_provider = 'omniroute'`, `base_url = 'http://127.0.0.1:20128/v1'`,
		`env_key = 'AGENT_BRAIN_OMNIROUTE_API_KEY'`, `wire_api = 'responses'`,
		`supports_websockets = false`, `X-Task-Id`, `X-Session-Id`, `X-Request-Id`,
	} {
		if !strings.Contains(raw, required) {
			t.Fatalf("generated Codex config omitted controlled field %q", required)
		}
	}
	for _, forbidden := range []string{syntheticSecret, "OPENAI_API_KEY", "experimental_bearer_token", "requires_openai_auth", "[model_providers.omniroute.auth]"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("generated Codex config contains forbidden field class %q", forbidden)
		}
	}
	if err := contract.Validate(); err != nil {
		t.Fatalf("generated Codex config failed validation: %v", err)
	}
}

func TestCodexConfigContractRejectsAlternateCredentialAndProvider(t *testing.T) {
	correlation := testCorrelation()
	model := brain.RouteModel("approved/codex-model")
	contract, err := NewCodexConfigContract("http://127.0.0.1:20128", model, correlation)
	if err != nil {
		t.Fatalf("NewCodexConfigContract returned error: %v", err)
	}
	raw := strings.Replace(string(contract.Bytes()), CodexOmniRouteAPIKeyEnv, "OPENAI_API_KEY", 1)
	if err := ValidateCodexConfigBytes([]byte(raw), model, "http://127.0.0.1:20128/v1", correlation); !errors.Is(err, ErrCodexConfigContract) {
		t.Fatalf("alternate env_key error = %v", err)
	}
	raw = string(contract.Bytes()) + "\n[model_providers.other]\nbase_url = 'https://provider.invalid/v1'\n"
	if err := ValidateCodexConfigBytes([]byte(raw), model, "http://127.0.0.1:20128/v1", correlation); !errors.Is(err, ErrCodexConfigContract) {
		t.Fatalf("extra provider error = %v", err)
	}
}

func TestCodexConfigContractRejectsGatewayPathInput(t *testing.T) {
	_, err := NewCodexConfigContract("http://127.0.0.1:20128/v1", brain.RouteModel("approved/codex-model"), testCorrelation())
	if err == nil {
		t.Fatal("expected gateway root containing /v1 to be rejected")
	}
}

func testCorrelation() brain.Correlation {
	return brain.Correlation{TaskID: "task-1", SessionID: "session-1", RequestID: "request-1"}
}
