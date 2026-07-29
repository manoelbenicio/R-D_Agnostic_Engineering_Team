package gateway

import (
	"context"
	"sort"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const operationRouteModelProjection = "registry.project_route_models"

// ErrorDeadlineExceeded names the established timeout classification at the
// route-projection boundary. It is an alias so projection callers can state
// deadline intent without changing the gateway-wide ErrorTimeout contract.
const ErrorDeadlineExceeded ErrorClass = ErrorTimeout

// RouteDisplayNamespace is a presentation grouping only. It is not a model
// vendor, provider, router owner, credential owner, or account-pool identity.
type RouteDisplayNamespace string

const (
	RouteDisplayClaudeCode RouteDisplayNamespace = "frontend/claude-code"
	RouteDisplayCodex      RouteDisplayNamespace = "frontend/codex"
)

// ProjectedRouteModel is the content-free gateway route shape exposed to a
// model-list bridge. ID remains the exact OmniRoute registry identifier.
// Routing, fallback, provider, credential, and account ownership are omitted.
type ProjectedRouteModel struct {
	ID               brain.RouteModel      `json:"id"`
	DisplayNamespace RouteDisplayNamespace `json:"display_namespace"`
	CLIKind          brain.CLIKind         `json:"cli_kind"`
	Protocol         brain.ProtocolFamily  `json:"protocol"`
	TrustedProfile   ProfileID             `json:"trusted_profile"`
}

type RouteModelProjection struct {
	RegistryVersion string                `json:"registry_version"`
	Models          []ProjectedRouteModel `json:"models"`
}

// CredentiallessAdapterFilter is supplied by the composition owner that
// already holds the authoritative credentialless adapter decision. Gateway
// does not import or duplicate runtime-environment adapter policy.
type CredentiallessAdapterFilter func(cli brain.CLIKind, protocol brain.ProtocolFamily) bool

// ProjectRouteModels fetches one required registry snapshot and projects it.
// It has no native-catalog fallback callback: every registry or projection
// failure returns a zero result and fails closed.
func (r *Registry) ProjectRouteModels(
	ctx context.Context,
	cli brain.CLIKind,
	accepted CredentiallessAdapterFilter,
) (RouteModelProjection, error) {
	if r == nil {
		return RouteModelProjection{}, routeModelProjectionError(ErrorInvalidConfiguration)
	}
	if ctx.Err() != nil {
		return RouteModelProjection{}, classifyContextError(operationRouteModelProjection, ctx)
	}
	if _, err := routeDisplayNamespace(cli, accepted); err != nil {
		return RouteModelProjection{}, err
	}
	snapshot, err := r.Snapshot(ctx)
	if ctx.Err() != nil {
		return RouteModelProjection{}, classifyContextError(operationRouteModelProjection, ctx)
	}
	if err != nil {
		return RouteModelProjection{}, err
	}
	return ProjectSnapshotRouteModels(snapshot, cli, accepted)
}

// ProjectSnapshotRouteModels is the pure projection used after a successful
// Registry.Snapshot. Output ordering is the bytewise order of exact route IDs.
func ProjectSnapshotRouteModels(
	snapshot RegistrySnapshot,
	cli brain.CLIKind,
	accepted CredentiallessAdapterFilter,
) (RouteModelProjection, error) {
	namespace, err := routeDisplayNamespace(cli, accepted)
	if err != nil {
		return RouteModelProjection{}, err
	}
	if !validProjectionRegistryVersion(snapshot.Version) || len(snapshot.Models) == 0 || len(snapshot.Models) > MaxRegistryModels {
		return RouteModelProjection{}, routeModelProjectionError(ErrorProtocol)
	}

	models := make([]brain.RouteModel, 0, len(snapshot.Models))
	for model, spec := range snapshot.Models {
		if err := validateProjectionModelSpec(model, spec); err != nil {
			return RouteModelProjection{}, err
		}
		models = append(models, model)
	}
	sort.Slice(models, func(left, right int) bool {
		return string(models[left]) < string(models[right])
	})

	type protocolDecision struct {
		profile  RuntimeProfile
		accepted bool
	}
	decisions := make(map[brain.ProtocolFamily]protocolDecision)
	projected := make([]ProjectedRouteModel, 0, len(models))
	for _, model := range models {
		spec := snapshot.Models[model]
		if !spec.Available {
			continue
		}
		decision, decided := decisions[spec.Capability.Protocol]
		if !decided {
			profile, profileErr := LookupRuntimeProfile(spec.Capability.Protocol, cli)
			if profileErr != nil {
				decisions[spec.Capability.Protocol] = decision
				continue
			}
			if profileErr = profile.Validate(); profileErr != nil {
				return RouteModelProjection{}, profileErr
			}
			decision = protocolDecision{
				profile:  profile,
				accepted: accepted(cli, spec.Capability.Protocol),
			}
			decisions[spec.Capability.Protocol] = decision
		}
		if !decision.accepted {
			continue
		}
		projected = append(projected, ProjectedRouteModel{
			ID:               model,
			DisplayNamespace: namespace,
			CLIKind:          cli,
			Protocol:         spec.Capability.Protocol,
			TrustedProfile:   decision.profile.ID,
		})
	}
	if len(projected) == 0 {
		return RouteModelProjection{}, routeModelProjectionError(ErrorCapability)
	}
	return RouteModelProjection{
		RegistryVersion: snapshot.Version,
		Models:          projected,
	}, nil
}

func routeDisplayNamespace(cli brain.CLIKind, accepted CredentiallessAdapterFilter) (RouteDisplayNamespace, error) {
	if accepted == nil {
		return "", routeModelProjectionError(ErrorInvalidConfiguration)
	}
	parsed, err := brain.ParseCLIKind(string(cli))
	if err != nil || parsed != cli {
		return "", routeModelProjectionError(ErrorInvalidConfiguration)
	}
	switch cli {
	case brain.CLIClaudeCode:
		return RouteDisplayClaudeCode, nil
	case brain.CLICodex:
		return RouteDisplayCodex, nil
	default:
		return "", routeModelProjectionError(ErrorCapability)
	}
}

func validProjectionRegistryVersion(version string) bool {
	return version != "" && version == strings.TrimSpace(version) && len(version) <= 128 && !strings.ContainsAny(version, "\r\n\x00")
}

func validateProjectionModelSpec(model brain.RouteModel, spec ModelSpec) error {
	parsedModel, err := brain.ParseRouteModel(string(model))
	if err != nil || parsedModel != model || spec.Capability.RouteModel != model || spec.Capability.ContextLimit <= 0 {
		return routeModelProjectionError(ErrorProtocol)
	}
	parsedProtocol, err := protocolFromWire(string(spec.Capability.Protocol))
	if err != nil || parsedProtocol != spec.Capability.Protocol {
		return routeModelProjectionError(ErrorProtocol)
	}
	accountPool := strings.TrimSpace(spec.AccountPool)
	if accountPool == "" || accountPool != spec.AccountPool || len(accountPool) > 128 || strings.ContainsAny(accountPool, "\r\n\x00") {
		return routeModelProjectionError(ErrorProtocol)
	}
	if _, err := parseRotation(string(spec.Rotation)); err != nil {
		return routeModelProjectionError(ErrorProtocol)
	}
	if _, err := parseAffinity(string(spec.Affinity)); err != nil {
		return routeModelProjectionError(ErrorProtocol)
	}
	return nil
}

func routeModelProjectionError(class ErrorClass) error {
	return &GatewayError{Operation: operationRouteModelProjection, Class: class}
}
