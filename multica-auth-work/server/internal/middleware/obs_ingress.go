package middleware

import (
	"errors"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

var errInvalidIngressInput = errors.New("invalid ingress span input")

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
	if recorder == nil {
		return errInvalidIngressInput
	}
	if method == "" || route == "" || principalPseudonym == "" {
		return errInvalidIngressInput
	}
	if httpStatus < 100 || httpStatus > 599 {
		return errInvalidIngressInput
	}
	if startedAt.IsZero() {
		return errInvalidIngressInput
	}

	span := e2e.NewSpan(e2e.HopIngress, e2e.Correlation{
		RequestID: reqID,
		TaskID:    taskID,
	})
	span.StartedAt = startedAt

	span.WithLabel("method", method)
	span.WithLabel("route_template", route)
	span.WithLabel("principal_pseudonym", principalPseudonym)

	span.WithOutcome(outcome, reason)
	span.WithHTTPStatus(httpStatus)
	span.Finish()

	latency := span.DurationMs()
	if latency >= 0 {
		span.WithCounter("latency_ms", latency)
	}

	return recorder.Emit(span)
}
