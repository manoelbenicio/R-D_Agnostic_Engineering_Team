package gateway

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestCoordinator(t *testing.T) *Coordinator {
	t.Helper()
	coordinator, err := NewCoordinator(RetryPolicy{
		MaxAttempts:      3,
		EndToEndDeadline: 30 * time.Second,
		PreCommitOnly:    true,
		MinimumBackoff:   0,
		MaximumBackoff:   0,
		Jitter:           false,
	})
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}
	return coordinator
}

func retryableUpstreamError() *GatewayError {
	signal := FailureSignal{StatusCode: http.StatusInternalServerError}
	return ClassifyFailure(signal).AsError("test.dispatch", signal)
}

func terminalAuthError() *GatewayError {
	signal := FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeRefreshRevoked}
	return ClassifyFailure(signal).AsError("test.dispatch", signal)
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestCoordinatorRetriesBeforeFirstOutput(t *testing.T) {
	coordinator := newTestCoordinator(t)
	var calls int
	result, err := coordinator.Execute(context.Background(), "req-retry", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls++
		if attempt == 1 {
			return AttemptResult{}, retryableUpstreamError()
		}
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if result.Attempts != 2 || calls != 2 {
		t.Fatalf("expected 2 attempts, got result=%d calls=%d", result.Attempts, calls)
	}
	if result.Deduplicated {
		t.Fatal("leader must not be marked deduplicated")
	}
}

func TestCoordinatorDoesNotReplayAfterPartialOutput(t *testing.T) {
	coordinator := newTestCoordinator(t)
	var calls int
	result, err := coordinator.Execute(context.Background(), "req-partial-output", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls++
		return AttemptResult{OutputCommitted: true}, retryableUpstreamError()
	})
	if err == nil {
		t.Fatal("expected the partial-output failure to surface")
	}
	if result.Attempts != 1 || calls != 1 {
		t.Fatalf("request was replayed after partial output: attempts=%d calls=%d", result.Attempts, calls)
	}
}

func TestCoordinatorDoesNotReplayAfterToolAction(t *testing.T) {
	coordinator := newTestCoordinator(t)
	var calls int
	result, err := coordinator.Execute(context.Background(), "req-tool-action", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls++
		return AttemptResult{ToolActionCommitted: true}, retryableUpstreamError()
	})
	if err == nil {
		t.Fatal("expected the tool-action failure to surface")
	}
	if result.Attempts != 1 || calls != 1 {
		t.Fatalf("non-idempotent tool action was replayed: attempts=%d calls=%d", result.Attempts, calls)
	}
}

func TestCoordinatorDoesNotRetryTerminalFailure(t *testing.T) {
	coordinator := newTestCoordinator(t)
	var calls int
	result, err := coordinator.Execute(context.Background(), "req-terminal-auth", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls++
		return AttemptResult{}, terminalAuthError()
	})
	if !IsErrorClass(err, ErrorAuthentication) {
		t.Fatalf("expected authentication failure, got %v", err)
	}
	if result.Attempts != 1 || calls != 1 {
		t.Fatalf("non-retryable error was retried: attempts=%d calls=%d", result.Attempts, calls)
	}
}

func TestCoordinatorIsBoundedByMaxAttempts(t *testing.T) {
	coordinator := newTestCoordinator(t)
	var calls atomic.Int64
	result, err := coordinator.Execute(context.Background(), "req-exhaust", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls.Add(1)
		return AttemptResult{}, retryableUpstreamError()
	})
	if err == nil {
		t.Fatal("expected exhausted retries to surface an error")
	}
	if result.Attempts != 3 || calls.Load() != 3 {
		t.Fatalf("retry bound not honored: attempts=%d calls=%d", result.Attempts, calls.Load())
	}
}

