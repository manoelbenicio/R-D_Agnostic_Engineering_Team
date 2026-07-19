package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const priority0LeakSentinel = "synthetic-refresh-secret-content"

func TestPriority0CredentialSourceFailureIsBoundedAuthentication(t *testing.T) {
	credential := &syntheticCredentialSource{err: errors.New(priority0LeakSentinel)}
	var transportCalls int
	client, err := NewClient(ClientOptions{
		Gateway:    testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:  EndpointSet{Liveness: "/health/live", Readiness: "/health/ready"},
		Credential: credential,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			transportCalls++
			return syntheticResponse(request, http.StatusOK, "application/json", `{}`), nil
		})},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.CheckReadiness(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorAuthentication) {
		t.Fatalf("credential source failure class=%v want=%s", err, ErrorAuthentication)
	}
	if strings.Contains(err.Error(), priority0LeakSentinel) {
		t.Fatal("credential source failure detail escaped the bounded gateway error")
	}
	if credential.calls.Load() != 1 || transportCalls != 0 {
		t.Fatalf("credential failure crossed transport boundary: source_calls=%d transport_calls=%d", credential.calls.Load(), transportCalls)
	}
}

func TestPriority0RegistryHeaderBodyVersionMismatchIsProtocolError(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := syntheticResponse(request, http.StatusOK, "application/json", `{"object":"list","registry_version":"synthetic-body-v1","data":[]}`)
		response.Header.Set(HeaderRegistryVersion, "synthetic-header-v2")
		return response, nil
	})
	client, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/health/ready"},
		Credential:     &syntheticCredentialSource{},
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.FetchModels(context.Background(), testCorrelation())
	if !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("registry version mismatch class=%v want=%s", err, ErrorProtocol)
	}
	if strings.Contains(err.Error(), "synthetic-body-v1") || strings.Contains(err.Error(), "synthetic-header-v2") {
		t.Fatal("registry version mismatch echoed bounded upstream details")
	}
}

