package realtime

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func newTestDeliveryObserver(rec *e2e.Recorder) *deliveryObserver {
	o := newDeliveryObserver()
	o.setRecorder(rec)
	o.now = func() time.Time { return time.Unix(0, 0).UTC() }
	var n int64
	o.newID = func() string { n++; return "delivery-test-" + itoa2(n) }
	return o
}

func itoa2(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// flakySink fails the first Record call, then succeeds — to prove the observer
// marks (session,event) seen ONLY after a successful Emit (retry-safe).
type flakySink struct {
	mu     sync.Mutex
	failed bool
	spans  []e2e.Span
}

func (s *flakySink) Record(sp e2e.Span) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.failed {
		s.failed = true
		return errors.New("synthetic sink failure")
	}
	s.spans = append(s.spans, sp)
	return nil
}
func (s *flakySink) count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.spans) }

func termMeta() TerminalDeliveryMeta {
	return TerminalDeliveryMeta{TaskID: "task-1", ChatSessionID: "chat-1", Event: EventTaskCompleted}
}

func addRoomClient(t *testing.T, h *Hub, scopeType, scopeID string, deliverable bool) *Client {
	t.Helper()
	var ch chan []byte
	if deliverable {
		ch = make(chan []byte, 8)
	} else {
		ch = make(chan []byte) // unbuffered + no reader => slow (backpressure)
	}
	c := &Client{send: ch}
	key := sk(scopeType, scopeID)
	h.mu.Lock()
	if h.rooms[key] == nil {
		h.rooms[key] = make(map[*Client]bool)
	}
	h.rooms[key][c] = true
	h.clients[c] = true
	h.mu.Unlock()
	return c
}

func assertOneDeliverySpan(t *testing.T, sink *e2e.MemorySink) e2e.Span {
	t.Helper()
	if sink.Len() != 1 {
		t.Fatalf("want exactly 1 delivery span, got %d", sink.Len())
	}
	sp := sink.Spans()[0]
	if sp.Hop != e2e.HopDelivery || sp.SecretsPresent {
		t.Fatalf("unexpected span shape: %+v", sp)
	}
	if sp.Correlation.SessionID == "" || sp.Correlation.DeliveryID == "" {
		t.Fatalf("missing required delivery correlation ids: %+v", sp.Correlation)
	}
	if r := e2e.ScanSpans([]e2e.Span{sp}); !r.Clean {
		t.Fatalf("delivery span not metadata-clean: %+v", r.Findings)
	}
	return sp
}

func TestTerminalDeliveryManyClientsEmitExactlyOneDeliveredSpan(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(e2e.NewRecorder(sink))
	h.deliveryObs.newID = func() string { return "delivery-tabs-01" }
	for i := 0; i < 5; i++ { // 5 browser tabs in the same chat scope
		addRoomClient(t, h, ScopeChat, "chat-1", true)
	}
	h.BroadcastTerminalDelivery(ScopeChat, "chat-1", []byte(`{"e":"x"}`), termMeta(), "evt-1")

	sp := assertOneDeliverySpan(t, sink)
	if sp.Outcome != "delivered" {
		t.Fatalf("outcome=%q, want delivered (>=1 client enqueued)", sp.Outcome)
	}
	if sp.Correlation.SessionID != e2e.CanonicalSessionID("chat-1", "task-1") {
		t.Fatalf("session_id=%q not canonical", sp.Correlation.SessionID)
	}
}

func TestTerminalDeliveryZeroClientsIsDropped(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	if !o.emitTerminalDelivery(termMeta(), 0, 0, 0) {
		t.Fatal("expected a span for a terminal event with zero clients")
	}
	sp := assertOneDeliverySpan(t, sink)
	if sp.Outcome != "dropped" {
		t.Fatalf("outcome=%q, want dropped (0 clients)", sp.Outcome)
	}
	if sp.Counters["drop_count"] < 1 {
		t.Fatalf("drop_count=%d, want >=1 (one aggregate frame dropped)", sp.Counters["drop_count"])
	}
}

func TestTerminalDeliverySlowConsumerBackpressure(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(e2e.NewRecorder(sink))
	h.deliveryObs.newID = func() string { return "delivery-bp-01" }
	addRoomClient(t, h, ScopeChat, "chat-2", true)  // deliverable
	addRoomClient(t, h, ScopeChat, "chat-2", false) // slow -> backpressure
	h.BroadcastTerminalDelivery(ScopeChat, "chat-2", []byte(`{}`),
		TerminalDeliveryMeta{TaskID: "task-2", ChatSessionID: "chat-2", Event: EventTaskFailed}, "evt-2")

	sp := assertOneDeliverySpan(t, sink)
	if sp.Outcome != "delivered" { // >=1 enqueued
		t.Fatalf("outcome=%q, want delivered", sp.Outcome)
	}
	if sp.Counters["backpressure_count"] != 1 || sp.Counters["drop_count"] != 1 {
		t.Fatalf("backpressure/drop counters=%v, want 1/1", sp.Counters)
	}
	if sp.Labels["backpressure_state"] != "backpressure" {
		t.Fatalf("backpressure_state=%q", sp.Labels["backpressure_state"])
	}
}

