package brain

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
)

func TestLifecycleCapacityReconcilesOverloadCancellationAndRelease(t *testing.T) {
	capacity, err := NewLifecycleCapacity(1)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}

	firstAttempt, decision := capacity.TryBegin()
	if !decision.Admitted() {
		t.Fatalf("first decision=%+v, want reserved", decision)
	}
	first := firstAttempt.Admit()
	if first == nil || !first.Start() {
		t.Fatal("first task did not start")
	}

	if attempt, decision := capacity.TryBegin(); attempt != nil || decision.State != AdmissionOverloaded || !decision.Retryable || decision.TaskStatus != TaskStatusOverloaded {
		t.Fatalf("overload attempt=%v decision=%+v", attempt, decision)
	}

	if !first.Finish(TaskStatusCompleted) || first.Finish(TaskStatusCompleted) {
		t.Fatal("completed lease was not released exactly once")
	}

	secondAttempt, decision := capacity.TryBegin()
	if !decision.Admitted() {
		t.Fatalf("post-release decision=%+v, want reserved", decision)
	}
	second := secondAttempt.Admit()
	if second == nil || !second.Finish(TaskStatusCancelled) || second.Finish(TaskStatusCancelled) {
		t.Fatal("pre-start cancellation was not released exactly once")
	}

	snapshot := capacity.Snapshot()
	if err := snapshot.Reconcile(); err != nil {
		t.Fatalf("Reconcile: %v; counters=%+v", err, snapshot)
	}
	if snapshot.Offered != 3 || snapshot.Admitted != 2 || snapshot.Rejected != 1 || snapshot.Overloaded != 1 {
		t.Fatalf("admission counters=%+v", snapshot)
	}
	if snapshot.Started != 1 || snapshot.Completed != 1 || snapshot.Cancelled != 1 || snapshot.CancelledBeforeStart != 1 {
		t.Fatalf("terminal counters=%+v", snapshot)
	}
	if snapshot.InUse != 0 || snapshot.CapacityAcquired != snapshot.CapacityReleased || snapshot.PeakInUse != 1 {
		t.Fatalf("capacity release counters=%+v", snapshot)
	}
}

func TestLifecycleCapacityRejectedAdmissionReleasesReservation(t *testing.T) {
	capacity, err := NewLifecycleCapacity(1)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}
	attempt, decision := capacity.TryBegin()
	if !decision.Admitted() || attempt == nil {
		t.Fatalf("decision=%+v attempt=%v", decision, attempt)
	}
	if !attempt.Reject() || attempt.Reject() {
		t.Fatal("rejected attempt was not released exactly once")
	}
	snapshot := capacity.Snapshot()
	if err := snapshot.Reconcile(); err != nil {
		t.Fatalf("Reconcile: %v; counters=%+v", err, snapshot)
	}
	if snapshot.Offered != 1 || snapshot.Rejected != 1 || snapshot.InUse != 0 || snapshot.CapacityAcquired != snapshot.CapacityReleased {
		t.Fatalf("rejection counters=%+v", snapshot)
	}
}

