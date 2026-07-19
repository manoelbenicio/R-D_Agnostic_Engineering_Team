package daemonws

import (
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// EmitDeliverySpan emits OBS-8 (Hop 7) metrics: delivery latency, backpressure/drops,
// reconnects, joined on session_id/delivery_id. No delivered payload content.
func EmitDeliverySpan(
	recorder *e2e.Recorder,
	sessionID, deliveryID string,
	outcome, reason string,
	startedAt time.Time,
	deliveryLatencyMs int64,
	dropCount int64,
	reconnectCount int64,
	backpressureCount int64,
	backpressureState string,
) error {
	span := e2e.NewSpan(e2e.HopDelivery, e2e.Correlation{
		SessionID:  sessionID,
		DeliveryID: deliveryID,
	})
	span.StartedAt = startedAt

	if backpressureState != "" {
		span.WithLabel("backpressure_state", backpressureState)
	}

	span.WithOutcome(outcome, reason)
	span.Finish()

	span.WithCounter("delivery_latency_ms", deliveryLatencyMs)
	if dropCount > 0 {
		span.WithCounter("drop_count", dropCount)
	}
	if reconnectCount > 0 {
		span.WithCounter("reconnect_count", reconnectCount)
	}
	if backpressureCount > 0 {
		span.WithCounter("backpressure_count", backpressureCount)
	}

	return recorder.Emit(span)
}
