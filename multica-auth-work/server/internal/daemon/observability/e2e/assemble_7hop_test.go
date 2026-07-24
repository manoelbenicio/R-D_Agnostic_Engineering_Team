package e2e

// assemble_7hop_test.go — deterministic assembler-logic guard for e2e.Assemble.
// Builds all seven emitting-hop spans with join keys from the real e2e.Canonical*
// helpers (so ingress↔route join on request_id and admission↔delivery join on
// session_id), then asserts the happy continuous assembly plus the three
// negative guards. This is the assembler-logic guard, NOT a live run.

import "testing"

// guard7Span builds a minimal valid finished span for one hop (uniquely named to
// avoid colliding with helpers in other test files in this package).
func guard7Span(hop HopKind, c Correlation) Span {
	return *NewSpan(hop, c).WithOutcome("ok", "").Finish()
}

// sevenHopHappySpans returns the 7 emitting-hop spans for one task, joined by the
// canonical request_id/session_id and the raw task UUID (as production uses for
// task_id/queue_msg_id/result_id). launch_id/proc_id/omni_request_id/delivery_id
// are opaque leaf ids shared where a join requires it (launch_id: admission↔cli).
func sevenHopHappySpans() ([]Span, string) {
	const (
		task = "b7e2c1a0-1111-2222-3333-444455556666" // raw task-row UUID
		chat = "c1111111-2222-3333-4444-555555555555" // raw chat-session UUID
	)
	req := CanonicalRequestID(task)        // ingress == route
	sess := CanonicalSessionID(chat, task) // admission == delivery
	const (
		launch   = "launch7hopguardopaque" // admission == cli
		proc     = "4242"                  // real-shaped decimal pid
		omni     = "omnireq7hopguardopaque"
		delivery = "deliv7hopguardopaque"
	)
	return []Span{
		guard7Span(HopIngress, Correlation{RequestID: req, TaskID: task}),
		guard7Span(HopQueue, Correlation{QueueMsgID: task, TaskID: task}),
		guard7Span(HopAdmission, Correlation{TaskID: task, SessionID: sess, LaunchID: launch}),
		guard7Span(HopCLI, Correlation{LaunchID: launch, ProcID: proc}),
		guard7Span(HopRoute, Correlation{RequestID: req, OmniRequestID: omni}),
		guard7Span(HopPersist, Correlation{TaskID: task, ResultID: task}),
		guard7Span(HopDelivery, Correlation{SessionID: sess, DeliveryID: delivery}),
	}, task
}

// TestAssemble7HopHappyPath: all 7 hops join into one continuous trace with no
// orphans/anomalies and the spans are leak-clean.
func TestAssemble7HopHappyPath(t *testing.T) {
	spans, task := sevenHopHappySpans()

	report := Assemble(spans)
	if !report.AllContinuous {
		t.Fatalf("AllContinuous=false; orphans=%+v anomalies=%+v", report.Orphans, report.Anomalies)
	}
	if len(report.Traces) != 1 || len(report.Orphans) != 0 || len(report.Anomalies) != 0 {
		t.Fatalf("traces=%d orphans=%d anomalies=%d, want 1/0/0", len(report.Traces), len(report.Orphans), len(report.Anomalies))
	}
	tr := report.Traces[0]
	if tr.TaskID != task || !tr.Continuous || len(tr.Present) != 7 || len(tr.Missing) != 0 {
		t.Fatalf("trace not continuous/complete: %+v", tr)
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

	if scan := ScanSpans(spans); !scan.Clean || len(scan.Findings) != 0 {
		t.Fatalf("leak scan not clean: %+v", scan.Findings)
	}
}

// TestAssemble7HopDuplicateDeliveryIsAnomaly (guard a): a second delivery span for
// the same task yields a duplicate_task_hop anomaly and breaks continuity.
func TestAssemble7HopDuplicateDeliveryIsAnomaly(t *testing.T) {
	spans, _ := sevenHopHappySpans()
	var delivery Span
	for _, s := range spans {
		if s.Hop == HopDelivery {
			delivery = s
		}
	}
	spans = append(spans, delivery) // duplicate delivery, same session/task

	report := Assemble(spans)
	if report.AllContinuous {
		t.Fatal("expected AllContinuous=false with a duplicate delivery span")
	}
	found := false
	for _, a := range report.Anomalies {
		if a.Kind == AnomalyDuplicateSpan && a.Hop == HopDelivery {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate_task_hop anomaly for the delivery hop; anomalies=%+v", report.Anomalies)
	}
}

// TestAssemble7HopMissingHopNotContinuous (guard b): dropping one emitting hop
// leaves the trace incomplete (not AllContinuous) with that hop in Missing.
func TestAssemble7HopMissingHopNotContinuous(t *testing.T) {
	spans, _ := sevenHopHappySpans()
	// Drop the persist hop (index 5); the 3-index cap prevents clobbering.
	dropped := append(spans[:5:5], spans[6:]...)

	report := Assemble(dropped)
	if report.AllContinuous {
		t.Fatal("expected AllContinuous=false with a missing hop")
	}
	if len(report.Traces) != 1 {
		t.Fatalf("traces=%d, want 1", len(report.Traces))
	}
	tr := report.Traces[0]
	if tr.Continuous {
		t.Fatal("trace should be incomplete after dropping a hop")
	}
	missingPersist := false
	for _, h := range tr.Missing {
		if h == HopPersist {
			missingPersist = true
		}
	}
	if !missingPersist {
		t.Fatalf("HopPersist should be reported missing; missing=%+v", tr.Missing)
	}
}

// TestAssemble7HopWrongIDOrphans (guard c): a backfilled/wrong request_id on the
// route hop (not the canonical value) cannot resolve to the task and is reported
// as an orphan; the trace is not AllContinuous.
func TestAssemble7HopWrongIDOrphans(t *testing.T) {
	spans, _ := sevenHopHappySpans()
	for i := range spans {
		if spans[i].Hop == HopRoute {
			spans[i] = guard7Span(HopRoute, Correlation{
				RequestID:     "reqbackfilledwrongvalue", // NOT CanonicalRequestID(task)
				OmniRequestID: spans[i].Correlation.OmniRequestID,
			})
		}
	}

	report := Assemble(spans)
	if report.AllContinuous {
		t.Fatal("expected AllContinuous=false with a wrong/backfilled route request_id")
	}
	found := false
	for _, o := range report.Orphans {
		if o.Hop == HopRoute {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a HopRoute orphan (request_id_unresolved); orphans=%+v", report.Orphans)
	}
}
