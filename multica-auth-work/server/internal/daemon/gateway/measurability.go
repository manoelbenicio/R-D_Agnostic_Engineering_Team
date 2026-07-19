package gateway

import "github.com/multica-ai/multica/server/internal/daemon/brain"

// GatewaySyntheticTelemetry converts one terminal, content-free gateway
// telemetry record into the frozen I3 counters. The registry proof and the
// ordered cross-model approval assertion remain separate from the neutral
// value and both fail closed before it is exposed.
func (t Telemetry) GatewaySyntheticTelemetry(
	primaryModel brain.RouteModel,
	bound uint64,
	crossModelFallbackApproved bool,
	cycleProof FallbackCycleProof,
) (brain.GatewaySyntheticTelemetry, error) {
	if !cycleProof.accepted {
		return brain.GatewaySyntheticTelemetry{}, measurementConfigurationError("telemetry.measure")
	}
	primary, err := brain.ParseRouteModel(string(primaryModel))
	if err != nil {
		return brain.GatewaySyntheticTelemetry{}, measurementConfigurationError("telemetry.measure")
	}
	if t.RetryCount < 0 {
		return brain.GatewaySyntheticTelemetry{}, measurementProtocolError("telemetry.measure")
	}

	result := brain.GatewaySyntheticTelemetry{RetryCount: uint64(t.RetryCount)}
	if t.FallbackUsed {
		actual, parseErr := brain.ParseRouteModel(string(t.ActualModel))
		if parseErr != nil {
			return brain.GatewaySyntheticTelemetry{}, measurementProtocolError("telemetry.measure")
		}
		if actual == primary {
			result.FallbackSameModel = 1
		} else {
			result.FallbackCrossModel = 1
		}
	} else if t.ActualModel != "" && t.ActualModel != primary {
		// An upstream model change without the fallback signal would otherwise
		// undercount cross-model fallback, so fail closed.
		return brain.GatewaySyntheticTelemetry{}, measurementProtocolError("telemetry.measure")
	}
	if err := result.Validate(bound, crossModelFallbackApproved); err != nil {
		return brain.GatewaySyntheticTelemetry{}, measurementConfigurationError("telemetry.measure")
	}
	return result, nil
}

// QueueDepthFunc adapts an existing content-free queue sampler to the frozen
// I4 boundary. It neither creates a queue nor adds timestamps or cadence.
type QueueDepthFunc func() (brain.QueueDepthSample, error)

func (f QueueDepthFunc) QueueDepth() (brain.QueueDepthSample, error) {
	if f == nil {
		return brain.QueueDepthSample{}, measurementConfigurationError("queue_depth.measure")
	}
	sample, err := f()
	if err != nil {
		return brain.QueueDepthSample{}, err
	}
	if err := sample.Validate(); err != nil {
		return brain.QueueDepthSample{}, measurementConfigurationError("queue_depth.measure")
	}
	return sample, nil
}

// SteadyStateSource returns one atomic set of the three gateway-owned I5 facts.
// Cancellation is represented by its returned error because the frozen brain
// interface deliberately has no context or timing fields.
type SteadyStateSource func() (ready, circuitsClosed, accountsReentered bool, err error)

// SteadyStateProducer evaluates the full I5 conjunction from exactly one
// source call. Recovery deadlines and cadence remain consumer-owned.
type SteadyStateProducer struct {
	source SteadyStateSource
}

func NewSteadyStateProducer(source SteadyStateSource) (*SteadyStateProducer, error) {
	if source == nil {
		return nil, measurementConfigurationError("steady_state.measure")
	}
	return &SteadyStateProducer{source: source}, nil
}

func (p *SteadyStateProducer) SteadyState() (brain.SteadyStateFacts, error) {
	if p == nil || p.source == nil {
		return brain.SteadyStateFacts{}, measurementConfigurationError("steady_state.measure")
	}
	ready, circuitsClosed, accountsReentered, err := p.source()
	if err != nil {
		return brain.SteadyStateFacts{}, err
	}
	facts := brain.SteadyStateFacts{
		Ready:             ready,
		CircuitsClosed:    circuitsClosed,
		AccountsReentered: accountsReentered,
		Steady:            ready && circuitsClosed && accountsReentered,
	}
	if err := facts.Validate(); err != nil {
		return brain.SteadyStateFacts{}, measurementProtocolError("steady_state.measure")
	}
	return facts, nil
}

func measurementConfigurationError(operation string) error {
	return &GatewayError{Operation: operation, Class: ErrorInvalidConfiguration}
}

func measurementProtocolError(operation string) error {
	return &GatewayError{Operation: operation, Class: ErrorProtocol}
}

var _ brain.QueueDepthAccessor = QueueDepthFunc(nil)
var _ brain.SteadyStatePredicate = (*SteadyStateProducer)(nil)