func TestCoordinatorDeduplicatesCompletedRequest(t *testing.T) {
	coordinator := newTestCoordinator(t)
	first, err := coordinator.Execute(context.Background(), "req-dedup", func(ctx context.Context, attempt int) (AttemptResult, error) {
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil || first.Deduplicated {
		t.Fatalf("leader outcome unexpected: result=%+v err=%v", first, err)
	}
	var reran atomic.Bool
	second, err := coordinator.Execute(context.Background(), "req-dedup", func(ctx context.Context, attempt int) (AttemptResult, error) {
		reran.Store(true)
		return AttemptResult{}, nil
	})
	if err != nil {
		t.Fatalf("dedup returned error: %v", err)
	}
	if !second.Deduplicated || second.Attempts != 0 || reran.Load() {
		t.Fatalf("completed request was re-executed: result=%+v reran=%v", second, reran.Load())
	}
}

func TestCoordinatorDeduplicatesConcurrentInFlightRequests(t *testing.T) {
	coordinator := newTestCoordinator(t)
	started := make(chan struct{})
	release := make(chan struct{})
	leaderDone := make(chan ExecutionResult, 1)
	go func() {
		result, _ := coordinator.Execute(context.Background(), "req-inflight", func(ctx context.Context, attempt int) (AttemptResult, error) {
			close(started)
			<-release
			return AttemptResult{OutputCommitted: true}, nil
		})
		leaderDone <- result
	}()
	<-started

	const followers = 64
	followerResults := make(chan ExecutionResult, followers)
	var executed atomic.Int64
	for i := 0; i < followers; i++ {
		go func() {
			result, _ := coordinator.Execute(context.Background(), "req-inflight", func(ctx context.Context, attempt int) (AttemptResult, error) {
				executed.Add(1)
				return AttemptResult{}, nil
			})
			followerResults <- result
		}()
	}
	waitFor(t, func() bool { return coordinator.FollowerSlots() == followers })
	if coordinator.ActiveSlots() != 1 {
		t.Fatalf("expected exactly one active leader, got %d", coordinator.ActiveSlots())
	}
	close(release)

	if leader := <-leaderDone; leader.Deduplicated || leader.Attempts != 1 {
		t.Fatalf("leader outcome unexpected: %+v", leader)
	}
	for i := 0; i < followers; i++ {
		if result := <-followerResults; !result.Deduplicated || result.Attempts != 0 {
			t.Fatalf("follower executed independently: %+v", result)
		}
	}
	if executed.Load() != 0 {
		t.Fatalf("deduplicated followers executed the attempt %d times", executed.Load())
	}
	waitFor(t, func() bool { return coordinator.ActiveSlots() == 0 && coordinator.FollowerSlots() == 0 })
}

func TestCoordinatorCancellationReleasesSlotAndDoesNotPoisonRequestID(t *testing.T) {
	coordinator := newTestCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := coordinator.Execute(ctx, "req-cancel", func(runCtx context.Context, attempt int) (AttemptResult, error) {
			close(started)
			<-runCtx.Done()
			return AttemptResult{}, runCtx.Err()
		})
		done <- err
	}()
	<-started
	waitFor(t, func() bool { return coordinator.ActiveSlots() == 1 })
	cancel()
	if err := <-done; !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("expected cancellation class, got %v", err)
	}
	waitFor(t, func() bool { return coordinator.ActiveSlots() == 0 })

	// A pre-output cancellation must not poison the id: a fresh submission runs.
	var reran atomic.Bool
	result, err := coordinator.Execute(context.Background(), "req-cancel", func(ctx context.Context, attempt int) (AttemptResult, error) {
		reran.Store(true)
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil {
		t.Fatalf("resubmission after pre-output cancel failed: %v", err)
	}
	if result.Deduplicated || result.Attempts != 1 || !reran.Load() {
		t.Fatalf("pre-output cancellation incorrectly poisoned dedup: result=%+v reran=%v", result, reran.Load())
	}
}

func TestCoordinatorIsBoundedByEndToEndDeadline(t *testing.T) {
	coordinator, err := NewCoordinator(RetryPolicy{
		MaxAttempts:      5,
		EndToEndDeadline: 40 * time.Millisecond,
		PreCommitOnly:    true,
		MinimumBackoff:   0,
		MaximumBackoff:   0,
	})
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}
	var calls atomic.Int64
	result, execErr := coordinator.Execute(context.Background(), "req-deadline", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls.Add(1)
		<-ctx.Done()
		return AttemptResult{}, ctx.Err()
	})
	if !IsErrorClass(execErr, ErrorTimeout) {
		t.Fatalf("expected end-to-end deadline timeout, got %v", execErr)
	}
	if result.Attempts != 1 || calls.Load() != 1 {
		t.Fatalf("deadline bound not honored: attempts=%d calls=%d", result.Attempts, calls.Load())
	}
}

