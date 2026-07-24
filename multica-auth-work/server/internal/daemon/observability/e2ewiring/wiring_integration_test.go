package e2ewiring_test

import (
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon"
	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/daemonws"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/service"
)

func findHop(spans []e2e.Span, hop e2e.HopKind) (e2e.Span, bool) {
	for _, s := range spans {
		if s.Hop == hop {
			return s, true
		}
	}
	return e2e.Span{}, false
}

func actualTelemetry(omniReqID string) gateway.Telemetry {
	return gateway.Telemetry{
		RequestID:              omniReqID,
		ActualModel:            "cp/cline-pass/glm-5.2",
		ActualRoute:            "route-a",
		PseudonymousAccount:    "acct_0123456789abcdef",
		PseudonymousConnection: "conn_fedcba9876543210",
		SelectionReason:        gateway.SelectionIndependentRotation,
		Quota:                  gateway.QuotaAvailable,
		Circuit:                gateway.CircuitClosed,
		Usage:                  gateway.Usage{Input: 10, Output: 20, Total: 30},
	}
}

// TestProductionSeamsEmitContinuousLifecycle drives the REAL per-seam emitters
// (not SyntheticTraceSpans) into one shared sink and proves they produce real,
// spec-valid hop spans with real IDs that assemble into one continuous trace:
// all 7 source hops, all 9 IDs, AllContinuous, zero gaps/orphans/anomalies,
// leak-clean, exactly one persist + one delivery.
func TestProductionSeamsEmitContinuousLifecycle(t *testing.T) {
	const task = "task-prod-1"
	reqID := "req-" + task
	qmsg := "qmsg-" + task
	sess := "sess-" + task
	proc := "proc-" + task
	omni := "omni-" + task
	result := "result-" + task
	delivery := "delivery-" + task

	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	// hop 1 — ingress (middleware)
	if err := middleware.EmitIngress(rec, middleware.IngressObservation{
		RequestID: reqID, TaskID: task, Method: "POST", RouteTemplate: "/v1/tasks",
		HTTPStatus: 202, Outcome: "accepted", LatencyMs: 5,
	}); err != nil {
		t.Fatalf("ingress emit refused: %v", err)
	}
	// hop 2 — queue (service)
	if err := service.EmitQueue(rec, service.QueueObservation{
		QueueMsgID: qmsg, TaskID: task, Outcome: "dequeued",
		WaitMs: 1, QueueDepth: 0, EnqueueUnixMs: 1, DequeueUnixMs: 2,
	}); err != nil {
		t.Fatalf("queue emit refused: %v", err)
	}
	// hop 3 — admission (brain observer over the SAME sink)
	admObserver := brain.NewAdmissionObserver(sink)
	if err := admObserver.Emit(brain.AdmissionObservation{
		Correlation: brain.Correlation{TaskID: task, SessionID: sess},
		CLIKind:     brain.CLIClaudeCode, RouteModel: brain.RouteModel("cp/cline-pass/glm-5.2"),
		Decision: brain.AdmissionAdmitted, Readiness: brain.GatewayReadinessReady,
		StartedAt: time.Now().Add(-2 * time.Millisecond),
	}); err != nil {
		t.Fatalf("admission emit refused: %v", err)
	}
	// The admission hop derives its own opaque launch_id; read it back so the
	// CLI hop joins on the SAME launch_id (real cross-hop join, not a guess).
	adm, ok := findHop(sink.Spans(), e2e.HopAdmission)
	if !ok || adm.Correlation.LaunchID == "" {
		t.Fatalf("admission span/launch_id missing: ok=%v span=%+v", ok, adm)
	}
	launchID := adm.Correlation.LaunchID

	// hop 4 — CLI (daemon)
	if err := daemon.EmitCLI(rec, daemon.CLIObservation{
		LaunchID: launchID, ProcID: proc, TaskID: task,
		CLIKind: "agent", ExitCodeClass: "exit_0", LatencyMs: 3,
		Outcome: "completed", ReasonCode: "ok",
	}); err != nil {
		t.Fatalf("cli emit refused: %v", err)
	}
	// hop 5 — route (gateway) from an actual-telemetry fixture
	start := time.Now().Add(-50 * time.Millisecond)
	if err := gateway.EmitProviderSpan(rec, gateway.ProviderSpanRecord{
		RequestID: reqID, PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol: brain.ProtocolAnthropicMessages, Telemetry: actualTelemetry(omni),
		StartedAt: start, EndedAt: start.Add(40 * time.Millisecond), Outcome: "ok",
	}); err != nil {
		t.Fatalf("route emit refused: %v", err)
	}
	// hop 6 — persist (service)
	if err := service.EmitPersist(rec, service.PersistObservation{
		TaskID: task, ResultID: result, TerminalStatus: "completed",
		PersistLatencyMs: 2, ByteCount: 100, TokenCount: 30, Outcome: "persisted",
	}); err != nil {
		t.Fatalf("persist emit refused: %v", err)
	}
	// hop 7 — delivery (daemonws)
	if err := daemonws.NewDeliveryRecorder(rec).EmitDelivery(daemonws.DeliveryResult{
		SessionID: sess, DeliveryID: delivery, Outcome: "delivered", LatencyMs: 1,
	}); err != nil {
		t.Fatalf("delivery emit refused: %v", err)
	}

	spans := sink.Spans()

	// all 7 source hops, exactly one each; exactly one persist + one delivery.
	byHop := map[e2e.HopKind]int{}
	for _, s := range spans {
		byHop[s.Hop]++
	}
	if len(spans) != 7 {
		t.Fatalf("expected 7 real hop spans, got %d (%v)", len(spans), byHop)
	}
	for _, h := range e2e.EmittingHops() {
		if byHop[h] != 1 {
			t.Fatalf("hop %q emitted %d times, want 1", h, byHop[h])
		}
	}
	if byHop[e2e.HopPersist] != 1 || byHop[e2e.HopDelivery] != 1 {
		t.Fatalf("want exactly one persist + one delivery, got persist=%d delivery=%d", byHop[e2e.HopPersist], byHop[e2e.HopDelivery])
	}

	// all 9 IDs present across the real spans.
	ids := []e2e.IDField{e2e.IDRequest, e2e.IDQueueMsg, e2e.IDTask, e2e.IDSession, e2e.IDLaunch, e2e.IDProc, e2e.IDOmniReq, e2e.IDResult, e2e.IDDelivery}
	seen := map[e2e.IDField]bool{}
	for _, s := range spans {
		for _, id := range ids {
			if s.Correlation.Get(id) != "" {
				seen[id] = true
			}
		}
	}
	for _, id := range ids {
		if !seen[id] {
			t.Fatalf("id %q never present across real seams", id)
		}
	}

	// assembler continuity over the real spans.
	report := e2e.Assemble(spans)
	if !report.AllContinuous {
		t.Fatalf("AllContinuous=false; orphans=%d anomalies=%d", len(report.Orphans), len(report.Anomalies))
	}
	if len(report.Traces) != 1 || len(report.Orphans) != 0 || len(report.Anomalies) != 0 {
		t.Fatalf("traces=%d orphans=%d anomalies=%d, want 1/0/0", len(report.Traces), len(report.Orphans), len(report.Anomalies))
	}
	if tr := report.Traces[0]; !tr.Continuous || len(tr.Missing) != 0 || len(tr.Present) != 7 || tr.TaskID != task {
		t.Fatalf("trace not continuous/complete: %+v", tr)
	}

	// leak scan clean over the real recorded spans.
	if scan := e2e.ScanFromSink(sink); !scan.Clean || scan.Scanned != 7 || len(scan.Findings) != 0 {
		t.Fatalf("leak scan not clean: clean=%v scanned=%d findings=%+v", scan.Clean, scan.Scanned, scan.Findings)
	}
}

