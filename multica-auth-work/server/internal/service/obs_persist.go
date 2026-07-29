package service

// obs_persist.go — OBS-7 hop-6 (terminal persistence) span helper for the
// end-to-end correlation contract (OpenSpec 6.2, AB-REQ-39/40). Pure,
// metadata-only builder over the FROZEN L5 contract `agent-brain.e2e.v1`. It
// carries no result content — only persist latency, numeric byte/token counts,
// a bounded terminal-status class, and the task_id/result_id correlation. The
// terminal-result persist call site lives in the shared anchor
// server/internal/service/task.go, owned by L1 (Wave C).

import (
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// PersistObservation is the metadata-only input for the persist span. Counts
// are numeric only; no result bytes/text are accepted.
type PersistObservation struct {
	TaskID   string // correlation: task_id (required)
	ResultID string // correlation: result_id (required)

	TerminalStatus string // bounded status class, e.g. "completed"/"blocked"; empty omits the label

	PersistLatencyMs int64 // non-negative persistence latency in milliseconds
	ByteCount        int64 // non-negative persisted byte count
	TokenCount       int64 // non-negative token count

	Outcome    string // bounded safe code
	ReasonCode string // optional bounded safe code
}

// NewPersistSpan builds the finished hop-6 span using only the frozen
// persist-hop counter keys and the bounded terminal_status label, so it is
// metadata-only by construction.
func NewPersistSpan(obs PersistObservation) *e2e.Span {
	span := e2e.NewSpan(e2e.HopPersist, e2e.Correlation{
		TaskID:   obs.TaskID,
		ResultID: obs.ResultID,
	}).
		WithCounter("persist_latency_ms", obs.PersistLatencyMs).
		WithCounter("byte_count", obs.ByteCount).
		WithCounter("token_count", obs.TokenCount).
		WithOutcome(obs.Outcome, obs.ReasonCode)
	if obs.TerminalStatus != "" {
		span = span.WithLabel("terminal_status", obs.TerminalStatus)
	}
	return span.Finish()
}

// EmitPersist builds and emits the persist span through rec. A nil recorder is
// a no-op; content or invalid fields are refused fail-closed by the recorder.
func EmitPersist(rec *e2e.Recorder, obs PersistObservation) error {
	if rec == nil {
		return nil
	}
	return rec.Emit(NewPersistSpan(obs))
}