func TestLifecycleLedgerSnapshotReconcilesInjectedFailureAndRelease(t *testing.T) {
	capacity, err := NewLifecycleCapacity(3)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}

	leases := make([]*CapacityLease, 0, 3)
	for range 3 {
		attempt, decision := capacity.TryBegin()
		if attempt == nil || !decision.Admitted() {
			t.Fatalf("admission attempt=%v decision=%+v", attempt, decision)
		}
		leases = append(leases, attempt.Admit())
	}
	if !leases[0].Start() || !leases[1].Start() {
		t.Fatal("started leases were not committed")
	}
	if attempt, decision := capacity.TryBegin(); attempt != nil || decision.State != AdmissionOverloaded {
		t.Fatalf("overload attempt=%v decision=%+v", attempt, decision)
	}

	if !leases[0].FinishResult(TaskResult{Status: TaskStatusFailed, InjectedFailure: FailureReal}) {
		t.Fatal("real failure was not committed")
	}
	if !leases[1].FinishResult(TaskResult{Status: TaskStatusGatewayUnavailable, InjectedFailure: FailureSyntheticInjected}) {
		t.Fatal("synthetic injected failure was not committed")
	}
	if leases[1].FinishResult(TaskResult{Status: TaskStatusCompleted}) {
		t.Fatal("duplicate terminal result was committed")
	}
	if !leases[2].Finish(TaskStatusCancelled) {
		t.Fatal("pre-start cancellation was not committed")
	}

	ledger := capacity.LedgerSnapshot()
	if err := ledger.Reconcile(); err != nil {
		t.Fatalf("LedgerCounters.Reconcile: %v; counters=%+v", err, ledger)
	}
	preStartTerminal, err := ledger.PreStartTerminal()
	if err != nil || preStartTerminal != 1 {
		t.Fatalf("pre-start terminal=%d error=%v, want 1", preStartTerminal, err)
	}
	if ledger.Offered != 4 || ledger.Admitted != 3 || ledger.Rejected != 0 || ledger.Overloaded != 1 {
		t.Fatalf("I1 admission counters=%+v", ledger)
	}
	if ledger.Started != 2 || ledger.Failed != 2 || ledger.FailedInjected != 1 || ledger.Cancelled != 1 {
		t.Fatalf("I1 terminal counters=%+v", ledger)
	}

	full := capacity.Snapshot()
	if err := full.Reconcile(); err != nil {
		t.Fatalf("CapacityCounters.Reconcile: %v; counters=%+v", err, full)
	}
	if full.CapacityAcquired != 3 || full.CapacityReleased != 3 || full.InUse != 0 {
		t.Fatalf("exactly-once release counters=%+v", full)
	}
}

func TestInjectedFailureMarkerRejectsCallerContentAndNonFailureStatus(t *testing.T) {
	invalidMarkers := []struct {
		name  string
		value string
	}{
		{name: "null", value: `null`},
		{name: "string", value: `"caller-content"`},
		{name: "number", value: `1`},
		{name: "object", value: `{}`},
		{name: "array", value: `[]`},
	}
	for _, test := range invalidMarkers {
		t.Run(test.name, func(t *testing.T) {
			result := TaskResult{InjectedFailure: FailureSyntheticInjected}
			data := []byte(`{"status":"failed","injected_failure":` + test.value + `}`)
			if err := json.Unmarshal(data, &result); err == nil {
				t.Fatalf("I2 accepted explicit %s instead of a boolean marker", test.name)
			}
			if result.InjectedFailure != FailureReal {
				t.Fatalf("invalid %s left I2 classified as injected", test.name)
			}
		})
	}

	var omitted TaskResult
	if err := json.Unmarshal([]byte(`{"status":"failed"}`), &omitted); err != nil {
		t.Fatalf("omitted I2 marker failed compatibility decode: %v", err)
	}
	if omitted.InjectedFailure != FailureReal {
		t.Fatalf("omitted I2 marker=%v, want real/non-injected", omitted.InjectedFailure)
	}

	for _, test := range []struct {
		name string
		json string
		want InjectedFailureMarker
	}{
		{name: "explicit false", json: `{"status":"failed","injected_failure":false}`, want: FailureReal},
		{name: "explicit true", json: `{"status":"failed","injected_failure":true}`, want: FailureSyntheticInjected},
	} {
		t.Run(test.name, func(t *testing.T) {
			var result TaskResult
			if err := json.Unmarshal([]byte(test.json), &result); err != nil {
				t.Fatalf("boolean I2 marker decode: %v", err)
			}
			if result.InjectedFailure != test.want {
				t.Fatalf("I2 marker=%v, want %v", result.InjectedFailure, test.want)
			}
		})
	}

	capacity, err := NewLifecycleCapacity(1)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}
	attempt, decision := capacity.TryBegin()
	if attempt == nil || !decision.Admitted() {
		t.Fatalf("admission attempt=%v decision=%+v", attempt, decision)
	}
	lease := attempt.Admit()
	if !lease.Start() {
		t.Fatal("lease did not start")
	}
	if lease.FinishResult(TaskResult{Status: TaskStatusCompleted, InjectedFailure: FailureSyntheticInjected}) {
		t.Fatal("injected marker was accepted for a non-failure terminal status")
	}
	if !lease.FinishResult(TaskResult{Status: TaskStatusCompleted, InjectedFailure: FailureReal}) {
		t.Fatal("valid terminal result did not commit after rejected classification")
	}
	full := capacity.Snapshot()
	if full.Completed != 1 || full.Failed != 0 || full.FailedInjected != 0 || full.CapacityReleased != 1 {
		t.Fatalf("invalid I2 marker changed terminal accounting: %+v", full)
	}
}

