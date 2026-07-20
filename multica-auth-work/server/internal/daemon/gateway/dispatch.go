package gateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// AttemptResult reports what an individual upstream attempt committed. Once a
// request has delivered user-visible output or performed a potentially
// non-idempotent tool action it can never be safely replayed.
type AttemptResult struct {
	OutputCommitted     bool
	ToolActionCommitted bool
}

// AttemptFunc executes a single upstream attempt. The attempt index is 1-based.
// It returns what the attempt committed and a classified error (nil on
// success). Implementations must honor ctx cancellation.
type AttemptFunc func(ctx context.Context, attempt int) (AttemptResult, error)

// ExecutionResult summarizes a coordinated execution. Attempts counts the
// upstream attempts actually run by this call; Deduplicated is true when the
// call joined or short-circuited an equivalent request instead of executing.
type ExecutionResult struct {
	Attempts     int
	Deduplicated bool
}

type coordinatorFlight struct {
	done   chan struct{}
	result ExecutionResult
	err    error
}

type terminalRecord struct {
	result ExecutionResult
	err    error
}

// Coordinator bounds and deduplicates upstream execution for a route. It:
//   - retries only before the first user-visible output or tool action, and
//     only for retryable classified errors, bounded by attempts and the
//     end-to-end deadline;
//   - never replays a request after partial output or a committed tool action;
//   - deduplicates concurrent and repeated requests that share a request id so
//     a completed request is not re-executed; and
//   - releases its in-flight capacity slot deterministically on completion or
//     cancellation, and a pre-output cancellation does not poison the id for a
//     later legitimate retry.
type Coordinator struct {
	policy RetryPolicy
	now    func() time.Time
	sleep  func(ctx context.Context, d time.Duration) error

	mu       sync.Mutex
	terminal map[string]terminalRecord
	inFlight map[string]*coordinatorFlight

	active    atomic.Int64
	followers atomic.Int64
}

// NewCoordinator builds a Coordinator from a validated RetryPolicy. The policy
// must be pre-commit-only, matching the routing contract.
func NewCoordinator(policy RetryPolicy) (*Coordinator, error) {
	if policy.MaxAttempts < 1 || policy.MaxAttempts > 10 ||
		policy.EndToEndDeadline <= 0 || policy.EndToEndDeadline > 10*time.Minute ||
		!policy.PreCommitOnly ||
		policy.MinimumBackoff < 0 || policy.MaximumBackoff < policy.MinimumBackoff || policy.MaximumBackoff > time.Minute {
		return nil, &GatewayError{Operation: "coordinator", Class: ErrorInvalidConfiguration}
	}
	return &Coordinator{
		policy:   policy,
		now:      time.Now,
		sleep:    sleepContext,
		terminal: make(map[string]terminalRecord),
		inFlight: make(map[string]*coordinatorFlight),
	}, nil
}

// ActiveSlots is the number of leader executions currently holding capacity.
func (c *Coordinator) ActiveSlots() int64 { return c.active.Load() }

// FollowerSlots is the number of callers currently waiting on a deduplicated
// in-flight request.
func (c *Coordinator) FollowerSlots() int64 { return c.followers.Load() }

// Execute runs run under the coordinator's retry, replay, dedup and
// cancellation contract, keyed by requestID.
func (c *Coordinator) Execute(ctx context.Context, requestID string, run AttemptFunc) (ExecutionResult, error) {
	if strings.TrimSpace(requestID) == "" || run == nil {
		return ExecutionResult{}, &GatewayError{Operation: "coordinator.execute", Class: ErrorInvalidRequest}
	}

	c.mu.Lock()
	if record, terminal := c.terminal[requestID]; terminal {
		c.mu.Unlock()
		return ExecutionResult{Deduplicated: true}, record.err
	}
	if flight, exists := c.inFlight[requestID]; exists {
		c.mu.Unlock()
		return c.follow(ctx, flight)
	}
	flight := &coordinatorFlight{done: make(chan struct{})}
	c.inFlight[requestID] = flight
	c.mu.Unlock()

	return c.lead(ctx, requestID, flight, run)
}

