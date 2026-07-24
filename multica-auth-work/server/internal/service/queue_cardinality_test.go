package service

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// The queue hop must produce EXACTLY ONE span per task across its full
// lifecycle (enqueue then dequeue), emitted at dequeue and carrying both
// timestamps. A second span would trip the assembler's duplicate_task_hop.
func TestQueueEmitsExactlyOneSpanPerTaskLifecycle(t *testing.T) {
	sink := e2e.NewMemorySink()
	s := &TaskService{Obs: e2e.NewRecorder(sink)}

	task := db.AgentTaskQueue{
		ID:        pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-2 * time.Second), Valid: true},
	}

	s.emitQueueEnqueued(task) // must NOT emit a span
	s.emitQueueDequeued(task) // the single completed queue span

	spans := sink.Spans()
	queueSpans := 0
	for _, sp := range spans {
		if sp.Hop == e2e.HopQueue {
			queueSpans++
			if sp.Counters["enqueue_unix_ms"] == 0 || sp.Counters["dequeue_unix_ms"] == 0 {
				t.Fatalf("dequeue span must carry both enqueue and dequeue timestamps: %+v", sp.Counters)
			}
		}
	}
	if queueSpans != 1 {
		t.Fatalf("expected exactly one queue span per task lifecycle, got %d", queueSpans)
	}
}
