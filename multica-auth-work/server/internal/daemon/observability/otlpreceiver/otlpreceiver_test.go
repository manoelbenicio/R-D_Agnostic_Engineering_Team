package otlpreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type captureSink struct {
	recs   []SanitizedRecord
	calls  int
	failOn int // 1-based call index to fail; 0 = never
}

func (c *captureSink) Record(r SanitizedRecord) error {
	c.calls++
	if c.failOn > 0 && c.calls == c.failOn {
		return errors.New("synthetic sink failure")
	}
	c.recs = append(c.recs, r)
	return nil
}

const (
	loopback    = "127.0.0.1:54321"
	nonLoopback = "192.0.2.5:9000"
	goodTime    = "1730000000000000000"
	// observedScopeName is an UNDOCUMENTED instrumentation scope; the receiver
	// must accept real events regardless of it (identity is service.name).
	observedScopeName = "io.opentelemetry.contrib.claudecode"
)

func attrsJSON(m map[string]string) string {
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, fmt.Sprintf(`{"key":%q,"value":{"stringValue":%q}}`, k, v))
	}
	return strings.Join(parts, ",")
}

// buildBody assembles an official-shape OTLP/HTTP JSON logs payload: a resource
// with resource.attributes, one instrumentation scope, and one log record that
// carries a content body (always dropped) plus the given log attributes.
func buildBody(scopeName string, resAttrs, logAttrs map[string]string) string {
	return fmt.Sprintf(
		`{"resourceLogs":[{"resource":{"attributes":[%s]},"scopeLogs":[{"scope":{"name":%q},"logRecords":[{"timeUnixNano":%q,"body":{"stringValue":"PROMPT: top-secret user content and response text"},"attributes":[%s]}]}]}]}`,
		attrsJSON(resAttrs), scopeName, goodTime, attrsJSON(logAttrs))
}

// officialFixture is the documented live shape: resource service.name
// "claude-code" + trusted correlation in resource.attributes, event.name
// "api_request", an UNDOCUMENTED instrumentation scope (which must not matter),
// plus forbidden identity/prompt/tool attributes (and a content body) that must
// be dropped, and a log-record agent_brain.task_id that must NOT override trust.
func officialFixture() string {
	res := map[string]string{
		"service.name":           "claude-code",
		"agent_brain.task_id":    "task-77",
		"agent_brain.request_id": "abreq-9",
		"account_email":          "user@example.com", // forbidden resource attr (dropped)
	}
	log := map[string]string{
		"event.name":          TargetEventName,
		"request_id":          "req-123",
		"client_request_id":   "cli-9",
		"model":               "claude-opus-4",
		"status":              "ok",
		"duration_ms":         "1234",
		"prompt":              "do-the-harmful-thing", // dropped
		"response.text":       "here-is-the-answer",   // dropped
		"tool_calls":          "tool-payload",         // dropped
		"agent_brain.task_id": "EVIL-override",        // must NOT override trusted resource value
	}
	return buildBody(observedScopeName, res, log)
}

func newReq(method, ct, remote, body string) *http.Request {
	req := httptest.NewRequest(method, "/v1/logs", strings.NewReader(body))
	req.RemoteAddr = remote
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	return req
}

func serve(rec *Receiver, req *http.Request) *httptest.ResponseRecorder {
	rw := httptest.NewRecorder()
	rec.ServeHTTP(rw, req)
	return rw
}

func post(rec *Receiver, body string) *httptest.ResponseRecorder {
	return serve(rec, newReq(http.MethodPost, "application/json", loopback, body))
}

