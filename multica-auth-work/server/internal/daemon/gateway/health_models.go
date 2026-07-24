package gateway

import (
	"context"
	"log/slog"
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
	diag        *slog.Logger
}

// SetDiagnosticsLogger enables safe per-predicate strict-readiness diagnostics
// (transport status/error class, registry version, and each predicate outcome).
// It never logs response bodies, secrets, URLs, or credentials.
func (c *ReadinessChecker) SetDiagnosticsLogger(logger *slog.Logger) {
	if c != nil {
		c.diag = logger
	}
}

var _ brain.GatewayReadinessChecker = (*ReadinessChecker)(nil)

func NewReadinessChecker(client *Client, registry *Registry, policy brain.ReadinessPolicy, correlation CorrelationSource) (*ReadinessChecker, error) {
	if client == nil || registry == nil || correlation == nil || policy.Name != brain.ReadinessStrict || !policy.FailClosed {
		return nil, &GatewayError{Operation: "readiness_checker", Class: ErrorInvalidConfiguration}
	}
	return &ReadinessChecker{client: client, registry: registry, policy: policy, correlation: correlation}, nil
}

func (c *ReadinessChecker) CheckGatewayReadiness(ctx context.Context, request brain.ReadinessRequest) (snapshot brain.ReadinessSnapshot, err error) {
	var registryVersion string
	defer func() { c.emitPredicates(request, snapshot, registryVersion, err) }()
	if _, perr := brain.ParseRouteModel(string(request.RouteModel)); perr != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorInvalidRequest}
	}
	if _, perr := protocolFromWire(string(request.Protocol)); perr != nil {
		return snapshot, perr
	}
	correlation, cerr := c.correlation()
	if cerr != nil || correlation.Validate() != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorInvalidRequest}
	}
	if _, lerr := c.client.CheckLiveness(ctx, correlation); lerr != nil {
		return snapshot, lerr
	}
	snapshot.Live = true
	if _, rerr := c.client.CheckReadiness(ctx, correlation); rerr != nil {
		return snapshot, rerr
	}
	snapshot.Authenticated = true
	registrySnapshot, serr := c.registry.Snapshot(ctx)
	if serr != nil {
		return snapshot, serr
	}
	registryVersion = registrySnapshot.Version
	snapshot.ModelRegistryReady = registrySnapshot.Version != ""
	model, ok := registrySnapshot.Models[request.RouteModel]
	if ok && model.Available {
		snapshot.SelectedModelReady = true
		snapshot.SelectedProtocolReady = model.Capability.Protocol == request.Protocol
	}
	if perr := c.policy.Evaluate(snapshot); perr != nil {
		return snapshot, &GatewayError{Operation: operationReadiness, Class: ErrorCapability}
	}
	return snapshot, nil
}

// emitPredicates logs a single safe diagnostic line describing every strict
// readiness predicate outcome and, on failure, the failing sub-check operation,
// error class, and transport status. No response bodies, URLs, secrets, or
// credentials are ever logged.
func (c *ReadinessChecker) emitPredicates(request brain.ReadinessRequest, snapshot brain.ReadinessSnapshot, registryVersion string, err error) {
	if c == nil || c.diag == nil {
		return
	}
	failOperation, failClass, failStatus := "", "", 0
	failDetail := ""
	if err != nil {
		if ge, okErr := err.(*GatewayError); okErr && ge != nil {
			failOperation, failClass, failStatus, failDetail = ge.Operation, string(ge.Class), ge.StatusCode, ge.Detail
		} else {
			failClass = "non_gateway_error"
		}
	}
	c.diag.Info("strict_readiness_predicate",
		"route_model", string(request.RouteModel),
		"protocol", string(request.Protocol),
		"live", snapshot.Live,
		"authenticated", snapshot.Authenticated,
		"model_registry_ready", snapshot.ModelRegistryReady,
		"selected_model_ready", snapshot.SelectedModelReady,
		"selected_protocol_ready", snapshot.SelectedProtocolReady,
		"registry_version_present", registryVersion != "",
		"ok", err == nil,
		"fail_operation", failOperation,
		"fail_error_class", failClass,
		"fail_status_code", failStatus,
		"fail_detail", failDetail,
	)
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
