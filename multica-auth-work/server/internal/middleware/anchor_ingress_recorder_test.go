package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// TestRequestLoggerAnchorEmitsIngressSpanWhenWired drives the REAL production
// ingress call site — the RequestLogger middleware handler — end-to-end and
// proves it emits a hop-1 span through the installed recorder. This is the
// anchor/seam guard: it FAILS if the handler stops invoking the ingress emitter
// or the seam is wired to a nil/discard recorder (the MemorySink then stays
// empty). It complements TestEmitIngressSpanRecordsMetadataOnlyWithBothIDs,
// which exercises the emitIngressSpan helper directly rather than the handler.
func TestRequestLoggerAnchorEmitsIngressSpanWhenWired(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	// The real handler chain: the inner handler stamps the task_id (as the
	// production auth/routing layers do) and returns 202; RequestLogger reads
	// request_id + task_id AFTER ServeHTTP and emits the ingress span.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetIngressTaskID(r.Context(), "task-anchor-1")
		w.WriteHeader(http.StatusAccepted)
	})
	handler := RequestLogger(inner)

	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	// Seed the chi request id the RequestLogger reads via chimw.GetReqID.
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-anchor-1"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("ingress anchor not wired: got %d spans, want 1 (handler must emit hop-1 through the installed recorder)", len(spans))
	}
	got := spans[0]
	if got.Hop != e2e.HopIngress {
		t.Fatalf("span hop = %q, want %q", got.Hop, e2e.HopIngress)
	}
	if got.Correlation.RequestID != e2e.CanonicalRequestID("task-anchor-1") || got.Correlation.TaskID != "task-anchor-1" {
		t.Fatalf("ingress span missing canonical correlation IDs: %+v", got.Correlation)
	}
	if got.SecretsPresent {
		t.Fatalf("ingress span must be metadata-only (secrets_present=false)")
	}
}

// TestRequestLoggerAnchorFailsClosedWithoutTaskID proves the anchor is genuinely
// gated (not blindly emitting): with the recorder wired but no task_id stamped,
// the required-ID contract fails closed and no span is recorded.
func TestRequestLoggerAnchorFailsClosedWithoutTaskID(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted) // no SetIngressTaskID → missing required task_id
	})
	handler := RequestLogger(inner)

	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-anchor-2"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if n := sink.Len(); n != 0 {
		t.Fatalf("expected 0 spans when task_id is absent (fail-closed), got %d", n)
	}
}

// TestRequestLoggerAnchorEmitsExactlyOneIngressSpanPerAcceptedRequest is the
// ingress CARDINALITY guard at the REAL RequestLogger call site: one accepted
// control request that stamps the task_id must yield EXACTLY ONE HopIngress span,
// and e2e.Assemble must see it once with no duplicate/orphan for the ingress hop.
//
// NOTE (production gap, see T20-live-correlation-audit.md M1): no production
// accepted-control endpoint calls SetIngressTaskID, so in production the ingress
// cardinality is currently ZERO (the span never emits). Driving the real
// task-creating handler end-to-end (which SHOULD stamp the id) requires the
// handler+DB harness owned by the handler lane; this test guards the middleware
// call site with the id stamped, proving cardinality==1 when wired.
func TestRequestLoggerAnchorEmitsExactlyOneIngressSpanPerAcceptedRequest(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetIngressTaskID(r.Context(), "task-card-1")
		w.WriteHeader(http.StatusAccepted)
	})
	handler := RequestLogger(inner)
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-card-1"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := sink.Spans()
	ingressCount := 0
	for _, s := range spans {
		if s.Hop == e2e.HopIngress {
			ingressCount++
		}
	}
	if ingressCount != 1 {
		t.Fatalf("ingress hop produced %d spans for one accepted request; want exactly 1", ingressCount)
	}
	report := e2e.Assemble(spans)
	for _, a := range report.Anomalies {
		if a.Kind == e2e.AnomalyDuplicateSpan && a.Hop == e2e.HopIngress {
			t.Fatalf("e2e.Assemble flagged duplicate_task_hop for ingress: %+v", a)
		}
	}
}
