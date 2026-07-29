package gateway

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func newTestExecutor(t *testing.T, affinity AffinityMode, accounts ...string) (*Executor, *Selector) {
	t.Helper()
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation: RotationStrictIndependentRequest,
		Affinity: affinity,
		Accounts: accounts,
	})
	if err != nil {
		t.Fatalf("NewSelectorFromConfig: %v", err)
	}
	coordinator, err := NewCoordinator(RetryPolicy{
		MaxAttempts: 3, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true,
	})
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}
	executor, err := NewExecutor(selector, coordinator)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	return executor, selector
}

func successOutcome(handle ContinuationRefs) ProviderCall {
	return func(context.Context, string, int) ProviderOutcome {
		return ProviderOutcome{OutputCommitted: true, ProducedHandle: handle}
	}
}

func TestExecutorBindsOwnerOnlyAfterSuccessAndPinsContinuation(t *testing.T) {
	executor, selector := newTestExecutor(t, AffinityOriginAccount, "acct-a", "acct-b", "acct-c")

	// First turn is independent and, on success, binds the produced handle to
	// the actual selected account.
	first, err := executor.Execute(context.Background(), "req-1", ContinuationRefs{}, successOutcome(ContinuationRefs{PreviousResponseID: "resp-1"}))
	if err != nil || first.BindError != nil {
		t.Fatalf("first request failed: err=%v bindErr=%v", err, first.BindError)
	}
	if first.Reason != SelectionIndependentRotation {
		t.Fatalf("first request reason=%s want independent", first.Reason)
	}
	if selector.BindingCount() != 1 {
		t.Fatalf("expected exactly one binding after success, got %d", selector.BindingCount())
	}

	// A continuation referencing the produced handle pins to the same account.
	cont, err := executor.Execute(context.Background(), "req-2", ContinuationRefs{PreviousResponseID: "resp-1"}, successOutcome(ContinuationRefs{}))
	if err != nil {
		t.Fatalf("continuation failed: %v", err)
	}
	if cont.Account != first.Account {
		t.Fatalf("continuation not pinned: got %s want %s", cont.Account, first.Account)
	}
	if cont.Reason != SelectionContinuation {
		t.Fatalf("continuation reason=%s want continuation-affinity", cont.Reason)
	}

	// An unrelated independent request keeps rotating.
	indep, err := executor.Execute(context.Background(), "req-3", ContinuationRefs{}, successOutcome(ContinuationRefs{}))
	if err != nil {
		t.Fatalf("independent request failed: %v", err)
	}
	if indep.Account == first.Account {
		t.Fatalf("independent request reused affinity owner %s", first.Account)
	}
}

func TestExecutorFailsClosedWithoutProviderTrafficWhenOwnerUnknown(t *testing.T) {
	executor, _ := newTestExecutor(t, AffinityOriginAccount, "acct-a", "acct-b")
	var providerCalled atomic.Bool
	_, err := executor.Execute(context.Background(), "req-unknown", ContinuationRefs{PreviousResponseID: "never-bound"}, func(context.Context, string, int) ProviderOutcome {
		providerCalled.Store(true)
		return ProviderOutcome{OutputCommitted: true}
	})
	if !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("expected fail-closed continuation-unavailable, got %v", err)
	}
	if providerCalled.Load() {
		t.Fatal("provider was called despite fail-closed selection")
	}
}

