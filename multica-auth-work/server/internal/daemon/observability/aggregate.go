package observability

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// AggregateResultSchemaVersion is distinct from both realtime recorder and
// process measurement schemas. Content is never captured by this layer.
const AggregateResultSchemaVersion = "agent-brain.realtime-aggregate.v1"

type MeasurementPhase string

const (
	PhaseWarmup   MeasurementPhase = "warm-up"
	PhaseSteady   MeasurementPhase = "steady"
	PhaseCooldown MeasurementPhase = "cool-down"
)

type PhaseDurations struct {
	Warmup   time.Duration
	Steady   time.Duration
	Cooldown time.Duration
}

func (d PhaseDurations) Validate() error {
	if d.Warmup <= 0 || d.Steady <= 0 || d.Cooldown <= 0 {
		return fmt.Errorf("all phase durations are required and must be positive")
	}
	max := time.Duration(math.MaxInt64)
	if d.Steady > max-d.Warmup || d.Cooldown > max-d.Warmup-d.Steady {
		return fmt.Errorf("phase duration sum overflows duration")
	}
	return nil
}

func (d PhaseDurations) PhaseAt(elapsed time.Duration) (MeasurementPhase, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	if elapsed < 0 {
		return "", fmt.Errorf("phase elapsed time must not be negative")
	}
	if elapsed < d.Warmup {
		return PhaseWarmup, nil
	}
	if elapsed < d.Warmup+d.Steady {
		return PhaseSteady, nil
	}
	if elapsed < d.Warmup+d.Steady+d.Cooldown {
		return PhaseCooldown, nil
	}
	return "", fmt.Errorf("elapsed time is outside the declared phase windows")
}

type PhaseSample struct {
	Phase MeasurementPhase `json:"phase"`
	Value time.Duration    `json:"value_nanos"`
}

type TaskObservation struct {
	Phase         MeasurementPhase
	Duration      time.Duration
	FirstOutput   *time.Duration
	NoFirstOutput bool
}

type AggregateConfig struct {
	Phases         PhaseDurations
	MinimumSamples int
	ContentCapture bool
}

func (c AggregateConfig) Validate() error {
	if err := c.Phases.Validate(); err != nil {
		return err
	}
	if c.MinimumSamples <= 0 {
		return fmt.Errorf("minimum percentile sample count is required")
	}
	if c.ContentCapture {
		return fmt.Errorf("content capture is forbidden")
	}
	return nil
}

type AggregateResult struct {
	SchemaVersion     string                               `json:"schema_version"`
	ContentCapture    bool                                 `json:"content_capture"`
	PhaseDurations    PhaseDurations                       `json:"phase_durations"`
	MinimumSamples    int                                  `json:"minimum_samples"`
	Latencies         map[LatencyKind]PercentileSet        `json:"latencies"`
	NoFirstOutput     map[MeasurementPhase]int             `json:"no_first_output"`
	E2EByPhase        map[MeasurementPhase][]time.Duration `json:"e2e_by_phase_nanos"`
	CPUPercentByPhase map[MeasurementPhase]float64         `json:"cpu_percent_by_phase"`
	Reconciliation    ReconciliationResult                 `json:"reconciliation"`
}

func (a *RealtimeAggregator) Finalize(hooks ReconciliationHooks) (AggregateResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.finalized {
		return AggregateResult{}, fmt.Errorf("aggregate is already finalized")
	}
	// Freeze before validation: a failed finalize is still a fail-closed,
	// immutable terminal state and cannot be amended by later writers.
	a.finalized = true
	for _, span := range a.recovery {
		if span == nil {
			return AggregateResult{}, fmt.Errorf("finalize STOP: nil recovery span")
		}
		span.mu.Lock()
		open := !span.ended
		if open {
			span.ended = true
		}
		span.mu.Unlock()
		if open {
			return AggregateResult{}, fmt.Errorf("finalize STOP: recovery span remains open")
		}
	}
	result := AggregateResult{SchemaVersion: AggregateResultSchemaVersion, ContentCapture: false, PhaseDurations: a.config.Phases, MinimumSamples: a.config.MinimumSamples, Latencies: make(map[LatencyKind]PercentileSet), NoFirstOutput: make(map[MeasurementPhase]int), E2EByPhase: make(map[MeasurementPhase][]time.Duration), CPUPercentByPhase: make(map[MeasurementPhase]float64)}
	for kind := range a.latencies {
		set, err := a.percentilesLocked(kind, PhaseSteady)
		if err != nil {
			return AggregateResult{}, err
		}
		result.Latencies[kind] = set
	}
	for phase, values := range a.e2e {
		result.E2EByPhase[phase] = append([]time.Duration(nil), values...)
	}
	for phase, count := range a.noFirst {
		result.NoFirstOutput[phase] = count
	}
	for phase, values := range a.cpu {
		if len(values) == 0 {
			continue
		}
		var total float64
		for _, value := range values {
			percent, err := CPUPercent(value.CPU, value.Duration, value.Budget)
			if err != nil {
				return AggregateResult{}, err
			}
			total += percent
		}
		result.CPUPercentByPhase[phase] = total / float64(len(values))
	}
	reconciled, err := a.reconcileLocked(hooks)
	if err != nil {
		return AggregateResult{}, err
	}
	if reconciled.QueueDepth != 0 || !reconciled.SteadyState {
		return AggregateResult{}, fmt.Errorf("finalize STOP: terminal queue/steady-state quorum not satisfied")
	}
	result.Reconciliation = reconciled
	return result, nil
}

