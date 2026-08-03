-- Empty-only rollback. ACCESS EXCLUSIVE locks close the check/drop race.
LOCK TABLE
    runtime_task_home_epoch,
    runtime_home_lifetime,
    runtime_home_lifetime_event,
    native_rotation_operation,
    native_rotation_operation_event,
    native_rotation_retirement_attempt,
    native_rotation_retirement_attempt_event
IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM native_rotation_retirement_attempt_event LIMIT 1)
       OR EXISTS (SELECT 1 FROM native_rotation_retirement_attempt LIMIT 1)
       OR EXISTS (SELECT 1 FROM native_rotation_operation_event LIMIT 1)
       OR EXISTS (SELECT 1 FROM native_rotation_operation LIMIT 1)
       OR EXISTS (SELECT 1 FROM runtime_task_home_epoch LIMIT 1)
       OR EXISTS (SELECT 1 FROM runtime_home_lifetime LIMIT 1) THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'cannot roll back native rotation authority after use';
    END IF;
END
$$;

DROP TRIGGER native_rotation_retirement_attempt_event_guard
    ON native_rotation_retirement_attempt_event;
DROP TRIGGER native_rotation_operation_event_guard
    ON native_rotation_operation_event;
DROP TRIGGER native_rotation_retirement_attempt_event_required
    ON native_rotation_retirement_attempt;
DROP FUNCTION require_native_retirement_attempt_event();
DROP TRIGGER native_rotation_operation_event_required
    ON native_rotation_operation;
DROP FUNCTION require_native_rotation_operation_event();
DROP TRIGGER native_rotation_retirement_attempt_state_guard
    ON native_rotation_retirement_attempt;
DROP FUNCTION enforce_native_retirement_attempt_state();
DROP TRIGGER native_rotation_operation_state_guard
    ON native_rotation_operation;
DROP FUNCTION enforce_native_rotation_operation_state();
DROP FUNCTION enforce_native_rotation_immutable_event();
DROP TRIGGER runtime_task_home_epoch_operation_required ON runtime_task_home_epoch;
DROP FUNCTION require_later_epoch_operation();
DROP TRIGGER runtime_task_home_epoch_guard ON runtime_task_home_epoch;
DROP FUNCTION enforce_runtime_task_home_epoch();

