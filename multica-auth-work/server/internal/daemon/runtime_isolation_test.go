package daemon

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/rotation"
)

// TestRuntimeSetWatcherFanOut pins the multi-subscriber contract: every
// subscribed channel must receive a nudge on each notify, and unsubscribed
// channels must not.
func TestRuntimeSetWatcherFanOut(t *testing.T) {
	t.Parallel()

	w := newRuntimeSetWatcher()
	chA, unsubA := w.Subscribe()
	chB, unsubB := w.Subscribe()
	defer unsubA()
	defer unsubB()

	w.notify()
	for _, ch := range []<-chan struct{}{chA, chB} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("expected each subscriber to receive a nudge")
		}
	}

	// Coalescing: a second notify before the subscriber drains must not
	// block, and the subscriber should still see exactly one pending nudge.
	w.notify()
	w.notify()
	select {
	case <-chA:
	default:
		t.Fatal("expected coalesced nudge to be pending")
	}
	select {
	case <-chA:
		t.Fatal("expected only one coalesced nudge to be queued")
	default:
	}

	// Unsubscribed channels must not get nudges. Drain any in-flight nudge
	// on chB first so we observe only post-unsubscribe behaviour.
	select {
	case <-chB:
	default:
	}
	unsubB()
	w.notify()
	select {
	case <-chB:
		t.Fatal("unsubscribed channel must not receive a nudge")
	case <-time.After(50 * time.Millisecond):
	}
}

