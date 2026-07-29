package daemon

import (
	"context"
	"testing"
	"time"
)

// TestNewTaskSlotSemaphoreCapacityAndTokens proves the semaphore is pre-populated
// with exactly n distinct slot indices [0,n) and holds no more than n tokens.
func TestNewTaskSlotSemaphoreCapacityAndTokens(t *testing.T) {
	const n = 3
	sem := newTaskSlotSemaphore(n)
	if cap(sem) != n {
		t.Fatalf("cap = %d, want %d", cap(sem), n)
	}
	if len(sem) != n {
		t.Fatalf("initial len = %d, want %d (all slots available)", len(sem), n)
	}
	seen := map[int]bool{}
	for i := 0; i < n; i++ {
		slot := <-sem
		if slot < 0 || slot >= n || seen[slot] {
			t.Fatalf("slot %d invalid or duplicate (seen=%v)", slot, seen)
		}
		seen[slot] = true
	}
	if len(sem) != 0 {
		t.Fatalf("after draining all slots len = %d, want 0", len(sem))
	}
}

// TestWaitForTaskSlotDrainsUpToCapacityNeverExceeds proves the drain acquires up
// to capacity and, once empty, reports not-acquired (backpressure) rather than
// exceeding the limit.
func TestWaitForTaskSlotDrainsUpToCapacityNeverExceeds(t *testing.T) {
	const n = 2
	sem := newTaskSlotSemaphore(n)
	wakeup := make(chan struct{}) // never signalled here
	ctx := context.Background()

	acquired := make(map[int]bool)
	for i := 0; i < n; i++ {
		slot, ok, woke, err := waitForTaskSlot(ctx, sem, wakeup, 50*time.Millisecond)
		if !ok || woke || err != nil {
			t.Fatalf("acquire %d: ok=%v woke=%v err=%v", i, ok, woke, err)
		}
		if acquired[slot] {
			t.Fatalf("slot %d handed out twice", slot)
		}
		acquired[slot] = true
	}
	// Semaphore now empty: a bounded wait must time out as not-acquired
	// (at capacity) — the limit is never exceeded.
	slot, ok, woke, err := waitForTaskSlot(ctx, sem, wakeup, 10*time.Millisecond)
	if ok || woke || err != nil {
		t.Fatalf("at-capacity wait must not acquire: slot=%d ok=%v woke=%v err=%v", slot, ok, woke, err)
	}
	// The non-blocking (wait<=0) fast path must also report not-acquired.
	if _, ok, _, err := waitForTaskSlot(ctx, sem, wakeup, 0); ok || err != nil {
		t.Fatalf("wait<=0 on empty sem must be immediate not-acquired, got ok=%v err=%v", ok, err)
	}
}

// TestWaitForTaskSlotHonorsCancellation proves a cancelled context releases the
// wait deterministically with the context error and no slot acquired.
func TestWaitForTaskSlotHonorsCancellation(t *testing.T) {
	sem := newTaskSlotSemaphore(0) // empty: no slots ever available
	wakeup := make(chan struct{})

	// Cancelled before call -> first (non-blocking) select takes ctx.Done().
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok, _, err := waitForTaskSlot(ctx, sem, wakeup, 50*time.Millisecond); ok || err == nil {
		t.Fatalf("pre-cancelled wait must return ctx error, got ok=%v err=%v", ok, err)
	}

	// Cancelled during the blocking wait.
	ctx2, cancel2 := context.WithCancel(context.Background())
	go func() { time.Sleep(5 * time.Millisecond); cancel2() }()
	_, ok, woke, err := waitForTaskSlot(ctx2, sem, wakeup, time.Second)
	if ok || woke || err == nil {
		t.Fatalf("cancel during wait must return ctx error, got ok=%v woke=%v err=%v", ok, woke, err)
	}
}

// TestWaitForTaskSlotHonorsWakeupBackpressure proves a wakeup signal releases the
// bounded wait as woke=true (loop re-drains) without acquiring a slot.
func TestWaitForTaskSlotHonorsWakeupBackpressure(t *testing.T) {
	sem := newTaskSlotSemaphore(0) // empty: forces the blocking select
	wakeup := make(chan struct{}, 1)
	wakeup <- struct{}{} // pre-signalled

	slot, ok, woke, err := waitForTaskSlot(context.Background(), sem, wakeup, time.Second)
	if ok || !woke || err != nil {
		t.Fatalf("wakeup must return woke=true, not acquired: slot=%d ok=%v woke=%v err=%v", slot, ok, woke, err)
	}
}

// TestWaitForTaskSlotReleaseResumesDrain proves that releasing a slot back to the
// semaphore lets a subsequent wait acquire it (the drain resumes after a slot
// frees), and the limit is still respected.
func TestWaitForTaskSlotReleaseResumesDrain(t *testing.T) {
	const n = 1
	sem := newTaskSlotSemaphore(n)
	wakeup := make(chan struct{})
	ctx := context.Background()

	// Acquire the only slot.
	slot, ok, _, err := waitForTaskSlot(ctx, sem, wakeup, 50*time.Millisecond)
	if !ok || err != nil {
		t.Fatalf("first acquire: ok=%v err=%v", ok, err)
	}
	// Empty now: bounded wait times out (at capacity).
	if _, ok, _, _ := waitForTaskSlot(ctx, sem, wakeup, 10*time.Millisecond); ok {
		t.Fatal("must be at capacity before release")
	}
	// Release the slot; the next wait must acquire it again.
	sem <- slot
	slot2, ok, _, err := waitForTaskSlot(ctx, sem, wakeup, 50*time.Millisecond)
	if !ok || err != nil {
		t.Fatalf("acquire after release: ok=%v err=%v", ok, err)
	}
	if slot2 != slot {
		t.Fatalf("released slot %d, reacquired %d", slot, slot2)
	}
	if len(sem) != 0 {
		t.Fatalf("limit exceeded: len=%d, want 0", len(sem))
	}
}
