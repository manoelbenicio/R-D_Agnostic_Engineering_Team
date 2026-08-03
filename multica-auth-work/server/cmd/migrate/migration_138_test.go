package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMigration138ImmutableAttestationUpdateDeleteRejected55006(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	acceptMigration138(t, ctx, f, migration138ProbeID, "ready", "ready", time.Now())

	for _, statement := range []string{
		`UPDATE runtime_credential_readiness_attestation
		 SET expires_at = expires_at + interval '1 second'`,
		`DELETE FROM runtime_credential_readiness_attestation`,
		`UPDATE runtime_credential_readiness_probe_request
		 SET request_digest = repeat('d', 64)`,
		`DELETE FROM runtime_credential_readiness_probe_result`,
	} {
		expectMigration138SQLState(t, ctx, f, statement, "55006")
	}
}

func TestMigration138ExactReplayReturnsOriginalWithoutExpiryExtension(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	observed := time.Now().UTC().Truncate(time.Microsecond)
	firstID, firstAccepted, firstExpires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "ready", "ready", observed,
	)
	time.Sleep(10 * time.Millisecond)
	secondID, secondAccepted, secondExpires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "ready", "ready", observed,
	)
	if firstID != secondID || !firstAccepted.Equal(secondAccepted) ||
		!firstExpires.Equal(secondExpires) {
		t.Fatalf("exact replay changed immutable result: first=%s/%s/%s second=%s/%s/%s",
			firstID, firstAccepted, firstExpires, secondID, secondAccepted, secondExpires)
	}
}

func TestMigration138ChangedReplayRejected(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	observed := time.Now().UTC().Truncate(time.Microsecond)
	acceptMigration138(t, ctx, f, migration138ProbeID, "ready", "ready", observed)

	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	var id string
	err := tx.QueryRow(ctx, migration138AcceptOrReplaySQL,
		migration138ProbeID, strings.Repeat("b", 64), "unready",
		"probe_timeout", observed,
	).Scan(&id, new(time.Time), new(time.Time))
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("changed replay err=%v, want no rows", err)
	}
}

func TestMigration138ObservationAndServerExpiryBoundaries(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)

	var issued time.Time
	if err := migration138QueryRow(t, ctx, f,
		`SELECT issued_at FROM runtime_credential_readiness_probe_request
		 WHERE probe_request_id = $1`, migration138ProbeID,
	).Scan(&issued); err != nil {
		t.Fatal(err)
	}
	expectMigration138AcceptRejected(t, ctx, f, issued.Add(-time.Microsecond))
	expectMigration138AcceptRejected(t, ctx, f, time.Now().Add(time.Minute))

	_, accepted, expires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "ready", "ready", issued,
	)
	if !expires.Equal(accepted.Add(5 * time.Minute)) {
		t.Fatalf("server expiry=%s, want accepted+300s=%s",
			expires, accepted.Add(5*time.Minute))
	}
}

func TestMigration138CurrentGenerationAndDaemonBootRaceFailsClosed(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)

	expectMigration138SQLState(t, ctx, f, fmt.Sprintf(`
		INSERT INTO runtime_credential_readiness_probe_request (
			probe_request_id, request_digest, task_id, home_epoch, workspace_id,
			agent_id, runtime_id, runtime_session_id, daemon_id, daemon_boot_id,
			provider, transport_binding, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_entry_id, catalog_generation,
			home_ref, lifetime_id, acquisition_request_id
		) SELECT
			'%s', repeat('a', 64), task_id, home_epoch, workspace_id, agent_id,
			runtime_id, runtime_session_id, daemon_id,
			'00000000-0000-0000-0000-000000000099', provider,
			transport_binding, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_entry_id,
			catalog_generation, home_ref, lifetime_id, acquisition_request_id
		FROM runtime_task_home_epoch WHERE task_id = '%s'`,
		migration138ProbeID, migration138TaskID), "23514")

	insertMigration138Request(t, ctx, f, migration138ProbeID)
	execMigration138(t, ctx, f, `
		UPDATE runtime_binding SET state = 'draining'
		WHERE id = '00000000-0000-0000-0000-000000000013'`)
	expectMigration138AcceptRejected(t, ctx, f, time.Now())
}

