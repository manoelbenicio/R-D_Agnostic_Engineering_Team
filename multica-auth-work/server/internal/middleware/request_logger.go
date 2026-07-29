package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	chi "github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// webhookTriggerIDKeyType is unexported so foreign packages cannot collide on
// the context key — they go through SetWebhookTriggerID instead.
type webhookTriggerIDKeyType struct{}

var webhookTriggerIDKey = webhookTriggerIDKeyType{}

// SetWebhookTriggerID stashes the resolved trigger ID on the request context
// so the request logger can include it in the audit line without revealing
// the bearer token in the URL path. Called by the webhook handler right after
// the trigger row is looked up.
//
// Mutates `*r` in place so the wrapping middleware (which is still holding the
// original `*http.Request`) reads the value back out of context after
// ServeHTTP returns. Reassigning a local `r` variable would not propagate the
// new context back up to the caller, which is the trap a previous version of
// this helper fell into.
func SetWebhookTriggerID(r *http.Request, triggerID string) {
	if triggerID == "" {
		return
	}
	*r = *r.WithContext(context.WithValue(r.Context(), webhookTriggerIDKey, triggerID))
}

// webhookTriggerIDFromContext returns the trigger ID stashed by
// SetWebhookTriggerID, or "" when none was set.
func webhookTriggerIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(webhookTriggerIDKey).(string)
	return v
}

// webhookIngressPathPrefix is the public webhook ingress path. The path
// segment after this prefix IS a bearer credential, so the logger must
// redact it — see redactWebhookPath.
const webhookIngressPathPrefix = "/api/webhooks/autopilots/"

// redactWebhookPath returns a logger-safe version of a request path. For
// the autopilot webhook ingress path the trailing token segment is replaced
// with "[redacted]"; every other path passes through untouched.
//
// Why this exists: r.URL.Path for a successful webhook delivery is
// "/api/webhooks/autopilots/awt_<32-byte-base64>", and the token is the
// only credential gating the route. Without redaction, every successful
// delivery prints a replayable URL into the structured log stream.
func redactWebhookPath(path string) string {
	if !strings.HasPrefix(path, webhookIngressPathPrefix) {
		return path
	}
	rest := path[len(webhookIngressPathPrefix):]
	if rest == "" {
		return path
	}
	// Preserve any sub-path after the token (currently none, but defensive).
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		return webhookIngressPathPrefix + "[redacted]" + rest[slash:]
	}
	return webhookIngressPathPrefix + "[redacted]"
}

// boundedBuffer captures up to Cap bytes from a stream then silently drops the
// rest. Used by RequestLogger so a large response body cannot blow up logger
// memory while we mirror just enough bytes to classify the response.
type boundedBuffer struct {
	buf bytes.Buffer
	cap int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remain := b.cap - b.buf.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		b.buf.Write(p[:remain])
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *boundedBuffer) Bytes() []byte { return b.buf.Bytes() }

// softNotFoundBodyCaptureLimit is the maximum number of body bytes the
// request logger inspects to decide whether a 404 is an expected stale-state
// signal (runtime/task deleted server-side). The JSON error envelope is small
// — 256 bytes is enough to see the "error" field — and the cap means an
// unbounded handler body cannot blow up logger memory.
const softNotFoundBodyCaptureLimit = 256

// softNotFoundMarkers are 404 response bodies the daemon emits routinely as
// part of normal lifecycle events: a runtime deleted from the UI, a task GC'd
// after an issue was removed, etc. Logging these at Warn turned production
// stderr into a flood whenever a runtime was deleted (see issue #2391). They
// stay machine-recognizable at Info, while genuine 4xx (wrong path, bad
// auth, real bugs) keep Warn.
var softNotFoundMarkers = []string{
	"runtime not found",
	"task not found",
}