func TestExecutorDoesNotBindOnFailedResponse(t *testing.T) {
	executor, selector := newTestExecutor(t, AffinityOriginAccount, "acct-a", "acct-b", "acct-c")
	// A failed (non-retryable) response that nonetheless names a produced handle
	// must NOT create a binding.
	_, err := executor.Execute(context.Background(), "req-fail", ContinuationRefs{}, func(context.Context, string, int) ProviderOutcome {
		return ProviderOutcome{Failed: true, Failure: FailureSignal{StatusCode: 403}, ProducedHandle: ContinuationRefs{PreviousResponseID: "resp-fail"}}
	})
	if !IsErrorClass(err, ErrorAuthorization) {
		t.Fatalf("expected authorization failure, got %v", err)
	}
	if selector.BindingCount() != 0 {
		t.Fatalf("failed response created a binding: count=%d", selector.BindingCount())
	}
	// The unbound continuation therefore fails closed.
	_, contErr := executor.Execute(context.Background(), "req-fail-cont", ContinuationRefs{PreviousResponseID: "resp-fail"}, successOutcome(ContinuationRefs{}))
	if !IsErrorClass(contErr, ErrorContinuationUnavailable) {
		t.Fatalf("expected fail-closed for unbound continuation, got %v", contErr)
	}
}

func TestExecutorClassifiesRetryableFailureAndRetriesBeforeOutput(t *testing.T) {
	executor, _ := newTestExecutor(t, AffinityOriginAccount, "acct-a", "acct-b")
	var attempts atomic.Int64
	out, err := executor.Execute(context.Background(), "req-retry", ContinuationRefs{}, func(context.Context, string, int) ProviderOutcome {
		if attempts.Add(1) == 1 {
			// Retryable upstream 5xx before any output.
			return ProviderOutcome{Failed: true, Failure: FailureSignal{StatusCode: 500}}
		}
		return ProviderOutcome{OutputCommitted: true}
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if out.Execution.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", out.Execution.Attempts)
	}
}

func TestExecutorRejectsNilArguments(t *testing.T) {
	_, selector := newTestExecutor(t, AffinityOriginAccount, "acct-a")
	coordinator, _ := NewCoordinator(RetryPolicy{MaxAttempts: 1, EndToEndDeadline: time.Second, PreCommitOnly: true})
	if _, err := NewExecutor(nil, coordinator); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for nil selector, got %v", err)
	}
	executor, err := NewExecutor(selector, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.Execute(context.Background(), "req", ContinuationRefs{}, nil); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid request for nil call, got %v", err)
	}
}

