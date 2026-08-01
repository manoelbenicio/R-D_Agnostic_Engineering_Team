package credentialregistry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	testOwnerID     = "71000000-0000-4000-8000-000000000001"
	testWorkspaceID = "71000000-0000-4000-8000-000000000002"
	testRuntimeID   = "71000000-0000-4000-8000-000000000003"
	testAgentID     = "71000000-0000-4000-8000-000000000004"
	testStandardID  = "71000000-0000-4000-8000-000000000005"
	testStdVersion  = "71000000-0000-4000-8000-000000000006"
	testSessionID   = "71000000-0000-4000-8000-000000000007"
	testEnrollment  = "71000000-0000-4000-8000-000000000008"
	testCatalogID   = "71000000-0000-4000-8000-000000000009"
	testCatalogGen  = "71000000-0000-4000-8000-00000000000a"
	testEntryID     = "71000000-0000-4000-8000-00000000000b"
	testHomeRef     = "71000000-0000-4000-8000-00000000000c"
	testBindingID   = "71000000-0000-4000-8000-00000000000d"
	testAssignment  = "71000000-0000-4000-8000-00000000000e"
	testConfig      = "71000000-0000-4000-8000-00000000000f"
)

func TestPostgresStoreMigrations132To134AtomicAdmission(t *testing.T) {
	adminURL := os.Getenv("SPE6_CREDENTIALREGISTRY_DATABASE_URL")
	if adminURL == "" {
		t.Skip("SPE6_CREDENTIALREGISTRY_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool := newTemporaryRegistryDatabase(t, ctx, adminURL)
	applyRegistryMigrations(t, ctx, pool)
	seedRegistry(t, ctx, pool)
	resolver := NewResolver(NewPostgresStore(pool))

	t.Run("uses singular C2 contract", func(t *testing.T) {
		var singular, plural bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass('runtime_binding') IS NOT NULL,
            to_regclass('runtime_binding_task_reservations') IS NOT NULL`).Scan(&singular, &plural); err != nil {
			t.Fatalf("inspect schema: %v", err)
		}
		if !singular || plural {
			t.Fatalf("singular=%v nonexistent_reservation_table=%v", singular, plural)
		}
	})

	t.Run("validation rollback cannot leak capacity", func(t *testing.T) {
		taskID := "71000000-0000-4000-8001-000000000001"
		insertTask(t, ctx, pool, taskID)
		request := registryRequest(taskID)
		request.Provider = "kiro"
		if _, err := resolver.Resolve(ctx, request); !errors.Is(err, ErrProviderMismatch) {
			t.Fatalf("Resolve() error=%v, want provider mismatch", err)
		}
		assertTaskState(t, ctx, pool, taskID, "queued", 0, 0)
	})

	t.Run("snapshot failure rolls back capacity and claim", func(t *testing.T) {
		taskID := "71000000-0000-4000-8001-000000000002"
		insertTask(t, ctx, pool, taskID)
		if _, err := pool.Exec(ctx, fmt.Sprintf(`
CREATE OR REPLACE FUNCTION credentialregistry_test_snapshot_failure()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.task_id = %s::uuid THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'injected snapshot failure';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER credentialregistry_test_snapshot_failure
BEFORE INSERT ON runtime_task_snapshot
FOR EACH ROW EXECUTE FUNCTION credentialregistry_test_snapshot_failure();`, quoteLiteral(taskID))); err != nil {
			t.Fatalf("install failure trigger: %v", err)
		}
		if _, err := resolver.Resolve(ctx, registryRequest(taskID)); !errors.Is(err, ErrRegistryUnavailable) {
			t.Fatalf("Resolve() error=%v, want registry unavailable", err)
		}
		assertTaskState(t, ctx, pool, taskID, "queued", 0, 0)
		if _, err := pool.Exec(ctx, `DROP TRIGGER credentialregistry_test_snapshot_failure ON runtime_task_snapshot`); err != nil {
			t.Fatalf("drop failure trigger: %v", err)
		}
	})

	t.Run("stale generation fails before mutation", func(t *testing.T) {
		taskID := "71000000-0000-4000-8001-000000000003"
		insertTask(t, ctx, pool, taskID)
		request := registryRequest(taskID)
		request.ExpectedCatalogGeneration++
		if _, err := resolver.Resolve(ctx, request); !errors.Is(err, ErrGenerationConflict) {
			t.Fatalf("Resolve() error=%v, want generation conflict", err)
		}
		assertTaskState(t, ctx, pool, taskID, "queued", 0, 0)
	})

	t.Run("concurrent claims are bounded and exact retry is idempotent", func(t *testing.T) {
		const workers, limit = 24, 3
		if _, err := pool.Exec(ctx, `UPDATE runtime_binding SET max_concurrent_tasks = $1 WHERE id = $2`, limit, testBindingID); err != nil {
			t.Fatalf("set concurrency: %v", err)
		}
		requests := make([]Request, workers)
		for i := range requests {
			taskID := fmt.Sprintf("71000000-0000-4000-8002-%012d", i)
			insertTask(t, ctx, pool, taskID)
			requests[i] = registryRequest(taskID)
		}

		start := make(chan struct{})
		results := make(chan struct {
			request Request
			err     error
		}, workers)
		var wait sync.WaitGroup
		for _, request := range requests {
			request := request
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				_, err := resolver.Resolve(context.Background(), request)
				results <- struct {
					request Request
					err     error
				}{request: request, err: err}
			}()
		}
		close(start)
		wait.Wait()
		close(results)

		var successful []Request
		capacityFailures := 0
		for result := range results {
			switch {
			case result.err == nil:
				successful = append(successful, result.request)
			case errors.Is(result.err, ErrCapacityExhausted):
				capacityFailures++
			default:
				t.Errorf("unexpected concurrent result: %v", result.err)
			}
		}
		if len(successful) != limit || capacityFailures != workers-limit {
			t.Fatalf("success=%d capacity_failures=%d", len(successful), capacityFailures)
		}
		assertBindingCount(t, ctx, pool, limit)

		if _, err := resolver.Resolve(ctx, successful[0]); err != nil {
			t.Fatalf("exact retry: %v", err)
		}
		assertBindingCount(t, ctx, pool, limit)

		if _, err := pool.Exec(ctx, `UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id = $1`, successful[0].TaskID); err != nil {
			t.Fatalf("complete task: %v", err)
		}
		release := ReleaseRequest{
			TaskID: successful[0].TaskID, BindingID: testBindingID,
			BindingGeneration: 1, CatalogGeneration: 1,
		}
		if err := resolver.Release(ctx, release); err != nil {
			t.Fatalf("Release(): %v", err)
		}
		if err := resolver.Release(ctx, release); err != nil {
			t.Fatalf("idempotent Release(): %v", err)
		}
		assertBindingCount(t, ctx, pool, limit-1)

		stale := release
		stale.BindingGeneration++
		if err := resolver.Release(ctx, stale); !errors.Is(err, ErrGenerationConflict) {
			t.Fatalf("stale Release() error=%v", err)
		}
		assertBindingCount(t, ctx, pool, limit-1)
	})

	t.Run("revoked replacement generation fails closed", func(t *testing.T) {
		before := bindingCount(t, ctx, pool)
		if _, err := pool.Exec(ctx, `
INSERT INTO credential_home_catalog_generation (
    id, catalog_id, previous_generation, generation, scan_kind, counters,
    catalog_digest, started_at
) VALUES (
    '71000000-0000-4000-8003-000000000001', $1, 1, 2, 'requested', '{}', repeat('6', 64), now()
);
INSERT INTO credential_home_catalog_entry (
    id, generation_id, catalog_id, generation, home_ref, provider, approved, state,
    first_seen_at, last_seen_at, last_full_scan_at, health_watermark, ttl, retention_deadline
) VALUES (
    '71000000-0000-4000-8003-000000000002',
    '71000000-0000-4000-8003-000000000001', $1, 2, $2, 'codex', false, 'healthy',
    now(), now(), now(), now(), interval '1 hour', now() + interval '1 day'
);
UPDATE credential_home_catalog SET generation = 2 WHERE id = $1`, testCatalogID, testHomeRef); err != nil {
			t.Fatalf("publish revoked generation: %v", err)
		}
		taskID := "71000000-0000-4000-8003-000000000003"
		insertTask(t, ctx, pool, taskID)
		request := registryRequest(taskID)
		request.ExpectedCatalogGeneration = 2
		if _, err := resolver.Resolve(ctx, request); !errors.Is(err, ErrGenerationConflict) {
			t.Fatalf("Resolve() error=%v, want generation conflict", err)
		}
		assertTaskState(t, ctx, pool, taskID, "queued", before, 0)
	})
}

func newTemporaryRegistryDatabase(t *testing.T, ctx context.Context, adminURL string) *pgxpool.Pool {
	t.Helper()
	config, err := pgx.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse admin URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}
	databaseName := fmt.Sprintf("credentialregistry_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create temporary database: %v", err)
	}
	_ = admin.Close(ctx)

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cleanupConfig, err := pgx.ParseConfig(adminURL)
		if err != nil {
			return
		}
		cleanup, err := pgx.ConnectConfig(cleanupCtx, cleanupConfig)
		if err != nil {
			return
		}
		defer cleanup.Close(cleanupCtx)
		_, _ = cleanup.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)")
	})

	databaseConfig, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	databaseConfig.ConnConfig.Database = databaseName
	databaseConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("connect temporary database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func applyRegistryMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	base := `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE "user" (id UUID PRIMARY KEY);
CREATE TABLE workspace (id UUID PRIMARY KEY);
CREATE TABLE runtime_standard (
    id UUID PRIMARY KEY, active_version_id UUID
);
CREATE TABLE runtime_standard_version (
    id UUID PRIMARY KEY, standard_id UUID NOT NULL,
    CONSTRAINT runtime_standard_version_standard_id_key UNIQUE (standard_id, id)
);
ALTER TABLE runtime_standard ADD CONSTRAINT runtime_standard_active_version_fkey
    FOREIGN KEY (id, active_version_id)
    REFERENCES runtime_standard_version(standard_id, id) ON DELETE RESTRICT;
CREATE TABLE runtime_session (
    id UUID PRIMARY KEY, standard_id UUID NOT NULL REFERENCES runtime_standard(id),
    provider TEXT NOT NULL
);
CREATE TABLE runtime_session_enrollment (
    id UUID PRIMARY KEY, session_id UUID NOT NULL, workspace_id UUID NOT NULL,
    runtime_id UUID NOT NULL, agent_id UUID NOT NULL
);
CREATE TABLE agent_task_queue (
    id UUID PRIMARY KEY, agent_id UUID NOT NULL, runtime_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued','dispatched','running','waiting_local_directory','completed','failed','cancelled')),
    dispatched_at TIMESTAMPTZ, completed_at TIMESTAMPTZ
);
CREATE FUNCTION reject_runtime_manager_immutable_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'immutable row';
END
$$;`
	if _, err := pool.Exec(ctx, base); err != nil {
		t.Fatalf("create migration prerequisites: %v", err)
	}
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	migrationsDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	for _, version := range []string{
		"132_credential_home_catalog",
		"133_runtime_bindings",
		"134_runtime_configuration_snapshots",
	} {
		migration, err := os.ReadFile(filepath.Join(migrationsDir, version+".up.sql"))
		if err != nil {
			t.Fatalf("read migration %s: %v", version, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", version, err)
		}
	}
}

func seedRegistry(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	seed := `
INSERT INTO "user" (id) VALUES ($1);
INSERT INTO workspace (id) VALUES ($2);
INSERT INTO runtime_standard_version (id, standard_id) VALUES ($6, $5);
INSERT INTO runtime_standard (id, active_version_id) VALUES ($5, $6);
INSERT INTO runtime_session (id, standard_id, provider) VALUES ($7, $5, 'codex');
INSERT INTO runtime_session_enrollment (id, session_id, workspace_id, runtime_id, agent_id)
VALUES ($8, $7, $2, $3, $4);
INSERT INTO credential_home_catalog (id, workspace_id, daemon_id, generation)
VALUES ($9, $2, 'test-daemon', 1);
INSERT INTO credential_home_catalog_generation (
    id, catalog_id, previous_generation, generation, scan_kind, counters,
    catalog_digest, started_at
) VALUES ($10, $9, 0, 1, 'startup', '{}', repeat('1', 64), now());
INSERT INTO credential_home_catalog_entry (
    id, generation_id, catalog_id, generation, home_ref, provider, approved, state,
    first_seen_at, last_seen_at, last_full_scan_at, health_watermark, ttl, retention_deadline
) VALUES (
    $11, $10, $9, 1, $12, 'codex', true, 'healthy',
    now(), now(), now(), now(), interval '1 hour', now() + interval '1 day'
);
INSERT INTO runtime_binding (
    id, enrollment_id, session_id, workspace_id, runtime_id, agent_id,
    transport_binding, max_concurrent_tasks, created_by
) VALUES ($13, $8, $7, $2, $3, $4, 'native_credential_home', 1, $1);
INSERT INTO runtime_home_assignment (
    id, binding_id, workspace_id, catalog_id, catalog_entry_id,
    catalog_generation, home_ref, binding_generation, assigned_by
) VALUES ($14, $13, $2, $9, $11, 1, $12, 1, $1);
INSERT INTO runtime_configuration_version (
    id, binding_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason
) VALUES ($15, $13, 1, '{}', repeat('2', 64), 'restart', $1, 'test');
UPDATE runtime_binding
SET active_configuration_version_id = $15,
    effective_configuration_digest = repeat('3', 64)
WHERE id = $13;
INSERT INTO runtime_configuration_activation (
    binding_id, new_version_id, binding_generation,
    effective_configuration_digest, capability_digest, apply_class,
    actor_id, request_id, reason
) VALUES ($13, $15, 1, repeat('3', 64), repeat('4', 64), 'restart', $1, 'test-activation', 'test');`
	if _, err := pool.Exec(ctx, seed,
		testOwnerID, testWorkspaceID, testRuntimeID, testAgentID, testStandardID,
		testStdVersion, testSessionID, testEnrollment, testCatalogID, testCatalogGen,
		testEntryID, testHomeRef, testBindingID, testAssignment, testConfig,
	); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
}

func insertTask(t *testing.T, ctx context.Context, pool *pgxpool.Pool, taskID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO agent_task_queue (id, agent_id, runtime_id) VALUES ($1, $2, $3)`, taskID, testAgentID, testRuntimeID); err != nil {
		t.Fatalf("insert task: %v", err)
	}
}

func registryRequest(taskID string) Request {
	return Request{
		TaskID: taskID, BindingID: testBindingID, AgentID: testAgentID,
		WorkspaceID: testWorkspaceID, Provider: "codex",
		ExpectedBindingGeneration: 1, ExpectedCatalogGeneration: 1,
	}
}

func assertTaskState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, taskID, wantStatus string, wantActive, wantSnapshots int) {
	t.Helper()
	var status string
	var active, snapshots int
	if err := pool.QueryRow(ctx, `
SELECT t.status, b.active_task_count,
       (SELECT count(*)::integer FROM runtime_task_snapshot WHERE task_id = t.id)
FROM agent_task_queue t CROSS JOIN runtime_binding b
WHERE t.id = $1 AND b.id = $2`, taskID, testBindingID).Scan(&status, &active, &snapshots); err != nil {
		t.Fatalf("inspect task state: %v", err)
	}
	if status != wantStatus || active != wantActive || snapshots != wantSnapshots {
		t.Fatalf("status=%s active=%d snapshots=%d, want %s/%d/%d", status, active, snapshots, wantStatus, wantActive, wantSnapshots)
	}
}

func assertBindingCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int) {
	t.Helper()
	if count := bindingCount(t, ctx, pool); count != want {
		t.Fatalf("active_task_count=%d, want %d", count, want)
	}
}

func bindingCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT active_task_count FROM runtime_binding WHERE id = $1`, testBindingID).Scan(&count); err != nil {
		t.Fatalf("read active count: %v", err)
	}
	return count
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
