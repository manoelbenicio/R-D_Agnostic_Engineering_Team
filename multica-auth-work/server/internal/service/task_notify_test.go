package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// stubWakeup records every call so the test can assert that notify
// reaches the daemon hub and carries the right runtime / task IDs.
type stubWakeup struct {
	calls []struct{ runtimeID, taskID string }
}

func (s *stubWakeup) NotifyTaskAvailable(runtimeID, taskID string) {
	s.calls = append(s.calls, struct{ runtimeID, taskID string }{runtimeID, taskID})
}

// TestNotifyTaskAvailable_BumpsBeforeWakeup pins the contract noted in
// the EmptyClaimCache docs: the version Bump MUST run before the
// daemon WS wakeup, otherwise the wakeup-driven claim could read a
// still-current empty verdict and return null while the freshly
// queued task sits idle. The test (1) marks the runtime empty under
// the current version, (2) fires notifyTaskAvailable, then (3)
// asserts the prior verdict is rejected AND the wakeup hook saw the
// new task — proving every enqueue path (issue / mention /
// quick-create / chat / autopilot / retry) gets the same
// bump-then-notify behaviour for free.
func TestNotifyTaskAvailable_BumpsBeforeWakeup(t *testing.T) {
	rdb := newRedisTestClient(t)
	cache := NewEmptyClaimCache(rdb)
	wakeup := &stubWakeup{}

	svc := &TaskService{
		EmptyClaim: cache,
		Wakeup:     wakeup,
	}

	runtimeID := testUUID(7)
	taskID := testUUID(8)
	runtimeKey := util.UUIDToString(runtimeID)

	ctx := context.Background()
	v0 := cache.CurrentVersion(ctx, runtimeKey)
	cache.MarkEmpty(ctx, runtimeKey, v0)
	if !cache.IsEmpty(ctx, runtimeKey) {
		t.Fatal("precondition: cache should report empty after MarkEmpty under current version")
	}

	svc.notifyTaskAvailable(db.AgentTaskQueue{
		ID:        taskID,
		RuntimeID: runtimeID,
	})

	if cache.IsEmpty(ctx, runtimeKey) {
		t.Fatal("notifyTaskAvailable must Bump the version so the prior empty verdict is rejected")
	}
	if got := len(wakeup.calls); got != 1 {
		t.Fatalf("expected 1 wakeup call, got %d", got)
	}
	if wakeup.calls[0].runtimeID != runtimeKey {
		t.Fatalf("wakeup runtime mismatch: got %q want %q", wakeup.calls[0].runtimeID, runtimeKey)
	}
	if wakeup.calls[0].taskID != util.UUIDToString(taskID) {
		t.Fatalf("wakeup task mismatch: got %q want %q", wakeup.calls[0].taskID, util.UUIDToString(taskID))
	}
}

// TestNotifyTaskAvailable_InvalidWithoutRuntimeIsNoOp guards the
// no-RuntimeID early return — chat / quick-create / autopilot all set
// it on insert, but a buggy caller that forgot must not silently bump
// every workspace's version. The cache treats Bump("") as a no-op,
// but this test pins that the RuntimeID guard sits above the Bump
// call so a future refactor cannot drop the guard without test
// coverage.
func TestNotifyTaskAvailable_InvalidWithoutRuntimeIsNoOp(t *testing.T) {
	rdb := newRedisTestClient(t)
	cache := NewEmptyClaimCache(rdb)
	wakeup := &stubWakeup{}

	svc := &TaskService{
		EmptyClaim: cache,
		Wakeup:     wakeup,
	}

	ctx := context.Background()
	v0 := cache.CurrentVersion(ctx, "rt-stays")
	cache.MarkEmpty(ctx, "rt-stays", v0)

	svc.notifyTaskAvailable(db.AgentTaskQueue{
		// RuntimeID intentionally invalid (zero value, Valid=false).
		ID: testUUID(9),
	})

	if !cache.IsEmpty(ctx, "rt-stays") {
		t.Fatal("notifyTaskAvailable with invalid RuntimeID must not touch cache")
	}
	if got := len(wakeup.calls); got != 0 {
		t.Fatalf("expected 0 wakeup calls when RuntimeID is invalid, got %d", got)
	}
}

// --- F6: queue-hop (OBS-3) span wiring at enqueue/dequeue (metadata-only) ---

func TestEmitQueueEnqueuedRecordsMetadataOnlySpan(t *testing.T) {
	// Cardinality contract: the queue hop emits EXACTLY ONE completed span per
	// task, at DEQUEUE. Enqueue must NOT emit a span (a second span would trip
	// the assembler's duplicate_task_hop). This asserts enqueue is a no-op.
	sink := e2e.NewMemorySink()
	svc := &TaskService{Obs: e2e.NewRecorder(sink)}
	svc.emitQueueEnqueued(db.AgentTaskQueue{
		ID:        testUUID(11),
		CreatedAt: pgtype.Timestamptz{Time: time.Now().Add(-2 * time.Second), Valid: true},
	})
	if sink.Len() != 0 {
		t.Fatalf("enqueue must emit NO span (single completed queue span is at dequeue), got %d", sink.Len())
	}
}

func TestEmitQueueDequeuedRecordsWaitAndMetadataOnly(t *testing.T) {
	sink := e2e.NewMemorySink()
	svc := &TaskService{Obs: e2e.NewRecorder(sink)}
	id := testUUID(12)
	created := time.Now().Add(-3 * time.Second)
	svc.emitQueueDequeued(db.AgentTaskQueue{
		ID:        id,
		CreatedAt: pgtype.Timestamptz{Time: created, Valid: true},
	})
	if sink.Len() != 1 {
		t.Fatalf("sink len = %d, want 1", sink.Len())
	}
	sp := sink.Spans()[0]
	if sp.Hop != e2e.HopQueue || sp.Outcome != "dequeued" {
		t.Fatalf("unexpected dequeue span: hop=%q outcome=%q", sp.Hop, sp.Outcome)
	}
	if sp.Counters["dequeue_unix_ms"] <= 0 {
		t.Fatal("dequeue_unix_ms not set")
	}
	if sp.Counters["wait_ms"] < 0 {
		t.Fatal("wait_ms must be non-negative")
	}
	if r := e2e.ScanSpans([]e2e.Span{sp}); !r.Clean {
		t.Fatalf("leak scan not clean: %+v", r.Findings)
	}
}

func TestEmitQueueNilObsIsNoop(t *testing.T) {
	svc := &TaskService{} // Obs nil
	// Must not panic and must not require a sink.
	svc.emitQueueEnqueued(db.AgentTaskQueue{ID: testUUID(13)})
	svc.emitQueueDequeued(db.AgentTaskQueue{ID: testUUID(13)})
}
