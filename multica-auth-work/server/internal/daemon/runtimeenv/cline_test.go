package runtimeenv

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const (
	clineTestRoot      = "http://127.0.0.1:20128"
	clineTestUpdatedAt = "2026-07-19T00:00:00Z"
)

// clineAgentBrainRoutes are the Agent Brain-selectable Cline routes:
// Cline -> Kimi-K2.7 and Cline -> GLM-5.2. No credential material is used.
var clineAgentBrainRoutes = []string{
	"cline-kimi-k2.7-dedicated",
	"cp/cline-pass/glm-5.2",
}

func TestNewClineConfigContractKimiAndGLMRoutes(t *testing.T) {
	for _, model := range clineAgentBrainRoutes {
		contract, err := NewClineConfigContract(clineTestRoot, brain.RouteModel(model), clineTestUpdatedAt)
		if err != nil {
			t.Fatalf("route %q: unexpected error: %v", model, err)
		}
		if got := contract.BaseURL(); got != clineTestRoot+"/v1" {
			t.Fatalf("route %q: base URL = %q, want %q", model, got, clineTestRoot+"/v1")
		}
		if got := contract.CredentialEnvKey(); got != ClineOmniRouteAPIKeyEnv {
			t.Fatalf("route %q: credential env key = %q, want %q", model, got, ClineOmniRouteAPIKeyEnv)
		}
		if err := contract.Validate(); err != nil {
			t.Fatalf("route %q: self-validation failed: %v", model, err)
		}
		var generic map[string]any
		if err := json.Unmarshal(contract.Bytes(), &generic); err != nil {
			t.Fatalf("route %q: generated config is not valid JSON: %v", model, err)
		}
	}
}

func TestNewClineConfigContractMatchesInstalledCarrierSchema(t *testing.T) {
	contract, err := NewClineConfigContract(clineTestRoot, brain.RouteModel("cp/cline-pass/glm-5.2"), clineTestUpdatedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var doc clineProvidersDocument
	if err := json.Unmarshal(contract.Bytes(), &doc); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}
	if doc.Version != clineProvidersSchemaVersion {
		t.Fatalf("version = %d, want %d", doc.Version, clineProvidersSchemaVersion)
	}
	if doc.LastUsedProvider != ClineOpenAICompatibleProviderID {
		t.Fatalf("lastUsedProvider = %q", doc.LastUsedProvider)
	}
	entry, ok := doc.Providers[ClineOpenAICompatibleProviderID]
	if !ok {
		t.Fatal("providers is missing the openai-compatible entry")
	}
	if entry.Settings.Provider != ClineOpenAICompatibleProviderID {
		t.Fatalf("settings.provider = %q, want openai-compatible", entry.Settings.Provider)
	}
	if entry.Settings.Model != "cp/cline-pass/glm-5.2" {
		t.Fatalf("settings.model = %q", entry.Settings.Model)
	}
	if entry.Settings.BaseURL != clineTestRoot+"/v1" {
		t.Fatalf("settings.baseUrl = %q", entry.Settings.BaseURL)
	}
	if entry.TokenSource != ClineTokenSourceOmniRoute {
		t.Fatalf("tokenSource = %q", entry.TokenSource)
	}
}

func TestNewClineConfigContractEmbedsNoSecretValue(t *testing.T) {
	contract, err := NewClineConfigContract(clineTestRoot, brain.RouteModel("cline-kimi-k2.7-dedicated"), clineTestUpdatedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var doc clineProvidersDocument
	if err := json.Unmarshal(contract.Bytes(), &doc); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}
	got := doc.Providers[ClineOpenAICompatibleProviderID].Settings.APIKey
	if got != ClineSecretReferenceSentinel {
		t.Fatalf("apiKey must be the reference sentinel, got %q", got)
	}
	// The sentinel references the env var by name only; the raw config must not
	// contain any inline provider key material beyond the sentinel token.
	if !strings.Contains(string(contract.Bytes()), ClineSecretReferenceSentinel) {
		t.Fatal("reference sentinel missing from generated config")
	}
}

func TestNewClineConfigContractRejectsNVIDIAFallbackRoute(t *testing.T) {
	for _, model := range []string{"nvidia/z-ai/glm-5.2", "NVIDIA/z-ai/glm-5.2", "nvidia/other"} {
		if _, err := NewClineConfigContract(clineTestRoot, brain.RouteModel(model), clineTestUpdatedAt); err != ErrClineRouteNotAgentBrainSelectable {
			t.Fatalf("route %q: expected ErrClineRouteNotAgentBrainSelectable, got %v", model, err)
		}
	}
}

func TestNewClineConfigContractRejectsInvalidInputs(t *testing.T) {
	if _, err := NewClineConfigContract(clineTestRoot, brain.RouteModel(""), clineTestUpdatedAt); err == nil {
		t.Fatal("empty route model must be rejected")
	}
	if _, err := NewClineConfigContract("", brain.RouteModel("cp/cline-pass/glm-5.2"), clineTestUpdatedAt); err == nil {
		t.Fatal("empty gateway root must be rejected")
	}
	if _, err := NewClineConfigContract(clineTestRoot, brain.RouteModel("cp/cline-pass/glm-5.2"), ""); err == nil {
		t.Fatal("empty updatedAt must be rejected")
	}
}

func TestValidateClineConfigBytesDetectsTamper(t *testing.T) {
	contract, err := NewClineConfigContract(clineTestRoot, brain.RouteModel("cp/cline-pass/glm-5.2"), clineTestUpdatedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateClineConfigBytes(contract.Bytes(), contract.Model(), contract.BaseURL(), clineTestUpdatedAt); err != nil {
		t.Fatalf("baseline validation failed: %v", err)
	}
	// Embedding a concrete apiKey value (a real secret) must be rejected.
	tampered := strings.Replace(string(contract.Bytes()), ClineSecretReferenceSentinel, "sk-embedded-secret", 1)
	if err := ValidateClineConfigBytes([]byte(tampered), contract.Model(), contract.BaseURL(), clineTestUpdatedAt); err != ErrClineConfigContract {
		t.Fatalf("embedded apiKey value must be rejected, got %v", err)
	}
	// An unexpected extra top-level key must be rejected.
	extra := `{"version":1,"lastUsedProvider":"openai-compatible","providers":{"openai-compatible":{"settings":{"provider":"openai-compatible","apiKey":"` + ClineSecretReferenceSentinel + `","model":"cp/cline-pass/glm-5.2","baseUrl":"http://127.0.0.1:20128/v1"},"updatedAt":"` + clineTestUpdatedAt + `","tokenSource":"omniroute-stable-secret"}},"extra":1}`
	if err := ValidateClineConfigBytes([]byte(extra), contract.Model(), contract.BaseURL(), clineTestUpdatedAt); err != ErrClineConfigContract {
		t.Fatalf("extra top-level key must be rejected, got %v", err)
	}
	// A model mismatch must be rejected.
	if err := ValidateClineConfigBytes(contract.Bytes(), brain.RouteModel("other/model"), contract.BaseURL(), clineTestUpdatedAt); err != ErrClineConfigContract {
		t.Fatalf("model mismatch must be rejected, got %v", err)
	}
	// An NVIDIA-owned route must be rejected even if the document is otherwise valid.
	if err := ValidateClineConfigBytes(contract.Bytes(), brain.RouteModel("nvidia/z-ai/glm-5.2"), contract.BaseURL(), clineTestUpdatedAt); err != ErrClineRouteNotAgentBrainSelectable {
		t.Fatalf("nvidia route must be rejected, got %v", err)
	}
}
