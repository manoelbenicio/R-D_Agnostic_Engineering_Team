package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

	// F1: rollback must restore the exact Migration-136 trigger semantics.
	seedMigration136Authority(t, ctx, f)
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, migration136LifetimeInsert); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO runtime_home_lifetime_event (
				lifetime_id, state_version, state, transition_request_id, reason_code
			) VALUES (
				'00000000-0000-0000-0000-000000000016', 1, 'pending_local',
				'00000000-0000-0000-0000-000000000019', 'reserved'
			)`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE runtime_home_lifetime
			SET state = 'acquired', state_version = 2
			WHERE id = '00000000-0000-0000-0000-000000000016'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO runtime_home_lifetime_event (
				lifetime_id, state_version, state, transition_request_id, reason_code
			) VALUES (
				'00000000-0000-0000-0000-000000000016', 2, 'acquired',
				'00000000-0000-0000-0000-000000000022', 'local_gate_acquired'
			)`)
		return err
	})
}

func TestMigration137NativeRetirementStateMachine(t *testing.T) {
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
	seedMigration136Authority(t, ctx, f)
	seedMigration137RetirementState(t, ctx, f)

	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000060",
			"scheduled", "accepted", 1, 2,
			"00000000-0000-0000-0000-000000000062")
	})
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000060",
			"accepted", "executing", 2, 3,
			"00000000-0000-0000-0000-000000000063")
	})

	// At most one scheduled/accepted/executing attempt may exist.
	expectMigration136ExecRejected(t, ctx, f, `
		INSERT INTO native_rotation_retirement_attempt (
			id, retirement_request_id, operation_id, task_id, home_epoch,
			lifetime_id, home_ref, daemon_id, daemon_boot_id,
			runtime_session_id, runtime_binding_id, binding_generation,
			scheduler_owner, attempt_number, state
		) VALUES (
			'00000000-0000-0000-0000-000000000064',
			'00000000-0000-0000-0000-000000000065',
			'00000000-0000-0000-0000-000000000058',
			'00000000-0000-0000-0000-000000000008', 1,
			'00000000-0000-0000-0000-000000000054',
			'00000000-0000-0000-0000-000000000012',
			'daemon-a', '00000000-0000-0000-0000-000000000018',
			'00000000-0000-0000-0000-000000000004',
			'00000000-0000-0000-0000-000000000013', 1,
			'test-scheduler', 2, 'scheduled'
		)`)

	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000060",
			"executing", "succeeded", 3, 4,
			"00000000-0000-0000-0000-000000000066")
	})
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000060",
			"executing", "succeeded", 3, 4,
			"00000000-0000-0000-0000-000000000067")
	})

	// A later attempt may retry only after the prior terminal failure; an
	// explicitly quarantined scheduled attempt is terminal and non-replayable.
	insertMigration137Attempt(t, ctx, f,
		"00000000-0000-0000-0000-000000000068",
		"00000000-0000-0000-0000-000000000069", 2)
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		if err := advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000068",
			"scheduled", "accepted", 1, 2,
			"00000000-0000-0000-0000-000000000070"); err != nil {
			return err
		}
		if err := advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000068",
			"accepted", "executing", 2, 3,
			"00000000-0000-0000-0000-000000000071"); err != nil {
			return err
		}
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000068",
			"executing", "retryable_failure", 3, 4,
			"00000000-0000-0000-0000-000000000072")
	})
	insertMigration137Attempt(t, ctx, f,
		"00000000-0000-0000-0000-000000000073",
		"00000000-0000-0000-0000-000000000074", 3)
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		if err := advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000073",
			"scheduled", "accepted", 1, 2,
			"00000000-0000-0000-0000-000000000075"); err != nil {
			return err
		}
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000073",
			"accepted", "quarantined", 2, 3,
			"00000000-0000-0000-0000-000000000076")
	})
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		return advanceMigration137Attempt(ctx, tx,
			"00000000-0000-0000-0000-000000000073",
			"accepted", "quarantined", 2, 3,
			"00000000-0000-0000-0000-000000000084")
	})
}

