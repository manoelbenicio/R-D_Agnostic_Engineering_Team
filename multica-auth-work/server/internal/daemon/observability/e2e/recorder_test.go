package e2e

import (
	"strings"
	"testing"
)

type mutatingSink struct{}

func (mutatingSink) Record(span Span) error {
	span.Labels["method"] = "MUTATED"
	span.Counters["latency_ms"] = 999
	span.ArgvShape[0] = "flag"
	return nil
}

func TestRecorderSeparatesCallerFromSinkAliases(t *testing.T) {
	span := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithLabel("method", "POST").
		WithCounter("latency_ms", 1).
		WithArgvShape([]string{"subcommand"}).
		WithOutcome("accepted", "").
		Finish()

	if err := NewRecorder(mutatingSink{}).Emit(span); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if span.Labels["method"] != "POST" || span.Counters["latency_ms"] != 1 || span.ArgvShape[0] != "subcommand" {
		t.Fatal("sink mutation aliased the caller span")
	}
}

func TestMemorySinkDeepClonesAtRecordAndReadBoundaries(t *testing.T) {
	sink := NewMemorySink()
	source := Span{
		Labels:    map[string]string{"method": "POST"},
		Counters:  map[string]int64{"latency_ms": 1},
		ArgvShape: []string{"subcommand"},
	}
	if err := sink.Record(source); err != nil {
		t.Fatalf("record: %v", err)
	}

	source.Labels["method"] = "PATCH"
	source.Counters["latency_ms"] = 2
	source.ArgvShape[0] = "flag"

	first := sink.Spans()
	if first[0].Labels["method"] != "POST" || first[0].Counters["latency_ms"] != 1 || first[0].ArgvShape[0] != "subcommand" {
		t.Fatal("recorded span aliases source reference fields")
	}

	first[0].Labels["method"] = "DELETE"
	first[0].Counters["latency_ms"] = 3
	first[0].ArgvShape[0] = "arg=<redacted>"

	second := sink.Spans()
	if second[0].Labels["method"] != "POST" || second[0].Counters["latency_ms"] != 1 || second[0].ArgvShape[0] != "subcommand" {
		t.Fatal("returned span aliases MemorySink storage")
	}
}

func TestRecorderRefusalDoesNotEchoUntrustedMetadata(t *testing.T) {
	const untrusted = "untrusted-payload-marker"
	span := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithLabel(untrusted, "value").
		WithOutcome("accepted", "").
		Finish()
	err := NewRecorder(nil).Emit(span)
	if err == nil {
		t.Fatal("unapproved metadata accepted")
	}
	if strings.Contains(err.Error(), untrusted) {
		t.Fatal("refusal error echoed untrusted metadata")
	}
}
