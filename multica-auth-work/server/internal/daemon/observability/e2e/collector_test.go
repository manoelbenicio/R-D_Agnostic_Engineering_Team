package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeSpanLog emits the given spans to path as JSONLSink JSON lines, exactly as
// a live process would export them for cross-process collection.
func writeSpanLog(t *testing.T, path string, spans []Span) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()
	sink := NewJSONLSink(f)
	for _, s := range spans {
		if err := sink.Record(s); err != nil {
			t.Fatalf("record span %s: %v", s.Hop, err)
		}
	}
}

func okSpan(hop HopKind, c Correlation) Span {
	return *NewSpan(hop, c).WithOutcome("ok", "").Finish()
}

// trace1Backend/Daemon model the real cross-process split: the control API + DB
// + WS delivery run in the backend; admission/CLI/route run in the daemon.
func trace1BackendSpans() []Span {
	return []Span{
		okSpan(HopIngress, Correlation{RequestID: "req1", TaskID: "task1"}),
		okSpan(HopQueue, Correlation{QueueMsgID: "q1", TaskID: "task1"}),
		okSpan(HopPersist, Correlation{TaskID: "task1", ResultID: "res1"}),
		okSpan(HopDelivery, Correlation{SessionID: "sess1", DeliveryID: "deliv1"}),
	}
}

func trace1DaemonSpans() []Span {
	return []Span{
		okSpan(HopAdmission, Correlation{TaskID: "task1", SessionID: "sess1", LaunchID: "launch1"}),
		okSpan(HopCLI, Correlation{LaunchID: "launch1", ProcID: "proc1"}),
		okSpan(HopRoute, Correlation{RequestID: "req1", OmniRequestID: "omni1"}),
	}
}

func TestAssembleFromLogs_CrossProcessMerge_DroppedZeroAndContinuous(t *testing.T) {
	dir := t.TempDir()
	backend := filepath.Join(dir, "backend.jsonl")
	daemon := filepath.Join(dir, "daemon.jsonl")
	writeSpanLog(t, backend, trace1BackendSpans())
	writeSpanLog(t, daemon, trace1DaemonSpans())

	report, err := AssembleFromLogs(backend, daemon)
	if err != nil {
		t.Fatalf("AssembleFromLogs: %v", err)
	}
	if report.Dropped() != 0 {
		t.Fatalf("dropped=%d want 0 (durable log source); reasons=%v", report.Dropped(), report.Stats.DropReasons)
	}
	if report.Stats.SpanLines != 7 || report.Stats.Reconstructed != 7 {
		t.Fatalf("span_lines=%d reconstructed=%d want 7/7", report.Stats.SpanLines, report.Stats.Reconstructed)
	}
	if !report.Assembly.AllContinuous {
		t.Fatalf("expected AllContinuous; traces=%+v orphans=%+v anomalies=%+v",
			report.Assembly.Traces, report.Assembly.Orphans, report.Assembly.Anomalies)
	}
	if len(report.Assembly.Traces) != 1 {
		t.Fatalf("traces=%d want 1", len(report.Assembly.Traces))
	}
	tr := report.Assembly.Traces[0]
	if tr.TaskID != "task1" || !tr.Continuous || len(tr.Present) != 7 || len(tr.Missing) != 0 {
		t.Fatalf("trace not continuous: task=%s continuous=%v present=%v missing=%v",
			tr.TaskID, tr.Continuous, tr.Present, tr.Missing)
	}
}

func TestAssembleFromLogs_SecretsPresentLineFailsClosed(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tainted.jsonl")
	// A span that (illegally) declares secrets_present. Recorder.Emit would
	// reject it before export in production; here we prove the collector
	// independently fails closed on the durable line.
	tainted := Span{
		ContractVersion: ContractVersion,
		Hop:             HopIngress,
		Correlation:     Correlation{RequestID: "req1", TaskID: "task1"},
		StartedAt:       time.Now().UTC(),
		Outcome:         "ok",
		SecretsPresent:  true,
	}
	writeSpanLog(t, logPath, []Span{tainted})

	report, err := AssembleFromLogs(logPath)
	if err != nil {
		t.Fatalf("AssembleFromLogs: %v", err)
	}
	if report.Stats.SpanLines != 1 {
		t.Fatalf("span_lines=%d want 1", report.Stats.SpanLines)
	}
	if report.Dropped() != 1 || report.Stats.Reconstructed != 0 {
		t.Fatalf("dropped=%d reconstructed=%d want 1/0", report.Dropped(), report.Stats.Reconstructed)
	}
	if report.Stats.DropReasons["secrets_present"] != 1 {
		t.Fatalf("drop reasons=%v want secrets_present=1", report.Stats.DropReasons)
	}
	if len(report.Assembly.Traces) != 0 {
		t.Fatalf("tainted span must not assemble into a trace; traces=%+v", report.Assembly.Traces)
	}
}

func TestCollectSpans_IgnoresBlankAndNonJSONLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "mixed.jsonl")
	// Non-'{' lines (blank + plain text) are ignored, not dropped: a dedicated
	// JSONL export contains only span objects; incidental banners are skipped.
	content := "\nplain text banner line\n   \nnot json at all\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	spans, stats, err := CollectSpans(logPath)
	if err != nil {
		t.Fatalf("CollectSpans: %v", err)
	}
	if len(spans) != 0 || stats.SpanLines != 0 || stats.Dropped != 0 {
		t.Fatalf("spans=%d span_lines=%d dropped=%d want 0/0/0 (blank/non-JSON ignored)",
			len(spans), stats.SpanLines, stats.Dropped)
	}
	if stats.LinesTotal != 4 {
		t.Fatalf("lines_total=%d want 4", stats.LinesTotal)
	}
}
