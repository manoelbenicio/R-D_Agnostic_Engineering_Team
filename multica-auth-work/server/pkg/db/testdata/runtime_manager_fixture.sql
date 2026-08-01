-- Deterministic, metadata-only fixture for real migration/query checks.
-- Run inside a disposable database after migrations 001-134. The transaction
-- is intentionally rolled back so it cannot alter an existing development DB.
BEGIN;

INSERT INTO "user" (id, name, email)
VALUES ('60000000-0000-4000-8000-000000000001', 'SPE-6 fixture owner', 'spe6-fixture@example.invalid');

INSERT INTO workspace (id, name, slug)
VALUES ('60000000-0000-4000-8000-000000000002', 'SPE-6 fixture workspace', 'spe6-fixture');

INSERT INTO agent_runtime (
    id, workspace_id, daemon_id, name, runtime_mode, provider, owner_id
)
VALUES (
    '60000000-0000-4000-8000-000000000003',
    '60000000-0000-4000-8000-000000000002',
    'spe6-fixture-daemon', 'SPE-6 fixture runtime', 'local', 'codex',
    '60000000-0000-4000-8000-000000000001'
);

INSERT INTO agent (id, workspace_id, name, runtime_mode, owner_id, runtime_id)
VALUES (
    '60000000-0000-4000-8000-000000000004',
    '60000000-0000-4000-8000-000000000002',
    'SPE-6 fixture agent', 'local',
    '60000000-0000-4000-8000-000000000001',
    '60000000-0000-4000-8000-000000000003'
);

INSERT INTO runtime_standard (id, owner_id, name, request_id)
VALUES (
    '60000000-0000-4000-8000-000000000005',
    '60000000-0000-4000-8000-000000000001',
    'SPE-6 fixture standard', 'fixture-standard-create'
);

INSERT INTO runtime_standard_version (
    id, standard_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason, request_id
)
VALUES (
    '60000000-0000-4000-8000-000000000006',
    '60000000-0000-4000-8000-000000000005', 1,
    '{"routing":{"provider":"codex","transport_binding":"native_credential_home"}}',
    repeat('1', 64), 'restart', '60000000-0000-4000-8000-000000000001',
    'fixture', 'fixture-standard-version-create'
);

UPDATE runtime_standard
SET active_version_id = '60000000-0000-4000-8000-000000000006'
WHERE id = '60000000-0000-4000-8000-000000000005';

INSERT INTO runtime_session (
    id, owner_id, standard_id, name, provider, runtime_kind, created_by
)
VALUES (
    '60000000-0000-4000-8000-000000000007',
    '60000000-0000-4000-8000-000000000001',
    '60000000-0000-4000-8000-000000000005',
    'SPE-6 fixture session', 'codex', 'native',
    '60000000-0000-4000-8000-000000000001'
);

INSERT INTO runtime_session_enrollment (
    id, session_id, workspace_id, runtime_id, agent_id, enrolled_by
)
VALUES (
    '60000000-0000-4000-8000-000000000008',
    '60000000-0000-4000-8000-000000000007',
    '60000000-0000-4000-8000-000000000002',
    '60000000-0000-4000-8000-000000000003',
    '60000000-0000-4000-8000-000000000004',
    '60000000-0000-4000-8000-000000000001'
);

INSERT INTO credential_home_catalog (
    id, workspace_id, daemon_id, generation, lifecycle_generation
)
VALUES (
    '60000000-0000-4000-8000-000000000009',
    '60000000-0000-4000-8000-000000000002', 'spe6-fixture-daemon', 1, 1
);

INSERT INTO credential_home_catalog_generation (
    id, catalog_id, previous_generation, generation, scan_kind, counters,
    catalog_digest, started_at
)
VALUES (
    '60000000-0000-4000-8000-00000000000a',
    '60000000-0000-4000-8000-000000000009', 0, 1, 'startup',
    '{"healthy":1}', repeat('2', 64), now()
);

