package daemon

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/daemon/observability/otlpreceiver"
)

type failOnceSink struct {
	mu        sync.Mutex
	calls     int
	persisted int
	failFirst bool
}

func (f *failOnceSink) Record(e2e.Span) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.failFirst && f.calls == 1 {
		return errors.New("simulated jsonl write failure")
	}
	f.persisted++
	return nil
}

func validRouteRecord() otlpreceiver.SanitizedRecord {
	return otlpreceiver.SanitizedRecord{
		EventName: "api_request", RequestID: "omni-1", ClientRequestID: "c-1",
		Model: "claude_code_kimi_2.7_Code", Status: "200", DurationMs: 120,
		StartUnixNano: uint64(time.Now().UnixNano()),
		TrustedTaskID: "task-1", TrustedRequestID: "req-1",
	}
}

func TestRouteSinkRetryEmitsExactlyOnce(t *testing.T) {
	sink := &failOnceSink{failFirst: true}
	rs := newRouteSpanSink(e2e.NewRecorder(sink), nil)
	rec := validRouteRecord()
	if err := rs.Record(rec); err == nil {
		t.Fatal("first Emit must return error so receiver retries")
	}
	if err := rs.Record(rec); err != nil { // retry
		t.Fatalf("retry must succeed: %v", err)
	}
	if err := rs.Record(rec); err != nil { // duplicate api_request
		t.Fatalf("duplicate must be accepted without error: %v", err)
	}
	if sink.persisted != 1 {
		t.Fatalf("exactly one route span must persist, got %d", sink.persisted)
	}
}

func TestRouteSinkConcurrentSameTaskEmitsOnce(t *testing.T) {
	sink := &failOnceSink{}
	rs := newRouteSpanSink(e2e.NewRecorder(sink), nil)
	rec := validRouteRecord()
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = rs.Record(rec) }()
	}
	wg.Wait()
	if sink.persisted != 1 {
		t.Fatalf("concurrent same-task distinct events must persist one span, got %d", sink.persisted)
	}
}

func TestRouteSinkRejectsOverflowAndMissing(t *testing.T) {
	sink := &failOnceSink{}
	rs := newRouteSpanSink(e2e.NewRecorder(sink), nil)
	bad := []otlpreceiver.SanitizedRecord{
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.StartUnixNano = 0; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.StartUnixNano = math.MaxUint64; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.DurationMs = -1; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.DurationMs = maxRouteDurationMs + 1; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.TrustedRequestID = ""; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.RequestID = ""; return r }(),
		func() otlpreceiver.SanitizedRecord { r := validRouteRecord(); r.TrustedTaskID = ""; return r }(),
	}
	for i, r := range bad {
		if err := rs.Record(r); err == nil {
			t.Fatalf("case %d must be rejected", i)
		}
	}
	if sink.persisted != 0 {
		t.Fatalf("no invalid record may persist, got %d", sink.persisted)
	}
}
