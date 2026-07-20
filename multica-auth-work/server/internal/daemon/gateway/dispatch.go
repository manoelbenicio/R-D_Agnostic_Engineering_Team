package gateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultDedupTTL   = 5 * time.Minute
	maxDedupTTL       = time.Hour
	defaultMaxTracked = 4096
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
	result    ExecutionResult
	err       error
	expiresAt time.Time
}

// CoordinatorConfig configures a Coordinator. Policy is required; DedupTTL and
// MaxTracked default when zero.
type CoordinatorConfig struct {
	Policy     RetryPolicy
	DedupTTL   time.Duration
	MaxTracked int

	now   func() time.Time                                 // test clock injection
	sleep func(ctx context.Context, d time.Duration) error // test sleep injection
}

// Coordinator bounds and deduplicates upstream execution for a route. It:
//   - admits a request atomically — the in-flight slot and dedup registration
//     are committed under one lock before any attempt runs, and released
//     exactly once on every exit;
//   - retries only before the first user-visible output or tool action, and
//     only for retryable classified errors, bounded by attempts and the
//     end-to-end deadline;
//   - never replays a request after partial output or a committed tool action;
//   - deduplicates concurrent and repeated requests that share a request id,
//     retaining deterministic terminal outcomes (success and non-retryable
//     terminal failures) so a completed request is not re-executed, while a
//     pre-output cancellation or a transient/deadline outcome does not poison
//     the id for a legitimate retry;
//   - bounds dedup state: terminal records expire after DedupTTL, can be
//     dropped explicitly via Forget, and are capped at MaxTracked with
//     oldest-first eviction that never touches an in-flight entry.
type Coordinator struct {
	policy     RetryPolicy
	dedupTTL   time.Duration
	maxTracked int
	now        func() time.Time
	sleep      func(ctx context.Context, d time.Duration) error

	mu       sync.Mutex
	terminal map[string]terminalRecord
	inFlight map[string]*coordinatorFlight

	active    atomic.Int64
	followers atomic.Int64
}

// NewCoordinator builds a Coordinator from a validated RetryPolicy with default
// dedup bounds.
func NewCoordinator(policy RetryPolicy) (*Coordinator, error) {
	return NewCoordinatorFromConfig(CoordinatorConfig{Policy: policy})
}

// NewCoordinatorFromConfig builds a Coordinator from an explicit configuration.
func NewCoordinatorFromConfig(cfg CoordinatorConfig) (*Coordinator, error) {
	policy := cfg.Policy
	if policy.MaxAttempts < 1 || policy.MaxAttempts > 10 ||
		policy.EndToEndDeadline <= 0 || policy.EndToEndDeadline > 10*time.Minute ||
		!policy.PreCommitOnly ||
		policy.MinimumBackoff < 0 || policy.MaximumBackoff < policy.MinimumBackoff || policy.MaximumBackoff > time.Minute {
		return nil, &GatewayError{Operation: "coordinator", Class: ErrorInvalidConfiguration}
	}
	ttl := cfg.DedupTTL
	if ttl == 0 {
		ttl = defaultDedupTTL
	}
	if ttl < 0 || ttl > maxDedupTTL {
		return nil, &GatewayError{Operation: "coordinator", Class: ErrorInvalidConfiguration}
	}
	maxTracked := cfg.MaxTracked
	if maxTracked == 0 {
		maxTracked = defaultMaxTracked
	}
	if maxTracked < 1 {
		return nil, &GatewayError{Operation: "coordinator", Class: ErrorInvalidConfiguration}
	}
	now := cfg.now
	if now == nil {
		now = time.Now
	}
	sleep := cfg.sleep
	if sleep == nil {
		sleep = sleepContext
	}
	return &Coordinator{
		policy:     policy,
		dedupTTL:   ttl,
		maxTracked: maxTracked,
		now:        now,
		sleep:      sleep,
		terminal:   make(map[string]terminalRecord),
		inFlight:   make(map[string]*coordinatorFlight),
	}, nil
}

// ActiveSlots is the number of leader executions currently holding capacity.
func (c *Coordinator) ActiveSlots() int64 { return c.active.Load() }

// FollowerSlots is the number of callers currently waiting on a deduplicated
// in-flight request.
func (c *Coordinator) FollowerSlots() int64 { return c.followers.Load() }

// InFlightCount is the number of distinct leader requests currently tracked.
func (c *Coordinator) InFlightCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.inFlight)
}

// TerminalCount is the number of retained terminal records after reclaiming any
// that have expired.
func (c *Coordinator) TerminalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweepExpiredTerminalLocked(c.now())
	return len(c.terminal)
}

// Forget drops any retained terminal record for a request id, e.g. when the
// owning session completes or is cancelled, returning dedup state toward zero.
func (c *Coordinator) Forget(requestID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.terminal, requestID)
}

// Execute runs run under the coordinator's retry, replay, dedup, cancellation
// and bounded-state contract, keyed by requestID.
func (c *Coordinator) Execute(ctx context.Context, requestID string, run AttemptFunc) (ExecutionResult, error) {
	if strings.TrimSpace(requestID) == "" || run == nil {
		return ExecutionResult{}, &GatewayError{Operation: "coordinator.execute", Class: ErrorInvalidRequest}
	}

	c.mu.Lock()
	c.sweepExpiredTerminalLocked(c.now())
	if record, terminal := c.terminal[requestID]; terminal {
		c.mu.Unlock()
		return ExecutionResult{Deduplicated: true}, record.err
	}
	if flight, exists := c.inFlight[requestID]; exists {
		c.mu.Unlock()
		return c.follow(ctx, flight)
	}
	// Atomic admission: commit the dedup registration and acquire the in-flight
	// slot under a single lock, before any attempt can run.
	flight := &coordinatorFlight{done: make(chan struct{})}
	c.inFlight[requestID] = flight
	c.active.Add(1)
	c.mu.Unlock()

	return c.lead(ctx, requestID, flight, run)
}

