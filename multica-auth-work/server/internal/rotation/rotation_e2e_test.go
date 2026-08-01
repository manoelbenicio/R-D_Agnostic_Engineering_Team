package rotation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRotationE2E_PostgresAccountRotationRestoresCredential(t *testing.T) {
	pool := e2eRotationPool(t)
	ctx := context.Background()
	e2eEnsureRotationSchema(t, pool)
	e2eCleanRotationTables(t, pool)

	now := time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC)
	tenantID := "11111111-1111-1111-1111-111111111111"
	agentID := "22222222-2222-2222-2222-222222222222"
	accountAID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	accountBID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	accountAConfig := filepath.Join(t.TempDir(), "account-a-config")
	accountAHome := filepath.Join(t.TempDir(), "account-a-home")
	accountBConfig := filepath.Join(t.TempDir(), "account-b-config")
	accountBHome := filepath.Join(t.TempDir(), "account-b-home")
	accountACredential := []byte(`{"account":"a","token":"fake-a"}`)
	accountBCredential := []byte(`{"account":"b","token":"fake-b"}`)
	e2eWriteCredential(t, filepath.Join(accountAConfig, "auth.json"), accountACredential)
	e2eWriteCredential(t, filepath.Join(accountBConfig, "auth.json"), accountBCredential)

	cooldownUntil := now.Add(5 * time.Hour)
	e2eInsertAccount(t, pool, accountAID, "codex", tenantID, 10, accountAHome, accountAConfig, StatusExhausted, &cooldownUntil)
	e2eInsertAccount(t, pool, accountBID, "codex", tenantID, 20, accountBHome, accountBConfig, StatusAvailable, nil)
	if err := NewPGStore(pool).Assign(ctx, agentID, accountAID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	service := NewService(NewPGStore(pool), NewExhaustionDetector(), NewCredentialAuthenticator())
	got, err := service.OnExhaustion(ctx, agentID, "codex", tenantID, ReasonQuotaReactive, now)
	if err != nil {
		t.Fatalf("OnExhaustion: %v", err)
	}
	if got.AccountID != accountBID {
		t.Fatalf("rotated account = %s, want %s", got.AccountID, accountBID)
	}

	var assignedAccountID string
	if err := pool.QueryRow(ctx, `SELECT account_id FROM assignments WHERE agent_id = $1`, agentID).Scan(&assignedAccountID); err != nil {
		t.Fatalf("read assignment: %v", err)
	}
	if assignedAccountID != accountBID {
		t.Fatalf("assignment account = %s, want %s", assignedAccountID, accountBID)
	}

	var fromAccountID, toAccountID, reason string
	if err := pool.QueryRow(ctx, `
		SELECT from_account_id, to_account_id, reason
		  FROM rotation_events
		 WHERE agent_id = $1
		 ORDER BY at DESC, created_at DESC
		 LIMIT 1
	`, agentID).Scan(&fromAccountID, &toAccountID, &reason); err != nil {
		t.Fatalf("read rotation event: %v", err)
	}
	if fromAccountID != accountAID || toAccountID != accountBID || reason != string(ReasonQuotaReactive) {
		t.Fatalf("rotation event = %s -> %s (%s), want %s -> %s (%s)", fromAccountID, toAccountID, reason, accountAID, accountBID, ReasonQuotaReactive)
	}

	restoredB, err := os.ReadFile(filepath.Join(accountBHome, "auth.json"))
	if err != nil {
		t.Fatalf("read restored account B credential: %v", err)
	}
	if !bytes.Equal(restoredB, accountBCredential) {
		t.Fatal("restored account B credential did not match account B source")
	}
	if bytes.Equal(restoredB, accountACredential) {
		t.Fatal("restored account B credential was contaminated by account A")
	}

	if err := NewPGStore(pool).UpdateAccountStatus(ctx, accountBID, StatusExhausted, &cooldownUntil); err != nil {
		t.Fatalf("mark account B exhausted: %v", err)
	}
	_, err = service.OnExhaustion(ctx, agentID, "codex", tenantID, ReasonQuotaReactive, now)
	if !errors.Is(err, ErrNoAccountAvailable) {
		t.Fatalf("OnExhaustion with all accounts exhausted error = %v, want %v", err, ErrNoAccountAvailable)
	}
}

func e2eRotationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; rotation E2E requires real Postgres")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect Postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping Postgres: %v", err)
	}
	return pool
}

func e2eEnsureRotationSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.accounts') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("check rotation schema: %v", err)
	}
	if exists {
		return
	}
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "123_rotation.up.sql"))
	if err != nil {
		t.Fatalf("read rotation migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply rotation migration: %v", err)
	}
}

func e2eCleanRotationTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `TRUNCATE rotation_events, assignments, credentials, accounts RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("clean rotation tables: %v", err)
	}
}

func e2eInsertAccount(t *testing.T, pool *pgxpool.Pool, accountID, vendor, tenantID string, priority int, homeDir, configDir string, status AccountStatus, cooldownUntil *time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO accounts (
			account_id, vendor, tenant_id, priority, home_dir, config_dir, status,
			tokens_per_window, tokens_used, cooldown_until
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 100000, 0, $8)
	`, accountID, vendor, tenantID, priority, homeDir, configDir, string(status), cooldownUntil)
	if err != nil {
		t.Fatalf("insert account fixture: %v", err)
	}
}

func e2eWriteCredential(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create credential fixture parent: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write credential fixture: %v", err)
	}
}
