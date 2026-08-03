package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestMigration136RuntimeHomeLifetimeReadiness(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createMigration136Prerequisites(t, ctx, f)
	base := scopedRealMigrations(t, f.schema,
		"132_credential_home_catalog.up.sql",
		"133_runtime_bindings.up.sql",
		"134_runtime_configuration_snapshots.up.sql",
	)
	opts := f.opts()
	opts.Files = base
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply real migrations 132-134: %v", err)
	}

	up := scopedRealMigrations(t, f.schema, "136_runtime_home_lifetime_readiness.up.sql")
	opts.Files = up
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply migration 136: %v", err)
	}
	assertMigration136Relations(t, ctx, f)

	// An unused authority can be rolled back and reapplied exactly.
	down := scopedRealMigrations(t, f.schema, "136_runtime_home_lifetime_readiness.down.sql")
	downOpts := f.opts()
	downOpts.Direction = "down"
	downOpts.Files = down
	if err := runMigrations(ctx, f.pool, downOpts); err != nil {
		t.Fatalf("empty down migration 136: %v", err)
	}
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("reapply migration 136 after empty down: %v", err)
	}

	seedMigration136Authority(t, ctx, f)
	seedMigration136AlternateNativeAssignment(t, ctx, f)

	// A task lifetime cannot cross-link one task snapshot to another otherwise
	// valid native assignment identity.
	expectMigration136ExecRejected(t, ctx, f, migration136CrossLinkedLifetimeInsert)

	// A binding changed to OmniRouter cannot acquire a native-home lifetime,
	// even when its immutable task snapshot still says native.
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE runtime_binding
			SET transport_binding = 'omniroute'
			WHERE id = '00000000-0000-0000-0000-000000000013'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, migration136LifetimeInsert)
		return err
	})

	// Deferred event coupling must reject a lifetime with no matching v1 event.
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, migration136LifetimeInsert)
		return err
	})

	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, migration136LifetimeInsert); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO runtime_home_lifetime_event (
				lifetime_id, state_version, state, transition_request_id, reason_code
			) VALUES (
				'00000000-0000-0000-0000-000000000016', 1, 'pending_local',
				'00000000-0000-0000-0000-000000000019', 'reserved'
			)`)
		return err
	})

	// The accepted transition and its event commit atomically.
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
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

	// Illegal state edges and version gaps fail before any event can legitimize them.
	expectMigration136ExecRejected(t, ctx, f, `
		UPDATE runtime_home_lifetime
		SET state = 'released', state_version = 3, released_at = now()
		WHERE id = '00000000-0000-0000-0000-000000000016'`)
	expectMigration136ExecRejected(t, ctx, f, `
		UPDATE runtime_home_lifetime
		SET state = 'recovery_pending', state_version = 4
		WHERE id = '00000000-0000-0000-0000-000000000016'`)

	// Readiness is an immutable, exact-generation, bounded and expiring result.
	execMigration136(t, ctx, f, migration136ReadinessInsert)
	// Historical exact identity remains evidence, but it cannot authorize a new
	// readiness result after the assignment stops being current.
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE runtime_home_assignment
			SET state = 'released', released_at = now()
			WHERE id = '00000000-0000-0000-0000-000000000014'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, migration136ReadinessReplayInsert,
			"00000000-0000-0000-0000-000000000040",
			"00000000-0000-0000-0000-000000000041",
		)
		return err
	})
	// The current binding must remain native at result acceptance.
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE runtime_binding
			SET transport_binding = 'omniroute'
			WHERE id = '00000000-0000-0000-0000-000000000013'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, migration136ReadinessReplayInsert,
			"00000000-0000-0000-0000-000000000042",
			"00000000-0000-0000-0000-000000000043",
		)
		return err
	})
	// A future observation is not a current server-accepted readiness fact.
	expectMigration136ExecRejected(t, ctx, f, `
		INSERT INTO runtime_credential_readiness_attestation (
			id, probe_request_id, workspace_id, agent_id, runtime_id,
			runtime_session_id, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_generation, home_ref,
			provider, daemon_id, daemon_boot_id, state, reason_code,
			observed_at, expires_at
		) SELECT
			'00000000-0000-0000-0000-000000000044',
			'00000000-0000-0000-0000-000000000045',
			workspace_id, agent_id, runtime_id, runtime_session_id,
			runtime_binding_id, binding_generation, home_assignment_id,
			catalog_id, catalog_generation, home_ref, provider, daemon_id,
			daemon_boot_id, state, reason_code,
			now() + interval '1 minute', now() + interval '2 minutes'
		FROM runtime_credential_readiness_attestation
		WHERE id = '00000000-0000-0000-0000-000000000020'`)
	expectMigration136ExecRejected(t, ctx, f, `
		UPDATE runtime_credential_readiness_attestation
		SET expires_at = expires_at + interval '1 minute'
		WHERE id = '00000000-0000-0000-0000-000000000020'`)
	expectMigration136ExecRejected(t, ctx, f, `
		INSERT INTO runtime_credential_readiness_attestation (
			id, probe_request_id, workspace_id, agent_id, runtime_id,
			runtime_session_id, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_generation, home_ref,
			provider, daemon_id, daemon_boot_id, state, reason_code,
			observed_at, expires_at
		) SELECT
			'00000000-0000-0000-0000-000000000023', probe_request_id,
			workspace_id, agent_id, runtime_id, runtime_session_id,
			runtime_binding_id, binding_generation, home_assignment_id,
			catalog_id, catalog_generation, home_ref, provider, daemon_id,
			daemon_boot_id, state, reason_code, observed_at, expires_at
		FROM runtime_credential_readiness_attestation
		WHERE id = '00000000-0000-0000-0000-000000000020'`)
	expectMigration136ExecRejected(t, ctx, f, `
		INSERT INTO runtime_credential_readiness_attestation (
			id, probe_request_id, workspace_id, agent_id, runtime_id,
			runtime_session_id, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_generation, home_ref,
			provider, daemon_id, daemon_boot_id, state, reason_code,
			observed_at, expires_at
		) VALUES (
			'00000000-0000-0000-0000-000000000024',
			'00000000-0000-0000-0000-000000000024',
			'00000000-0000-0000-0000-000000000001',
			'00000000-0000-0000-0000-000000000006',
			'00000000-0000-0000-0000-000000000005',
			'00000000-0000-0000-0000-000000000004',
			'00000000-0000-0000-0000-000000000013', 1,
			'00000000-0000-0000-0000-000000000014',
			'00000000-0000-0000-0000-000000000009', 1,
			'00000000-0000-0000-0000-000000000012', 'codex', 'daemon-a',
			'00000000-0000-0000-0000-000000000018',
			'ready', 'internal_error', now(), now() + interval '1 minute'
		)`)
	expectMigration136ExecRejected(t, ctx, f, `
		INSERT INTO runtime_credential_readiness_attestation (
			id, probe_request_id, workspace_id, agent_id, runtime_id,
			runtime_session_id, runtime_binding_id, binding_generation,
			home_assignment_id, catalog_id, catalog_generation, home_ref,
			provider, daemon_id, daemon_boot_id, state, reason_code,
			observed_at, expires_at
		) VALUES (
			'00000000-0000-0000-0000-000000000025',
			'00000000-0000-0000-0000-000000000025',
			'00000000-0000-0000-0000-000000000001',
			'00000000-0000-0000-0000-000000000006',
			'00000000-0000-0000-0000-000000000005',
			'00000000-0000-0000-0000-000000000004',
			'00000000-0000-0000-0000-000000000013', 1,
			'00000000-0000-0000-0000-000000000014',
			'00000000-0000-0000-0000-000000000009', 1,
			'00000000-0000-0000-0000-000000000012', 'codex', 'daemon-a',
			'00000000-0000-0000-0000-000000000018',
			'unready', 'probe_timeout', now(), now()
		)`)

	assertMigration136Pathless(t, ctx, f)

	// Populated safety/audit state makes rollback fail closed.
	if err := runMigrations(ctx, f.pool, downOpts); err == nil {
		t.Fatal("populated down migration 136 succeeded; want fail-closed refusal")
	}
}

func TestMigration136RollbackSerializesConcurrentWriter(t *testing.T) {
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
	)
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply migrations 132-136: %v", err)
	}
	seedMigration136Authority(t, ctx, f)

	writer, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin concurrent writer: %v", err)
	}
	defer writer.Rollback(ctx)
	if _, err := writer.Exec(ctx, fmt.Sprintf(
		"SET LOCAL search_path TO %s", pgx.Identifier{f.schema}.Sanitize(),
	)); err != nil {
		t.Fatalf("set concurrent writer search path: %v", err)
	}
	if _, err := writer.Exec(ctx, migration136LifetimeInsert); err != nil {
		t.Fatalf("stage concurrent lifetime: %v", err)
	}
	if _, err := writer.Exec(ctx, `
		INSERT INTO runtime_home_lifetime_event (
			lifetime_id, state_version, state, transition_request_id, reason_code
		) VALUES (
			'00000000-0000-0000-0000-000000000016', 1, 'pending_local',
			'00000000-0000-0000-0000-000000000019', 'reserved'
		)`); err != nil {
		t.Fatalf("stage concurrent lifetime event: %v", err)
	}

	downOpts := f.opts()
	downOpts.Direction = "down"
	downOpts.Files = scopedRealMigrations(t, f.schema,
		"136_runtime_home_lifetime_readiness.down.sql",
	)
	downResult := make(chan error, 1)
	go func() {
		downResult <- runMigrations(ctx, f.pool, downOpts)
	}()

	relation := f.schema + ".runtime_home_lifetime"
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := f.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE relation = to_regclass($1)
				  AND mode = 'AccessExclusiveLock'
				  AND NOT granted
			)`, relation).Scan(&waiting); err != nil {
			t.Fatalf("inspect rollback table lock: %v", err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("rollback did not wait for ACCESS EXCLUSIVE lifetime lock")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := writer.Commit(ctx); err != nil {
		t.Fatalf("commit concurrent writer: %v", err)
	}
	if err := <-downResult; err == nil {
		t.Fatal("rollback discarded a row committed while waiting for its table lock")
	}

	var rows int
	if err := f.pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT count(*) FROM %s.runtime_home_lifetime",
		pgx.Identifier{f.schema}.Sanitize(),
	)).Scan(&rows); err != nil {
		t.Fatalf("read preserved lifetime after refused rollback: %v", err)
	}
	if rows != 1 {
		t.Fatalf("preserved lifetime rows = %d, want 1", rows)
	}
}

