-- ORQ-109: shared pathless PostgreSQL authority for native credential-home
-- occupancy and fresh-session readiness. Filesystem paths, credential values,
-- account names, and provider response bodies are deliberately not representable.

-- This projection key lets both new relations bind to the exact immutable
-- assignment identity without preventing a runtime binding generation change.
ALTER TABLE runtime_home_assignment
    ADD CONSTRAINT runtime_home_assignment_state_projection_key
    UNIQUE (
        id, binding_id, workspace_id, catalog_id, home_ref,
        catalog_generation, binding_generation
    );

CREATE TABLE runtime_home_lifetime (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    acquisition_request_id UUID NOT NULL UNIQUE,
    task_id UUID NOT NULL UNIQUE
        REFERENCES runtime_task_snapshot(task_id) ON DELETE RESTRICT,
    runtime_binding_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    home_assignment_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    catalog_id UUID NOT NULL,
    catalog_generation BIGINT NOT NULL CHECK (catalog_generation > 0),
    home_ref UUID NOT NULL,
    daemon_boot_id UUID NOT NULL,
    state TEXT NOT NULL CHECK (
        state IN (
            'pending_local', 'acquired', 'process_started',
            'recovery_pending', 'released', 'quarantined'
        )
    ),
    process_identity_digest TEXT
        CHECK (
            process_identity_digest IS NULL
            OR process_identity_digest ~ '^[0-9a-f]{64}$'
        ),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    process_started_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    state_version SMALLINT NOT NULL DEFAULT 1 CHECK (state_version > 0),
    CONSTRAINT runtime_home_lifetime_assignment_fkey
        FOREIGN KEY (
            home_assignment_id, runtime_binding_id, workspace_id, catalog_id,
            home_ref, catalog_generation, binding_generation
        ) REFERENCES runtime_home_assignment (
            id, binding_id, workspace_id, catalog_id,
            home_ref, catalog_generation, binding_generation
        ) ON DELETE RESTRICT,
    CONSTRAINT runtime_home_lifetime_catalog_fkey
        FOREIGN KEY (catalog_id, workspace_id)
        REFERENCES credential_home_catalog(id, workspace_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_home_lifetime_release_check
        CHECK (
            (state = 'released' AND released_at IS NOT NULL)
            OR (state <> 'released' AND released_at IS NULL)
        ),
    CONSTRAINT runtime_home_lifetime_process_check
        CHECK (
            state <> 'process_started'
            OR (process_started_at IS NOT NULL AND process_identity_digest IS NOT NULL)
        )
);

CREATE UNIQUE INDEX runtime_home_lifetime_active_task
    ON runtime_home_lifetime(task_id)
    WHERE state <> 'released';

CREATE INDEX runtime_home_lifetime_active_home
    ON runtime_home_lifetime(catalog_id, home_ref, state, id);

CREATE TABLE runtime_home_lifetime_event (
    lifetime_id UUID NOT NULL
        REFERENCES runtime_home_lifetime(id) ON DELETE RESTRICT,
    state_version SMALLINT NOT NULL CHECK (state_version > 0),
    state TEXT NOT NULL CHECK (
        state IN (
            'pending_local', 'acquired', 'process_started',
            'recovery_pending', 'released', 'quarantined'
        )
    ),
    transition_request_id UUID NOT NULL UNIQUE,
    reason_code TEXT CHECK (
        reason_code IS NULL OR reason_code ~ '^[a-z0-9_]{1,64}$'
    ),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (lifetime_id, state_version)
);

-- Each readiness row is one immutable, value-free probe result. Freshness and
-- current-generation authority are evaluated from the exact pinned identity;
-- heartbeat state cannot insert or extend an attestation.
CREATE TABLE runtime_credential_readiness_attestation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    probe_request_id UUID NOT NULL UNIQUE,
    workspace_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    runtime_id UUID NOT NULL,
    runtime_session_id UUID NOT NULL,
    runtime_binding_id UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    home_assignment_id UUID NOT NULL,
    catalog_id UUID NOT NULL,
    catalog_generation BIGINT NOT NULL CHECK (catalog_generation > 0),
    home_ref UUID NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('antigravity', 'codex', 'kiro')),
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    daemon_boot_id UUID NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('ready', 'unready')),
    reason_code TEXT NOT NULL CHECK (
        reason_code IN (
            'ready',
            'provider_unreachable',
            'authentication_rejected',
            'subscription_exhausted',
            'provider_rate_limited',
            'probe_timeout',
            'unsupported_safe_probe',
            'assignment_mismatch',
            'daemon_unavailable',
            'internal_error'
        )
    ),
    observed_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_credential_readiness_state_reason_check
        CHECK (
            (state = 'ready' AND reason_code = 'ready')
            OR (state = 'unready' AND reason_code <> 'ready')
        ),
    CONSTRAINT runtime_credential_readiness_expiry_check
        CHECK (expires_at > observed_at),
    CONSTRAINT runtime_credential_readiness_binding_fkey
        FOREIGN KEY (
            runtime_binding_id, workspace_id, runtime_session_id,
            runtime_id, agent_id
        ) REFERENCES runtime_binding (
            id, workspace_id, session_id, runtime_id, agent_id
        ) ON DELETE RESTRICT,
    CONSTRAINT runtime_credential_readiness_assignment_fkey
        FOREIGN KEY (
            home_assignment_id, runtime_binding_id, workspace_id, catalog_id,
            home_ref, catalog_generation, binding_generation
        ) REFERENCES runtime_home_assignment (
            id, binding_id, workspace_id, catalog_id,
            home_ref, catalog_generation, binding_generation
        ) ON DELETE RESTRICT,
    CONSTRAINT runtime_credential_readiness_catalog_fkey
        FOREIGN KEY (catalog_id, workspace_id)
        REFERENCES credential_home_catalog(id, workspace_id)
        ON DELETE RESTRICT
);

