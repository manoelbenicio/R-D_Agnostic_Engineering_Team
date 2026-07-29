package service

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestNewQueueSpanIsMetadataOnlyAndValid(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewQueueSpan(QueueObservation{
		QueueMsgID: "qmsg-1", TaskID: "task-abc",
		WaitMs: 5, QueueDepth: 3, EnqueueUnixMs: 1700000000000, DequeueUnixMs: 1700000000005,
		Outcome: "dequeued",
	})
	if err := sp.Validate(); err != nil {
		t.Fatalf("queue span invalid: %v", err)
	}
	if err := rec.Emit(sp); err != nil {
		t.Fatalf("emit queue: %v", err)
	}
	if sink.Len() != 1 {
		t.Fatalf("sink len = %d, want 1", sink.Len())
	}
	got := sink.Spans()[0]
	if got.Hop != e2e.HopQueue || got.SecretsPresent {
		t.Fatalf("unexpected span shape: %+v", got)
	}
	if got.Correlation.QueueMsgID != "qmsg-1" || got.Correlation.TaskID != "task-abc" {
		t.Fatalf("correlation not carried: %+v", got.Correlation)
	}
	if len(got.Labels) != 0 {
		t.Fatalf("queue span must carry no labels, got %v", got.Labels)
	}
	for k := range got.Counters {
		switch k {
		case "wait_ms", "queue_depth", "enqueue_unix_ms", "dequeue_unix_ms":
		default:
			t.Fatalf("unexpected counter key %q (helper must stay metadata-only)", k)
		}
	}
	if report := e2e.ScanSpans([]e2e.Span{got}); !report.Clean {
		t.Fatalf("leak scan not clean: %+v", report.Findings)
	}
}

func TestNewQueueSpanFailsClosedOnMissingQueueMsgID(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewQueueSpan(QueueObservation{TaskID: "task-abc", WaitMs: 1, Outcome: "enqueued"})
	if err := rec.Emit(sp); err == nil {
		t.Fatal("queue span missing required queue_msg_id was emitted; want fail-closed refusal")
	}
	if sink.Len() != 0 {
		t.Fatalf("sink len = %d, want 0 after fail-closed refusal", sink.Len())
	}
}
