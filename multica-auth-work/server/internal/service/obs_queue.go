package service

// obs_queue.go — OBS-3 hop-2 (DB queue) span helper for the end-to-end
// correlation contract (OpenSpec 6.2, AB-REQ-39/40). Pure, metadata-only builder
// over the FROZEN L5 contract `agent-brain.e2e.v1`. It carries no task payload —
// only enqueue/dequeue timestamps, queue depth, wait duration, and the
// queue_msg_id/task_id correlation. Enqueue/dequeue call sites live in the
// shared anchor server/internal/service/task.go, owned by L1 (Wave C).

import (
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// QueueObservation is the metadata-only input for the queue span. Timestamps
// are unix-millisecond counters; no task content is accepted.
type QueueObservation struct {
	QueueMsgID string // correlation: queue_msg_id (required)
	TaskID     string // correlation: task_id (required)

	WaitMs        int64 // non-negative wait time in milliseconds
	QueueDepth    int64 // non-negative queue depth at the observation
	EnqueueUnixMs int64 // enqueue timestamp (unix ms), non-negative
	DequeueUnixMs int64 // dequeue timestamp (unix ms), non-negative; 0 when not yet dequeued

	Outcome    string // bounded safe code, e.g. "enqueued" / "dequeued"
	ReasonCode string // optional bounded safe code
}

// NewQueueSpan builds the finished hop-2 span using only the frozen queue-hop
// counter keys, so it is metadata-only by construction.
func NewQueueSpan(obs QueueObservation) *e2e.Span {
	return e2e.NewSpan(e2e.HopQueue, e2e.Correlation{
		QueueMsgID: obs.QueueMsgID,
		TaskID:     obs.TaskID,
	}).
		WithCounter("wait_ms", obs.WaitMs).
		WithCounter("queue_depth", obs.QueueDepth).
		WithCounter("enqueue_unix_ms", obs.EnqueueUnixMs).
		WithCounter("dequeue_unix_ms", obs.DequeueUnixMs).
		WithOutcome(obs.Outcome, obs.ReasonCode).
		Finish()
}

// EmitQueue builds and emits the queue span through rec. A nil recorder is a
// no-op; content or invalid fields are refused fail-closed by the recorder.
func EmitQueue(rec *e2e.Recorder, obs QueueObservation) error {
	if rec == nil {
		return nil
	}
	return rec.Emit(NewQueueSpan(obs))
}