func TestOfficialShapeAllowlistTrustedCorrelationAndContentOff(t *testing.T) {
	sink := &captureSink{}
	rec := New(sink, Config{})
	rw := post(rec, officialFixture())
	if rw.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rw.Code, rw.Body.String())
	}
	if len(sink.recs) != 1 {
		t.Fatalf("want 1 persisted record, got %d", len(sink.recs))
	}
	got := sink.recs[0]
	if got.EventName != TargetEventName || got.RequestID != "req-123" || got.ClientRequestID != "cli-9" ||
		got.Model != "claude-opus-4" || got.Status != "ok" || got.DurationMs != 1234 || got.StartUnixNano != 1730000000000000000 {
		t.Fatalf("log allowlist fields wrong: %+v", got)
	}
	// Trusted correlation comes from resource.attributes and is NOT overridden
	// by the log-record's agent_brain.task_id "EVIL-override".
	if got.TrustedTaskID != "task-77" || got.TrustedRequestID != "abreq-9" {
		t.Fatalf("trusted correlation wrong (override leak?): %+v", got)
	}
	blob, _ := json.Marshal(got)
	for _, forbidden := range []string{
		"top-secret", "user content", "do-the-harmful-thing", "here-is-the-answer",
		"user@example.com", "tool-payload", "EVIL-override", "PROMPT",
	} {
		if strings.Contains(string(blob), forbidden) {
			t.Fatalf("persisted record leaked forbidden data %q: %s", forbidden, blob)
		}
	}
	if s := rec.Stats(); s.Targets != 1 || s.Persisted != 1 || s.DroppedIncomplete != 0 || s.DuplicatesDropped != 0 {
		t.Fatalf("stats=%+v", s)
	}
}

func TestServiceNameAndEventNameIdentifyTarget(t *testing.T) {
	full := map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.task_id": "t", "agent_brain.request_id": "r"}
	target := map[string]string{"event.name": TargetEventName, "request_id": "req-1"}

	// Non-Claude-Code service.name -> not our telemetry -> dropped (not a target).
	sink := &captureSink{}
	rec := New(sink, Config{})
	wrongSvc := map[string]string{"service.name": "some-other-service", "agent_brain.task_id": "t", "agent_brain.request_id": "r"}
	if rw := post(rec, buildBody(observedScopeName, wrongSvc, target)); rw.Code != http.StatusOK {
		t.Fatalf("status=%d", rw.Code)
	}
	if len(sink.recs) != 0 || rec.Stats().Targets != 0 {
		t.Fatalf("wrong service.name must be dropped; recs=%d stats=%+v", len(sink.recs), rec.Stats())
	}

	// Missing service.name -> dropped (fail closed on identity).
	sink2 := &captureSink{}
	rec2 := New(sink2, Config{})
	noSvc := map[string]string{"agent_brain.task_id": "t", "agent_brain.request_id": "r"}
	if rw := post(rec2, buildBody(observedScopeName, noSvc, target)); rw.Code != http.StatusOK {
		t.Fatalf("status=%d", rw.Code)
	}
	if len(sink2.recs) != 0 || rec2.Stats().Targets != 0 {
		t.Fatalf("missing service.name must be dropped; recs=%d", len(sink2.recs))
	}

	// Correct service.name but the OLD dotted event.name value -> not a target.
	sink3 := &captureSink{}
	rec3 := New(sink3, Config{})
	logDotted := map[string]string{"event.name": "claude_code.api_request", "request_id": "req-1"}
	if rw := post(rec3, buildBody(observedScopeName, full, logDotted)); rw.Code != http.StatusOK {
		t.Fatalf("status=%d", rw.Code)
	}
	if len(sink3.recs) != 0 {
		t.Fatalf("dotted event.name must not be a target; persisted %d", len(sink3.recs))
	}
}

