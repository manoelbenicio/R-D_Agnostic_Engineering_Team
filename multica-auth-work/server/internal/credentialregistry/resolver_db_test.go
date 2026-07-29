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
	otherAgentID := uuid.NewString()
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
	if _, err := tx.Exec(ctx, `UPDATE accounts SET vendor='antigravity' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("set canonical antigravity vendor: %v", err)
	}
	if got, err := resolver.Resolve(ctx, agentID, "agy"); err != nil || got.Vendor != "antigravity" {
		t.Fatalf("canonical vendor with runtime alias: got=%+v err=%v", got, err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET vendor='agy' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("set noncanonical vendor fixture: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "agy"); !errors.Is(err, ErrProviderMismatch) {
		t.Fatalf("noncanonical stored vendor=%v, want ErrProviderMismatch", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET vendor='kiro' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("restore canonical kiro vendor: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO agent (id, workspace_id, name, runtime_mode, runtime_id)
		VALUES ($1, $2, 'ORQ21 other agent', 'local', $3)`, otherAgentID, workspaceID, runtimeID); err != nil {
		t.Fatalf("insert other agent: %v", err)
	}
	var accountUniqueIndex bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1
		  FROM pg_index AS i
		  JOIN pg_class AS tab ON tab.oid=i.indrelid
		 WHERE tab.relname='assignments'
		   AND i.indisunique
		   AND pg_get_indexdef(i.indexrelid) LIKE '%(account_id)%'
	)`).Scan(&accountUniqueIndex); err != nil {
		t.Fatalf("detect account uniqueness: %v", err)
	}
	if !accountUniqueIndex {
		if _, err := tx.Exec(ctx, `INSERT INTO assignments (agent_id, account_id) VALUES ($1, $2)`, otherAgentID, accountID); err != nil {
			t.Fatalf("duplicate account assignment fixture: %v", err)
		}
		// Before the ORQ-12 unique index is promoted, the resolver remains a
		// second fail-closed defense against a legacy duplicate.
		if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrAccountAlreadyUsed) {
			t.Fatalf("shared account=%v, want ErrAccountAlreadyUsed", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM assignments WHERE agent_id=$1`, otherAgentID); err != nil {
			t.Fatalf("remove duplicate account assignment fixture: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET status='leased' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("mark account leased: %v", err)
	}
	if got, err := resolver.Resolve(ctx, agentID, "kiro"); err != nil || got.AccountID != accountID {
		t.Fatalf("exclusively assigned leased account: got=%+v err=%v", got, err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET status='cooldown' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("mark account cooldown: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("cooldown account=%v, want ErrAccountUnavailable", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET status='available' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("restore account available: %v", err)
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
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET worktype_scope='CHEAP' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("set unsupported worktype scope: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("unsupported worktype scope=%v, want ErrInvalidMetadata", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET worktype_scope='GENERAL' WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("restore GENERAL worktype scope: %v", err)
	}
	otherWorkspaceID := uuid.NewString()
	otherAccountID := uuid.NewString()
	if _, err := tx.Exec(ctx, `INSERT INTO workspace (id, name, slug) VALUES ($1, 'ORQ21 other workspace', $2)`,
		otherWorkspaceID, "orq21-other-"+uuid.NewString()); err != nil {
		t.Fatalf("insert other workspace: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO accounts
		(account_id, vendor, tenant_id, priority, home_dir, config_dir, status)
		VALUES ($1, 'kiro', $2, 10, '/private/orq21/cross/home', '', 'available')`,
		otherAccountID, otherWorkspaceID); err != nil {
		t.Fatalf("insert cross-workspace account: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO approved_accounts
		(tenant_id, account_id, allowed, worktype_scope) VALUES ($1, $2, true, 'GENERAL')`,
		otherWorkspaceID, otherAccountID); err != nil {
		t.Fatalf("approve cross-workspace account: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE assignments SET account_id=$1 WHERE agent_id=$2`, otherAccountID, agentID); err != nil {
		t.Fatalf("set cross-workspace assignment: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("cross-workspace assignment=%v, want ErrNoApprovedAssignment", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE assignments SET account_id=$1 WHERE agent_id=$2`, accountID, agentID); err != nil {
		t.Fatalf("restore same-workspace assignment: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE approved_accounts SET allowed=false WHERE account_id=$1`, accountID); err != nil {
		t.Fatalf("revoke approval: %v", err)
	}
	if _, err := resolver.Resolve(ctx, agentID, "kiro"); !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("revoked assignment=%v, want ErrNoApprovedAssignment", err)
	}
}