CREATE INDEX runtime_credential_readiness_current_identity
    ON runtime_credential_readiness_attestation (
        runtime_binding_id, binding_generation, home_assignment_id,
        catalog_generation, daemon_boot_id, expires_at DESC, id
    );

CREATE FUNCTION enforce_runtime_home_lifetime_state()
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

CREATE TRIGGER runtime_home_lifetime_state_guard
BEFORE INSERT OR UPDATE OR DELETE ON runtime_home_lifetime
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_home_lifetime_state();

CREATE FUNCTION require_runtime_home_lifetime_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM runtime_home_lifetime_event e
        WHERE e.lifetime_id = NEW.id
          AND e.state_version = NEW.state_version
          AND e.state = NEW.state
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'runtime-home lifetime transition requires matching event';
    END IF;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER runtime_home_lifetime_event_required
AFTER INSERT OR UPDATE ON runtime_home_lifetime
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION require_runtime_home_lifetime_event();

CREATE FUNCTION enforce_runtime_home_lifetime_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    lifetime_state TEXT;
    lifetime_version SMALLINT;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'runtime-home lifetime events are immutable';
    END IF;

    SELECT state, state_version
      INTO lifetime_state, lifetime_version
      FROM runtime_home_lifetime
      WHERE id = NEW.lifetime_id
      FOR KEY SHARE;

    IF NOT FOUND
       OR NEW.state_version > lifetime_version
       OR (NEW.state_version = lifetime_version AND NEW.state <> lifetime_state) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'runtime-home lifetime event does not match lifetime state';
    END IF;

    RETURN NEW;
END
$$;

CREATE TRIGGER runtime_home_lifetime_event_guard
BEFORE INSERT OR UPDATE OR DELETE ON runtime_home_lifetime_event
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_home_lifetime_event();

CREATE FUNCTION enforce_runtime_credential_readiness_attestation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    expected_provider TEXT;
    expected_daemon_id TEXT;
    binding_transport TEXT;
    binding_state TEXT;
    current_binding_generation BIGINT;
    assignment_state TEXT;
    catalog_state TEXT;
    current_catalog_generation BIGINT;
    entry_state TEXT;
    entry_approved BOOLEAN;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'credential-readiness attestations are immutable';
    END IF;

    SELECT
        e.provider, c.daemon_id, b.transport_binding, b.state, b.generation,
        a.state, c.state, c.generation, e.state, e.approved
      INTO
        expected_provider, expected_daemon_id, binding_transport, binding_state,
        current_binding_generation, assignment_state, catalog_state,
        current_catalog_generation, entry_state, entry_approved
      FROM runtime_home_assignment a
      JOIN runtime_binding b
        ON b.id = a.binding_id
       AND b.workspace_id = a.workspace_id
      JOIN credential_home_catalog c ON c.id = a.catalog_id
      JOIN credential_home_catalog_entry e
        ON e.id = a.catalog_entry_id
       AND e.catalog_id = a.catalog_id
       AND e.generation = a.catalog_generation
       AND e.home_ref = a.home_ref
      WHERE a.id = NEW.home_assignment_id
        AND a.binding_id = NEW.runtime_binding_id
        AND a.workspace_id = NEW.workspace_id
        AND a.catalog_id = NEW.catalog_id
        AND a.catalog_generation = NEW.catalog_generation
        AND a.home_ref = NEW.home_ref
        AND a.binding_generation = NEW.binding_generation;

    IF NOT FOUND
       OR expected_provider IS DISTINCT FROM NEW.provider
       OR expected_daemon_id IS DISTINCT FROM NEW.daemon_id
       OR binding_transport IS DISTINCT FROM 'native_credential_home'
       OR binding_state IS DISTINCT FROM 'active'
       OR current_binding_generation IS DISTINCT FROM NEW.binding_generation
       OR assignment_state IS DISTINCT FROM 'active'
       OR catalog_state IS DISTINCT FROM 'available'
       OR current_catalog_generation IS DISTINCT FROM NEW.catalog_generation
       OR entry_state IS DISTINCT FROM 'healthy'
       OR entry_approved IS DISTINCT FROM true
       OR NEW.observed_at > statement_timestamp()
       OR NEW.expires_at <= statement_timestamp() THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'credential-readiness authority is not current native identity';
    END IF;

    RETURN NEW;
END
$$;

CREATE TRIGGER runtime_credential_readiness_attestation_guard
BEFORE INSERT OR UPDATE OR DELETE ON runtime_credential_readiness_attestation
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_credential_readiness_attestation();
