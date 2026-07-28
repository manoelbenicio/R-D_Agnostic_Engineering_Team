//go:build orq21db

package credentialregistry

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestResolverDBApprovedAssignmentContract(t *testing.T) {
	dbURL := os.Getenv("ORQ21_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("ORQ21_TEST_DATABASE_URL is required for the orq21db gate")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	workspaceID := uuid.NewString()
	runtimeID := uuid.NewString()
	agentID := uuid.NewString()
	accountID := uuid.NewString()
	homeDir := "/private/orq21/slot-1/xdg-data"
	slug := "orq21-" + uuid.NewString()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin fixture: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })

	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO workspace (id, name, slug) VALUES ($1, 'ORQ21 resolver DB gate', $2)`, []any{workspaceID, slug}},
		{`INSERT INTO agent_runtime (
			id, workspace_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at
		  ) VALUES ($1, $2, 'ORQ21 runtime', 'local', 'kiro', 'online', 'orq21 gate', '{}'::jsonb, now())`, []any{runtimeID, workspaceID}},
		{`INSERT INTO agent (id, workspace_id, name, runtime_mode, runtime_id)
		  VALUES ($1, $2, 'ORQ21 agent', 'local', $3)`, []any{agentID, workspaceID, runtimeID}},
		{`INSERT INTO accounts (account_id, vendor, tenant_id, priority, home_dir, config_dir, status)
		  VALUES ($1, 'kiro', $2, 10, $3, '/private/orq21/slot-1/xdg-config', 'available')`, []any{accountID, workspaceID, homeDir}},
		{`INSERT INTO approved_accounts (tenant_id, account_id, allowed, worktype_scope)
		  VALUES ($1, $2, true, 'GENERAL')`, []any{workspaceID, accountID}},
		{`INSERT INTO assignments (agent_id, account_id) VALUES ($1, $2)`, []any{agentID, accountID}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("insert metadata-only fixture: %v", err)
		}
	}

	resolver := NewResolver(tx)
	got, err := resolver.Resolve(ctx, agentID, "kiro")
	if err != nil {
		t.Fatalf("Resolve approved assignment: %v", err)
	}
	if got.AccountID != accountID || got.TenantID != workspaceID || got.HomeDir != homeDir || got.Vendor != "kiro" {
		t.Fatalf("unexpected metadata assignment: %+v", got)
	}

	var credentialRows int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM credentials WHERE account_id=$1`, accountID).Scan(&credentialRows); err != nil {
		t.Fatalf("count credential metadata: %v", err)
	}
	if credentialRows != 0 {
		t.Fatalf("credential rows=%d, want 0", credentialRows)
	}

	if _, err := resolver.Resolve(ctx, agentID, "codex"); !errors.Is(err, ErrProviderMismatch) {
		t.Fatalf("provider mismatch=%v, want ErrProviderMismatch", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET worktype_scope=NULL WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("clear worktype scope: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("missing worktype scope=%v, want ErrInvalidMetadata", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET worktype_scope='GENERAL' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("restore worktype scope: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET allowed=false WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("revoke approval: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("revoked assignment=%v, want ErrNoApprovedAssignment", err)
	}
}