// TestArbitraryScopeNeitherGatesNorPersists proves the instrumentation scope is
// ignored entirely: a real event is accepted under ANY scope (documented
// identity is service.name), and no scope value — arbitrary or malicious —
// appears in the marshaled closed-schema record (which has no scope field).
func TestArbitraryScopeNeitherGatesNorPersists(t *testing.T) {
	full := map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.task_id": "t", "agent_brain.request_id": "r"}
	target := map[string]string{"event.name": TargetEventName, "request_id": "req-1"}

	scopes := []string{
		observedScopeName,
		"com.anthropic.claude_code",
		"",
		"totally.unknown.scope",
		`<script>alert(1)</script>`, // malicious free-form
		"PROMPT-leak-" + strings.Repeat("x", 2048), // long/hostile
	}
	for _, scope := range scopes {
		sink := &captureSink{}
		rec := New(sink, Config{})
		if rw := post(rec, buildBody(scope, full, target)); rw.Code != http.StatusOK {
			t.Fatalf("scope(len=%d): status=%d", len(scope), rw.Code)
		}
		// Scope does not gate: the real event is accepted regardless of scope.
		if len(sink.recs) != 1 || rec.Stats().Targets != 1 {
			t.Fatalf("scope(len=%d) must not gate; recs=%d stats=%+v", len(scope), len(sink.recs), rec.Stats())
		}
		blob, _ := json.Marshal(sink.recs[0])
		// Scope value never persisted.
		if scope != "" && strings.Contains(string(blob), scope) {
			t.Fatalf("scope leaked into persisted record: %s", blob)
		}
		// Closed schema exposes NO scope field at all.
		if strings.Contains(string(blob), "Scope") {
			t.Fatalf("record schema must not expose a scope field: %s", blob)
		}
	}
}

func TestFailClosedDropAndCountIncomplete(t *testing.T) {
	full := map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.task_id": "task-77", "agent_brain.request_id": "abreq-9"}
	target := map[string]string{"event.name": TargetEventName, "request_id": "req-123"}

	cases := []struct {
		name     string
		res, log map[string]string
	}{
		{"missing_log_request_id", full, map[string]string{"event.name": TargetEventName}},
		{"missing_trusted_task_id", map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.request_id": "abreq-9"}, target},
		{"missing_trusted_request_id", map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.task_id": "task-77"}, target},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sink := &captureSink{}
			rec := New(sink, Config{})
			if rw := post(rec, buildBody(observedScopeName, tc.res, tc.log)); rw.Code != http.StatusOK {
				t.Fatalf("status=%d", rw.Code)
			}
			if len(sink.recs) != 0 {
				t.Fatalf("incomplete target must not persist, got %d", len(sink.recs))
			}
			if s := rec.Stats(); s.Targets != 1 || s.DroppedIncomplete != 1 || s.Persisted != 0 {
				t.Fatalf("incomplete must be dropped+counted, stats=%+v", s)
			}
		})
	}
}

func TestIdempotencyDeduplicatesRetries(t *testing.T) {
	sink := &captureSink{}
	rec := New(sink, Config{})
	body := officialFixture()

	if rw := post(rec, body); rw.Code != http.StatusOK {
		t.Fatalf("first status=%d", rw.Code)
	}
	if rw := post(rec, body); rw.Code != http.StatusOK { // OTLP client retry of same batch
		t.Fatalf("retry status=%d", rw.Code)
	}
	if len(sink.recs) != 1 {
		t.Fatalf("duplicate retry must persist once, got %d", len(sink.recs))
	}
	if s := rec.Stats(); s.Persisted != 1 || s.DuplicatesDropped != 1 {
		t.Fatalf("stats after retry=%+v", s)
	}
}

func TestSinkErrorReleasesKeyForLaterRetry(t *testing.T) {
	sink := &captureSink{failOn: 1} // fail the first Record call
	rec := New(sink, Config{})
	body := officialFixture()

	if rw := post(rec, body); rw.Code != http.StatusServiceUnavailable {
		t.Fatalf("sink failure want 503, got %d", rw.Code)
	}
	if len(sink.recs) != 0 {
		t.Fatalf("failed write must not persist, got %d", len(sink.recs))
	}
	// Retry after the transient sink failure MUST re-persist (key was released,
	// not swallowed by the dedup set).
	sink.failOn = 0
	if rw := post(rec, body); rw.Code != http.StatusOK {
		t.Fatalf("retry after sink failure want 200, got %d", rw.Code)
	}
	if len(sink.recs) != 1 {
		t.Fatalf("retry after failure must re-persist exactly once, got %d", len(sink.recs))
	}
}