func TestExecutorSurfacesBindErrorWithoutFailingSuccessfulResponse(t *testing.T) {
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Accounts:    []string{"acct-a", "acct-b"},
		MaxBindings: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewCoordinator(RetryPolicy{MaxAttempts: 1, EndToEndDeadline: time.Second, PreCommitOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(selector, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	// First success fills the single binding slot.
	if _, err := executor.Execute(context.Background(), "req-1", ContinuationRefs{}, successOutcome(ContinuationRefs{PreviousResponseID: "resp-1"})); err != nil {
		t.Fatalf("first execute: %v", err)
	}
	// Second success wants to bind a new handle but the table is full of live
	// entries: the response still succeeds and the bind error is surfaced.
	out, err := executor.Execute(context.Background(), "req-2", ContinuationRefs{}, successOutcome(ContinuationRefs{PreviousResponseID: "resp-2"}))
	if err != nil {
		t.Fatalf("second execute should still succeed: %v", err)
	}
	if !IsErrorClass(out.BindError, ErrorContinuationCapacity) {
		t.Fatalf("expected surfaced capacity bind error, got %v", out.BindError)
	}
}

func TestExecutorEmitsProviderSpanOnTerminalOutcome(t *testing.T) {
	executor, _ := newTestExecutor(t, AffinityOriginAccount, "acct-a")
	sink := e2e.NewMemorySink()
	recorder := e2e.NewRecorder(sink)

	principal := "principal_0123456789abcdef"
	executor.SetSpanRecorder(recorder, brain.ProtocolAnthropicMessages, principal)

	validTelemetry := Telemetry{
		RequestID:              "omni-req-1",
		ActualModel:            "agy/claude-opus-4-6-thinking",
		ActualRoute:            "route-a",
		PseudonymousAccount:    "acct_0123456789abcdef",
		PseudonymousConnection: "conn_fedcba9876543210",
		SelectionReason:        SelectionIndependentRotation,
		Quota:                  QuotaAvailable,
		Circuit:                CircuitClosed,
		Usage:                  Usage{Input: 10, Output: 20, Total: 30},
	}

	outcome, err := executor.Execute(context.Background(), "req-span-1", ContinuationRefs{}, func(ctx context.Context, acct string, attempt int) ProviderOutcome {
		return ProviderOutcome{
			OutputCommitted: true,
			Telemetry:       validTelemetry,
		}
	})

	if err != nil {
		t.Fatalf("expected successful execution, got %v", err)
	}
	if outcome.Account != "acct-a" {
		t.Fatalf("expected account acct-a, got %s", outcome.Account)
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span emitted, got %d", len(spans))
	}
	span := spans[0]
	if span.Hop != e2e.HopRoute {
		t.Fatalf("expected HopRoute, got %s", span.Hop)
	}
	if span.Correlation.RequestID != "req-span-1" || span.Correlation.OmniRequestID != "omni-req-1" {
		t.Fatalf("unexpected correlation: %+v", span.Correlation)
	}
	if span.Labels["principal_pseudonym"] != principal {
		t.Fatalf("expected principal_pseudonym %s, got %s", principal, span.Labels["principal_pseudonym"])
	}
}

func TestExecutorFailsClosedWhenSpanTelemetryMissingOrInvalid(t *testing.T) {
	executor, _ := newTestExecutor(t, AffinityOriginAccount, "acct-a")
	sink := e2e.NewMemorySink()
	recorder := e2e.NewRecorder(sink)

	// Set invalid protocol family to force EmitProviderSpan failure.
	executor.SetSpanRecorder(recorder, brain.ProtocolFamily("invalid-protocol"), "principal_0123456789abcdef")

	_, err := executor.Execute(context.Background(), "req-fail-closed", ContinuationRefs{}, func(ctx context.Context, acct string, attempt int) ProviderOutcome {
		return ProviderOutcome{
			OutputCommitted: true,
		}
	})

	if err == nil {
		t.Fatal("expected Execute to fail closed when EmitProviderSpan fails")
	}
	if !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("expected ErrorProtocol, got %v", err)
	}
	if sink.Len() != 0 {
		t.Fatalf("expected no spans recorded on fail closed, got %d", sink.Len())
	}
}

func TestExecutorSpanPreservesRetryAndNoReplay(t *testing.T) {
	executor, _ := newTestExecutor(t, AffinityOriginAccount, "acct-a")
	sink := e2e.NewMemorySink()
	recorder := e2e.NewRecorder(sink)

	principal := "principal_0123456789abcdef"
	executor.SetSpanRecorder(recorder, brain.ProtocolAnthropicMessages, principal)

	var attempts atomic.Int64
	telemetry := Telemetry{
		RequestID:              "omni-req-retry",
		ActualModel:            "agy/claude-opus-4-6-thinking",
		ActualRoute:            "route-a",
		PseudonymousAccount:    "acct_0123456789abcdef",
		PseudonymousConnection: "conn_fedcba9876543210",
		SelectionReason:        SelectionRetry,
		Quota:                  QuotaAvailable,
		Circuit:                CircuitClosed,
		Usage:                  Usage{Input: 5, Output: 5, Total: 10},
	}

	_, err := executor.Execute(context.Background(), "req-retry-span", ContinuationRefs{}, func(ctx context.Context, acct string, attempt int) ProviderOutcome {
		cur := attempts.Add(1)
		if cur == 1 {
			return ProviderOutcome{
				Failed:    true,
				Failure:   FailureSignal{StatusCode: 503},
				Telemetry: telemetry,
			}
		}
		telemetry.RetryCount = 1
		return ProviderOutcome{
			OutputCommitted: true,
			Telemetry:       telemetry,
		}
	})

	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 terminal route span, got %d", len(spans))
	}
	if spans[0].Counters["retry_count"] != 1 {
		t.Fatalf("expected retry_count=1, got %d", spans[0].Counters["retry_count"])
	}
}