func seedMigration137RetirementState(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return execMigration137Batch(ctx, tx, `
			INSERT INTO credential_home_catalog_lifecycle (
				catalog_id, home_ref, provider, state, active_refs, generation,
				updated_at, retention_deadline
			) VALUES (
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000012',
				'codex', 'draining', 1, 1, now(), now() + interval '1 day'
			);
			INSERT INTO credential_home_catalog_entry (
				id, generation_id, catalog_id, generation, home_ref, provider,
				approved, state, first_seen_at, last_seen_at, last_full_scan_at,
				health_watermark, ttl_nanoseconds, retention_deadline
			) VALUES (
				'00000000-0000-0000-0000-000000000051',
				'00000000-0000-0000-0000-000000000010',
				'00000000-0000-0000-0000-000000000009', 1,
				'00000000-0000-0000-0000-000000000052', 'codex', true, 'healthy',
				now(), now(), now(), now(), 60000000000, now() + interval '1 day'
			);
			INSERT INTO credential_home_catalog_lifecycle (
				catalog_id, home_ref, provider, state, active_refs, generation,
				updated_at, retention_deadline
			) VALUES (
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000052',
				'codex', 'draining', 1, 1, now(), now() + interval '1 day'
			);
			INSERT INTO runtime_home_assignment (
				id, binding_id, workspace_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, binding_generation, state,
				assigned_by, reason_code
			) VALUES (
				'00000000-0000-0000-0000-000000000053',
				'00000000-0000-0000-0000-000000000013',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000051', 1,
				'00000000-0000-0000-0000-000000000052', 1, 'preparing',
				'00000000-0000-0000-0000-000000000002', 'test_rotation'
			);

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
				'daemon-a', '00000000-0000-0000-0000-000000000018', 'codex',
				'native_credential_home',
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
			);

			INSERT INTO runtime_task_home_epoch (
				task_id, home_epoch, workspace_id, agent_id, runtime_id,
				runtime_session_id, daemon_id, daemon_boot_id, provider,
				transport_binding, runtime_binding_id, binding_generation,
				home_assignment_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, lifetime_id, acquisition_request_id
			) VALUES (
				'00000000-0000-0000-0000-000000000008', 2,
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'daemon-a', '00000000-0000-0000-0000-000000000018', 'codex',
				'native_credential_home',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000053',
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000051', 1,
				'00000000-0000-0000-0000-000000000052',
				'00000000-0000-0000-0000-000000000056',
				'00000000-0000-0000-0000-000000000057'
			);
			INSERT INTO runtime_home_lifetime (
				id, acquisition_request_id, task_id, home_epoch, agent_id,
				runtime_id, runtime_session_id, runtime_binding_id,
				binding_generation, home_assignment_id, workspace_id, daemon_id,
				catalog_id, catalog_generation, home_ref, daemon_boot_id,
				provider, transport_binding, state
			) VALUES (
				'00000000-0000-0000-0000-000000000056',
				'00000000-0000-0000-0000-000000000057',
				'00000000-0000-0000-0000-000000000008', 2,
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000053',
				'00000000-0000-0000-0000-000000000001', 'daemon-a',
				'00000000-0000-0000-0000-000000000009', 1,
				'00000000-0000-0000-0000-000000000052',
				'00000000-0000-0000-0000-000000000018',
				'codex', 'native_credential_home', 'pending_local'
			);
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000056', 1, 'pending_local',
				'00000000-0000-0000-0000-000000000080', 'reserved', now()
			);

			INSERT INTO native_rotation_operation (
				id, operation_request_id, task_id, runtime_binding_id,
				current_home_epoch, current_home_ref, current_assignment_id,
				current_lifetime_id, target_home_epoch, target_home_ref,
				target_assignment_id, target_lifetime_id, state, reason_code
			) VALUES (
				'00000000-0000-0000-0000-000000000058',
				'00000000-0000-0000-0000-000000000059',
				'00000000-0000-0000-0000-000000000008',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000054', 2,
				'00000000-0000-0000-0000-000000000052',
				'00000000-0000-0000-0000-000000000053',
				'00000000-0000-0000-0000-000000000056',
				'candidate_reserved', 'test_rotation'
			);
			INSERT INTO native_rotation_operation_event VALUES (
				'00000000-0000-0000-0000-000000000058', 1,
				'candidate_reserved',
				'00000000-0000-0000-0000-000000000081',
				'test_rotation', now()
			);
			UPDATE native_rotation_operation
			SET state = 'candidate_prepared', state_version = 2
			WHERE id = '00000000-0000-0000-0000-000000000058';
			INSERT INTO native_rotation_operation_event VALUES (
				'00000000-0000-0000-0000-000000000058', 2,
				'candidate_prepared',
				'00000000-0000-0000-0000-000000000082',
				'test_rotation', now()
			);
			UPDATE native_rotation_operation
			SET state = 'committed_retirement_pending', state_version = 3
			WHERE id = '00000000-0000-0000-0000-000000000058';
			INSERT INTO native_rotation_operation_event VALUES (
				'00000000-0000-0000-0000-000000000058', 3,
				'committed_retirement_pending',
				'00000000-0000-0000-0000-000000000083',
				'test_rotation', now()
			);
		`)
	})
	insertMigration137Attempt(t, ctx, f,
		"00000000-0000-0000-0000-000000000060",
		"00000000-0000-0000-0000-000000000061", 1)
}