// TestRouteHopEmitsFromActualTelemetryFixture tests the route hop separately with
// an actual gateway.Telemetry fixture: EmitProviderSpan produces a real,
// leak-clean HopRoute span carrying request_id + omni_request_id and the bounded
// route labels/counters.
func TestRouteHopEmitsFromActualTelemetryFixture(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	start := time.Now().Add(-30 * time.Millisecond)

	if err := gateway.EmitProviderSpan(rec, gateway.ProviderSpanRecord{
		RequestID: "req-route-1", PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol: brain.ProtocolAnthropicMessages, Telemetry: actualTelemetry("omni-route-1"),
		StartedAt: start, EndedAt: start.Add(25 * time.Millisecond), Outcome: "ok",
	}); err != nil {
		t.Fatalf("route emit from actual telemetry refused: %v", err)
	}

	spans := sink.Spans()
	if len(spans) != 1 || spans[0].Hop != e2e.HopRoute {
		t.Fatalf("expected exactly one HopRoute span, got %d (%+v)", len(spans), spans)
	}
	got := spans[0]
	if got.Correlation.RequestID != "req-route-1" || got.Correlation.OmniRequestID != "omni-route-1" {
		t.Fatalf("route span missing real IDs: %+v", got.Correlation)
	}
	if scan := e2e.ScanSpans(spans); !scan.Clean {
		t.Fatalf("route span not leak-clean: %+v", scan.Findings)
	}
}
