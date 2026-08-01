package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRuntimeManagerReservationPrimitives(t *testing.T) {
	databaseURL := os.Getenv("SPE6_RUNTIME_MANAGER_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("SPE6_RUNTIME_MANAGER_DATABASE_URL is not set; run pkg/db/testdata/run_runtime_manager_gate.sh")
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	fixture, err := os.ReadFile("../testdata/runtime_manager_fixture.sql")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	committedFixture := strings.Replace(string(fixture), "ROLLBACK;", "COMMIT;", 1)
	if _, err := pool.Exec(ctx, committedFixture); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)

	taskID := mustGateUUID(t, "60000000-0000-4000-8000-000000000011")
	agentID := mustGateUUID(t, "60000000-0000-4000-8000-000000000004")
	runtimeID := mustGateUUID(t, "60000000-0000-4000-8000-000000000003")
	bindingID := mustGateUUID(t, "60000000-0000-4000-8000-00000000000d")
	workspaceID := mustGateUUID(t, "60000000-0000-4000-8000-000000000002")
	assignmentID := mustGateUUID(t, "60000000-0000-4000-8000-00000000000e")
	ownerID := mustGateUUID(t, "60000000-0000-4000-8000-000000000001")
	catalogID := mustGateUUID(t, "60000000-0000-4000-8000-000000000009")

	lifecycle, err := q.LoadCredentialHomeCatalogLifecycle(ctx, LoadCredentialHomeCatalogLifecycleParams{
		CatalogID: catalogID, WorkspaceID: workspaceID, Provider: "codex",
	})
	if err != nil {
		t.Fatalf("load lifecycle: %v", err)
	}
	if len(lifecycle) != 1 || lifecycle[0].State != "retired" ||
		!lifecycle[0].ReasonCode.Valid || lifecycle[0].ReasonCode.String != "tombstoned" ||
		lifecycle[0].ActiveRefs != 0 || lifecycle[0].Generation != 1 {
		t.Fatalf("loaded lifecycle = %+v, want durable generation-1 tombstone", lifecycle)
	}

	if _, err := tx.Exec(ctx, "SAVEPOINT lifecycle_store_probe"); err != nil {
		t.Fatalf("create lifecycle savepoint: %v", err)
	}
	if _, err := q.BeginCredentialHomeCatalogReconciliation(ctx, BeginCredentialHomeCatalogReconciliationParams{
		ID: catalogID, WorkspaceID: workspaceID, ExpectedGeneration: 1,
	}); err != nil {
		t.Fatalf("begin lifecycle reconciliation: %v", err)
	}
	if _, err := q.AdvanceCredentialHomeCatalogLifecycleGeneration(ctx, AdvanceCredentialHomeCatalogLifecycleGenerationParams{
		LifecycleGeneration:         2,
		CatalogID:                   catalogID,
		WorkspaceID:                 workspaceID,
		ExpectedPublishedGeneration: 1,
		ExpectedLifecycleGeneration: 1,
	}); err != nil {
		t.Fatalf("advance lifecycle generation: %v", err)
	}
	if _, err := q.SaveCredentialHomeCatalogLifecycleRecord(ctx, SaveCredentialHomeCatalogLifecycleRecordParams{
		HomeRef:             mustGateUUID(t, "60000000-0000-5000-8000-00000000001d"),
		NameRef:             "name_" + strings.Repeat("e", 43),
		Provider:            "codex",
		LifecycleState:      "draining",
		ActiveRefs:          2,
		LifecycleGeneration: 2,
		UpdatedAt:           pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		RetentionDeadline:   pgtype.Timestamptz{Time: time.Now().UTC().Add(30 * 24 * time.Hour), Valid: true},
		CatalogID:           catalogID,
		WorkspaceID:         workspaceID,
	}); err != nil {
		t.Fatalf("save lifecycle record: %v", err)
	}
	if _, err := q.CreateCredentialHomeCatalogGeneration(ctx, CreateCredentialHomeCatalogGenerationParams{
		CatalogID: catalogID, PreviousGeneration: 1, Generation: 2,
		ScanKind: "requested", Counters: []byte(`{"draining":1}`),
		CatalogDigest: strings.Repeat("6", 64),
		StartedAt:     pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		PublishedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}); err != nil {
		t.Fatalf("create catalog generation: %v", err)
	}
	published, err := q.PublishCredentialHomeCatalogGeneration(ctx, PublishCredentialHomeCatalogGenerationParams{
		Generation: 2, Watermark: "normal", CatalogID: catalogID, PreviousGeneration: 1,
	})
	if err != nil {
		t.Fatalf("publish catalog generation after lifecycle: %v", err)
	}
	if published.Generation != 2 || published.LifecycleGeneration != 2 {
		t.Fatalf("published generations = %d/%d, want 2/2", published.Generation, published.LifecycleGeneration)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT lifecycle_store_probe"); err != nil {
		t.Fatalf("rollback successful lifecycle probe: %v", err)
	}

	// A durable tombstone cannot be resurrected. PostgreSQL aborts to the
	// savepoint, proving the speculative lifecycle generation/publication is
	// rollback-safe when SaveLifecycle encounters a conflicting record.
	if _, err := tx.Exec(ctx, "SAVEPOINT lifecycle_store_conflict"); err != nil {
		t.Fatalf("create lifecycle conflict savepoint: %v", err)
	}
	if _, err := q.BeginCredentialHomeCatalogReconciliation(ctx, BeginCredentialHomeCatalogReconciliationParams{
		ID: catalogID, WorkspaceID: workspaceID, ExpectedGeneration: 1,
	}); err != nil {
		t.Fatalf("begin conflicting lifecycle reconciliation: %v", err)
	}
	if _, err := q.AdvanceCredentialHomeCatalogLifecycleGeneration(ctx, AdvanceCredentialHomeCatalogLifecycleGenerationParams{
		LifecycleGeneration:         2,
		CatalogID:                   catalogID,
		WorkspaceID:                 workspaceID,
		ExpectedPublishedGeneration: 1,
		ExpectedLifecycleGeneration: 1,
	}); err != nil {
		t.Fatalf("advance conflicting lifecycle generation: %v", err)
	}
	if _, err := q.SaveCredentialHomeCatalogLifecycleRecord(ctx, SaveCredentialHomeCatalogLifecycleRecordParams{
		HomeRef:             mustGateUUID(t, "60000000-0000-5000-8000-00000000001c"),
		NameRef:             "name_" + strings.Repeat("n", 43),
		Provider:            "codex",
		LifecycleState:      "missing",
		ActiveRefs:          0,
		LifecycleGeneration: 2,
		UpdatedAt:           pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		RetentionDeadline:   pgtype.Timestamptz{Time: time.Now().UTC().Add(30 * 24 * time.Hour), Valid: true},
		CatalogID:           catalogID,
		WorkspaceID:         workspaceID,
	}); err == nil {
		t.Fatal("tombstone resurrection unexpectedly succeeded")
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT lifecycle_store_conflict"); err != nil {
		t.Fatalf("rollback lifecycle conflict: %v", err)
	}

	if _, err := q.LockAgentTaskForRuntimeSnapshot(ctx, LockAgentTaskForRuntimeSnapshotParams{
		TaskID: taskID, AgentID: agentID, RuntimeID: runtimeID,
	}); err != nil {
		t.Fatalf("lock task: %v", err)
	}
	snapshot, err := q.GetRuntimeTaskSnapshot(ctx, taskID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}

	exact := CreateRuntimeTaskSnapshotParams{
		TaskID:                        snapshot.TaskID,
		RuntimeSessionID:              snapshot.RuntimeSessionID,
		RuntimeID:                     snapshot.RuntimeID,
		AgentID:                       snapshot.AgentID,
		WorkspaceID:                   snapshot.WorkspaceID,
		RuntimeStandardVersionID:      snapshot.RuntimeStandardVersionID,
		RuntimeConfigurationVersionID: snapshot.RuntimeConfigurationVersionID,
		EffectiveConfigurationDigest:  snapshot.EffectiveConfigurationDigest,
		RuntimeBindingID:              snapshot.RuntimeBindingID,
		BindingGeneration:             snapshot.BindingGeneration,
		TransportBinding:              snapshot.TransportBinding,
		HomeAssignmentID:              snapshot.HomeAssignmentID,
		HomeRef:                       snapshot.HomeRef,
		CatalogGeneration:             snapshot.CatalogGeneration,
		CapabilityDigest:              snapshot.CapabilityDigest,
	}
	if _, err := q.CreateRuntimeTaskSnapshot(ctx, exact); err != nil {
		t.Fatalf("exact snapshot retry: %v", err)
	}

	conflict := exact
	conflict.EffectiveConfigurationDigest = strings.Repeat("9", 64)
	if _, err := q.CreateRuntimeTaskSnapshot(ctx, conflict); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("conflicting duplicate error = %v, want pgx.ErrNoRows", err)
	}

	if _, err := q.ReserveNativeRuntimeBindingCapacity(ctx, ReserveNativeRuntimeBindingCapacityParams{
		BindingID:                 bindingID,
		WorkspaceID:               workspaceID,
		ExpectedBindingGeneration: 1,
		HomeAssignmentID:          assignmentID,
		ExpectedCatalogGeneration: 1,
		HealthFreshAfter: pgtype.Timestamptz{
			Time: time.Now().Add(-time.Hour), Valid: true,
		},
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("capacity exhaustion error = %v, want pgx.ErrNoRows", err)
	}

	if _, err := tx.Exec(ctx,
		"UPDATE agent_task_queue SET status = 'completed', completed_at = now() WHERE id = $1",
		taskID,
	); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	released, err := q.ReleaseRuntimeTaskSnapshotCapacity(ctx, ReleaseRuntimeTaskSnapshotCapacityParams{
		ReleasedBy: ownerID, ReasonCode: "fixture_terminal",
		TaskID: taskID, RuntimeBindingID: bindingID,
	})
	if err != nil {
		t.Fatalf("release capacity: %v", err)
	}
	if released.ActiveTaskCount != 0 {
		t.Fatalf("active task count = %d, want 0", released.ActiveTaskCount)
	}
	if _, err := q.ReleaseRuntimeTaskSnapshotCapacity(ctx, ReleaseRuntimeTaskSnapshotCapacityParams{
		ReleasedBy: ownerID, ReasonCode: "fixture_terminal",
		TaskID: taskID, RuntimeBindingID: bindingID,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate release error = %v, want pgx.ErrNoRows", err)
	}
}

func mustGateUUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		t.Fatalf("parse UUID %q: %v", value, err)
	}
	return id
}
