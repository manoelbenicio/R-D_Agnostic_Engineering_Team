package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type CompatibilitySurface string

const (
	SurfaceDaemonAPI    CompatibilitySurface = "legacy-daemon-api"
	SurfaceTaskToken    CompatibilitySurface = "legacy-task-token"
	SurfaceRouterOwner  CompatibilitySurface = "legacy-router-owner"
	SurfaceEnvironment  CompatibilitySurface = "legacy-environment"
	SurfaceStoredConfig CompatibilitySurface = "legacy-stored-config"
	SurfaceCLICommand   CompatibilitySurface = "legacy-cli-command"
	SurfaceRuntimeBrief CompatibilitySurface = "legacy-runtime-brief"
)

type CompatibilitySurfaceDefinition struct {
	Surface     CompatibilitySurface
	LegacyName  string
	NeutralName string
	RemovalGate string
}

func FrozenCompatibilitySurfaces() []CompatibilitySurfaceDefinition {
	return []CompatibilitySurfaceDefinition{
		{SurfaceDaemonAPI, "provider/model", "cli_kind/route_model", "zero-use telemetry and migrated control API"},
		{SurfaceTaskToken, "auth_token and MULTICA_TOKEN", "task-scoped opaque control token", "all CLI consumers migrated"},
		{SurfaceRouterOwner, "runtime_router_owner", "router_owner", "legacy tasks drained"},
		{SurfaceEnvironment, "MULTICA_*", "AGENT_BRAIN_*", "neutral environment adoption reaches zero legacy use"},
		{SurfaceStoredConfig, "multica stored config", "agent-brain stored config", "deterministic migration completed"},
		{SurfaceCLICommand, "multica daemon", "agent-brain daemon", "neutral command released and consumers migrated"},
		{SurfaceRuntimeBrief, "Multica Agent Runtime brief", "Agent Brain runtime brief", "brief compatibility telemetry reaches zero"},
	}
}

// TaskToken keeps the legacy task-scoped token opaque and redacted. The raw
// value is available only inside a narrowly scoped callback.
type TaskToken struct {
	value string
}

func ParseLegacyTaskToken(raw string) (TaskToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "mat_") {
		return TaskToken{}, fmt.Errorf("legacy task token is missing or not task-scoped")
	}
	return TaskToken{value: raw}, nil
}

func (TaskToken) String() string { return "[redacted]" }

func (TaskToken) MarshalJSON() ([]byte, error) { return json.Marshal("[redacted]") }

func (t TaskToken) WithValue(use func(string) error) error {
	if t.value == "" {
		return fmt.Errorf("task token is empty")
	}
	if use == nil {
		return fmt.Errorf("task token consumer is nil")
	}
	return use(t.value)
}

type LegacyTaskInput struct {
	Provider           string
	Model              string
	RuntimeRouterOwner string
	AuthToken          string
}

type TranslatedTask struct {
	Request TaskRequest
	Token   TaskToken
}

func TranslateLegacyTask(input LegacyTaskInput, correlation Correlation, policyID string, gatewayRequired bool) (TranslatedTask, error) {
	kind, err := LegacyProviderCLIKind(input.Provider)
	if err != nil {
		return TranslatedTask{}, err
	}
	model, err := ParseRouteModel(input.Model)
	if err != nil {
		return TranslatedTask{}, err
	}
	owner, err := translateLegacyRouterOwner(input.RuntimeRouterOwner, gatewayRequired)
	if err != nil {
		return TranslatedTask{}, err
	}
	token, err := ParseLegacyTaskToken(input.AuthToken)
	if err != nil {
		return TranslatedTask{}, err
	}
	request := TaskRequest{
		Version:         ContractVersion,
		Correlation:     correlation,
		CLIKind:         kind,
		RouteModel:      model,
		RouterOwner:     owner,
		RoutePolicyID:   policyID,
		GatewayRequired: gatewayRequired,
	}
	if err := request.Validate(); err != nil {
		return TranslatedTask{}, err
	}
	return TranslatedTask{Request: request, Token: token}, nil
}

// LegacyProviderCLIKind maps the persisted runtime provider vocabulary to the
// frozen executable-frontend identity. It does not infer model, route,
// credential, or account ownership.
func LegacyProviderCLIKind(provider string) (CLIKind, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude":
		return CLIClaudeCode, nil
	case "codex":
		return CLICodex, nil
	case "kimi":
		return CLIKimi, nil
	case "antigravity", "agy":
		return CLIAntigravity, nil
	case "nim":
		return CLINIM, nil
	case "cline", "opencode":
		return CLIOpenAICompatible, nil
	default:
		return "", fmt.Errorf("legacy provider %q has no frozen CLI mapping", provider)
	}
}

func translateLegacyRouterOwner(raw string, gatewayRequired bool) (RouterOwner, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if gatewayRequired {
		if raw != "" && raw != string(RouterOwnerOmniRoute) {
			return "", fmt.Errorf("legacy router owner conflicts with gateway-required mode")
		}
		return RouterOwnerOmniRoute, nil
	}
	if raw == "" {
		return RouterOwnerLegacyNativeCLI, nil
	}
	return ParseRouterOwner(raw)
}

type LegacyUseEvent struct {
	Surface CompatibilitySurface
	Alias   string
	Outcome LegacyUseOutcome
}

type LegacyUseRecorder interface {
	RecordLegacyUse(context.Context, LegacyUseEvent) error
}
