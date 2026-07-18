package brain

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	EnvControlURL              = "AGENT_BRAIN_CONTROL_URL"
	EnvGatewayRequired         = "AGENT_BRAIN_GATEWAY_REQUIRED"
	EnvGatewayBaseURL          = "AGENT_BRAIN_GATEWAY_BASE_URL"
	EnvGatewaySecretFile       = "AGENT_BRAIN_GATEWAY_SECRET_FILE"
	EnvGatewayReadiness        = "AGENT_BRAIN_GATEWAY_READINESS_POLICY"
	EnvTaskCapacityTier        = "AGENT_BRAIN_TASK_CAPACITY_TIER"
	EnvLegacyExecution         = "AGENT_BRAIN_LEGACY_EXECUTION_ENABLED"
	ChildEnvOmniRouteAPIKey    = "AGENT_BRAIN_OMNIROUTE_API_KEY"
	DefaultHostGatewayURL      = "http://127.0.0.1:20128"
	DefaultContainerGatewayURL = "http://omniroute:20128"
)

type RuntimeTopology string

const (
	TopologyHostWSL   RuntimeTopology = "host-wsl"
	TopologyContainer RuntimeTopology = "container"
)

func DefaultGatewayURL(topology RuntimeTopology) (string, error) {
	switch topology {
	case TopologyHostWSL:
		return DefaultHostGatewayURL, nil
	case TopologyContainer:
		return DefaultContainerGatewayURL, nil
	default:
		return "", fmt.Errorf("unsupported runtime topology %q", topology)
	}
}

type CapacityTier int

const (
	CapacityTier20  CapacityTier = 20
	CapacityTier50  CapacityTier = 50
	CapacityTier100 CapacityTier = 100
)

type TierState string

const (
	TierStateCanaryAuthorized TierState = "canary-authorized"
	TierStateEvidenceRequired TierState = "evidence-required"
)

type TierDefinition struct {
	Tier           CapacityTier
	MaxActiveTasks int
	State          TierState
}

func FrozenTierSchema() []TierDefinition {
	return []TierDefinition{
		{Tier: CapacityTier20, MaxActiveTasks: 20, State: TierStateCanaryAuthorized},
		{Tier: CapacityTier50, MaxActiveTasks: 50, State: TierStateEvidenceRequired},
		{Tier: CapacityTier100, MaxActiveTasks: 100, State: TierStateEvidenceRequired},
	}
}

func (t CapacityTier) Validate() error {
	switch t {
	case CapacityTier20, CapacityTier50, CapacityTier100:
		return nil
	default:
		return fmt.Errorf("capacity tier must be one of 20, 50, or 100")
	}
}

type SecretFileRef struct {
	Path string
}

func NewSecretFileRef(path string) (SecretFileRef, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return SecretFileRef{}, fmt.Errorf("gateway secret file reference is required")
	}
	if !filepath.IsAbs(path) {
		return SecretFileRef{}, fmt.Errorf("gateway secret file reference must be absolute")
	}
	return SecretFileRef{Path: filepath.Clean(path)}, nil
}

type ReadinessPolicyName string

const ReadinessStrict ReadinessPolicyName = "strict"

type ReadinessPolicy struct {
	Name                    ReadinessPolicyName
	Timeout                 time.Duration
	RequireLiveness         bool
	RequireAuthentication   bool
	RequireModelRegistry    bool
	RequireSelectedModel    bool
	RequireSelectedProtocol bool
	FailClosed              bool
}

func StrictReadinessPolicy() ReadinessPolicy {
	return ReadinessPolicy{
		Name:                    ReadinessStrict,
		Timeout:                 5 * time.Second,
		RequireLiveness:         true,
		RequireAuthentication:   true,
		RequireModelRegistry:    true,
		RequireSelectedModel:    true,
		RequireSelectedProtocol: true,
		FailClosed:              true,
	}
}

type ReadinessRequest struct {
	RouteModel RouteModel
	Protocol   ProtocolFamily
}

type ReadinessSnapshot struct {
	Live                  bool
	Authenticated         bool
	ModelRegistryReady    bool
	SelectedModelReady    bool
	SelectedProtocolReady bool
}

