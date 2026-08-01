package rotation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("rotation store tests require DATABASE_URL")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("rotation store tests require Postgres: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("rotation store tests require Postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func setupPGStore(t *testing.T) *PGStore {
	t.Helper()
	pool := testPool(t)
	ensureRotationSchema(t, pool)
	return NewPGStore(pool)
}

func ensureRotationSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.accounts') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("check rotation schema: %v", err)
	}
	if exists {
		return
	}

	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "123_rotation.up.sql"))
	if err != nil {
		t.Fatalf("read 123_rotation.up.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply 123_rotation.up.sql: %v", err)
	}
}

func seedPGAccount(t *testing.T, pool *pgxpool.Pool, account Account) string {
	t.Helper()
	if account.AccountID == "" {
		account.AccountID = uuid.NewString()
	}
	if account.TokensPerWin == 0 {
		account.TokensPerWin = 100000
	}
	if account.Status == "" {
		account.Status = StatusAvailable
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO accounts (
			account_id, vendor, tenant_id, priority, home_dir, config_dir, status,
			tokens_per_window, tokens_used, window_start, cooldown_until, last_error
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, account.AccountID, account.Vendor, account.TenantID, account.Priority, account.HomeDir,
		account.ConfigDir, string(account.Status), account.TokensPerWin, account.TokensUsed,
		account.WindowStart, account.CooldownUntil, account.LastError); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return account.AccountID
}

func cleanupPGTenant(t *testing.T, pool *pgxpool.Pool, tenantID string, agentIDs ...string) {
	t.Helper()
	ctx := context.Background()
	for _, agentID := range agentIDs {
		if _, err := pool.Exec(ctx, `DELETE FROM rotation_events WHERE agent_id = $1`, agentID); err != nil {
			t.Logf("cleanup rotation_events for %s: %v", agentID, err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM assignments WHERE agent_id = $1`, agentID); err != nil {
			t.Logf("cleanup assignments for %s: %v", agentID, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM accounts WHERE tenant_id = $1`, tenantID); err != nil {
		t.Logf("cleanup accounts for tenant %s: %v", tenantID, err)
	}
}

func TestPGStoreListAccountsMapsRowsAndOrdersByPriority(t *testing.T) {
	store := setupPGStore(t)
	tenantID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)
	cooldownUntil := now.Add(time.Hour)
	t.Cleanup(func() { cleanupPGTenant(t, store.pool, tenantID) })

	low := seedPGAccount(t, store.pool, Account{
		Vendor:        "codex",
		TenantID:      tenantID,
		Priority:      30,
		HomeDir:       "/home/low",
		ConfigDir:     "/config/low",
		Status:        StatusAvailable,
		TokensPerWin:  9000,
		TokensUsed:    90,
		WindowStart:   &now,
		CooldownUntil: &cooldownUntil,
		LastError:     "previous failure",
	})
	high := seedPGAccount(t, store.pool, Account{
		Vendor:       "codex",
		TenantID:     tenantID,
		Priority:     10,
		HomeDir:      "/home/high",
		ConfigDir:    "/config/high",
		Status:       StatusLeased,
		TokensPerWin: 1000,
		TokensUsed:   10,
	})

	got, err := store.ListAccounts(context.Background(), "codex", tenantID)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAccounts count = %d, want 2", len(got))
	}
	if got[0].AccountID != high || got[1].AccountID != low {
		t.Fatalf("ListAccounts order = %s,%s; want %s,%s", got[0].AccountID, got[1].AccountID, high, low)
	}

	mapped := got[1]
	if mapped.Vendor != "codex" || mapped.TenantID != tenantID || mapped.Priority != 30 ||
		mapped.HomeDir != "/home/low" || mapped.ConfigDir != "/config/low" ||
		mapped.Status != StatusAvailable || mapped.TokensPerWin != 9000 ||
		mapped.TokensUsed != 90 || mapped.LastError != "previous failure" {
		t.Fatalf("mapped account = %+v", mapped)
	}
	if mapped.WindowStart == nil || !mapped.WindowStart.Truncate(time.Second).Equal(now) {
		t.Fatalf("window_start = %v, want %v", mapped.WindowStart, now)
	}
	if mapped.CooldownUntil == nil || !mapped.CooldownUntil.Truncate(time.Second).Equal(cooldownUntil) {
		t.Fatalf("cooldown_until = %v, want %v", mapped.CooldownUntil, cooldownUntil)
	}
}

func TestPGStoreGetAccountAndMissing(t *testing.T) {
	store := setupPGStore(t)
	tenantID := uuid.NewString()
	t.Cleanup(func() { cleanupPGTenant(t, store.pool, tenantID) })

	accountID := seedPGAccount(t, store.pool, Account{
		Vendor:   "kiro",
		TenantID: tenantID,
		Priority: 5,
		Status:   StatusAvailable,
	})

	got, err := store.GetAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.AccountID != accountID || got.Vendor != "kiro" || got.TenantID != tenantID || got.Priority != 5 {
		t.Fatalf("GetAccount = %+v", got)
	}

	if _, err := store.GetAccount(context.Background(), uuid.NewString()); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("GetAccount(missing) err = %v, want ErrAccountNotFound", err)
	}
}

