package realtime

// obs_delivery.go — aggregate, metadata-only UI terminal-delivery observability
// for realtime.Hub (hop 7, e2e.HopDelivery). It records EXACTLY ONE aggregate
// HopDelivery span per task terminal event regardless of browser-tab count or
// Redis loopback. It never reads or persists prompt/response/tool content, frame
// bodies, or identity/email/headers — the caller (cmd/server listener) supplies
// only content-free terminal metadata + the frame bytes to fan out.

import (
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// Terminal task events this observer classifies. Anything else emits no span.
const (
	EventTaskCompleted = "task:completed"
	EventTaskFailed    = "task:failed"
	EventTaskCancelled = "task:cancelled"
)

func isTerminalDeliveryEvent(ev string) bool {
	switch ev {
	case EventTaskCompleted, EventTaskFailed, EventTaskCancelled:
		return true
	default:
		return false
	}
}

// TerminalDeliveryMeta is the content-free terminal-delivery metadata supplied
// by the caller. It carries NO payload/body/content/identity — only correlation
// identifiers and the terminal event kind.
type TerminalDeliveryMeta struct {
	TaskID        string
	ChatSessionID string
	Event         string // one of EventTask{Completed,Failed,Cancelled}
}

// deliveryDedupCapacity bounds the per-(session,event) dedup set so a long-lived
// hub cannot grow it without limit.
const deliveryDedupCapacity = 4096

// deliveryObserver owns the injectable recorder and the bounded dedup state that
// guarantees at-most-one aggregate HopDelivery span per (session_id, event).
type deliveryObserver struct {
	mu       sync.Mutex
	recorder *e2e.Recorder
	seen     map[string]struct{}
	seenList []string

	// now/newID are injectable for deterministic tests.
	now   func() time.Time
	newID func() string
}

func newDeliveryObserver() *deliveryObserver {
	return &deliveryObserver{
		seen:  make(map[string]struct{}, 64),
		now:   time.Now,
		newID: newDeliveryULID,
	}
}

// newDeliveryULID mints a genuine ULID for one terminal-delivery operation. The
// Crockford-base32 form is a safe correlation identifier for the e2e contract.
func newDeliveryULID() string { return ulid.Make().String() }

func (o *deliveryObserver) setRecorder(rec *e2e.Recorder) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.recorder = rec
	o.mu.Unlock()
}

// markSeenLocked records key with bounded FIFO eviction. Caller holds o.mu.
func (o *deliveryObserver) markSeenLocked(key string) {
	if _, ok := o.seen[key]; ok {
		return
	}
	o.seen[key] = struct{}{}
	o.seenList = append(o.seenList, key)
	if len(o.seenList) > deliveryDedupCapacity {
		drop := o.seenList[0]
		o.seenList = o.seenList[1:]
		delete(o.seen, drop)
	}
}

