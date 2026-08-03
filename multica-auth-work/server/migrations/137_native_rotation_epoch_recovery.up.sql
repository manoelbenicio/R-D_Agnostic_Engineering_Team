-- ORQ-121: pathless native credential-home rotation epochs and durable
-- recovery. This migration contains no filesystem path, account identity,
-- credential reference/value, token, or provider response.

-- One shared advisory-key authority for admission, rotation, retirement and
-- cleanup. Every multi-home writer acquires these keys in UUID text order.
CREATE FUNCTION multica_credential_home_advisory_key(home_ref UUID)
RETURNS BIGINT
LANGUAGE sql
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
    SELECT hashtextextended(home_ref::text, 137)
$$;

-- The epoch/lifetime model is a new authority. Refuse a partial conversion of
-- already-used Migration-136 lifetime state.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM runtime_home_lifetime LIMIT 1)
       OR EXISTS (SELECT 1 FROM runtime_home_lifetime_event LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'native rotation epoch migration requires unused lifetime authority';
    END IF;
END
$$;

ALTER TABLE runtime_binding
    ADD CONSTRAINT runtime_binding_native_epoch_projection_key
    UNIQUE (
        id, workspace_id, session_id, runtime_id, agent_id,
        generation, transport_binding
    );

ALTER TABLE credential_home_catalog
    ADD CONSTRAINT credential_home_catalog_native_epoch_key
    UNIQUE (id, workspace_id, daemon_id, generation);

ALTER TABLE credential_home_catalog_entry
    ADD CONSTRAINT credential_home_catalog_entry_native_epoch_key
    UNIQUE (id, catalog_id, generation, home_ref, provider);

ALTER TABLE runtime_home_assignment
    ADD CONSTRAINT runtime_home_assignment_native_epoch_key
    UNIQUE (
        id, binding_id, workspace_id, catalog_id, catalog_entry_id,
        home_ref, catalog_generation, binding_generation
    );

DROP INDEX runtime_home_assignment_active_binding_key;
DROP INDEX runtime_home_assignment_active_home_key;
ALTER TABLE runtime_home_assignment
    DROP CONSTRAINT runtime_home_assignment_state_check;
ALTER TABLE runtime_home_assignment
    ADD CONSTRAINT runtime_home_assignment_state_check
    CHECK (state IN ('preparing', 'active', 'draining', 'released'));
CREATE UNIQUE INDEX runtime_home_assignment_active_binding_key
    ON runtime_home_assignment(binding_id)
    WHERE state = 'active';
CREATE UNIQUE INDEX runtime_home_assignment_preparing_binding_key
    ON runtime_home_assignment(binding_id)
    WHERE state = 'preparing';
CREATE UNIQUE INDEX runtime_home_assignment_active_home_key
    ON runtime_home_assignment(home_ref)
    WHERE state IN ('preparing', 'active', 'draining');

ALTER TABLE runtime_home_lifetime
    DROP CONSTRAINT runtime_home_lifetime_task_id_key;
DROP INDEX runtime_home_lifetime_active_task;
ALTER TABLE runtime_home_lifetime
    DROP CONSTRAINT runtime_home_lifetime_state_check;
ALTER TABLE runtime_home_lifetime
    ADD COLUMN home_epoch BIGINT NOT NULL CHECK (home_epoch > 0),
    ADD COLUMN agent_id UUID NOT NULL,
    ADD COLUMN runtime_id UUID NOT NULL,
    ADD COLUMN runtime_session_id UUID NOT NULL,
    ADD COLUMN provider TEXT NOT NULL
        CHECK (provider IN ('antigravity', 'codex', 'kiro')),
    ADD COLUMN transport_binding TEXT NOT NULL
        CHECK (transport_binding = 'native_credential_home'),
    ADD CONSTRAINT runtime_home_lifetime_state_check
        CHECK (
            state IN (
                'pending_local', 'acquired', 'process_started', 'retiring',
                'recovery_pending', 'released', 'quarantined'
            )
        ),
    ADD CONSTRAINT runtime_home_lifetime_epoch_key
        UNIQUE (task_id, home_epoch),
    ADD CONSTRAINT runtime_home_lifetime_epoch_identity_key
        UNIQUE (id, task_id, home_epoch, acquisition_request_id);
CREATE UNIQUE INDEX runtime_home_lifetime_active_task_epoch
    ON runtime_home_lifetime(task_id, home_epoch)
    WHERE state <> 'released';

ALTER TABLE runtime_home_lifetime_event
    DROP CONSTRAINT runtime_home_lifetime_event_state_check;
ALTER TABLE runtime_home_lifetime_event
    ADD CONSTRAINT runtime_home_lifetime_event_state_check
    CHECK (
        state IN (
            'pending_local', 'acquired', 'process_started', 'retiring',
            'recovery_pending', 'released', 'quarantined'
        )
    );

CREATE TABLE runtime_task_home_epoch (
    task_id UUID NOT NULL,
    home_epoch BIGINT NOT NULL CHECK (home_epoch > 0),
    workspace_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    runtime_id UUID NOT NULL,
    runtime_session_id UUID NOT NULL,
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    daemon_boot_id UUID NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('antigravity', 'codex', 'kiro')),
    transport_binding TEXT NOT NULL CHECK (transport_binding = 'native_credential_home'),
    runtime_binding_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    home_assignment_id UUID NOT NULL,
    catalog_id UUID NOT NULL,
    catalog_entry_id UUID NOT NULL,
    catalog_generation BIGINT NOT NULL CHECK (catalog_generation > 0),
    home_ref UUID NOT NULL,
    lifetime_id UUID NOT NULL UNIQUE,
    acquisition_request_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, home_epoch),
    CONSTRAINT runtime_task_home_epoch_task_fkey
        FOREIGN KEY (task_id, agent_id, runtime_id)
        REFERENCES agent_task_queue(id, agent_id, runtime_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_task_home_epoch_binding_fkey
        FOREIGN KEY (
            runtime_binding_id, workspace_id, runtime_session_id, runtime_id,
            agent_id, binding_generation, transport_binding
        ) REFERENCES runtime_binding (
            id, workspace_id, session_id, runtime_id, agent_id,
            generation, transport_binding
        ) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT runtime_task_home_epoch_assignment_fkey
        FOREIGN KEY (
            home_assignment_id, runtime_binding_id, workspace_id, catalog_id,
            catalog_entry_id, home_ref, catalog_generation, binding_generation
        ) REFERENCES runtime_home_assignment (
            id, binding_id, workspace_id, catalog_id, catalog_entry_id,
            home_ref, catalog_generation, binding_generation
        ) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT runtime_task_home_epoch_catalog_fkey
        FOREIGN KEY (catalog_id, workspace_id, daemon_id, catalog_generation)
        REFERENCES credential_home_catalog(id, workspace_id, daemon_id, generation)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT runtime_task_home_epoch_entry_fkey
        FOREIGN KEY (
            catalog_entry_id, catalog_id, catalog_generation, home_ref, provider
        ) REFERENCES credential_home_catalog_entry (
            id, catalog_id, generation, home_ref, provider
        ) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT runtime_task_home_epoch_lifetime_identity_key
        UNIQUE (task_id, home_epoch, lifetime_id, acquisition_request_id),
    CONSTRAINT runtime_task_home_epoch_operation_identity_key
        UNIQUE (task_id, home_epoch, lifetime_id)
);

