package daemon

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// A pre-existing export file with a broader mode must be tightened to 0600 on
// open (O_CREATE mode does not apply to an existing file).
func TestSpanExportFileTightenedTo0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	t.Setenv("AGENT_BRAIN_E2E_EXPORT_FILE", path)
	_, f := daemonAdmissionSink(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if f == nil {
		t.Fatal("expected an owned export file")
	}
	defer f.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export file mode=%o, want 0600", info.Mode().Perm())
	}
}

// An early Run return (health-port conflict during preflight) must still close
// the process-owned span export fd — the close defer is registered before the
// first possible return.
func TestRunEarlyReturnClosesSpanExportFile(t *testing.T) {
	// Occupy a port so listenHealth fails immediately.
	occupier, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer occupier.Close()
	port := occupier.Addr().(*net.TCPAddr).Port

	f, err := os.CreateTemp(t.TempDir(), "spans-*.jsonl")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	d := &Daemon{
		cfg:            Config{HealthPort: port},
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		spanExportFile: f,
	}
	if runErr := d.Run(context.Background()); runErr == nil {
		t.Fatal("expected health-port conflict error from Run")
	}
	// The fd must be closed: a write after close returns an error.
	if _, werr := f.Write([]byte("x")); werr == nil {
		t.Fatal("spanExportFile must be closed on early Run return (fd leak)")
	}
}