func TestCoordinatorRejectsInvalidPolicyAndArguments(t *testing.T) {
	if _, err := NewCoordinator(RetryPolicy{MaxAttempts: 0, EndToEndDeadline: time.Second, PreCommitOnly: true}); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for zero attempts, got %v", err)
	}
	if _, err := NewCoordinator(RetryPolicy{MaxAttempts: 2, EndToEndDeadline: time.Second, PreCommitOnly: false}); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for non-pre-commit policy, got %v", err)
	}
	coordinator := newTestCoordinator(t)
	if _, err := coordinator.Execute(context.Background(), "  ", func(context.Context, int) (AttemptResult, error) { return AttemptResult{}, nil }); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid request for blank id, got %v", err)
	}
	if _, err := coordinator.Execute(context.Background(), "req", nil); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid request for nil attempt, got %v", err)
	}
}

// Guard against slot accounting drift under a burst of independent successful
// executions.
func TestCoordinatorDrainsSlotsUnderConcurrentDistinctRequests(t *testing.T) {
	coordinator := newTestCoordinator(t)
	const requests = 200
	var workers sync.WaitGroup
	for i := 0; i < requests; i++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			_, _ = coordinator.Execute(context.Background(), "req-"+string(rune('A'+id%26))+string(rune('0'+id/26)), func(context.Context, int) (AttemptResult, error) {
				return AttemptResult{OutputCommitted: true}, nil
			})
		}(i)
	}
	workers.Wait()
	if coordinator.ActiveSlots() != 0 || coordinator.FollowerSlots() != 0 {
		t.Fatalf("slots leaked: active=%d followers=%d", coordinator.ActiveSlots(), coordinator.FollowerSlots())
	}
}

// --- W2 remediation regression tests (defects 1, 4, 5, 7) ---

func TestCoordinatorAdmissionCommitsSlotAndDedupBeforeAnyAttempt(t *testing.T) {
	// Defect 1: no attempt may begin before the slot + dedup registration are
	// committed. The leader's own attempt observes an active slot, and a
	// duplicate submitted while it runs is deduplicated (proving registration
	// was committed before the attempt started).
	coordinator := newTestCoordinator(t)
	started := make(chan struct{})
	release := make(chan struct{})
	observed := make(chan struct {
		active   int64
		inflight int
	}, 1)
	leaderDone := make(chan ExecutionResult, 1)
	go func() {
		result, _ := coordinator.Execute(context.Background(), "req-admit", func(ctx context.Context, attempt int) (AttemptResult, error) {
			observed <- struct {
				active   int64
				inflight int
			}{active: coordinator.ActiveSlots(), inflight: coordinator.InFlightCount()}
			close(started)
			<-release
			return AttemptResult{OutputCommitted: true}, nil
		})
		leaderDone <- result
	}()
	<-started
	seen := <-observed
	if seen.active < 1 || seen.inflight < 1 {
		t.Fatalf("attempt began before admission committed: active=%d inflight=%d", seen.active, seen.inflight)
	}
	// Duplicate arriving mid-flight must be deduplicated, not a second leader.
	dupDone := make(chan ExecutionResult, 1)
	var dupExecuted atomic.Bool
	go func() {
		result, _ := coordinator.Execute(context.Background(), "req-admit", func(context.Context, int) (AttemptResult, error) {
			dupExecuted.Store(true)
			return AttemptResult{}, nil
		})
		dupDone <- result
	}()
	waitFor(t, func() bool { return coordinator.FollowerSlots() == 1 })
	close(release)
	if leader := <-leaderDone; leader.Attempts != 1 || leader.Deduplicated {
		t.Fatalf("leader outcome unexpected: %+v", leader)
	}
	if dup := <-dupDone; !dup.Deduplicated || dup.Attempts != 0 || dupExecuted.Load() {
		t.Fatalf("duplicate was not deduplicated: %+v executed=%v", dup, dupExecuted.Load())
	}
	waitFor(t, func() bool { return coordinator.ActiveSlots() == 0 && coordinator.FollowerSlots() == 0 })
}