// follow waits on an in-flight leader and shares its terminal outcome. A
// follower's own cancellation/deadline is reported distinctly and never poisons
// the leader or the request id.
func (c *Coordinator) follow(ctx context.Context, flight *coordinatorFlight) (ExecutionResult, error) {
	c.followers.Add(1)
	defer c.followers.Add(-1)
	select {
	case <-ctx.Done():
		return ExecutionResult{Deduplicated: true}, contextError("coordinator.execute", ctx, ctx.Err())
	case <-flight.done:
		// A follower ran no attempts of its own; it only inherits the leader's
		// terminal error under the deduplication contract.
		return ExecutionResult{Deduplicated: true}, flight.err
	}
}

func (c *Coordinator) lead(ctx context.Context, requestID string, flight *coordinatorFlight, run AttemptFunc) (result ExecutionResult, resultErr error) {
	execCtx, cancel := context.WithTimeout(ctx, c.policy.EndToEndDeadline)
	defer cancel()

	committed := false
	defer func() {
		c.mu.Lock()
		// Retain deterministic terminal outcomes so duplicates are deduplicated:
		// a success, a committed request, or a non-retryable classified failure.
		// A pre-output cancellation or a transient/deadline outcome is NOT
		// terminal, so a legitimate resubmission may still run.
		if resultErr == nil || committed || isTerminalFailure(resultErr) {
			c.recordTerminalLocked(requestID, result, resultErr, c.now())
		}
		flight.result = result
		flight.err = resultErr
		delete(c.inFlight, requestID)
		close(flight.done)
		// Release the in-flight slot exactly once, paired with the acquisition
		// in Execute.
		c.active.Add(-1)
		c.mu.Unlock()
	}()

	var lastErr error
	for attempt := 1; attempt <= c.policy.MaxAttempts; attempt++ {
		if execCtx.Err() != nil {
			return result, contextError("coordinator.execute", execCtx, execCtx.Err())
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
		// A caller cancellation or end-to-end deadline is terminal for this call
		// but not for the id; surface cancelled vs timeout distinctly (honoring a
		// deadline the callback returned even while execCtx is still active) and
		// release the slot.
		if isContextError(execCtx, err) {
			return result, contextError("coordinator.execute", execCtx, err)
		}
		// Only retryable classified errors may be replayed before first output.
		if !isRetryable(err) {
			return result, err
		}
		if attempt == c.policy.MaxAttempts {
			break
		}
		if sleepErr := c.sleep(execCtx, c.backoff(attempt, err)); sleepErr != nil {
			return result, contextError("coordinator.execute", execCtx, sleepErr)
		}
	}
	return result, lastErr
}

// recordTerminalLocked stores a terminal record under the capacity bound.
// Callers must hold c.mu. Eviction removes the oldest completed record; it can
// never remove an in-flight entry because those live in a separate map.
func (c *Coordinator) recordTerminalLocked(requestID string, result ExecutionResult, err error, now time.Time) {
	if _, exists := c.terminal[requestID]; !exists && len(c.terminal) >= c.maxTracked {
		c.evictOldestTerminalLocked()
	}
	c.terminal[requestID] = terminalRecord{result: result, err: err, expiresAt: now.Add(c.dedupTTL)}
}

func (c *Coordinator) evictOldestTerminalLocked() {
	var oldestID string
	var oldest time.Time
	first := true
	for id, record := range c.terminal {
		if first || record.expiresAt.Before(oldest) {
			oldestID, oldest, first = id, record.expiresAt, false
		}
	}
	if !first {
		delete(c.terminal, oldestID)
	}
}

func (c *Coordinator) sweepExpiredTerminalLocked(now time.Time) {
	for id, record := range c.terminal {
		if !record.expiresAt.IsZero() && !now.Before(record.expiresAt) {
			delete(c.terminal, id)
		}
	}
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

// isTerminalFailure reports whether a non-nil error is a deterministic terminal
// failure that should be deduplicated. Retryable errors are transient;
// cancellation is never terminal (a pre-output cancel must not poison the id);
// an end-to-end deadline surfaces as a retryable timeout and is likewise not
// terminal.
func isTerminalFailure(err error) bool {
	var gatewayErr *GatewayError
	if !errors.As(err, &gatewayErr) {
		return false
	}
	if gatewayErr.Retryable || gatewayErr.Class == ErrorCancelled {
		return false
	}
	return true
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

// contextError preserves the distinction between an end-to-end deadline
// (timeout, retryable) and an explicit cancellation (cancelled), considering
// both the context state and any context error the callback returned. A
// callback that returns context.DeadlineExceeded while execCtx is still active
// is still a timeout; it is never collapsed into a cancellation.
func contextError(operation string, ctx context.Context, err error) error {
	// An explicit cancellation of the context is authoritative.
	if errors.Is(ctx.Err(), context.Canceled) {
		return &GatewayError{Operation: operation, Class: ErrorCancelled}
	}
	// A deadline from the context or surfaced by the callback is a timeout.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return &GatewayError{Operation: operation, Class: ErrorTimeout, Retryable: true}
	}
	// Otherwise honor a cancellation the callback reported directly.
	return &GatewayError{Operation: operation, Class: ErrorCancelled}
}
