package gateway

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestGatewaySyntheticTelemetryProducesDeterministicCounters(t *testing.T) {
	snapshot, err := buildSnapshot(validModelsDocument(), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	snapshot = cloneSnapshot(snapshot)
	primary := brain.RouteModel("agy/claude-opus-4-6-thinking")
	tests := []struct {
		name          string
		telemetry     Telemetry
		crossApproved bool
		want          brain.GatewaySyntheticTelemetry
	}{
		{
			name:      "retry without fallback",
			telemetry: Telemetry{ActualModel: primary, RetryCount: 2},
			want:      brain.GatewaySyntheticTelemetry{RetryCount: 2},
		},
		{
			name:      "same model fallback",
			telemetry: Telemetry{ActualModel: primary, FallbackUsed: true},
			want:      brain.GatewaySyntheticTelemetry{FallbackSameModel: 1},
		},
		{
			name:          "approved cross model fallback",
			telemetry:     Telemetry{ActualModel: "synthetic/fallback", FallbackUsed: true},
			crossApproved: true,
			want:          brain.GatewaySyntheticTelemetry{FallbackCrossModel: 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, measureErr := test.telemetry.GatewaySyntheticTelemetry(
				primary,
				2,
				test.crossApproved,
				snapshot.FallbackCycleProof(),
			)
			if measureErr != nil {
				t.Fatalf("GatewaySyntheticTelemetry: %v", measureErr)
			}
			if got != test.want {
				t.Fatalf("counters=%+v, want %+v", got, test.want)
			}
		})
	}
}

func TestGatewaySyntheticTelemetryFailsClosedOnBoundApprovalAndCycleProof(t *testing.T) {
	validSnapshot, err := buildSnapshot(validModelsDocument(), time.Unix(0, 0))
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	primary := brain.RouteModel("agy/claude-opus-4-6-thinking")
	tests := []struct {
		name      string
		telemetry Telemetry
		bound     uint64
		approved  bool
		proof     FallbackCycleProof
		wantClass ErrorClass
	}{
		{
			name:      "retry counter over declared bound",
			telemetry: Telemetry{ActualModel: primary, RetryCount: 2},
			bound:     1,
			proof:     validSnapshot.FallbackCycleProof(),
			wantClass: ErrorInvalidConfiguration,
		},
		{
			name:      "cross model fallback without approval",
			telemetry: Telemetry{ActualModel: "synthetic/fallback", FallbackUsed: true},
			bound:     1,
			proof:     validSnapshot.FallbackCycleProof(),
			wantClass: ErrorInvalidConfiguration,
		},
		{
			name:      "missing separate cycle proof",
			telemetry: Telemetry{ActualModel: primary},
			bound:     1,
			proof:     FallbackCycleProof{},
			wantClass: ErrorInvalidConfiguration,
		},
		{
			name:      "negative unsanitized retry count",
			telemetry: Telemetry{ActualModel: primary, RetryCount: -1},
			bound:     1,
			proof:     validSnapshot.FallbackCycleProof(),
			wantClass: ErrorProtocol,
		},
		{
			name:      "unreported upstream model change",
			telemetry: Telemetry{ActualModel: "synthetic/fallback"},
			bound:     1,
			approved:  true,
			proof:     validSnapshot.FallbackCycleProof(),
			wantClass: ErrorProtocol,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, measureErr := test.telemetry.GatewaySyntheticTelemetry(primary, test.bound, test.approved, test.proof)
			if !IsErrorClass(measureErr, test.wantClass) {
				t.Fatalf("error=%v, want class %s", measureErr, test.wantClass)
			}
			if got != (brain.GatewaySyntheticTelemetry{}) {
				t.Fatalf("failed measurement exposed counters: %+v", got)
			}
		})
	}

	cycleDocument := fallbackGraphDocument(
		[]string{"synthetic/a", "synthetic/b"},
		map[string][]string{"synthetic/a": {"synthetic/b"}, "synthetic/b": {"synthetic/a"}},
	)
	failedSnapshot, buildErr := buildSnapshot(cycleDocument, time.Unix(0, 0))
	if !IsErrorClass(buildErr, ErrorProtocol) {
		t.Fatalf("cyclic registry was not rejected: %v", buildErr)
	}
	if _, measureErr := (Telemetry{ActualModel: primary}).GatewaySyntheticTelemetry(
		primary, 1, false, failedSnapshot.FallbackCycleProof(),
	); !IsErrorClass(measureErr, ErrorInvalidConfiguration) {
		t.Fatalf("failed cycle validation exposed telemetry: %v", measureErr)
	}
}

