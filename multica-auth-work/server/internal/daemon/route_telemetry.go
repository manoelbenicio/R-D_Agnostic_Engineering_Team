package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/daemon/observability/otlpreceiver"
)

// daemonOTLPLogsPort is the fixed loopback port the daemon's OTLP-logs receiver
// binds and injects (trusted-last) into the Claude child via
// OTEL_EXPORTER_OTLP_LOGS_ENDPOINT. Loopback-only, no auth, http/json.
const daemonOTLPLogsPort = 24318

// daemonOTLPLogsEndpoint is the exact endpoint injected into the child env.
const daemonOTLPLogsEndpoint = "http://127.0.0.1:24318/v1/logs"

// routeSeenCap bounds the per-task dedup set so a long-lived daemon cannot grow
// it unbounded; oldest entries are evicted FIFO.
const routeSeenCap = 8192

// maxRouteDurationMs rejects absurd durations before time math (overflow guard).
const maxRouteDurationMs = int64(24 * 60 * 60 * 1000)

// routeSpanSink bridges the closed-schema OTLP receiver's SanitizedRecord into a
// validated e2e HopRoute span. request_id is the trusted canonical request id
// (so route joins ingress); omni_request_id is the real Claude API request id.
// It emits EXACTLY ONE route span per task and counts additional api_request
// records. Emit errors are RETURNED (the receiver must not commit on a failed
// span persist), and a task is marked "seen" ONLY after a successful Emit so a
// receiver retry after an Emit failure re-emits exactly once (no acknowledged
// loss). The whole decision+emit+commit is serialized per sink.
type routeSpanSink struct {
	rec    *e2e.Recorder
	logger *slog.Logger
	mu     sync.Mutex
	seen   map[string]struct{}
	order  []string
}

func newRouteSpanSink(rec *e2e.Recorder, logger *slog.Logger) *routeSpanSink {
	return &routeSpanSink{rec: rec, logger: logger, seen: map[string]struct{}{}}
}

func (s *routeSpanSink) Record(r otlpreceiver.SanitizedRecord) error {
	if s == nil || s.rec == nil {
		return fmt.Errorf("route sink unavailable")
	}
	if r.TrustedRequestID == "" || r.RequestID == "" || r.TrustedTaskID == "" {
		return fmt.Errorf("route record missing required correlation")
	}
	// Official timing mandatory + overflow-guarded before any time conversion.
	if r.StartUnixNano == 0 || r.StartUnixNano > uint64(math.MaxInt64) {
		return fmt.Errorf("route record start time missing or out of range")
	}
	if r.DurationMs < 0 || r.DurationMs > maxRouteDurationMs {
		return fmt.Errorf("route record duration out of range")
	}

	// Serialize the dedup decision + emit + commit so a failed Emit leaves the
	// task UNSEEN (retry re-emits) and concurrent same-task events emit once.
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[r.TrustedTaskID]; ok {
		if s.logger != nil {
			s.logger.Debug("additional route api_request counted (one route hop per task)")
		}
		return nil // already persisted once; committed, no duplicate span
	}

	start := time.Unix(0, int64(r.StartUnixNano)).UTC()
	span := e2e.NewSpan(e2e.HopRoute, e2e.Correlation{
		RequestID:     r.TrustedRequestID,
		OmniRequestID: r.RequestID,
		TaskID:        r.TrustedTaskID,
	})
	span.StartedAt = start
	span.EndedAt = start.Add(time.Duration(r.DurationMs) * time.Millisecond)
	statusClass := "success"
	if r.Status != "" && r.Status != "success" && r.Status != "ok" && r.Status != "200" {
		statusClass = "error"
	}
	if r.Model != "" {
		span.WithLabel("route_model", r.Model)
	}
	span.WithLabel("status_class", statusClass)
	if statusClass == "success" {
		span.WithOutcome("completed", "ok")
	} else {
		span.WithOutcome("failed", "error")
	}
	span.WithCounter("latency_ms", r.DurationMs)

	if err := s.rec.Emit(span); err != nil {
		// Do NOT mark seen: a receiver retry must re-emit (no acknowledged loss).
		return err
	}
	// Commit dedup ONLY after a successful persist; bound the set FIFO.
	s.seen[r.TrustedTaskID] = struct{}{}
	s.order = append(s.order, r.TrustedTaskID)
	if len(s.order) > routeSeenCap {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.seen, oldest)
	}
	return nil
}

// startRouteTelemetryReceiver binds the fixed loopback OTLP-logs listener and
// serves it (see startRouteTelemetryReceiverOn).
func startRouteTelemetryReceiver(rec *e2e.Recorder, logger *slog.Logger) (*http.Server, error) {
	return startRouteTelemetryReceiverOn(rec, logger, "127.0.0.1:"+strconv.Itoa(daemonOTLPLogsPort))
}

// startRouteTelemetryReceiverOn binds addr SYNCHRONOUSLY (so a port conflict is
// reported immediately, before admission), then serves it on a background
// goroutine. Returns the server for graceful shutdown, or an error if the bind
// fails. Factored out so tests can bind an ephemeral loopback address.
func startRouteTelemetryReceiverOn(rec *e2e.Recorder, logger *slog.Logger, addr string) (*http.Server, error) {
	if rec == nil {
		return nil, fmt.Errorf("route telemetry recorder is nil")
	}
	receiver := otlpreceiver.New(newRouteSpanSink(rec, logger), otlpreceiver.Config{})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("otlp route receiver bind failed: %w", err)
	}
	// LoopbackServer's port only sets a cosmetic Addr; Serve(ln) uses the
	// already-bound listener.
	srv := receiver.LoopbackServer(daemonOTLPLogsPort)
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed && logger != nil {
			logger.Warn("otlp route receiver stopped", "error_class", "receiver_stopped")
		}
	}()
	return srv, nil
}

// shutdownRouteTelemetryReceiver gracefully stops the receiver server.
func shutdownRouteTelemetryReceiver(srv *http.Server) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
