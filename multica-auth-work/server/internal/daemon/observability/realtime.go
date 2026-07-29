package observability

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

const RealtimeMeasurementSchemaVersion = "agent-brain.realtime-measurement.v2"
const MaxRealtimeSamples = 4096
const MaxFairnessKeys = 256
const MaxQueueObservations = 4096
const MaxLatencySamples = 4096

type LatencyKind string

const (
	LatencySelection   LatencyKind = "selection"
	LatencyQueue       LatencyKind = "queue"
	LatencyFirstOutput LatencyKind = "first-output"
	LatencyRequest     LatencyKind = "request"
)

type monotonicClock interface {
	Now() time.Time
}

type systemMonotonicClock struct{}

func (systemMonotonicClock) Now() time.Time { return time.Now() }

// RealtimeRecorder records only monotonic durations, queue depths, and
// pseudonymous fairness counters. It has no payload, process output, command,
// environment, account identity, credential, or free-form metadata field.
type RealtimeRecorder struct {
	mu sync.Mutex

	clock   monotonicClock
	started time.Time

	latencies    map[LatencyKind][]time.Duration
	queueDepths  []QueueDepthObservation
	cancellation []time.Duration
	recovery     []time.Duration
	fairness     map[string]FairnessInput
	closed       bool
}

type QueueDepthObservation struct {
	Elapsed time.Duration `json:"elapsed_nanos"`
	Depth   int           `json:"depth"`
}

type LatencyMeasurements struct {
	Kind    LatencyKind     `json:"kind"`
	Samples []time.Duration `json:"samples_nanos"`
}

type FairnessInput struct {
	Slot                  string        `json:"slot"`
	EligibleDuration      time.Duration `json:"eligible_duration_nanos"`
	IndependentSelections int64         `json:"independent_selections"`
	AffinityExclusions    int64         `json:"affinity_exclusions"`
}

type RealtimeMeasurements struct {
	SchemaVersion       string                  `json:"schema_version"`
	ClockSource         string                  `json:"clock_source"`
	ContentCapture      bool                    `json:"content_capture"`
	Elapsed             time.Duration           `json:"elapsed_nanos"`
	Latencies           []LatencyMeasurements   `json:"latencies"`
	QueueDepths         []QueueDepthObservation `json:"queue_depths"`
	PeakQueueDepth      int                     `json:"peak_queue_depth"`
	CancellationRelease []time.Duration         `json:"cancellation_release_nanos"`
	Recovery            []time.Duration         `json:"recovery_nanos"`
	FairnessInputs      []FairnessInput         `json:"fairness_inputs"`
}

func NewRealtimeRecorder() *RealtimeRecorder {
	return newRealtimeRecorder(systemMonotonicClock{})
}

func newRealtimeRecorder(clock monotonicClock) *RealtimeRecorder {
	now := clock.Now()
	return &RealtimeRecorder{
		clock:     clock,
		started:   now,
		latencies: make(map[LatencyKind][]time.Duration),
		fairness:  make(map[string]FairnessInput),
	}
}

// MonotonicSpan measures an interval using one recorder-owned monotonic clock.
// End is exactly-once so duplicate terminal events cannot alter the result.
type MonotonicSpan struct {
	mu sync.Mutex

	recorder *RealtimeRecorder
	started  time.Time
	latency  LatencyKind
	target   spanTarget
	ended    bool
}

type spanTarget uint8

const (
	spanLatency spanTarget = iota + 1
	spanCancellation
	spanRecovery
)

func (r *RealtimeRecorder) BeginLatency(kind LatencyKind) (*MonotonicSpan, error) {
	if !validLatencyKind(kind) {
		return nil, fmt.Errorf("unsupported latency kind %q", kind)
	}
	return r.beginSpan(spanLatency, kind), nil
}

func (r *RealtimeRecorder) BeginCancellationRelease() *MonotonicSpan {
	return r.beginSpan(spanCancellation, "")
}

func (r *RealtimeRecorder) BeginRecovery() *MonotonicSpan {
	return r.beginSpan(spanRecovery, "")
}

func (r *RealtimeRecorder) beginSpan(target spanTarget, latency LatencyKind) *MonotonicSpan {
	return &MonotonicSpan{
		recorder: r,
		started:  r.clock.Now(),
		latency:  latency,
		target:   target,
	}
}

func (s *MonotonicSpan) End() (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return 0, fmt.Errorf("monotonic span already ended")
	}

	duration := s.recorder.clock.Now().Sub(s.started)
	if duration < 0 {
		return 0, fmt.Errorf("monotonic clock moved backwards")
	}
	s.ended = true

	s.recorder.mu.Lock()
	defer s.recorder.mu.Unlock()
	if s.recorder.closed {
		return 0, fmt.Errorf("realtime recorder is closed")
	}
	switch s.target {
	case spanLatency:
		if len(s.recorder.latencies[s.latency]) >= MaxLatencySamples {
			return 0, fmt.Errorf("realtime recorder latency sample limit exceeded")
		}
		s.recorder.latencies[s.latency] = append(s.recorder.latencies[s.latency], duration)
	case spanCancellation:
		if len(s.recorder.cancellation) >= MaxRealtimeSamples {
			return 0, fmt.Errorf("realtime recorder cancellation sample limit exceeded")
		}
		s.recorder.cancellation = append(s.recorder.cancellation, duration)
	case spanRecovery:
		if len(s.recorder.recovery) >= MaxRealtimeSamples {
			return 0, fmt.Errorf("realtime recorder recovery sample limit exceeded")
		}
		s.recorder.recovery = append(s.recorder.recovery, duration)
	default:
		return 0, fmt.Errorf("unsupported monotonic span target")
	}
	return duration, nil
}

