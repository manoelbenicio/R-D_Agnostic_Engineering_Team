package gateway

import (
	"fmt"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type RetryPolicy struct {
	MaxAttempts      int
	EndToEndDeadline time.Duration
	PreCommitOnly    bool
	MinimumBackoff   time.Duration
	MaximumBackoff   time.Duration
	Jitter           bool
}

type ApprovedFallback struct {
	Model                brain.RouteModel
	ApprovalID           string
	CapabilityEquivalent bool
}

type FallbackPolicy struct {
	SameModelAccounts bool
	CrossModel        []ApprovedFallback
}

// FallbackCycleProof is minted only after the registry's deterministic,
// bounded fallback-graph validation succeeds. Keeping its state private makes
// a zero value fail closed at the telemetry producer boundary.
type FallbackCycleProof struct {
	accepted bool
}

func acceptedFallbackCycleProof() FallbackCycleProof {
	return FallbackCycleProof{accepted: true}
}

type CircuitScope string

const (
	CircuitAccount  CircuitScope = "account"
	CircuitModel    CircuitScope = "model"
	CircuitProvider CircuitScope = "provider-global"
	CircuitLocal    CircuitScope = "local-overload"

	// MaxCircuitObservationWindow keeps development failure history bounded;
	// fifteen minutes covers sustained throttling without carrying stale
	// failures across normal local recovery cycles.
	MaxCircuitObservationWindow = 15 * time.Minute
	// MaxCircuitOpenDuration admits the documented development quota cooldown
	// of roughly one hour while keeping all timer deadlines far below
	// time.Duration overflow and operationally observable within one cycle.
	MaxCircuitOpenDuration = time.Hour
)

type CircuitPolicy struct {
	Scopes            []CircuitScope
	FailureThreshold  int
	ObservationWindow time.Duration
	OpenDuration      time.Duration
	HalfOpenMaxProbes int
	HonorRetryAfter   bool
}

type SmartContextMode string

const (
	SmartContextOff    SmartContextMode = "off"
	SmartContextShadow SmartContextMode = "shadow"
	SmartContextCanary SmartContextMode = "canary"
)

type SmartContextPolicy struct {
	Mode                      SmartContextMode
	KillSwitch                bool
	StructuralValidation      bool
	ExactWholeRequestFallback bool
}

type RoutePolicy struct {
	ID           string
	RouterOwner  brain.RouterOwner
	Rotation     RotationMode
	Affinity     AffinityMode
	Retry        RetryPolicy
	Fallback     FallbackPolicy
	Circuit      CircuitPolicy
	SmartContext SmartContextPolicy
}

func (p RoutePolicy) Validate(primaryModel brain.RouteModel) error {
	model, err := brain.ParseRouteModel(string(primaryModel))
	if err != nil || strings.TrimSpace(p.ID) == "" || len(p.ID) > 128 || strings.ContainsAny(p.ID, "\r\n\x00") {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if p.RouterOwner != brain.RouterOwnerOmniRoute {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if p.Rotation != RotationStrictIndependentRequest && p.Rotation != RotationFailureOnly {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if p.Affinity != AffinityNone && p.Affinity != AffinityOriginAccount && p.Affinity != AffinityStatelessMaterialize {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if p.Retry.MaxAttempts < 1 || p.Retry.MaxAttempts > 10 || p.Retry.EndToEndDeadline <= 0 || p.Retry.EndToEndDeadline > 10*time.Minute || !p.Retry.PreCommitOnly {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if p.Retry.MinimumBackoff < 0 || p.Retry.MaximumBackoff < p.Retry.MinimumBackoff || p.Retry.MaximumBackoff > time.Minute {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	if !p.Fallback.SameModelAccounts {
		return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
	}
	seenFallback := make(map[brain.RouteModel]struct{}, len(p.Fallback.CrossModel))
	for _, fallback := range p.Fallback.CrossModel {
		fallbackModel, parseErr := brain.ParseRouteModel(string(fallback.Model))
		if parseErr != nil || fallbackModel == model || strings.TrimSpace(fallback.ApprovalID) == "" || !fallback.CapabilityEquivalent {
			return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
		}
		if _, exists := seenFallback[fallbackModel]; exists {
			return &GatewayError{Operation: "route_policy.validate", Class: ErrorInvalidConfiguration}
		}
		seenFallback[fallbackModel] = struct{}{}
	}
	if err := p.Circuit.Validate(); err != nil {
		return err
	}
	if err := p.SmartContext.Validate(); err != nil {
		return err
	}
	return nil
}

func (p CircuitPolicy) Validate() error {
	if len(p.Scopes) == 0 || p.FailureThreshold < 1 ||
		p.ObservationWindow <= 0 || p.ObservationWindow > MaxCircuitObservationWindow ||
		p.OpenDuration <= 0 || p.OpenDuration > MaxCircuitOpenDuration ||
		p.HalfOpenMaxProbes < 1 || !p.HonorRetryAfter {
		return &GatewayError{Operation: "circuit_policy.validate", Class: ErrorInvalidConfiguration}
	}
	seen := make(map[CircuitScope]struct{}, len(p.Scopes))
	for _, scope := range p.Scopes {
		switch scope {
		case CircuitAccount, CircuitModel, CircuitProvider, CircuitLocal:
		default:
			return &GatewayError{Operation: "circuit_policy.validate", Class: ErrorInvalidConfiguration}
		}
		if _, exists := seen[scope]; exists {
			return &GatewayError{Operation: "circuit_policy.validate", Class: ErrorInvalidConfiguration}
		}
		seen[scope] = struct{}{}
	}
	return nil
}

func (p SmartContextPolicy) Validate() error {
	switch p.Mode {
	case SmartContextOff:
		if !p.KillSwitch {
			return &GatewayError{Operation: "smart_context.validate", Class: ErrorInvalidConfiguration}
		}
	case SmartContextShadow, SmartContextCanary:
		if !p.KillSwitch || !p.StructuralValidation || !p.ExactWholeRequestFallback {
			return &GatewayError{Operation: "smart_context.validate", Class: ErrorInvalidConfiguration}
		}
	default:
		return &GatewayError{Operation: "smart_context.validate", Class: ErrorInvalidConfiguration}
	}
	return nil
}

func FrozenTier20CanaryPolicy() RoutePolicy {
	return RoutePolicy{
		ID:          "omniroute-tier20-canary-v1",
		RouterOwner: brain.RouterOwnerOmniRoute,
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Retry: RetryPolicy{
			MaxAttempts: 3, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true,
			MinimumBackoff: 100 * time.Millisecond, MaximumBackoff: 2 * time.Second, Jitter: true,
		},
		Fallback: FallbackPolicy{SameModelAccounts: true},
		Circuit: CircuitPolicy{
			Scopes:           []CircuitScope{CircuitAccount, CircuitModel, CircuitProvider, CircuitLocal},
			FailureThreshold: 3, ObservationWindow: time.Minute, OpenDuration: time.Minute,
			HalfOpenMaxProbes: 1, HonorRetryAfter: true,
		},
		SmartContext: SmartContextPolicy{Mode: SmartContextOff, KillSwitch: true},
	}
}

func (p RoutePolicy) String() string {
	return fmt.Sprintf("gateway.RoutePolicy{id:%q, owner:%q, rotation:%q, cross_model_fallbacks:%d}", p.ID, p.RouterOwner, p.Rotation, len(p.Fallback.CrossModel))
}
