// Package otlpreceiver implements a loopback-only (127.0.0.1) OTLP logs
// http/json receiver for Claude Code `api_request` telemetry. It persists ONLY
// an explicit allowlist of metadata fields and drops all identity/content/
// prompt/response/tool data.
//
// Official live shape (per Anthropic monitoring docs):
//   - Records are identified by DOCUMENTED signals only: the resource attribute
//     service.name == "claude-code" AND the log attribute event.name ==
//     TargetEventName ("api_request"). The instrumentation SCOPE name is
//     undocumented/untrusted free-form: it is IGNORED entirely — it neither
//     gates target selection nor is persisted.
//   - The per-task correlation lives in resourceLogs.resource.attributes, NOT in
//     the log record. Only the TRUSTED resource keys agent_brain.task_id and
//     agent_brain.request_id are read and merged into each target record; log
//     record attributes can never override the trusted correlation.
//
// Security posture (all fail-closed):
//   - Loopback only (403 otherwise; LoopbackServer binds 127.0.0.1).
//   - Default-deny: the log body is never modeled; only allowlisted resource +
//     log attributes are copied. Identity/prompt/response/tool/content dropped.
//   - A target record missing its log request_id OR the trusted task/request
//     correlation is DROPPED and counted (not persisted).
//   - Idempotency: records are de-duplicated by request_id/client_request_id so
//     OTLP client retries after a sink error do not double-persist; duplicates
//     are counted. A record whose sink write fails is NOT marked seen, so a
//     later retry re-persists it.
//   - Bounded body/records/attribute values + server timeouts; honors request
//     context cancellation. No logging (never reads/echoes Authorization).
package otlpreceiver

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TargetEventName is the only accepted event.name attribute VALUE.
const TargetEventName = "api_request"

// ServiceNameClaudeCode is the DOCUMENTED resource service.name for Claude Code
// telemetry (Anthropic monitoring docs). Together with event.name ==
// TargetEventName it identifies target records. The instrumentation scope name
// is undocumented and is never used to accept/reject events.
const ServiceNameClaudeCode = "claude-code"

// Resource attribute keys (resourceLogs.resource.attributes).
const (
	resAttrServiceName = "service.name" // documented identity gate (not merged into records)
	resAttrTaskID      = "agent_brain.task_id"
	resAttrRequestID   = "agent_brain.request_id"
)

// Allowlisted LOG-record attribute keys.
const (
	attrEventName       = "event.name"
	attrRequestID       = "request_id"
	attrClientRequestID = "client_request_id"
	attrModel           = "model"
	attrStatus          = "status"
	attrDurationMs      = "duration_ms"
)

// AllowlistedResourceAttributes / AllowlistedLogAttributes expose the closed
// key sets for audit/tests.
func AllowlistedResourceAttributes() []string {
	return []string{resAttrTaskID, resAttrRequestID}
}
func AllowlistedLogAttributes() []string {
	return []string{attrEventName, attrRequestID, attrClientRequestID, attrModel, attrStatus, attrDurationMs}
}

// SanitizedRecord is the ONLY persisted shape. It has no field capable of
// holding prompt/response/tool/identity/content by construction.
type SanitizedRecord struct {
	EventName        string // always TargetEventName
	RequestID        string // log attribute request_id (Claude API request id)
	ClientRequestID  string // log attribute client_request_id
	Model            string
	Status           string
	DurationMs       int64
	StartUnixNano    uint64
	TrustedTaskID    string // resource agent_brain.task_id (trusted correlation)
	TrustedRequestID string // resource agent_brain.request_id (trusted correlation)
}

// Sink persists sanitized records.
type Sink interface {
	Record(SanitizedRecord) error
}

// Stats are cumulative counters for observability of the receiver itself.
type Stats struct {
	Targets           int64 // records matching service.name==claude-code + event.name==api_request
	Persisted         int64
	DroppedIncomplete int64 // target records missing request_id or trusted correlation
	DuplicatesDropped int64 // idempotency hits (retries)
}

