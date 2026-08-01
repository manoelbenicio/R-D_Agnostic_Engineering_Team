-- SPE-6: immutable per-binding configuration history, daemon application
-- evidence, and atomic claim-time runtime snapshots.

CREATE TABLE runtime_configuration_version (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id UUID NOT NULL REFERENCES runtime_binding(id) ON DELETE RESTRICT,
    version_number BIGINT NOT NULL CHECK (version_number > 0),
    configuration JSONB NOT NULL CHECK (jsonb_typeof(configuration) = 'object'),
    configuration_digest TEXT NOT NULL CHECK (configuration_digest ~ '^[0-9a-f]{64}$'),
    apply_class TEXT NOT NULL CHECK (apply_class IN ('hot', 'restart')),
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    request_id TEXT NOT NULL CHECK (btrim(request_id) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_configuration_version_number_key
        UNIQUE (binding_id, version_number),
    CONSTRAINT runtime_configuration_version_request_key
        UNIQUE (binding_id, request_id),
    CONSTRAINT runtime_configuration_version_binding_id_key
        UNIQUE (binding_id, id)
);

ALTER TABLE runtime_binding
    ADD COLUMN active_configuration_version_id UUID,
    ADD COLUMN effective_configuration_digest TEXT
        CHECK (effective_configuration_digest ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT runtime_binding_active_configuration_fkey
        FOREIGN KEY (id, active_configuration_version_id)
        REFERENCES runtime_configuration_version(binding_id, id)
        ON DELETE RESTRICT,
    ADD CONSTRAINT runtime_binding_active_configuration_digest_check
        CHECK (
            (active_configuration_version_id IS NULL AND effective_configuration_digest IS NULL)
            OR (active_configuration_version_id IS NOT NULL AND effective_configuration_digest IS NOT NULL)
        );

CREATE TABLE runtime_configuration_activation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id UUID NOT NULL REFERENCES runtime_binding(id) ON DELETE RESTRICT,
    previous_version_id UUID,
    new_version_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    effective_configuration_digest TEXT NOT NULL
        CHECK (effective_configuration_digest ~ '^[0-9a-f]{64}$'),
    capability_digest TEXT NOT NULL CHECK (capability_digest ~ '^[0-9a-f]{64}$'),
    apply_class TEXT NOT NULL CHECK (apply_class IN ('hot', 'restart')),
    actor_id UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    request_id TEXT NOT NULL CHECK (btrim(request_id) <> ''),
    correlation_id TEXT CHECK (correlation_id IS NULL OR btrim(correlation_id) <> ''),
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    activated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_configuration_activation_request_key
        UNIQUE (binding_id, request_id),
    CONSTRAINT runtime_configuration_activation_previous_version_fkey
        FOREIGN KEY (binding_id, previous_version_id)
        REFERENCES runtime_configuration_version(binding_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_configuration_activation_new_version_fkey
        FOREIGN KEY (binding_id, new_version_id)
        REFERENCES runtime_configuration_version(binding_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_configuration_activation_change_check
        CHECK (previous_version_id IS NULL OR previous_version_id <> new_version_id)
);

CREATE TABLE runtime_configuration_acknowledgement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id UUID NOT NULL REFERENCES runtime_binding(id) ON DELETE RESTRICT,
    configuration_version_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    status TEXT NOT NULL
        CHECK (status IN ('applied', 'restart_required', 'drift_detected', 'rejected')),
    applied_configuration_digest TEXT
        CHECK (applied_configuration_digest IS NULL OR applied_configuration_digest ~ '^[0-9a-f]{64}$'),
    reason_code TEXT CHECK (reason_code IS NULL OR btrim(reason_code) <> ''),
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_configuration_acknowledgement_version_fkey
        FOREIGN KEY (binding_id, configuration_version_id)
        REFERENCES runtime_configuration_version(binding_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_configuration_acknowledgement_digest_check
        CHECK (
            (status = 'applied' AND applied_configuration_digest IS NOT NULL)
            OR status <> 'applied'
        ),
    CONSTRAINT runtime_configuration_acknowledgement_dedupe_key
        UNIQUE (binding_id, configuration_version_id, binding_generation, status, acknowledged_at)
);

ALTER TABLE agent_task_queue
    ADD CONSTRAINT agent_task_queue_runtime_snapshot_projection_key
    UNIQUE (id, agent_id, runtime_id);

CREATE TABLE runtime_task_snapshot (
    task_id UUID PRIMARY KEY,
    runtime_session_id UUID NOT NULL,
    runtime_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    runtime_standard_version_id UUID NOT NULL REFERENCES runtime_standard_version(id) ON DELETE RESTRICT,
    runtime_configuration_version_id UUID NOT NULL,
    effective_configuration_digest TEXT NOT NULL
        CHECK (effective_configuration_digest ~ '^[0-9a-f]{64}$'),
    runtime_binding_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    transport_binding TEXT NOT NULL
        CHECK (transport_binding IN ('omniroute', 'native_credential_home')),
    home_assignment_id UUID,
    home_ref UUID,
    catalog_generation BIGINT CHECK (catalog_generation > 0),
    capability_digest TEXT NOT NULL CHECK (capability_digest ~ '^[0-9a-f]{64}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_task_snapshot_task_projection_fkey
        FOREIGN KEY (task_id, agent_id, runtime_id)
        REFERENCES agent_task_queue(id, agent_id, runtime_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_task_snapshot_binding_projection_fkey
        FOREIGN KEY (runtime_binding_id, workspace_id, runtime_session_id, runtime_id, agent_id)
        REFERENCES runtime_binding(id, workspace_id, session_id, runtime_id, agent_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_task_snapshot_configuration_fkey
        FOREIGN KEY (runtime_binding_id, runtime_configuration_version_id)
        REFERENCES runtime_configuration_version(binding_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_task_snapshot_home_assignment_fkey
        FOREIGN KEY (home_assignment_id, runtime_binding_id, home_ref, catalog_generation)
        REFERENCES runtime_home_assignment(id, binding_id, home_ref, catalog_generation)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_task_snapshot_transport_home_check
        CHECK (
            (transport_binding = 'omniroute'
             AND home_assignment_id IS NULL AND home_ref IS NULL AND catalog_generation IS NULL)
            OR
            (transport_binding = 'native_credential_home'
             AND home_assignment_id IS NOT NULL AND home_ref IS NOT NULL AND catalog_generation IS NOT NULL)
        )
);

-- A release is separate immutable evidence: task snapshots themselves are
-- never rewritten. The task primary key makes decrement idempotent.
CREATE TABLE runtime_task_snapshot_release (
    task_id UUID PRIMARY KEY REFERENCES runtime_task_snapshot(task_id) ON DELETE RESTRICT,
    runtime_binding_id UUID NOT NULL REFERENCES runtime_binding(id) ON DELETE RESTRICT,
    released_by UUID REFERENCES "user"(id) ON DELETE RESTRICT,
    reason_code TEXT NOT NULL CHECK (btrim(reason_code) <> ''),
    released_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_task_snapshot_release_task_binding_key
        UNIQUE (task_id, runtime_binding_id)
);

-- Snapshot inserts consume capacity reserved on the locked binding. Direct
-- inserts cannot bypass the counter/concurrency fence. A duplicate task is
-- allowed through to ON CONFLICT so the idempotent query can compare its
-- immutable pinned identity.
CREATE FUNCTION enforce_runtime_task_snapshot_reservation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    binding_row runtime_binding%ROWTYPE;
    retained_count BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM runtime_task_snapshot WHERE task_id = NEW.task_id) THEN
        RETURN NEW;
    END IF;

    SELECT * INTO binding_row
    FROM runtime_binding
    WHERE id = NEW.runtime_binding_id
    FOR UPDATE;

    IF NOT FOUND
       OR binding_row.state <> 'active'
       OR binding_row.generation <> NEW.binding_generation
       OR binding_row.transport_binding <> NEW.transport_binding THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'runtime snapshot binding generation conflict';
    END IF;

    SELECT count(*)
      INTO retained_count
      FROM runtime_task_snapshot s
      LEFT JOIN runtime_task_snapshot_release r ON r.task_id = s.task_id
      WHERE s.runtime_binding_id = NEW.runtime_binding_id
        AND r.task_id IS NULL;

    IF retained_count >= binding_row.active_task_count THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'runtime snapshot requires reserved binding capacity';
    END IF;

    RETURN NEW;
END;
$$;

CREATE FUNCTION enforce_runtime_task_snapshot_terminal_release()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    task_status TEXT;
    snapshot_binding_id UUID;
BEGIN
    SELECT t.status, s.runtime_binding_id
      INTO task_status, snapshot_binding_id
      FROM runtime_task_snapshot s
      JOIN agent_task_queue t ON t.id = s.task_id
      WHERE s.task_id = NEW.task_id
      FOR KEY SHARE OF t;

    IF snapshot_binding_id IS DISTINCT FROM NEW.runtime_binding_id THEN
        RAISE EXCEPTION USING
            ERRCODE = '23503',
            MESSAGE = 'runtime snapshot release binding conflict';
    END IF;

    IF task_status IN ('queued', 'dispatched', 'running', 'waiting_local_directory') THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'active runtime snapshot cannot be released';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER runtime_task_snapshot_requires_reservation
BEFORE INSERT ON runtime_task_snapshot
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_task_snapshot_reservation();

CREATE TRIGGER runtime_task_snapshot_release_terminal_only
BEFORE INSERT ON runtime_task_snapshot_release
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_task_snapshot_terminal_release();

CREATE INDEX idx_runtime_configuration_version_binding
    ON runtime_configuration_version(binding_id, version_number DESC);
CREATE INDEX idx_runtime_configuration_activation_binding
    ON runtime_configuration_activation(binding_id, activated_at DESC, id);
CREATE INDEX idx_runtime_configuration_acknowledgement_binding
    ON runtime_configuration_acknowledgement(binding_id, acknowledged_at DESC, id);
CREATE INDEX idx_runtime_task_snapshot_binding
    ON runtime_task_snapshot(runtime_binding_id, binding_generation, created_at, task_id);
CREATE INDEX idx_runtime_task_snapshot_home
    ON runtime_task_snapshot(home_ref, catalog_generation)
    WHERE home_ref IS NOT NULL;
CREATE INDEX idx_runtime_task_snapshot_release_binding
    ON runtime_task_snapshot_release(runtime_binding_id, released_at, task_id);

CREATE TRIGGER runtime_configuration_version_immutable
BEFORE UPDATE OR DELETE ON runtime_configuration_version
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER runtime_configuration_activation_immutable
BEFORE UPDATE OR DELETE ON runtime_configuration_activation
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER runtime_configuration_acknowledgement_immutable
BEFORE UPDATE OR DELETE ON runtime_configuration_acknowledgement
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER runtime_task_snapshot_immutable
BEFORE UPDATE OR DELETE ON runtime_task_snapshot
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER runtime_task_snapshot_release_immutable
BEFORE UPDATE OR DELETE ON runtime_task_snapshot_release
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();
