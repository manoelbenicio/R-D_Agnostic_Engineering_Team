package brain

import (
	"fmt"
	"sync"
)

// CapacityCounters is the content-free lifecycle ledger owned by the cold
// plane. It deliberately counts task state only; route/account concurrency
// remains the gateway's responsibility.
type CapacityCounters struct {
	Offered              uint64 `json:"offered"`
	Admitted             uint64 `json:"admitted"`
	Rejected             uint64 `json:"rejected"`
	Overloaded           uint64 `json:"overloaded"`
	Started              uint64 `json:"started"`
	Completed            uint64 `json:"completed"`
	Failed               uint64 `json:"failed"`
	FailedInjected       uint64 `json:"failed_injected"`
	Cancelled            uint64 `json:"cancelled"`
	CancelledBeforeStart uint64 `json:"cancelled_before_start"`
	CancelledAfterStart  uint64 `json:"cancelled_after_start"`
	FailedBeforeStart    uint64 `json:"failed_before_start"`
	FailedAfterStart     uint64 `json:"failed_after_start"`
	PendingAdmission     uint64 `json:"pending_admission"`
	PendingStart         uint64 `json:"pending_start"`
	Active               uint64 `json:"active"`
	InUse                uint64 `json:"in_use"`
	PeakInUse            uint64 `json:"peak_in_use"`
	CapacityAcquired     uint64 `json:"capacity_acquired"`
	CapacityReleased     uint64 `json:"capacity_released"`
}

// LedgerCounters is the frozen, content-free I1 lifecycle snapshot consumed by
// offline measurability. Rejected excludes deterministic overload so every
// offered task has exactly one mutually exclusive admission disposition.
// Identifiers, classifications, error text, and task content are intentionally
// absent.
type LedgerCounters struct {
	Offered        uint64 `json:"offered"`
	Admitted       uint64 `json:"admitted"`
	Rejected       uint64 `json:"rejected"`
	Overloaded     uint64 `json:"overloaded"`
	Started        uint64 `json:"started"`
	Completed      uint64 `json:"completed"`
	Failed         uint64 `json:"failed"`
	FailedInjected uint64 `json:"failed_injected"`
	Cancelled      uint64 `json:"cancelled"`
}

// PreStartTerminal returns the terminal-before-start count implied by a
// quiescent I1 snapshot. Reconcile must succeed before consumers use it.
func (c LedgerCounters) PreStartTerminal() (uint64, error) {
	if c.Started > c.Admitted {
		return 0, fmt.Errorf("started=%d exceeds admitted=%d", c.Started, c.Admitted)
	}
	return c.Admitted - c.Started, nil
}

// Reconcile validates a quiescent I1 snapshot. In-flight callers use the full
// CapacityCounters.Reconcile contract, which includes pending and active state.
func (c LedgerCounters) Reconcile() error {
	if c.Offered != c.Admitted+c.Rejected+c.Overloaded {
		return fmt.Errorf("offered=%d does not equal admitted=%d + rejected=%d + overloaded=%d", c.Offered, c.Admitted, c.Rejected, c.Overloaded)
	}
	preStartTerminal, err := c.PreStartTerminal()
	if err != nil {
		return err
	}
	if c.Completed > c.Started {
		return fmt.Errorf("completed=%d exceeds started=%d", c.Completed, c.Started)
	}
	if preStartTerminal > c.Failed+c.Cancelled {
		return fmt.Errorf("pre_start_terminal=%d exceeds failed=%d + cancelled=%d", preStartTerminal, c.Failed, c.Cancelled)
	}
	if c.Admitted != c.Started+preStartTerminal {
		return fmt.Errorf("admitted lifecycle does not reconcile")
	}
	if c.Admitted != c.Completed+c.Failed+c.Cancelled {
		return fmt.Errorf("admitted=%d does not equal completed=%d + failed=%d + cancelled=%d", c.Admitted, c.Completed, c.Failed, c.Cancelled)
	}
	if c.FailedInjected > c.Failed {
		return fmt.Errorf("failed_injected=%d exceeds failed=%d", c.FailedInjected, c.Failed)
	}
	return nil
}