func (p ReadinessPolicy) Evaluate(snapshot ReadinessSnapshot) error {
	checks := []struct {
		required bool
		ready    bool
		name     string
	}{
		{p.RequireLiveness, snapshot.Live, "liveness"},
		{p.RequireAuthentication, snapshot.Authenticated, "authentication"},
		{p.RequireModelRegistry, snapshot.ModelRegistryReady, "model registry"},
		{p.RequireSelectedModel, snapshot.SelectedModelReady, "selected model"},
		{p.RequireSelectedProtocol, snapshot.SelectedProtocolReady, "selected protocol"},
	}
	for _, check := range checks {
		if check.required && !check.ready {
			return fmt.Errorf("gateway readiness failed: %s unavailable", check.name)
		}
	}
	return nil
}

type GatewayConfig struct {
	Required   bool
	BaseURL    string
	SecretFile SecretFileRef
	Readiness  ReadinessPolicy
}

func (c GatewayConfig) Validate() error {
	parsed, err := url.Parse(strings.TrimSpace(c.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("gateway base URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("gateway base URL scheme must be http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("gateway base URL must not contain user info, query, or fragment")
	}
	if c.Required && c.SecretFile.Path == "" {
		return fmt.Errorf("gateway-required mode requires a secret file reference")
	}
	if c.Required && (!c.Readiness.FailClosed || c.Readiness.Name != ReadinessStrict) {
		return fmt.Errorf("gateway-required mode requires strict fail-closed readiness")
	}
	return nil
}

type Config struct {
	ControlURL      string
	Gateway         GatewayConfig
	CapacityTier    CapacityTier
	LegacyExecution bool
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ControlURL) == "" {
		return fmt.Errorf("control URL is required")
	}
	if err := c.Gateway.Validate(); err != nil {
		return err
	}
	return c.CapacityTier.Validate()
}

type ValueSource string

const (
	SourceNeutralCLI    ValueSource = "neutral-cli"
	SourceLegacyCLI     ValueSource = "legacy-cli"
	SourceNeutralEnv    ValueSource = "neutral-env"
	SourceLegacyEnv     ValueSource = "legacy-env"
	SourceNeutralStored ValueSource = "neutral-stored"
	SourceLegacyStored  ValueSource = "legacy-stored"
	SourceDefault       ValueSource = "default"
)

var sourcePriority = map[ValueSource]int{
	SourceNeutralCLI:    0,
	SourceLegacyCLI:     1,
	SourceNeutralEnv:    2,
	SourceLegacyEnv:     3,
	SourceNeutralStored: 4,
	SourceLegacyStored:  5,
	SourceDefault:       6,
}

type ConfigCandidate struct {
	Name   string
	Value  string
	Source ValueSource
	Set    bool
}

type ResolvedConfigValue struct {
	Name   string
	Value  string
	Source ValueSource
}

func ResolveConfigValue(candidates ...ConfigCandidate) (ResolvedConfigValue, error) {
	set := make([]ConfigCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Set {
			continue
		}
		if _, ok := sourcePriority[candidate.Source]; !ok {
			return ResolvedConfigValue{}, fmt.Errorf("unknown config source %q", candidate.Source)
		}
		set = append(set, candidate)
	}
	if len(set) == 0 {
		return ResolvedConfigValue{}, fmt.Errorf("no configuration value is set")
	}
	sort.SliceStable(set, func(i, j int) bool {
		return sourcePriority[set[i].Source] < sourcePriority[set[j].Source]
	})
	winner := set[0]
	for _, candidate := range set[1:] {
		if candidate.Source == winner.Source && candidate.Value != winner.Value {
			return ResolvedConfigValue{}, fmt.Errorf("conflicting values at source %q", winner.Source)
		}
	}
	return ResolvedConfigValue{Name: winner.Name, Value: winner.Value, Source: winner.Source}, nil
}

type ConfigAlias struct {
	Neutral               string
	Legacy                []string
	SemanticCompatibility bool
}

func FrozenConfigAliases() []ConfigAlias {
	return []ConfigAlias{
		{Neutral: EnvControlURL, Legacy: []string{"MULTICA_SERVER_URL"}, SemanticCompatibility: true},
		{Neutral: EnvTaskCapacityTier, Legacy: []string{"MULTICA_DAEMON_MAX_CONCURRENT_TASKS"}, SemanticCompatibility: true},
		{Neutral: EnvGatewayRequired, Legacy: []string{"MULTICA_PRODEX_REQUIRED"}, SemanticCompatibility: false},
		{Neutral: EnvGatewayBaseURL, Legacy: []string{"MULTICA_L2_BASE_URL"}, SemanticCompatibility: false},
		{Neutral: EnvGatewaySecretFile, Legacy: []string{"MULTICA_L2_BEARER_TOKEN"}, SemanticCompatibility: false},
	}
}
