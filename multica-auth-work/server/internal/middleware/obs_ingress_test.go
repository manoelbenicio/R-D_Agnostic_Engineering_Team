package middleware

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestNewIngressSpanIsMetadataOnlyAndValid(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewIngressSpan(IngressObservation{
		RequestID: "req-abc", TaskID: "task-abc",
		Method: "POST", RouteTemplate: "/v1/tasks", PrincipalClass: "service",
		HTTPStatus: 202, Outcome: "accepted", ReasonCode: "ok", LatencyMs: 12,
	})
	if err := sp.Validate(); err != nil {
		t.Fatalf("ingress span invalid: %v", err)
	}
	if err := rec.Emit(sp); err != nil {
		t.Fatalf("emit ingress: %v", err)
	}
	if sink.Len() != 1 {
		t.Fatalf("sink len = %d, want 1", sink.Len())
	}
	got := sink.Spans()[0]
	if got.Hop != e2e.HopIngress || got.ContractVersion != e2e.ContractVersion || got.SecretsPresent {
		t.Fatalf("unexpected span shape: %+v", got)
	}
	if got.Correlation.RequestID != "req-abc" || got.Correlation.TaskID != "task-abc" {
		t.Fatalf("correlation not carried: %+v", got.Correlation)
	}
	for k := range got.Labels {
		switch k {
		case "method", "route_template", "principal_class":
		default:
			t.Fatalf("unexpected label key %q (helper must stay metadata-only)", k)
		}
	}
	for k := range got.Counters {
		if k != "latency_ms" {
			t.Fatalf("unexpected counter key %q", k)
		}
	}
	if report := e2e.ScanSpans([]e2e.Span{got}); !report.Clean {
		t.Fatalf("leak scan not clean: %+v", report.Findings)
	}
}

func TestNewIngressSpanFailsClosedOnMissingTaskID(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewIngressSpan(IngressObservation{
		RequestID: "req-abc", Method: "GET", RouteTemplate: "/v1/tasks",
		HTTPStatus: 200, Outcome: "accepted", LatencyMs: 1,
	})
	if err := rec.Emit(sp); err == nil {
		t.Fatal("ingress span missing required task_id was emitted; want fail-closed refusal")
	}
	if sink.Len() != 0 {
		t.Fatalf("sink len = %d, want 0 after fail-closed refusal", sink.Len())
	}
}

func TestEmitIngressNilRecorderIsNoop(t *testing.T) {
	if err := EmitIngress(nil, IngressObservation{RequestID: "r", TaskID: "t", Outcome: "accepted"}); err != nil {
		t.Fatalf("nil recorder must be a no-op, got %v", err)
	}
}