func TestLedgerCountersRejectsUnreconciledSnapshots(t *testing.T) {
	tests := []LedgerCounters{
		{Offered: 2, Admitted: 1},
		{Offered: 1, Admitted: 1, Started: 2},
		{Offered: 1, Admitted: 1, Completed: 1},
		{Offered: 1, Admitted: 1, Failed: 1, FailedInjected: 2},
	}
	for _, counters := range tests {
		if err := counters.Reconcile(); err == nil {
			t.Fatalf("unreconciled I1 snapshot was accepted: %+v", counters)
		}
	}
}

func TestLedgerCountersAcceptsPreStartFailureAndCancellation(t *testing.T) {
	tests := []struct {
		name     string
		counters LedgerCounters
	}{
		{
			name:     "failure",
			counters: LedgerCounters{Offered: 1, Admitted: 1, Failed: 1},
		},
		{
			name:     "cancellation",
			counters: LedgerCounters{Offered: 1, Admitted: 1, Cancelled: 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.counters.Reconcile(); err != nil {
				t.Fatalf("valid pre-start %s did not reconcile: %v", test.name, err)
			}
			preStartTerminal, err := test.counters.PreStartTerminal()
			if err != nil || preStartTerminal != 1 {
				t.Fatalf("pre-start terminal=%d error=%v, want 1", preStartTerminal, err)
			}
		})
	}
}

func TestLedgerCountersFrozenContentFreeJSONShape(t *testing.T) {
	data, err := json.Marshal(LedgerCounters{})
	if err != nil {
		t.Fatalf("marshal I1 snapshot: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal I1 snapshot shape: %v", err)
	}
	want := []string{
		"offered", "admitted", "rejected", "overloaded", "started",
		"completed", "failed", "failed_injected", "cancelled",
	}
	if len(fields) != len(want) {
		t.Fatalf("I1 fields=%d, want %d", len(fields), len(want))
	}
	for _, name := range want {
		if _, present := fields[name]; !present {
			t.Fatalf("I1 field %q is missing", name)
		}
	}
}

func TestLifecycleCapacityConcurrentClassifiedFinishReleasesOnce(t *testing.T) {
	capacity, err := NewLifecycleCapacity(1)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}
	attempt, decision := capacity.TryBegin()
	if attempt == nil || !decision.Admitted() {
		t.Fatalf("admission attempt=%v decision=%+v", attempt, decision)
	}
	lease := attempt.Admit()
	if !lease.Start() {
		t.Fatal("lease did not start")
	}

	ready := make(chan struct{})
	var workers sync.WaitGroup
	var committed atomic.Uint64
	workers.Add(64)
	for index := range 64 {
		go func(index int) {
			defer workers.Done()
			<-ready
			marker := FailureReal
			if index%2 == 0 {
				marker = FailureSyntheticInjected
			}
			if lease.FinishResult(TaskResult{Status: TaskStatusFailed, InjectedFailure: marker}) {
				committed.Add(1)
			}
		}(index)
	}
	close(ready)
	workers.Wait()

	if committed.Load() != 1 {
		t.Fatalf("terminal commits=%d, want 1", committed.Load())
	}
	ledger := capacity.LedgerSnapshot()
	if err := ledger.Reconcile(); err != nil {
		t.Fatalf("LedgerCounters.Reconcile: %v; counters=%+v", err, ledger)
	}
	if ledger.Failed != 1 || ledger.FailedInjected > 1 {
		t.Fatalf("classified failure counters=%+v", ledger)
	}
	full := capacity.Snapshot()
	if full.CapacityAcquired != 1 || full.CapacityReleased != 1 || full.InUse != 0 {
		t.Fatalf("concurrent exactly-once release counters=%+v", full)
	}
}