// RequestLogger is a structured HTTP request logger using slog.
// It replaces Chi's built-in chimw.Logger with colored, structured output.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip the hot liveness endpoint to keep logs readable.
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		// Capture a small body prefix so 404s can be classified by content.
		// chimw.WrapResponseWriter exposes Tee for exactly this — the body
		// keeps flowing to the client; we mirror up to N bytes for inspection.
		bodyPrefix := &boundedBuffer{cap: softNotFoundBodyCaptureLimit}
		ww.Tee(bodyPrefix)

		// Attach a mutable ingress holder so a downstream accepted-task
		// transition (service) can record the task id that this logger reads
		// back after the handler to emit the single hop-1 ingress span.
		r = r.WithContext(withIngressHolder(r.Context()))

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		status := ww.Status()
		requestID := chimw.GetReqID(r.Context())

		// OBS-2 (hop-1) ingress span: exactly one emission per request, with
		// real HTTP method/route-template/status/latency; request_id is the
		// canonical value derived from the task id (joins the route hop),
		// fail-closed when no task id was resolved.
		emitIngressSpan(r, status, duration)

		attrs := []any{
			"method", r.Method,
			"path", redactWebhookPath(r.URL.Path),
			"status", status,
			"duration", duration.Round(time.Microsecond).String(),
		}
		if requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}
		if uid := r.Header.Get("X-User-ID"); uid != "" {
			attrs = append(attrs, "user_id", uid)
		}
		if tid := webhookTriggerIDFromContext(r.Context()); tid != "" {
			attrs = append(attrs, "webhook_trigger_id", tid)
		}
		if platform, version, os := ClientMetadataFromContext(r.Context()); platform != "" || version != "" || os != "" {
			if platform != "" {
				attrs = append(attrs, "client_platform", platform)
			}
			if version != "" {
				attrs = append(attrs, "client_version", version)
			}
			if os != "" {
				attrs = append(attrs, "client_os", os)
			}
		}

		switch {
		case status >= 500:
			slog.Error("http request", attrs...)
		case status == http.StatusNotFound && isSoftNotFound(bodyPrefix.Bytes()):
			// Lifecycle 404 — runtime/task was deleted server-side. The daemon
			// catches this exact body and triggers its own self-heal, so it is
			// neither noise nor a bug; logging at Info keeps the signal in
			// structured logs without flooding the warn channel.
			slog.Info("http request", attrs...)
		case status >= 400:
			slog.Warn("http request", attrs...)
		default:
			slog.Info("http request", attrs...)
		}
	})
}

// isSoftNotFound reports whether the captured response body matches one of
// the expected stale-state 404 signals listed in softNotFoundMarkers.
func isSoftNotFound(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	lower := strings.ToLower(string(body))
	for _, marker := range softNotFoundMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// --- OBS-2 (hop-1) ingress correlation wiring (metadata-only) ---
//
// EmitIngress/IngressObservation live in obs_ingress.go (same package) over the
// FROZEN L5 contract `agent-brain.e2e.v1`. RequestLogger is the single ingress
// emission point; internal/metrics/http.go intentionally does NOT emit, to
// avoid a duplicate hop-1 span per request. The span carries ONLY
// request_id/task_id plus a bounded method/route-template/principal-class
// classification, an HTTP status class, and a latency counter — never a body,
// header, query string, or raw path.

// ingressRecorder is the process-wide sink for ingress spans. It is nil until
// startup wiring calls SetIngressRecorder, so instrumentation is a no-op and
// never changes request behavior until an owner provisions the recorder.
var ingressRecorder *e2e.Recorder

// SetIngressRecorder installs the ingress span recorder once at startup (router
// construction, before serving). A nil recorder disables emission. Not safe to
// call concurrently with live requests.
func SetIngressRecorder(rec *e2e.Recorder) { ingressRecorder = rec }

type ingressTaskIDKeyType struct{}
type ingressPrincipalClassKeyType struct{}

var (
	ingressTaskIDKey         = ingressTaskIDKeyType{}
	ingressPrincipalClassKey = ingressPrincipalClassKeyType{}
)

// maxIngressTasksPerRequest bounds how many distinct accepted-task ingress
// spans one HTTP request may emit. Beyond it we fail closed (drop + flag) rather
// than grow unbounded on a pathological request.
const maxIngressTasksPerRequest = 64

// ingressHolder is a mutable per-request carrier for the accepted task ids. A
// single HTTP request can enqueue MULTIPLE tasks (agent/mention/squad fan-out),
// so it holds a bounded, deduplicated, ORDERED set — never a single scalar that
// a later enqueue would overwrite (which would orphan all but the last). The
// RequestLogger attaches it BEFORE next.ServeHTTP; downstream layers add each
// accepted task id; the logger emits one ingress span per task afterwards.
type ingressHolder struct {
	mu      sync.Mutex
	ids     []string
	seen    map[string]struct{}
	dropped bool
}

func (h *ingressHolder) add(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seen == nil {
		h.seen = make(map[string]struct{})
	}
	if _, dup := h.seen[id]; dup {
		return // duplicate Notify for the same task must not duplicate the span
	}
	if len(h.ids) >= maxIngressTasksPerRequest {
		h.dropped = true // fail closed beyond the bound
		return
	}
	h.seen[id] = struct{}{}
	h.ids = append(h.ids, id)
}

func (h *ingressHolder) snapshot() (ids []string, overflowed bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.ids...), h.dropped
}

