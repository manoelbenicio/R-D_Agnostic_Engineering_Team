package rotation

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

// RetryDecision classifies whether a failed attempt should be retried against
// the same account/item, or trigger an immediate failover to the next item in
// the fallback chain.
//
// Semantics (design.md §3):
//   - RETRY:        transient failure — retry the same item with backoff+jitter
//     (rate-limit 429, timeout, transient 5xx like 503).
//   - FAILOVER_NOW: non-retryable failure — skip to the next item immediately
//     (invalid auth 401, invalid request 400).
type RetryDecision int

const (
	// RETRY indicates the attempt should be retried with backoff.
	RETRY RetryDecision = iota
	// FAILOVER_NOW indicates an immediate failover to the next chain item.
	FAILOVER_NOW
)

func (d RetryDecision) String() string {
	switch d {
	case RETRY:
		return "RETRY"
	case FAILOVER_NOW:
		return "FAILOVER_NOW"
	default:
		return "UNKNOWN"
	}
}

const (
	// baseBackoff is the first retry delay (attempt 0).
	baseBackoff = 500 * time.Millisecond
	// maxBackoff caps the exponential backoff (design §3: cap 4s).
	maxBackoff = 4 * time.Second
	// jitterFraction is the +/- jitter applied to a backoff duration (±10%).
	jitterFraction = 0.10
	// maxRetriesAllowed is the upper bound for configurable retries (design §3: 0–10).
	maxRetriesAllowed = 10
)

// NextBackoff returns the exponential backoff delay for the given zero-based
// attempt number: 500ms → 1s → 2s → 4s, capped at 4s (design.md §3).
//
// Negative attempts are treated as attempt 0. Large attempts saturate at the
// cap without overflowing.
func NextBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return baseBackoff
	}
	// Saturate early: 500ms << 3 == 4s already hits the cap, so any attempt
	// beyond 3 is capped. This also avoids shift overflow for large attempts.
	if attempt >= 3 {
		return maxBackoff
	}
	d := baseBackoff << uint(attempt)
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// JitterBounds returns the inclusive [min, max] range that Jitter(d) can
// produce for a given duration, applying ±jitterFraction. Exposed for
// deterministic testing (design.md §3: "deterministic-testable").
func JitterBounds(d time.Duration) (min, max time.Duration) {
	delta := time.Duration(float64(d) * jitterFraction)
	return d - delta, d + delta
}

// jitterRand is the package-level randomness source used by Jitter. It is
// guarded by jitterMu and may be replaced in tests for full determinism.
var (
	jitterMu   sync.Mutex
	jitterRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// SetJitterSource replaces the internal random source used by Jitter. It is
// intended for deterministic testing. Passing a nil source is a no-op.
func SetJitterSource(src rand.Source) {
	if src == nil {
		return
	}
	jitterMu.Lock()
	jitterRand = rand.New(src)
	jitterMu.Unlock()
}

// Jitter applies ±10% uniform jitter to a backoff duration to avoid the
// thundering-herd effect (design.md §3). The result is always within the
// bounds reported by JitterBounds(d).
func Jitter(d time.Duration) time.Duration {
	min, max := JitterBounds(d)
	span := int64(max - min)
	if span <= 0 {
		return d
	}
	jitterMu.Lock()
	offset := jitterRand.Int63n(span + 1)
	jitterMu.Unlock()
	return min + time.Duration(offset)
}

// ClassifyError decides whether a failed attempt should be retried or trigger
// an immediate failover, based on the returned error and HTTP status
// (design.md §3).
//
//	RETRY:        timeout errors, 429 (rate-limit), transient 5xx (>=500, e.g. 503)
//	FAILOVER_NOW: auth/invalid-request errors — 401, 403, 400 (and other 4xx)
//
// A nil error with a status code is classified purely by the status, which is
// the common path when the transport succeeded but the API returned an error.
func ClassifyError(err error, httpStatus int) RetryDecision {
	// Timeouts and cancelled deadlines are transient → retry.
	if isTimeout(err) {
		return RETRY
	}

	switch {
	case httpStatus == 429:
		return RETRY
	case httpStatus >= 500:
		// Transient server-side errors (503, etc.).
		return RETRY
	case httpStatus == 401 || httpStatus == 403 || httpStatus == 400:
		// Auth failures / invalid requests are not retryable.
		return FAILOVER_NOW
	case httpStatus >= 400:
		// Any other 4xx is a client-side, non-retryable error.
		return FAILOVER_NOW
	}

	// No status signal: an unclassified transport error fails over rather than
	// spinning on the same account.
	if err != nil {
		return FAILOVER_NOW
	}
	// No error and no status → nothing to retry.
	return FAILOVER_NOW
}

// isTimeout reports whether err represents a timeout or deadline-exceeded
// condition, which design §3 treats as retryable.
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

// RetryPlan encapsulates the retry budget for a single fallback chain item
// (design.md §3: retries 0–10, default 1).
type RetryPlan struct {
	// MaxRetries is the number of retries permitted before failing over.
	MaxRetries int
}

// NewRetryPlan builds a RetryPlan clamped to the valid range [0, 10].
func NewRetryPlan(maxRetries int) RetryPlan {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries > maxRetriesAllowed {
		maxRetries = maxRetriesAllowed
	}
	return RetryPlan{MaxRetries: maxRetries}
}

// ShouldRetry reports whether another retry is allowed for the given zero-based
// attempt number. It returns true while attempt < MaxRetries.
func (rp RetryPlan) ShouldRetry(attempt int) bool {
	return attempt < rp.MaxRetries
}
