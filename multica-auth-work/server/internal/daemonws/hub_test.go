package daemonws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

func TestNotifyTaskAvailable(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{RuntimeIDs: []string{"runtime-1"}})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Second)
	for hub.RuntimeConnectionCount("runtime-1") == 0 {
		if time.Now().After(deadline) {
			t.Fatal("runtime connection was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	hub.NotifyTaskAvailable("runtime-1", "task-1")

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var msg protocol.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if msg.Type != protocol.EventDaemonTaskAvailable {
		t.Fatalf("message type = %q, want %q", msg.Type, protocol.EventDaemonTaskAvailable)
	}

	var payload protocol.TaskAvailablePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.RuntimeID != "runtime-1" || payload.TaskID != "task-1" {
		t.Fatalf("payload = %+v, want runtime/task IDs", payload)
	}
}

func TestNotifyRuntimeProfilesChanged(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{
			WorkspaceID: "ws-1",
			RuntimeIDs:  []string{"runtime-1"},
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Second)
	for hub.WorkspaceConnectionCount("ws-1") == 0 {
		if time.Now().After(deadline) {
			t.Fatal("workspace connection was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	hub.NotifyRuntimeProfilesChanged("ws-1", "profile-1")

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var msg protocol.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if msg.Type != protocol.EventDaemonRuntimeProfilesChanged {
		t.Fatalf("message type = %q, want %q", msg.Type, protocol.EventDaemonRuntimeProfilesChanged)
	}

	var payload protocol.RuntimeProfilesChangedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.WorkspaceID != "ws-1" || payload.RuntimeProfileID != "profile-1" {
		t.Fatalf("payload = %+v, want workspace/profile IDs", payload)
	}
}

func TestNotifyRuntimeProfilesChangedIndexesAllAuthorizedWorkspaces(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{
			WorkspaceIDs: []string{"ws-1", "ws-2"},
			RuntimeIDs:   []string{"runtime-1", "runtime-2"},
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Second)
	for hub.WorkspaceConnectionCount("ws-1") == 0 || hub.WorkspaceConnectionCount("ws-2") == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("workspace connections not registered: ws-1=%d ws-2=%d",
				hub.WorkspaceConnectionCount("ws-1"),
				hub.WorkspaceConnectionCount("ws-2"))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := hub.WorkspaceConnectionCount("ws-3"); got != 0 {
		t.Fatalf("workspace ws-3 connection count = %d, want 0", got)
	}

	hub.NotifyRuntimeProfilesChanged("ws-1", "profile-1")
	hub.NotifyRuntimeProfilesChanged("ws-2", "profile-2")

	got := map[string]string{}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	for len(got) < 2 {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		var msg protocol.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal message: %v", err)
		}
		if msg.Type != protocol.EventDaemonRuntimeProfilesChanged {
			t.Fatalf("message type = %q, want %q", msg.Type, protocol.EventDaemonRuntimeProfilesChanged)
		}
		var payload protocol.RuntimeProfilesChangedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		got[payload.WorkspaceID] = payload.RuntimeProfileID
	}
	if got["ws-1"] != "profile-1" || got["ws-2"] != "profile-2" {
		t.Fatalf("profile refresh payloads = %+v, want ws-1/profile-1 and ws-2/profile-2", got)
	}
}

func TestRelayNotifierPublishesDaemonRuntimeScope(t *testing.T) {
	M.Reset()
	defer M.Reset()

	relay := &recordingRelayPublisher{}
	notifier := NewRelayNotifier(nil, relay)

	notifier.NotifyTaskAvailable("runtime-1", "task-1")

	if relay.scopeType != realtime.ScopeDaemonRuntime {
		t.Fatalf("scopeType = %q, want %q", relay.scopeType, realtime.ScopeDaemonRuntime)
	}
	if relay.scopeID != "task-1" {
		t.Fatalf("scopeID = %q, want task_id shard key", relay.scopeID)
	}
	if relay.eventID == "" {
		t.Fatal("expected event id")
	}
	if M.WakeupPublishedTotal.Load() != 1 {
		t.Fatalf("published metric = %d, want 1", M.WakeupPublishedTotal.Load())
	}

	var msg protocol.Message
	if err := json.Unmarshal(relay.frame, &msg); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	if msg.Type != protocol.EventDaemonTaskAvailable {
		t.Fatalf("message type = %q, want %q", msg.Type, protocol.EventDaemonTaskAvailable)
	}
	var payload protocol.TaskAvailablePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.RuntimeID != "runtime-1" || payload.TaskID != "task-1" {
		t.Fatalf("payload = %+v, want runtime/task IDs", payload)
	}
}

func TestRelayNotifierPublishesRuntimeProfilesChanged(t *testing.T) {
	M.Reset()
	defer M.Reset()

	relay := &recordingRelayPublisher{}
	notifier := NewRelayNotifier(nil, relay)

	notifier.NotifyRuntimeProfilesChanged("ws-1", "profile-1")

	if relay.scopeType != realtime.ScopeDaemonRuntime {
		t.Fatalf("scopeType = %q, want %q", relay.scopeType, realtime.ScopeDaemonRuntime)
	}
	if relay.scopeID != "ws-1" {
		t.Fatalf("scopeID = %q, want workspace shard key", relay.scopeID)
	}
	if relay.eventID == "" {
		t.Fatal("expected event id")
	}
	if M.WakeupPublishedTotal.Load() != 1 {
		t.Fatalf("published metric = %d, want 1", M.WakeupPublishedTotal.Load())
	}

	var msg protocol.Message
	if err := json.Unmarshal(relay.frame, &msg); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	if msg.Type != protocol.EventDaemonRuntimeProfilesChanged {
		t.Fatalf("message type = %q, want %q", msg.Type, protocol.EventDaemonRuntimeProfilesChanged)
	}
	var payload protocol.RuntimeProfilesChangedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.WorkspaceID != "ws-1" || payload.RuntimeProfileID != "profile-1" {
		t.Fatalf("payload = %+v, want workspace/profile IDs", payload)
	}
}

func TestRelayNotifierDedupsLocalRedisLoopback(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	client := attachDaemonTestClient(hub, "runtime-1")
	relay := &localFirstDaemonRelayPublisher{t: t, client: client}
	notifier := NewRelayNotifier(hub, relay)

	notifier.NotifyTaskAvailable("runtime-1", "task-1")

	if !relay.called {
		t.Fatal("expected relay publish to be invoked")
	}
	if relay.eventID == "" {
		t.Fatal("expected event id")
	}
	if M.WakeupDeliveredHit.Load() != 1 {
		t.Fatalf("delivered hit metric = %d, want 1", M.WakeupDeliveredHit.Load())
	}

	hub.DeliverDaemonRuntime(relay.scopeID, relay.frame, relay.eventID)

	select {
	case duplicate := <-client.send:
		t.Fatalf("expected redis loopback to be deduped, got duplicate %s", duplicate)
	case <-time.After(20 * time.Millisecond):
	}
	if M.WakeupDeliveredHit.Load() != 1 {
		t.Fatalf("delivered hit metric after loopback = %d, want 1", M.WakeupDeliveredHit.Load())
	}
	if M.WakeupDeliveredMiss.Load() != 0 {
		t.Fatalf("delivered miss metric after dedup = %d, want 0", M.WakeupDeliveredMiss.Load())
	}
}

func TestRelayNotifierDedupsRuntimeProfilesChangedLoopback(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	client := attachDaemonWorkspaceTestClient(hub, "ws-1")
	relay := &localFirstDaemonRelayPublisher{t: t, client: client}
	notifier := NewRelayNotifier(hub, relay)

	notifier.NotifyRuntimeProfilesChanged("ws-1", "profile-1")

	if !relay.called {
		t.Fatal("expected relay publish to be invoked")
	}
	if relay.eventID == "" {
		t.Fatal("expected event id")
	}
	if M.WakeupDeliveredHit.Load() != 0 {
		t.Fatalf("delivered hit metric = %d, want 0 before redis relay delivery", M.WakeupDeliveredHit.Load())
	}

	hub.DeliverDaemonRuntime(relay.scopeID, relay.frame, relay.eventID)

	select {
	case duplicate := <-client.send:
		t.Fatalf("expected redis loopback to be deduped, got duplicate %s", duplicate)
	case <-time.After(20 * time.Millisecond):
	}
	if M.WakeupDeliveredHit.Load() != 0 {
		t.Fatalf("delivered hit metric after loopback = %d, want 0", M.WakeupDeliveredHit.Load())
	}
	if M.WakeupDeliveredMiss.Load() != 0 {
		t.Fatalf("delivered miss metric after dedup = %d, want 0", M.WakeupDeliveredMiss.Load())
	}
}

// TestHeartbeatRoundTrip pins the WS heartbeat contract: a daemon:heartbeat
// frame invokes the registered HeartbeatHandler with the runtime ID, and the
// hub serializes the returned ack as a daemon:heartbeat_ack on the wire.
func TestHeartbeatRoundTrip(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	var calls atomic.Int32
	hub.SetHeartbeatHandler(func(_ context.Context, identity ClientIdentity, runtimeID string, _ bool) (*protocol.DaemonHeartbeatAckPayload, error) {
		calls.Add(1)
		if identity.WorkspaceID != "ws-1" {
			t.Errorf("identity workspace = %q, want ws-1", identity.WorkspaceID)
		}
		return &protocol.DaemonHeartbeatAckPayload{
			RuntimeID: runtimeID,
			Status:    "ok",
			PendingUpdate: &protocol.DaemonHeartbeatPendingUpdate{
				ID:            "update-1",
				TargetVersion: "0.1.99",
			},
		}, nil
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{
			WorkspaceID: "ws-1",
			RuntimeIDs:  []string{"runtime-1"},
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	hbFrame, err := json.Marshal(protocol.Message{
		Type:    protocol.EventDaemonHeartbeat,
		Payload: mustMarshalRaw(protocol.DaemonHeartbeatRequestPayload{RuntimeID: "runtime-1"}),
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, hbFrame); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var msg protocol.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal ack envelope: %v", err)
	}
	if msg.Type != protocol.EventDaemonHeartbeatAck {
		t.Fatalf("ack type = %q, want %q", msg.Type, protocol.EventDaemonHeartbeatAck)
	}
	var ack protocol.DaemonHeartbeatAckPayload
	if err := json.Unmarshal(msg.Payload, &ack); err != nil {
		t.Fatalf("unmarshal ack payload: %v", err)
	}
	if ack.RuntimeID != "runtime-1" {
		t.Fatalf("ack runtime_id = %q, want runtime-1", ack.RuntimeID)
	}
	if ack.PendingUpdate == nil || ack.PendingUpdate.ID != "update-1" {
		t.Fatalf("ack pending_update = %+v, want update-1", ack.PendingUpdate)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("HeartbeatHandler invocations = %d, want 1", got)
	}
}

// TestHeartbeatHandlerCtxNotTimeBounded pins the PopPending invariant: the
// hub must not wrap the handler ctx with a short WithTimeout, otherwise the
// Redis Lua claim script can be cancelled mid-flight after its side effects
// have already landed. We assert by stalling the handler past any timeout
// the hub might be tempted to add and verifying the ack still arrives.
func TestHeartbeatHandlerCtxNotTimeBounded(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	const stall = 250 * time.Millisecond
	hub.SetHeartbeatHandler(func(ctx context.Context, _ ClientIdentity, runtimeID string, _ bool) (*protocol.DaemonHeartbeatAckPayload, error) {
		select {
		case <-time.After(stall):
		case <-ctx.Done():
			t.Errorf("handler ctx was cancelled (deadline=%v) — PopPending invariant violated", ctx.Err())
			return nil, ctx.Err()
		}
		if _, ok := ctx.Deadline(); ok {
			t.Errorf("handler ctx must not carry a deadline; PopPending side effects cannot be safely un-run")
		}
		return &protocol.DaemonHeartbeatAckPayload{RuntimeID: runtimeID, Status: "ok"}, nil
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{RuntimeIDs: []string{"runtime-1"}})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	hbFrame, err := json.Marshal(protocol.Message{
		Type:    protocol.EventDaemonHeartbeat,
		Payload: mustMarshalRaw(protocol.DaemonHeartbeatRequestPayload{RuntimeID: "runtime-1"}),
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, hbFrame); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(stall + 2*time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	var msg protocol.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	if msg.Type != protocol.EventDaemonHeartbeatAck {
		t.Fatalf("ack type = %q, want %q", msg.Type, protocol.EventDaemonHeartbeatAck)
	}
}

// TestHeartbeatRejectsUnauthorizedRuntime verifies that a heartbeat for a
// runtime outside the connection's authenticated set is dropped silently —
// no handler call, no ack frame.
func TestHeartbeatRejectsUnauthorizedRuntime(t *testing.T) {
	M.Reset()
	defer M.Reset()

	hub := NewHub()
	var called atomic.Bool
	hub.SetHeartbeatHandler(func(context.Context, ClientIdentity, string, bool) (*protocol.DaemonHeartbeatAckPayload, error) {
		called.Store(true)
		return &protocol.DaemonHeartbeatAckPayload{Status: "ok"}, nil
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ClientIdentity{RuntimeIDs: []string{"runtime-1"}})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	hbFrame, err := json.Marshal(protocol.Message{
		Type:    protocol.EventDaemonHeartbeat,
		Payload: mustMarshalRaw(protocol.DaemonHeartbeatRequestPayload{RuntimeID: "runtime-other"}),
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, hbFrame); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatalf("expected no ack for unauthorized runtime, got message")
	}
	if called.Load() {
		t.Fatalf("HeartbeatHandler invoked for unauthorized runtime")
	}
}

func attachDaemonTestClient(hub *Hub, runtimeID string) *client {
	c := &client{
		send:     make(chan []byte, 2),
		runtimes: map[string]struct{}{runtimeID: {}},
	}

	hub.mu.Lock()
	hub.clients[c] = true
	hub.byRuntime[runtimeID] = map[*client]bool{c: true}
	hub.mu.Unlock()

	return c
}

func attachDaemonWorkspaceTestClient(hub *Hub, workspaceID string) *client {
	c := &client{
		send:     make(chan []byte, 2),
		identity: ClientIdentity{WorkspaceIDs: []string{workspaceID}},
		runtimes: map[string]struct{}{},
	}

	hub.mu.Lock()
	hub.clients[c] = true
	hub.byWorkspace[workspaceID] = map[*client]bool{c: true}
	hub.mu.Unlock()

	return c
}

type recordingRelayPublisher struct {
	scopeType string
	scopeID   string
	exclude   string
	frame     []byte
	eventID   string
}

func (r *recordingRelayPublisher) PublishWithID(scopeType, scopeID, exclude string, frame []byte, id string) error {
	r.scopeType = scopeType
	r.scopeID = scopeID
	r.exclude = exclude
	r.frame = append([]byte(nil), frame...)
	r.eventID = id
	return nil
}

type localFirstDaemonRelayPublisher struct {
	t      *testing.T
	client *client

	called     bool
	scopeType  string
	scopeID    string
	exclude    string
	frame      []byte
	eventID    string
	localFrame []byte
}

func (p *localFirstDaemonRelayPublisher) PublishWithID(scopeType, scopeID, exclude string, frame []byte, id string) error {
	p.called = true
	p.scopeType = scopeType
	p.scopeID = scopeID
	p.exclude = exclude
	p.frame = append([]byte(nil), frame...)
	p.eventID = id

	select {
	case p.localFrame = <-p.client.send:
	default:
		p.t.Fatal("expected local fanout to happen before relay publish")
	}
	return nil
}

// --- OBS-8 WS delivery hop wiring (F4) ---------------------------------------

func TestHub_EmitDelivery_OutcomesAreValidMetadataSpans(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(NewDeliveryRecorder(e2e.NewRecorder(sink)))

	h.emitDelivery("wssess-1", "delivered", "", 0, 0)
	h.emitDelivery("wssess-1", "dropped", "slow_consumer", 1, 1)
	h.emitDelivery("wssess-1", "backpressure", "send_buffer_full", 0, 1)

	spans := sink.Spans()
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(spans))
	}
	wantOutcome := []string{"delivered", "dropped", "backpressure"}
	seenDelivery := map[string]bool{}
	for i, s := range spans {
		if s.Hop != e2e.HopDelivery {
			t.Errorf("span %d hop=%q want %q", i, s.Hop, e2e.HopDelivery)
		}
		if s.Outcome != wantOutcome[i] {
			t.Errorf("span %d outcome=%q want %q", i, s.Outcome, wantOutcome[i])
		}
		if s.SecretsPresent {
			t.Errorf("span %d secrets_present must be false", i)
		}
		if s.Correlation.SessionID != "wssess-1" {
			t.Errorf("span %d session_id=%q want wssess-1", i, s.Correlation.SessionID)
		}
		if s.Correlation.DeliveryID == "" {
			t.Errorf("span %d missing delivery_id", i)
		}
		if seenDelivery[s.Correlation.DeliveryID] {
			t.Errorf("delivery_id %q reused; must be unique per attempt", s.Correlation.DeliveryID)
		}
		seenDelivery[s.Correlation.DeliveryID] = true
		// Metadata-only invariant: no labels/argv, span self-validates.
		if len(s.Labels) != 0 || len(s.ArgvShape) != 0 {
			t.Errorf("span %d must carry no labels/argv content", i)
		}
		if err := s.Validate(); err != nil {
			t.Errorf("span %d failed contract validation: %v", i, err)
		}
	}
}

func TestHub_NotifyFrame_EmitsDeliveredSpan(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(NewDeliveryRecorder(e2e.NewRecorder(sink)))

	c := &client{
		hub:       h,
		send:      make(chan []byte, 4),
		identity:  ClientIdentity{RuntimeIDs: []string{"rt-1"}, WorkspaceID: "ws-1"},
		runtimes:  map[string]struct{}{"rt-1": {}},
		sessionID: "wssess-int-1",
	}
	h.register(c)

	frame := []byte(`{"secret-looking":"payload-body"}`)
	delivered, _ := h.notifyFrame("rt-1", frame, "evt-1")
	if !delivered {
		t.Fatal("expected delivered=true")
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 delivered span, got %d", len(spans))
	}
	s := spans[0]
	if s.Outcome != "delivered" || s.Correlation.SessionID != "wssess-int-1" || s.Correlation.DeliveryID == "" {
		t.Fatalf("unexpected delivered span: %+v", s)
	}
	// The frame body must never leak into the metadata span.
	if len(s.Labels) != 0 {
		t.Fatalf("delivery span must not carry frame content in labels: %+v", s.Labels)
	}
}

func TestHub_HeartbeatAckBackpressure_EmitsSpan(t *testing.T) {
	sink := e2e.NewMemorySink()
	h := NewHub()
	h.SetDeliveryRecorder(NewDeliveryRecorder(e2e.NewRecorder(sink)))
	h.SetHeartbeatHandler(func(ctx context.Context, id ClientIdentity, runtimeID string, batch bool) (*protocol.DaemonHeartbeatAckPayload, error) {
		return &protocol.DaemonHeartbeatAckPayload{}, nil
	})

	c := &client{
		hub:       h,
		send:      make(chan []byte, 1),
		identity:  ClientIdentity{DaemonID: "d1", RuntimeIDs: []string{"rt-1"}},
		runtimes:  map[string]struct{}{"rt-1": {}},
		sessionID: "wssess-hb-1",
	}
	// Occupy the single send slot so the ack cannot be enqueued -> backpressure.
	c.send <- []byte("occupied")

	payload, err := json.Marshal(protocol.DaemonHeartbeatRequestPayload{RuntimeID: "rt-1"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	c.handleHeartbeatFrame(payload)

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 backpressure span, got %d", len(spans))
	}
	s := spans[0]
	if s.Outcome != "backpressure" || s.ReasonCode != "send_buffer_full" || s.Correlation.SessionID != "wssess-hb-1" {
		t.Fatalf("unexpected backpressure span: %+v", s)
	}
	if s.Counters["backpressure_count"] != 1 {
		t.Errorf("backpressure_count=%d want 1", s.Counters["backpressure_count"])
	}
}

func TestHub_NoDeliveryRecorder_NoEmitNoPanic(t *testing.T) {
	h := NewHub() // no delivery recorder set

	c := &client{
		hub:       h,
		send:      make(chan []byte, 1),
		identity:  ClientIdentity{RuntimeIDs: []string{"rt-1"}},
		runtimes:  map[string]struct{}{"rt-1": {}},
		sessionID: "wssess-none",
	}
	h.register(c)

	delivered, _ := h.notifyFrame("rt-1", []byte("{}"), "evt-none")
	if !delivered {
		t.Fatal("delivery behavior must be unchanged when no recorder is set")
	}
	// Direct call must be a safe no-op (no panic) with a nil recorder.
	h.emitDelivery("wssess-none", "delivered", "", 0, 0)
}
