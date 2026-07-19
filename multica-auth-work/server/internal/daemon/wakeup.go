package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

var errRuntimeSetChanged = errors.New("runtime set changed")

type taskWakeup struct {
	runtimeID string
}

func (d *Daemon) taskWakeupLoop(ctx context.Context, taskWakeups chan<- taskWakeup) {
	backoff := time.Second
	runtimeSetCh, unsub := d.runtimeSet.Subscribe()
	defer unsub()

	for {
		runtimeIDs := d.allRuntimeIDs()
		if len(runtimeIDs) == 0 {
			if err := sleepWithContextOrRuntimeChange(ctx, 5*time.Second, runtimeSetCh); err != nil {
				return
			}
			continue
		}

		err := d.runTaskWakeupConnection(ctx, runtimeIDs, taskWakeups, runtimeSetCh)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errRuntimeSetChanged) {
			backoff = time.Second
			continue
		}
		if err != nil {
			d.logger.Debug("task wakeup websocket unavailable; polling fallback remains active", "error", err, "retry_in", backoff)
		}

		if err := sleepWithContextOrRuntimeChange(ctx, jitterDuration(backoff), runtimeSetCh); err != nil {
			return
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func jitterDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	spread := d / 5
	if spread <= 0 {
		return d
	}
	delta := time.Duration(rand.Int63n(int64(spread)*2+1)) - spread
	return d + delta
}

func (d *Daemon) runTaskWakeupConnection(ctx context.Context, runtimeIDs []string, taskWakeups chan<- taskWakeup, runtimeSetCh <-chan struct{}) error {
	wsURL, err := taskWakeupURL(d.cfg.ServerBaseURL, runtimeIDs)
	if err != nil {
		return err
	}

	headers := http.Header{}
	if token := d.client.Token(); token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	if d.client.platform != "" {
		headers.Set("X-Client-Platform", d.client.platform)
	}
	if d.client.version != "" {
		headers.Set("X-Client-Version", d.client.version)
	}
	if d.client.os != "" {
		headers.Set("X-Client-OS", d.client.os)
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()
	// HTTP heartbeats resume the moment WS detaches so the freshness window
	// from a previous connection cannot keep them silenced past disconnect.
	defer d.clearWSHeartbeatAcks()

	d.logger.Info("task wakeup websocket connected", "runtimes", len(runtimeIDs))
	signalTaskWakeup(taskWakeups, "")

	// Serialize all writes through a single channel: the gorilla/websocket
	// Conn does not allow concurrent WriteMessage calls, and the heartbeat
	// sender now coexists with future server-initiated writes. The buffer
	// is sized to fit a full per-runtime heartbeat batch plus headroom; a
	// fixed 8-slot queue would silently drop heartbeats once a daemon
	// watched more than ~8 runtimes (typical when one machine connects to
	// several workspaces), even when the network was healthy.
	writeBufSize := 16
	if 2*len(runtimeIDs) > writeBufSize {
		writeBufSize = 2 * len(runtimeIDs)
	}
	writes := make(chan []byte, writeBufSize)
	writerDone := make(chan struct{})
	go d.runWSWriter(conn, writes, writerDone)

	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		d.runWSHeartbeatSender(heartbeatCtx, runtimeIDs, writes)
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.readTaskWakeupMessages(conn, taskWakeups)
	}()

	// Defer cleanup must shut goroutines down in this order:
	//   1. cancel the heartbeat sender's ctx
	//   2. wait for the sender to actually return — only then is it safe
	//      to close the writes channel without a "send on closed channel"
	//      panic from sendWSHeartbeats
	//   3. close writes; the writer drains and exits
	//   4. wait for the writer to finish so it doesn't outlive the conn
	//
	// LIFO defer order would close writes before the sender stops, so the
	// teardown is folded into a single deferred function instead.
	defer func() {
		cancelHeartbeat()
		<-hbDone
		close(writes)
		<-writerDone
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-runtimeSetCh:
		return errRuntimeSetChanged
	case err := <-errCh:
		return err
	}
}

// runWSWriter funnels writes from the heartbeat sender (and any future
// daemon-initiated message) into a single goroutine. gorilla/websocket
// requires that all WriteMessage calls happen from the same goroutine.
func (d *Daemon) runWSWriter(conn *websocket.Conn, writes <-chan []byte, done chan<- struct{}) {
	defer close(done)
	for frame := range writes {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
			d.logger.Debug("task wakeup websocket write failed", "error", err)
			conn.Close()
			// Drain remaining frames so the producers don't block forever
			// while waiting for runTaskWakeupConnection to close the channel.
			for range writes {
			}
			return
		}
	}
}