func TestTerminalDeliveryDedupRedisLoopbackOneSpan(t *testing.T) {
	// Observer-level: two emits for the same (session,event) -> one span.
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	first := o.emitTerminalDelivery(termMeta(), 2, 0, 0)
	second := o.emitTerminalDelivery(termMeta(), 2, 0, 0)
	if !first || second {
		t.Fatalf("dedup: first=%v second=%v, want true/false", first, second)
	}
	assertOneDeliverySpan(t, sink)

	// Hub-level: the same terminal frame replayed (local + Redis loopback) -> one span.
	sink2 := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(e2e.NewRecorder(sink2))
	h.deliveryObs.newID = func() string { return "delivery-loop-01" }
	addRoomClient(t, h, ScopeChat, "chat-3", true)
	meta := TerminalDeliveryMeta{TaskID: "task-3", ChatSessionID: "chat-3", Event: EventTaskCancelled}
	h.BroadcastTerminalDelivery(ScopeChat, "chat-3", []byte(`{}`), meta, "evt-3")
	h.BroadcastTerminalDelivery(ScopeChat, "chat-3", []byte(`{}`), meta, "evt-3")
	if sink2.Len() != 1 {
		t.Fatalf("Redis loopback produced %d spans, want exactly 1", sink2.Len())
	}
}

func TestTerminalDeliveryConcurrentSameTaskOneSpan(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	var emitted int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if o.emitTerminalDelivery(termMeta(), 3, 1, 0) {
				atomic.AddInt64(&emitted, 1)
			}
		}()
	}
	wg.Wait()
	if emitted != 1 {
		t.Fatalf("concurrent emits produced %d spans, want exactly 1", emitted)
	}
	assertOneDeliverySpan(t, sink)
}

func TestTerminalDeliveryRecorderFirstFailRetriesNoFalseSuccess(t *testing.T) {
	sink := &flakySink{}
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	if o.emitTerminalDelivery(termMeta(), 1, 0, 0) {
		t.Fatal("first emit reported success despite sink failure (false success)")
	}
	if sink.count() != 0 {
		t.Fatalf("sink recorded %d spans after a failed Emit, want 0", sink.count())
	}
	// Not marked seen -> the retry succeeds and records exactly one span.
	if !o.emitTerminalDelivery(termMeta(), 1, 0, 0) {
		t.Fatal("retry after sink recovery did not emit")
	}
	if sink.count() != 1 {
		t.Fatalf("after retry sink has %d spans, want exactly 1", sink.count())
	}
	// Now seen -> a third call is a no-op.
	if o.emitTerminalDelivery(termMeta(), 1, 0, 0) {
		t.Fatal("third emit re-recorded an already-seen (session,event)")
	}
}

func TestTerminalDeliveryBoundedDedupState(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	total := deliveryDedupCapacity + 200
	for i := 0; i < total; i++ {
		// Distinct canonical session per iteration: CanonicalSessionID prefers
		// chat_session_id, so vary the task id with an empty chat id.
		meta := TerminalDeliveryMeta{TaskID: "task-" + itoa2(int64(i)), ChatSessionID: "", Event: EventTaskCompleted}
		if !o.emitTerminalDelivery(meta, 1, 0, 0) {
			t.Fatalf("emit %d unexpectedly deduped", i)
		}
	}
	o.mu.Lock()
	seenLen := len(o.seen)
	listLen := len(o.seenList)
	o.mu.Unlock()
	if seenLen != deliveryDedupCapacity || listLen != deliveryDedupCapacity {
		t.Fatalf("dedup state unbounded: seen=%d list=%d, want %d", seenLen, listLen, deliveryDedupCapacity)
	}
}

func TestTerminalDeliveryCanonicalSessionID(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	meta := TerminalDeliveryMeta{TaskID: "T", ChatSessionID: "C", Event: EventTaskCompleted}
	o.emitTerminalDelivery(meta, 1, 0, 0)
	sp := assertOneDeliverySpan(t, sink)
	if got, want := sp.Correlation.SessionID, e2e.CanonicalSessionID("C", "T"); got != want {
		t.Fatalf("session_id=%q, want canonical %q", got, want)
	}
}