INSERT INTO credential_home_catalog_entry (
    id, generation_id, catalog_id, generation, home_ref, name_ref, provider,
    approved, state, active_refs, first_seen_at, last_seen_at,
    last_full_scan_at, health_watermark, ttl_nanoseconds, retention_deadline
)
VALUES (
    '60000000-0000-4000-8000-00000000000b',
    '60000000-0000-4000-8000-00000000000a',
    '60000000-0000-4000-8000-000000000009', 1,
    '60000000-0000-5000-8000-00000000000c', 'name_' || repeat('c', 43),
    'codex', true, 'healthy', 0,
    now(), now(), now(), now(), 3600000000000, now() + interval '30 days'
);

INSERT INTO credential_home_catalog_lifecycle (
    catalog_id, home_ref, name_ref, provider, state, reason_code, active_refs,
    generation, updated_at, retention_deadline
)
VALUES (
    '60000000-0000-4000-8000-000000000009',
    '60000000-0000-5000-8000-00000000001c', 'name_' || repeat('n', 43),
    'codex', 'retired', 'tombstoned', 0, 1, now(), now() + interval '30 days'
);

-- Lifecycle state must be durable before a generation can be published, and
-- tombstone/non-reuse failures must roll the complete caller transaction back.
DO $$
DECLARE affected BIGINT;
BEGIN
    UPDATE credential_home_catalog
    SET generation = 2
    WHERE id = '60000000-0000-4000-8000-000000000009'
      AND generation = 1
      AND lifecycle_generation = 2;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 0 THEN
        RAISE EXCEPTION 'SPE-6 catalog published before lifecycle persistence';
    END IF;

    BEGIN
        UPDATE credential_home_catalog
        SET state = 'reconciling', lifecycle_generation = 2
        WHERE id = '60000000-0000-4000-8000-000000000009'
          AND generation = 1
          AND lifecycle_generation = 1;

        UPDATE credential_home_catalog_lifecycle
        SET state = 'missing', reason_code = NULL, generation = 2
        WHERE catalog_id = '60000000-0000-4000-8000-000000000009'
          AND home_ref = '60000000-0000-5000-8000-00000000001c';

        RAISE EXCEPTION 'SPE-6 tombstone reuse was accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    IF (SELECT lifecycle_generation FROM credential_home_catalog
        WHERE id = '60000000-0000-4000-8000-000000000009') <> 1
       OR (SELECT state FROM credential_home_catalog_lifecycle
           WHERE catalog_id = '60000000-0000-4000-8000-000000000009'
             AND home_ref = '60000000-0000-5000-8000-00000000001c') <> 'retired'
       OR (SELECT reason_code FROM credential_home_catalog_lifecycle
           WHERE catalog_id = '60000000-0000-4000-8000-000000000009'
             AND home_ref = '60000000-0000-5000-8000-00000000001c') <> 'tombstoned' THEN
        RAISE EXCEPTION 'SPE-6 lifecycle failure did not roll back atomically';
    END IF;
END
$$;

INSERT INTO runtime_binding (
    id, enrollment_id, session_id, workspace_id, runtime_id, agent_id,
    transport_binding, created_by
)
VALUES (
    '60000000-0000-4000-8000-00000000000d',
    '60000000-0000-4000-8000-000000000008',
    '60000000-0000-4000-8000-000000000007',
    '60000000-0000-4000-8000-000000000002',
    '60000000-0000-4000-8000-000000000003',
    '60000000-0000-4000-8000-000000000004',
    'native_credential_home', '60000000-0000-4000-8000-000000000001'
);

INSERT INTO runtime_home_assignment (
    id, binding_id, workspace_id, catalog_id, catalog_entry_id,
    catalog_generation, home_ref, binding_generation, assigned_by
)
VALUES (
    '60000000-0000-4000-8000-00000000000e',
    '60000000-0000-4000-8000-00000000000d',
    '60000000-0000-4000-8000-000000000002',
    '60000000-0000-4000-8000-000000000009',
    '60000000-0000-4000-8000-00000000000b', 1,
    '60000000-0000-5000-8000-00000000000c', 1,
    '60000000-0000-4000-8000-000000000001'
);

INSERT INTO runtime_configuration_version (
    id, binding_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason, request_id
)
VALUES (
    '60000000-0000-4000-8000-00000000000f',
    '60000000-0000-4000-8000-00000000000d', 1,
    '{"limits":{"max_concurrent_tasks":1}}', repeat('3', 64), 'restart',
    '60000000-0000-4000-8000-000000000001', 'fixture',
    'fixture-binding-version-create'
);