func TestPGStoreUpdateAccountStatusAndRecordUsage(t *testing.T) {
	store := setupPGStore(t)
	tenantID := uuid.NewString()
	t.Cleanup(func() { cleanupPGTenant(t, store.pool, tenantID) })

	accountID := seedPGAccount(t, store.pool, Account{
		Vendor:   "codex",
		TenantID: tenantID,
		Priority: 1,
		Status:   StatusAvailable,
	})
	cooldownUntil := time.Now().UTC().Add(5 * time.Hour).Truncate(time.Second)
	windowStart := time.Now().UTC().Truncate(time.Second)

	if err := store.UpdateAccountStatus(context.Background(), accountID, StatusCooldown, &cooldownUntil); err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if err := store.RecordUsage(context.Background(), accountID, 42000, windowStart); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	got, err := store.GetAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Status != StatusCooldown {
		t.Fatalf("status = %s, want cooldown", got.Status)
	}
	if got.CooldownUntil == nil || !got.CooldownUntil.Truncate(time.Second).Equal(cooldownUntil) {
		t.Fatalf("cooldown_until = %v, want %v", got.CooldownUntil, cooldownUntil)
	}
	if got.TokensUsed != 42000 {
		t.Fatalf("tokens_used = %d, want 42000", got.TokensUsed)
	}
	if got.WindowStart == nil || !got.WindowStart.Truncate(time.Second).Equal(windowStart) {
		t.Fatalf("window_start = %v, want %v", got.WindowStart, windowStart)
	}

	if err := store.UpdateAccountStatus(context.Background(), accountID, StatusAvailable, nil); err != nil {
		t.Fatalf("UpdateAccountStatus clear cooldown: %v", err)
	}
	got, err = store.GetAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("GetAccount after clear: %v", err)
	}
	if got.CooldownUntil != nil {
		t.Fatalf("cooldown_until = %v, want nil after clear", got.CooldownUntil)
	}

	if err := store.UpdateAccountStatus(context.Background(), uuid.NewString(), StatusDegraded, nil); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("UpdateAccountStatus(missing) err = %v, want ErrAccountNotFound", err)
	}
	if err := store.RecordUsage(context.Background(), uuid.NewString(), 1, windowStart); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("RecordUsage(missing) err = %v, want ErrAccountNotFound", err)
	}
}

func TestPGStoreAssignCurrentAssignmentAndRecordRotation(t *testing.T) {
	store := setupPGStore(t)
	tenantID := uuid.NewString()
	agentID := uuid.NewString()
	t.Cleanup(func() { cleanupPGTenant(t, store.pool, tenantID, agentID) })

	fromAccountID := seedPGAccount(t, store.pool, Account{
		Vendor:   "codex",
		TenantID: tenantID,
		Priority: 1,
		Status:   StatusExhausted,
	})
	toAccountID := seedPGAccount(t, store.pool, Account{
		Vendor:   "codex",
		TenantID: tenantID,
		Priority: 2,
		Status:   StatusAvailable,
	})

	if _, err := store.CurrentAssignment(context.Background(), agentID); !errors.Is(err, ErrNoAssignment) {
		t.Fatalf("CurrentAssignment(empty) err = %v, want ErrNoAssignment", err)
	}
	if err := store.Assign(context.Background(), agentID, fromAccountID); err != nil {
		t.Fatalf("Assign first: %v", err)
	}
	current, err := store.CurrentAssignment(context.Background(), agentID)
	if err != nil {
		t.Fatalf("CurrentAssignment first: %v", err)
	}
	if current != fromAccountID {
		t.Fatalf("current = %s, want %s", current, fromAccountID)
	}
	if err := store.Assign(context.Background(), agentID, toAccountID); err != nil {
		t.Fatalf("Assign second: %v", err)
	}
	current, err = store.CurrentAssignment(context.Background(), agentID)
	if err != nil {
		t.Fatalf("CurrentAssignment second: %v", err)
	}
	if current != toAccountID {
		t.Fatalf("current = %s, want %s", current, toAccountID)
	}

	at := time.Now().UTC().Truncate(time.Second)
	if err := store.RecordRotation(context.Background(), agentID, "", toAccountID, ReasonQuotaReactive, at); err != nil {
		t.Fatalf("RecordRotation without from: %v", err)
	}
	if err := store.RecordRotation(context.Background(), agentID, fromAccountID, toAccountID, ReasonLoginFailed, at.Add(time.Second)); err != nil {
		t.Fatalf("RecordRotation with from: %v", err)
	}

	var count int
	if err := store.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM rotation_events WHERE agent_id = $1`, agentID).Scan(&count); err != nil {
		t.Fatalf("count rotation_events: %v", err)
	}
	if count != 2 {
		t.Fatalf("rotation_events count = %d, want 2", count)
	}

	var hasNullFrom bool
	if err := store.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM rotation_events
			 WHERE agent_id = $1 AND from_account_id IS NULL
		)
	`, agentID).Scan(&hasNullFrom); err != nil {
		t.Fatalf("check null from_account_id: %v", err)
	}
	if !hasNullFrom {
		t.Fatal("expected a rotation_event with NULL from_account_id")
	}

	var lastReason string
	if err := store.pool.QueryRow(context.Background(),
		`SELECT reason FROM rotation_events WHERE agent_id = $1 ORDER BY at DESC LIMIT 1`, agentID).Scan(&lastReason); err != nil {
		t.Fatalf("read last reason: %v", err)
	}
	if lastReason != string(ReasonLoginFailed) {
		t.Fatalf("last reason = %s, want %s", lastReason, ReasonLoginFailed)
	}
}
