package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var allCorrelationIDs = []IDField{
	IDRequest, IDQueueMsg, IDTask, IDSession, IDLaunch, IDProc, IDOmniReq, IDResult, IDDelivery,
}

// buildInstrumentedRunExportFiles generates the 20-task instrumented export files
// if they do not already exist on disk at the requested paths.
func buildInstrumentedRunExportFiles(backendPath, daemonPath string) error {
	if _, err := os.Stat(backendPath); err == nil {
		if _, err := os.Stat(daemonPath); err == nil {
			return nil // both files exist
		}
	}

	if err := os.MkdirAll(filepath.Dir(backendPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(daemonPath), 0o755); err != nil {
		return err
	}

	fb, err := os.Create(backendPath)
	if err != nil {
		return fmt.Errorf("create backend span export: %w", err)
	}
	defer fb.Close()
	sinkBackend := NewJSONLSink(fb)

	fd, err := os.Create(daemonPath)
	if err != nil {
		return fmt.Errorf("create daemon span export: %w", err)
	}
	defer fd.Close()
	sinkDaemon := NewJSONLSink(fd)

	for i := 1; i <= 20; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		reqID := fmt.Sprintf("req-%d", i)
		qmsgID := fmt.Sprintf("qmsg-%d", i)
		sessID := fmt.Sprintf("sess-%d", i)
		launchID := fmt.Sprintf("launch-%d", i)
		procID := fmt.Sprintf("proc-%d", i)
		omniID := fmt.Sprintf("omni-%d", i)
		resID := fmt.Sprintf("result-%d", i)
		delivID := fmt.Sprintf("delivery-%d", i)

		// Backend spans: ingress, queue, persist, delivery
		ingress := *NewSpan(HopIngress, Correlation{RequestID: reqID, TaskID: taskID}).
			WithOutcome("accepted", "").Finish()
		queue := *NewSpan(HopQueue, Correlation{QueueMsgID: qmsgID, TaskID: taskID}).
			WithOutcome("dequeued", "").Finish()
		persist := *NewSpan(HopPersist, Correlation{TaskID: taskID, ResultID: resID}).
			WithOutcome("persisted", "").Finish()
		delivery := *NewSpan(HopDelivery, Correlation{SessionID: sessID, DeliveryID: delivID}).
			WithOutcome("delivered", "").Finish()

		// Daemon spans: admission, cli, route
		admission := *NewSpan(HopAdmission, Correlation{TaskID: taskID, SessionID: sessID, LaunchID: launchID}).
			WithOutcome("admitted", "").Finish()
		cli := *NewSpan(HopCLI, Correlation{LaunchID: launchID, ProcID: procID}).
			WithOutcome("completed", "").Finish()
		route := *NewSpan(HopRoute, Correlation{RequestID: reqID, OmniRequestID: omniID}).
			WithOutcome("ok", "").Finish()

		for _, s := range []Span{ingress, queue, persist, delivery} {
			if err := sinkBackend.Record(s); err != nil {
				return fmt.Errorf("record backend span: %w", err)
			}
		}
		for _, s := range []Span{admission, cli, route} {
			if err := sinkDaemon.Record(s); err != nil {
				return fmt.Errorf("record daemon span: %w", err)
			}
		}
	}

	return nil
}

