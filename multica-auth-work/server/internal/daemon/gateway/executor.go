package gateway

import "context"

// ProviderOutcome is what one upstream provider attempt reports to the Executor.
// On success it may name a continuation handle the response produced, which the
// Executor binds to the selected account. On failure it carries a content-free
// FailureSignal that the Executor classifies. It never carries prompts, tool
// payloads, credentials or response bodies.
type ProviderOutcome struct {
	OutputCommitted     bool
	ToolActionCommitted bool
	// ProducedHandle names continuation state a successful response created
	// (e.g. a new previous_response_id). Zero when the response created none.
	ProducedHandle ContinuationRefs
	// Failure describes a failed attempt; meaningful only when Failed is true.
	Failure FailureSignal
	Failed  bool
}

// ProviderCall performs one upstream attempt against the selected account. It
// must honor ctx cancellation. attempt is 1-based.
type ProviderCall func(ctx context.Context, account string, attempt int) ProviderOutcome

// ExecutionOutcome reports the account chosen for a request, why it was chosen,
// the coordinator's execution summary, and any post-success binding error
// (which never fails an already-successful response).
type ExecutionOutcome struct {
	Account   string
	Reason    SelectionReason
	Sequence  uint64
	Execution ExecutionResult
	BindError error
}

// Executor is the single gateway-side execution entrypoint for OpenSpec
// 8.4–8.6. It composes:
//   - account selection + continuation affinity (Selector, 8.4);
//   - pre-commit retry / no-replay / dedup / cancellation (Coordinator, 8.6);
//   - deterministic failure classification (ClassifyFailure, 8.5); and
//   - post-success continuation binding (Selector.Bind, 8.4 ownership).
//
// A request selects an account once (honoring continuation affinity), then the
// coordinator bounds retries against that account. Only after a successful
// response is any produced continuation handle bound to the actual selected
// account, so a later continuation reference routes back to it. Selection
// failures (no eligible account, or an unavailable continuation owner) fail
// closed before any provider traffic.
//
// # W1 INTEGRATION HANDOFF (the sole required central call site)
//
// This is the only gateway execution entrypoint the central daemon needs to
// call. Today internal/daemon/brain_integration.go wires gateway.Client,
// gateway.NewRegistry and gateway.NewReadinessChecker for readiness/models only
// and performs no routed model dispatch, so Selector/Coordinator/ClassifyFailure
// have no shipping caller. To wire acceptance for 8.4–8.6, W1 (central daemon
// owner) must, at the per-request model-dispatch point in brain_integration.go:
//
//  1. build one *Selector (NewSelectorFromConfig) and one *Coordinator
//     (NewCoordinatorFromConfig) per route from the route's RoutePolicy
//     (gateway.FrozenTier20CanaryPolicy) and the route's account source;
//  2. construct an *Executor via gateway.NewExecutor(selector, coordinator); and
//  3. call Executor.Execute(ctx, requestID, refs, providerCall) where
//     providerCall performs the actual OmniRoute request for the selected
//     account and reports a ProviderOutcome.
//
// No gateway/** change can perform this wiring without editing the central
// daemon, which is outside W2 ownership. Acceptance for 8.4–8.6 remains blocked
// until W1 wires this call site.
type Executor struct {
	selector    *Selector
	coordinator *Coordinator
}

// NewExecutor composes a Selector and Coordinator into the gateway execution
// entrypoint. Both are required.
func NewExecutor(selector *Selector, coordinator *Coordinator) (*Executor, error) {
	if selector == nil || coordinator == nil {
		return nil, &GatewayError{Operation: "executor", Class: ErrorInvalidConfiguration}
	}
	return &Executor{selector: selector, coordinator: coordinator}, nil
}

// Execute selects an account for the request (honoring continuation affinity),
// runs the provider call under the coordinator's retry/dedup/cancellation
// contract, classifies failures deterministically, and — only after a
// successful response — binds any produced continuation handle to the selected
// account. A binding failure never fails an already-successful response; it is
// surfaced via ExecutionOutcome.BindError.
func (e *Executor) Execute(ctx context.Context, requestID string, refs ContinuationRefs, call ProviderCall) (ExecutionOutcome, error) {
	if call == nil {
		return ExecutionOutcome{}, &GatewayError{Operation: "executor.execute", Class: ErrorInvalidRequest}
	}
	// Fail closed on selection before any provider traffic.
	selection, err := e.selector.Select(refs)
	if err != nil {
		return ExecutionOutcome{}, err
	}
	account := selection.Account

	// The closure runs only on the leader path and only sequentially, so the
	// captured bindErr needs no synchronization.
	var bindErr error
	execResult, execErr := e.coordinator.Execute(ctx, requestID, func(runCtx context.Context, attempt int) (AttemptResult, error) {
		outcome := call(runCtx, account, attempt)
		attemptResult := AttemptResult{
			OutputCommitted:     outcome.OutputCommitted,
			ToolActionCommitted: outcome.ToolActionCommitted,
		}
		if !outcome.Failed {
			// Bind only after a successful response, using the actual produced
			// handle and the actual selected account.
			if _, key := outcome.ProducedHandle.affinity(); key != "" {
				bindErr = e.selector.Bind(outcome.ProducedHandle, account)
			}
			return attemptResult, nil
		}
		decision := ClassifyFailure(outcome.Failure)
		return attemptResult, decision.AsError("executor.execute", outcome.Failure)
	})

	return ExecutionOutcome{
		Account:   account,
		Reason:    selection.Reason,
		Sequence:  selection.Sequence,
		Execution: execResult,
		BindError: bindErr,
	}, execErr
}