func TestTerminalDeliveryNoNonTerminalEmission(t *testing.T) {
	// Observer-level.
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	for _, ev := range []string{"task:progress", "task:started", "chat:message", ""} {
		if o.emitTerminalDelivery(TerminalDeliveryMeta{TaskID: "t", ChatSessionID: "c", Event: ev}, 3, 0, 0) {
			t.Fatalf("non-terminal event %q emitted a span", ev)
		}
	}
	if sink.Len() != 0 {
		t.Fatalf("non-terminal events produced %d spans, want 0", sink.Len())
	}
	// Hub-level: a non-terminal frame is fanned out but records no span.
	h := NewHub()
	h.SetDeliveryRecorder(e2e.NewRecorder(sink))
	addRoomClient(t, h, ScopeChat, "chat-nt", true)
	h.BroadcastTerminalDelivery(ScopeChat, "chat-nt", []byte(`{}`),
		TerminalDeliveryMeta{TaskID: "t", ChatSessionID: "chat-nt", Event: "task:progress"}, "evt-nt")
	if sink.Len() != 0 {
		t.Fatalf("hub non-terminal produced %d spans, want 0", sink.Len())
	}
}

func TestTerminalDeliveryNilRecorderIsNoop(t *testing.T) {
	o := newDeliveryObserver() // no recorder
	if o.emitTerminalDelivery(termMeta(), 5, 0, 0) {
		t.Fatal("emit with nil recorder must be a no-op")
	}
	h := NewHub() // recorder never set
	addRoomClient(t, h, ScopeChat, "chat-nil", true)
	h.BroadcastTerminalDelivery(ScopeChat, "chat-nil", []byte(`{}`), termMeta(), "evt-nil") // must not panic
}

func TestTerminalDeliveryDistinctTasksSameSessionEmitTwoSpans(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	metaA := TerminalDeliveryMeta{TaskID: "taskA", ChatSessionID: "chatX", Event: EventTaskCompleted}
	metaB := TerminalDeliveryMeta{TaskID: "taskB", ChatSessionID: "chatX", Event: EventTaskCompleted}

	if !o.emitTerminalDelivery(metaA, 1, 0, 0) {
		t.Fatal("taskA terminal not emitted")
	}
	if !o.emitTerminalDelivery(metaB, 1, 0, 0) {
		t.Fatal("taskB terminal (same session, different task) must NOT collapse into taskA")
	}
	if o.emitTerminalDelivery(metaA, 1, 0, 0) {
		t.Fatal("taskA replay must dedup to a single span")
	}
	if sink.Len() != 2 {
		t.Fatalf("distinct tasks in one session produced %d spans, want exactly 2", sink.Len())
	}
	// Both spans JOIN on the identical canonical session id (chat-session seed).
	want := e2e.CanonicalSessionID("chatX", "taskA")
	if want != e2e.CanonicalSessionID("chatX", "taskB") {
		t.Fatal("precondition: same chat session must canonicalize identically")
	}
	for _, sp := range sink.Spans() {
		if sp.Correlation.SessionID != want {
			t.Fatalf("span session_id=%q, want canonical join %q", sp.Correlation.SessionID, want)
		}
	}
}

func TestTerminalDeliveryRequiresNonEmptyTaskID(t *testing.T) {
	sink := e2e.NewMemorySink()
	o := newTestDeliveryObserver(e2e.NewRecorder(sink))
	// Empty task id: even though chat_session_id would yield a non-empty
	// canonical session, the dedup key needs the raw task id -> no span.
	if o.emitTerminalDelivery(TerminalDeliveryMeta{TaskID: "  ", ChatSessionID: "chatY", Event: EventTaskCompleted}, 1, 0, 0) {
		t.Fatal("terminal delivery with empty task id must not emit")
	}
	if sink.Len() != 0 {
		t.Fatalf("empty-task-id produced %d spans, want 0", sink.Len())
	}
}

func TestHubImplementsTerminalBroadcaster(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(e2e.NewRecorder(sink))
	h.deliveryObs.newID = func() string { return "delivery-tb-01" }
	addRoomClient(t, h, ScopeWorkspace, "ws-1", true)

	var b TerminalBroadcaster = h // type-assert seam the listener uses
	b.BroadcastTerminalToWorkspace("ws-1", []byte(`{}`),
		TerminalDeliveryMeta{TaskID: "task-tb", ChatSessionID: "", Event: EventTaskCompleted})
	if sink.Len() != 1 {
		t.Fatalf("TerminalBroadcaster produced %d spans, want 1", sink.Len())
	}
	// Non-terminal via the interface fans out but records no span.
	b.BroadcastTerminalToWorkspace("ws-1", []byte(`{}`),
		TerminalDeliveryMeta{TaskID: "task-tb2", ChatSessionID: "", Event: "task:progress"})
	if sink.Len() != 1 {
		t.Fatalf("non-terminal via interface changed span count to %d, want 1", sink.Len())
	}
}
