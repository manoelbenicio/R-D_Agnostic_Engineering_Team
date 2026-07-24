package main

// boot_recorder_wiring_test.go — real, ungated assertions that the testable
// server observability wiring helper installs a non-nil recorder on EVERY
// server-side hop seam and owns a closable 0600 JSONL export file. No t.Skip,
// no env gate: these exercise installServerObservability / *With directly with
// an in-memory recorder, so they need no database and prove the exact seams
// main() wires at boot (see e2e_recorder.go).
//
// NOTE: package `main` tests are only executed when the package TestMain
// (integration_test.go) has a reachable database; that DB gate is pre-existing
// and not added here. The database-free runnable proof of the delivery/terminal
// seam lives in internal/realtime (redis_relay_terminal_test.go, obs_delivery_test.go)
// and the production-anchor Assemble proof in
// internal/daemon/observability/e2ewiring — both run without a database.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
)

func bootSpansHaveHop(spans []e2e.Span, hop e2e.HopKind) bool {
	for _, s := range spans {
		if s.Hop == hop {
			return true
		}
	}
	return false
}

// TestInstallServerObservabilityWiresAllServerHops proves the boot wiring helper
// installs the recorder on TaskService.Obs (hops 2/6), the realtime Hub delivery
// recorder (hop 7), and middleware ingress (hop 1) — asserted behaviorally by
// driving each real seam and observing the span in an in-memory sink.
func TestInstallServerObservabilityWiresAllServerHops(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	hub := realtime.NewHub()
	ts := &service.TaskService{}

	installServerObservabilityWith(rec, hub, ts, nil)
	t.Cleanup(func() { middleware.SetIngressRecorder(nil) })

	// hops 2/6 — TaskService.Obs installed.
	if ts.Obs != rec {
		t.Fatalf("TaskService.Obs not wired to the server recorder")
	}

	// hop 7 — driving the real realtime Hub terminal-delivery seam records a
	// HopDelivery span through the installed recorder (proves hub.SetDeliveryRecorder).
	hub.BroadcastTerminalDelivery(realtime.ScopeWorkspace, "ws-boot",
		[]byte(`{"type":"task:completed"}`),
		realtime.TerminalDeliveryMeta{TaskID: "task-boot", ChatSessionID: "chat-boot", Event: realtime.EventTaskCompleted},
		"evt-boot")
	if !bootSpansHaveHop(sink.Spans(), e2e.HopDelivery) {
		t.Fatalf("hub delivery recorder not wired: no HopDelivery span recorded")
	}

	// hop 1 — driving the real RequestLogger handler records a HopIngress span
	// through the installed recorder (proves middleware.SetIngressRecorder).
	handler := middleware.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleware.SetIngressTaskID(r.Context(), "task-boot")
		w.WriteHeader(http.StatusAccepted)
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/tasks", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-boot"))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !bootSpansHaveHop(sink.Spans(), e2e.HopIngress) {
		t.Fatalf("ingress recorder not wired: no HopIngress span recorded")
	}
}

// TestInstallServerObservabilityOwnsClosable0600ExportFile proves the production
// wrapper builds a recorder over a DEDICATED 0600 JSONL export file and returns
// an idempotent closer that owns it.
func TestInstallServerObservabilityOwnsClosable0600ExportFile(t *testing.T) {
	path := t.TempDir() + "/server-spans.jsonl"
	t.Setenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE", path)
	t.Cleanup(func() { middleware.SetIngressRecorder(nil) })

	obs, err := installServerObservability(realtime.NewHub(), &service.TaskService{})
	if err != nil {
		t.Fatalf("installServerObservability: %v", err)
	}
	if obs == nil || obs.recorder == nil {
		t.Fatalf("nil observability/recorder")
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("export file not created: %v", statErr)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("export file mode = %v, want 0600", perm)
	}
	obs.closeFn()
	obs.closeFn() // idempotent: must not panic
}