func TestPriority0CredentialAndQuotaSignalsAreSafeAndFailClosed(t *testing.T) {
	type transition struct {
		name          string
		status        int
		quota         QuotaState
		circuit       CircuitState
		wantClass     ErrorClass
		wantRetryable bool
		wantSteady    bool
	}

	transitions := []transition{
		{
			name: "quota-limited", status: http.StatusTooManyRequests,
			quota: QuotaLimited, circuit: CircuitHalfOpen,
			wantClass: ErrorRateLimited, wantRetryable: true,
		},
		{
			name: "authentication-failed", status: http.StatusUnauthorized,
			quota: QuotaExhausted, circuit: CircuitOpen,
			wantClass: ErrorAuthentication,
		},
		{
			name: "quota-recovered", status: http.StatusOK,
			quota: QuotaAvailable, circuit: CircuitClosed,
			wantSteady: true,
		},
	}

	for _, transition := range transitions {
		t.Run(transition.name, func(t *testing.T) {
			headers := make(http.Header)
			headers.Set(HeaderOmniRouteRequestID, "omni-refresh-1")
			headers.Set(HeaderActualModel, "synthetic/model")
			headers.Set(HeaderActualRoute, "route-refresh")
			headers.Set(HeaderAccountID, "account-"+priority0LeakSentinel)
			headers.Set(HeaderConnectionID, "connection-"+priority0LeakSentinel)
			headers.Set(HeaderSelectionReason, string(SelectionRetry))
			headers.Set(HeaderRetryCount, "1")
			headers.Set(HeaderFallbackUsed, "false")
			headers.Set(HeaderQuotaState, string(transition.quota))
			headers.Set(HeaderCircuitState, string(transition.circuit))
			telemetry, err := ParseTelemetryHeaders(headers)
			if err != nil {
				t.Fatalf("parse synthetic refresh telemetry: %v", err)
			}
			if telemetry.Quota != transition.quota || telemetry.Circuit != transition.circuit {
				t.Fatalf("refresh transition mismatch: quota=%q circuit=%q", telemetry.Quota, telemetry.Circuit)
			}
			if telemetry.PseudonymousAccount == "" || telemetry.PseudonymousConnection == "" {
				t.Fatal("refresh transition omitted pseudonymous routing metadata")
			}
			if strings.Contains(fmt.Sprint(telemetry), priority0LeakSentinel) {
				t.Fatal("refresh telemetry exposed synthetic account or connection input")
			}

			producer, err := NewSteadyStateProducer(func() (bool, bool, bool, error) {
				ready := telemetry.Quota == QuotaAvailable && telemetry.Circuit == CircuitClosed
				return ready, telemetry.Circuit == CircuitClosed, telemetry.Quota == QuotaAvailable, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			facts, err := producer.SteadyState()
			if err != nil || facts.Steady != transition.wantSteady || facts.Ready != transition.wantSteady {
				t.Fatalf("refresh readiness transition mismatch: facts=%+v err=%v", facts, err)
			}

			if transition.status >= 400 {
				response := &http.Response{
					StatusCode: transition.status,
					Header:     headers,
					Body:       io.NopCloser(strings.NewReader(priority0LeakSentinel)),
				}
				classified := classifyStatus("credential.refresh", response)
				if classified.Class != transition.wantClass || classified.Retryable != transition.wantRetryable {
					t.Fatalf("refresh classification mismatch: %+v", classified)
				}
				if strings.Contains(classified.Error(), priority0LeakSentinel) {
					t.Fatal("refresh classification echoed response content")
				}
				_ = response.Body.Close()
			}
		})
	}

	malformed := make(http.Header)
	malformed.Set(HeaderQuotaState, priority0LeakSentinel)
	_, err := ParseTelemetryHeaders(malformed)
	if !IsErrorClass(err, ErrorProtocol) || strings.Contains(err.Error(), priority0LeakSentinel) {
		t.Fatalf("malformed refresh signal was not safely refused: %v", err)
	}
}

func TestPriority0QuarantineCooldownReentryUnderConcurrentRoundRobin(t *testing.T) {
	router := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c")

	router.setStatus("synthetic-account-b", syntheticQuarantined)
	assertConcurrentStrictRoundRobin(t, router, 96, map[string]bool{
		"synthetic-account-a": true,
		"synthetic-account-c": true,
	})

	router.setStatus("synthetic-account-b", syntheticCooldown)
	assertConcurrentStrictRoundRobin(t, router, 96, map[string]bool{
		"synthetic-account-a": true,
		"synthetic-account-c": true,
	})

	router.setStatus("synthetic-account-b", syntheticEligible)
	selections := assertConcurrentStrictRoundRobin(t, router, ninetySelections, map[string]bool{
		"synthetic-account-a": true,
		"synthetic-account-b": true,
		"synthetic-account-c": true,
	})
	counts := map[string]int{}
	for _, selection := range selections {
		counts[selection.Account]++
	}
	for account := range counts {
		if counts[account] != ninetySelections/3 {
			t.Fatalf("re-entered rotation was imbalanced: account=%s count=%d", account, counts[account])
		}
	}
}

const ninetySelections = 90

func assertConcurrentStrictRoundRobin(t *testing.T, router *syntheticRouter, count int, eligible map[string]bool) []syntheticRouteSelection {
	t.Helper()
	before := router.snapshot()
	results := make(chan syntheticRouteSelection, count)
	errors := make(chan error, count)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index := 0; index < count; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			selection, err := router.selectRoute(syntheticRouteRequest{})
			if err != nil {
				errors <- err
				return
			}
			results <- selection
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent rotation failed: %v", err)
	}

	selections := make([]syntheticRouteSelection, 0, count)
	for selection := range results {
		if !eligible[selection.Account] {
			t.Fatalf("ineligible account selected during lifecycle transition: %s", selection.Account)
		}
		selections = append(selections, selection)
	}
	if len(selections) != count {
		t.Fatalf("concurrent selections=%d want=%d", len(selections), count)
	}
	sort.Slice(selections, func(i, j int) bool { return selections[i].Sequence < selections[j].Sequence })

	cursor := before.Cursor
	for offset, selection := range selections {
		found := false
		for scan := 0; scan < len(before.Order); scan++ {
			index := (cursor + scan) % len(before.Order)
			account := before.Order[index]
			if !eligible[account] {
				continue
			}
			if selection.Account != account {
				t.Fatalf("strict round-robin mismatch at %d: got=%s want=%s", offset, selection.Account, account)
			}
			cursor = (index + 1) % len(before.Order)
			found = true
			break
		}
		if !found {
			t.Fatal("no eligible account available in expected rotation")
		}
	}
	return selections
}

func TestPriority0PartialToolTurnIsTerminalAcrossDedupAndCancellation(t *testing.T) {
	harness := newSyntheticRetryHarness()
	started := make(chan struct{})
	release := make(chan struct{})
	leaderDone := make(chan struct {
		result syntheticExecution
		err    error
	}, 1)
	go func() {
		result, err := harness.execute(context.Background(), "synthetic-partial-tool-turn", []syntheticAttempt{
			{
				Failure:         syntheticTimeout,
				OutputCommitted: true,
				Started:         started,
				Release:         release,
			},
			{},
		})
		leaderDone <- struct {
			result syntheticExecution
			err    error
		}{result: result, err: err}
	}()
	<-started

	followerCtx, cancelFollower := context.WithCancel(context.Background())
	followerDone := make(chan error, 1)
	go func() {
		_, err := harness.execute(followerCtx, "synthetic-partial-tool-turn", []syntheticAttempt{{}})
		followerDone <- err
	}()
	deadline := time.Now().Add(time.Second)
	for harness.followers.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if harness.followers.Load() != 1 {
		t.Fatal("deduplicated follower did not join the in-flight tool turn")
	}
	cancelFollower()
	if err := <-followerDone; !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("follower cancellation mismatch: %v", err)
	}
	close(release)
	leader := <-leaderDone
	if leader.result.Attempts != 1 || !IsErrorClass(leader.err, ErrorTimeout) {
		t.Fatalf("partial tool turn replay boundary mismatch: result=%+v err=%v", leader.result, leader.err)
	}
	if harness.active.Load() != 0 || harness.followers.Load() != 0 {
		t.Fatalf("partial tool turn leaked slots: active=%d followers=%d", harness.active.Load(), harness.followers.Load())
	}

	duplicate, err := harness.execute(context.Background(), "synthetic-partial-tool-turn", []syntheticAttempt{{}})
	if !duplicate.Deduplicated || duplicate.Attempts != 0 || !IsErrorClass(err, ErrorTimeout) {
		t.Fatalf("partial tool turn was replayed after terminal output: result=%+v err=%v", duplicate, err)
	}

	cancelCtx, cancelLeader := context.WithCancel(context.Background())
	cancelStarted := make(chan struct{})
	cancelRelease := make(chan struct{})
	cancelDone := make(chan error, 1)
	go func() {
		_, err := harness.execute(cancelCtx, "synthetic-cancel-before-output", []syntheticAttempt{{
			Started: cancelStarted,
			Release: cancelRelease,
		}})
		cancelDone <- err
	}()
	<-cancelStarted
	cancelLeader()
	if err := <-cancelDone; !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("leader cancellation mismatch: %v", err)
	}
	retry, err := harness.execute(context.Background(), "synthetic-cancel-before-output", []syntheticAttempt{{}})
	if err != nil || retry.Deduplicated || retry.Attempts != 1 {
		t.Fatalf("pre-output cancellation incorrectly poisoned dedup: result=%+v err=%v", retry, err)
	}
}
