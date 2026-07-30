package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestMigration129TaskUsagePriceSnapshot(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createMigration129Prerequisites(t, ctx, f)
	upFiles := scopedRealMigrations(t, f.schema,
		"127_task_usage_thinking_level.up.sql",
		"128_task_usage_account_id.up.sql",
		"129_task_usage_price_snapshot.up.sql",
	)

	opts := f.opts()
	opts.Files = upFiles[:2]
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply real migrations 127 and 128: %v", err)
	}

	taskUsage := pgx.Identifier{f.schema, "task_usage"}.Sanitize()
	if _, err := f.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, task_id, provider, model, input_tokens, output_tokens)
		VALUES ('00000000-0000-0000-0000-000000000129', '00000000-0000-0000-0000-000000000001', 'codex', 'gpt-5', 10, 5)
	`, taskUsage)); err != nil {
		t.Fatalf("seed populated task_usage before migration 129: %v", err)
	}

	opts.Files = upFiles[2:]
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("fresh apply real migration 129: %v", err)
	}
	assertMigration129Constraint(t, ctx, f)
	assertMigration129Semantics(t, ctx, f)

	version := "129_task_usage_price_snapshot"
	trackingTable := pgx.Identifier{f.schema, "schema_migrations"}.Sanitize()
	if _, err := f.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, trackingTable), version); err != nil {
		t.Fatalf("simulate applied-but-unrecorded migration 129: %v", err)
	}
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("recover applied-but-unrecorded migration 129: %v", err)
	}
	var records int
	if err := f.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE version = $1`, trackingTable), version).Scan(&records); err != nil {
		t.Fatalf("count migration 129 records: %v", err)
	}
	if records != 1 {
		t.Fatalf("migration 129 records = %d, want 1", records)
	}
	assertMigration129Constraint(t, ctx, f)

	downFiles := scopedRealMigrations(t, f.schema, "129_task_usage_price_snapshot.down.sql")
	downOpts := f.opts()
	downOpts.Direction = "down"
	downOpts.Files = downFiles
	if err := runMigrations(ctx, f.pool, downOpts); err != nil {
		t.Fatalf("down migration 129: %v", err)
	}
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("up migration 129 after down: %v", err)
	}
	assertMigration129Constraint(t, ctx, f)
}

func createMigration129Prerequisites(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	schema := pgx.Identifier{f.schema}.Sanitize()
	_, err := f.pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.accounts (
			account_id UUID PRIMARY KEY,
			vendor TEXT NOT NULL
		);
		CREATE TABLE %[1]s.agent_runtime (
			id UUID PRIMARY KEY,
			provider TEXT NOT NULL
		);
		CREATE TABLE %[1]s.assignments (
			agent_id UUID PRIMARY KEY,
			account_id UUID NOT NULL REFERENCES %[1]s.accounts(account_id)
		);
		CREATE TABLE %[1]s.agent_task_queue (
			id UUID PRIMARY KEY
		);
		CREATE TABLE %[1]s.task_usage (
			id UUID PRIMARY KEY,
			task_id UUID NOT NULL REFERENCES %[1]s.agent_task_queue(id) ON DELETE CASCADE,
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			input_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			cache_read_tokens BIGINT NOT NULL DEFAULT 0,
			cache_write_tokens BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (task_id, provider, model)
		);
		INSERT INTO %[1]s.agent_task_queue (id)
		VALUES ('00000000-0000-0000-0000-000000000001');
	`, schema))
	if err != nil {
		t.Fatalf("create migration 129 prerequisites: %v", err)
	}
}

func scopedRealMigrations(t *testing.T, schema string, names ...string) []string {
	t.Helper()
	dir := t.TempDir()
	files := make([]string, 0, len(names))
	for _, name := range names {
		contents, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatalf("read real migration %s: %v", name, err)
		}
		path := filepath.Join(dir, name)
		prefix := fmt.Sprintf("SET search_path TO %s;\n", pgx.Identifier{schema}.Sanitize())
		if err := os.WriteFile(path, append([]byte(prefix), contents...), 0o600); err != nil {
			t.Fatalf("scope real migration %s: %v", name, err)
		}
		files = append(files, path)
	}
	return files
}

func assertMigration129Constraint(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	var count int
	var validated bool
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*), bool_and(c.convalidated)
		FROM pg_constraint c
		JOIN pg_class r ON r.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = r.relnamespace
		WHERE n.nspname = $1
		  AND r.relname = 'task_usage'
		  AND c.conname = 'task_usage_price_snapshot_consistent'
	`, f.schema).Scan(&count, &validated); err != nil {
		t.Fatalf("read migration 129 constraint: %v", err)
	}
	if count != 1 || !validated {
		t.Fatalf("constraint count = %d, validated = %t; want exactly one validated constraint", count, validated)
	}
}

func assertMigration129Semantics(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	table := pgx.Identifier{f.schema, "task_usage"}.Sanitize()
	assertUsageInsertAccepted(t, ctx, f, fmt.Sprintf(`INSERT INTO %s (id, task_id, provider, model) VALUES ('00000000-0000-0000-0000-000000000130', '00000000-0000-0000-0000-000000000001', 'codex', 'null-price')`, table))
	assertUsageInsertAccepted(t, ctx, f, fmt.Sprintf(`INSERT INTO %s (id, task_id, provider, model, price_version, computed_cost_usd) VALUES ('00000000-0000-0000-0000-000000000131', '00000000-0000-0000-0000-000000000001', 'codex', 'valid-price', 'v1', 1.25)`, table))
	assertUsageInsertRejected(t, ctx, f, fmt.Sprintf(`INSERT INTO %s (id, task_id, provider, model, price_version) VALUES ('00000000-0000-0000-0000-000000000132', '00000000-0000-0000-0000-000000000001', 'codex', 'half-version', 'v1')`, table))
	assertUsageInsertRejected(t, ctx, f, fmt.Sprintf(`INSERT INTO %s (id, task_id, provider, model, computed_cost_usd) VALUES ('00000000-0000-0000-0000-000000000133', '00000000-0000-0000-0000-000000000001', 'codex', 'half-cost', 1.25)`, table))
	assertUsageInsertRejected(t, ctx, f, fmt.Sprintf(`INSERT INTO %s (id, task_id, provider, model, price_version, computed_cost_usd) VALUES ('00000000-0000-0000-0000-000000000134', '00000000-0000-0000-0000-000000000001', 'codex', 'negative', 'v1', -0.01)`, table))
}

func assertUsageInsertAccepted(t *testing.T, ctx context.Context, f *fixture, query string) {
	t.Helper()
	if _, err := f.pool.Exec(ctx, query); err != nil {
		t.Fatalf("expected insert to be accepted: %v", err)
	}
}

func assertUsageInsertRejected(t *testing.T, ctx context.Context, f *fixture, query string) {
	t.Helper()
	if _, err := f.pool.Exec(ctx, query); err == nil {
		t.Fatal("expected insert to violate task_usage_price_snapshot_consistent")
	}
}