func TestMigration138RollbackRestoresMigration137State(t *testing.T) {
	f, ctx := setupMigration138(t)
	down := f.opts()
	down.Direction = "down"
	down.Files = scopedRealMigrations(t, f.schema,
		"138_runtime_credential_readiness_probes.down.sql")
	if err := runMigrations(ctx, f.pool, down); err != nil {
		t.Fatalf("empty rollback migration 138: %v", err)
	}
	for _, relation := range []string{
		"runtime_credential_readiness_probe_request",
		"runtime_credential_readiness_probe_result",
	} {
		var exists bool
		if err := f.pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`,
			f.schema+"."+relation).Scan(&exists); err != nil || exists {
			t.Fatalf("post-down relation %s exists=%v err=%v", relation, exists, err)
		}
	}
	seedMigration136Authority(t, ctx, f)
	seedMigration137RetirementState(t, ctx, f)
}

func TestMigration138SchemaRejectsPathSecretTokenColumns(t *testing.T) {
	f, ctx := setupMigration138(t)
	rows, err := f.pool.Query(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name IN (
		    'runtime_credential_readiness_probe_request',
		    'runtime_credential_readiness_probe_result'
		  )`, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			"path", "account", "credential", "token", "secret", "slot", "body",
		} {
			if strings.Contains(column, forbidden) {
				t.Fatalf("%s.%s contains forbidden material marker %q",
					table, column, forbidden)
			}
		}
	}
}

const (
	migration138TaskID  = "00000000-0000-0000-0000-000000000008"
	migration138ProbeID = "00000000-0000-0000-0000-000000000090"
)

func setupMigration138(t *testing.T) (*fixture, context.Context) {
	t.Helper()
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	createMigration136Prerequisites(t, ctx, f)
	opts := f.opts()
	opts.Files = scopedRealMigrations(t, f.schema,
		"132_credential_home_catalog.up.sql",
		"133_runtime_bindings.up.sql",
		"134_runtime_configuration_snapshots.up.sql",
		"136_runtime_home_lifetime_readiness.up.sql",
		"137_native_rotation_epoch_recovery.up.sql",
		"138_runtime_credential_readiness_probes.up.sql",
	)
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply migrations 132-138: %v", err)
	}
	return f, ctx
}

func seedMigration138CurrentNativeEpoch(
	t *testing.T, ctx context.Context, f *fixture,
) {
	t.Helper()
	seedMigration136Authority(t, ctx, f)
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return execMigration137Batch(ctx, tx, `
			INSERT INTO runtime_task_home_epoch (
				task_id, home_epoch, workspace_id, agent_id, runtime_id,
				runtime_session_id, daemon_id, daemon_boot_id, provider,
				transport_binding, runtime_binding_id, binding_generation,
				home_assignment_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, lifetime_id, acquisition_request_id
			) VALUES (
				'00000000-0000-0000-0000-000000000008', 1,
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'daemon-a', '00000000-0000-0000-0000-000000000018',
				'codex', 'native_credential_home',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000011', 1,
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000054',
				'00000000-0000-0000-0000-000000000055'
			);
			INSERT INTO runtime_home_lifetime (
				id, acquisition_request_id, task_id, home_epoch, agent_id,
				runtime_id, runtime_session_id, runtime_binding_id,
				binding_generation, home_assignment_id, workspace_id, daemon_id,
				catalog_id, catalog_generation, home_ref, daemon_boot_id,
				provider, transport_binding, state
			) VALUES (
				'00000000-0000-0000-0000-000000000054',
				'00000000-0000-0000-0000-000000000055',
				'00000000-0000-0000-0000-000000000008', 1,
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000001', 'daemon-a',
				'00000000-0000-0000-0000-000000000009', 1,
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000018',
				'codex', 'native_credential_home', 'pending_local'
			);
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 1, 'pending_local',
				'00000000-0000-0000-0000-000000000077', 'reserved', now()
			);
			UPDATE runtime_home_lifetime
			SET state = 'acquired', state_version = 2
			WHERE id = '00000000-0000-0000-0000-000000000054';
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 2, 'acquired',
				'00000000-0000-0000-0000-000000000078', 'acquired', now()
			);
			UPDATE runtime_home_lifetime
			SET state = 'process_started', state_version = 3,
				process_started_at = now(), process_identity_digest = repeat('a', 64)
			WHERE id = '00000000-0000-0000-0000-000000000054';
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 3, 'process_started',
				'00000000-0000-0000-0000-000000000079', 'started', now()
			)
		`)
	})
}