// Config bounds the receiver. Zero values select safe defaults.
type Config struct {
	MaxBodyBytes int64 // default 256 KiB
	MaxRecords   int   // default 1000
	MaxValueLen  int   // default 512
	MaxDedupKeys int   // default 65536
}

func (c Config) withDefaults() Config {
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 256 << 10
	}
	if c.MaxRecords <= 0 {
		c.MaxRecords = 1000
	}
	if c.MaxValueLen <= 0 {
		c.MaxValueLen = 512
	}
	if c.MaxDedupKeys <= 0 {
		c.MaxDedupKeys = 65536
	}
	return c
}

// Receiver is a fail-closed OTLP logs http/json handler with idempotent persist.
type Receiver struct {
	sink Sink
	cfg  Config

	mu    sync.Mutex
	seen  map[string]*keyState
	order []string // committed keys only, FIFO for bounded eviction
	stats Stats
}

// keyState tracks one idempotency key. A key is first reserved by a LEADER
// (committed=false, done open) while its sink write is in flight; concurrent
// duplicates wait on done. On success the leader commits (committed=true,
// done closed) so waiters see a real persist and count as duplicates. On
// failure the leader aborts (entry removed, done closed) so a waiter re-leads
// and retries — an in-flight leader is NEVER treated as persisted.
type keyState struct {
	done      chan struct{}
	committed bool
}

// New builds a Receiver over sink. A nil sink is rejected on use (fail-closed).
func New(sink Sink, cfg Config) *Receiver {
	return &Receiver{sink: sink, cfg: cfg.withDefaults(), seen: map[string]*keyState{}}
}

// Stats returns a snapshot of the cumulative counters.
func (r *Receiver) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// LoopbackServer returns an *http.Server bound to 127.0.0.1:port with bounded
// timeouts, serving POST /v1/logs.
func (r *Receiver) LoopbackServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/v1/logs", r)
	return &http.Server{
		Addr:              "127.0.0.1:" + strconv.Itoa(port),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
}

// --- minimal OTLP/HTTP JSON logs shapes. Body is deliberately NOT modeled. ---

type exportLogsServiceRequest struct {
	ResourceLogs []resourceLogs `json:"resourceLogs"`
}
type resourceLogs struct {
	Resource  resource    `json:"resource"`
	ScopeLogs []scopeLogs `json:"scopeLogs"`
}
type resource struct {
	Attributes []keyValue `json:"attributes"`
}
type scopeLogs struct {
	Scope      scope       `json:"scope"`
	LogRecords []logRecord `json:"logRecords"`
}
type scope struct {
	Name string `json:"name"`
}
type logRecord struct {
	TimeUnixNano string     `json:"timeUnixNano"`
	Attributes   []keyValue `json:"attributes"`
	// Body intentionally omitted: content is never read.
}
type keyValue struct {
	Key   string   `json:"key"`
	Value anyValue `json:"value"`
}
type anyValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
}

