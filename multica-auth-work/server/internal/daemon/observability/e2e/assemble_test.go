package e2e

import (
	"strings"
	"testing"
)

func TestAssembleContinuousSyntheticTasks(t *testing.T) {
	sink := NewMemorySink()
	rec := NewRecorder(sink)
	for _, id := range []string{"t1", "t2", "t3"} {
		if err := EmitSyntheticTask(rec, id); err != nil {
			t.Fatalf("emit synthetic %s: %v", id, err)
		}
	}
	report := AssembleFromSink(sink)
	if !report.AllContinuous {
		t.Fatalf("expected all continuous, got %+v", report)
	}
	if len(report.Traces) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(report.Traces))
	}
	if len(report.Orphans) != 0 {
		t.Fatalf("expected 0 orphans, got %+v", report.Orphans)
	}
	for _, tr := range report.Traces {
		if !tr.Continuous || len(tr.Present) != 7 || len(tr.Missing) != 0 {
			t.Fatalf("trace %s not continuous: present=%v missing=%v", tr.TaskID, tr.Present, tr.Missing)
		}
	}
}

func TestAssembleDetectsGap(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	// Drop the CLI hop (index 3) to create a gap.
	gapped := append([]Span{}, spans[:3]...)
	gapped = append(gapped, spans[4:]...)

	report := Assemble(gapped)
	if report.AllContinuous {
		t.Fatalf("expected gap to break continuity")
	}
	if len(report.Traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(report.Traces))
	}
	tr := report.Traces[0]
	if tr.Continuous {
		t.Fatalf("expected trace to be non-continuous")
	}
	foundCLI := false
	for _, m := range tr.Missing {
		if m == HopCLI {
			foundCLI = true
		}
	}
	if !foundCLI {
		t.Fatalf("expected HopCLI reported missing, got %v", tr.Missing)
	}
}

func TestAssembleDetectsOrphan(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	// Add a route span whose request_id matches no ingress task.
	orphan := *NewSpan(HopRoute, Correlation{RequestID: "req-unknown", OmniRequestID: "omni-x"}).
		WithLabel("route_model", "model-a").WithOutcome("ok", "ok").Finish()
	report := Assemble(append(spans, orphan))
	if report.AllContinuous {
		t.Fatalf("expected orphan to break continuity")
	}
	if len(report.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d (%+v)", len(report.Orphans), report.Orphans)
	}
	if report.Orphans[0].Hop != HopRoute {
		t.Fatalf("expected route orphan, got %q", report.Orphans[0].Hop)
	}
}

func TestAssembleInvalidSpanIsOrphan(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	bad := spans[0]
	bad.SecretsPresent = true // invalidate
	replaced := append([]Span{bad}, spans[1:]...)
	report := Assemble(replaced)
	if report.AllContinuous {
		t.Fatalf("expected invalid span to break continuity")
	}
	if len(report.Orphans) == 0 {
		t.Fatalf("expected invalid span reported as orphan")
	}
}

func TestAssembleEmptyIsNotContinuous(t *testing.T) {
	report := Assemble(nil)
	if report.AllContinuous {
		t.Fatalf("empty assembly must not be continuous")
	}
}

func TestAssembleRejectsDuplicateDirectHop(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	duplicate := cloneSpan(spans[1])
	duplicate.Outcome = "duplicate"

	report := Assemble(append(spans, duplicate))
	assertAssemblyAnomaly(t, report, AnomalyDuplicateSpan, HopQueue)
	if report.AllContinuous {
		t.Fatal("duplicate direct hop must break continuity")
	}
	if got := report.Traces[0].Hops[HopQueue].Outcome; got != spans[1].Outcome {
		t.Fatalf("duplicate overwrote first direct hop: got %q", got)
	}
}

func TestAssembleRejectsDuplicateViaJoinHop(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	duplicate := cloneSpan(spans[4])
	duplicate.Correlation.OmniRequestID = "omni-duplicate"
	duplicate.Outcome = "duplicate"

	report := Assemble(append(spans, duplicate))
	assertAssemblyAnomaly(t, report, AnomalyDuplicateSpan, HopRoute)
	if report.AllContinuous {
		t.Fatal("duplicate via-join hop must break continuity")
	}
	if got := report.Traces[0].Hops[HopRoute].Outcome; got != spans[4].Outcome {
		t.Fatalf("duplicate overwrote first via-join hop: got %q", got)
	}
}

func TestAssembleRejectsConflictingResolverMappings(t *testing.T) {
	tests := []struct {
		name string
		hop  HopKind
		join func(first, second []Span)
	}{
		{
			name: "request",
			hop:  HopIngress,
			join: func(first, second []Span) {
				second[0].Correlation.RequestID = first[0].Correlation.RequestID
			},
		},
		{
			name: "launch",
			hop:  HopAdmission,
			join: func(first, second []Span) {
				second[2].Correlation.LaunchID = first[2].Correlation.LaunchID
			},
		},
		{
			name: "session",
			hop:  HopAdmission,
			join: func(first, second []Span) {
				second[2].Correlation.SessionID = first[2].Correlation.SessionID
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := SyntheticTraceSpans("t1")
			second := SyntheticTraceSpans("t2")
			test.join(first, second)

			report := Assemble(append(first, second...))
			assertAssemblyAnomaly(t, report, AnomalyConflictingJoin, test.hop)
			if report.AllContinuous {
				t.Fatal("conflicting resolver mapping must break continuity")
			}
			for _, anomaly := range report.Anomalies {
				for _, identifier := range []string{
					anomaly.Correlation.RequestID,
					anomaly.Correlation.LaunchID,
					anomaly.Correlation.SessionID,
				} {
					if identifier != "" && strings.Contains(anomaly.Reason, identifier) {
						t.Fatal("anomaly reason echoed a correlation identifier")
					}
				}
			}
		})
	}
}

func TestAssembleClonesPlacedSpanReferenceFields(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	report := Assemble(spans)
	if !report.AllContinuous {
		t.Fatalf("expected continuous baseline: %+v", report)
	}

	spans[0].Labels["method"] = "PATCH"
	spans[0].Counters["latency_ms"] = 999
	spans[3].ArgvShape[0] = "flag"

	trace := report.Traces[0]
	if trace.Hops[HopIngress].Labels["method"] != "POST" {
		t.Fatal("assembled labels alias caller input")
	}
	if trace.Hops[HopIngress].Counters["latency_ms"] != 8 {
		t.Fatal("assembled counters alias caller input")
	}
	if trace.Hops[HopCLI].ArgvShape[0] != "subcommand" {
		t.Fatal("assembled argv shape aliases caller input")
	}
}

func assertAssemblyAnomaly(t *testing.T, report AssemblyReport, kind AssemblyAnomalyKind, hop HopKind) {
	t.Helper()
	for _, anomaly := range report.Anomalies {
		if anomaly.Kind == kind && anomaly.Hop == hop {
			return
		}
	}
	t.Fatalf("missing anomaly kind=%q hop=%q: %+v", kind, hop, report.Anomalies)
}
