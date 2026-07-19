package brain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTaskRequestSeparatesCLIModelAndOwner(t *testing.T) {
	request := TaskRequest{
		Version:         ContractVersion,
		Correlation:     Correlation{TaskID: "task", SessionID: "session", RequestID: "request"},
		CLIKind:         CLIClaudeCode,
		RouteModel:      RouteModel("agy/claude-opus-4-6-thinking"),
		RouterOwner:     RouterOwnerOmniRoute,
		RoutePolicyID:   "canary-20",
		GatewayRequired: true,
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	request.RouterOwner = RouterOwnerLegacyRustL2
	if err := request.Validate(); err == nil {
		t.Fatal("gateway-required request accepted a legacy router owner")
	}
}

func TestResolveConfigValuePrecedence(t *testing.T) {
	got, err := ResolveConfigValue(
		ConfigCandidate{Name: "legacy-stored", Value: "a", Source: SourceLegacyStored, Set: true},
		ConfigCandidate{Name: "neutral-env", Value: "b", Source: SourceNeutralEnv, Set: true},
		ConfigCandidate{Name: "legacy-env", Value: "c", Source: SourceLegacyEnv, Set: true},
	)
	if err != nil {
		t.Fatalf("ResolveConfigValue: %v", err)
	}
	if got.Name != "neutral-env" || got.Value != "b" || got.Source != SourceNeutralEnv {
		t.Fatalf("unexpected winner: %+v", got)
	}
}

func TestStrictReadinessFailsClosed(t *testing.T) {
	policy := StrictReadinessPolicy()
	ready := ReadinessSnapshot{Live: true, Authenticated: true, ModelRegistryReady: true, SelectedModelReady: true, SelectedProtocolReady: true}
	if err := policy.Evaluate(ready); err != nil {
		t.Fatalf("ready snapshot rejected: %v", err)
	}
	ready.Authenticated = false
	if err := policy.Evaluate(ready); err == nil {
		t.Fatal("unauthenticated gateway accepted")
	}
}

func TestLegacyTranslationAndTokenRedaction(t *testing.T) {
	translated, err := TranslateLegacyTask(LegacyTaskInput{
		Provider:  "claude",
		Model:     "agy/claude-opus-4-6-thinking",
		AuthToken: "mat_synthetic_test_value",
	}, Correlation{TaskID: "task", SessionID: "session", RequestID: "request"}, "canary-20", true)
	if err != nil {
		t.Fatalf("TranslateLegacyTask: %v", err)
	}
	if translated.Request.CLIKind != CLIClaudeCode || translated.Request.RouterOwner != RouterOwnerOmniRoute {
		t.Fatalf("unexpected translation: %+v", translated.Request)
	}
	encoded, err := json.Marshal(translated.Token)
	if err != nil {
		t.Fatalf("Marshal token: %v", err)
	}
	if strings.Contains(string(encoded), "synthetic_test_value") || translated.Token.String() != "[redacted]" {
		t.Fatal("task token was not redacted")
	}
}

func TestInitialRouteHasNoCrossModelFallback(t *testing.T) {
	routes := InitialModelSet()
	if len(routes) != 1 || routes[0].Approval != RouteApprovalEvidenceRequired {
		t.Fatalf("unexpected initial routes: %+v", routes)
	}
	if len(routes[0].Fallback.CrossModelFallback) != 0 || !routes[0].Fallback.PreCommitOnly {
		t.Fatalf("unsafe initial fallback: %+v", routes[0].Fallback)
	}
}
