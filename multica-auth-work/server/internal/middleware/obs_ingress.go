package middleware

// obs_ingress.go — OBS-2 hop-1 (ingress) span helper for the end-to-end
// correlation contract (OpenSpec 6.2, AB-REQ-39/40). It is a pure, metadata-only
// builder over the FROZEN L5 contract `agent-brain.e2e.v1`
// (server/internal/daemon/observability/e2e). It never reads request/response
// bodies, headers, or query strings; it carries only bounded classification
// labels, a numeric latency counter, an HTTP status class, and the
// request_id/task_id correlation. Call sites live in shared anchors owned by L1
// (metrics/router chain) — this package emits no call sites of its own.

import (
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// IngressObservation is the metadata-only input for the ingress span. All
// fields are caller-derived classifications/counters — never raw content.
type IngressObservation struct {
	RequestID string // correlation: request_id (required)
	TaskID    string // correlation: task_id (required)

	Method         string // bounded token, e.g. "POST"; empty omits the label
	RouteTemplate  string // normalized route template, e.g. "/v1/tasks"; never a raw path
	PrincipalClass string // bounded principal class; empty omits the label

	HTTPStatus int    // 0..599 status class
	Outcome    string // bounded safe code, e.g. "accepted"
	ReasonCode string // optional bounded safe code
	LatencyMs  int64  // non-negative latency in milliseconds
}

// NewIngressSpan builds the finished hop-1 span. It only ever sets label keys
// and counter keys that the frozen contract allows for the ingress hop, so the
// result is metadata-only by construction. The span is validated (fail-closed)
// by the recorder on Emit.
func NewIngressSpan(obs IngressObservation) *e2e.Span {
	span := e2e.NewSpan(e2e.HopIngress, e2e.Correlation{
		RequestID: obs.RequestID,
		TaskID:    obs.TaskID,
	}).
		WithCounter("latency_ms", obs.LatencyMs).
		WithHTTPStatus(obs.HTTPStatus).
		WithOutcome(obs.Outcome, obs.ReasonCode)
	if obs.Method != "" {
		span = span.WithLabel("method", obs.Method)
	}
	if obs.RouteTemplate != "" {
		span = span.WithLabel("route_template", obs.RouteTemplate)
	}
	if obs.PrincipalClass != "" {
		span = span.WithLabel("principal_class", obs.PrincipalClass)
	}
	return span.Finish()
}

// EmitIngress builds and emits the ingress span through rec. A nil recorder is
// a no-op (instrumentation must never crash a caller). Any content or invalid
// field causes the recorder to refuse the span fail-closed and return an error
// the caller classifies without printing values.
func EmitIngress(rec *e2e.Recorder, obs IngressObservation) error {
	if rec == nil {
		return nil
	}
	return rec.Emit(NewIngressSpan(obs))
}
