package brain

import (
	"context"
	"errors"
	"fmt"
)

type AdmissionState string

const (
	AdmissionAdmitted            AdmissionState = "admitted"
	AdmissionGatewayUnavailable  AdmissionState = "gateway-unavailable"
	AdmissionGatewayAuthFailed   AdmissionState = "gateway-authentication-failed"
	AdmissionCapabilityRejected  AdmissionState = "capability-rejected"
	AdmissionRoutePolicyRejected AdmissionState = "route-policy-rejected"
	AdmissionOverloaded          AdmissionState = "overloaded"
)

type GatewayReadinessState string

const (
	GatewayReadinessNotRequired      GatewayReadinessState = "not-required"
	GatewayReadinessReady            GatewayReadinessState = "ready"
	GatewayReadinessUnavailable      GatewayReadinessState = "unavailable"
	GatewayReadinessAuthentication   GatewayReadinessState = "authentication-failed"
	GatewayReadinessModelRegistry    GatewayReadinessState = "model-registry-unavailable"
	GatewayReadinessSelectedModel    GatewayReadinessState = "selected-model-unavailable"
	GatewayReadinessSelectedProtocol GatewayReadinessState = "selected-protocol-unavailable"
)

type AdmissionDecision struct {
	State          AdmissionState
	ReadinessState GatewayReadinessState
	TaskStatus     TaskStatus
	Retryable      bool
	ErrorClass     string
}

func (d AdmissionDecision) Admitted() bool { return d.State == AdmissionAdmitted }

type AdmissionController interface {
	Admit(context.Context, Task) (AdmissionDecision, error)
}

// GatewayAdmissionController models fail-closed admission without enabling
// the new path. The readiness checker remains an injected interface until G3.
type GatewayAdmissionController struct {
	Checker GatewayReadinessChecker
	Policy  ReadinessPolicy
}

func NewGatewayAdmissionController(checker GatewayReadinessChecker, policy ReadinessPolicy) (*GatewayAdmissionController, error) {
	if policy.Name != ReadinessStrict || !policy.FailClosed {
		return nil, fmt.Errorf("gateway admission requires strict fail-closed readiness")
	}
	return &GatewayAdmissionController{Checker: checker, Policy: policy}, nil
}

func (a *GatewayAdmissionController) Admit(ctx context.Context, task Task) (AdmissionDecision, error) {
	if err := task.Request.Validate(); err != nil {
		return AdmissionDecision{}, err
	}
	if err := task.RoutePolicy.validateFor(task.Request); err != nil {
		return AdmissionDecision{
			State:          AdmissionRoutePolicyRejected,
			ReadinessState: GatewayReadinessNotRequired,
			TaskStatus:     TaskStatusCapabilityRejected,
			ErrorClass:     "route_policy_rejected",
		}, nil
	}
	if !task.Request.GatewayRequired {
		return AdmissionDecision{State: AdmissionAdmitted, ReadinessState: GatewayReadinessNotRequired}, nil
	}
	if a == nil || a.Checker == nil {
		return unavailableDecision(), nil
	}
	snapshot, err := a.Checker.CheckGatewayReadiness(ctx, ReadinessRequest{
		RouteModel: task.Request.RouteModel,
		Protocol:   task.RoutePolicy.Protocol,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return AdmissionDecision{}, err
		}
		return unavailableDecision(), nil
	}
	if !snapshot.Live {
		return unavailableDecision(), nil
	}
	if !snapshot.Authenticated {
		return AdmissionDecision{
			State:          AdmissionGatewayAuthFailed,
			ReadinessState: GatewayReadinessAuthentication,
			TaskStatus:     TaskStatusGatewayAuthFailed,
			ErrorClass:     "gateway_authentication_failed",
		}, nil
	}
	if !snapshot.ModelRegistryReady {
		return capabilityDecision(GatewayReadinessModelRegistry, "model_registry_unavailable"), nil
	}
	if !snapshot.SelectedModelReady {
		return capabilityDecision(GatewayReadinessSelectedModel, "selected_model_unavailable"), nil
	}
	if !snapshot.SelectedProtocolReady {
		return capabilityDecision(GatewayReadinessSelectedProtocol, "selected_protocol_unavailable"), nil
	}
	if err := a.Policy.Evaluate(snapshot); err != nil {
		return capabilityDecision(GatewayReadinessUnavailable, "strict_readiness_failed"), nil
	}
	return AdmissionDecision{State: AdmissionAdmitted, ReadinessState: GatewayReadinessReady}, nil
}

func unavailableDecision() AdmissionDecision {
	return AdmissionDecision{
		State:          AdmissionGatewayUnavailable,
		ReadinessState: GatewayReadinessUnavailable,
		TaskStatus:     TaskStatusGatewayUnavailable,
		Retryable:      true,
		ErrorClass:     "gateway_unavailable",
	}
}

func capabilityDecision(state GatewayReadinessState, class string) AdmissionDecision {
	return AdmissionDecision{
		State:          AdmissionCapabilityRejected,
		ReadinessState: state,
		TaskStatus:     TaskStatusCapabilityRejected,
		ErrorClass:     class,
	}
}