// Reconcile verifies the exact lifecycle and capacity equations after any
// point-in-time snapshot. PendingStart, Active, and InUse make the equations
// valid during a run as well as after recovery.
func (c CapacityCounters) Reconcile() error {
	if c.Offered != c.Admitted+c.Rejected+c.PendingAdmission {
		return fmt.Errorf("offered=%d does not equal admitted=%d + rejected=%d + pending_admission=%d", c.Offered, c.Admitted, c.Rejected, c.PendingAdmission)
	}
	if c.Overloaded > c.Rejected {
		return fmt.Errorf("overloaded=%d exceeds rejected=%d", c.Overloaded, c.Rejected)
	}
	if c.Admitted != c.Started+c.CancelledBeforeStart+c.FailedBeforeStart+c.PendingStart {
		return fmt.Errorf("admitted lifecycle does not reconcile")
	}
	if c.Started != c.Completed+c.FailedAfterStart+c.CancelledAfterStart+c.Active {
		return fmt.Errorf("started lifecycle does not reconcile")
	}
	if c.Failed != c.FailedBeforeStart+c.FailedAfterStart {
		return fmt.Errorf("failed lifecycle does not reconcile")
	}
	if c.FailedInjected > c.Failed {
		return fmt.Errorf("failed_injected=%d exceeds failed=%d", c.FailedInjected, c.Failed)
	}
	if c.Cancelled != c.CancelledBeforeStart+c.CancelledAfterStart {
		return fmt.Errorf("cancelled lifecycle does not reconcile")
	}
	if c.InUse != c.PendingAdmission+c.PendingStart+c.Active {
		return fmt.Errorf("in_use=%d does not equal pending_admission=%d + pending_start=%d + active=%d", c.InUse, c.PendingAdmission, c.PendingStart, c.Active)
	}
	if c.CapacityAcquired != c.CapacityReleased+c.InUse {
		return fmt.Errorf("capacity acquisition/release does not reconcile")
	}
	return nil
}

// LifecycleCapacity provides bounded, non-queuing task admission. A caller
// reserves capacity before any external readiness or credential callback,
// then either rejects the attempt or commits it as admitted. Each admitted
// lease releases capacity exactly once on its terminal result.
type LifecycleCapacity struct {
	mu       sync.Mutex
	limit    uint64
	counters CapacityCounters
}

func NewLifecycleCapacity(limit int) (*LifecycleCapacity, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("lifecycle capacity limit must be positive")
	}
	return &LifecycleCapacity{limit: uint64(limit)}, nil
}

func (c *LifecycleCapacity) Limit() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return int(c.limit)
}

