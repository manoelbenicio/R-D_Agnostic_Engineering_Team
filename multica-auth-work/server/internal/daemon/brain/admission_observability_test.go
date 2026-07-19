package brain

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestAdmissionObserverEmitsApprovedMetadata(t *testing.T) {
	sink := e2e.NewMemorySink()
	observer := NewAdmissionObserver(sink)
	correlation := AdmissionCorrelation("task-001", "session-001")
	correlation.RequestID = "request-001"

	err := observer.Emit(AdmissionObservation{
		Correlation: correlation,
		CLIKind:     CLICodex,
		RouteModel:  RouteModel("agy/claude-opus-4-6-thinking"),
		Decision:    AdmissionAdmitted,
		Readiness:   GatewayReadinessReady,
		StartedAt:   time.Now().Add(-time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if sink.Len() != 1 {
		t.Fatalf("recorded spans = %d, want 1", sink.Len())
	}
	span := sink.Spans()[0]
	if span.Hop != e2e.HopAdmission || span.Correlation.TaskID != correlation.TaskID ||
		span.Correlation.SessionID != correlation.SessionID || span.Correlation.LaunchID == "" {
		t.Fatalf("unexpected correlation: %+v", span.Correlation)
	}
	if span.Labels["admission_decision"] != string(AdmissionAdmitted) ||
		span.Labels["readiness_result"] != string(GatewayReadinessReady) ||
		span.Labels["cli_kind"] != string(CLICodex) ||
		span.Labels["route_model"] != "agy/claude-opus-4-6-thinking" ||
		span.Labels["fail_closed_class"] != "none" {
		t.Fatalf("unexpected labels: %+v", span.Labels)
	}
	if span.Outcome != "admitted" || span.ReasonCode != "" || span.SecretsPresent {
		t.Fatalf("unexpected outcome: %+v", span)
	}
	if report := e2e.ScanFromSink(sink); !report.Clean || len(report.Findings) != 0 {
		t.Fatalf("OBS-4 span failed structural leak scan: %+v", report)
	}
}

func TestAdmissionObserverEmitsFailClosedClassification(t *testing.T) {
	sink := e2e.NewMemorySink()
	observer := NewAdmissionObserver(sink)
	decision, readiness, class := AdmissionClassification("credential_source_unavailable")
	err := observer.Emit(AdmissionObservation{
		Correlation:     AdmissionCorrelation("task-002", ""),
		CLIKind:         CLIClaudeCode,
		RouteModel:      RouteModel("anthropic/claude-opus"),
		Decision:        decision,
		Readiness:       readiness,
		FailClosedClass: class,
		StartedAt:       time.Now(),
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	span := sink.Spans()[0]
	if span.Outcome != "rejected" || span.ReasonCode != class ||
		span.Labels["admission_decision"] != string(AdmissionGatewayAuthFailed) ||
		span.Labels["readiness_result"] != string(GatewayReadinessAuthentication) ||
		span.Labels["fail_closed_class"] != class {
		t.Fatalf("unexpected rejected span: %+v", span)
	}
}

func TestAdmissionObserverFailsClosedWithoutRecordingInvalidMetadata(t *testing.T) {
	sink := e2e.NewMemorySink()
	observer := NewAdmissionObserver(sink)
	err := observer.Emit(AdmissionObservation{
		Correlation: AdmissionCorrelation("task-003", "session-003"),
		CLIKind:     CLICodex,
		RouteModel:  RouteModel("invalid route model"),
		Decision:    AdmissionAdmitted,
		Readiness:   GatewayReadinessReady,
		StartedAt:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected invalid metadata to be refused")
	}
	if sink.Len() != 0 {
		t.Fatalf("recorded spans = %d, want 0", sink.Len())
	}
}

func TestAdmissionClassificationUnknownFailsClosed(t *testing.T) {
	decision, readiness, class := AdmissionClassification("unexpected_internal_failure")
	if decision != AdmissionGatewayUnavailable || readiness != GatewayReadinessUnavailable || class != "admission_fail_closed" {
		t.Fatalf("classification = (%q, %q, %q)", decision, readiness, class)
	}
}

func TestAdmissionClassificationPreservesAdmissionTaxonomy(t *testing.T) {
	tests := []struct {
		class     string
		decision  AdmissionState
		readiness GatewayReadinessState
	}{
		{"local_capacity_overloaded", AdmissionOverloaded, GatewayReadinessNotRequired},
		{"selected_protocol_unavailable", AdmissionCapabilityRejected, GatewayReadinessSelectedProtocol},
		{"selected_model_unavailable", AdmissionCapabilityRejected, GatewayReadinessSelectedModel},
		{"model_registry_unavailable", AdmissionCapabilityRejected, GatewayReadinessModelRegistry},
		{"route_policy_rejected", AdmissionRoutePolicyRejected, GatewayReadinessNotRequired},
		{"gateway_authentication_failed", AdmissionGatewayAuthFailed, GatewayReadinessAuthentication},
		{"gateway_unavailable", AdmissionGatewayUnavailable, GatewayReadinessUnavailable},
	}
	for _, test := range tests {
		t.Run(test.class, func(t *testing.T) {
			decision, readiness, class := AdmissionClassification(test.class)
			if decision != test.decision || readiness != test.readiness || class != test.class {
				t.Fatalf("classification = (%q, %q, %q)", decision, readiness, class)
			}
		})
	}
}

func TestAdmissionCorrelationMatchesExistingOpaqueIDs(t *testing.T) {
	got := AdmissionCorrelation("task-004", "session-004")
	if got.TaskID != admissionSafeID("task", "task-004") || got.SessionID != admissionSafeID("session", "session-004") {
		t.Fatalf("unexpected correlation: %+v", got)
	}
	launchID := AdmissionLaunchID(got)
	if launchID != AdmissionLaunchID(got) {
		t.Fatal("launch ID is not deterministic")
	}
	got.RequestID = "request-next"
	if launchID == AdmissionLaunchID(got) {
		t.Fatal("distinct admission requests share a launch ID")
	}
}
