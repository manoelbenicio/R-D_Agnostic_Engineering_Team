package deploy

import (
	"fmt"
	"net"
	"net/url"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type ProtocolPath struct {
	Name     string
	BaseRule string
	Path     string
}

type ReachabilityPrerequisite struct {
	ID          string
	Requirement string
}

type ProcedurePhase string

const (
	PhaseValidate      ProcedurePhase = "validate"
	PhaseStart         ProcedurePhase = "start"
	PhaseWaitReadiness ProcedurePhase = "wait-readiness"
	PhaseLaunchBrain   ProcedurePhase = "launch-agent-brain"
	PhaseRecreate      ProcedurePhase = "recreate"
	PhaseRollback      ProcedurePhase = "rollback"
)

type OperationalStep struct {
	Phase       ProcedurePhase
	Instruction string
	Evidence    string
}

// EndpointPlan describes topology-specific reachability. It performs no
// network access and does not start or recreate any service.
type EndpointPlan struct {
	EvidenceID    string
	Topology      brain.RuntimeTopology
	BaseURL       string
	AuthorizedNow bool
	TrustBoundary string
	Protocols     []ProtocolPath
	Prerequisites []ReachabilityPrerequisite
	Procedure     []OperationalStep
}

func HostWSLEndpointPlan() EndpointPlan {
	return EndpointPlan{
		EvidenceID:    "EV-G2D-02",
		Topology:      brain.TopologyHostWSL,
		BaseURL:       brain.DefaultHostGatewayURL,
		AuthorizedNow: true,
		TrustBoundary: "host loopback only; plain HTTP is not accepted beyond this local boundary",
		Protocols:     frozenProtocolPaths(),
		Prerequisites: []ReachabilityPrerequisite{
			{"HOST-01", "OmniRoute is pinned to an accepted image digest and publishes port 20128 to host loopback"},
			{"HOST-02", "the restricted secret reference passes metadata validation without exposing content"},
			{"HOST-03", "liveness and authenticated readiness are distinct and readiness includes the selected model/protocol"},
			{"HOST-04", "the host daemon uses the frozen host gateway URL and never resolves Docker service DNS"},
		},
		Procedure: defaultServiceProcedure("the approved host container/compose service", "the previous accepted image/config revision"),
	}
}

func ContainerEndpointPlan() EndpointPlan {
	return EndpointPlan{
		EvidenceID:    "EV-G2D-02",
		Topology:      brain.TopologyContainer,
		BaseURL:       brain.DefaultContainerGatewayURL,
		AuthorizedNow: false,
		TrustBoundary: "private user-defined container network; container loopback must not target another container",
		Protocols:     frozenProtocolPaths(),
		Prerequisites: []ReachabilityPrerequisite{
			{"CTR-01", "Agent Brain and OmniRoute share an explicitly configured private network"},
			{"CTR-02", "the OmniRoute service name is resolvable only for the container topology"},
			{"CTR-03", "the restricted secret is supplied by an approved runtime secret mount or credential facility"},
			{"CTR-04", "container health ordering waits for authenticated readiness, not process liveness alone"},
		},
		Procedure: defaultServiceProcedure("the approved container stack", "the previous accepted stack/image/config revision"),
	}
}

func frozenProtocolPaths() []ProtocolPath {
	return []ProtocolPath{
		{"anthropic-messages", "Claude receives the gateway root without a trailing /v1", "/v1/messages"},
		{"openai-responses", "Codex receives the gateway /v1 base over HTTP/SSE", "/v1/responses"},
		{"openai-chat", "approved compatible adapters receive the gateway /v1 base", "/v1/chat/completions"},
		{"antigravity-direct", "enabled only after exact native endpoint conformance", "/v1/antigravity"},
	}
}

func defaultServiceProcedure(service, rollbackTarget string) []OperationalStep {
	return []OperationalStep{
		{PhaseValidate, "verify pinned image/config revision, state backup checkpoint, endpoint topology, secret-reference metadata, and rollback target", "redacted preflight record"},
		{PhaseStart, "start " + service + " without placing credential values on a command line or in committed configuration", "service generation and image digest"},
		{PhaseWaitReadiness, "wait for liveness, then authenticated readiness for the selected protocol/model; hold admissions while not ready", "readiness transition and safe reason code"},
		{PhaseLaunchBrain, "start Agent Brain with frozen gateway-required, base-URL, secret-file-reference, strict-readiness, and tier-20 settings", "effective non-secret configuration revision"},
		{PhaseRecreate, "drain or hold new admissions, checkpoint state, recreate one component at a time, then repeat authenticated readiness", "drain, restart, and recovery timestamps"},
		{PhaseRollback, "select " + rollbackTarget + "; keep provider-native and Prodex fallback disabled; drain or reject until readiness recovers", "rollback trigger, revision, duration, and outcome"},
	}
}

func (p EndpointPlan) Validate() error {
	if p.EvidenceID != "EV-G2D-02" {
		return fmt.Errorf("unexpected endpoint evidence id")
	}
	want, err := brain.DefaultGatewayURL(p.Topology)
	if err != nil {
		return err
	}
	if p.BaseURL != want {
		return fmt.Errorf("endpoint does not match frozen topology default")
	}
	parsed, err := url.Parse(p.BaseURL)
	if err != nil {
		return fmt.Errorf("parse endpoint: %w", err)
	}
	if parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("local endpoint must be a credential-free HTTP origin")
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil || port != "20128" {
		return fmt.Errorf("endpoint must use the declared port")
	}
	switch p.Topology {
	case brain.TopologyHostWSL:
		if host != "127.0.0.1" || !p.AuthorizedNow {
			return fmt.Errorf("host topology must use authorized loopback")
		}
	case brain.TopologyContainer:
		if host != "omniroute" || p.AuthorizedNow {
			return fmt.Errorf("container topology must remain future and use its service name")
		}
	default:
		return fmt.Errorf("unsupported topology")
	}
	if len(p.Protocols) != 4 || len(p.Prerequisites) == 0 {
		return fmt.Errorf("endpoint protocol and prerequisite specifications are required")
	}
	requiredPhases := map[ProcedurePhase]bool{
		PhaseValidate:      false,
		PhaseStart:         false,
		PhaseWaitReadiness: false,
		PhaseLaunchBrain:   false,
		PhaseRecreate:      false,
		PhaseRollback:      false,
	}
	for _, step := range p.Procedure {
		if _, ok := requiredPhases[step.Phase]; !ok || step.Instruction == "" || step.Evidence == "" {
			return fmt.Errorf("invalid service procedure step")
		}
		requiredPhases[step.Phase] = true
	}
	for phase, present := range requiredPhases {
		if !present {
			return fmt.Errorf("missing service procedure phase %q", phase)
		}
	}
	return nil
}