const migration136LifetimeInsert = `
	INSERT INTO runtime_home_lifetime (
		id, acquisition_request_id, task_id, runtime_binding_id,
		binding_generation, home_assignment_id, workspace_id, daemon_id,
		catalog_id, catalog_generation, home_ref, daemon_boot_id, state
	) VALUES (
		'00000000-0000-0000-0000-000000000016',
		'00000000-0000-0000-0000-000000000017',
		'00000000-0000-0000-0000-000000000008',
		'00000000-0000-0000-0000-000000000013', 1,
		'00000000-0000-0000-0000-000000000014',
		'00000000-0000-0000-0000-000000000001', 'daemon-a',
		'00000000-0000-0000-0000-000000000009', 1,
		'00000000-0000-0000-0000-000000000012',
		'00000000-0000-0000-0000-000000000018', 'pending_local'
	)`

const migration136CrossLinkedLifetimeInsert = `
	INSERT INTO runtime_home_lifetime (
		id, acquisition_request_id, task_id, runtime_binding_id,
		binding_generation, home_assignment_id, workspace_id, daemon_id,
		catalog_id, catalog_generation, home_ref, daemon_boot_id, state
	) VALUES (
		'00000000-0000-0000-0000-000000000038',
		'00000000-0000-0000-0000-000000000039',
		'00000000-0000-0000-0000-000000000008',
		'00000000-0000-0000-0000-000000000036', 1,
		'00000000-0000-0000-0000-000000000037',
		'00000000-0000-0000-0000-000000000001', 'daemon-b',
		'00000000-0000-0000-0000-000000000031', 1,
		'00000000-0000-0000-0000-000000000034',
		'00000000-0000-0000-0000-000000000018', 'pending_local'
	)`

