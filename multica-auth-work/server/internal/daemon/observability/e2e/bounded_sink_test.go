package e2e

import (
	"sync"
	"testing"
	"time"
)

func testSpan(reqID string) Span {
	s := NewSpan(HopIngress, Correlation{RequestID: reqID, TaskID: "task-x"})
	s.StartedAt = time.Now().UTC()
	s.Finish()
	return *s
}

func TestBoundedSinkEnforcesCapacityAndEvictsOldest(t *testing.T) {
	b := NewBoundedSink(3)
	for i := 0; i < 10; i++ {
		if err := b.Record(testSpan(string(rune('a' + i)))); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	spans := b.Spans()
	if len(spans) != 3 {
		t.Fatalf("retained=%d, want bounded 3", len(spans))
	}
	// Oldest evicted: should retain the last 3 request IDs (h, i, j) in order.
	want := []string{"h", "i", "j"}
	for i, s := range spans {
		if s.Correlation.RequestID != want[i] {
			t.Fatalf("ring order[%d]=%q want %q", i, s.Correlation.RequestID, want[i])
		}
	}
	total, retained, dropped := b.Stats()
	if total != 10 || retained != 3 || dropped != 7 {
		t.Fatalf("stats total=%d retained=%d dropped=%d", total, retained, dropped)
	}
}

func TestBoundedSinkClampsNonPositiveCapacity(t *testing.T) {
	b := NewBoundedSink(0)
	_ = b.Record(testSpan("a"))
	_ = b.Record(testSpan("b"))
	if got := len(b.Spans()); got != 1 {
		t.Fatalf("clamped capacity retained=%d, want 1", got)
	}
}

func TestBoundedSinkConcurrentSafe(t *testing.T) {
	b := NewBoundedSink(50)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = b.Record(testSpan("r"))
			}
		}(i)
	}
	wg.Wait()
	total, retained, _ := b.Stats()
	if total != 1000 {
		t.Fatalf("total=%d want 1000", total)
	}
	if retained != 50 {
		t.Fatalf("retained=%d want bounded 50", retained)
	}
}
