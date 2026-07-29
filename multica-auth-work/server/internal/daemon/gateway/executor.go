package gateway

import (
	"context"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

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
	// Telemetry carries sanitized OmniRoute response telemetry for span emission.
	Telemetry Telemetry
}

// ProviderCall performs one upstream attempt against the selected account. It
// must honor ctx cancellation. attempt is 1-based.
type ProviderCall func(ctx context.Context, account string, attempt int) ProviderOutcome

// ExecutionOutcome reports the account chosen for a request, why it was chosen,
// the coordinator's execution summary, and any post-success binding error
// (which never fails an already-successful response). AccountAlias is the
// pseudonymous form of the selected account (never the raw id), safe for
// selection evidence/telemetry.
type ExecutionOutcome struct {
	Account      string
	AccountAlias string
	Reason       SelectionReason
	Sequence     uint64
	Execution    ExecutionResult
	BindError    error
}

// SelectionRecord is the content-free, pseudonymous evidence of one selection
// decision. It carries the request correlation and a pseudonymous account alias
// (never the raw account id, credential, or any content), so selection order
// can be asserted/audited without exposing account identity.
type SelectionRecord struct {
	RequestID    string
	AccountAlias string
	Reason       SelectionReason
	Sequence     uint64
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
	selector     *Selector
	coordinator  *Coordinator
	recorder     func(SelectionRecord)
	spanRecorder *e2e.Recorder
	protocol     brain.ProtocolFamily
	principal    string
}

// NewExecutor composes a Selector and Coordinator into the gateway execution
// entrypoint. Both are required.
func NewExecutor(selector *Selector, coordinator *Coordinator) (*Executor, error) {
	if selector == nil || coordinator == nil {
		return nil, &GatewayError{Operation: "executor", Class: ErrorInvalidConfiguration}
	}
	return &Executor{selector: selector, coordinator: coordinator}, nil
}

// SetSelectionRecorder installs an optional sink that receives one
// SelectionRecord per Execute call (pseudonymous account alias + correlation).
// It is intended for selection-order evidence/telemetry and tests; nil disables
// recording. The sink must not block.
func (e *Executor) SetSelectionRecorder(fn func(SelectionRecord)) {
	e.recorder = fn
}

// SetSpanRecorder installs an optional e2e.Recorder for emitting ProviderSpanRecord
// on terminal request outcomes. It also sets the protocol family and principal
// pseudonym used for span attributes.
func (e *Executor) SetSpanRecorder(recorder *e2e.Recorder, protocol brain.ProtocolFamily, principalPseudonym string) {
	e.spanRecorder = recorder
	e.protocol = protocol
	e.principal = principalPseudonym
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
	// Observational, pseudonymous selection record: alias only (never the raw
	// account id), plus the request correlation. Emitting it does not affect
	// selection.
	alias := pseudonymizeIdentifier("acct_", account)
	if e.recorder != nil {
		e.recorder(SelectionRecord{
			RequestID:    requestID,
			AccountAlias: alias,
			Reason:       selection.Reason,
			Sequence:     selection.Sequence,
		})
	}

	startedAt := time.Now().UTC()
	var lastTelemetry Telemetry
	var lastOutcome ProviderOutcome

	// The closure runs only on the leader path and only sequentially, so the
	// captured bindErr needs no synchronization.
	var bindErr error
	execResult, execErr := e.coordinator.Execute(ctx, requestID, func(runCtx context.Context, attempt int) (AttemptResult, error) {
		outcome := call(runCtx, account, attempt)
		lastOutcome = outcome
		if outcome.Telemetry.RequestID != "" || outcome.Telemetry.ActualModel != "" {
			lastTelemetry = outcome.Telemetry
		} else if lastTelemetry.RequestID == "" {
			lastTelemetry = outcome.Telemetry
		}

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

	endedAt := time.Now().UTC()
	if endedAt.Before(startedAt) {
		endedAt = startedAt
	}

	if e.spanRecorder != nil {
		telemetry := lastTelemetry
		if telemetry.RequestID == "" {
			telemetry.RequestID = requestID
		}
		if telemetry.PseudonymousAccount == "" {
			telemetry.PseudonymousAccount = alias
		}
		if telemetry.PseudonymousConnection == "" {
			telemetry.PseudonymousConnection = pseudonymizeIdentifier("conn_", account)
		}
		if telemetry.SelectionReason == "" {
			telemetry.SelectionReason = selection.Reason
		}
		if telemetry.Quota == "" {
			telemetry.Quota = QuotaAvailable
		}
		if telemetry.Circuit == "" {
			telemetry.Circuit = CircuitClosed
		}
		if telemetry.RetryCount == 0 && execResult.Attempts > 1 {
			telemetry.RetryCount = execResult.Attempts - 1
		}

		outcomeStr := "ok"
		reasonCode := "completed"
		httpStatus := 200

		if execErr != nil || lastOutcome.Failed {
			outcomeStr = "error"
			decision := ClassifyFailure(lastOutcome.Failure)
			if decision.Class != "" {
				reasonCode = string(decision.Class)
			} else {
				reasonCode = "error"
			}
			if lastOutcome.Failure.StatusCode != 0 {
				httpStatus = lastOutcome.Failure.StatusCode
			} else {
				httpStatus = 500
			}
		}

		principal := e.principal
		if principal == "" {
			principal = pseudonymizeIdentifier("principal_", "default")
		}

		protocol := e.protocol
		if protocol == "" {
			protocol = brain.ProtocolAnthropicMessages
		}

		record := ProviderSpanRecord{
			RequestID:          requestID,
			PrincipalPseudonym: principal,
			Protocol:           protocol,
			Telemetry:          telemetry,
			StartedAt:          startedAt,
			EndedAt:            endedAt,
			Outcome:            outcomeStr,
			ReasonCode:         reasonCode,
			HTTPStatus:         httpStatus,
		}

		if spanErr := EmitProviderSpan(e.spanRecorder, record); spanErr != nil {
			return ExecutionOutcome{
				Account:      account,
				AccountAlias: alias,
				Reason:       selection.Reason,
				Sequence:     selection.Sequence,
				Execution:    execResult,
				BindError:    bindErr,
			}, spanErr
		}
	}

	return ExecutionOutcome{
		Account:      account,
		AccountAlias: alias,
		Reason:       selection.Reason,
		Sequence:     selection.Sequence,
		Execution:    execResult,
		BindError:    bindErr,
	}, execErr
}
