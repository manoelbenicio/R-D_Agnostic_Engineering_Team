package realtime

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// fakeTerminalRelay is a RelayPublisher stub that records every PublishWithID
// call so tests can assert the relay leg is exercised exactly once (no bypass)
// with the shared event id.
type fakeTerminalRelay struct {
	calls []fakeTerminalPublish
}

type fakeTerminalPublish struct {
	scopeType, scopeID, exclude, id string
	frame                           []byte
}

func (f *fakeTerminalRelay) PublishWithID(scopeType, scopeID, exclude string, frame []byte, id string) error {
	f.calls = append(f.calls, fakeTerminalPublish{scopeType, scopeID, exclude, id, append([]byte(nil), frame...)})
	return nil
}

func newTerminalHubWithSink() (*Hub, *e2e.MemorySink) {
	sink := e2e.NewMemorySink()
	hub := NewHub()
	hub.SetDeliveryRecorder(e2e.NewRecorder(sink))
	return hub, sink
}

func deliverySpans(sink *e2e.MemorySink) []e2e.Span {
	var out []e2e.Span
	for _, s := range sink.Spans() {
		if s.Hop == e2e.HopDelivery {
			out = append(out, s)
		}
	}
	return out
}

// TestDualWriteTerminalToWorkspace_OneSpanOneRelayCanonicalSession proves the
// DualWrite terminal seam: ONE shared event id drives the local Hub terminal leg
// (exactly one aggregate HopDelivery span on the canonical session) AND exactly
// one relay publish to the workspace scope with that same id — no local
// duplicate, no relay bypass.
func TestDualWriteTerminalToWorkspace_OneSpanOneRelayCanonicalSession(t *testing.T) {
	hub, sink := newTerminalHubWithSink()
	relay := &fakeTerminalRelay{}
	dw := NewDualWriteBroadcaster(hub, relay)

	dw.BroadcastTerminalToWorkspace("ws-1", []byte(`{"type":"task:completed"}`),
		TerminalDeliveryMeta{TaskID: "t1", ChatSessionID: "c1", Event: EventTaskCompleted})

	if len(relay.calls) != 1 {
		t.Fatalf("relay publishes = %d, want exactly 1 (no bypass)", len(relay.calls))
	}
	call := relay.calls[0]
	if call.scopeType != ScopeWorkspace || call.scopeID != "ws-1" || call.id == "" {
		t.Fatalf("relay publish mismatch: %+v", call)
	}

	spans := deliverySpans(sink)
	if len(spans) != 1 {
		t.Fatalf("HopDelivery spans = %d, want exactly 1", len(spans))
	}
	if want := e2e.CanonicalSessionID("c1", "t1"); spans[0].Correlation.SessionID != want {
		t.Fatalf("delivery session_id = %q, want canonical %q", spans[0].Correlation.SessionID, want)
	}
}

// TestDualWriteTerminalToWorkspace_AllThreeTerminalKinds proves each terminal
// event kind (completed/failed/cancelled) records exactly one span and one relay
// publish.
func TestDualWriteTerminalToWorkspace_AllThreeTerminalKinds(t *testing.T) {
	hub, sink := newTerminalHubWithSink()
	relay := &fakeTerminalRelay{}
	dw := NewDualWriteBroadcaster(hub, relay)

	kinds := []struct {
		task, event string
	}{
		{"tc", EventTaskCompleted},
		{"tf", EventTaskFailed},
		{"tx", EventTaskCancelled},
	}
	for _, k := range kinds {
		dw.BroadcastTerminalToWorkspace("ws-1", []byte(`{"type":"`+k.event+`"}`),
			TerminalDeliveryMeta{TaskID: k.task, ChatSessionID: "c-" + k.task, Event: k.event})
	}
	if got := len(deliverySpans(sink)); got != 3 {
		t.Fatalf("HopDelivery spans = %d, want 3 (one per terminal kind)", got)
	}
	if len(relay.calls) != 3 {
		t.Fatalf("relay publishes = %d, want 3", len(relay.calls))
	}
}

// TestDualWriteTerminalToWorkspace_NonTerminalFansOutWithoutSpan proves a
// non-terminal event is still fanned out (relay publish happens) but records NO
// delivery span.
func TestDualWriteTerminalToWorkspace_NonTerminalFansOutWithoutSpan(t *testing.T) {
	hub, sink := newTerminalHubWithSink()
	relay := &fakeTerminalRelay{}
	dw := NewDualWriteBroadcaster(hub, relay)

	dw.BroadcastTerminalToWorkspace("ws-1", []byte(`{"type":"task:running"}`),
		TerminalDeliveryMeta{TaskID: "t1", ChatSessionID: "c1", Event: "task:running"})

	if got := len(deliverySpans(sink)); got != 0 {
		t.Fatalf("HopDelivery spans = %d, want 0 for a non-terminal event", got)
	}
	if len(relay.calls) != 1 {
		t.Fatalf("relay publishes = %d, want 1 (non-terminal still fans out)", len(relay.calls))
	}
}

// TestDualWriteTerminalToWorkspace_SameChatTwoTasksTwoSpans proves two DISTINCT
// tasks in the SAME chat session (identical canonical session) each record their
// own delivery span (dedup is per task+event, not per session).
func TestDualWriteTerminalToWorkspace_SameChatTwoTasksTwoSpans(t *testing.T) {
	hub, sink := newTerminalHubWithSink()
	relay := &fakeTerminalRelay{}
	dw := NewDualWriteBroadcaster(hub, relay)

	dw.BroadcastTerminalToWorkspace("ws-1", []byte(`{"type":"task:completed"}`),
		TerminalDeliveryMeta{TaskID: "t1", ChatSessionID: "same-chat", Event: EventTaskCompleted})
	dw.BroadcastTerminalToWorkspace("ws-1", []byte(`{"type":"task:completed"}`),
		TerminalDeliveryMeta{TaskID: "t2", ChatSessionID: "same-chat", Event: EventTaskCompleted})

	if got := len(deliverySpans(sink)); got != 2 {
		t.Fatalf("HopDelivery spans = %d, want 2 (two tasks, same chat)", got)
	}
}
