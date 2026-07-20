package gateway

import (
	"context"
	"net/http"
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