// follow waits on an in-flight leader and shares its terminal outcome. A
// follower's own cancellation is reported as cancelled and never poisons the
// leader or the request id.
func (c *Coordinator) follow(ctx context.Context, flight *coordinatorFlight) (ExecutionResult, error) {
	c.followers.Add(1)
	defer c.followers.Add(-1)
	select {
	case <-ctx.Done():
		return ExecutionResult{Deduplicated: true}, contextError("coordinator.execute", ctx)
	case <-flight.done:
		// A follower ran no attempts of its own; it only inherits the leader's
		// terminal error under the deduplication contract.
		return ExecutionResult{Deduplicated: true}, flight.err
	}
}

func (c *Coordinator) lead(ctx context.Context, requestID string, flight *coordinatorFlight, run AttemptFunc) (result ExecutionResult, resultErr error) {
	c.active.Add(1)

	execCtx, cancel := context.WithTimeout(ctx, c.policy.EndToEndDeadline)
	defer cancel()

	committed := false
	defer func() {
		c.active.Add(-1)
		c.mu.Lock()
		// A completed (successful) or committed request is terminal and must
		// not be replayed by a later duplicate. A pre-output cancellation is
		// NOT terminal so a legitimate resubmission may still run.
		if resultErr == nil || committed {
			c.terminal[requestID] = terminalRecord{result: result, err: resultErr}
		}
		flight.result = result
		flight.err = resultErr
		delete(c.inFlight, requestID)
		close(flight.done)
		c.mu.Unlock()
	}()

	var lastErr error
	for attempt := 1; attempt <= c.policy.MaxAttempts; attempt++ {
		if err := execCtx.Err(); err != nil {
			return result, contextError("coordinator.execute", execCtx)
		}

		result.Attempts++
		attemptResult, err := run(execCtx, attempt)
		if attemptResult.OutputCommitted || attemptResult.ToolActionCommitted {
			committed = true
		}

		if err == nil {
			return result, nil
		}
		lastErr = err

		// Never replay after partial output or a committed tool action.
		if committed {
			return result, err
		}
		// A caller/deadline cancellation is terminal-for-this-call but not for
		// the id; surface it as cancelled/timeout and release the slot.
		if isContextError(execCtx, err) {
			return result, contextError("coordinator.execute", execCtx)
		}
		// Only retryable classified errors may be replayed before first output.
		if !isRetryable(err) {
			return result, err
		}
		if attempt == c.policy.MaxAttempts {
			break
		}
		if err := c.sleep(execCtx, c.backoff(attempt, err)); err != nil {
			return result, contextError("coordinator.execute", execCtx)
		}
	}
	return result, lastErr
}

// backoff computes the pre-commit retry delay, honoring an upstream Retry-After
// hint when present and otherwise applying bounded exponential backoff.
func (c *Coordinator) backoff(attempt int, err error) time.Duration {
	if hint := retryAfter(err); hint > 0 {
		if hint > c.policy.MaximumBackoff {
			return c.policy.MaximumBackoff
		}
		return hint
	}
	delay := c.policy.MinimumBackoff
	for i := 1; i < attempt && delay < c.policy.MaximumBackoff; i++ {
		delay *= 2
	}
	if delay > c.policy.MaximumBackoff {
		delay = c.policy.MaximumBackoff
	}
	return delay
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryable(err error) bool {
	var gatewayErr *GatewayError
	return errors.As(err, &gatewayErr) && gatewayErr.Retryable
}

func retryAfter(err error) time.Duration {
	var gatewayErr *GatewayError
	if errors.As(err, &gatewayErr) {
		return gatewayErr.RetryAfter
	}
	return 0
}

func isContextError(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func contextError(operation string, ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &GatewayError{Operation: operation, Class: ErrorTimeout, Retryable: true}
	}
	return &GatewayError{Operation: operation, Class: ErrorCancelled}
}
