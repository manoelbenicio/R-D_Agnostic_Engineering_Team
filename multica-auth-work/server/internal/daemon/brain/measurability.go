package brain

import "fmt"

// GatewaySyntheticTelemetry is the frozen, content-free I3 handoff from a
// gateway producer to an observability consumer. Counts cover one bounded
// synthetic measurement scope. Request, model, route, account, correlation,
// error, and content identifiers are deliberately absent.
//
// Cross-model fallback remains invalid unless the caller separately proves an
// approved ordered policy and a cycle-free bounded chain. Those proofs are not
// telemetry fields and must fail closed outside this value.
type GatewaySyntheticTelemetry struct {
	RetryCount         uint64 `json:"retry_count"`
	FallbackSameModel  uint64 `json:"fallback_same_model"`
	FallbackCrossModel uint64 `json:"fallback_cross_model"`
}

// Validate checks every I3 counter against the caller's declared numeric
// bound and rejects cross-model fallback unless an approved ordered policy is
// asserted separately. The bound belongs to the synthetic run contract; this
// package does not invent a capacity or acceptance threshold.
func (t GatewaySyntheticTelemetry) Validate(maxCount uint64, crossModelFallbackApproved bool) error {
	if t.RetryCount > maxCount {
		return fmt.Errorf("retry_count=%d exceeds declared bound=%d", t.RetryCount, maxCount)
	}
	if t.FallbackSameModel > maxCount {
		return fmt.Errorf("fallback_same_model=%d exceeds declared bound=%d", t.FallbackSameModel, maxCount)
	}
	if t.FallbackCrossModel > maxCount {
		return fmt.Errorf("fallback_cross_model=%d exceeds declared bound=%d", t.FallbackCrossModel, maxCount)
	}
	if t.FallbackCrossModel != 0 && !crossModelFallbackApproved {
		return fmt.Errorf("cross-model fallback requires an approved ordered policy")
	}
	return nil
}

// QueueDepthSample is the frozen, content-free I4 sample. Bound is the finite
// queue limit in effect for the same instantaneous sample as Depth. Sampling
// time and cadence belong to the consumer and are intentionally not carried in
// this neutral value.
type QueueDepthSample struct {
	Depth uint64 `json:"depth"`
	Bound uint64 `json:"bound"`
}

func (s QueueDepthSample) Validate() error {
	if s.Bound == 0 {
		return fmt.Errorf("queue depth sample requires a positive bound")
	}
	if s.Depth > s.Bound {
		return fmt.Errorf("queue depth=%d exceeds bound=%d", s.Depth, s.Bound)
	}
	return nil
}

// QueueDepthAccessor is the frozen I4 producer boundary. Implementations are
// read-only and return one bounded numeric sample without queue item data.
type QueueDepthAccessor interface {
	QueueDepth() (QueueDepthSample, error)
}

// SteadyStateFacts is the frozen, content-free I5 result. Steady is true if and
// only if readiness is true, all relevant circuits are closed, and accounts
// affected by the synthetic recovery case have re-entered eligibility.
type SteadyStateFacts struct {
	Ready             bool `json:"ready"`
	CircuitsClosed    bool `json:"circuits_closed"`
	AccountsReentered bool `json:"accounts_reentered"`
	Steady            bool `json:"steady"`
}

func (f SteadyStateFacts) Validate() error {
	wantSteady := f.Ready && f.CircuitsClosed && f.AccountsReentered
	if f.Steady != wantSteady {
		return fmt.Errorf("steady-state fact does not equal readiness conjunction")
	}
	return nil
}

// SteadyStatePredicate is the frozen I5 producer boundary. It reports neutral
// facts only; bounded recovery timing and STOP decisions remain consumer-owned.
type SteadyStatePredicate interface {
	SteadyState() (SteadyStateFacts, error)
}
