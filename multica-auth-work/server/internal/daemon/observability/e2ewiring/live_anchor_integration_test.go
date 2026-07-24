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

// TestLiveAnchorProductionDiscrepanciesExposeFailingJoins drives the actual production
// call sites with unaligned raw ID derivations across seams. It proves that without
// canonical ID propagation, Assembly FAILS due to three specific discrepancies:
//
// 1. Raw UUID vs Task-Hash: Ingress/Queue/Persist use raw UUID ("11111111-2222-3333-4444-555555555555"),
//    while Admission uses a derived task-hash ("thash-a1b2c3d4e5f6"), splitting the lifecycle into
//    two fragmented, discontinuous traces.
// 2. Independent Request IDs: Ingress stamps "req-http-ingress-9999", while Gateway Route
//    independently stamps "req-gateway-route-7777", causing the Route span to be orphaned.
// 3. Random WS Session Mismatch: Admission stamps "sess-chat-user-123", while DaemonWS Delivery
//    mints a random per-connection session ID "wsdel-session-random-99", causing the Delivery span to be orphaned.
func TestLiveAnchorProductionDiscrepanciesExposeFailingJoins(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	// -------------------------------------------------------------------------
	// 1. Ingress (middleware) - uses raw HTTP Request ID and raw Task UUID
	// -------------------------------------------------------------------------
	rawTaskUUID := "11111111-2222-3333-4444-555555555555"
	ingressReqID := "req-http-ingress-9999"

	if err := middleware.EmitIngress(rec, middleware.IngressObservation{
		RequestID:     ingressReqID,
		TaskID:        rawTaskUUID,
		Method:        "POST",
		RouteTemplate: "/v1/tasks",
		HTTPStatus:    202,
		Outcome:       "accepted",
		LatencyMs:     5,
	}); err != nil {
		t.Fatalf("ingress emit failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 2. Queue (service) - uses raw Task UUID
	// -------------------------------------------------------------------------
	if err := service.EmitQueue(rec, service.QueueObservation{
		QueueMsgID:    "qmsg-101",
		TaskID:        rawTaskUUID,
		Outcome:       "dequeued",
		WaitMs:        2,
		EnqueueUnixMs: 10,
		DequeueUnixMs: 12,
	}); err != nil {
		t.Fatalf("queue emit failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 3. Admission (brain) - DISCREPANCY 1: uses task-hash instead of raw UUID
	// -------------------------------------------------------------------------
	derivedTaskHash := "thash-a1b2c3d4e5f6"
	admissionSessID := "sess-chat-user-123"

	admObserver := brain.NewAdmissionObserver(sink)
	if err := admObserver.Emit(brain.AdmissionObservation{
		Correlation: brain.Correlation{
			TaskID:    derivedTaskHash, // task-hash mismatch
			SessionID: admissionSessID,
		},
		CLIKind:     brain.CLIClaudeCode,
		RouteModel:  brain.RouteModel("cp/cline-pass/glm-5.2"),
		Decision:    brain.AdmissionAdmitted,
		Readiness:   brain.GatewayReadinessReady,
		StartedAt:   time.Now().Add(-5 * time.Millisecond),
	}); err != nil {
		t.Fatalf("admission emit failed: %v", err)
	}

	// Extract generated launch_id from admission span
	admSpan, ok := findHop(sink.Spans(), e2e.HopAdmission)
	if !ok || admSpan.Correlation.LaunchID == "" {
		t.Fatalf("admission span missing launch_id")
	}
	launchID := admSpan.Correlation.LaunchID

	// -------------------------------------------------------------------------
	// 4. CLI (daemon) - joins on launch_id
	// -------------------------------------------------------------------------
	if err := daemon.EmitCLI(rec, daemon.CLIObservation{
		LaunchID:      launchID,
		ProcID:        "proc-555",
		TaskID:        derivedTaskHash,
		CLIKind:       "agent",
		ExitCodeClass: "exit_0",
		LatencyMs:     10,
		Outcome:       "completed",
		ReasonCode:    "ok",
	}); err != nil {
		t.Fatalf("cli emit failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 5. Route (gateway) - DISCREPANCY 2: uses independent gateway request ID
	// -------------------------------------------------------------------------
	gatewayReqID := "req-gateway-route-7777" // independent request ID mismatch
	omniReqID := "omni-probe-8888"

	start := time.Now().Add(-30 * time.Millisecond)
	if err := gateway.EmitProviderSpan(rec, gateway.ProviderSpanRecord{
		RequestID:           gatewayReqID,
		PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol:           brain.ProtocolAnthropicMessages,
		Telemetry:           actualTelemetry(omniReqID),
		StartedAt:           start,
		EndedAt:             start.Add(20 * time.Millisecond),
		Outcome:             "ok",
	}); err != nil {
		t.Fatalf("route emit failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 6. Persist (service) - uses raw Task UUID
	// -------------------------------------------------------------------------
	if err := service.EmitPersist(rec, service.PersistObservation{
		TaskID:           rawTaskUUID,
		ResultID:         "result-303",
		TerminalStatus:   "completed",
		PersistLatencyMs: 3,
		ByteCount:        150,
		TokenCount:       45,
		Outcome:          "persisted",
	}); err != nil {
		t.Fatalf("persist emit failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 7. Delivery (daemonws) - DISCREPANCY 3: uses random WS session ID
	// -------------------------------------------------------------------------
	randomWSSessionID := "wsdel-session-random-99" // random WS session mismatch

	if err := daemonws.NewDeliveryRecorder(rec).EmitDelivery(daemonws.DeliveryResult{
		SessionID:  randomWSSessionID,
		DeliveryID: "delivery-404",
		Outcome:    "delivered",
		LatencyMs:  1,
	}); err != nil {
		t.Fatalf("delivery emit failed: %v", err)
	}

	// =========================================================================
	// ASSERTIONS ON FAILING STATE
	// =========================================================================
	spans := sink.Spans()
	if len(spans) != 7 {
		t.Fatalf("expected 7 spans recorded from production call sites, got %d", len(spans))
	}

	report := e2e.Assemble(spans)

	// Invariant 1: AllContinuous MUST BE FALSE due to ID mismatches
	if report.AllContinuous {
		t.Errorf("EXPECTED FAIL: AllContinuous should be false for unaligned production call sites, got true")
	}

	// Invariant 2: Exactly 2 Orphan Spans expected (Route & Delivery)
	if len(report.Orphans) != 2 {
		t.Errorf("EXPECTED 2 ORPHANS (Route + Delivery), got %d: %+v", len(report.Orphans), report.Orphans)
	} else {
		// Confirm orphaned hops
		orphanHops := map[e2e.HopKind]bool{}
		for _, o := range report.Orphans {
			orphanHops[o.Hop] = true
		}
		if !orphanHops[e2e.HopRoute] {
			t.Errorf("expected Route span to be orphaned due to independent request ID mismatch")
		}
		if !orphanHops[e2e.HopDelivery] {
			t.Errorf("expected Delivery span to be orphaned due to random WS session mismatch")
		}
	}

	// Invariant 3: Traces are fragmented into 2 incomplete task traces
	if len(report.Traces) != 2 {
		t.Errorf("EXPECTED 2 FRAGMENTED TRACES (raw UUID vs task-hash), got %d", len(report.Traces))
	} else {
		for _, tr := range report.Traces {
			if tr.Continuous {
				t.Errorf("fragmented trace %s should be discontinuous", tr.TaskID)
			}
		}
	}
}

// TestTargetCanonicalIDPropagationProducesContinuousTrace drives the same production
// call sites with canonical ID propagation applied across all seams. It proves that
// when canonical IDs are passed correctly across seams, Assembly SUCCEEDS.
func TestTargetCanonicalIDPropagationProducesContinuousTrace(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	// Canonical IDs propagated end-to-end
	const canonicalTaskID = "task-canonical-uuid-100"
	const canonicalReqID = "req-canonical-http-200"
	const canonicalSessID = "sess-canonical-user-300"

	qmsgID := "qmsg-canonical-400"
	procID := "proc-canonical-500"
	omniReqID := "omni-canonical-600"
	resultID := "result-canonical-700"
	deliveryID := "delivery-canonical-800"

	// 1. Ingress
	if err := middleware.EmitIngress(rec, middleware.IngressObservation{
		RequestID:     canonicalReqID,
		TaskID:        canonicalTaskID,
		Method:        "POST",
		RouteTemplate: "/v1/tasks",
		HTTPStatus:    202,
		Outcome:       "accepted",
		LatencyMs:     4,
	}); err != nil {
		t.Fatalf("ingress emit failed: %v", err)
	}

	// 2. Queue
	if err := service.EmitQueue(rec, service.QueueObservation{
		QueueMsgID:    qmsgID,
		TaskID:        canonicalTaskID,
		Outcome:       "dequeued",
		WaitMs:        1,
		EnqueueUnixMs: 20,
		DequeueUnixMs: 21,
	}); err != nil {
		t.Fatalf("queue emit failed: %v", err)
	}

	// 3. Admission - uses canonicalTaskID and canonicalSessID
	admObserver := brain.NewAdmissionObserver(sink)
	if err := admObserver.Emit(brain.AdmissionObservation{
		Correlation: brain.Correlation{
			TaskID:    canonicalTaskID,
			SessionID: canonicalSessID,
		},
		CLIKind:    brain.CLIClaudeCode,
		RouteModel: brain.RouteModel("cp/cline-pass/glm-5.2"),
		Decision:   brain.AdmissionAdmitted,
		Readiness:  brain.GatewayReadinessReady,
		StartedAt:  time.Now().Add(-4 * time.Millisecond),
	}); err != nil {
		t.Fatalf("admission emit failed: %v", err)
	}

	admSpan, ok := findHop(sink.Spans(), e2e.HopAdmission)
	if !ok || admSpan.Correlation.LaunchID == "" {
		t.Fatalf("admission span missing launch_id")
	}
	launchID := admSpan.Correlation.LaunchID

	// 4. CLI
	if err := daemon.EmitCLI(rec, daemon.CLIObservation{
		LaunchID:      launchID,
		ProcID:        procID,
		TaskID:        canonicalTaskID,
		CLIKind:       "agent",
		ExitCodeClass: "exit_0",
		LatencyMs:     8,
		Outcome:       "completed",
		ReasonCode:    "ok",
	}); err != nil {
		t.Fatalf("cli emit failed: %v", err)
	}

	// 5. Route - uses canonicalReqID (propagated via X-AB-Request-Id header)
	start := time.Now().Add(-25 * time.Millisecond)
	if err := gateway.EmitProviderSpan(rec, gateway.ProviderSpanRecord{
		RequestID:           canonicalReqID, // aligned request_id
		PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol:           brain.ProtocolAnthropicMessages,
		Telemetry:           actualTelemetry(omniReqID),
		StartedAt:           start,
		EndedAt:             start.Add(18 * time.Millisecond),
		Outcome:             "ok",
	}); err != nil {
		t.Fatalf("route emit failed: %v", err)
	}

	// 6. Persist
	if err := service.EmitPersist(rec, service.PersistObservation{
		TaskID:           canonicalTaskID,
		ResultID:         resultID,
		TerminalStatus:   "completed",
		PersistLatencyMs: 2,
		ByteCount:        120,
		TokenCount:       35,
		Outcome:          "persisted",
	}); err != nil {
		t.Fatalf("persist emit failed: %v", err)
	}

	// 7. Delivery - uses canonicalSessID (propagated from admission session)
	if err := daemonws.NewDeliveryRecorder(rec).EmitDelivery(daemonws.DeliveryResult{
		SessionID:  canonicalSessID, // aligned session_id
		DeliveryID: deliveryID,
		Outcome:    "delivered",
		LatencyMs:  1,
	}); err != nil {
		t.Fatalf("delivery emit failed: %v", err)
	}

	// =========================================================================
	// ASSERTIONS ON TARGET CANONICAL STATE
	// =========================================================================
	spans := sink.Spans()
	report := e2e.Assemble(spans)

	if !report.AllContinuous {
		t.Fatalf("TARGET FAIL: AllContinuous should be true for canonical ID propagation, orphans=%d anomalies=%d",
			len(report.Orphans), len(report.Anomalies))
	}
	if len(report.Orphans) != 0 {
		t.Fatalf("target orphans = %d, want 0: %+v", len(report.Orphans), report.Orphans)
	}
	if len(report.Anomalies) != 0 {
		t.Fatalf("target anomalies = %d, want 0: %+v", len(report.Anomalies), report.Anomalies)
	}
	if len(report.Traces) != 1 {
		t.Fatalf("target traces = %d, want 1", len(report.Traces))
	}

	tr := report.Traces[0]
	if !tr.Continuous || len(tr.Present) != 7 || len(tr.Missing) != 0 {
		t.Fatalf("trace not complete and continuous: %+v", tr)
	}
	if tr.TaskID != canonicalTaskID {
		t.Fatalf("trace TaskID = %s, want %s", tr.TaskID, canonicalTaskID)
	}
}
