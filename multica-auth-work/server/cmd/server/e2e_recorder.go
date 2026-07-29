package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/service"
)

// newServerSpanRecorder builds the SERVER-process e2e recorder. It writes to a
// DEDICATED 0600 JSONL export file (distinct env/path from the daemon's) when
// AGENT_BRAIN_E2E_SERVER_EXPORT_FILE is set; otherwise a no-op recorder.
//
// It is FAIL-CLOSED: when the export file is CONFIGURED but cannot be opened, or
// its mode cannot be tightened to 0600, it returns an error (and never a
// silent discard recorder) so startup can fail — sensitive metadata is never
// written to an unopened or broader-mode file. The returned closeFn owns its
// OWN sync.Once, so independent recorder instances each close their own file
// (no shared/global Once that a first caller consumes).
func newServerSpanRecorder() (rec *e2e.Recorder, closeFn func(), err error) {
	path := strings.TrimSpace(os.Getenv("AGENT_BRAIN_E2E_SERVER_EXPORT_FILE"))
	if path == "" {
		// Telemetry entirely absent: no-op recorder, no file, no-op close.
		return e2e.NewRecorder(nil), func() {}, nil
	}
	f, ferr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if ferr != nil {
		return nil, func() {}, fmt.Errorf("server e2e span export open failed: %w", ferr)
	}
	// O_CREATE mode does not tighten an existing broader-mode file. On chmod
	// failure, close immediately and fail closed — never write to a file whose
	// mode we could not restrict.
	if chmodErr := f.Chmod(0o600); chmodErr != nil {
		_ = f.Close()
		return nil, func() {}, fmt.Errorf("server e2e span export chmod failed: %w", chmodErr)
	}
	var once sync.Once
	closeFn = func() { once.Do(func() { _ = f.Close() }) }
	return e2e.NewRecorder(e2e.NewJSONLSink(f)), closeFn, nil
}

// serverObservability holds the installed SERVER-process e2e recorder and the
// closer that owns its 0600 JSONL export file. closeFn is idempotent (owns its
// own sync.Once) and must be invoked on every process exit path.
type serverObservability struct {
	recorder *e2e.Recorder
	closeFn  func()
}

// installServerObservabilityWith installs an already-built recorder on every
// server-side observability hop seam:
//   - middleware ingress span         (hop 1) via middleware.SetIngressRecorder
//   - each supplied TaskService.Obs   (hops 2 queue / 6 persist) — HTTP + sweeper
//   - the realtime Hub delivery span  (hop 7) via hub.SetDeliveryRecorder
//
// A nil hub and nil TaskService entries are skipped, so the seam is unit-testable
// in isolation without constructing the whole process. Passing rec built over an
// e2e.MemorySink lets a test assert emissions on each seam.
func installServerObservabilityWith(rec *e2e.Recorder, hub *realtime.Hub, taskServices ...*service.TaskService) {
	for _, ts := range taskServices {
		if ts != nil {
			ts.Obs = rec
		}
	}
	if hub != nil {
		hub.SetDeliveryRecorder(rec)
	}
	middleware.SetIngressRecorder(rec)
}

// installServerObservability builds the SERVER e2e recorder (dedicated 0600 JSONL
// export when AGENT_BRAIN_E2E_SERVER_EXPORT_FILE is set; no-op otherwise) and
// installs it on all server-side hop seams via installServerObservabilityWith.
// It is fail-closed: a configured-but-unopenable/untightenable export file
// returns an error so startup can abort. The returned serverObservability owns
// the closable export file; callers must invoke closeFn on every exit path.
func installServerObservability(hub *realtime.Hub, taskServices ...*service.TaskService) (*serverObservability, error) {
	rec, closeFn, err := newServerSpanRecorder()
	if err != nil {
		return nil, err
	}
	installServerObservabilityWith(rec, hub, taskServices...)
	return &serverObservability{recorder: rec, closeFn: closeFn}, nil
}
