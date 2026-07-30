package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestMigration129TaskUsagePriceSnapshot(t *testing.T) {
	pool := openTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := fmt.Sprintf("migration_129_test_%d_%d", time.Now().UnixNano(), rand.Uint32())
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", pgx.Identifier{schema}.Sanitize())); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", pgx.Identifier{schema}.Sanitize())); err != nil {
			t.Logf("drop schema %s: %v", schema, err)
		}
	})

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", pgx.Identifier{schema}.Sanitize())); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE task_usage (id BIGSERIAL PRIMARY KEY)`); err != nil {
		t.Fatalf("create task_usage fixture: %v", err)
	}

	up := readMigration129(t, "129_task_usage_price_snapshot.up.sql")
	down := readMigration129(t, "129_task_usage_price_snapshot.down.sql")

	if _, err := conn.Exec(ctx, up); err != nil {
		t.Fatalf("fresh apply: %v", err)
	}
	if _, err := conn.Exec(ctx, up); err != nil {
		t.Fatalf("reapply after DDL completed but migration was not recorded: %v", err)
	}

	assertInsertAccepted(t, ctx, conn.Conn(), `INSERT INTO task_usage DEFAULT VALUES`)
	assertInsertAccepted(t, ctx, conn.Conn(), `INSERT INTO task_usage (price_version, computed_cost_usd) VALUES ('2026-07-30', 1.25)`)
	assertInsertRejected(t, ctx, conn.Conn(), `INSERT INTO task_usage (price_version) VALUES ('2026-07-30')`)
	assertInsertRejected(t, ctx, conn.Conn(), `INSERT INTO task_usage (computed_cost_usd) VALUES (1.25)`)
	assertInsertRejected(t, ctx, conn.Conn(), `INSERT INTO task_usage (price_version, computed_cost_usd) VALUES ('2026-07-30', -0.01)`)

	if _, err := conn.Exec(ctx, down); err != nil {
		t.Fatalf("down migration: %v", err)
	}
	if _, err := conn.Exec(ctx, up); err != nil {
		t.Fatalf("up migration after down: %v", err)
	}
	assertInsertRejected(t, ctx, conn.Conn(), `INSERT INTO task_usage (price_version) VALUES ('after-down-up')`)
}

func readMigration129(t *testing.T, name string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(contents)
}

func assertInsertAccepted(t *testing.T, ctx context.Context, conn *pgx.Conn, query string) {
	t.Helper()
	if _, err := conn.Exec(ctx, query); err != nil {
		t.Fatalf("expected insert to be accepted: %v", err)
	}
}

func assertInsertRejected(t *testing.T, ctx context.Context, conn *pgx.Conn, query string) {
	t.Helper()
	if _, err := conn.Exec(ctx, query); err == nil {
		t.Fatal("expected insert to violate task_usage_price_snapshot_consistent")
	}
}
