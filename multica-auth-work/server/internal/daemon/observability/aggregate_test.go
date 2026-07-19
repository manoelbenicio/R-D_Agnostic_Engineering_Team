package observability

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestPhaseDurationsRequireExplicitWindows(t *testing.T) {
	if _, err := (PhaseDurations{}).PhaseAt(time.Millisecond); err == nil {
		t.Fatal("missing phase durations were accepted")
	}
	d := PhaseDurations{Warmup: time.Second, Steady: 2 * time.Second, Cooldown: time.Second}
	for elapsed, want := range map[time.Duration]MeasurementPhase{0: PhaseWarmup, time.Second: PhaseSteady, 3 * time.Second: PhaseCooldown} {
		got, err := d.PhaseAt(elapsed)
		if err != nil || got != want {
			t.Fatalf("phase at %v: got %q err=%v", elapsed, got, err)
		}
	}
}

func TestNearestRankAndNoFirstOutput(t *testing.T) {
	a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 4})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []time.Duration{4, 1, 3, 2} {
		if err := a.AddLatency(LatencyRequest, PhaseSteady, value); err != nil {
			t.Fatal(err)
		}
	}
	set, err := a.Percentiles(LatencyRequest, PhaseSteady)
	if err != nil || set.P50 != 2 || set.P95 != 4 || set.P99 != 4 {
		t.Fatalf("nearest rank: %+v err=%v", set, err)
	}
	if err := a.AddTask(TaskObservation{Phase: PhaseSteady, Duration: time.Second}); err != nil {
		t.Fatal(err)
	}
	if got := a.noFirst[PhaseSteady]; got != 1 {
		t.Fatalf("no-first-output count=%d", got)
	}
}

