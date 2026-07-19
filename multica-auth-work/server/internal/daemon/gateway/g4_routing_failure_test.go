package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type syntheticAccountStatus uint8

const (
	syntheticEligible syntheticAccountStatus = iota
	syntheticQuarantined
	syntheticRemoved
)

var errSyntheticRouterUnavailable = errors.New("synthetic router unavailable")

type syntheticRouteRequest struct {
	PreviousResponseID string
	PromptCacheID      string
	ToolTurnID         string
}

func (r syntheticRouteRequest) affinityKey() string {
	for _, candidate := range []struct {
		kind  string
		value string
	}{
		{kind: "previous_response_id", value: r.PreviousResponseID},
		{kind: "prompt_cache", value: r.PromptCacheID},
		{kind: "tool_turn", value: r.ToolTurnID},
	} {
		if candidate.value != "" {
			return candidate.kind + ":" + candidate.value
		}
	}
	return ""
}

type syntheticRouteSelection struct {
	Sequence uint64
	Account  string
}

type syntheticRouterSnapshot struct {
	Order         []string
	Status        map[string]syntheticAccountStatus
	Cursor        int
	Affinity      map[string]string
	ConfigVersion string
}

type syntheticRouter struct {
	mu            sync.Mutex
	order         []string
	status        map[string]syntheticAccountStatus
	cursor        int
	sequence      uint64
	affinity      map[string]string
	running       bool
	configVersion string
}

func newSyntheticRouter(accounts ...string) *syntheticRouter {
	status := make(map[string]syntheticAccountStatus, len(accounts))
	for _, account := range accounts {
		status[account] = syntheticEligible
	}
	return &syntheticRouter{
		order: append([]string(nil), accounts...), status: status,
		affinity: make(map[string]string), running: true, configVersion: "synthetic-v1",
	}
}

func (r *syntheticRouter) selectRoute(request syntheticRouteRequest) (syntheticRouteSelection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return syntheticRouteSelection{}, errSyntheticRouterUnavailable
	}
	key := request.affinityKey()
	if account, exists := r.affinity[key]; key != "" && exists && r.status[account] == syntheticEligible {
		r.sequence++
		return syntheticRouteSelection{Sequence: r.sequence, Account: account}, nil
	}
	for offset := 0; offset < len(r.order); offset++ {
		index := (r.cursor + offset) % len(r.order)
		account := r.order[index]
		if r.status[account] != syntheticEligible {
			continue
		}
		r.cursor = (index + 1) % len(r.order)
		r.sequence++
		if key != "" {
			r.affinity[key] = account
		}
		return syntheticRouteSelection{Sequence: r.sequence, Account: account}, nil
	}
	return syntheticRouteSelection{}, errSyntheticRouterUnavailable
}

func (r *syntheticRouter) add(account string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.status[account]; !exists {
		r.order = append(r.order, account)
	}
	r.status[account] = syntheticEligible
}

func (r *syntheticRouter) setStatus(account string, status syntheticAccountStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.status[account]; exists {
		r.status[account] = status
	}
}

func (r *syntheticRouter) snapshot() syntheticRouterSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotLocked()
}

func (r *syntheticRouter) snapshotLocked() syntheticRouterSnapshot {
	status := make(map[string]syntheticAccountStatus, len(r.status))
	for account, state := range r.status {
		status[account] = state
	}
	affinity := make(map[string]string, len(r.affinity))
	for key, account := range r.affinity {
		affinity[key] = account
	}
	return syntheticRouterSnapshot{
		Order: append([]string(nil), r.order...), Status: status, Cursor: r.cursor,
		Affinity: affinity, ConfigVersion: r.configVersion,
	}
}

func (r *syntheticRouter) restart(version string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configVersion = version
	r.running = true
}

func (r *syntheticRouter) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
}

func (r *syntheticRouter) rollback(snapshot syntheticRouterSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append([]string(nil), snapshot.Order...)
	r.status = make(map[string]syntheticAccountStatus, len(snapshot.Status))
	for account, state := range snapshot.Status {
		r.status[account] = state
	}
	r.affinity = make(map[string]string, len(snapshot.Affinity))
	for key, account := range snapshot.Affinity {
		r.affinity[key] = account
	}
	r.cursor = snapshot.Cursor
	r.configVersion = snapshot.ConfigVersion
	r.running = true
}