func TestBoundsAndErrorPathsFailClosed(t *testing.T) {
	res := map[string]string{"service.name": ServiceNameClaudeCode, "agent_brain.task_id": "t", "agent_brain.request_id": "r"}
	target := map[string]string{"event.name": TargetEventName, "request_id": "req-1"}
	good := buildBody(observedScopeName, res, target)

	t.Run("method", func(t *testing.T) {
		sink := &captureSink{}
		rw := serve(New(sink, Config{}), newReq(http.MethodGet, "application/json", loopback, good))
		if rw.Code != http.StatusMethodNotAllowed || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("non_loopback", func(t *testing.T) {
		sink := &captureSink{}
		rw := serve(New(sink, Config{}), newReq(http.MethodPost, "application/json", nonLoopback, good))
		if rw.Code != http.StatusForbidden || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("content_type", func(t *testing.T) {
		sink := &captureSink{}
		rw := serve(New(sink, Config{}), newReq(http.MethodPost, "text/plain", loopback, good))
		if rw.Code != http.StatusUnsupportedMediaType || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("malformed_json", func(t *testing.T) {
		sink := &captureSink{}
		if rw := post(New(sink, Config{}), `{"resourceLogs":[`); rw.Code != http.StatusBadRequest {
			t.Fatalf("code=%d", rw.Code)
		}
	})
	t.Run("oversized_body", func(t *testing.T) {
		sink := &captureSink{}
		if rw := post(New(sink, Config{MaxBodyBytes: 32}), good); rw.Code != http.StatusBadRequest {
			t.Fatalf("code=%d", rw.Code)
		}
	})
	t.Run("too_many_records", func(t *testing.T) {
		two := `{"resourceLogs":[{"resource":{"attributes":[{"key":"agent_brain.task_id","value":{"stringValue":"t"}},{"key":"agent_brain.request_id","value":{"stringValue":"r"}}]},"scopeLogs":[{"scope":{"name":"claude_code"},"logRecords":[` +
			`{"attributes":[{"key":"event.name","value":{"stringValue":"api_request"}},{"key":"request_id","value":{"stringValue":"a"}}]},` +
			`{"attributes":[{"key":"event.name","value":{"stringValue":"api_request"}},{"key":"request_id","value":{"stringValue":"b"}}]}]}]}]}`
		sink := &captureSink{}
		if rw := post(New(sink, Config{MaxRecords: 1}), two); rw.Code != http.StatusRequestEntityTooLarge || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("bad_duration", func(t *testing.T) {
		log := map[string]string{"event.name": TargetEventName, "request_id": "req-1", "duration_ms": "-5"}
		sink := &captureSink{}
		if rw := post(New(sink, Config{}), buildBody(observedScopeName, res, log)); rw.Code != http.StatusBadRequest || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("oversized_value", func(t *testing.T) {
		log := map[string]string{"event.name": TargetEventName, "request_id": "req-1", "model": strings.Repeat("m", 64)}
		sink := &captureSink{}
		if rw := post(New(sink, Config{MaxValueLen: 8}), buildBody(observedScopeName, res, log)); rw.Code != http.StatusBadRequest || len(sink.recs) != 0 {
			t.Fatalf("code=%d recs=%d", rw.Code, len(sink.recs))
		}
	})
	t.Run("nil_sink", func(t *testing.T) {
		if rw := post(New(nil, Config{}), good); rw.Code != http.StatusServiceUnavailable {
			t.Fatalf("code=%d", rw.Code)
		}
	})
	t.Run("canceled_context", func(t *testing.T) {
		sink := &configurableSink{entered: make(chan struct{}), proceed: make(chan struct{})}
		rec := New(sink, Config{})

		// 1. Leader starts and blocks inside sink.Record
		go func() { post(rec, good) }()
		<-sink.entered

		// 2. Follower starts with a pre-canceled context while leader is in-flight
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := newReq(http.MethodPost, "application/json", loopback, good).WithContext(ctx)
		rw := serve(rec, req)

		// 3. Clean up leader
		close(sink.proceed)

		if rw.Code != http.StatusRequestTimeout {
			t.Fatalf("canceled context while waiting on leader want 408, got code=%d", rw.Code)
		}
	})
}

func TestAllowlistAudit(t *testing.T) {
	if r := AllowlistedResourceAttributes(); len(r) != 2 || r[0] != "agent_brain.task_id" || r[1] != "agent_brain.request_id" {
		t.Fatalf("resource allowlist=%v", r)
	}
	logKeys := AllowlistedLogAttributes()
	want := map[string]bool{"event.name": true, "request_id": true, "client_request_id": true, "model": true, "status": true, "duration_ms": true}
	if len(logKeys) != len(want) {
		t.Fatalf("log allowlist size=%d want %d (%v)", len(logKeys), len(want), logKeys)
	}
	for _, k := range logKeys {
		if !want[k] {
			t.Fatalf("unexpected log allowlist key %q", k)
		}
	}
}

func TestLoopbackServerBindsLoopbackWithTimeouts(t *testing.T) {
	srv := New(&captureSink{}, Config{}).LoopbackServer(0)
	if !strings.HasPrefix(srv.Addr, "127.0.0.1:") {
		t.Fatalf("server must bind loopback, got %q", srv.Addr)
	}
	if srv.ReadTimeout == 0 || srv.ReadHeaderTimeout == 0 || srv.WriteTimeout == 0 {
		t.Fatalf("server must set bounded timeouts: %+v", srv)
	}
}

type configurableSink struct {
	mu        sync.Mutex
	entered   chan struct{}
	proceed   chan struct{}
	records   []SanitizedRecord
	failCount int // number of initial Record calls to fail
	callCount int
}

func (s *configurableSink) Record(r SanitizedRecord) error {
	s.mu.Lock()
	s.callCount++
	currentCall := s.callCount
	if currentCall == 1 && s.entered != nil {
		close(s.entered) // signal test runner that leader is in Record
	}
	s.mu.Unlock()

	if currentCall == 1 && s.proceed != nil {
		<-s.proceed // wait for unblock signal
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if currentCall <= s.failCount {
		return errors.New("synthetic sink failure")
	}
	s.records = append(s.records, r)
	return nil
}

func TestConcurrentIdenticalPOSTs_LeaderBlocksThenFails_FollowerReLeadsAndSucceeds(t *testing.T) {
	sink := &configurableSink{
		entered:   make(chan struct{}),
		proceed:   make(chan struct{}),
		failCount: 1, // fail leader (call #1), allow follower (call #2)
	}
	rec := New(sink, Config{})
	body := officialFixture()

	var leaderCode, followerCode int
	var leaderBody, followerBody string
	leaderDone := make(chan struct{})
	followerDone := make(chan struct{})

	// 1. Leader request starts
	go func() {
		defer close(leaderDone)
		rw := post(rec, body)
		leaderCode = rw.Code
		leaderBody = rw.Body.String()
	}()

	// Wait until leader enters sink.Record
	<-sink.entered

	// 2. Follower request starts while leader is blocked inside sink.Record
	go func() {
		defer close(followerDone)
		rw := post(rec, body)
		followerCode = rw.Code
		followerBody = rw.Body.String()
	}()

	// Give follower a moment to enter acquire() and block on st.done
	time.Sleep(10 * time.Millisecond)

	// 3. Unblock leader, forcing it to fail
	close(sink.proceed)

	// Both requests complete
	<-leaderDone
	<-followerDone

	t.Logf("Leader HTTP Code: %d (body: %s)", leaderCode, leaderBody)
	t.Logf("Follower HTTP Code: %d (body: %s)", followerCode, followerBody)
	t.Logf("Persisted Records Count: %d", len(sink.records))

	// Leader failed -> 503
	if leaderCode != http.StatusServiceUnavailable {
		t.Fatalf("Leader should return 503 Service Unavailable, got %d", leaderCode)
	}

	// Follower waited, re-led after leader failure, succeeded -> 200 OK
	if followerCode != http.StatusOK {
		t.Fatalf("Follower should re-lead and return 200 OK after leader failure, got %d", followerCode)
	}

	// Exactly 1 record persisted (by follower)
	if len(sink.records) != 1 {
		t.Fatalf("Expected exactly 1 persisted record, got %d", len(sink.records))
	}
}

func TestConcurrentIdenticalPOSTs_LeaderBlocksThenFails_BothFail(t *testing.T) {
	sink := &configurableSink{
		entered:   make(chan struct{}),
		proceed:   make(chan struct{}),
		failCount: 2, // fail both leader (call #1) and follower (call #2)
	}
	rec := New(sink, Config{})
	body := officialFixture()

	var leaderCode, followerCode int
	leaderDone := make(chan struct{})
	followerDone := make(chan struct{})

	// 1. Leader starts
	go func() {
		defer close(leaderDone)
		rw := post(rec, body)
		leaderCode = rw.Code
	}()

	<-sink.entered

	// 2. Follower starts and blocks waiting for leader
	go func() {
		defer close(followerDone)
		rw := post(rec, body)
		followerCode = rw.Code
	}()

	time.Sleep(10 * time.Millisecond)

	// 3. Unblock leader
	close(sink.proceed)

	<-leaderDone
	<-followerDone

	// Both failed -> 503
	if leaderCode != http.StatusServiceUnavailable || followerCode != http.StatusServiceUnavailable {
		t.Fatalf("Both requests should return 503, got leader=%d follower=%d", leaderCode, followerCode)
	}

	// 0 records persisted, NO FALSE 200 OK!
	if len(sink.records) != 0 {
		t.Fatalf("Expected 0 persisted records, got %d", len(sink.records))
	}
}

func TestConcurrentIdenticalPOSTs_LeaderSucceeds_FollowerWaitsAndDups(t *testing.T) {
	sink := &configurableSink{
		entered:   make(chan struct{}),
		proceed:   make(chan struct{}),
		failCount: 0, // leader succeeds
	}
	rec := New(sink, Config{})
	body := officialFixture()

	var leaderCode, followerCode int
	leaderDone := make(chan struct{})
	followerDone := make(chan struct{})

	// 1. Leader starts
	go func() {
		defer close(leaderDone)
		rw := post(rec, body)
		leaderCode = rw.Code
	}()

	<-sink.entered

	// 2. Follower starts and blocks waiting for leader
	go func() {
		defer close(followerDone)
		rw := post(rec, body)
		followerCode = rw.Code
	}()

	time.Sleep(10 * time.Millisecond)

	// 3. Unblock leader
	close(sink.proceed)

	<-leaderDone
	<-followerDone

	// Both return 200 OK
	if leaderCode != http.StatusOK || followerCode != http.StatusOK {
		t.Fatalf("Both requests should return 200 OK, got leader=%d follower=%d", leaderCode, followerCode)
	}

	// Exactly 1 record persisted (no duplicate writes!)
	if len(sink.records) != 1 {
		t.Fatalf("Expected exactly 1 persisted record (no duplicates), got %d", len(sink.records))
	}

	// Stats check: 1 persisted, 1 duplicate dropped
	stats := rec.Stats()
	if stats.Persisted != 1 || stats.DuplicatesDropped != 1 {
		t.Fatalf("Stats mismatch: persisted=%d dups=%d, want 1/1", stats.Persisted, stats.DuplicatesDropped)
	}
}

// gateSink lets a test control exactly when a Record call is in flight and what
// it returns, so the concurrent leader/duplicate race is deterministic.
type gateSink struct {
	mu      sync.Mutex
	recs    []SanitizedRecord
	enter   chan struct{}
	release chan error
}

func (g *gateSink) Record(r SanitizedRecord) error {
	g.enter <- struct{}{}
	err := <-g.release
	if err == nil {
		g.mu.Lock()
		g.recs = append(g.recs, r)
		g.mu.Unlock()
	}
	return err
}

func (g *gateSink) count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.recs)
}

// TestConcurrentDuplicateLeaderFailureNoAcknowledgedLoss reproduces the
// lost-ack race: leader A reserves the key and is mid-Record; concurrent
// duplicate B arrives for the same key; A's sink write FAILS. B must NOT be
// acknowledged (200) as persisted while nothing was written. With the wait/
// state fix, B waits for A's outcome and — because A aborted — re-leads and
// persists: exactly one eventual persist, no acknowledged loss, correct counters.
func TestConcurrentDuplicateLeaderFailureNoAcknowledgedLoss(t *testing.T) {
	gs := &gateSink{enter: make(chan struct{}), release: make(chan error)}
	rec := New(gs, Config{})
	body := officialFixture()

	aCode := make(chan int, 1)
	go func() { aCode <- post(rec, body).Code }()
	<-gs.enter // leader A reserved the key and is inside sink.Record (in-flight)

	bCode := make(chan int, 1)
	go func() { bCode <- post(rec, body).Code }() // concurrent duplicate for the same key

	// Fail leader A's write. A aborts (releases the key) and returns 503.
	gs.release <- errors.New("leader sink failure")
	if code := <-aCode; code != http.StatusServiceUnavailable {
		t.Fatalf("leader A must fail closed with 503, got %d", code)
	}

	// B (waiter) re-leads after A's abort and enters the sink; let it succeed.
	<-gs.enter
	gs.release <- nil
	if code := <-bCode; code != http.StatusOK {
		t.Fatalf("re-leading B must succeed with 200, got %d", code)
	}

	if gs.count() != 1 {
		t.Fatalf("exactly one eventual persist expected, got %d", gs.count())
	}
	// B re-led (it was NOT a true duplicate, since A never committed): persisted
	// once, zero duplicates dropped. Crucially, no 200 was returned without a
	// backing persist (no acknowledged loss).
	if s := rec.Stats(); s.Persisted != 1 || s.DuplicatesDropped != 0 {
		t.Fatalf("counters wrong: Persisted=%d DuplicatesDropped=%d, want 1/0", s.Persisted, s.DuplicatesDropped)
	}
}

// TestConcurrentDuplicateLeaderSuccessCountsDuplicate is the success-path twin:
// when the leader COMMITS, the concurrent duplicate is acknowledged as a true
// duplicate (200) and counted, with exactly one persist.
func TestConcurrentDuplicateLeaderSuccessCountsDuplicate(t *testing.T) {
	gs := &gateSink{enter: make(chan struct{}), release: make(chan error)}
	rec := New(gs, Config{})
	body := officialFixture()

	aCode := make(chan int, 1)
	go func() { aCode <- post(rec, body).Code }()
	<-gs.enter // leader A in-flight

	bCode := make(chan int, 1)
	go func() { bCode <- post(rec, body).Code }() // concurrent duplicate

	gs.release <- nil // leader A commits
	if code := <-aCode; code != http.StatusOK {
		t.Fatalf("leader A want 200, got %d", code)
	}
	// B was waiting; A committed -> B sees a true duplicate (no second sink call).
	if code := <-bCode; code != http.StatusOK {
		t.Fatalf("duplicate B want 200, got %d", code)
	}
	if gs.count() != 1 {
		t.Fatalf("exactly one persist expected, got %d", gs.count())
	}
	if s := rec.Stats(); s.Persisted != 1 || s.DuplicatesDropped != 1 {
		t.Fatalf("counters wrong: Persisted=%d DuplicatesDropped=%d, want 1/1", s.Persisted, s.DuplicatesDropped)
	}
}