UPDATE runtime_binding
SET active_configuration_version_id = '60000000-0000-4000-8000-00000000000f',
    effective_configuration_digest = repeat('4', 64)
WHERE id = '60000000-0000-4000-8000-00000000000d';

-- Capacity reservation is fenced by the locked binding/catalog generations.
UPDATE runtime_binding b
SET active_task_count = active_task_count + 1
WHERE b.id = '60000000-0000-4000-8000-00000000000d'
  AND b.generation = 1
  AND b.active_task_count < b.max_concurrent_tasks
  AND EXISTS (
      SELECT 1
      FROM runtime_home_assignment a
      JOIN credential_home_catalog c ON c.id = a.catalog_id
      JOIN credential_home_catalog_entry e ON e.id = a.catalog_entry_id
      WHERE a.binding_id = b.id
        AND a.state = 'active'
        AND c.generation = a.catalog_generation
        AND c.state = 'available'
        AND e.approved = true
        AND e.state = 'healthy'
  );

INSERT INTO issue (
    id, workspace_id, title, creator_type, creator_id, number
)
VALUES (
    '60000000-0000-4000-8000-000000000010',
    '60000000-0000-4000-8000-000000000002',
    'SPE-6 fixture task', 'member',
    '60000000-0000-4000-8000-000000000001', 1
);

INSERT INTO agent_task_queue (
    id, agent_id, issue_id, runtime_id
)
VALUES (
    '60000000-0000-4000-8000-000000000011',
    '60000000-0000-4000-8000-000000000004',
    '60000000-0000-4000-8000-000000000010',
    '60000000-0000-4000-8000-000000000003'
);

INSERT INTO runtime_task_snapshot (
    task_id, runtime_session_id, runtime_id, agent_id, workspace_id,
    runtime_standard_version_id, runtime_configuration_version_id,
    effective_configuration_digest, runtime_binding_id, binding_generation,
    transport_binding, home_assignment_id, home_ref, catalog_generation,
    capability_digest
)
VALUES (
    '60000000-0000-4000-8000-000000000011',
    '60000000-0000-4000-8000-000000000007',
    '60000000-0000-4000-8000-000000000003',
    '60000000-0000-4000-8000-000000000004',
    '60000000-0000-4000-8000-000000000002',
    '60000000-0000-4000-8000-000000000006',
    '60000000-0000-4000-8000-00000000000f', repeat('4', 64),
    '60000000-0000-4000-8000-00000000000d', 1,
    'native_credential_home',
    '60000000-0000-4000-8000-00000000000e',
    '60000000-0000-5000-8000-00000000000c', 1, repeat('5', 64)
);

-- Capacity is exhausted at one, and an active snapshot prevents assignment
-- release. Both statements must affect zero rows.
DO $$
DECLARE affected BIGINT;
BEGIN
    UPDATE runtime_binding
    SET active_task_count = active_task_count + 1
    WHERE id = '60000000-0000-4000-8000-00000000000d'
      AND active_task_count < max_concurrent_tasks;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 0 THEN
        RAISE EXCEPTION 'SPE-6 capacity fence failed';
    END IF;

    UPDATE runtime_home_assignment a
    SET state = 'released', released_at = now()
    WHERE a.id = '60000000-0000-4000-8000-00000000000e'
      AND NOT EXISTS (
          SELECT 1 FROM runtime_task_snapshot s
          LEFT JOIN runtime_task_snapshot_release r ON r.task_id = s.task_id
          WHERE s.home_assignment_id = a.id AND r.task_id IS NULL
      );
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 0 THEN
        RAISE EXCEPTION 'SPE-6 active-reference release fence failed';
    END IF;
END
$$;