func TestG4ConcurrentStrictIndependentRequestRoundRobin(t *testing.T) {
	accounts := []string{"synthetic-account-a", "synthetic-account-b", "synthetic-account-c"}
	router := newSyntheticRouter(accounts...)
	const requestCount = 96
	selections := make(chan syntheticRouteSelection, requestCount)
	release := make(chan struct{})
	var workers sync.WaitGroup
	var active atomic.Int64
	var peak atomic.Int64
	for index := 0; index < requestCount; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			selection, err := router.selectRoute(syntheticRouteRequest{})
			if err != nil {
				selections <- syntheticRouteSelection{}
				return
			}
			current := active.Add(1)
			for observed := peak.Load(); current > observed && !peak.CompareAndSwap(observed, current); observed = peak.Load() {
			}
			selections <- selection
			<-release
			active.Add(-1)
		}()
	}
	ordered := make([]syntheticRouteSelection, 0, requestCount)
	for index := 0; index < requestCount; index++ {
		ordered = append(ordered, <-selections)
	}
	close(release)
	workers.Wait()
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].Sequence < ordered[right].Sequence })
	for index, selection := range ordered {
		if selection.Sequence == 0 || selection.Account != accounts[index%len(accounts)] {
			t.Fatalf("strict round-robin mismatch at sequence %d", index+1)
		}
	}
	if peak.Load() < 2 || active.Load() != 0 {
		t.Fatalf("concurrency accounting mismatch: peak=%d active=%d", peak.Load(), active.Load())
	}
}

func TestG4ContinuationAffinityFamilies(t *testing.T) {
	cases := []struct {
		name    string
		request syntheticRouteRequest
	}{
		{name: "previous_response_id", request: syntheticRouteRequest{PreviousResponseID: "synthetic-response-affinity"}},
		{name: "prompt_cache", request: syntheticRouteRequest{PromptCacheID: "synthetic-cache-affinity"}},
		{name: "tool_turn", request: syntheticRouteRequest{ToolTurnID: "synthetic-tool-affinity"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			router := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c")
			origin, err := router.selectRoute(test.request)
			if err != nil {
				t.Fatal(err)
			}
			independent, err := router.selectRoute(syntheticRouteRequest{})
			if err != nil {
				t.Fatal(err)
			}
			continuation, err := router.selectRoute(test.request)
			if err != nil {
				t.Fatal(err)
			}
			if origin.Account != continuation.Account || independent.Account == origin.Account {
				t.Fatalf("affinity or independent rotation contract mismatch")
			}
		})
	}
}

type syntheticFailureKind string

const (
	syntheticExpiredAccess  syntheticFailureKind = "expired-access"
	syntheticRevokedRefresh syntheticFailureKind = "revoked-refresh"
	syntheticForbidden      syntheticFailureKind = "forbidden"
	syntheticQuota          syntheticFailureKind = "quota"
	syntheticScoped429      syntheticFailureKind = "scoped-429"
	syntheticGlobal429      syntheticFailureKind = "global-429"
	syntheticServer5xx      syntheticFailureKind = "server-5xx"
	syntheticTimeout        syntheticFailureKind = "timeout"
	syntheticMalformed      syntheticFailureKind = "malformed-upstream"
)

type syntheticFailureDecision struct {
	Class     ErrorClass
	Scope     CircuitScope
	Retryable bool
}

func classifySyntheticFailure(kind syntheticFailureKind) syntheticFailureDecision {
	switch kind {
	case syntheticExpiredAccess:
		return syntheticFailureDecision{Class: ErrorAuthentication, Scope: CircuitAccount, Retryable: true}
	case syntheticRevokedRefresh:
		return syntheticFailureDecision{Class: ErrorAuthentication, Scope: CircuitAccount}
	case syntheticForbidden:
		return syntheticFailureDecision{Class: ErrorAuthorization, Scope: CircuitAccount}
	case syntheticQuota, syntheticScoped429:
		return syntheticFailureDecision{Class: ErrorRateLimited, Scope: CircuitAccount, Retryable: true}
	case syntheticGlobal429:
		return syntheticFailureDecision{Class: ErrorRateLimited, Scope: CircuitProvider, Retryable: true}
	case syntheticServer5xx:
		return syntheticFailureDecision{Class: ErrorUpstream, Scope: CircuitProvider, Retryable: true}
	case syntheticTimeout:
		return syntheticFailureDecision{Class: ErrorTimeout, Scope: CircuitLocal, Retryable: true}
	default:
		return syntheticFailureDecision{Class: ErrorProtocol, Scope: CircuitProvider}
	}
}

