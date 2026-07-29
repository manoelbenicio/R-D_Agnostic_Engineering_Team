package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// One HTTP request may enqueue MULTIPLE tasks (agent/mention/squad fan-out). The
// RequestLogger must emit exactly one ingress span PER DISTINCT accepted task —
// each with that task's canonical request id and the same HTTP metadata — and
// must NOT duplicate when the same task is added twice.
func TestRequestLoggerAnchorMultiAndDuplicateTasks(t *testing.T) {
	sink := e2e.NewMemorySink()
	SetIngressRecorder(e2e.NewRecorder(sink))
	t.Cleanup(func() { SetIngressRecorder(nil) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetIngressTaskID(r.Context(), "task-a")
		SetIngressTaskID(r.Context(), "task-b")
		SetIngressTaskID(r.Context(), "task-a") // duplicate -> must not double
		SetIngressTaskID(r.Context(), "task-c")
		w.WriteHeader(http.StatusAccepted)
	})
	handler := RequestLogger(inner)
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-x"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := sink.Spans()
	want := map[string]string{
		"task-a": e2e.CanonicalRequestID("task-a"),
		"task-b": e2e.CanonicalRequestID("task-b"),
		"task-c": e2e.CanonicalRequestID("task-c"),
	}
	seen := map[string]int{}
	for _, sp := range spans {
		if sp.Hop != e2e.HopIngress {
			continue
		}
		seen[sp.Correlation.TaskID]++
		if want[sp.Correlation.TaskID] != sp.Correlation.RequestID {
			t.Fatalf("task %s request_id=%q not canonical", sp.Correlation.TaskID, sp.Correlation.RequestID)
		}
		if sp.Labels["method"] != http.MethodPost {
			t.Fatalf("ingress span must carry real HTTP method, got %q", sp.Labels["method"])
		}
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct ingress spans (one per task), got %d (%v)", len(seen), seen)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("task %s emitted %d ingress spans, want exactly 1 (dedup)", id, n)
		}
	}
}

func TestIngressHolderBoundFailsClosedBeyondLimit(t *testing.T) {
	h := &ingressHolder{}
	for i := 0; i < maxIngressTasksPerRequest+10; i++ {
		h.add("task-" + string(rune('A'+i%26)) + string(rune('0'+i/26)))
	}
	ids, overflowed := h.snapshot()
	if len(ids) != maxIngressTasksPerRequest {
		t.Fatalf("holder must cap at %d, got %d", maxIngressTasksPerRequest, len(ids))
	}
	if !overflowed {
		t.Fatal("holder must flag overflow beyond the bound (fail closed)")
	}
}
