package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func validExportSpan(reqID string) Span {
	s := NewSpan(HopIngress, Correlation{RequestID: reqID, TaskID: "task-" + reqID})
	s.StartedAt = time.Now().UTC().Add(-30 * time.Millisecond)
	s.WithOutcome("received", "")
	s.Finish()
	return *s
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestJSONLRoundTripExact(t *testing.T) {
	span := validExportSpan("r1")
	var buf bytes.Buffer
	if err := NewJSONLSink(&buf).Record(span); err != nil {
		t.Fatalf("record: %v", err)
	}
	parsed, err := ParseSpanLine(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Exact round-trip: re-marshaling the parsed span reproduces the line, and
	// original timestamps are preserved (never synthesized).
	orig, _ := json.Marshal(span)
	round, _ := json.Marshal(parsed)
	if !bytes.Equal(orig, round) {
		t.Fatalf("round-trip mismatch:\n orig=%s\nround=%s", orig, round)
	}
	if !parsed.StartedAt.Equal(span.StartedAt) || !parsed.EndedAt.Equal(span.EndedAt) {
		t.Fatalf("timestamps not preserved")
	}
}

func TestJSONLNilWriterFailsClosed(t *testing.T) {
	if err := NewJSONLSink(nil).Record(validExportSpan("r1")); err == nil {
		t.Fatal("nil writer must fail closed, got nil error")
	}
}

func TestJSONLWriteErrorPropagates(t *testing.T) {
	if err := NewJSONLSink(errWriter{}).Record(validExportSpan("r1")); err == nil {
		t.Fatal("write error must propagate")
	}
}

func TestJSONLShortWritePropagates(t *testing.T) {
	if err := NewJSONLSink(shortWriter{}).Record(validExportSpan("r1")); err == nil {
		t.Fatal("short write must propagate")
	}
}

func TestParseSpanLineRejectsUnsupportedContract(t *testing.T) {
	span := validExportSpan("r1")
	span.ContractVersion = "agent-brain.e2e.v999"
	line, _ := json.Marshal(span)
	if _, err := ParseSpanLine(line); err == nil {
		t.Fatal("unsupported contract version must be rejected")
	}
}

func TestParseSpanLineRejectsMalformed(t *testing.T) {
	if _, err := ParseSpanLine([]byte("{not json")); err == nil {
		t.Fatal("malformed line must be rejected")
	}
	if _, err := ParseSpanLine([]byte(`{"contract_version":"agent-brain.e2e.v1","surprise":true}`)); err == nil {
		t.Fatal("unknown field must be rejected (DisallowUnknownFields)")
	}
}

func TestJSONLConcurrentAtomicLines(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONLSink(&buf)
	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			if err := sink.Record(validExportSpan("c")); err != nil {
				t.Errorf("record: %v", err)
			}
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != n {
		t.Fatalf("expected %d atomic lines, got %d", n, len(lines))
	}
	var parsed []Span
	for i, ln := range lines {
		s, err := ParseSpanLine([]byte(ln))
		if err != nil {
			t.Fatalf("line %d not a clean whole span (interleave?): %v", i, err)
		}
		parsed = append(parsed, s)
	}
	if rep := ScanSpans(parsed); !rep.Clean {
		t.Fatalf("no-secret structural scan not clean: %d findings", len(rep.Findings))
	}
}

func TestJSONLExportedSpansScanClean(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONLSink(&buf)
	for _, r := range []string{"a", "b", "c"} {
		if err := sink.Record(validExportSpan(r)); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	var parsed []Span
	for _, ln := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		s, err := ParseSpanLine([]byte(ln))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		parsed = append(parsed, s)
	}
	if rep := ScanSpans(parsed); !rep.Clean {
		t.Fatalf("structural scan not clean: %+v", rep.Findings)
	}
}