func TestG4FailureClassificationAndScopeMatrix(t *testing.T) {
	cases := []struct {
		kind      syntheticFailureKind
		class     ErrorClass
		scope     CircuitScope
		retryable bool
	}{
		{syntheticExpiredAccess, ErrorAuthentication, CircuitAccount, true},
		{syntheticRevokedRefresh, ErrorAuthentication, CircuitAccount, false},
		{syntheticForbidden, ErrorAuthorization, CircuitAccount, false},
		{syntheticQuota, ErrorRateLimited, CircuitAccount, true},
		{syntheticScoped429, ErrorRateLimited, CircuitAccount, true},
		{syntheticGlobal429, ErrorRateLimited, CircuitProvider, true},
		{syntheticServer5xx, ErrorUpstream, CircuitProvider, true},
		{syntheticTimeout, ErrorTimeout, CircuitLocal, true},
		{syntheticMalformed, ErrorProtocol, CircuitProvider, false},
	}
	for _, test := range cases {
		t.Run(string(test.kind), func(t *testing.T) {
			decision := classifySyntheticFailure(test.kind)
			if decision.Class != test.class || decision.Scope != test.scope || decision.Retryable != test.retryable {
				t.Fatalf("unexpected synthetic failure decision: %#v", decision)
			}
		})
	}
}

func TestG4MockTransportHTTPTimeoutAndMalformedFailures(t *testing.T) {
	statusCases := []struct {
		status    int
		class     ErrorClass
		retryable bool
	}{
		{http.StatusUnauthorized, ErrorAuthentication, false},
		{http.StatusForbidden, ErrorAuthorization, false},
		{http.StatusTooManyRequests, ErrorRateLimited, true},
		{http.StatusInternalServerError, ErrorUpstream, true},
		{http.StatusServiceUnavailable, ErrorOverloaded, true},
	}
	for _, test := range statusCases {
		response := &http.Response{StatusCode: test.status, Header: make(http.Header)}
		classified := classifyStatus("synthetic.failure", response)
		if classified.Class != test.class || classified.Retryable != test.retryable {
			t.Fatalf("status %d classification mismatch", test.status)
		}
	}

	timeoutClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	timeoutRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://synthetic.invalid/timeout", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, transportErr := timeoutClient.Do(timeoutRequest)
	classifiedErr := classifyTransportError("synthetic.timeout", timeoutRequest.Context(), transportErr)
	if !IsErrorClass(classifiedErr, ErrorTimeout) {
		t.Fatalf("timeout classification mismatch: %v", classifiedErr)
	}

	malformedClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return syntheticResponse(request, http.StatusOK, "application/json", "{synthetic-malformed"), nil
	})}
	malformedRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://synthetic.invalid/malformed", nil)
	if err != nil {
		t.Fatal(err)
	}
	malformedResponse, err := malformedClient.Do(malformedRequest)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	decodeErr := json.NewDecoder(io.LimitReader(malformedResponse.Body, 4096)).Decode(&document)
	_ = malformedResponse.Body.Close()
	if decodeErr == nil || classifySyntheticFailure(syntheticMalformed).Class != ErrorProtocol {
		t.Fatal("malformed upstream was not rejected as a protocol failure")
	}
}

type syntheticAttempt struct {
	Failure             syntheticFailureKind
	OutputCommitted     bool
	ToolActionCommitted bool
	Started             chan<- struct{}
	Release             <-chan struct{}
}

type syntheticExecution struct {
	Attempts     int
	Deduplicated bool
}

type syntheticRetryHarness struct {
	mu        sync.Mutex
	completed map[string]struct{}
	inFlight  map[string]*syntheticRetryFlight
	active    atomic.Int64
	followers atomic.Int64
}