ALTER TABLE runtime_home_lifetime
    ADD CONSTRAINT runtime_home_lifetime_epoch_fkey
        FOREIGN KEY (task_id, home_epoch, id, acquisition_request_id)
        REFERENCES runtime_task_home_epoch (
            task_id, home_epoch, lifetime_id, acquisition_request_id
        ) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE runtime_task_home_epoch
    ADD CONSTRAINT runtime_task_home_epoch_lifetime_fkey
        FOREIGN KEY (lifetime_id, task_id, home_epoch, acquisition_request_id)
        REFERENCES runtime_home_lifetime (
            id, task_id, home_epoch, acquisition_request_id
        ) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE native_rotation_operation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_request_id UUID NOT NULL UNIQUE,
    task_id UUID NOT NULL,
    runtime_binding_id UUID NOT NULL,
    current_home_epoch BIGINT NOT NULL CHECK (current_home_epoch > 0),
    current_home_ref UUID NOT NULL,
    current_assignment_id UUID NOT NULL,
    current_lifetime_id UUID NOT NULL,
    target_home_epoch BIGINT NOT NULL CHECK (target_home_epoch > 1),
    target_home_ref UUID NOT NULL,
    target_assignment_id UUID NOT NULL,
    target_lifetime_id UUID NOT NULL,
    state TEXT NOT NULL CHECK (
        state IN (
            'candidate_reserved', 'candidate_prepared',
            'swap_commit_unknown_fenced', 'committed_retirement_pending',
            'aborted_candidate_retirement_pending', 'committed_retired',
            'aborted_candidate_retired', 'quarantined'
        )
    ),
    state_version BIGINT NOT NULL DEFAULT 1 CHECK (state_version > 0),
    reason_code TEXT NOT NULL CHECK (reason_code ~ '^[a-z0-9_]{1,64}$'),
    next_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT native_rotation_operation_epoch_order
        CHECK (target_home_epoch = current_home_epoch + 1),
    CONSTRAINT native_rotation_operation_distinct_homes
        CHECK (
            target_home_ref <> current_home_ref
            AND target_assignment_id <> current_assignment_id
            AND target_lifetime_id <> current_lifetime_id
        ),
    CONSTRAINT native_rotation_operation_current_epoch_fkey
        FOREIGN KEY (task_id, current_home_epoch, current_lifetime_id)
        REFERENCES runtime_task_home_epoch(task_id, home_epoch, lifetime_id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT native_rotation_operation_target_epoch_fkey
        FOREIGN KEY (task_id, target_home_epoch, target_lifetime_id)
        REFERENCES runtime_task_home_epoch(task_id, home_epoch, lifetime_id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
);

CREATE UNIQUE INDEX native_rotation_operation_nonterminal_task
    ON native_rotation_operation(task_id)
    WHERE state NOT IN ('committed_retired', 'aborted_candidate_retired');
CREATE INDEX native_rotation_operation_due
    ON native_rotation_operation(next_attempt_at, id)
    WHERE state IN (
        'committed_retirement_pending',
        'aborted_candidate_retirement_pending'
    );

CREATE TABLE native_rotation_operation_event (
    operation_id UUID NOT NULL
        REFERENCES native_rotation_operation(id) ON DELETE RESTRICT,
    state_version BIGINT NOT NULL CHECK (state_version > 0),
    state TEXT NOT NULL,
    transition_request_id UUID NOT NULL UNIQUE,
    reason_code TEXT NOT NULL CHECK (reason_code ~ '^[a-z0-9_]{1,64}$'),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (operation_id, state_version)
);

CREATE TABLE native_rotation_retirement_attempt (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    retirement_request_id UUID NOT NULL UNIQUE,
    operation_id UUID NOT NULL
        REFERENCES native_rotation_operation(id) ON DELETE RESTRICT,
    task_id UUID NOT NULL,
    home_epoch BIGINT NOT NULL CHECK (home_epoch > 0),
    lifetime_id UUID NOT NULL,
    home_ref UUID NOT NULL,
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    daemon_boot_id UUID NOT NULL,
    runtime_session_id UUID NOT NULL,
    runtime_binding_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    scheduler_owner TEXT NOT NULL CHECK (scheduler_owner ~ '^[a-z0-9_-]{1,64}$'),
    attempt_number SMALLINT NOT NULL CHECK (attempt_number BETWEEN 1 AND 6),
    state TEXT NOT NULL CHECK (
        state IN (
            'scheduled', 'accepted', 'executing', 'succeeded',
            'retryable_failure', 'quarantined'
        )
    ),
    state_version BIGINT NOT NULL DEFAULT 1 CHECK (state_version > 0),
    lease_expires_at TIMESTAMPTZ,
    channel_binding_digest TEXT
        CHECK (channel_binding_digest IS NULL OR channel_binding_digest ~ '^[0-9a-f]{64}$'),
    request_body_digest TEXT
        CHECK (request_body_digest IS NULL OR request_body_digest ~ '^[0-9a-f]{64}$'),
    result_code TEXT CHECK (
        result_code IS NULL OR result_code ~ '^[a-z0-9_]{1,64}$'
    ),
    next_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT native_rotation_retirement_attempt_epoch_fkey
        FOREIGN KEY (task_id, home_epoch, lifetime_id)
        REFERENCES runtime_task_home_epoch(task_id, home_epoch, lifetime_id)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT native_rotation_retirement_attempt_channel_check
        CHECK (
            state = 'scheduled'
            OR (channel_binding_digest IS NOT NULL AND request_body_digest IS NOT NULL)
        )
);

CREATE UNIQUE INDEX native_rotation_retirement_attempt_inflight
    ON native_rotation_retirement_attempt(operation_id)
    WHERE state IN ('scheduled', 'accepted', 'executing');
CREATE INDEX native_rotation_retirement_attempt_due
    ON native_rotation_retirement_attempt(next_attempt_at, id)
    WHERE state IN ('scheduled', 'retryable_failure');

CREATE TABLE native_rotation_retirement_attempt_event (
    attempt_id UUID NOT NULL
        REFERENCES native_rotation_retirement_attempt(id) ON DELETE RESTRICT,
    state_version BIGINT NOT NULL CHECK (state_version > 0),
    state TEXT NOT NULL,
    transition_request_id UUID NOT NULL UNIQUE,
    reason_code TEXT NOT NULL CHECK (reason_code ~ '^[a-z0-9_]{1,64}$'),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (attempt_id, state_version)
);

CREATE FUNCTION enforce_runtime_task_home_epoch()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'native task home epochs are immutable';
    END IF;
    IF NEW.home_epoch = 1 THEN
        IF NOT EXISTS (
            SELECT 1
            FROM runtime_task_snapshot s
            WHERE s.task_id = NEW.task_id
              AND s.workspace_id = NEW.workspace_id
              AND s.agent_id = NEW.agent_id
              AND s.runtime_id = NEW.runtime_id
              AND s.runtime_session_id = NEW.runtime_session_id
              AND s.runtime_binding_id = NEW.runtime_binding_id
              AND s.binding_generation = NEW.binding_generation
              AND s.transport_binding = NEW.transport_binding
              AND s.home_assignment_id = NEW.home_assignment_id
              AND s.home_ref = NEW.home_ref
              AND s.catalog_generation = NEW.catalog_generation
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                MESSAGE = 'epoch one must match the immutable native task snapshot';
        END IF;
    ELSIF NOT EXISTS (
        SELECT 1
        FROM native_rotation_operation o
        WHERE o.task_id = NEW.task_id
          AND o.target_home_epoch = NEW.home_epoch
          AND o.target_home_ref = NEW.home_ref
          AND o.target_assignment_id = NEW.home_assignment_id
          AND o.target_lifetime_id = NEW.lifetime_id
    ) THEN
        -- Operation is inserted last in A1, so defer this test to commit.
        NULL;
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER runtime_task_home_epoch_guard
BEFORE INSERT OR UPDATE OR DELETE ON runtime_task_home_epoch
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_task_home_epoch();

CREATE FUNCTION require_later_epoch_operation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.home_epoch > 1 AND NOT EXISTS (
        SELECT 1 FROM native_rotation_operation o
        WHERE o.task_id = NEW.task_id
          AND o.target_home_epoch = NEW.home_epoch
          AND o.target_home_ref = NEW.home_ref
          AND o.target_assignment_id = NEW.home_assignment_id
          AND o.target_lifetime_id = NEW.lifetime_id
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'later native epoch requires its durable rotation reservation';
    END IF;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER runtime_task_home_epoch_operation_required
AFTER INSERT ON runtime_task_home_epoch
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION require_later_epoch_operation();

CREATE FUNCTION enforce_native_rotation_immutable_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'native rotation events are immutable';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER native_rotation_operation_event_guard
BEFORE INSERT OR UPDATE OR DELETE ON native_rotation_operation_event
FOR EACH ROW EXECUTE FUNCTION enforce_native_rotation_immutable_event();
CREATE TRIGGER native_rotation_retirement_attempt_event_guard
BEFORE INSERT OR UPDATE OR DELETE ON native_rotation_retirement_attempt_event
FOR EACH ROW EXECUTE FUNCTION enforce_native_rotation_immutable_event();

CREATE FUNCTION enforce_native_rotation_operation_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'native rotation operations cannot be deleted';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.state <> 'candidate_reserved' OR NEW.state_version <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                MESSAGE = 'native rotation operation must start reserved/v1';
        END IF;
        RETURN NEW;
    END IF;
    IF ROW(
        NEW.id, NEW.operation_request_id, NEW.task_id, NEW.runtime_binding_id,
        NEW.current_home_epoch, NEW.current_home_ref,
        NEW.current_assignment_id, NEW.current_lifetime_id,
        NEW.target_home_epoch, NEW.target_home_ref,
        NEW.target_assignment_id, NEW.target_lifetime_id, NEW.created_at
    ) IS DISTINCT FROM ROW(
        OLD.id, OLD.operation_request_id, OLD.task_id, OLD.runtime_binding_id,
        OLD.current_home_epoch, OLD.current_home_ref,
        OLD.current_assignment_id, OLD.current_lifetime_id,
        OLD.target_home_epoch, OLD.target_home_ref,
        OLD.target_assignment_id, OLD.target_lifetime_id, OLD.created_at
    ) OR NEW.state_version <> OLD.state_version + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'native rotation operation identity/version conflict';
    END IF;
    IF NOT (
        (OLD.state = 'candidate_reserved' AND NEW.state IN (
            'candidate_prepared', 'aborted_candidate_retirement_pending',
            'quarantined'
        ))
        OR (OLD.state = 'candidate_prepared' AND NEW.state IN (
            'committed_retirement_pending',
            'aborted_candidate_retirement_pending',
            'swap_commit_unknown_fenced', 'quarantined'
        ))
        OR (OLD.state = 'swap_commit_unknown_fenced' AND NEW.state IN (
            'committed_retirement_pending',
            'aborted_candidate_retirement_pending', 'quarantined'
        ))
        OR (OLD.state = 'committed_retirement_pending'
            AND NEW.state IN (
                'committed_retirement_pending', 'committed_retired', 'quarantined'
            ))
        OR (OLD.state = 'aborted_candidate_retirement_pending'
            AND NEW.state IN (
                'aborted_candidate_retirement_pending',
                'aborted_candidate_retired', 'quarantined'
            ))
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'invalid native rotation operation transition';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER native_rotation_operation_state_guard
BEFORE INSERT OR UPDATE OR DELETE ON native_rotation_operation
FOR EACH ROW EXECUTE FUNCTION enforce_native_rotation_operation_state();

CREATE FUNCTION enforce_native_retirement_attempt_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'native retirement attempts cannot be deleted';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.state <> 'scheduled' OR NEW.state_version <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                MESSAGE = 'native retirement attempt must start scheduled/v1';
        END IF;
        RETURN NEW;
    END IF;
    IF ROW(
        NEW.id, NEW.retirement_request_id, NEW.operation_id, NEW.task_id,
        NEW.home_epoch, NEW.lifetime_id, NEW.home_ref, NEW.daemon_id,
        NEW.daemon_boot_id, NEW.runtime_session_id, NEW.runtime_binding_id,
        NEW.binding_generation, NEW.attempt_number, NEW.created_at
    ) IS DISTINCT FROM ROW(
        OLD.id, OLD.retirement_request_id, OLD.operation_id, OLD.task_id,
        OLD.home_epoch, OLD.lifetime_id, OLD.home_ref, OLD.daemon_id,
        OLD.daemon_boot_id, OLD.runtime_session_id, OLD.runtime_binding_id,
        OLD.binding_generation, OLD.attempt_number, OLD.created_at
    ) OR NEW.state_version <> OLD.state_version + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'native retirement attempt identity/version conflict';
    END IF;
    IF NOT (
        (OLD.state = 'scheduled' AND NEW.state IN (
            'accepted', 'retryable_failure', 'quarantined'
        ))
        OR (OLD.state = 'accepted' AND NEW.state IN ('executing', 'quarantined'))
        OR (OLD.state = 'executing' AND NEW.state IN (
            'succeeded', 'retryable_failure', 'quarantined'
        ))
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'invalid native retirement attempt transition';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER native_rotation_retirement_attempt_state_guard
BEFORE INSERT OR UPDATE OR DELETE ON native_rotation_retirement_attempt
FOR EACH ROW EXECUTE FUNCTION enforce_native_retirement_attempt_state();

CREATE FUNCTION require_native_rotation_operation_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM native_rotation_operation_event e
        WHERE e.operation_id = NEW.id
          AND e.state_version = NEW.state_version
          AND e.state = NEW.state
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'native rotation operation requires matching event';
    END IF;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER native_rotation_operation_event_required
AFTER INSERT OR UPDATE ON native_rotation_operation
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION require_native_rotation_operation_event();

CREATE FUNCTION require_native_retirement_attempt_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM native_rotation_retirement_attempt_event e
        WHERE e.attempt_id = NEW.id
          AND e.state_version = NEW.state_version
          AND e.state = NEW.state
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'native retirement attempt requires matching event';
    END IF;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER native_rotation_retirement_attempt_event_required
AFTER INSERT OR UPDATE ON native_rotation_retirement_attempt
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION require_native_retirement_attempt_event();

-- Extend the Migration-136 guard with the exact epoch identity and retiring
-- transition. Events remain mandatory through its deferred constraint trigger.
CREATE OR REPLACE FUNCTION enforce_runtime_home_lifetime_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'runtime-home lifetime records cannot be deleted';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.state <> 'pending_local' OR NEW.state_version <> 1 OR NOT EXISTS (
            SELECT 1 FROM runtime_task_home_epoch e
            WHERE e.task_id = NEW.task_id
              AND e.home_epoch = NEW.home_epoch
              AND e.lifetime_id = NEW.id
              AND e.acquisition_request_id = NEW.acquisition_request_id
              AND e.workspace_id = NEW.workspace_id
              AND e.agent_id = NEW.agent_id
              AND e.runtime_id = NEW.runtime_id
              AND e.runtime_session_id = NEW.runtime_session_id
              AND e.daemon_id = NEW.daemon_id
              AND e.daemon_boot_id = NEW.daemon_boot_id
              AND e.provider = NEW.provider
              AND e.transport_binding = NEW.transport_binding
              AND e.runtime_binding_id = NEW.runtime_binding_id
              AND e.binding_generation = NEW.binding_generation
              AND e.home_assignment_id = NEW.home_assignment_id
              AND e.catalog_id = NEW.catalog_id
              AND e.catalog_generation = NEW.catalog_generation
              AND e.home_ref = NEW.home_ref
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                MESSAGE = 'runtime-home lifetime requires exact native epoch identity';
        END IF;
        RETURN NEW;
    END IF;
    IF ROW(
        NEW.id, NEW.acquisition_request_id, NEW.task_id, NEW.home_epoch,
        NEW.agent_id, NEW.runtime_id, NEW.runtime_session_id,
        NEW.runtime_binding_id, NEW.binding_generation, NEW.home_assignment_id,
        NEW.workspace_id, NEW.daemon_id, NEW.catalog_id,
        NEW.catalog_generation, NEW.home_ref, NEW.daemon_boot_id,
        NEW.provider, NEW.transport_binding, NEW.acquired_at
    ) IS DISTINCT FROM ROW(
        OLD.id, OLD.acquisition_request_id, OLD.task_id, OLD.home_epoch,
        OLD.agent_id, OLD.runtime_id, OLD.runtime_session_id,
        OLD.runtime_binding_id, OLD.binding_generation, OLD.home_assignment_id,
        OLD.workspace_id, OLD.daemon_id, OLD.catalog_id,
        OLD.catalog_generation, OLD.home_ref, OLD.daemon_boot_id,
        OLD.provider, OLD.transport_binding, OLD.acquired_at
    ) OR NEW.state_version <> OLD.state_version + 1 THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'runtime-home lifetime identity/version conflict';
    END IF;
    IF NOT (
        (OLD.state = 'pending_local' AND NEW.state IN ('acquired', 'recovery_pending'))
        OR (OLD.state = 'acquired' AND NEW.state IN ('process_started', 'recovery_pending'))
        OR (OLD.state = 'process_started' AND NEW.state IN ('retiring', 'recovery_pending'))
        OR (OLD.state = 'retiring' AND NEW.state IN ('released', 'quarantined'))
        OR (OLD.state = 'recovery_pending' AND NEW.state IN ('released', 'quarantined'))
        OR (OLD.state = 'quarantined' AND NEW.state = 'released')
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'invalid runtime-home lifetime transition';
    END IF;
    IF OLD.process_identity_digest IS NOT NULL
       AND NEW.process_identity_digest IS DISTINCT FROM OLD.process_identity_digest THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'runtime-home process identity is immutable once recorded';
    END IF;
    RETURN NEW;
END
$$;