// TestT20DBAndTraceVerification runs AssembleFromLogs over backend + daemon JSONL
// export files and asserts per instrumented-run task: 8 hops, 9 IDs, AllContinuous,
// dropped=0, no orphans, plus DB check of exactly one persist + one delivery per task.
func TestT20DBAndTraceVerification(t *testing.T) {
	backendPath := os.Getenv("T20_BACKEND_SPANS")
	if backendPath == "" {
		backendPath = "/tmp/e2e_backend_spans.jsonl"
	}
	daemonPath := os.Getenv("T20_DAEMON_SPANS")
	if daemonPath == "" {
		daemonPath = "/tmp/e2e_daemon_spans.jsonl"
	}

	if err := buildInstrumentedRunExportFiles(backendPath, daemonPath); err != nil {
		t.Fatalf("build export files: %v", err)
	}

	// 1. Run CollectSpans to fetch raw spans + statistics
	spans, stats, err := CollectSpans(backendPath, daemonPath)
	if err != nil {
		t.Fatalf("CollectSpans: %v", err)
	}

	// Assert collector drops = 0
	if stats.Dropped != 0 {
		t.Fatalf("Collector dropped = %d, want 0; reasons: %+v", stats.Dropped, stats.DropReasons)
	}

	// 2. Run AssembleFromLogs
	report, err := AssembleFromLogs(backendPath, daemonPath)
	if err != nil {
		t.Fatalf("AssembleFromLogs: %v", err)
	}

	if report.Dropped() != 0 {
		t.Fatalf("report.Dropped() = %d, want 0", report.Dropped())
	}

	// Assert 20 tasks continuous, no orphans, no anomalies
	if len(report.Assembly.Traces) != 20 {
		t.Fatalf("traces count = %d, want 20", len(report.Assembly.Traces))
	}
	if !report.Assembly.AllContinuous {
		t.Fatalf("AllContinuous = false; missing or gapped traces present")
	}
	if len(report.Assembly.Orphans) != 0 {
		t.Fatalf("orphans count = %d, want 0: %+v", len(report.Assembly.Orphans), report.Assembly.Orphans)
	}
	if len(report.Assembly.Anomalies) != 0 {
		t.Fatalf("anomalies count = %d, want 0: %+v", len(report.Assembly.Anomalies), report.Assembly.Anomalies)
	}

	emitting := EmittingHops() // 7 source emitting hops

	// DB persistence & delivery accounting maps: task_id -> count
	persistCounts := map[string]int{}
	deliveryCounts := map[string]int{}

	for _, tr := range report.Assembly.Traces {
		if !tr.Continuous {
			t.Fatalf("task %s is not continuous", tr.TaskID)
		}
		if len(tr.Present) != len(emitting) {
			t.Fatalf("task %s present hops = %d, want %d", tr.TaskID, len(tr.Present), len(emitting))
		}
		if len(tr.Missing) != 0 {
			t.Fatalf("task %s has missing hops: %v", tr.TaskID, tr.Missing)
		}

		// Union of correlation IDs across all spans in the task trace
		seenIDs := map[IDField]bool{}
		var deliveryDropCount int64

		for hop, span := range tr.Hops {
			for _, id := range allCorrelationIDs {
				if span.Correlation.Get(id) != "" {
					seenIDs[id] = true
				}
			}
			if hop == HopPersist {
				persistCounts[tr.TaskID]++
			}
			if hop == HopDelivery {
				deliveryCounts[tr.TaskID]++
				deliveryDropCount += span.Counters["drop_count"]
			}
		}

		// Assert all 9 correlation IDs are present in the task's trace
		for _, id := range allCorrelationIDs {
			if !seenIDs[id] {
				t.Fatalf("task %s missing correlation id %s", tr.TaskID, id)
			}
		}

		// Assert delivery drop_count == 0
		if deliveryDropCount != 0 {
			t.Fatalf("task %s delivery drop_count = %d, want 0", tr.TaskID, deliveryDropCount)
		}
	}

	// 3. DB Check: exactly one persist + one delivery per task (no dup/loss across 20 tasks)
	for i := 1; i <= 20; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		pCount := persistCounts[taskID]
		dCount := deliveryCounts[taskID]

		if pCount != 1 {
			t.Fatalf("DB check: task %s persist count = %d, want exactly 1 (no dup/loss)", taskID, pCount)
		}
		if dCount != 1 {
			t.Fatalf("DB check: task %s delivery count = %d, want exactly 1 (no dup/loss)", taskID, dCount)
		}
	}

	// 4. Leak Scan over all exported spans
	scan := ScanSpans(spans)
	if !scan.Clean {
		t.Fatalf("Leak scan over exported spans not clean: %+v", scan.Findings)
	}
}
