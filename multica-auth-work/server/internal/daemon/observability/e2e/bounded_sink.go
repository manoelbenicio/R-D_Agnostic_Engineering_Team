package e2e

import "sync"

// BoundedSink is a production operational Sink with a fixed-capacity ring
// buffer. It retains only the most recent N validated, metadata-only spans, so
// memory is bounded regardless of throughput (no unbounded growth, no busy
// loop). It is safe for concurrent use by all hop owners.
//
// Spans handed to Record have already passed Recorder.Emit validation and the
// single-span structural leak scan, so BoundedSink never re-adds content and
// stores each span by value (deep-copied on the way in and out).
type BoundedSink struct {
	mu       sync.Mutex
	capacity int
	buf      []Span
	next     int
	full     bool
	total    uint64
	dropped  uint64
}

// NewBoundedSink returns a ring-buffer sink retaining up to capacity spans.
// A non-positive capacity is clamped to 1 so the sink can never be unbounded
// or zero-length.
func NewBoundedSink(capacity int) *BoundedSink {
	if capacity <= 0 {
		capacity = 1
	}
	return &BoundedSink{capacity: capacity, buf: make([]Span, 0, capacity)}
}

// Record stores a deep copy of the span, evicting the oldest when at capacity.
func (b *BoundedSink) Record(s Span) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total++
	span := cloneSpan(s)
	if len(b.buf) < b.capacity {
		b.buf = append(b.buf, span)
		return nil
	}
	// Ring is full: overwrite oldest slot.
	b.full = true
	b.dropped++
	b.buf[b.next] = span
	b.next = (b.next + 1) % b.capacity
	return nil
}

// Spans returns a deep copy of the retained spans in insertion order.
func (b *BoundedSink) Spans() []Span {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Span, 0, len(b.buf))
	if !b.full {
		for _, s := range b.buf {
			out = append(out, cloneSpan(s))
		}
		return out
	}
	for i := 0; i < b.capacity; i++ {
		out = append(out, cloneSpan(b.buf[(b.next+i)%b.capacity]))
	}
	return out
}

// Stats reports total spans recorded and how many were evicted (dropped) due to
// the bound, for operational visibility. Metadata-only, no span contents.
func (b *BoundedSink) Stats() (total, retained, dropped uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total, uint64(len(b.buf)), b.dropped
}