const migration136ReadinessInsert = `
	INSERT INTO runtime_credential_readiness_attestation (
		id, probe_request_id, workspace_id, agent_id, runtime_id,
		runtime_session_id, runtime_binding_id, binding_generation,
		home_assignment_id, catalog_id, catalog_generation, home_ref,
		provider, daemon_id, daemon_boot_id, state, reason_code,
		observed_at, expires_at
	) VALUES (
		'00000000-0000-0000-0000-000000000020',
		'00000000-0000-0000-0000-000000000021',
		'00000000-0000-0000-0000-000000000001',
		'00000000-0000-0000-0000-000000000006',
		'00000000-0000-0000-0000-000000000005',
		'00000000-0000-0000-0000-000000000004',
		'00000000-0000-0000-0000-000000000013', 1,
		'00000000-0000-0000-0000-000000000014',
		'00000000-0000-0000-0000-000000000009', 1,
		'00000000-0000-0000-0000-000000000012', 'codex', 'daemon-a',
		'00000000-0000-0000-0000-000000000018',
		'ready', 'ready', now(), now() + interval '1 minute'
	)`

const migration136ReadinessReplayInsert = `
	INSERT INTO runtime_credential_readiness_attestation (
		id, probe_request_id, workspace_id, agent_id, runtime_id,
		runtime_session_id, runtime_binding_id, binding_generation,
		home_assignment_id, catalog_id, catalog_generation, home_ref,
		provider, daemon_id, daemon_boot_id, state, reason_code,
		observed_at, expires_at
	) SELECT
		$1, $2, workspace_id, agent_id, runtime_id, runtime_session_id,
		runtime_binding_id, binding_generation, home_assignment_id,
		catalog_id, catalog_generation, home_ref, provider, daemon_id,
		daemon_boot_id, state, reason_code, now(), now() + interval '1 minute'
	FROM runtime_credential_readiness_attestation
	WHERE id = '00000000-0000-0000-0000-000000000020'
	`

