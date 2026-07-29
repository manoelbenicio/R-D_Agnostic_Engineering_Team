package daemonws

import (
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// DeliveryRecorder emits metadata-only HopDelivery spans via the e2e Recorder.
// It does NOT carry delivered payloads, message bodies, or any content — only
// the structural metadata that the correlation contract requires: session_id,
// delivery_id, and bounded counters (delivery_latency_ms, drop_count, etc.).
//
// Call sites live in hub.go (owned by L1). This helper provides the span
// construction and validation; the hub calls EmitDelivery at the point where a
// WS frame is confirmed delivered or dropped.
type DeliveryRecorder struct {
	recorder *e2e.Recorder
}

// NewDeliveryRecorder wraps an e2e.Recorder for the delivery hop. A nil
// recorder is safe — the e2e.Recorder's NewRecorder(nil) already provides a
// no-op sink.
func NewDeliveryRecorder(rec *e2e.Recorder) *DeliveryRecorder {
	return &DeliveryRecorder{recorder: rec}
}

// DeliveryResult captures the metadata for one WS delivery attempt. All fields
// are classification-level — no message content, no free-form strings.
type DeliveryResult struct {
	SessionID  string // WS session (required)
	DeliveryID string // unique per delivery attempt (required)

	// Outcome: "delivered", "dropped", "backpressure", "disconnected"
	Outcome    string
	ReasonCode string // optional machine-readable reason (e.g. "slow_consumer")

	// Bounded counters
	LatencyMs         int64
	DropCount         int64
	ReconnectCount    int64
	BackpressureCount int64
}

// EmitDelivery constructs and emits a HopDelivery span from the given result.
// Returns an error only when the span fails validation (never panics). The
// caller MUST NOT retry failed emissions — fail-closed is the correct behavior.
func (d *DeliveryRecorder) EmitDelivery(r DeliveryResult) error {
	if d.recorder == nil {
		return nil
	}

	corr := e2e.Correlation{
		SessionID:  r.SessionID,
		DeliveryID: r.DeliveryID,
	}

	span := e2e.NewSpan(e2e.HopDelivery, corr).
		WithOutcome(r.Outcome, r.ReasonCode).
		WithCounter("delivery_latency_ms", r.LatencyMs).
		WithCounter("drop_count", r.DropCount).
		WithCounter("reconnect_count", r.ReconnectCount).
		WithCounter("backpressure_count", r.BackpressureCount).
		Finish()

	return d.recorder.Emit(span)
}
