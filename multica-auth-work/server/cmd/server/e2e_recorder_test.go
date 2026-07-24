package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func ingressSpan() *e2e.Span {
	s := e2e.NewSpan(e2e.HopIngress, e2e.Correlation{RequestID: "req-x", TaskID: "task-x"})
	s.StartedAt = time.Now().UTC()
	s.WithOutcome("received", "")
	s.Finish()
	return s
}

func TestServerSpanRecorderNoEnvIsNoop(t *testing.T) {
	os.Unsetenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE")
	rec, closeFn, err := newServerSpanRecorder()
	if err != nil || rec == nil || closeFn == nil {
		t.Fatalf("no-env must yield no-op recorder + nil err; got rec=%v err=%v", rec, err)
	}
	closeFn() // must be safe
}

func TestServerSpanRecorderTightensAndClosesIndependently(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "s1.jsonl")
	if err := os.WriteFile(p1, nil, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Setenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE", p1)
	r1, c1, e1 := newServerSpanRecorder()
	if e1 != nil {
		t.Fatalf("open1: %v", e1)
	}
	if info, _ := os.Stat(p1); info.Mode().Perm() != 0o600 {
		t.Fatalf("preexisting file not tightened: %o", info.Mode().Perm())
	}

	p2 := filepath.Join(dir, "s2.jsonl")
	t.Setenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE", p2)
	r2, c2, e2 := newServerSpanRecorder()
	if e2 != nil {
		t.Fatalf("open2: %v", e2)
	}

	// Both independent instances must actually close their OWN file — a global
	// sync.Once would leave the second file open. After close, Emit fails.
	if err := r1.Emit(ingressSpan()); err != nil {
		t.Fatalf("r1 emit before close should succeed: %v", err)
	}
	c1()
	c2()
	if err := r1.Emit(ingressSpan()); err == nil {
		t.Fatal("r1 file must be closed after c1()")
	}
	if err := r2.Emit(ingressSpan()); err == nil {
		t.Fatal("r2 file must be closed after c2() (per-instance Once, not global)")
	}
}

func TestServerSpanRecorderConfiguredOpenFailureNotSilent(t *testing.T) {
	// Pointing the export at a directory makes O_WRONLY open fail -> must error,
	// never a silent discard recorder.
	t.Setenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE", t.TempDir())
	rec, _, err := newServerSpanRecorder()
	if err == nil || rec != nil {
		t.Fatalf("configured open failure must return (nil, _, error); got rec=%v err=%v", rec, err)
	}
}