func TestCoordinatorRetainsAndReplaysNonRetryableTerminalFailure(t *testing.T) {
	// Defect 5: a deterministic non-retryable terminal failure (e.g. revoked
	// refresh token) must be retained and replayed to concurrent and later
	// duplicates, not re-executed.
	coordinator := newTestCoordinator(t)
	started := make(chan struct{})
	release := make(chan struct{})
	var leaderCalls atomic.Int64
	leaderDone := make(chan error, 1)
	go func() {
		_, err := coordinator.Execute(context.Background(), "req-nonretry", func(ctx context.Context, attempt int) (AttemptResult, error) {
			leaderCalls.Add(1)
			close(started)
			<-release
			return AttemptResult{}, terminalAuthError()
		})
		leaderDone <- err
	}()
	<-started
	// Concurrent follower joins the in-flight non-retryable request.
	followerDone := make(chan error, 1)
	var followerExecuted atomic.Bool
	go func() {
		_, err := coordinator.Execute(context.Background(), "req-nonretry", func(context.Context, int) (AttemptResult, error) {
			followerExecuted.Store(true)
			return AttemptResult{}, nil
		})
		followerDone <- err
	}()
	waitFor(t, func() bool { return coordinator.FollowerSlots() == 1 })
	close(release)

	if err := <-leaderDone; !IsErrorClass(err, ErrorAuthentication) {
		t.Fatalf("leader error class mismatch: %v", err)
	}
	if err := <-followerDone; !IsErrorClass(err, ErrorAuthentication) {
		t.Fatalf("concurrent follower did not inherit terminal failure: %v", err)
	}
	if followerExecuted.Load() {
		t.Fatal("concurrent follower re-executed a terminal failure")
	}
	// Later duplicate is deduplicated to the retained terminal failure.
	var lateExecuted atomic.Bool
	result, err := coordinator.Execute(context.Background(), "req-nonretry", func(context.Context, int) (AttemptResult, error) {
		lateExecuted.Store(true)
		return AttemptResult{}, nil
	})
	if !IsErrorClass(err, ErrorAuthentication) || !result.Deduplicated || result.Attempts != 0 || lateExecuted.Load() {
		t.Fatalf("later duplicate re-executed terminal failure: result=%+v err=%v executed=%v", result, err, lateExecuted.Load())
	}
	if leaderCalls.Load() != 1 {
		t.Fatalf("terminal failure executed %d times, want 1", leaderCalls.Load())
	}
}