// withIngressHolder attaches a fresh mutable ingress holder to ctx.
func withIngressHolder(ctx context.Context) context.Context {
	return context.WithValue(ctx, ingressTaskIDKey, &ingressHolder{})
}

func ingressHolderFrom(ctx context.Context) *ingressHolder {
	h, _ := ctx.Value(ingressTaskIDKey).(*ingressHolder)
	return h
}

// SetIngressTaskID records an accepted raw task id into the request's ingress
// holder (deduplicated). Context-based and safe to call from any downstream
// layer (handlers, services) as each task is accepted — e.g. from the
// accepted-task transition. No-op if no holder is present or id is empty.
func SetIngressTaskID(ctx context.Context, taskID string) {
	if taskID == "" {
		return
	}
	if h := ingressHolderFrom(ctx); h != nil {
		h.add(taskID)
	}
}

// SetIngressPrincipalClass stashes a bounded principal CLASS (e.g. "user",
// "service", "webhook", "anonymous") — never a raw principal identity or email.
func SetIngressPrincipalClass(r *http.Request, class string) {
	if class == "" {
		return
	}
	*r = *r.WithContext(context.WithValue(r.Context(), ingressPrincipalClassKey, class))
}

func ingressPrincipalClassFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ingressPrincipalClassKey).(string)
	return v
}

// ingressRouteTemplate returns the bounded chi route TEMPLATE (e.g. "/v1/tasks"
// or "/v1/tasks/{id}") for the request, never the raw r.URL.Path. An
// unmatched/empty/catch-all pattern yields "" so the caller omits the
// route_template label rather than emit a raw path.
func ingressRouteTemplate(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" && pattern != "/*" {
			return pattern
		}
	}
	return ""
}

// ingressOutcome maps an HTTP status to a bounded safe outcome code.
func ingressOutcome(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "accepted"
	case status >= 300 && status < 400:
		return "redirect"
	case status >= 400 && status < 500:
		return "client_error"
	case status >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}

// emitIngressSpan emits the hop-1 ingress span iff a recorder is installed and
// BOTH required correlation IDs (request_id, task_id) are present. Missing IDs
// fail closed: the span is simply not emitted. The recorder re-validates
// fail-closed; its error is intentionally dropped so instrumentation never
// crashes the request path or logs a value.
func emitIngressSpan(r *http.Request, status int, d time.Duration) {
	rec := ingressRecorder
	if rec == nil {
		return
	}
	h := ingressHolderFrom(r.Context())
	if h == nil {
		return
	}
	ids, overflowed := h.snapshot()
	if overflowed {
		slog.Warn("ingress accepted-task set exceeded per-request bound; excess not spanned", "error_class", "ingress_overflow")
	}
	if len(ids) == 0 {
		return
	}
	latency := d.Milliseconds()
	if latency < 0 {
		latency = 0
	}
	method := r.Method
	route := ingressRouteTemplate(r)
	principal := ingressPrincipalClassFromContext(r.Context())
	outcome := ingressOutcome(status)
	// Exactly one ingress span PER accepted task, each with that task's canonical
	// request id (joins the route hop) and the SAME real HTTP metadata.
	for _, taskID := range ids {
		requestID := e2e.CanonicalRequestID(taskID)
		if requestID == "" {
			continue
		}
		_ = EmitIngress(rec, IngressObservation{
			RequestID:      requestID,
			TaskID:         taskID,
			Method:         method,
			RouteTemplate:  route,
			PrincipalClass: principal,
			HTTPStatus:     status,
			Outcome:        outcome,
			LatencyMs:      latency,
		})
	}
}
