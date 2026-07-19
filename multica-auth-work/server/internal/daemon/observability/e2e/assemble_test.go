package e2e

import "testing"

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