-- This models the caller's required transaction rollback on a duplicate task
-- with conflicting snapshot identity. The speculative counter increment is
-- rolled back with the conflict and therefore cannot leak capacity.
DO $$
DECLARE inserted_task UUID;
BEGIN
    BEGIN
        UPDATE runtime_binding
        SET max_concurrent_tasks = 2,
            active_task_count = active_task_count + 1
        WHERE id = '60000000-0000-4000-8000-00000000000d';

        INSERT INTO runtime_task_snapshot (
            task_id, runtime_session_id, runtime_id, agent_id, workspace_id,
            runtime_standard_version_id, runtime_configuration_version_id,
            effective_configuration_digest, runtime_binding_id, binding_generation,
            transport_binding, home_assignment_id, home_ref, catalog_generation,
            capability_digest
        ) VALUES (
            '60000000-0000-4000-8000-000000000011',
            '60000000-0000-4000-8000-000000000007',
            '60000000-0000-4000-8000-000000000003',
            '60000000-0000-4000-8000-000000000004',
            '60000000-0000-4000-8000-000000000002',
            '60000000-0000-4000-8000-000000000006',
            '60000000-0000-4000-8000-00000000000f', repeat('9', 64),
            '60000000-0000-4000-8000-00000000000d', 1,
            'native_credential_home',
            '60000000-0000-4000-8000-00000000000e',
            '60000000-0000-5000-8000-00000000000c', 1, repeat('5', 64)
        )
        ON CONFLICT (task_id) DO NOTHING
        RETURNING task_id INTO inserted_task;

        IF inserted_task IS NULL THEN
            RAISE EXCEPTION 'duplicate_task_conflict';
        END IF;
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM <> 'duplicate_task_conflict' THEN
            RAISE;
        END IF;
    END;

    IF (SELECT active_task_count FROM runtime_binding
        WHERE id = '60000000-0000-4000-8000-00000000000d') <> 1 THEN
        RAISE EXCEPTION 'SPE-6 duplicate rollback leaked capacity';
    END IF;
END
$$;

-- Terminal release inserts one immutable marker and decrements once. The
-- outer fixture deliberately rolls this probe back so the retained snapshot
-- can still be asserted below.
DO $$
DECLARE decremented INTEGER;
BEGIN
    BEGIN
        UPDATE agent_task_queue
        SET status = 'completed', completed_at = now()
        WHERE id = '60000000-0000-4000-8000-000000000011';

        WITH released AS (
            INSERT INTO runtime_task_snapshot_release (
                task_id, runtime_binding_id, released_by, reason_code
            ) VALUES (
                '60000000-0000-4000-8000-000000000011',
                '60000000-0000-4000-8000-00000000000d',
                '60000000-0000-4000-8000-000000000001', 'fixture_terminal'
            )
            ON CONFLICT (task_id) DO NOTHING
            RETURNING runtime_binding_id
        )
        UPDATE runtime_binding b
        SET active_task_count = active_task_count - 1
        FROM released r
        WHERE b.id = r.runtime_binding_id AND b.active_task_count > 0;
        GET DIAGNOSTICS decremented = ROW_COUNT;

        IF decremented <> 1 OR
           (SELECT active_task_count FROM runtime_binding
            WHERE id = '60000000-0000-4000-8000-00000000000d') <> 0 THEN
            RAISE EXCEPTION 'SPE-6 terminal decrement failed';
        END IF;

        RAISE EXCEPTION 'rollback_release_probe';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM <> 'rollback_release_probe' THEN
            RAISE;
        END IF;
    END;

    IF EXISTS (
        SELECT 1 FROM runtime_task_snapshot_release
        WHERE task_id = '60000000-0000-4000-8000-000000000011'
    ) OR (SELECT active_task_count FROM runtime_binding
          WHERE id = '60000000-0000-4000-8000-00000000000d') <> 1 THEN
        RAISE EXCEPTION 'SPE-6 release rollback was not atomic';
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM runtime_binding b
        JOIN runtime_home_assignment a ON a.binding_id = b.id
        JOIN credential_home_catalog_entry e ON e.id = a.catalog_entry_id
        JOIN runtime_task_snapshot s ON s.runtime_binding_id = b.id
        WHERE b.id = '60000000-0000-4000-8000-00000000000d'
          AND b.effective_configuration_digest = repeat('4', 64)
          AND b.active_task_count = 1
          AND s.effective_configuration_digest = repeat('4', 64)
          AND e.state = 'healthy'
    ) THEN
        RAISE EXCEPTION 'SPE-6 runtime-manager fixture failed';
    END IF;
END
$$;

ROLLBACK;
