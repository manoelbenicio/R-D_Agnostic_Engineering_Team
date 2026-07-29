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
	snapshot := cloneSpan(*s)
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("span refused (invalid): %w", err)
	}
	// The sink receives its own copy. A sink that retains or mutates reference
	// fields cannot alias the caller's Span or the validated snapshot.
	return r.sink.Record(cloneSpan(snapshot))
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
	m.spans = append(m.spans, cloneSpan(s))
	return nil
}

// Spans returns a deep copy of the recorded spans.
func (m *MemorySink) Spans() []Span {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Span, len(m.spans))
	for i, span := range m.spans {
		out[i] = cloneSpan(span)
	}
	return out
}

// Len returns the number of recorded spans.
func (m *MemorySink) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.spans)
}

func cloneSpan(source Span) Span {
	cloned := source
	if source.Labels != nil {
		cloned.Labels = make(map[string]string, len(source.Labels))
		for key, value := range source.Labels {
			cloned.Labels[key] = value
		}
	}
	if source.Counters != nil {
		cloned.Counters = make(map[string]int64, len(source.Counters))
		for key, value := range source.Counters {
			cloned.Counters[key] = value
		}
	}
	if source.ArgvShape != nil {
		cloned.ArgvShape = append([]string(nil), source.ArgvShape...)
	}
	return cloned
}