func insertMigration137Attempt(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	attemptID string,
	requestID string,
	number int16,
) {
	t.Helper()
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO native_rotation_retirement_attempt (
				id, retirement_request_id, operation_id, task_id, home_epoch,
				lifetime_id, home_ref, daemon_id, daemon_boot_id,
				runtime_session_id, runtime_binding_id, binding_generation,
				scheduler_owner, attempt_number, state
			) VALUES (
				$1, $2, '00000000-0000-0000-0000-000000000058',
				'00000000-0000-0000-0000-000000000008', 1,
				'00000000-0000-0000-0000-000000000054',
				'00000000-0000-0000-0000-000000000012',
				'daemon-a', '00000000-0000-0000-0000-000000000018',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000013', 1,
				'test-scheduler', $3, 'scheduled'
			)
		`, attemptID, requestID, number); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO native_rotation_retirement_attempt_event (
				attempt_id, state_version, state, transition_request_id, reason_code
			) VALUES ($1, 1, 'scheduled', gen_random_uuid(), 'scheduled')
		`, attemptID)
		return err
	})
}

func advanceMigration137Attempt(
	ctx context.Context,
	tx pgx.Tx,
	attemptID string,
	expected string,
	next string,
	expectedVersion int,
	nextVersion int,
	eventID string,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE native_rotation_retirement_attempt
		SET state = $2, state_version = $4,
			channel_binding_digest = CASE
				WHEN $2 IN (
					'accepted', 'executing', 'succeeded',
					'retryable_failure', 'quarantined'
				)
				THEN repeat('a', 64) ELSE channel_binding_digest END,
			request_body_digest = CASE
				WHEN $2 IN (
					'accepted', 'executing', 'succeeded',
					'retryable_failure', 'quarantined'
				)
				THEN repeat('b', 64) ELSE request_body_digest END
		WHERE id = $1 AND state = $3 AND state_version = $5
	`, attemptID, next, expected, nextVersion, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("attempt CAS affected %d rows", tag.RowsAffected())
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO native_rotation_retirement_attempt_event (
			attempt_id, state_version, state, transition_request_id, reason_code
		) VALUES ($1, $2, $3, $4, 'test_transition')
	`, attemptID, nextVersion, next, eventID)
	return err
}

func execMigration137Batch(ctx context.Context, tx pgx.Tx, sql string) error {
	for index, statement := range strings.Split(sql, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			firstLine := strings.SplitN(statement, "\n", 2)[0]
			return fmt.Errorf("seed statement %d %q: %w", index+1, firstLine, err)
		}
	}
	return nil
}