func createMigration136Prerequisites(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	schema := pgx.Identifier{f.schema}.Sanitize()
	_, err := f.pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.workspace (id UUID PRIMARY KEY);
		CREATE TABLE %[1]s."user" (id UUID PRIMARY KEY);
		CREATE TABLE %[1]s.runtime_session_enrollment (
			id UUID PRIMARY KEY,
			session_id UUID NOT NULL,
			workspace_id UUID NOT NULL,
			runtime_id UUID NOT NULL,
			agent_id UUID NOT NULL
		);
		CREATE TABLE %[1]s.runtime_standard_version (id UUID PRIMARY KEY);
		CREATE TABLE %[1]s.agent_task_queue (
			id UUID PRIMARY KEY,
			agent_id UUID NOT NULL,
			runtime_id UUID NOT NULL,
			status TEXT NOT NULL,
			dispatched_at TIMESTAMPTZ
		);
		CREATE FUNCTION %[1]s.reject_runtime_manager_immutable_mutation()
		RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION USING ERRCODE = '55006', MESSAGE = 'immutable evidence';
		END
		$$;
	`, schema))
	if err != nil {
		t.Fatalf("create migration 136 prerequisites: %v", err)
	}
}

func seedMigration136Authority(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO workspace VALUES ('00000000-0000-0000-0000-000000000001');
			INSERT INTO "user" VALUES ('00000000-0000-0000-0000-000000000002');
			INSERT INTO runtime_session_enrollment VALUES (
				'00000000-0000-0000-0000-000000000003',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000006'
			);
			INSERT INTO runtime_standard_version VALUES ('00000000-0000-0000-0000-000000000007');
			INSERT INTO agent_task_queue (id, agent_id, runtime_id, status) VALUES (
				'00000000-0000-0000-0000-000000000008',
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005', 'queued'
			);
			INSERT INTO credential_home_catalog (
				id, workspace_id, daemon_id, generation
			) VALUES (
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000001', 'daemon-a', 1
			);
			INSERT INTO credential_home_catalog_generation (
				id, catalog_id, previous_generation, generation, scan_kind,
				catalog_digest, started_at
			) VALUES (
				'00000000-0000-0000-0000-000000000010',
				'00000000-0000-0000-0000-000000000009', 0, 1, 'startup',
				repeat('a', 64), now()
			);
			INSERT INTO credential_home_catalog_entry (
				id, generation_id, catalog_id, generation, home_ref, provider,
				approved, state, first_seen_at, last_seen_at, last_full_scan_at,
				health_watermark, ttl_nanoseconds, retention_deadline
			) VALUES (
				'00000000-0000-0000-0000-000000000011',
				'00000000-0000-0000-0000-000000000010',
				'00000000-0000-0000-0000-000000000009', 1,
				'00000000-0000-0000-0000-000000000012', 'codex', true, 'healthy',
				now(), now(), now(), now(), 60000000000, now() + interval '1 day'
			);
			INSERT INTO runtime_binding (
				id, enrollment_id, session_id, workspace_id, runtime_id, agent_id,
				transport_binding, created_by
			) VALUES (
				'00000000-0000-0000-0000-000000000013',
				'00000000-0000-0000-0000-000000000003',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000006',
				'native_credential_home',
				'00000000-0000-0000-0000-000000000002'
			);
			INSERT INTO runtime_home_assignment (
				id, binding_id, workspace_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, binding_generation, assigned_by
			) VALUES (
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000013',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000011', 1,
				'00000000-0000-0000-0000-000000000012', 1,
				'00000000-0000-0000-0000-000000000002'
			);
			INSERT INTO runtime_configuration_version (
				id, binding_id, version_number, configuration,
				configuration_digest, apply_class, created_by, reason, request_id
			) VALUES (
				'00000000-0000-0000-0000-000000000015',
				'00000000-0000-0000-0000-000000000013', 1, '{}'::jsonb,
				repeat('b', 64), 'hot',
				'00000000-0000-0000-0000-000000000002', 'test', 'config-1'
			);
			UPDATE runtime_binding
			SET active_configuration_version_id = '00000000-0000-0000-0000-000000000015',
				effective_configuration_digest = repeat('b', 64),
				active_task_count = 1
			WHERE id = '00000000-0000-0000-0000-000000000013';
			INSERT INTO runtime_task_snapshot (
				task_id, runtime_session_id, runtime_id, agent_id, workspace_id,
				runtime_standard_version_id, runtime_configuration_version_id,
				effective_configuration_digest, runtime_binding_id,
				binding_generation, transport_binding, home_assignment_id,
				home_ref, catalog_generation, capability_digest
			) VALUES (
				'00000000-0000-0000-0000-000000000008',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000007',
				'00000000-0000-0000-0000-000000000015', repeat('b', 64),
				'00000000-0000-0000-0000-000000000013', 1,
				'native_credential_home',
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000012', 1, repeat('c', 64)
			);
		`)
		return err
	})
}

