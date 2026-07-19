package e2e

import (
	"fmt"
	"sync"
)

// Sink receives validated, leak-free spans. Implementations must treat spans as
// already metadata-only; they must never re-add content.
type Sink interface {
	Record(Span) error
}

// Recorder is the clean API that every hop owner (W1/W2/W3/W4/W6/W7) uses to
// emit spans. Emit validates the span and runs a single-span structural leak
// scan BEFORE handing it to the sink. If either check fails, the span is
// refused and never recorded (fail-closed).
type Recorder struct {
	sink Sink
}

// NewRecorder builds a Recorder over the given sink. A nil sink uses a no-op
// sink so that instrumentation can never crash a caller.
func NewRecorder(sink Sink) *Recorder {
	if sink == nil {
		sink = discardSink{}
	}
	return &Recorder{sink: sink}
}

// Emit validates the span (a full structural, fail-closed check that enforces
// the metadata-only / secrets_present invariant) and then records it. On any
// validation error the span is dropped and the error is returned to the caller
// for classification (never printed with values). The OBS-10 batch scanner
// (ScanSpans) provides the auditable report form over recorded spans.
func (r *Recorder) Emit(s *Span) error {
	if s == nil {
		return fmt.Errorf("nil span")
	}
	if err := s.Validate(); err != nil {
		return fmt.Errorf("span refused (invalid): %w", err)
	}
	return r.sink.Record(*s)
}

type discardSink struct{}

func (discardSink) Record(Span) error { return nil }

// MemorySink is an in-memory Sink for tests, the trace assembler, and the
// synthetic acceptance harness. It is safe for concurrent use.
type MemorySink struct {
	mu    sync.Mutex
	spans []Span
}

// NewMemorySink returns an empty in-memory sink.
func NewMemorySink() *MemorySink { return &MemorySink{} }

// Record stores a copy of the span.
func (m *MemorySink) Record(s Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans = append(m.spans, s)
	return nil
}

// Spans returns a copy of the recorded spans.
func (m *MemorySink) Spans() []Span {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Span, len(m.spans))
	copy(out, m.spans)
	return out
}

// Len returns the number of recorded spans.
func (m *MemorySink) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.spans)
}