type syntheticRetryFlight struct {
	done   chan struct{}
	result syntheticExecution
	err    error
}

func newSyntheticRetryHarness() *syntheticRetryHarness {
	return &syntheticRetryHarness{
		completed: make(map[string]struct{}),
		inFlight:  make(map[string]*syntheticRetryFlight),
	}
}

func (h *syntheticRetryHarness) execute(ctx context.Context, requestID string, attempts []syntheticAttempt) (result syntheticExecution, resultErr error) {
	h.mu.Lock()
	_, exists := h.completed[requestID]
	if exists {
		h.mu.Unlock()
		return syntheticExecution{Deduplicated: true}, nil
	}
	if flight, exists := h.inFlight[requestID]; exists {
		h.mu.Unlock()
		h.followers.Add(1)
		defer h.followers.Add(-1)
		select {
		case <-ctx.Done():
			return syntheticExecution{Deduplicated: true}, &GatewayError{Operation: "synthetic.retry", Class: ErrorCancelled}
		case <-flight.done:
			return syntheticExecution{Deduplicated: true}, flight.err
		}
	}
	flight := &syntheticRetryFlight{done: make(chan struct{})}
	h.inFlight[requestID] = flight
	h.mu.Unlock()
	h.active.Add(1)
	defer func() {
		h.active.Add(-1)
		h.mu.Lock()
		if resultErr == nil {
			h.completed[requestID] = struct{}{}
		}
		flight.result = result
		flight.err = resultErr
		delete(h.inFlight, requestID)
		close(flight.done)
		h.mu.Unlock()
	}()
	for _, attempt := range attempts {
		if err := ctx.Err(); err != nil {
			return result, &GatewayError{Operation: "synthetic.retry", Class: ErrorCancelled}
		}
		result.Attempts++
		if attempt.Started != nil {
			select {
			case attempt.Started <- struct{}{}:
			case <-ctx.Done():
				return result, &GatewayError{Operation: "synthetic.retry", Class: ErrorCancelled}
			}
		}
		if attempt.Release != nil {
			select {
			case <-attempt.Release:
			case <-ctx.Done():
				return result, &GatewayError{Operation: "synthetic.retry", Class: ErrorCancelled}
			}
		}
		if attempt.Failure == "" {
			return result, nil
		}
		decision := classifySyntheticFailure(attempt.Failure)
		if attempt.OutputCommitted || attempt.ToolActionCommitted || !decision.Retryable {
			return result, &GatewayError{Operation: "synthetic.retry", Class: decision.Class, Retryable: false}
		}
	}
	return result, &GatewayError{Operation: "synthetic.retry", Class: ErrorUpstream, Retryable: false}
}

func (h *syntheticRetryHarness) waitForCancellation(ctx context.Context, started chan<- struct{}) error {
	h.active.Add(1)
	defer h.active.Add(-1)
	close(started)
	<-ctx.Done()
	return &GatewayError{Operation: "synthetic.cancel", Class: ErrorCancelled}
}

func TestG4ReplayBoundariesDedupAndCancellationRelease(t *testing.T) {
	harness := newSyntheticRetryHarness()
	preOutput, err := harness.execute(context.Background(), "synthetic-request-pre-output", []syntheticAttempt{
		{Failure: syntheticServer5xx}, {},
	})
	if err != nil || preOutput.Attempts != 2 {
		t.Fatalf("pre-output retry mismatch: attempts=%d err=%v", preOutput.Attempts, err)
	}
	postOutput, err := harness.execute(context.Background(), "synthetic-request-post-output", []syntheticAttempt{
		{Failure: syntheticServer5xx, OutputCommitted: true}, {},
	})
	if err == nil || postOutput.Attempts != 1 {
		t.Fatalf("post-output request was replayed")
	}
	toolAction, err := harness.execute(context.Background(), "synthetic-request-tool-action", []syntheticAttempt{
		{Failure: syntheticTimeout, ToolActionCommitted: true}, {},
	})
	if err == nil || toolAction.Attempts != 1 {
		t.Fatalf("non-idempotent tool action was replayed")
	}
	dedup, err := harness.execute(context.Background(), "synthetic-request-pre-output", []syntheticAttempt{{}})
	if err != nil || !dedup.Deduplicated || dedup.Attempts != 0 {
		t.Fatalf("completed request was not deduplicated")
	}

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- harness.waitForCancellation(ctx, started) }()
	<-started
	if harness.active.Load() != 1 {
		t.Fatalf("synthetic capacity was not acquired")
	}
	cancel()
	if cancelErr := <-done; !IsErrorClass(cancelErr, ErrorCancelled) {
		t.Fatalf("cancellation classification mismatch: %v", cancelErr)
	}
	if harness.active.Load() != 0 {
		t.Fatalf("synthetic capacity leaked after cancellation")
	}
}