-- Restore the exact Migration-136 trigger body before removing the epoch
-- relation and columns referenced by the Migration-137 replacement.
CREATE OR REPLACE FUNCTION enforce_runtime_home_lifetime_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'runtime-home lifetime records cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.state <> 'pending_local' OR NEW.state_version <> 1 THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'runtime-home lifetime must start pending_local/v1';
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM runtime_task_snapshot s
            JOIN runtime_binding b
              ON b.id = s.runtime_binding_id
             AND b.workspace_id = s.workspace_id
            JOIN runtime_home_assignment a
              ON a.id = s.home_assignment_id
             AND a.binding_id = s.runtime_binding_id
             AND a.workspace_id = s.workspace_id
             AND a.home_ref = s.home_ref
             AND a.catalog_generation = s.catalog_generation
            JOIN credential_home_catalog c
              ON c.id = a.catalog_id
             AND c.workspace_id = a.workspace_id
            JOIN credential_home_catalog_entry e
              ON e.id = a.catalog_entry_id
             AND e.catalog_id = a.catalog_id
             AND e.generation = a.catalog_generation
             AND e.home_ref = a.home_ref
            WHERE s.task_id = NEW.task_id
              AND s.transport_binding = 'native_credential_home'
              AND s.runtime_binding_id = NEW.runtime_binding_id
              AND s.binding_generation = NEW.binding_generation
              AND s.home_assignment_id = NEW.home_assignment_id
              AND s.workspace_id = NEW.workspace_id
              AND s.home_ref = NEW.home_ref
              AND s.catalog_generation = NEW.catalog_generation
              AND a.catalog_id = NEW.catalog_id
              AND b.transport_binding = 'native_credential_home'
              AND b.state = 'active'
              AND b.generation = NEW.binding_generation
              AND a.state = 'active'
              AND c.daemon_id = NEW.daemon_id
              AND c.state = 'available'
              AND c.generation = NEW.catalog_generation
              AND e.provider IN ('antigravity', 'codex', 'kiro')
              AND e.approved = true
              AND e.state = 'healthy'
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'runtime-home lifetime requires the same current native task-snapshot identity';
        END IF;

        RETURN NEW;
    END IF;

    IF ROW(
        NEW.id, NEW.acquisition_request_id, NEW.task_id,
        NEW.runtime_binding_id, NEW.binding_generation,
        NEW.home_assignment_id, NEW.workspace_id, NEW.daemon_id,
        NEW.catalog_id, NEW.catalog_generation, NEW.home_ref,
        NEW.daemon_boot_id, NEW.acquired_at
    ) IS DISTINCT FROM ROW(
        OLD.id, OLD.acquisition_request_id, OLD.task_id,
        OLD.runtime_binding_id, OLD.binding_generation,
        OLD.home_assignment_id, OLD.workspace_id, OLD.daemon_id,
        OLD.catalog_id, OLD.catalog_generation, OLD.home_ref,
        OLD.daemon_boot_id, OLD.acquired_at
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'runtime-home lifetime identity is immutable';
    END IF;

    IF NEW.state_version <> OLD.state_version + 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'runtime-home lifetime state version must advance by one';
    END IF;

    IF NOT (
        (OLD.state = 'pending_local' AND NEW.state IN ('acquired', 'recovery_pending'))
        OR (OLD.state = 'acquired' AND NEW.state IN ('process_started', 'recovery_pending'))
        OR (OLD.state = 'process_started' AND NEW.state IN ('released', 'recovery_pending'))
        OR (OLD.state = 'recovery_pending' AND NEW.state IN ('released', 'quarantined'))
        OR (OLD.state = 'quarantined' AND NEW.state = 'released')
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'invalid runtime-home lifetime transition';
    END IF;

    IF OLD.process_identity_digest IS NOT NULL
       AND NEW.process_identity_digest IS DISTINCT FROM OLD.process_identity_digest THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'runtime-home process identity is immutable once recorded';
    END IF;

    RETURN NEW;
END
$$;

DROP TABLE native_rotation_retirement_attempt_event;
DROP TABLE native_rotation_retirement_attempt;
DROP TABLE native_rotation_operation_event;
DROP TABLE native_rotation_operation;

ALTER TABLE runtime_task_home_epoch
    DROP CONSTRAINT runtime_task_home_epoch_lifetime_fkey;
ALTER TABLE runtime_home_lifetime
    DROP CONSTRAINT runtime_home_lifetime_epoch_fkey;
DROP TABLE runtime_task_home_epoch;

ALTER TABLE runtime_home_lifetime_event
    DROP CONSTRAINT runtime_home_lifetime_event_state_check;
ALTER TABLE runtime_home_lifetime_event
    ADD CONSTRAINT runtime_home_lifetime_event_state_check
    CHECK (
        state IN (
            'pending_local', 'acquired', 'process_started',
            'recovery_pending', 'released', 'quarantined'
        )
    );
DROP INDEX runtime_home_lifetime_active_task_epoch;
ALTER TABLE runtime_home_lifetime
    DROP CONSTRAINT runtime_home_lifetime_epoch_identity_key,
    DROP CONSTRAINT runtime_home_lifetime_epoch_key,
    DROP CONSTRAINT runtime_home_lifetime_state_check,
    DROP COLUMN transport_binding,
    DROP COLUMN provider,
    DROP COLUMN runtime_session_id,
    DROP COLUMN runtime_id,
    DROP COLUMN agent_id,
    DROP COLUMN home_epoch;
ALTER TABLE runtime_home_lifetime
    ADD CONSTRAINT runtime_home_lifetime_state_check
    CHECK (
        state IN (
            'pending_local', 'acquired', 'process_started',
            'recovery_pending', 'released', 'quarantined'
        )
    ),
    ADD CONSTRAINT runtime_home_lifetime_task_id_key UNIQUE (task_id);
CREATE UNIQUE INDEX runtime_home_lifetime_active_task
    ON runtime_home_lifetime(task_id) WHERE state <> 'released';

DROP INDEX runtime_home_assignment_active_binding_key;
DROP INDEX runtime_home_assignment_preparing_binding_key;
DROP INDEX runtime_home_assignment_active_home_key;
ALTER TABLE runtime_home_assignment
    DROP CONSTRAINT runtime_home_assignment_state_check;
ALTER TABLE runtime_home_assignment
    ADD CONSTRAINT runtime_home_assignment_state_check
    CHECK (state IN ('active', 'draining', 'released'));
CREATE UNIQUE INDEX runtime_home_assignment_active_binding_key
    ON runtime_home_assignment(binding_id) WHERE state IN ('active', 'draining');
CREATE UNIQUE INDEX runtime_home_assignment_active_home_key
    ON runtime_home_assignment(home_ref) WHERE state IN ('active', 'draining');

ALTER TABLE runtime_home_assignment
    DROP CONSTRAINT runtime_home_assignment_native_epoch_key;
ALTER TABLE credential_home_catalog_entry
    DROP CONSTRAINT credential_home_catalog_entry_native_epoch_key;
ALTER TABLE credential_home_catalog
    DROP CONSTRAINT credential_home_catalog_native_epoch_key;
ALTER TABLE runtime_binding
    DROP CONSTRAINT runtime_binding_native_epoch_projection_key;
DROP FUNCTION multica_credential_home_advisory_key(UUID);
