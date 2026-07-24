package daemon

// boot_recorder_wiring_test.go — T20 RED->target boot/constructor assertions for
// the daemon-process end-to-end observability recorder wiring. Test-only; asserts
// the TARGET from .deploy-control/p0/evidence/T20-production-wiring-map.md. The
// genuine RED assertions are gated behind T20_BOOT_TARGET=1 so the shared package
// build stays green for other lanes; running with the flag demonstrates the
// missing seams (RED), and the assertions flip to GREEN once wiring lands.

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func bootGatewayDaemonForTest(t *testing.T) *Daemon {
	t.Helper()
	cfg := Config{
		ServerBaseURL:  "ws://127.0.0.1:1/ws",
		WorkspacesRoot: t.TempDir(),
		AgentBrain:     syntheticAgentBrainConfig(t, "http://127.0.0.1:20128"),
	}
	return NewWithAgentBrainDependencies(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), AgentBrainDependencies{})
}

// TestBootDaemonInstallsSharedAdmissionAndCLIRecorder: the daemon must install ONE
// shared metadata-only recorder threaded into BOTH the admission hop
// (agentBrainOBS, daemon.go:283) and the CLI hop (d.cliObs, daemon.go:159). Today
// only admission is wired; d.cliObs is nil (CLI hop no-op, EmitCLI at daemon.go:3778).
func TestBootDaemonInstallsSharedAdmissionAndCLIRecorder(t *testing.T) {
	d := bootGatewayDaemonForTest(t)
	if d.agentBrainOBS == nil {
		t.Fatal("precondition: gateway-required daemon must install the admission observer (agentBrainOBS)")
	}
	if d.cliObs == nil {
		t.Fatal("MISSING SEAM: newDaemon did not install the CLI-hop recorder (d.cliObs == nil); admission and CLI must share one process recorder")
	}
}

// TestBootDaemonOwnsClosable0600ExportFileAndOTLPLifecycle asserts the daemon
// now owns a closable 0600 JSONL export-file handle (d.spanExportFile, closed on
// Run exit) when the export env is configured. The loopback OTLP receiver
// lifecycle is bound in Run (fail-closed) and shut down on Run exit — see
// TestRunEarlyReturnClosesSpanExportFile for the Run-path shutdown/close.
func TestBootDaemonOwnsClosable0600ExportFileAndOTLPLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon-spans.jsonl")
	t.Setenv("AGENT_BRAIN_E2E_EXPORT_FILE", path)
	d := bootGatewayDaemonForTest(t)
	if d.spanExportFile == nil {
		t.Fatal("MISSING SEAM: daemon does not own a closable 0600 export-file handle")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("export file stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export file mode=%o, want 0600", info.Mode().Perm())
	}
	if err := d.spanExportFile.Close(); err != nil {
		t.Fatalf("export file must be closable: %v", err)
	}
}