// runWSHeartbeatSender emits a daemon:heartbeat per runtime every
// HeartbeatInterval. The first batch fires immediately so the server learns
// the connection identity without waiting a full interval. Frames are queued
// to the writer; if the queue is full the heartbeat is dropped (the
// freshness window is short enough that one missed beat just means HTTP will
// pick it up next tick).
func (d *Daemon) runWSHeartbeatSender(ctx context.Context, runtimeIDs []string, writes chan<- []byte) {
	d.sendWSHeartbeats(ctx, runtimeIDs, writes)
	interval := d.cfg.HeartbeatInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.sendWSHeartbeats(ctx, runtimeIDs, writes)
		}
	}
}

func (d *Daemon) sendWSHeartbeats(ctx context.Context, runtimeIDs []string, writes chan<- []byte) {
	for _, rid := range runtimeIDs {
		if ctx.Err() != nil {
			return
		}
		frame, err := json.Marshal(protocol.Message{
			Type:    protocol.EventDaemonHeartbeat,
			Payload: marshalRaw(protocol.DaemonHeartbeatRequestPayload{RuntimeID: rid, SupportsBatchImport: true}),
		})
		if err != nil {
			d.logger.Debug("ws heartbeat marshal failed", "error", err, "runtime_id", rid)
			continue
		}
		select {
		case writes <- frame:
		case <-ctx.Done():
			return
		default:
			// Writer is backed up; drop this beat. HTTP heartbeat will resume
			// on its next tick once the freshness window expires.
			d.logger.Debug("ws heartbeat dropped: writer backlog", "runtime_id", rid)
		}
	}
}

func marshalRaw(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}

// handleWSHeartbeatAck dispatches one heartbeat_ack received over the WS
// task-wakeup connection. Extracted from readTaskWakeupMessages so tests can
// exercise the branching logic without a real WebSocket.
//
// A RuntimeGone=true ack is the WebSocket twin of an HTTP 404 "runtime not
// found": it tells the daemon the runtime row was deleted server-side. We
// route it through the same self-heal entry point as the HTTP path and do
// NOT record a heartbeat freshness mark — pretending the runtime is alive
// would let HTTP keep skipping its own heartbeat against the dead UUID.
//
// handleRuntimeGone uses the daemon root context for its register call, so
// this function can safely pass any caller context here.
func (d *Daemon) handleWSHeartbeatAck(ctx context.Context, ack *HeartbeatResponse) {
	if ack == nil || ack.RuntimeID == "" {
		return
	}
	if ack.RuntimeGone {
		go d.handleRuntimeGone(ack.RuntimeID)
		return
	}
	d.recordWSHeartbeatAck(ack.RuntimeID)
	d.handleHeartbeatActions(ctx, ack.RuntimeID, ack)
}

func (d *Daemon) readTaskWakeupMessages(conn *websocket.Conn, taskWakeups chan<- taskWakeup) error {
	conn.SetReadLimit(64 * 1024)
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg protocol.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			d.logger.Debug("task wakeup websocket invalid message", "error", err)
			continue
		}
		if msg.Type == eventDaemonCredentialSessionDiscovery {
			// Reassignment may include bounded vendor authentication, so never
			// block heartbeat/task-wakeup reads on it. Copy the payload because
			// the WebSocket frame buffer is reused by the read loop.
			payload := append(json.RawMessage(nil), msg.Payload...)
			go func() {
				ctx, cancel := context.WithTimeout(d.recoveryContext(), 2*time.Minute)
				defer cancel()
				d.dispatchAndReportCredentialSessionDiscoveryEvent(ctx, protocol.Message{
					Type:    eventDaemonCredentialSessionDiscovery,
					Payload: payload,
				}, time.Now())
			}()
			continue
		}
		switch msg.Type {
		case protocol.EventDaemonTaskAvailable:
			var payload protocol.TaskAvailablePayload
			if len(msg.Payload) > 0 {
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					d.logger.Debug("task wakeup websocket invalid payload", "error", err)
					continue
				}
			}
			if payload.RuntimeID != "" {
				d.logger.Debug("task wakeup received", "runtime_id", payload.RuntimeID, "task_id", payload.TaskID)
			}
			signalTaskWakeup(taskWakeups, payload.RuntimeID)
		case protocol.EventDaemonRuntimeProfilesChanged:
			var payload protocol.RuntimeProfilesChangedPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				d.logger.Debug("runtime profile refresh websocket invalid payload", "error", err)
				continue
			}
			if payload.WorkspaceID == "" {
				d.logger.Debug("runtime profile refresh websocket missing workspace_id")
				continue
			}
			go d.handleRuntimeProfilesChanged(payload)
		case protocol.EventDaemonHeartbeatAck:
			var ack HeartbeatResponse
			if err := json.Unmarshal(msg.Payload, &ack); err != nil {
				d.logger.Debug("ws heartbeat ack invalid payload", "error", err)
				continue
			}
			d.handleWSHeartbeatAck(context.Background(), &ack)
		}
	}
}