func TestQueueDepthFuncValidatesBoundsAndReadsExactlyOnce(t *testing.T) {
	tests := []struct {
		name    string
		sample  brain.QueueDepthSample
		wantErr bool
	}{
		{name: "empty bounded queue", sample: brain.QueueDepthSample{Bound: 20}},
		{name: "queue at bound", sample: brain.QueueDepthSample{Depth: 20, Bound: 20}},
		{name: "zero bound", sample: brain.QueueDepthSample{}, wantErr: true},
		{name: "depth above bound", sample: brain.QueueDepthSample{Depth: 21, Bound: 20}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			accessor := QueueDepthFunc(func() (brain.QueueDepthSample, error) {
				calls++
				return test.sample, nil
			})
			got, err := accessor.QueueDepth()
			if test.wantErr {
				if !IsErrorClass(err, ErrorInvalidConfiguration) {
					t.Fatalf("QueueDepth error=%v", err)
				}
				if got != (brain.QueueDepthSample{}) {
					t.Fatalf("invalid sample exposed: %+v", got)
				}
			} else if err != nil || got != test.sample {
				t.Fatalf("QueueDepth=(%+v, %v), want (%+v, nil)", got, err, test.sample)
			}
			if calls != 1 {
				t.Fatalf("source called %d times, want exactly once", calls)
			}
		})
	}

	calls := 0
	accessor := QueueDepthFunc(func() (brain.QueueDepthSample, error) {
		calls++
		return brain.QueueDepthSample{}, context.Canceled
	})
	if _, err := accessor.QueueDepth(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was not preserved: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cancelled source called %d times, want exactly once", calls)
	}
	if _, err := (QueueDepthFunc(nil)).QueueDepth(); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("nil source did not fail closed: %v", err)
	}
}

func TestSteadyStateProducerUsesFullConjunction(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		ready := bits&1 != 0
		circuitsClosed := bits&2 != 0
		accountsReentered := bits&4 != 0
		calls := 0
		producer, err := NewSteadyStateProducer(func() (bool, bool, bool, error) {
			calls++
			return ready, circuitsClosed, accountsReentered, nil
		})
		if err != nil {
			t.Fatalf("NewSteadyStateProducer: %v", err)
		}
		facts, err := producer.SteadyState()
		if err != nil {
			t.Fatalf("SteadyState bits=%03b: %v", bits, err)
		}
		wantSteady := ready && circuitsClosed && accountsReentered
		if facts.Ready != ready || facts.CircuitsClosed != circuitsClosed || facts.AccountsReentered != accountsReentered || facts.Steady != wantSteady {
			t.Fatalf("facts bits=%03b: %+v", bits, facts)
		}
		if calls != 1 {
			t.Fatalf("source bits=%03b called %d times, want exactly once", bits, calls)
		}
	}
}

func TestSteadyStateProducerPreservesCancellationAndSamplesExactlyOnce(t *testing.T) {
	var cancelCalls atomic.Int64
	cancelled, err := NewSteadyStateProducer(func() (bool, bool, bool, error) {
		cancelCalls.Add(1)
		return false, false, false, context.Canceled
	})
	if err != nil {
		t.Fatalf("NewSteadyStateProducer: %v", err)
	}
	if facts, steadyErr := cancelled.SteadyState(); !errors.Is(steadyErr, context.Canceled) || facts != (brain.SteadyStateFacts{}) {
		t.Fatalf("cancelled SteadyState=(%+v, %v)", facts, steadyErr)
	}
	if cancelCalls.Load() != 1 {
		t.Fatalf("cancelled source called %d times, want exactly once", cancelCalls.Load())
	}

	const callers = 64
	var calls atomic.Int64
	producer, err := NewSteadyStateProducer(func() (bool, bool, bool, error) {
		calls.Add(1)
		return true, true, true, nil
	})
	if err != nil {
		t.Fatalf("NewSteadyStateProducer: %v", err)
	}
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			facts, steadyErr := producer.SteadyState()
			if steadyErr != nil || !facts.Steady {
				t.Errorf("SteadyState=(%+v, %v)", facts, steadyErr)
			}
		}()
	}
	wait.Wait()
	if calls.Load() != callers {
		t.Fatalf("source called %d times for %d measurements", calls.Load(), callers)
	}

	if _, err := NewSteadyStateProducer(nil); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("nil source constructor did not fail closed: %v", err)
	}
	var nilProducer *SteadyStateProducer
	if _, err := nilProducer.SteadyState(); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("nil producer did not fail closed: %v", err)
	}
}
