package service

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TestQueueHopEmitsExactlyOneLifecycleSpanPerTask is a CARDINALITY anchor test at
// the REAL queue call sites (emitQueueEnqueued @ task.go:186 and emitQueueDequeued
// @ task.go:199). The end-to-end correlation contract requires EXACTLY ONE
// HopQueue span per task: two HopQueue spans sharing one task_id make
// e2e.Assemble record an AnomalyDuplicateSpan (duplicate_task_hop) and force
// AllContinuous=false.
//
// RED-BY-DESIGN: this FAILS against the current double-emit (enqueued + dequeued
// are two separate HopQueue spans for the same task) and is the executable
// specification of that defect (see T20-live-correlation-audit.md M5). It goes
// green once the queue hop emits a single completed lifecycle span per task
// (e.g. one span at dequeue carrying wait_ms, not a separate enqueue span).
func TestQueueHopEmitsExactlyOneLifecycleSpanPerTask(t *testing.T) {
	sink := e2e.NewMemorySink()
	svc := &TaskService{Obs: e2e.NewRecorder(sink)}
	row := db.AgentTaskQueue{
		ID:        testUUID(41),
		CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-2 * time.Second), Valid: true},
	}

	// Drive the actual production lifecycle call sites for ONE task.
	svc.emitQueueEnqueued(row) // enqueue
	svc.emitQueueDequeued(row) // claim/dequeue

	spans := sink.Spans()
	queueCount := 0
	for _, s := range spans {
		if s.Hop == e2e.HopQueue {
			queueCount++
		}
	}
	if queueCount != 1 {
		t.Fatalf("queue hop produced %d HopQueue spans for one task; want EXACTLY 1 completed "+
			"lifecycle span. enqueue+dequeue double-emit makes e2e.Assemble flag duplicate_task_hop. "+
			"Fix: emit a single completed queue span per task.", queueCount)
	}

	report := e2e.Assemble(spans)
	for _, a := range report.Anomalies {
		if a.Kind == e2e.AnomalyDuplicateSpan && a.Hop == e2e.HopQueue {
			t.Fatalf("e2e.Assemble flagged duplicate_task_hop for the queue hop: %+v", a)
		}
	}
	if len(report.Orphans) != 0 {
		t.Fatalf("unexpected orphans for a single-task queue lifecycle: %+v", report.Orphans)
	}
}
