package gateway

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Task 8.6 acceptance — Coordinator scenarios S1–S11: "safe retry before first
// output, no replay after partial output/tool action, request deduplication and
// prompt cancellation slot release." Deterministic, in-process, using the
// injected now/sleep hooks so no real time passes. Reuses the shared helpers
// (retryableUpstreamError, terminalAuthError, waitFor) rather than duplicating
// them. Component evidence only — not live OmniRoute proof.

func newTask86Coordinator(t *testing.T, clock *time.Time) *Coordinator {
	t.Helper()
	c, err := NewCoordinatorFromConfig(CoordinatorConfig{
		Policy:     RetryPolicy{MaxAttempts: 3, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true},
		DedupTTL:   time.Minute,
		MaxTracked: 8,
		now:        func() time.Time { return *clock },
		sleep:      func(ctx context.Context, _ time.Duration) error { return ctx.Err() }, // deterministic: no real backoff wait
	})
	if err != nil {
		t.Fatalf("NewCoordinatorFromConfig: %v", err)
	}
	return c
}

func TestTask86CoordinatorAcceptance(t *testing.T) {
	// S1: safe retry BEFORE first output — retryable failure then success.
	t.Run("S1_retry_before_first_output", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		var calls int
		res, err := c.Execute(context.Background(), "s1", func(context.Context, int) (AttemptResult, error) {
			calls++
			if calls == 1 {
				return AttemptResult{}, retryableUpstreamError()
			}
			return AttemptResult{OutputCommitted: true}, nil
		})
		if err != nil || res.Attempts != 2 {
			t.Fatalf("S1: attempts=%d err=%v want 2/nil", res.Attempts, err)
		}
	})

	// S2: NO replay after partial user-visible output.
	t.Run("S2_no_replay_after_partial_output", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		res, err := c.Execute(context.Background(), "s2", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{OutputCommitted: true}, retryableUpstreamError()
		})
		if err == nil || res.Attempts != 1 {
			t.Fatalf("S2: replayed after output: attempts=%d err=%v", res.Attempts, err)
		}
	})

	// S3: NO replay after a committed (potentially non-idempotent) tool action.
	t.Run("S3_no_replay_after_tool_action", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		res, err := c.Execute(context.Background(), "s3", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{ToolActionCommitted: true}, retryableUpstreamError()
		})
		if err == nil || res.Attempts != 1 {
			t.Fatalf("S3: replayed after tool action: attempts=%d err=%v", res.Attempts, err)
		}
	})

	// S4: non-retryable classified failure is not retried.
	t.Run("S4_non_retryable_no_retry", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		res, err := c.Execute(context.Background(), "s4", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{}, terminalAuthError()
		})
		if !IsErrorClass(err, ErrorAuthentication) || res.Attempts != 1 {
			t.Fatalf("S4: attempts=%d err=%v", res.Attempts, err)
		}
	})

	// S5: retries are bounded by MaxAttempts.
	t.Run("S5_retry_bounded_by_max_attempts", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		res, err := c.Execute(context.Background(), "s5", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{}, retryableUpstreamError()
		})
		if err == nil || res.Attempts != 3 {
			t.Fatalf("S5: attempts=%d want 3 err=%v", res.Attempts, err)
		}
	})

	// S6: bounded by the end-to-end deadline (surfaces as timeout).
	t.Run("S6_bounded_by_end_to_end_deadline", func(t *testing.T) {
		c, err := NewCoordinator(RetryPolicy{MaxAttempts: 5, EndToEndDeadline: 30 * time.Millisecond, PreCommitOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		res, execErr := c.Execute(context.Background(), "s6", func(ctx context.Context, _ int) (AttemptResult, error) {
			<-ctx.Done()
			return AttemptResult{}, ctx.Err()
		})
		if !IsErrorClass(execErr, ErrorTimeout) || res.Attempts != 1 {
			t.Fatalf("S6: attempts=%d err=%v", res.Attempts, execErr)
		}
	})

	// S7: deduplication of concurrent in-flight requests (one leader, N followers).
	t.Run("S7_dedup_concurrent_in_flight", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		started := make(chan struct{})
		release := make(chan struct{})
		leaderDone := make(chan ExecutionResult, 1)
		go func() {
			r, _ := c.Execute(context.Background(), "s7", func(context.Context, int) (AttemptResult, error) {
				close(started)
				<-release
				return AttemptResult{OutputCommitted: true}, nil
			})
			leaderDone <- r
		}()
		<-started
		const followers = 32
		var executed atomic.Int64
		done := make(chan ExecutionResult, followers)
		for i := 0; i < followers; i++ {
			go func() {
				r, _ := c.Execute(context.Background(), "s7", func(context.Context, int) (AttemptResult, error) {
					executed.Add(1)
					return AttemptResult{}, nil
				})
				done <- r
			}()
		}
		waitFor(t, func() bool { return c.FollowerSlots() == followers })
		close(release)
		if leader := <-leaderDone; leader.Deduplicated || leader.Attempts != 1 {
			t.Fatalf("S7 leader: %+v", leader)
		}
		for i := 0; i < followers; i++ {
			if r := <-done; !r.Deduplicated || r.Attempts != 0 {
				t.Fatalf("S7 follower executed independently: %+v", r)
			}
		}
		if executed.Load() != 0 {
			t.Fatalf("S7: %d followers executed", executed.Load())
		}
		waitFor(t, func() bool { return c.ActiveSlots() == 0 && c.FollowerSlots() == 0 })
	})

	// S8: deduplication of a completed (terminal success) request.
	t.Run("S8_dedup_completed_success", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		if _, err := c.Execute(context.Background(), "s8", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{OutputCommitted: true}, nil
		}); err != nil {
			t.Fatal(err)
		}
		var reran atomic.Bool
		res, err := c.Execute(context.Background(), "s8", func(context.Context, int) (AttemptResult, error) {
			reran.Store(true)
			return AttemptResult{}, nil
		})
		if err != nil || !res.Deduplicated || res.Attempts != 0 || reran.Load() {
			t.Fatalf("S8: re-executed completed request: %+v reran=%v", res, reran.Load())
		}
	})

	// S9: deterministic non-retryable terminal failure is retained + replayed.
	t.Run("S9_dedup_retains_nonretryable_terminal_failure", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		if _, err := c.Execute(context.Background(), "s9", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{}, terminalAuthError()
		}); !IsErrorClass(err, ErrorAuthentication) {
			t.Fatalf("S9 leader: %v", err)
		}
		var reran atomic.Bool
		res, err := c.Execute(context.Background(), "s9", func(context.Context, int) (AttemptResult, error) {
			reran.Store(true)
			return AttemptResult{}, nil
		})
		if !IsErrorClass(err, ErrorAuthentication) || !res.Deduplicated || reran.Load() {
			t.Fatalf("S9: terminal failure not replayed: %+v err=%v reran=%v", res, err, reran.Load())
		}
	})

	// S10: prompt cancellation releases the slot and does not poison the id.
	t.Run("S10_cancellation_releases_slot_no_poison", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, err := c.Execute(ctx, "s10", func(runCtx context.Context, _ int) (AttemptResult, error) {
				close(started)
				<-runCtx.Done()
				return AttemptResult{}, runCtx.Err()
			})
			done <- err
		}()
		<-started
		waitFor(t, func() bool { return c.ActiveSlots() == 1 })
		cancel()
		if err := <-done; !IsErrorClass(err, ErrorCancelled) {
			t.Fatalf("S10: expected cancelled, got %v", err)
		}
		waitFor(t, func() bool { return c.ActiveSlots() == 0 })
		// Not poisoned: a fresh resubmission runs.
		res, err := c.Execute(context.Background(), "s10", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{OutputCommitted: true}, nil
		})
		if err != nil || res.Deduplicated || res.Attempts != 1 {
			t.Fatalf("S10: pre-output cancel poisoned id: %+v err=%v", res, err)
		}
	})

	// S11: deadline vs cancellation stay distinct, incl. a callback-returned
	// DeadlineExceeded while the context is still active.
	t.Run("S11_deadline_vs_cancellation_distinct", func(t *testing.T) {
		clock := time.Unix(1_700_000_000, 0)
		c := newTask86Coordinator(t, &clock)
		_, tErr := c.Execute(context.Background(), "s11-timeout", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{}, context.DeadlineExceeded
		})
		if !IsErrorClass(tErr, ErrorTimeout) {
			t.Fatalf("S11: callback DeadlineExceeded collapsed: %v", tErr)
		}
		_, cErr := c.Execute(context.Background(), "s11-cancel", func(context.Context, int) (AttemptResult, error) {
			return AttemptResult{}, context.Canceled
		})
		if !IsErrorClass(cErr, ErrorCancelled) {
			t.Fatalf("S11: callback Canceled misclassified: %v", cErr)
		}
	})
}