func (r *RealtimeRecorder) ObserveQueueDepth(depth int) error {
	if depth < 0 {
		return fmt.Errorf("queue depth must not be negative")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return fmt.Errorf("realtime recorder is closed")
	}
	now := r.clock.Now()
	elapsed := now.Sub(r.started)
	if elapsed < 0 {
		return fmt.Errorf("monotonic clock moved backwards")
	}
	if len(r.queueDepths) >= MaxQueueObservations {
		return fmt.Errorf("realtime recorder queue observation limit exceeded")
	}
	r.queueDepths = append(r.queueDepths, QueueDepthObservation{Elapsed: elapsed, Depth: depth})
	return nil
}

func (r *RealtimeRecorder) ObserveFairness(input FairnessInput) error {
	if !strings.HasPrefix(input.Slot, "slot-") || !safeID(input.Slot, 96) {
		return fmt.Errorf("fairness slot must be ephemeral and pseudonymous")
	}
	if input.EligibleDuration < 0 || input.IndependentSelections < 0 || input.AffinityExclusions < 0 {
		return fmt.Errorf("fairness inputs must not be negative")
	}
	if input.EligibleDuration == 0 && input.IndependentSelections == 0 && input.AffinityExclusions == 0 {
		return fmt.Errorf("empty fairness input")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return fmt.Errorf("realtime recorder is closed")
	}
	if _, exists := r.fairness[input.Slot]; !exists && len(r.fairness) >= MaxFairnessKeys {
		return fmt.Errorf("realtime recorder fairness key limit exceeded")
	}
	current := r.fairness[input.Slot]
	current.Slot = input.Slot
	current.EligibleDuration += input.EligibleDuration
	current.IndependentSelections += input.IndependentSelections
	current.AffinityExclusions += input.AffinityExclusions
	r.fairness[input.Slot] = current
	return nil
}

func (r *RealtimeRecorder) Snapshot() (RealtimeMeasurements, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return RealtimeMeasurements{}, fmt.Errorf("realtime recorder is already closed")
	}
	// Freeze under the same mutex used by every writer. The returned snapshot is
	// therefore a linearization point: no writer can append after this point.
	r.closed = true
	now := r.clock.Now()
	elapsed := now.Sub(r.started)
	if elapsed < 0 {
		return RealtimeMeasurements{}, fmt.Errorf("monotonic clock moved backwards")
	}
	result := RealtimeMeasurements{
		SchemaVersion:       RealtimeMeasurementSchemaVersion,
		ClockSource:         "go-time-monotonic",
		ContentCapture:      false,
		Elapsed:             elapsed,
		QueueDepths:         slices.Clone(r.queueDepths),
		CancellationRelease: slices.Clone(r.cancellation),
		Recovery:            slices.Clone(r.recovery),
	}
	for kind, samples := range r.latencies {
		result.Latencies = append(result.Latencies, LatencyMeasurements{Kind: kind, Samples: slices.Clone(samples)})
	}
	slices.SortFunc(result.Latencies, func(a, b LatencyMeasurements) int {
		return strings.Compare(string(a.Kind), string(b.Kind))
	})
	for _, input := range r.fairness {
		result.FairnessInputs = append(result.FairnessInputs, input)
	}
	slices.SortFunc(result.FairnessInputs, func(a, b FairnessInput) int {
		return strings.Compare(a.Slot, b.Slot)
	})
	for _, observation := range result.QueueDepths {
		result.PeakQueueDepth = maxInt(result.PeakQueueDepth, observation.Depth)
	}
	if err := result.Validate(); err != nil {
		return RealtimeMeasurements{}, err
	}
	return result, nil
}

func (m RealtimeMeasurements) Validate() error {
	if m.SchemaVersion != RealtimeMeasurementSchemaVersion || m.ClockSource != "go-time-monotonic" || m.ContentCapture || m.Elapsed < 0 {
		return fmt.Errorf("invalid real-time measurement identity or clock semantics")
	}
	lastKind := ""
	for _, latency := range m.Latencies {
		if !validLatencyKind(latency.Kind) || string(latency.Kind) <= lastKind || len(latency.Samples) == 0 {
			return fmt.Errorf("invalid or unordered latency measurements")
		}
		lastKind = string(latency.Kind)
		for _, sample := range latency.Samples {
			if sample < 0 {
				return fmt.Errorf("latency sample must not be negative")
			}
		}
	}
	peakQueue := 0
	var lastElapsed time.Duration
	for index, observation := range m.QueueDepths {
		if observation.Elapsed < 0 || observation.Depth < 0 || (index > 0 && observation.Elapsed < lastElapsed) {
			return fmt.Errorf("invalid queue-depth observation")
		}
		lastElapsed = observation.Elapsed
		peakQueue = maxInt(peakQueue, observation.Depth)
	}
	if peakQueue != m.PeakQueueDepth {
		return fmt.Errorf("peak queue depth does not reconcile")
	}
	for _, samples := range [][]time.Duration{m.CancellationRelease, m.Recovery} {
		for _, sample := range samples {
			if sample < 0 {
				return fmt.Errorf("lifecycle duration must not be negative")
			}
		}
	}
	lastSlot := ""
	for _, input := range m.FairnessInputs {
		if !strings.HasPrefix(input.Slot, "slot-") || !safeID(input.Slot, 96) || input.Slot <= lastSlot ||
			input.EligibleDuration < 0 || input.IndependentSelections < 0 || input.AffinityExclusions < 0 {
			return fmt.Errorf("invalid or unordered fairness input")
		}
		lastSlot = input.Slot
	}
	return nil
}

func validLatencyKind(kind LatencyKind) bool {
	switch kind {
	case LatencySelection, LatencyQueue, LatencyFirstOutput, LatencyRequest:
		return true
	default:
		return false
	}
}
