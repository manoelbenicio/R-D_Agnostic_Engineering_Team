package e2ewiring_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon"
	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
)

// TestProductionSeamStructuralAssemblyProductionDerivedIDs is a STRUCTURAL
// production-seam test. It drives the real exported hop emitters (middleware
// ingress, service queue/persist, brain admission, daemon CLI, gateway route)
// plus the real realtime terminal delivery seam, and asserts e2e.Assemble
// reconstructs exactly one continuous 7-hop trace using PRODUCTION-DERIVED
// correlation join keys:
//   - request_id  = e2e.CanonicalRequestID(taskUUID)             (ingress == route)
//   - session_id  = e2e.CanonicalSessionID(chat, taskUUID)       (admission == delivery)
//   - task_id / queue_msg_id / result_id = the raw task-row UUID (production queue + persist)
//   - launch_id   = admission's own AdmissionLaunchID, read back for the CLI hop
//   - proc_id     = decimal os.Getpid() (a real process id, not a fixture prefix)
//
// It also asserts none of the assembled ids are of the rejected
// "<prefix><task_id>" synthetic form (mirroring the e2e t20NoGeneratedIDs gate),
// proving the generated-ID gate ACCEPTS these production-derived ids.
//
// SCOPE / NON-CLAIMS: STRUCTURAL seam+assembly test only — NOT live acceptance,
// NOT real OTLP, NOT inference evidence. The route hop uses a gateway.Telemetry
// FIXTURE (actualTelemetry with a literal omni id) through the legacy
// provider-span emitter; no network/provider/inference occurs. It proves the
// seams wire and assemble with production-derived correlation, not that a real
// end-to-end run happened.
func TestProductionSeamStructuralAssemblyProductionDerivedIDs(t *testing.T) {
	const (
		task = "b7e2c1a0-1111-2222-3333-444455556666" // raw task-row UUID
		chat = "c1111111-2222-3333-4444-555555555555" // raw chat-session UUID
	)
	reqID := e2e.CanonicalRequestID(task)              // production-derived (ingress == route)
	canonSession := e2e.CanonicalSessionID(chat, task) // production-derived (admission == delivery)
	procID := strconv.Itoa(os.Getpid())                // real decimal process id
	const omniFixture = "omnireq0f1e2d3c4b5a6978"      // OmniRoute runtime id — FIXTURE (no canonical derivation)
	const deliveryEventID = "evtstructural01"          // broadcast dedup id (fixture)

	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	// hop 1 — ingress
	if err := middleware.EmitIngress(rec, middleware.IngressObservation{
		RequestID: reqID, TaskID: task, Method: "POST", RouteTemplate: "/v1/tasks",
		HTTPStatus: 202, Outcome: "accepted", LatencyMs: 5,
	}); err != nil {
		t.Fatalf("ingress emit refused: %v", err)
	}
	// hop 2 — queue (production queue_msg_id == raw task UUID)
	if err := service.EmitQueue(rec, service.QueueObservation{
		QueueMsgID: task, TaskID: task, Outcome: "dequeued", WaitMs: 1, EnqueueUnixMs: 1, DequeueUnixMs: 2,
	}); err != nil {
		t.Fatalf("queue emit refused: %v", err)
	}
	// hop 3 — admission (session = canonical); launch_id read back for CLI.
	admObserver := brain.NewAdmissionObserver(sink)
	if err := admObserver.Emit(brain.AdmissionObservation{
		Correlation: brain.Correlation{TaskID: task, SessionID: canonSession},
		CLIKind:     brain.CLIClaudeCode, RouteModel: brain.RouteModel("cp/cline-pass/glm-5.2"),
		Decision: brain.AdmissionAdmitted, Readiness: brain.GatewayReadinessReady,
		StartedAt: time.Now().Add(-2 * time.Millisecond),
	}); err != nil {
		t.Fatalf("admission emit refused: %v", err)
	}
	adm, ok := findHop(sink.Spans(), e2e.HopAdmission)
	if !ok || adm.Correlation.LaunchID == "" {
		t.Fatalf("admission span/launch_id missing: ok=%v span=%+v", ok, adm)
	}
	launchID := adm.Correlation.LaunchID

	// hop 4 — CLI (proc_id = real decimal pid)
	if err := daemon.EmitCLI(rec, daemon.CLIObservation{
		LaunchID: launchID, ProcID: procID, TaskID: task,
		CLIKind: "agent", ExitCodeClass: "exit_0", LatencyMs: 3, Outcome: "completed", ReasonCode: "ok",
	}); err != nil {
		t.Fatalf("cli emit refused: %v", err)
	}
	// hop 5 — route (gateway) — FIXTURE telemetry via the legacy provider-span emitter.
	start := time.Now().Add(-40 * time.Millisecond)
	if err := gateway.EmitProviderSpan(rec, gateway.ProviderSpanRecord{
		RequestID: reqID, PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol: brain.ProtocolAnthropicMessages, Telemetry: actualTelemetry(omniFixture),
		StartedAt: start, EndedAt: start.Add(30 * time.Millisecond), Outcome: "ok",
	}); err != nil {
		t.Fatalf("route emit refused: %v", err)
	}
	// hop 6 — persist (production result_id == raw task UUID)
	if err := service.EmitPersist(rec, service.PersistObservation{
		TaskID: task, ResultID: task, TerminalStatus: "completed",
		PersistLatencyMs: 2, ByteCount: 100, Outcome: "persisted",
	}); err != nil {
		t.Fatalf("persist emit refused: %v", err)
	}
	// hop 7 — delivery via the real realtime terminal seam.
	hub := realtime.NewHub()
	hub.SetDeliveryRecorder(rec)
	hub.BroadcastTerminalDelivery(realtime.ScopeWorkspace, "ws-anchor",
		[]byte(`{"type":"task:completed"}`),
		realtime.TerminalDeliveryMeta{TaskID: task, ChatSessionID: chat, Event: realtime.EventTaskCompleted},
		deliveryEventID)

	spans := sink.Spans()
	byHop := map[e2e.HopKind]int{}
	for _, s := range spans {
		byHop[s.Hop]++
	}
	if len(spans) != 7 {
		t.Fatalf("expected 7 hop spans, got %d (%v)", len(spans), byHop)
	}
	for _, h := range e2e.EmittingHops() {
		if byHop[h] != 1 {
			t.Fatalf("hop %q emitted %d times, want exactly 1", h, byHop[h])
		}
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
			t.Fatalf("id %q never present across real production seams", id)
		}
	}

	report := e2e.Assemble(spans)
	if !report.AllContinuous || len(report.Traces) != 1 || len(report.Orphans) != 0 || len(report.Anomalies) != 0 {
		t.Fatalf("assemble not continuous: continuous=%v traces=%d orphans=%d anomalies=%d",
			report.AllContinuous, len(report.Traces), len(report.Orphans), len(report.Anomalies))
	}
	if tr := report.Traces[0]; !tr.Continuous || len(tr.Present) != 7 || len(tr.Missing) != 0 || tr.TaskID != task {
		t.Fatalf("trace not continuous/complete: %+v", tr)
	}

	// Prove the t20 generated-ID gate ACCEPTS these production-derived ids: none
	// is of the rejected "<synthetic-prefix><task_id>" form.
	synthPrefixes := []string{"req-", "qmsg-", "sess-", "launch-", "proc-", "omni-", "result-", "delivery-"}
	for _, s := range spans {
		for _, id := range []string{
			s.Correlation.RequestID, s.Correlation.QueueMsgID, s.Correlation.SessionID,
			s.Correlation.LaunchID, s.Correlation.ProcID, s.Correlation.OmniRequestID,
			s.Correlation.ResultID, s.Correlation.DeliveryID,
		} {
			if id == "" {
				continue
			}
			for _, p := range synthPrefixes {
				if strings.HasPrefix(id, p) && strings.HasSuffix(id, task) {
					t.Fatalf("production-derived id %q trips the t20 generated-ID gate (<prefix><task_id>)", id)
				}
			}
		}
	}
}
