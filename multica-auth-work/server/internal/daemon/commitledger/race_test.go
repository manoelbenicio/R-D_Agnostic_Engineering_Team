package commitledger

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestRace_ConcurrentRecordAndToolResult exercises the primary race path:
// many goroutines recording tool_use while another commits tool_results.
// Run with: go test -race -run TestRace_ConcurrentRecordAndToolResult
func TestRace_ConcurrentRecordAndToolResult(t *testing.T) {
	l := mustNew(t, "task-race-1")

	const writers = 5
	const entriesPerWriter = 50
	var wg sync.WaitGroup

	// Use a single writer for seq ordering (strict increasing)
	tokens := make([]string, writers*entriesPerWriter)
	for i := 0; i < writers*entriesPerWriter; i++ {
		tok, err := l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i+1))
		if err != nil {
			t.Fatalf("record %d failed: %v", i, err)
		}
		tokens[i] = tok
	}

	// Concurrent tool_result commits
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			start := writerID * entriesPerWriter
			for i := start; i < start+entriesPerWriter; i++ {
				_ = l.RecordToolResult(tokens[i])
			}
		}(w)
	}

	// Concurrent output ack
	wg.Add(1)
	go func() {
		defer wg.Done()
		for seq := int64(1); seq <= int64(writers*entriesPerWriter); seq += 5 {
			l.AcknowledgeOutputPersisted(seq)
			time.Sleep(time.Microsecond)
		}
	}()

	// Concurrent lookups
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			for _, tok := range tokens {
				_ = l.Lookup(tok)
			}
		}
	}()

	wg.Wait()
}

// TestRace_ConcurrentMarkAmbiguous exercises MarkAmbiguous under contention.
func TestRace_ConcurrentMarkAmbiguous(t *testing.T) {
	l := mustNew(t, "task-race-2")

	// Pre-populate
	tokens := make([]string, 50)
	for i := 0; i < 50; i++ {
		tok, _ := l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i+1))
		tokens[i] = tok
	}

	var wg sync.WaitGroup

	// Half get committed
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 25; i++ {
			_ = l.RecordToolResult(tokens[i])
		}
	}()

	// Half get marked ambiguous
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 25; i < 50; i++ {
			_ = l.MarkAmbiguous(tokens[i])
		}
	}()

	// Concurrent stats/summary
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = l.Stats()
			_ = l.Summary()
		}
	}()

	wg.Wait()
}

// TestRace_ConcurrentRegistryAccess exercises registry under contention.
func TestRace_ConcurrentRegistryAccess(t *testing.T) {
	reg := NewLedgerRegistry()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			id := fmt.Sprintf("corr-%d", i%20)
			l := mustNew(t, fmt.Sprintf("task-%d", i%20))
			reg.Register(id, l)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			id := fmt.Sprintf("corr-%d", i%20)
			_ = reg.Get(id)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			id := fmt.Sprintf("corr-%d", i%20)
			reg.Unregister(id)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			id := fmt.Sprintf("corr-%d", i%20)
			_ = reg.CheckReplayAllowed(id)
		}
	}()

	wg.Wait()
}

// TestRace_DrainOwnerConcurrentStart exercises multiple Start attempts.
func TestRace_DrainOwnerConcurrentStart(t *testing.T) {
	l := mustNew(t, "task-race-3")
	for i := 1; i <= 10; i++ {
		l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i))
	}

	owner := NewDrainOwner(l, 100*time.Millisecond, nil)

	var wg sync.WaitGroup
	started := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if owner.Start() {
				started <- struct{}{}
			}
		}()
	}

	wg.Wait()
	close(started)

	count := 0
	for range started {
		count++
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 successful Start, got %d", count)
	}

	// Finish without panic
	owner.Finish()
	owner.Finish() // idempotent
}

// TestRace_SnapshotDuringMutation exercises snapshot under concurrent mutations.
func TestRace_SnapshotDuringMutation(t *testing.T) {
	l := mustNew(t, "task-race-4")

	var wg sync.WaitGroup

	// Sequential writer (must maintain strict seq)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i))
		}
	}()

	// Acker
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			l.AcknowledgeOutputPersisted(int64(i))
			time.Sleep(time.Microsecond)
		}
	}()

	// Snapshot reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			snap := l.Snapshot()
			if snap.Version != SchemaVersion {
				t.Errorf("snapshot version: %d", snap.Version)
			}
		}
	}()

	wg.Wait()
}

// TestRace_MarkAllUnresolvedDuringCommit exercises MarkAllUnresolvedAmbiguous
// concurrent with RecordToolResult.
func TestRace_MarkAllUnresolvedDuringCommit(t *testing.T) {
	l := mustNew(t, "task-race-5")
	tokens := make([]string, 50)
	for i := 0; i < 50; i++ {
		tok, _ := l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i+1))
		tokens[i] = tok
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tok := range tokens[:25] {
			_ = l.RecordToolResult(tok)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.MarkAllUnresolvedAmbiguous()
	}()

	wg.Wait()

	// All entries should be in a terminal state
	for _, tok := range tokens {
		state := l.Lookup(tok)
		if state == CommitNone {
			t.Errorf("token %s should not be None after concurrent commit+markAll", tok[:8])
		}
	}
}

// TestRace_JoinOrTimeoutConcurrent exercises concurrent join attempts.
func TestRace_JoinOrTimeoutConcurrent(t *testing.T) {
	l := mustNew(t, "task-race-6")
	l.RecordToolUse("call-1", 1)

	owner := NewDrainOwner(l, 200*time.Millisecond, nil)
	owner.Start()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = owner.JoinOrTimeout(context.Background())
		}()
	}

	// Let them race for a bit, then finish
	time.Sleep(50 * time.Millisecond)
	owner.Finish()
	wg.Wait()
}
