package gateway

import (
	"context"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const HeaderRegistryVersion = "X-OmniRoute-Registry-Version"

type ProbeResult struct {
	StatusCode int
	RequestID  string
}

type ModelsDocument struct {
	Object          string          `json:"object"`
	RegistryVersion string          `json:"registry_version"`
	Models          []ModelDocument `json:"data"`
}

type ModelDocument struct {
	ID               string            `json:"id"`
	Protocol         string            `json:"protocol"`
	Streaming        *bool             `json:"streaming"`
	Tools            *bool             `json:"tools"`
	Reasoning        *bool             `json:"reasoning"`
	StructuredOutput *bool             `json:"structured_output"`
	ContextLimit     int               `json:"context_limit"`
	AccountPool      string            `json:"account_pool"`
	Rotation         string            `json:"rotation"`
	Affinity         string            `json:"affinity"`
	Fallback         []string          `json:"fallback"`
	Available        *bool             `json:"available"`
	Metadata         ModelDocumentMeta `json:"metadata"`
}

type ModelDocumentMeta struct {
	RegistryVersion string `json:"registry_version,omitempty"`
}

type ModelsFetchFunc func(context.Context) (ModelsDocument, error)

func (f ModelsFetchFunc) FetchModels(ctx context.Context) (ModelsDocument, error) {
	return f(ctx)
}

type ModelsFetcher interface {
	FetchModels(context.Context) (ModelsDocument, error)
}

var _ brain.ModelCapabilityRegistry = (*Registry)(nil)

type CorrelationSource func() (brain.Correlation, error)

type ReadinessChecker struct {
	client      *Client
	registry    *Registry
	policy      brain.ReadinessPolicy
	correlation CorrelationSource
}

var _ brain.GatewayReadinessChecker = (*ReadinessChecker)(nil)

func NewReadinessChecker(client *Client, registry *Registry, policy brain.ReadinessPolicy, correlation CorrelationSource) (*ReadinessChecker, error) {
	if client == nil || registry == nil || correlation == nil || policy.Name != brain.ReadinessStrict || !policy.FailClosed {
		return nil, &GatewayError{Operation: "readiness_checker", Class: ErrorInvalidConfiguration}
	}
	return &ReadinessChecker{client: client, registry: registry, policy: policy, correlation: correlation}, nil
}

func (c *ReadinessChecker) CheckGatewayReadiness(ctx context.Context, request brain.ReadinessRequest) (brain.ReadinessSnapshot, error) {
	var snapshot brain.ReadinessSnapshot
	if _, err := brain.ParseRouteModel(string(request.RouteModel)); err != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorInvalidRequest}
	}
	if _, err := protocolFromWire(string(request.Protocol)); err != nil {
		return snapshot, err
	}
	correlation, err := c.correlation()
	if err != nil || correlation.Validate() != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorInvalidRequest}
	}
	if _, err := c.client.CheckLiveness(ctx, correlation); err != nil {
		return snapshot, err
	}
	snapshot.Live = true
	if _, err := c.client.CheckReadiness(ctx, correlation); err != nil {
		return snapshot, err
	}
	snapshot.Authenticated = true
	registrySnapshot, err := c.registry.Snapshot(ctx)
	if err != nil {
		return snapshot, err
	}
	snapshot.ModelRegistryReady = registrySnapshot.Version != ""
	model, ok := registrySnapshot.Models[request.RouteModel]
	if ok && model.Available {
		snapshot.SelectedModelReady = true
		snapshot.SelectedProtocolReady = model.Capability.Protocol == request.Protocol
	}
	if err := c.policy.Evaluate(snapshot); err != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorCapability}
	}
	return snapshot, nil
}

func protocolFromWire(value string) (brain.ProtocolFamily, error) {
	protocol := brain.ProtocolFamily(strings.TrimSpace(value))
	switch protocol {
	case brain.ProtocolAnthropicMessages, brain.ProtocolOpenAIResponses, brain.ProtocolOpenAIChat, brain.ProtocolAntigravity:
		return protocol, nil
	default:
		return "", &GatewayError{Operation: operationModels, Class: ErrorProtocol}
	}
}