func TestG4SyntheticAccountLifecycleRestartAndRollbackUnderLoad(t *testing.T) {
	router := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c")
	baseline := router.snapshot()
	router.setStatus("synthetic-account-b", syntheticQuarantined)
	assertSyntheticAccountAbsent(t, router, "synthetic-account-b", 12)
	router.setStatus("synthetic-account-b", syntheticEligible)
	assertSyntheticAccountPresent(t, router, "synthetic-account-b", 12)
	router.setStatus("synthetic-account-c", syntheticRemoved)
	assertSyntheticAccountAbsent(t, router, "synthetic-account-c", 12)
	router.add("synthetic-account-d")
	assertSyntheticAccountPresent(t, router, "synthetic-account-d", 16)
	router.restart("synthetic-v2")
	if router.snapshot().ConfigVersion != "synthetic-v2" {
		t.Fatal("synthetic restart did not activate the staged version")
	}
	router.rollback(baseline)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var workers sync.WaitGroup
	var successful atomic.Int64
	var unavailable atomic.Int64
	var invalid atomic.Int64
	for worker := 0; worker < 12; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				selection, err := router.selectRoute(syntheticRouteRequest{})
				if errors.Is(err, errSyntheticRouterUnavailable) {
					unavailable.Add(1)
					continue
				}
				if err != nil || !strings.HasPrefix(selection.Account, "synthetic-account-") {
					invalid.Add(1)
					continue
				}
				successful.Add(1)
			}
		}()
	}

	router.add("synthetic-account-d")
	router.setStatus("synthetic-account-b", syntheticQuarantined)
	router.setStatus("synthetic-account-c", syntheticRemoved)
	router.setStatus("synthetic-account-b", syntheticEligible)
	router.stop()
	unavailableDeadline := time.Now().Add(2 * time.Second)
	for unavailable.Load() == 0 && time.Now().Before(unavailableDeadline) {
		time.Sleep(time.Millisecond)
	}
	router.restart("synthetic-v2")
	router.rollback(baseline)

	deadline := time.Now().Add(2 * time.Second)
	for successful.Load() < 500 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	workers.Wait()
	if successful.Load() < 500 || unavailable.Load() == 0 || invalid.Load() != 0 {
		t.Fatalf("synthetic lifecycle load mismatch: successful=%d invalid=%d unavailable=%d", successful.Load(), invalid.Load(), unavailable.Load())
	}
	rolledBack := router.snapshot()
	if rolledBack.ConfigVersion != baseline.ConfigVersion || fmt.Sprint(rolledBack.Order) != fmt.Sprint(baseline.Order) {
		t.Fatal("synthetic rollback did not restore the baseline")
	}
	for account, state := range baseline.Status {
		if rolledBack.Status[account] != state {
			t.Fatalf("synthetic rollback state mismatch for %s", account)
		}
	}
}

func assertSyntheticAccountAbsent(t *testing.T, router *syntheticRouter, account string, selections int) {
	t.Helper()
	for index := 0; index < selections; index++ {
		selection, err := router.selectRoute(syntheticRouteRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if selection.Account == account {
			t.Fatalf("ineligible synthetic account %s was selected", account)
		}
	}
}

func assertSyntheticAccountPresent(t *testing.T, router *syntheticRouter, account string, selections int) {
	t.Helper()
	for index := 0; index < selections; index++ {
		selection, err := router.selectRoute(syntheticRouteRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if selection.Account == account {
			return
		}
	}
	t.Fatalf("eligible synthetic account %s was not selected", account)
}
