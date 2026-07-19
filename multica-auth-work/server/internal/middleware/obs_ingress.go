package middleware

import (
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// EmitIngressSpan emits OBS-2 (Hop 1) metrics: method, route, pseudonymous principal,
// HTTP status, latency, and request_id -> task_id. No request/response bodies.
func EmitIngressSpan(
	recorder *e2e.Recorder,
	reqID, taskID string,
	method, route string,
	principalPseudonym string,
	outcome, reason string,
	httpStatus int,
	startedAt time.Time,
) error {
	span := e2e.NewSpan(e2e.HopIngress, e2e.Correlation{
		RequestID: reqID,
		TaskID:    taskID,
	})
	span.StartedAt = startedAt

	if method != "" {
		span.WithLabel("method", method)
	}
	if route != "" {
		span.WithLabel("route", route)
	}
	if principalPseudonym != "" {
		span.WithLabel("principal_pseudonym", principalPseudonym)
	}

	span.WithOutcome(outcome, reason)
	span.WithHTTPStatus(httpStatus)
	span.Finish()
	span.WithCounter("latency_ms", span.DurationMs())

	return recorder.Emit(span)
}