// Bounded-state acceptance (supports S8/S9 dedup): terminal state stays bounded
// by MaxTracked and expires to zero via the injected clock, and slots drain
// under concurrency — proving dedup memory is bounded.
func TestTask86CoordinatorDedupStateIsBounded(t *testing.T) {
	clock := time.Unix(1_700_000_000, 0)
	c, err := NewCoordinatorFromConfig(CoordinatorConfig{
		Policy:     RetryPolicy{MaxAttempts: 1, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true},
		DedupTTL:   time.Minute,
		MaxTracked: 4,
		now:        func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	success := func(context.Context, int) (AttemptResult, error) { return AttemptResult{OutputCommitted: true}, nil }
	for i := 0; i < 20; i++ {
		if _, err := c.Execute(context.Background(), "b-"+strconv.Itoa(i), success); err != nil {
			t.Fatalf("execute %d: %v", i, err)
		}
	}
	if got := c.TerminalCount(); got > 4 {
		t.Fatalf("dedup state unbounded: %d > MaxTracked 4", got)
	}
	clock = clock.Add(2 * time.Minute) // advance past DedupTTL
	if got := c.TerminalCount(); got != 0 {
		t.Fatalf("dedup state did not expire to zero: %d", got)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) { defer wg.Done(); _, _ = c.Execute(context.Background(), "c-"+strconv.Itoa(n), success) }(i)
	}
	wg.Wait()
	if c.ActiveSlots() != 0 || c.FollowerSlots() != 0 {
		t.Fatalf("slots leaked: active=%d followers=%d", c.ActiveSlots(), c.FollowerSlots())
	}
}
