package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigration137NativeRotationDefinition(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations",
		"137_native_rotation_epoch_recovery.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(up)
	for _, required := range []string{
		"multica_credential_home_advisory_key(home_ref UUID)",
		"runtime_task_home_epoch",
		"native_rotation_operation_nonterminal_task",
		"native_rotation_retirement_attempt_inflight",
		"runtime_task_home_epoch_operation_required",
		"native_rotation_operation_event_required",
		"native_rotation_retirement_attempt_event_required",
		"swap_commit_unknown_fenced",
		"aborted_candidate_retirement_pending",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 137 lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"home_dir", "config_dir", "account_id", "credential_value",
		"access_token", "refresh_token", "secret_value",
	} {
		if strings.Contains(strings.ToLower(sql), forbidden) {
			t.Fatalf("migration 137 contains forbidden material field %q", forbidden)
		}
	}
}

func TestMigration137NativeRotation(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createMigration136Prerequisites(t, ctx, f)
	opts := f.opts()
	opts.Files = scopedRealMigrations(t, f.schema,
		"132_credential_home_catalog.up.sql",
		"133_runtime_bindings.up.sql",
		"134_runtime_configuration_snapshots.up.sql",
		"136_runtime_home_lifetime_readiness.up.sql",
		"137_native_rotation_epoch_recovery.up.sql",
	)
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply migrations 132-137: %v", err)
	}

	for _, relation := range []string{
		"runtime_task_home_epoch", "native_rotation_operation",
		"native_rotation_operation_event", "native_rotation_retirement_attempt",
		"native_rotation_retirement_attempt_event",
	} {
		var exists bool
		if err := f.pool.QueryRow(ctx,
			`SELECT to_regclass($1) IS NOT NULL`, f.schema+"."+relation,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("relation %s exists=%v err=%v", relation, exists, err)
		}
	}

	var first, second int64
	query := `SELECT multica_credential_home_advisory_key(
		'00000000-0000-0000-0000-000000000001'::uuid)`
	if err := f.pool.QueryRow(ctx, query).Scan(&first); err != nil {
		t.Fatalf("call shared advisory helper: %v", err)
	}
	if err := f.pool.QueryRow(ctx, query).Scan(&second); err != nil || first != second {
		t.Fatalf("shared advisory helper is not deterministic: %d/%d err=%v", first, second, err)
	}

	rows, err := f.pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name IN (
		    'runtime_task_home_epoch', 'native_rotation_operation',
		    'native_rotation_retirement_attempt'
		  )`, f.schema)
	if err != nil {
		t.Fatalf("list native rotation columns: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"path", "home_dir", "account", "credential", "token", "secret"} {
			if strings.Contains(column, forbidden) {
				t.Fatalf("forbidden value-bearing column %q", column)
			}
		}
	}

	down := f.opts()
	down.Direction = "down"
	down.Files = scopedRealMigrations(t, f.schema,
		"137_native_rotation_epoch_recovery.down.sql",
	)
	if err := runMigrations(ctx, f.pool, down); err != nil {
		t.Fatalf("empty rollback migration 137: %v", err)
	}
}
