//go:build staging

package daemon

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/rotation"
)

func TestStagingRotationProactiveBannerRotatesOnce(t *testing.T) {
	ctx := context.Background()
	pool := stagingRotationPool(t)
	store := rotation.NewPGStore(pool)
	accounts := stagingCodexAccounts(t, pool)
	if len(accounts) < 2 {
		t.Skipf("STG-SEED requires at least two codex accounts, got %d", len(accounts))
	}

	from := accounts[0]
	to := accounts[1]
	agentID := "30000000-0000-4000-8000-000000000001"
	banner := "Heads up, you have less than 10% of your 5h limit left. Run /status for details."
	now := time.Now().UTC()
	cooldownUntil := now.Add(5 * time.Hour)
	fromConfig := filepath.Join(t.TempDir(), "from-config")
	fromHome := filepath.Join(t.TempDir(), "from-home")
	toConfig := filepath.Join(t.TempDir(), "to-config")
	toHome := filepath.Join(t.TempDir(), "to-home")
	fromCredential := []byte(`{"fixture":"from"}`)
	toCredential := []byte(`{"fixture":"to"}`)
	stagingWriteCredential(t, filepath.Join(fromConfig, "auth.json"), fromCredential)
	stagingWriteCredential(t, filepath.Join(toConfig, "auth.json"), toCredential)

	stagingCleanupAgentRows(t, pool, agentID)
	t.Cleanup(func() {
		stagingCleanupAgentRows(t, pool, agentID)
		stagingRestoreAccount(t, pool, from)
		stagingRestoreAccount(t, pool, to)
	})

	stagingUpdateAccountForSmoke(t, pool, from.AccountID, 1, fromHome, fromConfig, rotation.StatusExhausted, &cooldownUntil)
	stagingUpdateAccountForSmoke(t, pool, to.AccountID, 2, toHome, toConfig, rotation.StatusAvailable, nil)
	if err := store.Assign(ctx, agentID, from.AccountID); err != nil {
		t.Fatalf("seed staging assignment: %v", err)
	}

	d := &Daemon{
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		rotationStore:     store,
		rotationService:   rotation.NewService(store, rotation.NewExhaustionDetector(), rotation.NewCredentialAuthenticator(), rotation.WithAuthenticationTimeout(time.Second)),
		warningDetector:   rotation.NewWarningDetector(),
		usageDetector:     rotation.NewUsageDetector(0),
		credentialMetrics: nil,
	}
	task := Task{ID: "staging-rotation-task", AgentID: agentID, WorkspaceID: from.TenantID}
	var rotationTriggered atomic.Bool

	got, ok := d.maybeProactiveRotateOnText(ctx, task, "codex", banner, d.logger, &rotationTriggered)
	if !ok {
		t.Fatal("proactive rotation did not trigger")
	}
	if got.AccountID != to.AccountID {
		t.Fatalf("rotated account = %s, want priority-2 account %s", got.AccountID, to.AccountID)
	}
	stagingAssertAssignment(t, pool, agentID, to.AccountID)
	stagingAssertRotationEvent(t, pool, agentID, from.AccountID, to.AccountID, rotation.ReasonQuotaProactive, 1)
	stagingAssertCredentialRestored(t, filepath.Join(toHome, "auth.json"), toCredential, fromCredential)

	if _, ok := d.maybeProactiveRotateOnText(ctx, task, "codex", banner, d.logger, &rotationTriggered); ok {
		t.Fatal("duplicate proactive banner generated a second rotation")
	}
	stagingAssertRotationEvent(t, pool, agentID, from.AccountID, to.AccountID, rotation.ReasonQuotaProactive, 1)

	stagingUpdateAccountForSmoke(t, pool, to.AccountID, 2, toHome, toConfig, rotation.StatusExhausted, &cooldownUntil)
	var noAccountRotation atomic.Bool
	if _, ok := d.maybeProactiveRotateOnText(ctx, task, "codex", banner, d.logger, &noAccountRotation); ok {
		t.Fatal("proactive rotation succeeded when no account was available")
	}
	stagingAssertRotationEvent(t, pool, agentID, from.AccountID, to.AccountID, rotation.ReasonQuotaProactive, 1)

	nilServiceDaemon := &Daemon{
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		warningDetector: rotation.NewWarningDetector(),
		usageDetector:   rotation.NewUsageDetector(0),
	}
	var nilServiceRotation atomic.Bool
	if _, ok := nilServiceDaemon.maybeProactiveRotateOnText(ctx, task, "codex", banner, nilServiceDaemon.logger, &nilServiceRotation); ok {
		t.Fatal("nil rotation service changed proactive flow")
	}
	stagingAssertRotationEvent(t, pool, agentID, from.AccountID, to.AccountID, rotation.ReasonQuotaProactive, 1)
}

