package brain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// AdmissionObservation is the metadata-only OBS-4 input. It deliberately has
// no fields for task content, errors, credentials, argv, or repository data.
type AdmissionObservation struct {
	Correlation     Correlation
	CLIKind         CLIKind
	RouteModel      RouteModel
	Decision        AdmissionState
	Readiness       GatewayReadinessState
	FailClosedClass string
	StartedAt       time.Time
}

// AdmissionObserver emits the daemon admission/lifecycle hop through the
// approved W5 recorder. A nil sink is valid and discards spans after the same
// validation and structural leak checks used by a configured sink.
type AdmissionObserver struct {
	recorder *e2e.Recorder
}

// NewAdmissionObserver constructs the OBS-4 emitter over an approved W5 sink.
func NewAdmissionObserver(sink e2e.Sink) *AdmissionObserver {
	return &AdmissionObserver{recorder: e2e.NewRecorder(sink)}
}

// NewAdmissionLogSink returns a metadata-only sink for the daemon logger.
// A nil logger selects the W5 recorder's validated discard path.
func NewAdmissionLogSink(logger *slog.Logger) e2e.Sink {
	if logger == nil {
		return nil
	}
	return admissionLogSink{logger: logger}
}

// Emit records one finished OBS-4 span. Admission latency is measured from the
// caller-provided start so validation and logging overhead are excluded.
func (o *AdmissionObserver) Emit(observation AdmissionObservation) error {
	if o == nil || o.recorder == nil {
		return fmt.Errorf("admission observer unavailable")
	}
	classification := observation.FailClosedClass
	if classification == "" {
		classification = "none"
	}
	startedAt := observation.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	correlation := e2e.Correlation{
		TaskID:    observation.Correlation.TaskID,
		SessionID: observation.Correlation.SessionID,
		LaunchID:  AdmissionLaunchID(observation.Correlation),
	}
	span := e2e.NewSpan(e2e.HopAdmission, correlation)
	span.StartedAt = startedAt
	span.WithLabel("admission_decision", string(observation.Decision)).
		WithLabel("readiness_result", string(observation.Readiness)).
		WithLabel("cli_kind", string(observation.CLIKind)).
		WithLabel("route_model", string(observation.RouteModel)).
		WithLabel("fail_closed_class", classification)
	if observation.Decision == AdmissionAdmitted {
		span.WithOutcome("admitted", "")
	} else {
		span.WithOutcome("rejected", classification)
	}
	span.Finish().WithCounter("latency_ms", span.DurationMs())
	return o.recorder.Emit(span)
}

// AdmissionCorrelation derives the same opaque task/session identifiers used
// by the existing Agent Brain correlation path without exposing raw IDs.
func AdmissionCorrelation(taskID, sessionSeed string) Correlation {
	if sessionSeed == "" {
		sessionSeed = taskID
	}
	return Correlation{
		TaskID:    admissionSafeID("task", taskID),
		SessionID: admissionSafeID("session", sessionSeed),
	}
}

// AdmissionLaunchID is deterministic for an admission attempt's correlation,
// allowing the CLI hop to reproduce the join key without mutable global state.
func AdmissionLaunchID(correlation Correlation) string {
	return admissionSafeID("launch", correlation.TaskID+":"+correlation.SessionID+":"+correlation.RequestID)
}

// AdmissionClassification maps closed daemon failure classes to the public,
// bounded admission/readiness taxonomy. Unknown classes fail closed.
func AdmissionClassification(class string) (AdmissionState, GatewayReadinessState, string) {
	switch class {
	case "", "admitted":
		return AdmissionAdmitted, GatewayReadinessReady, "none"
	case "overloaded", "capacity_admission_closed", "capacity_gate_unavailable", "local_capacity_overloaded":
		return AdmissionOverloaded, GatewayReadinessNotRequired, class
	case "adapter_fail_closed", "trusted_profile_unavailable", "selected_protocol_unavailable":
		return AdmissionCapabilityRejected, GatewayReadinessSelectedProtocol, class
	case "builtin_runtime_mapping_unavailable", "builtin_runtime_unavailable":
		return AdmissionCapabilityRejected, GatewayReadinessSelectedProtocol, class
	case "route_model_not_approved", "capability_rejected", "capability_unavailable", "selected_model_unavailable":
		return AdmissionCapabilityRejected, GatewayReadinessSelectedModel, class
	case "model_registry_invalid", "model_registry_unavailable":
		return AdmissionCapabilityRejected, GatewayReadinessModelRegistry, class
	case "legacy_contract_rejected", "route_policy_rejected", "custom_runtime_not_allowed",
		"custom_args_not_allowed", "builtin_runtime_provider_mismatch":
		return AdmissionRoutePolicyRejected, GatewayReadinessNotRequired, class
	case "credential_source_unavailable", "gateway_authentication_failed":
		return AdmissionGatewayAuthFailed, GatewayReadinessAuthentication, class
	case "gateway_client_invalid", "readiness_checker_invalid", "admission_controller_invalid",
		"readiness_cancelled", "integration_initialization_failed", "gateway_unavailable", "strict_readiness_failed":
		return AdmissionGatewayUnavailable, GatewayReadinessUnavailable, class
	default:
		return AdmissionGatewayUnavailable, GatewayReadinessUnavailable, "admission_fail_closed"
	}
}

func admissionSafeID(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return prefix + "-" + hex.EncodeToString(digest[:8])
}

type admissionLogSink struct {
	logger *slog.Logger
}

func (s admissionLogSink) Record(span e2e.Span) error {
	s.logger.Info("agent brain admission span",
		"contract_version", span.ContractVersion,
		"hop", span.Hop,
		"task_id", span.Correlation.TaskID,
		"session_id", span.Correlation.SessionID,
		"launch_id", span.Correlation.LaunchID,
		"admission_decision", span.Labels["admission_decision"],
		"readiness_result", span.Labels["readiness_result"],
		"cli_kind", span.Labels["cli_kind"],
		"route_model", span.Labels["route_model"],
		"fail_closed_class", span.Labels["fail_closed_class"],
		"outcome", span.Outcome,
		"reason_code", span.ReasonCode,
		"latency_ms", span.Counters["latency_ms"],
		"secrets_present", span.SecretsPresent,
	)
	return nil
}