func seedMigration136AlternateNativeAssignment(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO runtime_session_enrollment VALUES (
				'00000000-0000-0000-0000-000000000035',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000006'
			);
			INSERT INTO credential_home_catalog (
				id, workspace_id, daemon_id, generation
			) VALUES (
				'00000000-0000-0000-0000-000000000031',
				'00000000-0000-0000-0000-000000000001', 'daemon-b', 1
			);
			INSERT INTO credential_home_catalog_generation (
				id, catalog_id, previous_generation, generation, scan_kind,
				catalog_digest, started_at
			) VALUES (
				'00000000-0000-0000-0000-000000000032',
				'00000000-0000-0000-0000-000000000031', 0, 1, 'startup',
				repeat('d', 64), now()
			);
			INSERT INTO credential_home_catalog_entry (
				id, generation_id, catalog_id, generation, home_ref, provider,
				approved, state, first_seen_at, last_seen_at, last_full_scan_at,
				health_watermark, ttl_nanoseconds, retention_deadline
			) VALUES (
				'00000000-0000-0000-0000-000000000033',
				'00000000-0000-0000-0000-000000000032',
				'00000000-0000-0000-0000-000000000031', 1,
				'00000000-0000-0000-0000-000000000034', 'codex', true, 'healthy',
				now(), now(), now(), now(), 60000000000, now() + interval '1 day'
			);
			INSERT INTO runtime_binding (
				id, enrollment_id, session_id, workspace_id, runtime_id, agent_id,
				transport_binding, created_by
			) VALUES (
				'00000000-0000-0000-0000-000000000036',
				'00000000-0000-0000-0000-000000000035',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000006',
				'native_credential_home',
				'00000000-0000-0000-0000-000000000002'
			);
			INSERT INTO runtime_home_assignment (
				id, binding_id, workspace_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, binding_generation, assigned_by
			) VALUES (
				'00000000-0000-0000-0000-000000000037',
				'00000000-0000-0000-0000-000000000036',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000031',
				'00000000-0000-0000-0000-000000000033', 1,
				'00000000-0000-0000-0000-000000000034', 1,
				'00000000-0000-0000-0000-000000000002'
			);
		`)
		return err
	})
}

func assertMigration136Relations(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relname IN (
			'runtime_home_lifetime',
			'runtime_home_lifetime_event',
			'runtime_credential_readiness_attestation'
		  )
	`, f.schema).Scan(&count); err != nil {
		t.Fatalf("inspect migration 136 relations: %v", err)
	}
	if count != 3 {
		t.Fatalf("migration 136 relation count = %d, want 3", count)
	}
}