func TestAggregateGuardsAndReconciliationHooks(t *testing.T) {
	if _, err := CPUPercent(time.Second, 0, 2); err == nil {
		t.Fatal("zero duration CPU guard accepted")
	}
	if _, err := CPUPercent(time.Second, time.Second, 0); err == nil {
		t.Fatal("zero CPU budget guard accepted")
	}
	span, err := BeginBoundedRecovery(time.Second, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := span.End(2*time.Second, steadyFacts(true, true, true, true)); err == nil {
		t.Fatal("recovery timeout accepted")
	}
	result, err := (&RealtimeAggregator{}).Reconcile(ReconciliationHooks{
		Ledger: func() (brain.LedgerCounters, error) {
			return brain.LedgerCounters{Offered: 4, Admitted: 2, Rejected: 1, Overloaded: 1, Started: 2, Completed: 2}, nil
		},
		Gateway: func() (uint64, uint64, uint64, bool, error) { return 1, 1, 0, true, nil },
		Queue:   func() (int, error) { return 0, nil },
		Steady:  func() (brain.SteadyStateFacts, error) { return steadyFacts(true, true, true, true), nil },
	})
	if err != nil || !result.Available || !result.LedgerReconciled || result.FallbackCrossModel != 0 {
		t.Fatalf("unexpected reconciliation: %+v err=%v", result, err)
	}
}

func TestAggregateFinalizeIsContentOffAndUsesTaskDuration(t *testing.T) {
	a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddLatency(LatencyRequest, PhaseSteady, 3*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := a.AddTask(TaskObservation{Phase: PhaseSteady, Duration: 9 * time.Millisecond, FirstOutput: ptrDuration(2 * time.Millisecond)}); err != nil {
		t.Fatal(err)
	}
	if err := a.AddCPU(PhaseSteady, time.Millisecond, 10*time.Millisecond, 2); err != nil {
		t.Fatal(err)
	}
	result, err := a.Finalize(testHooks())
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != AggregateResultSchemaVersion || result.ContentCapture || len(result.E2EByPhase[PhaseSteady]) != 1 || result.E2EByPhase[PhaseSteady][0] != 9*time.Millisecond {
		t.Fatalf("unexpected aggregate result: %+v", result)
	}
}

func testHooks() ReconciliationHooks {
	return ReconciliationHooks{
		Ledger: func() (brain.LedgerCounters, error) {
			return brain.LedgerCounters{Offered: 1, Admitted: 1, Started: 1, Completed: 1}, nil
		},
		Gateway: func() (uint64, uint64, uint64, bool, error) { return 0, 0, 0, true, nil },
		Queue:   func() (int, error) { return 0, nil },
		Steady:  func() (brain.SteadyStateFacts, error) { return steadyFacts(true, true, true, true), nil },
	}
}

func steadyFacts(ready, circuitsClosed, accountsReentered, steady bool) brain.SteadyStateFacts {
	return brain.SteadyStateFacts{Ready: ready, CircuitsClosed: circuitsClosed, AccountsReentered: accountsReentered, Steady: steady}
}

func TestSteadyFactsValidationRejectsEachMissingReadinessInput(t *testing.T) {
	cases := []struct {
		name  string
		facts brain.SteadyStateFacts
	}{
		{"readiness", steadyFacts(false, true, true, true)},
		{"circuits", steadyFacts(true, false, true, true)},
		{"accounts", steadyFacts(true, true, false, true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.facts.Validate(); err == nil {
				t.Fatal("invalid claimed steady facts accepted")
			}
			span, err := BeginBoundedRecovery(time.Second, time.Unix(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := span.End(time.Millisecond, tc.facts); err == nil {
				t.Fatal("recovery accepted invalid steady facts")
			}
			h := testHooks()
			h.Steady = func() (brain.SteadyStateFacts, error) { return tc.facts, nil }
			a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 1})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := a.Finalize(h); err == nil {
				t.Fatal("finalize accepted invalid steady facts")
			}
		})
	}
}

func TestSteadyFactsAllTrueCompletesRecovery(t *testing.T) {
	facts := steadyFacts(true, true, true, true)
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	span, err := BeginBoundedRecovery(time.Second, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if elapsed, err := span.End(time.Millisecond, facts); err != nil || elapsed != time.Millisecond {
		t.Fatalf("all-true steady facts did not complete recovery: elapsed=%v err=%v", elapsed, err)
	}
}

func TestAggregateFinalizeClosesWritesAndOpenRecovery(t *testing.T) {
	a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 1})
	if err != nil {
		t.Fatal(err)
	}
	span, err := a.BeginRecovery(time.Second, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Finalize(testHooks()); err == nil {
		t.Fatal("open recovery was accepted")
	}
	if err := a.AddTask(TaskObservation{Phase: PhaseSteady, Duration: time.Second}); err == nil {
		t.Fatal("post-finalize write was accepted")
	}
	if _, err := span.End(time.Millisecond, steadyFacts(true, true, true, true)); err == nil {
		t.Fatal("recovery span remained writable after fail-closed finalize")
	}
	if _, err := a.Finalize(testHooks()); err == nil {
		t.Fatal("second finalize was accepted")
	}
}

func TestLedgerContractRejectsInvalidSnapshotsAndAcceptsPreStartTerminal(t *testing.T) {
	valid := brain.LedgerCounters{Offered: 3, Admitted: 2, Rejected: 1, Started: 1, Completed: 1, Failed: 1}
	if err := valid.Reconcile(); err != nil {
		t.Fatalf("valid pre-start terminal rejected: %v", err)
	}
	startedTooHigh := valid
	startedTooHigh.Started = 3
	if err := startedTooHigh.Reconcile(); err == nil {
		t.Fatal("Started>Admitted accepted")
	}
	injectedTooHigh := valid
	injectedTooHigh.FailedInjected = 2
	if err := injectedTooHigh.Reconcile(); err == nil {
		t.Fatal("FailedInjected>Failed accepted")
	}
}

func TestFinalizeRequiresTerminalQueueAndSteadyQuorum(t *testing.T) {
	for name, hooks := range map[string]ReconciliationHooks{
		"queue": func() ReconciliationHooks {
			h := testHooks()
			h.Queue = func() (int, error) { return 1, nil }
			return h
		}(),
		"steady": func() ReconciliationHooks {
			h := testHooks()
			h.Steady = func() (brain.SteadyStateFacts, error) { return steadyFacts(false, true, true, false), nil }
			return h
		}(),
	} {
		a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 1})
		if err != nil {
			t.Fatal(err)
		}
		if err := a.AddLatency(LatencyRequest, PhaseSteady, time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Finalize(hooks); err == nil {
			t.Fatalf("%s quorum failure accepted", name)
		}
	}
}

func TestAggregateFinalizeRequiresAllReconciliationHooks(t *testing.T) {
	a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Finalize(ReconciliationHooks{}); err == nil {
		t.Fatal("missing I1-I5 hooks were accepted")
	}
}

func TestAggregateOverflowAndMinimumSampleStops(t *testing.T) {
	if _, err := (PhaseDurations{Warmup: time.Duration(1<<63 - 1), Steady: 1, Cooldown: 1}).PhaseAt(0); err == nil {
		t.Fatal("phase duration overflow was accepted")
	}
	if _, err := CPUPercent(time.Duration(1<<63-1), time.Nanosecond, 1e-320); err == nil {
		t.Fatal("CPU arithmetic overflow was accepted")
	}
	a, err := NewRealtimeAggregator(AggregateConfig{Phases: PhaseDurations{time.Second, time.Second, time.Second}, MinimumSamples: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddLatency(LatencyRequest, PhaseSteady, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	set, err := a.Percentiles(LatencyRequest, PhaseSteady)
	if err == nil || set != (PercentileSet{}) {
		t.Fatalf("below-minimum percentile did not STOP: %+v %v", set, err)
	}
}

func ptrDuration(value time.Duration) *time.Duration { return &value }
