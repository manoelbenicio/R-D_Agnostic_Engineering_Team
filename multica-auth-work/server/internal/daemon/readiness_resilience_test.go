package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
)

// transientErr is a gateway error that a resilient admitter should retry.
func transientErr(class gateway.ErrorClass) error {
	return &gateway.GatewayError{Operation: "models", Class: class, RetryAfter: time.Millisecond}
}

func admittedDecision() brain.AdmissionDecision {
	return brain.AdmissionDecision{State: brain.AdmissionAdmitted, ReadinessState: brain.GatewayReadinessReady}
}

// A transient fetch failure must not fail closed: the loop retries and returns
// the FRESH admitted result once readiness recovers.
func TestAdmitRetryLoop_TransientThenFreshReady(t *testing.T) {
	calls := 0
	fn := func(context.Context) (brain.AdmissionDecision, error) {
		calls++
		if calls < 3 {
			return brain.AdmissionDecision{}, transientErr(gateway.ErrorRateLimited)
		}
		return admittedDecision(), nil
	}
	decision, err := admitRetryLoop(context.Background(), 5*time.Second, fn)
	if err != nil {
		t.Fatalf("expected fresh ready after transient failures, got err=%v", err)
	}
	if !decision.Admitted() {
		t.Fatalf("expected admitted, got state=%s", decision.State)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts (2 transient + 1 fresh ready), got %d", calls)
	}
}

// A persistent transient failure must exhaust the bounded wait and fail closed
// (never invent readiness, never reuse stale ready).
func TestAdmitRetryLoop_PersistentFailsClosed(t *testing.T) {
	calls := 0
	fn := func(context.Context) (brain.AdmissionDecision, error) {
		calls++
		return brain.AdmissionDecision{}, transientErr(gateway.ErrorTimeout)
	}
	decision, err := admitRetryLoop(context.Background(), 30*time.Millisecond, fn)
	if err == nil {
		t.Fatalf("expected fail-closed error after persistent transient failure")
	}
	if decision.Admitted() {
		t.Fatalf("must not admit on persistent failure")
	}
	if calls < 2 {
		t.Fatalf("expected multiple bounded retries, got %d", calls)
	}
	var ge *gateway.GatewayError
	if !errors.As(err, &ge) || ge.Class != gateway.ErrorTimeout {
		t.Fatalf("expected last transient error preserved, got %v", err)
	}
}

// A deterministic rejection (auth) must fail closed immediately without retry.
func TestAdmitRetryLoop_DeterministicRejectionImmediate(t *testing.T) {
	calls := 0
	fn := func(context.Context) (brain.AdmissionDecision, error) {
		calls++
		return brain.AdmissionDecision{
			State:          brain.AdmissionGatewayAuthFailed,
			ReadinessState: brain.GatewayReadinessAuthentication,
			Retryable:      false,
		}, nil
	}
	decision, _ := admitRetryLoop(context.Background(), 5*time.Second, fn)
	if decision.Admitted() {
		t.Fatalf("auth failure must not admit")
	}
	if calls != 1 {
		t.Fatalf("deterministic rejection must not retry; got %d calls", calls)
	}
}

func TestTransientReadinessRetry_Classification(t *testing.T) {
	transientClasses := []gateway.ErrorClass{
		gateway.ErrorRateLimited, gateway.ErrorTimeout, gateway.ErrorOverloaded,
		gateway.ErrorUpstream, gateway.ErrorTransport,
	}
	for _, c := range transientClasses {
		if ok, _ := transientReadinessRetry(transientErr(c), brain.AdmissionDecision{}); !ok {
			t.Fatalf("class %s should be transient/retryable", c)
		}
	}
	deterministicClasses := []gateway.ErrorClass{
		gateway.ErrorAuthentication, gateway.ErrorAuthorization,
		gateway.ErrorInvalidRequest, gateway.ErrorCapability,
	}
	for _, c := range deterministicClasses {
		if ok, _ := transientReadinessRetry(&gateway.GatewayError{Class: c}, brain.AdmissionDecision{}); ok {
			t.Fatalf("class %s must not be retried (fail closed)", c)
		}
	}
	// Retry-After is surfaced for throttles.
	if _, ra := transientReadinessRetry(transientErr(gateway.ErrorRateLimited), brain.AdmissionDecision{}); ra != time.Millisecond {
		t.Fatalf("expected Retry-After surfaced, got %v", ra)
	}
	// decision-level transient gateway unavailability (nil err) is retryable...
	if ok, _ := transientReadinessRetry(nil, brain.AdmissionDecision{State: brain.AdmissionGatewayUnavailable, ReadinessState: brain.GatewayReadinessUnavailable, Retryable: true}); !ok {
		t.Fatalf("retryable gateway-unavailable decision should retry")
	}
	// ...but a deterministic selected-protocol mismatch must fail closed.
	if ok, _ := transientReadinessRetry(nil, brain.AdmissionDecision{State: brain.AdmissionCapabilityRejected, ReadinessState: brain.GatewayReadinessSelectedProtocol, Retryable: false}); ok {
		t.Fatalf("selected-protocol mismatch must not retry")
	}
}