func assertMigration136Pathless(t *testing.T, ctx context.Context, f *fixture) {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name IN (
			'runtime_home_lifetime',
			'runtime_home_lifetime_event',
			'runtime_credential_readiness_attestation'
		  )
		  AND column_name ~ '(path|name_ref|credential_value|token|secret)'
	`, f.schema).Scan(&count); err != nil {
		t.Fatalf("inspect pathless migration 136 columns: %v", err)
	}
	if count != 0 {
		t.Fatalf("migration 136 has %d forbidden path/secret-bearing columns", count)
	}
}

func execMigration136(t *testing.T, ctx context.Context, f *fixture, query string) {
	t.Helper()
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query)
		return err
	})
}

func execMigration136Tx(t *testing.T, ctx context.Context, f *fixture, fn func(pgx.Tx) error) {
	t.Helper()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin migration 136 assertion: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s", pgx.Identifier{f.schema}.Sanitize())); err != nil {
		t.Fatalf("set migration 136 search path: %v", err)
	}
	if err := fn(tx); err != nil {
		t.Fatalf("execute migration 136 assertion: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit migration 136 assertion: %v", err)
	}
}

func expectMigration136ExecRejected(t *testing.T, ctx context.Context, f *fixture, query string) {
	t.Helper()
	expectMigration136TxRejected(t, ctx, f, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query)
		return err
	})
}

func expectMigration136TxRejected(t *testing.T, ctx context.Context, f *fixture, fn func(pgx.Tx) error) {
	t.Helper()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rejected migration 136 assertion: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s", pgx.Identifier{f.schema}.Sanitize())); err != nil {
		t.Fatalf("set rejected migration 136 search path: %v", err)
	}
	if err := fn(tx); err != nil {
		return
	}
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("migration 136 operation succeeded; want rejection")
	}
}