func TestCoordinatorRetryableExhaustionIsNotTerminal(t *testing.T) {
	// Defect 5 complement: an exhausted *retryable* failure is transient, not a
	// deterministic terminal outcome, so a fresh resubmission is allowed.
	coordinator := newTestCoordinator(t)
	var calls atomic.Int64
	_, err := coordinator.Execute(context.Background(), "req-transient", func(context.Context, int) (AttemptResult, error) {
		calls.Add(1)
		return AttemptResult{}, retryableUpstreamError()
	})
	if err == nil {
		t.Fatal("expected exhausted retry error")
	}
	if coordinator.TerminalCount() != 0 {
		t.Fatalf("transient exhaustion was retained as terminal: count=%d", coordinator.TerminalCount())
	}
	// Resubmission runs again (not deduplicated to a terminal record).
	result, err := coordinator.Execute(context.Background(), "req-transient", func(context.Context, int) (AttemptResult, error) {
		calls.Add(1)
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil || result.Deduplicated {
		t.Fatalf("resubmission after transient exhaustion was blocked: result=%+v err=%v", result, err)
	}
}

func TestCoordinatorTerminalStateIsBounded(t *testing.T) {
	// Defect 4 (coordinator): terminal state is bounded by max size (oldest-first
	// eviction), TTL expiry, and explicit Forget, and returns to zero.
	clock := time.Unix(1_700_000_000, 0)
	coordinator, err := NewCoordinatorFromConfig(CoordinatorConfig{
		Policy:     RetryPolicy{MaxAttempts: 1, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true},
		DedupTTL:   time.Minute,
		MaxTracked: 4,
		now:        func() time.Time { return clock },
	})
	if err != nil {
		t.Fatalf("NewCoordinatorFromConfig: %v", err)
	}
	success := func(context.Context, int) (AttemptResult, error) { return AttemptResult{OutputCommitted: true}, nil }
	for i := 0; i < 20; i++ {
		if _, err := coordinator.Execute(context.Background(), "req-"+strconv.Itoa(i), success); err != nil {
			t.Fatalf("execute %d: %v", i, err)
		}
	}
	if got := coordinator.TerminalCount(); got > 4 {
		t.Fatalf("terminal state unbounded: count=%d want <=4", got)
	}
	// Explicit Forget drops a record.
	before := coordinator.TerminalCount()
	coordinator.Forget("req-19")
	if coordinator.TerminalCount() >= before && before > 0 {
		t.Fatalf("Forget did not drop a terminal record: before=%d after=%d", before, coordinator.TerminalCount())
	}
	// TTL expiry reclaims everything once the clock advances past DedupTTL.
	clock = clock.Add(2 * time.Minute)
	if got := coordinator.TerminalCount(); got != 0 {
		t.Fatalf("expired terminal records not reclaimed: count=%d", got)
	}
	if coordinator.ActiveSlots() != 0 || coordinator.InFlightCount() != 0 {
		t.Fatalf("in-flight state leaked: active=%d inflight=%d", coordinator.ActiveSlots(), coordinator.InFlightCount())
	}
}

func TestCoordinatorTerminalEvictionNeverDropsInFlightEntry(t *testing.T) {
	// Defect 4: flooding terminal to capacity while a request is in-flight must
	// never evict the in-flight entry, and the in-flight request completes and
	// records normally.
	clock := time.Unix(1_700_000_000, 0)
	coordinator, err := NewCoordinatorFromConfig(CoordinatorConfig{
		Policy:     RetryPolicy{MaxAttempts: 1, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true},
		DedupTTL:   time.Hour,
		MaxTracked: 3,
		now:        func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	leaderDone := make(chan ExecutionResult, 1)
	go func() {
		result, _ := coordinator.Execute(context.Background(), "req-inflight", func(ctx context.Context, attempt int) (AttemptResult, error) {
			close(started)
			<-release
			return AttemptResult{OutputCommitted: true}, nil
		})
		leaderDone <- result
	}()
	<-started
	if coordinator.InFlightCount() != 1 {
		t.Fatalf("expected 1 in-flight, got %d", coordinator.InFlightCount())
	}
	// Flood terminal well past capacity while the leader is in-flight.
	success := func(context.Context, int) (AttemptResult, error) { return AttemptResult{OutputCommitted: true}, nil }
	for i := 0; i < 10; i++ {
		if _, err := coordinator.Execute(context.Background(), "flood-"+strconv.Itoa(i), success); err != nil {
			t.Fatalf("flood %d: %v", i, err)
		}
	}
	// The in-flight entry is still tracked (never evicted by terminal bound).
	if coordinator.InFlightCount() != 1 {
		t.Fatalf("in-flight entry was disturbed by terminal eviction: %d", coordinator.InFlightCount())
	}
	if coordinator.TerminalCount() > 3 {
		t.Fatalf("terminal bound violated: %d", coordinator.TerminalCount())
	}
	close(release)
	if result := <-leaderDone; result.Attempts != 1 || result.Deduplicated {
		t.Fatalf("in-flight leader outcome unexpected: %+v", result)
	}
	waitFor(t, func() bool { return coordinator.ActiveSlots() == 0 })
}

func TestCoordinatorPreservesDeadlineVersusCancellation(t *testing.T) {
	// Defect 7: an end-to-end deadline must surface as timeout, a caller
	// cancellation as cancelled; the two must not collapse.
	deadlineCoordinator, err := NewCoordinator(RetryPolicy{
		MaxAttempts: 3, EndToEndDeadline: 30 * time.Millisecond, PreCommitOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, deadlineErr := deadlineCoordinator.Execute(context.Background(), "req-deadline", func(ctx context.Context, attempt int) (AttemptResult, error) {
		<-ctx.Done()
		return AttemptResult{}, ctx.Err()
	})
	if !IsErrorClass(deadlineErr, ErrorTimeout) {
		t.Fatalf("deadline collapsed: got %v want timeout", deadlineErr)
	}

	cancelCoordinator := newTestCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := cancelCoordinator.Execute(ctx, "req-cancel2", func(runCtx context.Context, attempt int) (AttemptResult, error) {
			close(started)
			<-runCtx.Done()
			return AttemptResult{}, runCtx.Err()
		})
		done <- err
	}()
	<-started
	cancel()
	if cancelErr := <-done; !IsErrorClass(cancelErr, ErrorCancelled) {
		t.Fatalf("cancellation collapsed: got %v want cancelled", cancelErr)
	}
	// A deadline outcome is transient (not retained); confirm no terminal record.
	if deadlineCoordinator.TerminalCount() != 0 {
		t.Fatalf("deadline outcome retained as terminal: %d", deadlineCoordinator.TerminalCount())
	}
}

// --- W2 second-pass regression tests (F7 callback DeadlineExceeded vs Canceled) ---

func TestCoordinatorCallbackDeadlineExceededWhileContextActiveStaysTimeout(t *testing.T) {
	// F7: if the callback returns context.DeadlineExceeded while execCtx is
	// still active (e.g. an inner per-attempt deadline), the outcome must be a
	// timeout, not collapsed into cancellation.
	coordinator := newTestCoordinator(t)
	var calls atomic.Int64
	_, err := coordinator.Execute(context.Background(), "req-inner-deadline", func(ctx context.Context, attempt int) (AttemptResult, error) {
		calls.Add(1)
		// execCtx is still active (30s end-to-end); the callback surfaces its
		// own DeadlineExceeded directly.
		return AttemptResult{}, context.DeadlineExceeded
	})
	if !IsErrorClass(err, ErrorTimeout) {
		t.Fatalf("callback DeadlineExceeded collapsed: got %v want timeout", err)
	}
	// A timeout is transient, not retained as terminal.
	if coordinator.TerminalCount() != 0 {
		t.Fatalf("timeout outcome retained as terminal: %d", coordinator.TerminalCount())
	}
}

func TestCoordinatorCallbackCanceledWhileContextActiveStaysCancelled(t *testing.T) {
	// F7 complement: a callback returning context.Canceled while execCtx is
	// active is a cancellation.
	coordinator := newTestCoordinator(t)
	_, err := coordinator.Execute(context.Background(), "req-inner-cancel", func(ctx context.Context, attempt int) (AttemptResult, error) {
		return AttemptResult{}, context.Canceled
	})
	if !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("callback Canceled misclassified: got %v want cancelled", err)
	}
	if coordinator.TerminalCount() != 0 {
		t.Fatalf("cancellation retained as terminal: %d", coordinator.TerminalCount())
	}
}

func TestCoordinatorFollowerPreservesDeadlineVersusCancellation(t *testing.T) {
	// F7 (follower path): a follower waiting on an in-flight leader reports its
	// own context outcome distinctly — deadline as timeout, cancel as cancelled.
	run := func(t *testing.T, followerCtx context.Context, wantTimeout bool) {
		t.Helper()
		coordinator := newTestCoordinator(t)
		started := make(chan struct{})
		release := make(chan struct{})
		leaderDone := make(chan struct{})
		go func() {
			_, _ = coordinator.Execute(context.Background(), "req-follow", func(ctx context.Context, attempt int) (AttemptResult, error) {
				close(started)
				<-release
				return AttemptResult{OutputCommitted: true}, nil
			})
			close(leaderDone)
		}()
		<-started
		followerDone := make(chan error, 1)
		go func() {
			_, err := coordinator.Execute(followerCtx, "req-follow", func(context.Context, int) (AttemptResult, error) {
				return AttemptResult{}, nil
			})
			followerDone <- err
		}()
		waitFor(t, func() bool { return coordinator.FollowerSlots() == 1 })
		err := <-followerDone
		if wantTimeout && !IsErrorClass(err, ErrorTimeout) {
			t.Fatalf("follower deadline collapsed: got %v want timeout", err)
		}
		if !wantTimeout && !IsErrorClass(err, ErrorCancelled) {
			t.Fatalf("follower cancel misclassified: got %v want cancelled", err)
		}
		close(release)
		<-leaderDone
	}

	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		run(t, ctx, true)
	})
	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		// Cancel shortly after the follower joins.
		go func() { time.Sleep(20 * time.Millisecond); cancel() }()
		defer cancel()
		run(t, ctx, false)
	})
}