func nonNegMillis(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

// buildSpan constructs the finished aggregate HopDelivery span. It sets only the
// closed delivery-hop label/counter keys, so it is metadata-only by construction.
func (o *deliveryObserver) buildSpan(sessionID string, delivered, backpressure int, latencyMs int64, event string) *e2e.Span {
	outcome := "delivered"
	if delivered <= 0 {
		outcome = "dropped"
	}
	backpressureState := "clear"
	if backpressure > 0 {
		backpressureState = "backpressure"
	}
	// The single aggregate frame counts as at least one drop when it reached no
	// client (no subscribers, or all slow) — never dropped-with-count-0.
	dropCount := int64(backpressure)
	if delivered <= 0 && dropCount == 0 {
		dropCount = 1
	}
	reason := strings.TrimPrefix(event, "task:")
	return e2e.NewSpan(e2e.HopDelivery, e2e.Correlation{
		SessionID:  sessionID,
		DeliveryID: o.newID(),
	}).
		WithLabel("backpressure_state", backpressureState).
		WithCounter("delivery_latency_ms", nonNegMillis(latencyMs)).
		WithCounter("drop_count", dropCount).
		WithCounter("backpressure_count", int64(backpressure)).
		WithOutcome(outcome, reason).
		Finish()
}

// emitTerminalDelivery records the single aggregate HopDelivery span for one
// terminal task event. delivered = clients that enqueued the frame; backpressure
// = slow clients that could not. Outcome is "delivered" iff delivered >= 1, else
// "dropped". It is idempotent per (session_id, event): the span is emitted at
// most once, and the (session,event) key is marked seen ONLY after a successful
// recorder Emit — so a sink failure can be retried on a later call and never
// yields a false success. The whole check→emit→mark is serialized so concurrent
// calls for the same task still produce exactly one span. Returns true iff this
// call emitted the span.
func (o *deliveryObserver) emitTerminalDelivery(meta TerminalDeliveryMeta, delivered, backpressure int, latencyMs int64) bool {
	if o == nil || !isTerminalDeliveryEvent(meta.Event) {
		return false
	}
	taskID := strings.TrimSpace(meta.TaskID)
	if taskID == "" {
		return false
	}
	sessionID := e2e.CanonicalSessionID(meta.ChatSessionID, meta.TaskID)
	if sessionID == "" {
		return false
	}
	// Dedup on the RAW task id + terminal event so two DISTINCT tasks in the
	// same chat session (identical CanonicalSessionID) never collapse. The
	// emitted span still JOINS on CanonicalSessionID.
	dedupKey := taskID + "|" + meta.Event

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.recorder == nil {
		return false
	}
	if _, already := o.seen[dedupKey]; already {
		return false
	}
	span := o.buildSpan(sessionID, delivered, backpressure, latencyMs, meta.Event)
	if err := o.recorder.Emit(span); err != nil {
		// Not marked seen: a later call retries. Never a false success.
		return false
	}
	o.markSeenLocked(dedupKey)
	return true
}

// SetDeliveryRecorder installs the metadata-only e2e recorder used for aggregate
// HopDelivery spans. Nil disables emission. Safe to call before or after Run.
func (h *Hub) SetDeliveryRecorder(rec *e2e.Recorder) {
	if h == nil || h.deliveryObs == nil {
		return
	}
	h.deliveryObs.setRecorder(rec)
}

// deliverScopeCounting fans message out to every client subscribed to
// (scopeType, scopeID), returning the aggregate delivered/backpressure counts.
// Per-client Redis-loopback dedup uses eventID (empty disables it). Slow clients
// are evicted, mirroring BroadcastToScopeDedup.
func (h *Hub) deliverScopeCounting(scopeType, scopeID string, message []byte, eventID string) (delivered, backpressure int) {
	if scopeType == "" || scopeID == "" {
		return 0, 0
	}
	key := sk(scopeType, scopeID)

	h.mu.RLock()
	clients := h.rooms[key]
	var slow []*Client
	for client := range clients {
		if !client.markSeen(eventID) {
			continue
		}
		select {
		case client.send <- message:
			delivered++
		default:
			slow = append(slow, client)
		}
	}
	h.mu.RUnlock()

	backpressure = len(slow)
	if delivered > 0 {
		M.MessagesSentTotal.Add(int64(delivered))
	}
	if backpressure > 0 {
		h.evictSlow(slow)
	}
	return delivered, backpressure
}

// BroadcastTerminalDelivery is the typed seam the cmd/server listener calls for a
// terminal task frame. It fans the frame out to (scopeType, scopeID), aggregates
// the delivered/backpressure counts, and records EXACTLY ONE HopDelivery span for
// the (session, event) — regardless of tab count or Redis loopback. meta is
// caller-supplied and content-free, so the Hub never parses the frame payload. A
// non-terminal meta.Event is fanned out without recording a span.
func (h *Hub) BroadcastTerminalDelivery(scopeType, scopeID string, message []byte, meta TerminalDeliveryMeta, eventID string) {
	if h == nil {
		return
	}
	var start time.Time
	if h.deliveryObs != nil {
		start = h.deliveryObs.now()
	}
	delivered, backpressure := h.deliverScopeCounting(scopeType, scopeID, message, eventID)
	if h.deliveryObs == nil || !isTerminalDeliveryEvent(meta.Event) {
		return
	}
	latency := h.deliveryObs.now().Sub(start).Milliseconds()
	h.deliveryObs.emitTerminalDelivery(meta, delivered, backpressure, latency)
}

// TerminalBroadcaster is the OPTIONAL capability a broadcaster exposes for
// terminal task frames. The cmd/server listener type-asserts its configured
// broadcaster to this interface for terminal events and, if present, calls it so
// the aggregate HopDelivery span is recorded exactly once through the SAME path
// that fans the frame out; if absent it falls back to a normal broadcast (no
// span). This prevents naively calling the Hub-only BroadcastTerminalDelivery
// alongside a DualWrite/relay broadcaster, which would double-deliver local
// frames and/or bypass the Redis relay.
type TerminalBroadcaster interface {
	// BroadcastTerminalToWorkspace fans a terminal task frame to the workspace
	// scope and records exactly one aggregate HopDelivery span. meta is
	// content-free; implementations MUST NOT parse the frame payload.
	BroadcastTerminalToWorkspace(workspaceID string, message []byte, meta TerminalDeliveryMeta)
}

// compile-time guarantee that *Hub is a TerminalBroadcaster.
var _ TerminalBroadcaster = (*Hub)(nil)

// BroadcastTerminalToWorkspace makes *Hub a TerminalBroadcaster for single-node
// (no-relay) deployments: it fans the terminal frame to the workspace scope and
// records the single aggregate HopDelivery span locally. In a DualWrite/relay
// deployment the central DualWriteBroadcaster implements TerminalBroadcaster and
// owns the shared event ULID + relay.PublishWithID; it calls the local Hub leg
// via BroadcastTerminalDelivery(...) with that shared id. This Hub method is the
// local leg only and mints no relay event.
func (h *Hub) BroadcastTerminalToWorkspace(workspaceID string, message []byte, meta TerminalDeliveryMeta) {
	if h == nil {
		return
	}
	h.BroadcastTerminalDelivery(ScopeWorkspace, workspaceID, message, meta, "")
}