type PercentileSet struct {
	P50   time.Duration `json:"p50_nanos"`
	P95   time.Duration `json:"p95_nanos"`
	P99   time.Duration `json:"p99_nanos"`
	Count int           `json:"count"`
}

type ReconciliationResult struct {
	Available              bool   `json:"available"`
	LedgerReconciled       bool   `json:"ledger_reconciled"`
	RetryCount             uint64 `json:"retry_count"`
	FallbackSameModel      uint64 `json:"fallback_same_model"`
	FallbackCrossModel     uint64 `json:"fallback_cross_model"`
	FallbackCycleValidated bool   `json:"fallback_cycle_validated"`
	QueueDepth             int    `json:"queue_depth"`
	SteadyState            bool   `json:"steady_state"`
	FailedReal             uint64 `json:"failed_real"`
}

// ReconciliationHooks consumes the frozen brain-owned I5 facts contract.
type ReconciliationHooks struct {
	Ledger  func() (brain.LedgerCounters, error)
	Gateway func() (retryCount, fallbackSameModel, fallbackCrossModel uint64, cycleValidated bool, err error)
	Queue   func() (depth int, err error)
	Steady  func() (brain.SteadyStateFacts, error)
}

type RealtimeAggregator struct {
	mu        sync.Mutex
	config    AggregateConfig
	latencies map[LatencyKind][]PhaseSample
	e2e       map[MeasurementPhase][]time.Duration
	noFirst   map[MeasurementPhase]int
	cpu       map[MeasurementPhase][]cpuObservation
	recovery  []*RecoverySpan
	finalized bool
}

type cpuObservation struct {
	CPU, Duration time.Duration
	Budget        float64
}

func NewRealtimeAggregator(config AggregateConfig) (*RealtimeAggregator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &RealtimeAggregator{config: config, latencies: make(map[LatencyKind][]PhaseSample), e2e: make(map[MeasurementPhase][]time.Duration), noFirst: make(map[MeasurementPhase]int), cpu: make(map[MeasurementPhase][]cpuObservation)}, nil
}

func (a *RealtimeAggregator) AddLatency(kind LatencyKind, phase MeasurementPhase, value time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.finalized {
		return fmt.Errorf("aggregate is finalized")
	}
	if !validLatencyKind(kind) || !validPhase(phase) || value < 0 {
		return fmt.Errorf("invalid phase-tagged latency sample")
	}
	a.latencies[kind] = append(a.latencies[kind], PhaseSample{Phase: phase, Value: value})
	return nil
}

func (a *RealtimeAggregator) AddTask(observation TaskObservation) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.finalized {
		return fmt.Errorf("aggregate is finalized")
	}
	if !validPhase(observation.Phase) || observation.Duration <= 0 {
		return fmt.Errorf("task phase and positive duration are required")
	}
	if observation.FirstOutput != nil && (*observation.FirstOutput < 0 || *observation.FirstOutput > observation.Duration) {
		return fmt.Errorf("invalid first-output duration")
	}
	a.e2e[observation.Phase] = append(a.e2e[observation.Phase], observation.Duration)
	if observation.NoFirstOutput || observation.FirstOutput == nil {
		a.noFirst[observation.Phase]++
	}
	return nil
}

func (a *RealtimeAggregator) AddCPU(phase MeasurementPhase, cpuTime, duration time.Duration, budgetCores float64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.finalized {
		return fmt.Errorf("aggregate is finalized")
	}
	if !validPhase(phase) {
		return fmt.Errorf("invalid CPU phase")
	}
	if cpuTime < 0 || duration <= 0 || budgetCores <= 0 {
		return fmt.Errorf("CPU measurement STOP: positive CPU time, duration, and budget are required")
	}
	a.cpu[phase] = append(a.cpu[phase], cpuObservation{CPU: cpuTime, Duration: duration, Budget: budgetCores})
	return nil
}

func (a *RealtimeAggregator) Percentiles(kind LatencyKind, phase MeasurementPhase) (PercentileSet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.percentilesLocked(kind, phase)
}