// dispatchAndReportCredentialSessionDiscoveryEvent makes the asynchronous
// reassignment result visible to operators. A successful service call has
// already persisted its rotation event before this method logs completion.
// Only assignment metadata is logged; credential homes, config paths, session
// identifiers, error strings, and credential material are deliberately absent.
func (d *Daemon) dispatchAndReportCredentialSessionDiscoveryEvent(ctx context.Context, message protocol.Message, now time.Time) {
	outcome, err := d.dispatchCredentialSessionDiscoveryEventWithOutcome(ctx, message, now)
	if !outcome.Handled || d == nil || d.logger == nil {
		return
	}

	attrs := []any{
		"agent_id", outcome.AgentID,
		"provider", outcome.Provider,
		"tenant_id", outcome.TenantID,
		"previous_account_id", outcome.PreviousAccountID,
	}
	if err != nil {
		class := credentialSessionReassignmentErrorClass(err)
		attrs = append(attrs, "alert", class)
		if errors.Is(err, rotation.ErrNoAccountAvailable) {
			d.logger.Warn("rotation: automatic credential account reassignment unavailable", attrs...)
			return
		}
		d.logger.Error("rotation: automatic credential account reassignment failed", attrs...)
		return
	}
	if !outcome.Reassigned {
		d.logger.Debug("rotation: credential session discovery produced no reassignment", attrs...)
		return
	}

	attrs = append(attrs,
		"next_account_id", outcome.NextAccountID,
		"reason", string(rotation.ReasonQuotaReactive),
	)
	d.logger.Warn("rotation: automatic credential account reassignment completed", attrs...)
}

func credentialSessionReassignmentErrorClass(err error) string {
	switch {
	case errors.Is(err, rotation.ErrNoAccountAvailable):
		return "no_account_available"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, errDiscoveryReassignmentUnavailable):
		return "service_unavailable"
	default:
		return "reassignment_failed"
	}
}

func (d *Daemon) handleRuntimeProfilesChanged(payload protocol.RuntimeProfilesChangedPayload) {
	if payload.WorkspaceID == "" {
		return
	}
	if err := d.refreshWorkspaceRuntimeProfiles(d.recoveryContext(), payload.WorkspaceID); err != nil {
		d.logger.Debug("runtime profile refresh websocket hint failed",
			"workspace_id", payload.WorkspaceID,
			"runtime_profile_id", payload.RuntimeProfileID,
			"error", err)
	}
}

func signalTaskWakeup(taskWakeups chan<- taskWakeup, runtimeID string) {
	select {
	case taskWakeups <- taskWakeup{runtimeID: runtimeID}:
	default:
	}
}

func taskWakeupURL(baseURL string, runtimeIDs []string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid daemon server URL: %w", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("daemon server URL must use http, https, ws, or wss")
	}

	u.Path = strings.TrimRight(u.Path, "/") + "/api/daemon/ws"
	u.RawPath = ""
	q := u.Query()
	ids := append([]string(nil), runtimeIDs...)
	sort.Strings(ids)
	q.Set("runtime_ids", strings.Join(ids, ","))
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String(), nil
}

func sleepWithContextOrRuntimeChange(ctx context.Context, d time.Duration, runtimeSetCh <-chan struct{}) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-runtimeSetCh:
		return nil
	case <-timer.C:
		return nil
	}
}
