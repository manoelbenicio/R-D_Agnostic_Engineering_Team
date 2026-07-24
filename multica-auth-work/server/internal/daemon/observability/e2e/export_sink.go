package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// JSONLSink is the durable cross-process export path. It serializes each
// validated, metadata-only Span as one JSON object per line (JSONL) to an
// io.Writer, writing the FULL closed schema — contract version, hop, all nine
// correlation IDs, ORIGINAL StartedAt/EndedAt, outcome/reason, http status,
// labels, counters, argv shape, secrets_present — so a collector reconstructs
// the exact emitted span (never synthesizing timestamps) and can rerun the full
// structural leak scan.
//
// It is fail-closed: a nil/unwritable writer makes Record return an error
// (never a silent success), and short writes are detected and surfaced. Lines
// are written atomically under a mutex so concurrent emitters never interleave.
type JSONLSink struct {
	mu sync.Mutex
	w  io.Writer
}

// NewJSONLSink builds an export sink over w. w must be non-nil; an enabled
// exporter with no writer fails closed on every Record.
func NewJSONLSink(w io.Writer) *JSONLSink {
	return &JSONLSink{w: w}
}

// Record serializes span as one JSONL line and writes it atomically, returning
// any marshal/write/short-write error so export failures are visible.
func (s *JSONLSink) Record(span Span) error {
	if s == nil || s.w == nil {
		return fmt.Errorf("e2e export: unwritable sink (fail closed)")
	}
	line, err := json.Marshal(span)
	if err != nil {
		return fmt.Errorf("e2e export: marshal: %w", err)
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.w.Write(line)
	if err != nil {
		return fmt.Errorf("e2e export: write: %w", err)
	}
	if n != len(line) {
		return fmt.Errorf("e2e export: short write %d/%d", n, len(line))
	}
	return nil
}

// ParseSpanLine reconstructs a Span from one JSONL line and fully validates it
// (schema + contract version + structural leak scan), so a collector accepts
// only spans identical in shape to what was emitted. It never fills defaults.
func ParseSpanLine(line []byte) (Span, error) {
	line = bytes.TrimSpace(line)
	var span Span
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&span); err != nil {
		return Span{}, fmt.Errorf("e2e collect: decode: %w", err)
	}
	if !SupportedContractVersion(span.ContractVersion) {
		return Span{}, fmt.Errorf("e2e collect: unsupported contract version %q", span.ContractVersion)
	}
	if err := span.Validate(); err != nil {
		return Span{}, fmt.Errorf("e2e collect: invalid span: %w", err)
	}
	return span, nil
}

// MultiSink fans a validated span out to several sinks (e.g. an in-memory
// BoundedSink for operational snapshots plus a JSONLSink for durable export).
// It attempts every sink and returns the first error, so a failing exporter is
// always visible and never silently swallowed.
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink builds a fan-out sink. Nil members are ignored.
func NewMultiSink(sinks ...Sink) *MultiSink {
	out := make([]Sink, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			out = append(out, s)
		}
	}
	return &MultiSink{sinks: out}
}

// Record forwards to every configured sink, returning the first error while
// still attempting the rest.
func (m *MultiSink) Record(span Span) error {
	var firstErr error
	for _, s := range m.sinks {
		if err := s.Record(span); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