type stagingAccountSnapshot struct {
	AccountID     string
	TenantID      string
	Priority      int
	HomeDir       string
	ConfigDir     string
	Status        rotation.AccountStatus
	CooldownUntil *time.Time
}

func stagingRotationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for staging rotation smoke test")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect staging Postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping staging Postgres: %v", err)
	}
	return pool
}

func stagingCodexAccounts(t *testing.T, pool *pgxpool.Pool) []stagingAccountSnapshot {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT account_id, tenant_id, priority, home_dir, config_dir, status, cooldown_until
		  FROM accounts
		 WHERE vendor = 'codex'
		 ORDER BY priority ASC, created_at ASC, account_id ASC
	`)
	if err != nil {
		t.Fatalf("query staging codex accounts: %v", err)
	}
	defer rows.Close()

	var accounts []stagingAccountSnapshot
	for rows.Next() {
		var account stagingAccountSnapshot
		if err := rows.Scan(&account.AccountID, &account.TenantID, &account.Priority, &account.HomeDir, &account.ConfigDir, &account.Status, &account.CooldownUntil); err != nil {
			t.Fatalf("scan staging codex account: %v", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate staging codex accounts: %v", err)
	}
	return accounts
}

func stagingCleanupAgentRows(t *testing.T, pool *pgxpool.Pool, agentID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM rotation_events WHERE agent_id = $1`, agentID); err != nil {
		t.Fatalf("cleanup staging rotation events: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM assignments WHERE agent_id = $1`, agentID); err != nil {
		t.Fatalf("cleanup staging assignment: %v", err)
	}
}

func stagingRestoreAccount(t *testing.T, pool *pgxpool.Pool, account stagingAccountSnapshot) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		UPDATE accounts
		   SET priority = $2,
		       home_dir = $3,
		       config_dir = $4,
		       status = $5,
		       cooldown_until = $6,
		       updated_at = now()
		 WHERE account_id = $1
	`, account.AccountID, account.Priority, account.HomeDir, account.ConfigDir, string(account.Status), account.CooldownUntil); err != nil {
		t.Fatalf("restore staging account: %v", err)
	}
}

func stagingUpdateAccountForSmoke(t *testing.T, pool *pgxpool.Pool, accountID string, priority int, homeDir, configDir string, status rotation.AccountStatus, cooldownUntil *time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		UPDATE accounts
		   SET priority = $2,
		       home_dir = $3,
		       config_dir = $4,
		       status = $5,
		       cooldown_until = $6,
		       updated_at = now()
		 WHERE account_id = $1
	`, accountID, priority, homeDir, configDir, string(status), cooldownUntil); err != nil {
		t.Fatalf("prepare staging account: %v", err)
	}
}

func stagingAssertAssignment(t *testing.T, pool *pgxpool.Pool, agentID, wantAccountID string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(context.Background(), `SELECT account_id FROM assignments WHERE agent_id = $1`, agentID).Scan(&got); err != nil {
		t.Fatalf("read staging assignment: %v", err)
	}
	if got != wantAccountID {
		t.Fatalf("assignment account = %s, want %s", got, wantAccountID)
	}
}

func stagingAssertRotationEvent(t *testing.T, pool *pgxpool.Pool, agentID, wantFrom, wantTo string, wantReason rotation.RotationReason, wantCount int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM rotation_events WHERE agent_id = $1`, agentID).Scan(&count); err != nil {
		t.Fatalf("count staging rotation events: %v", err)
	}
	if count != wantCount {
		t.Fatalf("rotation event count = %d, want %d", count, wantCount)
	}
	var gotFrom, gotTo, gotReason string
	if err := pool.QueryRow(context.Background(), `
		SELECT from_account_id, to_account_id, reason
		  FROM rotation_events
		 WHERE agent_id = $1
		 ORDER BY at DESC, created_at DESC
		 LIMIT 1
	`, agentID).Scan(&gotFrom, &gotTo, &gotReason); err != nil {
		t.Fatalf("read staging rotation event: %v", err)
	}
	if gotFrom != wantFrom || gotTo != wantTo || gotReason != string(wantReason) {
		t.Fatalf("rotation event = %s -> %s (%s), want %s -> %s (%s)", gotFrom, gotTo, gotReason, wantFrom, wantTo, wantReason)
	}
}

func stagingAssertCredentialRestored(t *testing.T, path string, want, forbidden []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read restored credential fixture: %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("restored credential fixture did not match selected account")
	}
	if string(got) == string(forbidden) {
		t.Fatal("restored credential fixture matched previous account")
	}
}

func stagingWriteCredential(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create staging credential fixture parent: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write staging credential fixture: %v", err)
	}
}
