package main

// listeners_terminal_test.go — real-call-site tests for the cmd/server terminal
// listener dispatch (registerListeners → realtime.TerminalBroadcaster seam). It
// uses a single realtime Hub + a DualWriteBroadcaster over a fake relay; no
// daemonws hub is involved (the listener only ever holds the realtime
// broadcaster). Real ungated assertions.
//
// package `main` tests run only under the DB-gated TestMain; the identical
// terminal dispatch behavior is also proven database-free at the DualWrite seam
// in internal/realtime/redis_relay_terminal_test.go.

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
)

// stubTerminalRelay implements realtime.RelayPublisher, counting publishes.
type stubTerminalRelay struct{ publishes int }

func (s *stubTerminalRelay) PublishWithID(scopeType, scopeID, exclude string, frame []byte, id string) error {
	s.publishes++
	return nil
}

func listenerDeliverySpanCount(sink *e2e.MemorySink) int {
	n := 0
	for _, s := range sink.Spans() {
		if s.Hop == e2e.HopDelivery {
			n++
		}
	}
	return n
}

// TestRegisterListenersTerminalDispatch proves the real listener routes terminal
// task events through the TerminalBroadcaster seam — exactly one aggregate
// HopDelivery span (canonical session) + one relay publish per terminal event
// across all three kinds — while a non-terminal event fans out with NO span, and
// two tasks in the same chat produce two spans.
func TestRegisterListenersTerminalDispatch(t *testing.T) {
	sink := e2e.NewMemorySink()
	hub := realtime.NewHub()
	hub.SetDeliveryRecorder(e2e.NewRecorder(sink))
	relay := &stubTerminalRelay{}
	dw := realtime.NewDualWriteBroadcaster(hub, relay)

	bus := events.New()
	registerListeners(bus, dw)

	// One terminal event → exactly one delivery span on the canonical session.
	bus.Publish(events.Event{Type: realtime.EventTaskCompleted, WorkspaceID: "ws-1", TaskID: "t1", ChatSessionID: "c1", ActorType: "system"})
	if got := listenerDeliverySpanCount(sink); got != 1 {
		t.Fatalf("HopDelivery spans after one terminal event = %d, want 1", got)
	}
	if want := e2e.CanonicalSessionID("c1", "t1"); sink.Spans()[0].Correlation.SessionID != want {
		t.Fatalf("delivery session_id = %q, want canonical %q", sink.Spans()[0].Correlation.SessionID, want)
	}
	if relay.publishes != 1 {
		t.Fatalf("relay publishes after one terminal event = %d, want 1 (no bypass)", relay.publishes)
	}

	// The other two terminal kinds each add exactly one span.
	bus.Publish(events.Event{Type: realtime.EventTaskFailed, WorkspaceID: "ws-1", TaskID: "t2", ChatSessionID: "c2", ActorType: "system"})
	bus.Publish(events.Event{Type: realtime.EventTaskCancelled, WorkspaceID: "ws-1", TaskID: "t3", ChatSessionID: "c3", ActorType: "system"})
	if got := listenerDeliverySpanCount(sink); got != 3 {
		t.Fatalf("HopDelivery spans after 3 terminal kinds = %d, want 3", got)
	}

	// Non-terminal event: normal workspace broadcast, no new delivery span.
	bus.Publish(events.Event{Type: "task:running", WorkspaceID: "ws-1", TaskID: "t1", ActorType: "system"})
	if got := listenerDeliverySpanCount(sink); got != 3 {
		t.Fatalf("HopDelivery spans after a non-terminal event = %d, want still 3", got)
	}

	// Same chat, two DISTINCT tasks → two spans (dedup is per task+event).
	bus.Publish(events.Event{Type: realtime.EventTaskCompleted, WorkspaceID: "ws-1", TaskID: "a1", ChatSessionID: "same", ActorType: "system"})
	bus.Publish(events.Event{Type: realtime.EventTaskCompleted, WorkspaceID: "ws-1", TaskID: "a2", ChatSessionID: "same", ActorType: "system"})
	if got := listenerDeliverySpanCount(sink); got != 5 {
		t.Fatalf("HopDelivery spans after same-chat two tasks = %d, want 5", got)
	}
}

// TestRegisterListenersTerminalWithoutTaskIDUsesNormalBroadcast proves a terminal
// event MISSING a task_id falls back to the normal workspace broadcast (no span),
// so the terminal seam is only taken when both task_id and workspace_id are set.
func TestRegisterListenersTerminalWithoutTaskIDUsesNormalBroadcast(t *testing.T) {
	sink := e2e.NewMemorySink()
	hub := realtime.NewHub()
	hub.SetDeliveryRecorder(e2e.NewRecorder(sink))
	relay := &stubTerminalRelay{}
	dw := realtime.NewDualWriteBroadcaster(hub, relay)

	bus := events.New()
	registerListeners(bus, dw)

	bus.Publish(events.Event{Type: realtime.EventTaskCompleted, WorkspaceID: "ws-1", TaskID: "", ChatSessionID: "c1", ActorType: "system"})
	if got := listenerDeliverySpanCount(sink); got != 0 {
		t.Fatalf("HopDelivery spans for terminal event without task_id = %d, want 0", got)
	}
	if relay.publishes != 1 {
		t.Fatalf("relay publishes = %d, want 1 (normal broadcast still fans out)", relay.publishes)
	}
}
