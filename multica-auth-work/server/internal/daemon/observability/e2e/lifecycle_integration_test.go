package e2e

import "testing"

func allIDFields() []IDField {
	return []IDField{IDRequest, IDQueueMsg, IDTask, IDSession, IDLaunch, IDProc, IDOmniReq, IDResult, IDDelivery}
}

// TestOneLifecycleEmitsAllHopsIDsContinuousAndLeakClean is the consolidated
// OBS integration/regression proof: one lifecycle emitted through the real
// Recorder -> MemorySink path yields all 7 source hops, all 9 correlation IDs,
// an assembler report with AllContinuous=true and zero gaps/orphans/anomalies,
// a clean leak scan, and exactly one persist + one delivery span.
func TestOneLifecycleEmitsAllHopsIDsContinuousAndLeakClean(t *testing.T) {
	const taskID = "task-integ-1"
	sink := NewMemorySink()
	rec := NewRecorder(sink)

	// Real emit path: each hop is validated + single-span leak-scanned before
	// the sink accepts it (fail-closed). A refusal here would fail the test.
	if err := EmitSyntheticTask(rec, taskID); err != nil {
		t.Fatalf("recorder refused a lifecycle hop (fail-closed emit): %v", err)
	}
	spans := sink.Spans()

	// (1) all 7 source hops, exactly once each.
	if len(spans) != 7 {
		t.Fatalf("expected 7 emitted spans, got %d", len(spans))
	}
	byHop := map[HopKind]int{}
	for _, s := range spans {
		byHop[s.Hop]++
	}
	for _, h := range EmittingHops() {
		if byHop[h] != 1 {
			t.Fatalf("hop %q emitted %d times, want exactly 1", h, byHop[h])
		}
	}
	// exactly one persist + one delivery.
	if byHop[HopPersist] != 1 {
		t.Fatalf("persist span count = %d, want exactly 1", byHop[HopPersist])
	}
	if byHop[HopDelivery] != 1 {
		t.Fatalf("delivery span count = %d, want exactly 1", byHop[HopDelivery])
	}

	// (2) all 9 correlation IDs present somewhere in the lifecycle.
	seen := map[IDField]bool{}
	for _, s := range spans {
		for _, id := range allIDFields() {
			if s.Correlation.Get(id) != "" {
				seen[id] = true
			}
		}
	}
	for _, id := range allIDFields() {
		if !seen[id] {
			t.Fatalf("correlation id %q never present across the lifecycle", id)
		}
	}
	if len(seen) != 9 {
		t.Fatalf("expected 9 distinct correlation ids, got %d", len(seen))
	}

	// (3) assembler: AllContinuous, one trace, zero gaps/orphans/anomalies.
	report := Assemble(spans)
	if !report.AllContinuous {
		t.Fatalf("AllContinuous=false; orphans=%d anomalies=%d", len(report.Orphans), len(report.Anomalies))
	}
	if len(report.Traces) != 1 {
		t.Fatalf("traces=%d, want exactly 1", len(report.Traces))
	}
	if len(report.Orphans) != 0 {
		t.Fatalf("orphans=%d, want 0: %+v", len(report.Orphans), report.Orphans)
	}
	if len(report.Anomalies) != 0 {
		t.Fatalf("anomalies=%d, want 0: %+v", len(report.Anomalies), report.Anomalies)
	}
	tr := report.Traces[0]
	if !tr.Continuous {
		t.Fatal("single trace must be continuous")
	}
	if len(tr.Missing) != 0 {
		t.Fatalf("trace gaps (missing hops)=%v, want none", tr.Missing)
	}
	if len(tr.Present) != 7 {
		t.Fatalf("present hops=%d, want 7", len(tr.Present))
	}
	if tr.TaskID != taskID {
		t.Fatalf("trace task id=%q, want %q", tr.TaskID, taskID)
	}

	// (4) structural leak scan over the recorded spans is clean.
	scan := ScanFromSink(sink)
	if !scan.Clean {
		t.Fatalf("leak scan not clean: %+v", scan.Findings)
	}
	if scan.Scanned != 7 {
		t.Fatalf("leak scan scanned=%d, want 7", scan.Scanned)
	}
	if len(scan.Findings) != 0 {
		t.Fatalf("leak findings=%d, want 0", len(scan.Findings))
	}
}

// TestLifecycleRegressionMissingHopBreaksContinuity proves the continuity
// assertion has teeth: dropping the persist hop yields a gap and AllContinuous=false.
func TestLifecycleRegressionMissingHopBreaksContinuity(t *testing.T) {
	spans := SyntheticTraceSpans("task-gap")
	reduced := make([]Span, 0, len(spans)-1)
	for _, s := range spans {
		if s.Hop != HopPersist {
			reduced = append(reduced, s)
		}
	}
	report := Assemble(reduced)
	if report.AllContinuous {
		t.Fatal("missing persist hop must break AllContinuous")
	}
	if len(report.Traces) != 1 {
		t.Fatalf("traces=%d, want 1", len(report.Traces))
	}
	if report.Traces[0].Continuous {
		t.Fatal("trace missing a hop must be discontinuous")
	}
	foundGap := false
	for _, m := range report.Traces[0].Missing {
		if m == HopPersist {
			foundGap = true
		}
	}
	if !foundGap {
		t.Fatalf("missing hops must include persist, got %v", report.Traces[0].Missing)
	}
}

// TestLifecycleRegressionSecretLeakNotClean proves the leak-clean assertion has
// teeth: a secrets_present violation makes the scan not clean.
func TestLifecycleRegressionSecretLeakNotClean(t *testing.T) {
	spans := SyntheticTraceSpans("task-leak")
	spans[0].SecretsPresent = true // tamper: violate the metadata-only invariant
	scan := ScanSpans(spans)
	if scan.Clean {
		t.Fatal("secrets_present violation must make the leak scan not-clean")
	}
	if len(scan.Findings) == 0 {
		t.Fatal("expected at least one leak finding for the tampered span")
	}
}