// TryBegin records an offered task and reserves capacity without queueing.
// Overload is deterministic, retryable, and consumes no credential callback
// or execution slot beyond this in-memory gate.
func (c *LifecycleCapacity) TryBegin() (*CapacityAttempt, AdmissionDecision) {
	if c == nil {
		return nil, overloadDecision("capacity_gate_unavailable")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters.Offered++
	if c.counters.InUse >= c.limit {
		c.counters.Rejected++
		c.counters.Overloaded++
		return nil, overloadDecision("local_capacity_overloaded")
	}
	c.counters.InUse++
	c.counters.PendingAdmission++
	c.counters.CapacityAcquired++
	if c.counters.InUse > c.counters.PeakInUse {
		c.counters.PeakInUse = c.counters.InUse
	}
	return &CapacityAttempt{capacity: c}, AdmissionDecision{State: AdmissionAdmitted}
}

func (c *LifecycleCapacity) Snapshot() CapacityCounters {
	if c == nil {
		return CapacityCounters{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counters
}

// LedgerSnapshot returns the read-only content-free I1 view. The existing raw
// Rejected counter includes overload for compatibility; I1 exposes the two
// mutually exclusive dispositions separately.
func (c *LifecycleCapacity) LedgerSnapshot() LedgerCounters {
	if c == nil {
		return LedgerCounters{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	rejected := c.counters.Rejected
	if c.counters.Overloaded <= rejected {
		rejected -= c.counters.Overloaded
	} else {
		rejected = 0
	}
	return LedgerCounters{
		Offered: c.counters.Offered, Admitted: c.counters.Admitted,
		Rejected: rejected, Overloaded: c.counters.Overloaded,
		Started: c.counters.Started, Completed: c.counters.Completed,
		Failed: c.counters.Failed, FailedInjected: c.counters.FailedInjected,
		Cancelled: c.counters.Cancelled,
	}
}

type CapacityAttempt struct {
	capacity *LifecycleCapacity
	closed   bool
}

// Reject closes a reserved attempt before admission and releases its capacity.
// Repeated calls are harmless and do not change counters.
func (a *CapacityAttempt) Reject() bool {
	if a == nil || a.capacity == nil {
		return false
	}
	a.capacity.mu.Lock()
	defer a.capacity.mu.Unlock()
	if a.closed {
		return false
	}
	a.closed = true
	a.capacity.counters.PendingAdmission--
	a.capacity.counters.Rejected++
	a.capacity.releaseLocked()
	return true
}

// Admit commits a reserved attempt and returns its exactly-once lifecycle
// lease. It returns nil when the attempt was already closed.
func (a *CapacityAttempt) Admit() *CapacityLease {
	if a == nil || a.capacity == nil {
		return nil
	}
	a.capacity.mu.Lock()
	defer a.capacity.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	a.capacity.counters.PendingAdmission--
	a.capacity.counters.Admitted++
	a.capacity.counters.PendingStart++
	return &CapacityLease{capacity: a.capacity}
}

type CapacityLease struct {
	capacity *LifecycleCapacity
	started  bool
	terminal bool
}

// Start moves one admitted task from pending to active exactly once.
func (l *CapacityLease) Start() bool {
	if l == nil || l.capacity == nil {
		return false
	}
	l.capacity.mu.Lock()
	defer l.capacity.mu.Unlock()
	if l.started || l.terminal {
		return false
	}
	l.started = true
	l.capacity.counters.PendingStart--
	l.capacity.counters.Started++
	l.capacity.counters.Active++
	return true
}

// Finish records one terminal outcome and releases capacity exactly once.
// Any non-completed/non-cancelled status is a failure for reconciliation.
func (l *CapacityLease) Finish(status TaskStatus) bool {
	return l.finish(status, FailureReal)
}

// FinishResult applies the neutral I2 failure classification to terminal
// accounting. Invalid injected classifications fail closed without consuming
// the lease, so the caller may still record the correct terminal result.
func (l *CapacityLease) FinishResult(result TaskResult) bool {
	if err := result.InjectedFailure.Validate(result.Status); err != nil {
		return false
	}
	return l.finish(result.Status, result.InjectedFailure)
}

func (l *CapacityLease) finish(status TaskStatus, injected InjectedFailureMarker) bool {
	if l == nil || l.capacity == nil {
		return false
	}
	l.capacity.mu.Lock()
	defer l.capacity.mu.Unlock()
	if l.terminal {
		return false
	}
	l.terminal = true
	if l.started {
		l.capacity.counters.Active--
		switch status {
		case TaskStatusCompleted:
			l.capacity.counters.Completed++
		case TaskStatusCancelled:
			l.capacity.counters.Cancelled++
			l.capacity.counters.CancelledAfterStart++
		default:
			l.capacity.counters.Failed++
			if bool(injected) {
				l.capacity.counters.FailedInjected++
			}
			l.capacity.counters.FailedAfterStart++
		}
	} else {
		l.capacity.counters.PendingStart--
		if status == TaskStatusCancelled {
			l.capacity.counters.Cancelled++
			l.capacity.counters.CancelledBeforeStart++
		} else {
			l.capacity.counters.Failed++
			if bool(injected) {
				l.capacity.counters.FailedInjected++
			}
			l.capacity.counters.FailedBeforeStart++
		}
	}
	l.capacity.releaseLocked()
	return true
}

func (c *LifecycleCapacity) releaseLocked() {
	if c.counters.InUse > 0 {
		c.counters.InUse--
	}
	c.counters.CapacityReleased++
}

func overloadDecision(class string) AdmissionDecision {
	return AdmissionDecision{
		State:      AdmissionOverloaded,
		TaskStatus: TaskStatusOverloaded,
		Retryable:  true,
		ErrorClass: class,
	}
}
