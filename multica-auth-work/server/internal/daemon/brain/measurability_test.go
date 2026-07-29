package brain

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestGatewaySyntheticTelemetryValidationAndFrozenJSONShape(t *testing.T) {
	valid := []struct {
		name                       string
		value                      GatewaySyntheticTelemetry
		maxCount                   uint64
		crossModelFallbackApproved bool
	}{
		{name: "zero scope", value: GatewaySyntheticTelemetry{}, maxCount: 0},
		{
			name: "bounded counters",
			value: GatewaySyntheticTelemetry{
				RetryCount:         4,
				FallbackSameModel:  2,
				FallbackCrossModel: 1,
			},
			maxCount:                   4,
			crossModelFallbackApproved: true,
		},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(test.maxCount, test.crossModelFallbackApproved); err != nil {
				t.Fatalf("valid I3 telemetry rejected: %v", err)
			}
		})
	}

	invalid := []struct {
		name                       string
		value                      GatewaySyntheticTelemetry
		crossModelFallbackApproved bool
	}{
		{name: "retry", value: GatewaySyntheticTelemetry{RetryCount: 5}},
		{name: "same model fallback", value: GatewaySyntheticTelemetry{FallbackSameModel: 5}},
		{
			name:                       "cross model fallback over bound",
			value:                      GatewaySyntheticTelemetry{FallbackCrossModel: 5},
			crossModelFallbackApproved: true,
		},
		{name: "cross model fallback without policy", value: GatewaySyntheticTelemetry{FallbackCrossModel: 1}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(4, test.crossModelFallbackApproved); err == nil {
				t.Fatal("invalid I3 telemetry was accepted")
			}
		})
	}

	assertFrozenPrimitiveJSONShape(t, GatewaySyntheticTelemetry{}, map[string]reflect.Kind{
		"retry_count":          reflect.Uint64,
		"fallback_same_model":  reflect.Uint64,
		"fallback_cross_model": reflect.Uint64,
	})
}

func TestQueueDepthAccessorBoundedSampleContract(t *testing.T) {
	tests := []struct {
		name    string
		sample  QueueDepthSample
		wantErr bool
	}{
		{name: "empty bounded queue", sample: QueueDepthSample{Bound: 20}},
		{name: "queue at bound", sample: QueueDepthSample{Depth: 20, Bound: 20}},
		{name: "missing bound", sample: QueueDepthSample{}, wantErr: true},
		{name: "depth over bound", sample: QueueDepthSample{Depth: 21, Bound: 20}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accessor := fixedQueueDepthAccessor{sample: test.sample}
			got, err := accessor.QueueDepth()
			if err != nil {
				t.Fatalf("read I4 sample: %v", err)
			}
			if got != test.sample {
				t.Fatalf("I4 sample=%+v, want %+v", got, test.sample)
			}
			if err := got.Validate(); (err != nil) != test.wantErr {
				t.Fatalf("I4 validation error=%v, wantErr=%v", err, test.wantErr)
			}
		})
	}

	wantErr := errors.New("synthetic accessor failure")
	if _, err := (fixedQueueDepthAccessor{err: wantErr}).QueueDepth(); !errors.Is(err, wantErr) {
		t.Fatalf("I4 accessor error=%v, want sentinel", err)
	}

	assertFrozenPrimitiveJSONShape(t, QueueDepthSample{}, map[string]reflect.Kind{
		"depth": reflect.Uint64,
		"bound": reflect.Uint64,
	})
}

func TestSteadyStatePredicateExplicitFactsContract(t *testing.T) {
	tests := []struct {
		name    string
		facts   SteadyStateFacts
		wantErr bool
	}{
		{name: "not ready", facts: SteadyStateFacts{}},
		{
			name: "ready but circuit open",
			facts: SteadyStateFacts{
				Ready:             true,
				AccountsReentered: true,
			},
		},
		{
			name: "steady",
			facts: SteadyStateFacts{
				Ready:             true,
				CircuitsClosed:    true,
				AccountsReentered: true,
				Steady:            true,
			},
		},
		{
			name: "false positive steady",
			facts: SteadyStateFacts{
				Ready:  true,
				Steady: true,
			},
			wantErr: true,
		},
		{
			name: "false negative steady",
			facts: SteadyStateFacts{
				Ready:             true,
				CircuitsClosed:    true,
				AccountsReentered: true,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			predicate := fixedSteadyStatePredicate{facts: test.facts}
			got, err := predicate.SteadyState()
			if err != nil {
				t.Fatalf("read I5 facts: %v", err)
			}
			if got != test.facts {
				t.Fatalf("I5 facts=%+v, want %+v", got, test.facts)
			}
			if err := got.Validate(); (err != nil) != test.wantErr {
				t.Fatalf("I5 validation error=%v, wantErr=%v", err, test.wantErr)
			}
		})
	}

	wantErr := errors.New("synthetic predicate failure")
	if _, err := (fixedSteadyStatePredicate{err: wantErr}).SteadyState(); !errors.Is(err, wantErr) {
		t.Fatalf("I5 predicate error=%v, want sentinel", err)
	}

	assertFrozenPrimitiveJSONShape(t, SteadyStateFacts{}, map[string]reflect.Kind{
		"ready":              reflect.Bool,
		"circuits_closed":    reflect.Bool,
		"accounts_reentered": reflect.Bool,
		"steady":             reflect.Bool,
	})
}

func assertFrozenPrimitiveJSONShape(t *testing.T, value any, want map[string]reflect.Kind) {
	t.Helper()
	typ := reflect.TypeOf(value)
	if typ.Kind() != reflect.Struct {
		t.Fatalf("contract type %s is not a struct", typ)
	}
	if typ.NumField() != len(want) {
		t.Fatalf("contract fields=%d, want %d", typ.NumField(), len(want))
	}
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		jsonName := field.Tag.Get("json")
		wantKind, ok := want[jsonName]
		if !ok {
			t.Fatalf("unexpected content-capable contract field %q", jsonName)
		}
		if field.Type.Kind() != wantKind {
			t.Fatalf("contract field %q kind=%s, want %s", jsonName, field.Type.Kind(), wantKind)
		}
	}

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal contract shape: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode contract shape: %v", err)
	}
	if len(fields) != len(want) {
		t.Fatalf("JSON fields=%d, want %d", len(fields), len(want))
	}
	for name := range want {
		if _, ok := fields[name]; !ok {
			t.Fatalf("required JSON field %q is absent", name)
		}
	}
}

type fixedQueueDepthAccessor struct {
	sample QueueDepthSample
	err    error
}

func (a fixedQueueDepthAccessor) QueueDepth() (QueueDepthSample, error) {
	return a.sample, a.err
}

type fixedSteadyStatePredicate struct {
	facts SteadyStateFacts
	err   error
}

func (p fixedSteadyStatePredicate) SteadyState() (SteadyStateFacts, error) {
	return p.facts, p.err
}

var _ QueueDepthAccessor = fixedQueueDepthAccessor{}
var _ SteadyStatePredicate = fixedSteadyStatePredicate{}