func (a *RealtimeAggregator) percentilesLocked(kind LatencyKind, phase MeasurementPhase) (PercentileSet, error) {
	if !validLatencyKind(kind) || !validPhase(phase) {
		return PercentileSet{}, fmt.Errorf("invalid percentile input")
	}
	values := make([]time.Duration, 0)
	for _, sample := range a.latencies[kind] {
		if sample.Phase == phase {
			values = append(values, sample.Value)
		}
	}
	return aggregateNearestRank(values, a.config.MinimumSamples)
}

func aggregateNearestRank(values []time.Duration, minimum int) (PercentileSet, error) {
	if minimum <= 0 || len(values) < minimum {
		return PercentileSet{}, fmt.Errorf("percentile STOP: minimum sample count not met")
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	rank := func(p int) time.Duration {
		n := len(values)
		whole, remainder := n/100, n%100
		oneBased := p*whole + (p*remainder+99)/100
		return values[oneBased-1]
	}
	return PercentileSet{P50: rank(50), P95: rank(95), P99: rank(99), Count: len(values)}, nil
}

func CPUPercent(cpuTime, duration time.Duration, budgetCores float64) (float64, error) {
	if cpuTime < 0 || duration <= 0 || budgetCores <= 0 || math.IsNaN(budgetCores) || math.IsInf(budgetCores, 0) {
		return 0, fmt.Errorf("CPU percentage STOP: missing, zero, or negative duration/budget")
	}
	percent := float64(cpuTime) / float64(duration) / budgetCores * 100
	if math.IsNaN(percent) || math.IsInf(percent, 0) {
		return 0, fmt.Errorf("CPU percentage STOP: arithmetic overflow")
	}
	return percent, nil
}

type RecoverySpan struct {
	mu      sync.Mutex
	max     time.Duration
	started time.Time
	ended   bool
}

func BeginBoundedRecovery(max time.Duration, now time.Time) (*RecoverySpan, error) {
	if max <= 0 {
		return nil, fmt.Errorf("recovery max timeout is required")
	}
	return &RecoverySpan{max: max, started: now}, nil
}

func (a *RealtimeAggregator) BeginRecovery(max time.Duration, now time.Time) (*RecoverySpan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.finalized {
		return nil, fmt.Errorf("aggregate is finalized")
	}
	span, err := BeginBoundedRecovery(max, now)
	if err != nil {
		return nil, err
	}
	a.recovery = append(a.recovery, span)
	return span, nil
}

func (s *RecoverySpan) End(elapsed time.Duration, facts brain.SteadyStateFacts) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s == nil || s.ended {
		return 0, fmt.Errorf("recovery span is already closed")
	}
	s.ended = true
	if err := facts.Validate(); err != nil {
		return elapsed, fmt.Errorf("recovery STOP: invalid steady-state facts: %w", err)
	}
	if elapsed < 0 || elapsed > s.max || !facts.Steady {
		return elapsed, fmt.Errorf("recovery STOP: steady state not reached within bound")
	}
	return elapsed, nil
}

func (a *RealtimeAggregator) Reconcile(h ReconciliationHooks) (ReconciliationResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.reconcileLocked(h)
}

func (a *RealtimeAggregator) reconcileLocked(h ReconciliationHooks) (ReconciliationResult, error) {
	var out ReconciliationResult
	if h.Ledger == nil || h.Gateway == nil || h.Queue == nil || h.Steady == nil {
		return out, fmt.Errorf("reconciliation STOP: all I1-I5 hooks are required")
	}
	if h.Ledger != nil {
		ledger, err := h.Ledger()
		if err != nil {
			return out, err
		}
		if err := ledger.Reconcile(); err != nil {
			return out, fmt.Errorf("ledger reconciliation STOP: %w", err)
		}
		out.FailedReal = ledger.Failed - ledger.FailedInjected
		out.LedgerReconciled = true
		out.Available = true
		if !out.LedgerReconciled {
			return out, fmt.Errorf("ledger reconciliation STOP")
		}
	}
	if h.Gateway != nil {
		var err error
		out.RetryCount, out.FallbackSameModel, out.FallbackCrossModel, out.FallbackCycleValidated, err = h.Gateway()
		if err != nil {
			return out, err
		}
		if !out.FallbackCycleValidated {
			return out, fmt.Errorf("fallback-cycle validation STOP")
		}
	}
	if h.Queue != nil {
		var err error
		out.QueueDepth, err = h.Queue()
		if err != nil || out.QueueDepth < 0 {
			return out, fmt.Errorf("queue reconciliation STOP")
		}
	}
	if h.Steady != nil {
		facts, err := h.Steady()
		if err != nil {
			return out, err
		}
		if err := facts.Validate(); err != nil {
			return out, fmt.Errorf("steady-state reconciliation STOP: %w", err)
		}
		out.SteadyState = facts.Steady
	}
	return out, nil
}

func validPhase(phase MeasurementPhase) bool {
	return phase == PhaseWarmup || phase == PhaseSteady || phase == PhaseCooldown
}