// TestRunRuntimePollerIsolatesSlowRuntime is the regression test for
// MUL-1744's main symptom: a slow ClaimTask on one runtime must not delay
// claims on any other runtime. The pre-refactor pollLoop's serial round-
// robin made every runtime wait behind the slow one's HTTP roundtrip.
//
// MaxConcurrentTasks=4 leaves headroom so each runtime gets its own slot.
// The poller does acquire a slot before claiming (see runRuntimePoller for
// why), so this test deliberately uses a capacity that fits both runtimes
// concurrently — that's the case where slot-before-claim still gives full
// isolation.
func TestRunRuntimePollerIsolatesSlowRuntime(t *testing.T) {
	t.Parallel()

	var fastClaims atomic.Int64
	slowEntered := make(chan struct{}, 1)
	releaseSlow := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/runtimes/runtime-slow/tasks/claim"):
			select {
			case slowEntered <- struct{}{}:
			default:
			}
			select {
			case <-releaseSlow:
			case <-r.Context().Done():
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"task":null}`))
		case strings.HasSuffix(path, "/runtimes/runtime-fast/tasks/claim"):
			fastClaims.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"task":null}`))
		default:
			http.Error(w, "unexpected path: "+path, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	defer close(releaseSlow)

	d := New(Config{
		ServerBaseURL:      srv.URL,
		HeartbeatInterval:  time.Hour, // disable WS-suppression effects
		PollInterval:       50 * time.Millisecond,
		MaxConcurrentTasks: 4,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := newTaskSlotSemaphore(d.cfg.MaxConcurrentTasks)
	var taskWG sync.WaitGroup

	slowCtx, slowCancel := context.WithCancel(ctx)
	defer slowCancel()
	go d.runRuntimePoller(slowCtx, ctx, "runtime-slow", sem, make(chan struct{}, 1), &taskWG)

	fastCtx, fastCancel := context.WithCancel(ctx)
	defer fastCancel()
	go d.runRuntimePoller(fastCtx, ctx, "runtime-fast", sem, make(chan struct{}, 1), &taskWG)

	// Wait for the slow handler to actually enter (so we know its claim is
	// in flight) before checking fast-runtime progress.
	select {
	case <-slowEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("slow runtime claim never entered server handler")
	}

	// Within a short window, the fast runtime should issue several claims.
	// Pre-isolation, it would be stuck behind the still-blocked slow claim.
	deadline := time.After(2 * time.Second)
	for fastClaims.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("fast runtime made only %d claims while slow runtime blocked; expected ≥3", fastClaims.Load())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// TestRunRuntimePollerSkipsClaimWhenAtCapacity pins the slot-before-claim
// invariant: when no execution slots are available, the poller must NOT
// call ClaimTask. Pre-claiming and then waiting for a slot would let the
// task pile up in server-side `dispatched` state and race the 5-minute
// `dispatchTimeoutSeconds` sweeper, recreating the exact failure mode this
// issue is fixing.
func TestRunRuntimePollerSkipsClaimWhenAtCapacity(t *testing.T) {
	t.Parallel()

	var claimAttempts atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tasks/claim") {
			claimAttempts.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"task":null}`))
	}))
	defer srv.Close()

	d := New(Config{
		ServerBaseURL:      srv.URL,
		HeartbeatInterval:  time.Hour,
		PollInterval:       20 * time.Millisecond,
		MaxConcurrentTasks: 1,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Drain the only slot to simulate a long-running handleTask occupying
	// capacity. The poller must observe an empty sem and skip ClaimTask.
	sem := newTaskSlotSemaphore(d.cfg.MaxConcurrentTasks)
	<-sem // hold it: never returned during this test

	var taskWG sync.WaitGroup
	go d.runRuntimePoller(ctx, ctx, "runtime-busy", sem, make(chan struct{}, 1), &taskWG)

	// Give the poller several PollInterval ticks to race against the empty
	// sem. With slot-before-claim it must report zero claim attempts; the
	// older "claim first" path would have hammered ClaimTask each tick.
	time.Sleep(200 * time.Millisecond)

	if got := claimAttempts.Load(); got != 0 {
		t.Fatalf("poller called ClaimTask %d times while at capacity; want 0 — pre-claiming risks server-side dispatch_timeout", got)
	}
}

func TestRunRuntimePollerClaimsWhenSlotBecomesAvailable(t *testing.T) {
	t.Parallel()

	var claimAttempts atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tasks/claim") {
			claimAttempts.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"task":null}`))
	}))
	defer srv.Close()

	d := New(Config{
		ServerBaseURL:      srv.URL,
		HeartbeatInterval:  time.Hour,
		PollInterval:       time.Hour,
		MaxConcurrentTasks: 1,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := newTaskSlotSemaphore(d.cfg.MaxConcurrentTasks)
	slot := <-sem

	var taskWG sync.WaitGroup
	wakeup := make(chan struct{}, 1)
	go d.runRuntimePoller(ctx, ctx, "runtime-waiting", sem, wakeup, &taskWG)
	wakeup <- struct{}{}

	time.Sleep(100 * time.Millisecond)
	if got := claimAttempts.Load(); got != 0 {
		t.Fatalf("poller claimed before a slot was available; got %d claims", got)
	}

	sem <- slot

	deadline := time.After(2 * time.Second)
	for claimAttempts.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("poller did not claim after a slot became available")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestPollLoopShutdownWaitsForPollersBeforeTaskWG is a race-detector
// regression for the WaitGroup misuse GPT-Boy flagged: pollLoop must not
// call taskWG.Wait while a poller goroutine could still execute
// taskWG.Add(1). The supervisor uses a separate pollerWG that this test
// implicitly exercises by running shutdown concurrently with a task being
// dispatched.
func TestPollLoopShutdownWaitsForPollersBeforeTaskWG(t *testing.T) {
	t.Parallel()

	taskID := "00000000-0000-0000-0000-000000000001"
	releaseClaim := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(path, "/tasks/claim"):
			// Block until the test releases. When released, return a real task
			// so the poller proceeds into the slot/dispatch path — exactly the
			// window where taskWG.Add(1) races with shutdown's taskWG.Wait.
			select {
			case <-releaseClaim:
			case <-r.Context().Done():
				w.Write([]byte(`{"task":null}`))
				return
			}
			w.Write([]byte(`{"task":{"id":"` + taskID + `","runtime_id":"runtime-1","issue_id":"issue-1","agent":{"name":"test"}}}`))
		case strings.HasSuffix(path, "/start"):
			w.Write([]byte(`{}`))
		case strings.HasSuffix(path, "/fail"):
			w.Write([]byte(`{}`))
		case strings.HasSuffix(path, "/complete"):
			w.Write([]byte(`{}`))
		case strings.HasSuffix(path, "/progress"):
			w.Write([]byte(`{}`))
		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	d := New(Config{
		ServerBaseURL:      srv.URL,
		HeartbeatInterval:  time.Hour,
		PollInterval:       50 * time.Millisecond,
		MaxConcurrentTasks: 1,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))
	d.workspaces["ws-1"] = &workspaceState{
		workspaceID: "ws-1",
		runtimeIDs:  []string{"runtime-1"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	pollDone := make(chan error, 1)
	go func() {
		pollDone <- d.pollLoop(ctx, nil)
	}()

	// Let the poller enter ClaimTask, then trigger shutdown right as the
	// claim is about to return a task. The race is the window between
	// ClaimTask returning and taskWG.Add(1) executing.
	time.Sleep(100 * time.Millisecond)
	close(releaseClaim)
	cancel()

	select {
	case <-pollDone:
	case <-time.After(5 * time.Second):
		t.Fatal("pollLoop did not return within shutdown deadline")
	}
}

func TestPollLoopTargetsRuntimeWakeup(t *testing.T) {
	t.Parallel()

	var fastClaims atomic.Int64
	var slowClaims atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/runtimes/runtime-fast/tasks/claim"):
			fastClaims.Add(1)
		case strings.HasSuffix(path, "/runtimes/runtime-slow/tasks/claim"):
			slowClaims.Add(1)
		default:
			http.Error(w, "unexpected path: "+path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"task":null}`))
	}))
	defer srv.Close()

	d := New(Config{
		ServerBaseURL:      srv.URL,
		HeartbeatInterval:  time.Hour,
		PollInterval:       time.Hour,
		MaxConcurrentTasks: 4,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))
	d.workspaces["ws-1"] = &workspaceState{
		workspaceID: "ws-1",
		runtimeIDs:  []string{"runtime-fast", "runtime-slow"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	taskWakeups := make(chan taskWakeup, 1)
	pollDone := make(chan error, 1)
	go func() {
		pollDone <- d.pollLoop(ctx, taskWakeups)
	}()

	taskWakeups <- taskWakeup{}

	deadline := time.After(2 * time.Second)
	for fastClaims.Load() < 1 || slowClaims.Load() < 1 {
		select {
		case <-deadline:
			t.Fatalf("initial poll did not claim both runtimes; fast=%d slow=%d", fastClaims.Load(), slowClaims.Load())
		case <-time.After(10 * time.Millisecond):
		}
	}

	fastClaims.Store(0)
	slowClaims.Store(0)
	taskWakeups <- taskWakeup{runtimeID: "runtime-fast"}

	deadline = time.After(2 * time.Second)
	for fastClaims.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("targeted wakeup did not wake runtime-fast")
		case <-time.After(10 * time.Millisecond):
		}
	}

	time.Sleep(100 * time.Millisecond)
	if got := slowClaims.Load(); got != 0 {
		t.Fatalf("targeted wakeup woke runtime-slow %d times; want 0", got)
	}

	cancel()
	select {
	case <-pollDone:
	case <-time.After(5 * time.Second):
		t.Fatal("pollLoop did not stop")
	}
}

// TestRunRuntimeHeartbeatIsolatesSlowRuntime is the heartbeat-side mirror of
// the poll-isolation test: a slow SendHeartbeat for one runtime must not
// block other runtimes' heartbeats.
func TestRunRuntimeHeartbeatIsolatesSlowRuntime(t *testing.T) {
	t.Parallel()

	var fastBeats atomic.Int64
	slowEntered := make(chan struct{}, 1)
	releaseSlow := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		payload := string(body[:n])
		switch {
		case strings.Contains(payload, `"runtime-slow"`):
			select {
			case slowEntered <- struct{}{}:
			default:
			}
			select {
			case <-releaseSlow:
			case <-r.Context().Done():
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		case strings.Contains(payload, `"runtime-fast"`):
			fastBeats.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		default:
			http.Error(w, "unexpected payload", http.StatusBadRequest)
		}
	}))
	defer srv.Close()
	defer close(releaseSlow)

	d := New(Config{
		ServerBaseURL:     srv.URL,
		HeartbeatInterval: 50 * time.Millisecond,
	}, slog.New(slog.NewTextHandler(noopWriter{}, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.runRuntimeHeartbeat(ctx, "runtime-slow")
	go d.runRuntimeHeartbeat(ctx, "runtime-fast")

	select {
	case <-slowEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("slow heartbeat never entered server handler")
	}

	deadline := time.After(2 * time.Second)
	for fastBeats.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("fast runtime sent only %d heartbeats while slow runtime blocked; expected ≥3", fastBeats.Load())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// noopWriter discards log output so the test runner doesn't get noisy.
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// ---------------------------------------------------------------------------
// Credential isolation per vendor — GLM52-CLINE1
//
// Extends the isolation suite to cover all six P0 vendors (Codex, Kiro,
// Antigravity, GLM, Cline, OpenCode) against the REAL PostgreSQL tables from
// migration 123 (accounts, credentials, assignments, rotation_events) — never
// the legacy rotation_* prefixed names.
//
// Per the central fix doc (FIX_ISOLAMENTO_CREDENCIAL_CENTRAL.md), each vendor
// must satisfy three acceptance criteria:
//  1. Two accounts of the same vendor coexist without overlap.
//  2. Fail-closed: without an account assignment the gate returns an error,
//     never an empty string the daemon would treat as "use shared credential".
//  3. No secret material appears in log output.
//
// The tests skip gracefully when DATABASE_URL is unset or Postgres is
// unreachable, mirroring the rotation package's testPool convention.
// ---------------------------------------------------------------------------

// allIsolationVendors is the complete P0 vendor matrix. Every entry must be
// covered by requiresCredentialIsolation in daemon.go.
var allIsolationVendors = []string{"codex", "kiro", "antigravity", "glm", "cline", "opencode"}

// isolationTestPool returns a pgxpool connected to the DATABASE_URL Postgres
// instance, ensuring the rotation schema (migration 123) exists first. The
// test is skipped when DATABASE_URL is empty or the server is unreachable.
func isolationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("credential isolation tests require DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("credential isolation tests require Postgres: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("credential isolation tests require Postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	isolationEnsureSchema(t, pool)
	return pool
}

// isolationEnsureSchema applies migration 123 (123_rotation.up.sql) when the
// real accounts table is missing. Uses the REAL table names — accounts,
// credentials, assignments, rotation_events — never the legacy rotation_*
// prefixed names.
func isolationEnsureSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.accounts') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("isolation: check accounts schema: %v", err)
	}
	if exists {
		return
	}
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "123_rotation.up.sql"))
	if err != nil {
		t.Fatalf("isolation: read 123_rotation.up.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("isolation: apply 123_rotation.up.sql: %v", err)
	}
}

// isolationSeedAccount inserts a row into the real accounts table, a
// corresponding credentials row (secret_ref is a KMS pointer, never the
// secret itself), and returns the account_id. The credential material lives
// on disk under homeDir (per-vendor layout); the DB stores only metadata.
func isolationSeedAccount(t *testing.T, pool *pgxpool.Pool, vendor, tenantID, homeDir, configDir string) string {
	t.Helper()
	accountID := uuid.NewString()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO accounts (account_id, vendor, tenant_id, priority, home_dir, config_dir, status, tokens_per_window, tokens_used)
		VALUES ($1, $2, $3, 0, $4, $5, 'available', 100000, 0)
	`, accountID, vendor, tenantID, homeDir, configDir); err != nil {
		t.Fatalf("isolation: seed account (%s): %v", vendor, err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO credentials (credential_id, account_id, vendor, secret_ref, format)
		VALUES ($1, $2, $3, $4, 'json')
	`, uuid.NewString(), accountID, vendor, "kms://"+vendor+"/"+accountID); err != nil {
		t.Fatalf("isolation: seed credential (%s): %v", vendor, err)
	}
	return accountID
}

// isolationAssign maps an agent to an account in the real assignments table.
func isolationAssign(t *testing.T, pool *pgxpool.Pool, agentID, accountID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO assignments (agent_id, account_id, assigned_at)
		VALUES ($1, $2, now())
		ON CONFLICT (agent_id) DO UPDATE SET account_id = EXCLUDED.account_id, assigned_at = EXCLUDED.assigned_at
	`, agentID, accountID); err != nil {
		t.Fatalf("isolation: assign: %v", err)
	}
}

// isolationCleanup removes only the rows this test created (scoped by tenant
// and agent IDs) so parallel subtests never clobber each other.
func isolationCleanup(t *testing.T, pool *pgxpool.Pool, tenantID string, agentIDs ...string) {
	t.Helper()
	ctx := context.Background()
	for _, agentID := range agentIDs {
		if _, err := pool.Exec(ctx, `DELETE FROM rotation_events WHERE agent_id = $1`, agentID); err != nil {
			t.Logf("isolation: cleanup rotation_events for %s: %v", agentID, err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM assignments WHERE agent_id = $1`, agentID); err != nil {
			t.Logf("isolation: cleanup assignments for %s: %v", agentID, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM credentials WHERE account_id IN (SELECT account_id FROM accounts WHERE tenant_id = $1)`, tenantID); err != nil {
		t.Logf("isolation: cleanup credentials for tenant %s: %v", tenantID, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM accounts WHERE tenant_id = $1`, tenantID); err != nil {
		t.Logf("isolation: cleanup accounts for tenant %s: %v", tenantID, err)
	}
}

// credentialMarker returns a unique, grep-able secret sentinel. If this string
// ever appears in a log capture, the "no secret in log" assertion fails.
func credentialMarker(vendor, label string) string {
	return "SECRET-" + strings.ToUpper(vendor) + "-" + label + "-" + uuid.NewString()[:8]
}

// setupVendorAccountHome creates a temp directory laid out as the vendor's
// per-account credential source, writes a credential file containing the
// secret marker, and returns the home dir path. This is the directory that
// lives in accounts.home_dir and is passed as CredentialAccountHome to
// execenv.Prepare.
//
// Vendor → native isolation lever (per FIX_ISOLAMENTO_CREDENCIAL_CENTRAL.md):
//   - codex       → CODEX_HOME, credential file auth.json
//   - kiro        → XDG_DATA_HOME, credential file kiro-cli/data.sqlite3
//   - antigravity → HOME, credential dir .gemini/antigravity-cli/
//   - cline       → CLINE_DATA_DIR, credential dir .cline/
//   - glm/opencode→ XDG_DATA_HOME + XDG_CONFIG_HOME, .local/share/opencode/
func setupVendorAccountHome(t *testing.T, vendor, marker string) string {
	t.Helper()
	home := t.TempDir()
	switch vendor {
	case "codex":
		writeFixture(t, filepath.Join(home, "auth.json"), []byte(`{"token":"`+marker+`","type":"oauth"}`))
	case "kiro":
		mkdirFixture(t, filepath.Join(home, "kiro-cli"), 0o700)
		writeFixture(t, filepath.Join(home, "kiro-cli", "data.sqlite3"), []byte(marker))
	case "antigravity":
		mkdirFixture(t, filepath.Join(home, ".gemini", "antigravity-cli"), 0o700)
		writeFixture(t, filepath.Join(home, ".gemini", "antigravity-cli", "token.json"), []byte(`{"access_token":"`+marker+`"}`))
	case "cline":
		mkdirFixture(t, filepath.Join(home, ".cline"), 0o700)
		writeFixture(t, filepath.Join(home, ".cline", "auth.json"), []byte(`{"token":"`+marker+`"}`))
	case "opencode", "glm":
		mkdirFixture(t, filepath.Join(home, ".local", "share", "opencode"), 0o700)
		writeFixture(t, filepath.Join(home, ".local", "share", "opencode", "auth.json"), []byte(`{"token":"`+marker+`"}`))
		mkdirFixture(t, filepath.Join(home, ".config", "opencode"), 0o700)
		writeFixture(t, filepath.Join(home, ".config", "opencode", "config.json"), []byte(`{"vendor":"`+vendor+`"}`))
	default:
		t.Fatalf("setupVendorAccountHome: unknown vendor %q", vendor)
	}
	return home
}

// vendorCredentialInEnv returns the path to the isolated credential file
// inside a prepared execenv.Environment for the given vendor, or "" when the
// vendor's isolated home was not populated.
func vendorCredentialInEnv(env *execenv.Environment, vendor string) string {
	switch vendor {
	case "codex":
		if env.CodexHome == "" {
			return ""
		}
		return filepath.Join(env.CodexHome, "auth.json")
	case "kiro":
		if env.KiroDataHome == "" {
			return ""
		}
		return filepath.Join(env.KiroDataHome, "kiro-cli", "data.sqlite3")
	case "antigravity":
		if env.AntigravityHome == "" {
			return ""
		}
		return filepath.Join(env.AntigravityHome, ".gemini", "antigravity-cli", "token.json")
	case "cline":
		if env.ClineDataDir == "" {
			return ""
		}
		return filepath.Join(env.ClineDataDir, "auth.json")
	case "opencode", "glm":
		if env.OpenCodeDataHome == "" {
			return ""
		}
		return filepath.Join(env.OpenCodeDataHome, "opencode", "auth.json")
	default:
		return ""
	}
}

// vendorIsolatedDirs returns the isolated home directories set on the env for
// the vendor. Used to assert two accounts produce non-overlapping dirs.
func vendorIsolatedDirs(env *execenv.Environment, vendor string) []string {
	switch vendor {
	case "codex":
		return nonEmptyDirs(env.CodexHome)
	case "kiro":
		return nonEmptyDirs(env.KiroDataHome)
	case "antigravity":
		return nonEmptyDirs(env.AntigravityHome)
	case "cline":
		return nonEmptyDirs(env.ClineDataDir, env.ClineSandboxDataDir)
	case "opencode", "glm":
		return nonEmptyDirs(env.OpenCodeDataHome, env.OpenCodeConfigHome)
	default:
		return nil
	}
}

func nonEmptyDirs(dirs ...string) []string {
	var out []string
	for _, d := range dirs {
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

// prepareIsolatedEnv calls execenv.Prepare with the given CredentialAccountHome
// for the vendor and returns the prepared environment. The caller must defer
// env.Cleanup(true).
func prepareIsolatedEnv(t *testing.T, vendor, accountHome string, logger *slog.Logger) *execenv.Environment {
	t.Helper()
	env, err := execenv.Prepare(execenv.PrepareParams{
		WorkspacesRoot:        t.TempDir(),
		WorkspaceID:           "ws-iso-" + vendor,
		TaskID:                uuid.NewString(),
		AgentName:             "Isolation Test Agent",
		Provider:              vendor,
		CodexVersion:          "0.121.0",
		CredentialAccountHome: accountHome,
		Task:                  execenv.TaskContextForEnv{IssueID: "iso-" + vendor, AgentID: uuid.NewString()},
	}, logger)
	if err != nil {
		t.Fatalf("execenv.Prepare(%s): %v", vendor, err)
	}
	return env
}

func writeFixture(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func mkdirFixture(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("mkdir fixture %s: %v", path, err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatalf("chmod fixture %s: %v", path, err)
	}
}

// (captureLogger is defined in token_renewal_test.go and reused here.)

// TestCredentialIsolationPerVendor is the P0 acceptance gate for credential
// isolation across all six vendors. It uses the REAL PostgreSQL tables from
// migration 123 (accounts, credentials, assignments, rotation_events) — never
// the legacy rotation_* names — and the per-vendor execenv isolation.
func TestCredentialIsolationPerVendor(t *testing.T) {
	// Ensure the schema exists before any parallel subtest starts. This call
	// also skips the entire test when Postgres is unavailable.
	pool := isolationTestPool(t)

	for _, vendor := range allIsolationVendors {
		vendor := vendor
		t.Run(vendor, func(t *testing.T) {
			t.Parallel()

			t.Run("two_accounts_coexist_without_overlap", func(t *testing.T) {
				testTwoAccountsCoexist(t, pool, vendor)
			})
			t.Run("fail_closed_no_assignment", func(t *testing.T) {
				testFailClosedNoAssignment(t, pool, vendor)
			})
			t.Run("no_secret_in_log", func(t *testing.T) {
				testNoSecretInLog(t, pool, vendor)
			})
		})
	}
}

func TestCredentialIsolationVendorMatrixCoversExactlySixP0Vendors(t *testing.T) {
	t.Parallel()

	want := []string{"codex", "kiro", "antigravity", "glm", "cline", "opencode"}
	if strings.Join(allIsolationVendors, ",") != strings.Join(want, ",") {
		t.Fatalf("credential isolation vendors = %v, want exactly %v", allIsolationVendors, want)
	}
	for _, vendor := range want {
		if !requiresCredentialIsolation(vendor) {
			t.Fatalf("requiresCredentialIsolation(%q) = false", vendor)
		}
		if _, err := isolatedCredentialEnv(vendor, "", &execenv.Environment{}); err == nil {
			t.Fatalf("isolatedCredentialEnv(%q) accepted an empty account home", vendor)
		}
		if _, err := isolatedCredentialEnv(vendor, t.TempDir(), &execenv.Environment{}); err == nil {
			t.Fatalf("isolatedCredentialEnv(%q) accepted a missing provider-native env", vendor)
		}
	}
}

// testTwoAccountsCoexist verifies that two accounts of the same vendor, backed
// by real PostgreSQL rows, resolve to non-overlapping home dirs and produce
// isolated exec environments whose credentials do not cross-contaminate.
func testTwoAccountsCoexist(t *testing.T, pool *pgxpool.Pool, vendor string) {
	tenantID := uuid.NewString()
	agentA := uuid.NewString()
	agentB := uuid.NewString()
	t.Cleanup(func() { isolationCleanup(t, pool, tenantID, agentA, agentB) })

	markerA := credentialMarker(vendor, "A")
	markerB := credentialMarker(vendor, "B")
	homeA := setupVendorAccountHome(t, vendor, markerA)
	homeB := setupVendorAccountHome(t, vendor, markerB)

	accountA := isolationSeedAccount(t, pool, vendor, tenantID, homeA, homeA)
	accountB := isolationSeedAccount(t, pool, vendor, tenantID, homeB, homeB)
	isolationAssign(t, pool, agentA, accountA)
	isolationAssign(t, pool, agentB, accountB)

	d := &Daemon{
		rotationStore: rotation.NewPGStore(pool),
		logger:        slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	}
	ctx := context.Background()
	taskLog := slog.New(slog.NewTextHandler(noopWriter{}, nil))

	// The gate must resolve each agent to its OWN account's home dir.
	resolvedA, err := d.credentialAccountHomeForTask(ctx, Task{AgentID: agentA}, vendor, taskLog)
	if err != nil {
		t.Fatalf("credentialAccountHomeForTask(agentA, %s): %v", vendor, err)
	}
	if resolvedA != homeA {
		t.Fatalf("agentA resolved to %q, want its own account home %q", resolvedA, homeA)
	}
	resolvedB, err := d.credentialAccountHomeForTask(ctx, Task{AgentID: agentB}, vendor, taskLog)
	if err != nil {
		t.Fatalf("credentialAccountHomeForTask(agentB, %s): %v", vendor, err)
	}
	if resolvedB != homeB {
		t.Fatalf("agentB resolved to %q, want its own account home %q", resolvedB, homeB)
	}
	if resolvedA == resolvedB {
		t.Fatalf("two accounts of vendor %s resolved to the same home dir %q — no isolation", vendor, resolvedA)
	}

	// Prepare two isolated execenvs from the two account homes and confirm
	// the on-disk credentials are isolated (no cross-contamination).
	envA := prepareIsolatedEnv(t, vendor, resolvedA, slog.New(slog.NewTextHandler(noopWriter{}, nil)))
	defer envA.Cleanup(true)
	envB := prepareIsolatedEnv(t, vendor, resolvedB, slog.New(slog.NewTextHandler(noopWriter{}, nil)))
	defer envB.Cleanup(true)

	for label, prepared := range map[string]*execenv.Environment{"A": envA, "B": envB} {
		isolatedEnv, err := isolatedCredentialEnv(vendor, map[string]string{"A": resolvedA, "B": resolvedB}[label], prepared)
		if err != nil {
			t.Fatalf("vendor %s account %s: isolated credential env: %v", vendor, label, err)
		}
		for _, key := range requiredCredentialEnvKeys(vendor) {
			if strings.TrimSpace(isolatedEnv[key]) == "" {
				t.Fatalf("vendor %s account %s: required env %s is empty", vendor, label, key)
			}
		}
	}

	dirsA := vendorIsolatedDirs(envA, vendor)
	dirsB := vendorIsolatedDirs(envB, vendor)
	if len(dirsA) == 0 {
		t.Fatalf("vendor %s: execenv.Prepare did not set any isolated home dir for account A", vendor)
	}
	if len(dirsB) == 0 {
		t.Fatalf("vendor %s: execenv.Prepare did not set any isolated home dir for account B", vendor)
	}
	for _, a := range dirsA {
		for _, b := range dirsB {
			if a == b {
				t.Fatalf("vendor %s: isolated dirs overlap (%q) — accounts share a credential path", vendor, a)
			}
		}
	}

	credA := vendorCredentialInEnv(envA, vendor)
	credB := vendorCredentialInEnv(envB, vendor)
	if credA == "" {
		t.Fatalf("vendor %s: credential file not found in account A's isolated env", vendor)
	}
	if credB == "" {
		t.Fatalf("vendor %s: credential file not found in account B's isolated env", vendor)
	}
	for label, credentialPath := range map[string]string{"A": credA, "B": credB} {
		info, err := os.Lstat(credentialPath)
		if err != nil {
			t.Fatalf("vendor %s: lstat account %s credential: %v", vendor, label, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			t.Fatalf("vendor %s: account %s credential mode = %s; want a regular copied file, never a symlink", vendor, label, info.Mode())
		}
	}

	contentA, err := os.ReadFile(credA)
	if err != nil {
		t.Fatalf("vendor %s: read account A credential %s: %v", vendor, credA, err)
	}
	contentB, err := os.ReadFile(credB)
	if err != nil {
		t.Fatalf("vendor %s: read account B credential %s: %v", vendor, credB, err)
	}
	if !strings.Contains(string(contentA), markerA) {
		t.Fatalf("vendor %s: account A credential does not contain A's marker; got %q", vendor, string(contentA))
	}
	if strings.Contains(string(contentA), markerB) {
		t.Fatalf("vendor %s: account A credential contaminated by B's marker", vendor)
	}
	if !strings.Contains(string(contentB), markerB) {
		t.Fatalf("vendor %s: account B credential does not contain B's marker; got %q", vendor, string(contentB))
	}
	if strings.Contains(string(contentB), markerA) {
		t.Fatalf("vendor %s: account B credential contaminated by A's marker", vendor)
	}
}

// testFailClosedNoAssignment verifies the fail-closed contract: when an agent
// has no row in the real assignments table, credentialAccountHomeForTask
// returns an ERROR — never an empty string the daemon would interpret as "use
// the shared credential". A nil rotation store must also fail closed.
func testFailClosedNoAssignment(t *testing.T, pool *pgxpool.Pool, vendor string) {
	d := &Daemon{
		rotationStore: rotation.NewPGStore(pool),
		logger:        slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	}
	ctx := context.Background()
	taskLog := slog.New(slog.NewTextHandler(noopWriter{}, nil))
	unassignedAgent := uuid.NewString() // no row in assignments for this agent

	home, err := d.credentialAccountHomeForTask(ctx, Task{AgentID: unassignedAgent}, vendor, taskLog)
	if err == nil {
		t.Fatalf("vendor %s: gate returned nil error for unassigned agent — fail-closed violated (home=%q)", vendor, home)
	}
	if home != "" {
		t.Fatalf("vendor %s: fail-closed returned non-empty home %q — daemon would use it as a shared credential", vendor, home)
	}
	if !strings.Contains(err.Error(), "no account assignment") {
		t.Fatalf("vendor %s: fail-closed error = %q, want an error mentioning no account assignment", vendor, err.Error())
	}

	// A nil rotation store must also fail closed, never silently fall back.
	dNil := &Daemon{
		rotationStore: nil,
		logger:        slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	}
	home2, err2 := dNil.credentialAccountHomeForTask(ctx, Task{AgentID: uuid.NewString()}, vendor, taskLog)
	if err2 == nil {
		t.Fatalf("vendor %s: nil-store gate returned nil error — fail-closed violated (home=%q)", vendor, home2)
	}
	if home2 != "" {
		t.Fatalf("vendor %s: nil-store gate returned non-empty home %q — shared credential leak", vendor, home2)
	}
}

// testNoSecretInLog verifies that neither the daemon's credential gate nor
// execenv.Prepare writes the credential's secret material to the log stream.
// The secret marker is confirmed present on disk first, so its absence from
// logs is meaningful rather than a false pass from a missing fixture.
func testNoSecretInLog(t *testing.T, pool *pgxpool.Pool, vendor string) {
	tenantID := uuid.NewString()
	agentID := uuid.NewString()
	t.Cleanup(func() { isolationCleanup(t, pool, tenantID, agentID) })

	marker := credentialMarker(vendor, "LOG")
	home := setupVendorAccountHome(t, vendor, marker)
	accountID := isolationSeedAccount(t, pool, vendor, tenantID, home, home)
	isolationAssign(t, pool, agentID, accountID)

	// Capture EVERYTHING the daemon gate logs.
	var gateBuf bytes.Buffer
	gateLog := captureLogger(&gateBuf)
	d := &Daemon{
		rotationStore: rotation.NewPGStore(pool),
		logger:        slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	}
	ctx := context.Background()

	resolved, err := d.credentialAccountHomeForTask(ctx, Task{AgentID: agentID}, vendor, gateLog)
	if err != nil {
		t.Fatalf("vendor %s: credentialAccountHomeForTask: %v", vendor, err)
	}

	// Capture EVERYTHING execenv.Prepare logs.
	var prepBuf bytes.Buffer
	prepLog := captureLogger(&prepBuf)
	env := prepareIsolatedEnv(t, vendor, resolved, prepLog)
	defer env.Cleanup(true)

	// Confirm the marker IS on disk (so its absence from logs is meaningful).
	credPath := vendorCredentialInEnv(env, vendor)
	if credPath == "" {
		t.Fatalf("vendor %s: no credential file in isolated env", vendor)
	}
	diskContent, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("vendor %s: read credential %s: %v", vendor, credPath, err)
	}
	if !strings.Contains(string(diskContent), marker) {
		t.Fatalf("vendor %s: credential on disk does not contain marker — fixture broken", vendor)
	}

	// The marker must not appear in either log buffer.
	for _, buf := range []*bytes.Buffer{&gateBuf, &prepBuf} {
		if strings.Contains(buf.String(), marker) {
			t.Fatalf("vendor %s: secret marker leaked into log output (len=%d):\n%s", vendor, buf.Len(), buf.String())
		}
	}
}