func (r *Receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLoopbackRemote(req.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if ct := req.Header.Get("Content-Type"); !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "application/json") {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	if r.sink == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}

	limited := http.MaxBytesReader(w, req.Body, r.cfg.MaxBodyBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Unmodeled OTLP fields (incl. the log body/message) are ignored by
	// encoding/json and never persisted — content-off by design.
	var payload exportLogsServiceRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extract target records (fail closed on structurally invalid values).
	type pending struct {
		rec SanitizedRecord
		key string
	}
	var toPersist []pending
	incomplete := int64(0)
	count := 0
	for _, rl := range payload.ResourceLogs {
		trustedTask, taskOK := r.trustedResourceValue(rl.Resource, resAttrTaskID)
		trustedReq, reqOK := r.trustedResourceValue(rl.Resource, resAttrRequestID)
		if !taskOK || !reqOK {
			// Malformed (out-of-bounds) trusted correlation is treated as absent.
			trustedTask, trustedReq = "", ""
		}
		// Documented identity gate: resource service.name == "claude-code".
		serviceName, _ := r.trustedResourceValue(rl.Resource, resAttrServiceName)
		serviceOK := serviceName == ServiceNameClaudeCode
		for _, sl := range rl.ScopeLogs {
			// Instrumentation scope is undocumented/untrusted free-form: it is
			// IGNORED entirely — it neither gates target selection nor is persisted.
			for _, lr := range sl.LogRecords {
				count++
				if count > r.cfg.MaxRecords {
					http.Error(w, "too many records", http.StatusRequestEntityTooLarge)
					return
				}
				rec, isTarget, bad := r.sanitizeLog(lr)
				if bad {
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				// Target = DOCUMENTED signals only: service.name==claude-code AND
				// event.name==api_request. Non-Claude-Code service or a different
				// event.name -> drop, not counted.
				if !isTarget || !serviceOK {
					continue
				}
				// Merge TRUSTED resource correlation; log attrs cannot override it.
				rec.TrustedTaskID = trustedTask
				rec.TrustedRequestID = trustedReq
				// Fail closed: a target record must carry log request_id AND the
				// trusted task/request correlation, else drop + count.
				if rec.RequestID == "" || rec.TrustedTaskID == "" || rec.TrustedRequestID == "" {
					incomplete++
					continue
				}
				toPersist = append(toPersist, pending{rec: rec, key: idempotencyKey(rec)})
			}
		}
	}

	persisted := int64(0)
	dups := int64(0)
	for _, p := range toPersist {
		if err := req.Context().Err(); err != nil {
			r.addStats(incomplete+int64(len(toPersist)), persisted, incomplete, dups)
			http.Error(w, "client canceled", http.StatusRequestTimeout)
			return
		}
		isLeader, err := r.acquire(req.Context(), p.key)
		if err != nil { // context canceled while waiting on an in-flight leader
			r.addStats(incomplete+int64(len(toPersist)), persisted, incomplete, dups)
			http.Error(w, "client canceled", http.StatusRequestTimeout)
			return
		}
		if !isLeader {
			dups++ // truly-committed duplicate (OTLP retry) — already persisted
			continue
		}
		if err := r.sink.Record(p.rec); err != nil {
			r.abort(p.key) // not committed -> wake waiters so one re-leads; retry re-persists
			r.addStats(incomplete+int64(len(toPersist)), persisted, incomplete, dups)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		r.commit(p.key)
		persisted++
	}

	r.addStats(incomplete+int64(len(toPersist)), persisted, incomplete, dups)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"partialSuccess":{}}`))
}

func idempotencyKey(rec SanitizedRecord) string {
	// request_id is the primary idempotency key; client_request_id disambiguates.
	return rec.TrustedRequestID + "|" + rec.RequestID + "|" + rec.ClientRequestID
}

// acquire resolves the idempotency role for key. It returns isLeader=true when
// the caller must perform the sink write; isLeader=false ONLY when the key is
// already COMMITTED (a true duplicate). If another leader's write is in flight,
// acquire blocks until that leader commits (-> duplicate) or aborts (-> the
// caller re-leads), so a concurrent duplicate can never be acknowledged as
// persisted while the leader's write ultimately fails. Honors ctx cancellation.
func (r *Receiver) acquire(ctx context.Context, key string) (bool, error) {
	for {
		r.mu.Lock()
		st, ok := r.seen[key]
		if !ok {
			r.seen[key] = &keyState{done: make(chan struct{})}
			r.mu.Unlock()
			return true, nil // leader
		}
		if st.committed {
			r.mu.Unlock()
			return false, nil // true duplicate (already persisted)
		}
		done := st.done
		r.mu.Unlock()
		select {
		case <-done:
			// leader finished: re-loop -> committed (dup) or removed (re-lead)
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

// commit marks the leader's key persisted, wakes waiters (which then see it as
// a duplicate), and enforces bounded FIFO eviction of committed keys.
func (r *Receiver) commit(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.seen[key]
	if !ok || st.committed {
		return
	}
	st.committed = true
	close(st.done)
	r.order = append(r.order, key)
	for len(r.order) > r.cfg.MaxDedupKeys {
		oldest := r.order[0]
		r.order = r.order[1:]
		if e, ok := r.seen[oldest]; ok && e.committed {
			delete(r.seen, oldest)
		}
	}
}

// abort removes a leader's uncommitted key after a failed sink write and wakes
// any waiters so exactly one of them re-leads and retries — the failed write is
// never observed as persisted.
func (r *Receiver) abort(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.seen[key]
	if !ok || st.committed {
		return
	}
	delete(r.seen, key)
	close(st.done)
}

func (r *Receiver) addStats(targets, persisted, incomplete, dups int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Targets += targets
	r.stats.Persisted += persisted
	r.stats.DroppedIncomplete += incomplete
	r.stats.DuplicatesDropped += dups
}

// sanitizeLog extracts allowlisted LOG attributes. Returns isTarget=false for a
// record whose event.name != api_request; bad=true (fail closed) if an
// allowlisted value is out of bounds or a numeric field is malformed.
func (r *Receiver) sanitizeLog(lr logRecord) (rec SanitizedRecord, isTarget bool, bad bool) {
	vals := map[string]string{}
	for _, kv := range lr.Attributes {
		if !isAllowlistedLog(kv.Key) {
			continue // default-deny: drop identity/prompt/response/tool/etc.
		}
		v, ok := scalarString(kv.Value)
		if !ok {
			continue
		}
		if !r.safeValue(v) {
			return SanitizedRecord{}, false, true
		}
		vals[kv.Key] = v
	}
	if vals[attrEventName] != TargetEventName {
		return SanitizedRecord{}, false, false
	}
	rec = SanitizedRecord{
		EventName:       vals[attrEventName],
		RequestID:       vals[attrRequestID],
		ClientRequestID: vals[attrClientRequestID],
		Model:           vals[attrModel],
		Status:          vals[attrStatus],
	}
	if d := vals[attrDurationMs]; d != "" {
		n, err := strconv.ParseInt(d, 10, 64)
		if err != nil || n < 0 {
			return SanitizedRecord{}, false, true
		}
		rec.DurationMs = n
	}
	if t := strings.TrimSpace(lr.TimeUnixNano); t != "" {
		n, err := strconv.ParseUint(t, 10, 64)
		if err != nil {
			return SanitizedRecord{}, false, true
		}
		rec.StartUnixNano = n
	}
	return rec, true, false
}

// trustedResourceValue reads a single allowlisted resource attribute. ok=false
// when absent or out of bounds (treated as absent -> fail-closed drop later).
func (r *Receiver) trustedResourceValue(res resource, key string) (string, bool) {
	for _, kv := range res.Attributes {
		if kv.Key != key {
			continue
		}
		v, ok := scalarString(kv.Value)
		if !ok || !r.safeValue(v) || v == "" {
			return "", false
		}
		return v, true
	}
	return "", false
}

func isAllowlistedLog(key string) bool {
	switch key {
	case attrEventName, attrRequestID, attrClientRequestID, attrModel, attrStatus, attrDurationMs:
		return true
	default:
		return false
	}
}

func (r *Receiver) safeValue(v string) bool {
	if len(v) > r.cfg.MaxValueLen {
		return false
	}
	for _, c := range v {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func scalarString(v anyValue) (string, bool) {
	switch {
	case v.StringValue != nil:
		return *v.StringValue, true
	case v.IntValue != nil:
		return *v.IntValue, true
	case v.BoolValue != nil:
		return strconv.FormatBool(*v.BoolValue), true
	default:
		return "", false
	}
}

func isLoopbackRemote(remoteAddr string) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