func insertMigration138Request(
	t *testing.T, ctx context.Context, f *fixture, probeID string,
) {
	t.Helper()
	execMigration138(t, ctx, f, fmt.Sprintf(`
		INSERT INTO runtime_credential_readiness_probe_request (
			probe_request_id, request_digest, task_id, home_epoch, workspace_id,
			agent_id, runtime_id, runtime_session_id, daemon_id, daemon_boot_id,
			provider, transport_binding, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_entry_id, catalog_generation,
			home_ref, lifetime_id, acquisition_request_id
		) SELECT
			'%s', repeat('a', 64), task_id, home_epoch, workspace_id, agent_id,
			runtime_id, runtime_session_id, daemon_id, daemon_boot_id, provider,
			transport_binding, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_entry_id,
			catalog_generation, home_ref, lifetime_id, acquisition_request_id
		FROM runtime_task_home_epoch WHERE task_id = '%s' AND home_epoch = 1`,
		probeID, migration138TaskID))
}

const migration138AcceptOrReplaySQL = `
WITH inserted_attestation AS (
	INSERT INTO runtime_credential_readiness_attestation (
		probe_request_id, workspace_id, agent_id, runtime_id,
		runtime_session_id, runtime_binding_id, binding_generation,
		home_assignment_id, catalog_id, catalog_generation, home_ref,
		provider, daemon_id, daemon_boot_id, state, reason_code,
		observed_at, expires_at
	)
	SELECT
		r.probe_request_id, r.workspace_id, r.agent_id, r.runtime_id,
		r.runtime_session_id, r.runtime_binding_id, r.binding_generation,
		r.home_assignment_id, r.catalog_id, r.catalog_generation, r.home_ref,
		r.provider, r.daemon_id, r.daemon_boot_id, $3, $4, $5,
		transaction_timestamp() + interval '5 minutes'
	FROM runtime_credential_readiness_probe_request r
	WHERE r.probe_request_id = $1
	  AND r.request_digest = repeat('a', 64)
	  AND $5 >= r.issued_at
	  AND $5 <= transaction_timestamp()
	  AND transaction_timestamp() <= r.request_expires_at
	ON CONFLICT (probe_request_id) DO NOTHING
	RETURNING *
),
inserted_result AS (
	INSERT INTO runtime_credential_readiness_probe_result (
		probe_request_id, attestation_id, result_digest, state, reason_code,
		probe_observed_at
	)
	SELECT probe_request_id, id, $2, state, reason_code, observed_at
	FROM inserted_attestation
	RETURNING *
)
SELECT id, accepted_at, expires_at FROM inserted_result
UNION ALL
SELECT x.id, x.accepted_at, x.expires_at
FROM runtime_credential_readiness_probe_result x
WHERE x.probe_request_id = $1
  AND x.result_digest = $2
  AND x.state = $3
  AND x.reason_code = $4
  AND x.probe_observed_at = $5
  AND NOT EXISTS (SELECT 1 FROM inserted_result)
LIMIT 1`

func acceptMigration138(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	probeID, state, reason string,
	observed time.Time,
) (string, time.Time, time.Time) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	var id string
	var accepted, expires time.Time
	err := tx.QueryRow(ctx, migration138AcceptOrReplaySQL,
		probeID, strings.Repeat("b", 64), state, reason, observed,
	).Scan(&id, &accepted, &expires)
	if err != nil {
		t.Fatalf("accept/replay probe result: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit probe result: %v", err)
	}
	return id, accepted, expires
}

func expectMigration138AcceptRejected(
	t *testing.T, ctx context.Context, f *fixture, observed time.Time,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	var id string
	err := tx.QueryRow(ctx, migration138AcceptOrReplaySQL,
		migration138ProbeID, strings.Repeat("b", 64), "ready", "ready", observed,
	).Scan(&id, new(time.Time), new(time.Time))
	if err == nil {
		t.Fatal("invalid probe acceptance succeeded")
	}
}

func beginMigration138Tx(
	t *testing.T, ctx context.Context, f *fixture,
) pgx.Tx {
	t.Helper()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s",
		pgx.Identifier{f.schema}.Sanitize())); err != nil {
		tx.Rollback(ctx)
		t.Fatal(err)
	}
	return tx
}

func execMigration138(
	t *testing.T, ctx context.Context, f *fixture, statement string,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, statement); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func migration138QueryRow(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	statement string,
	args ...any,
) pgx.Row {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx.QueryRow(ctx, statement, args...)
}

func expectMigration138SQLState(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	statement, code string,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	_, err := tx.Exec(ctx, statement)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("statement error=%v, want SQLSTATE %s", err, code)
	}
}
