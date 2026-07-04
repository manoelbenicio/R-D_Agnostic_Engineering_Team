package rotation

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestFallbackNextBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 4 * time.Second}, // cap
		{10, 4 * time.Second}, // cap holds for large attempts
		{-1, 500 * time.Millisecond}, // negative clamps to base
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("attempt=%d", tc.attempt), func(t *testing.T) {
			if got := NextBackoff(tc.attempt); got != tc.want {
				t.Fatalf("NextBackoff(%d) = %v, want %v", tc.attempt, got, tc.want)
			}
		})
	}
}

func TestFallbackJitterWithinBounds(t *testing.T) {
	// Deterministic source so the test is reproducible.
	SetJitterSource(rand.NewSource(1))
	t.Cleanup(func() { SetJitterSource(rand.NewSource(time.Now().UnixNano())) })

	const d = time.Second
	lo, hi := 900*time.Millisecond, 1100*time.Millisecond

	if min, max := JitterBounds(d); min != lo || max != hi {
		t.Fatalf("JitterBounds(%v) = [%v,%v], want [%v,%v]", d, min, max, lo, hi)
	}

	for i := 0; i < 1000; i++ {
		got := Jitter(d)
		if got < lo || got > hi {
			t.Fatalf("Jitter(%v) = %v, out of bounds [%v,%v]", d, got, lo, hi)
		}
	}
}

type fallbackTimeoutErr struct{}

func (fallbackTimeoutErr) Error() string { return "i/o timeout" }
func (fallbackTimeoutErr) Timeout() bool  { return true }
func (fallbackTimeoutErr) Temporary() bool { return true }

func TestFallbackClassifyError(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		want   RetryDecision
	}{
		{"429_rate_limit", nil, 429, RETRY},
		{"401_auth", nil, 401, FAILOVER_NOW},
		{"503_transient", nil, 503, RETRY},
		{"400_bad_request", nil, 400, FAILOVER_NOW},
		{"403_forbidden", nil, 403, FAILOVER_NOW},
		{"500_server_error", nil, 500, RETRY},
		{"net_timeout", fallbackTimeoutErr{}, 0, RETRY},
		{"deadline_exceeded", context.DeadlineExceeded, 0, RETRY},
		{"unknown_error_no_status", errors.New("boom"), 0, FAILOVER_NOW},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyError(tc.err, tc.status); got != tc.want {
				t.Fatalf("ClassifyError(%v, %d) = %v, want %v", tc.err, tc.status, got, tc.want)
			}
		})
	}
}

func TestFallbackRetryPlanShouldRetry(t *testing.T) {
	cases := []struct {
		maxRetries int
		attempt    int
		want       bool
	}{
		{2, 0, true},
		{2, 1, true},
		{2, 2, false},
		{0, 0, false},
		{1, 0, true},
		{1, 1, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("max=%d/attempt=%d", tc.maxRetries, tc.attempt), func(t *testing.T) {
			rp := RetryPlan{MaxRetries: tc.maxRetries}
			if got := rp.ShouldRetry(tc.attempt); got != tc.want {
				t.Fatalf("RetryPlan{%d}.ShouldRetry(%d) = %v, want %v",
					tc.maxRetries, tc.attempt, got, tc.want)
			}
		})
	}
}

func TestFallbackNewRetryPlanClamp(t *testing.T) {
	if rp := NewRetryPlan(-5); rp.MaxRetries != 0 {
		t.Fatalf("NewRetryPlan(-5).MaxRetries = %d, want 0", rp.MaxRetries)
	}
	if rp := NewRetryPlan(99); rp.MaxRetries != 10 {
		t.Fatalf("NewRetryPlan(99).MaxRetries = %d, want 10", rp.MaxRetries)
	}
	if rp := NewRetryPlan(3); rp.MaxRetries != 3 {
		t.Fatalf("NewRetryPlan(3).MaxRetries = %d, want 3", rp.MaxRetries)
	}
}

func TestRetryDecisionString(t *testing.T) {
	if RETRY.String() != "RETRY" {
		t.Fatalf("RETRY.String() = %q", RETRY.String())
	}
	if FAILOVER_NOW.String() != "FAILOVER_NOW" {
		t.Fatalf("FAILOVER_NOW.String() = %q", FAILOVER_NOW.String())
	}
}
