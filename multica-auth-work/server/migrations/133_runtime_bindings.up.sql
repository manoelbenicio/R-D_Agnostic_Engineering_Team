-- SPE-6: project an enrolled Runtime Session onto its existing workspace,
-- agent and runtime rows, with an optional exclusive opaque home assignment.

ALTER TABLE runtime_session_enrollment
    ADD CONSTRAINT runtime_session_enrollment_projection_key
    UNIQUE (id, session_id, workspace_id, runtime_id, agent_id);

CREATE TABLE runtime_binding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL,
    session_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    runtime_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    transport_binding TEXT NOT NULL
        CHECK (transport_binding IN ('omniroute', 'native_credential_home')),
    generation BIGINT NOT NULL DEFAULT 1 CHECK (generation > 0),
    max_concurrent_tasks INTEGER NOT NULL DEFAULT 1 CHECK (max_concurrent_tasks > 0),
    active_task_count INTEGER NOT NULL DEFAULT 0 CHECK (active_task_count >= 0),
    state TEXT NOT NULL DEFAULT 'active'
        CHECK (state IN ('active', 'draining', 'inactive', 'blocked')),
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at TIMESTAMPTZ,
    CONSTRAINT runtime_binding_enrollment_key UNIQUE (enrollment_id),
    CONSTRAINT runtime_binding_id_workspace_key
        UNIQUE (id, workspace_id),
    CONSTRAINT runtime_binding_projection_key
        UNIQUE (id, workspace_id, session_id, runtime_id, agent_id),
    CONSTRAINT runtime_binding_enrollment_projection_fkey
        FOREIGN KEY (enrollment_id, session_id, workspace_id, runtime_id, agent_id)
        REFERENCES runtime_session_enrollment(id, session_id, workspace_id, runtime_id, agent_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_binding_deactivation_check
        CHECK (
            (state = 'inactive' AND deactivated_at IS NOT NULL)
            OR (state <> 'inactive' AND deactivated_at IS NULL)
        ),
    CONSTRAINT runtime_binding_capacity_check
        CHECK (active_task_count <= max_concurrent_tasks)
);

CREATE TABLE runtime_home_assignment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    catalog_id UUID NOT NULL,
    catalog_entry_id UUID NOT NULL,
    catalog_generation BIGINT NOT NULL CHECK (catalog_generation > 0),
    home_ref UUID NOT NULL,
    binding_generation BIGINT NOT NULL CHECK (binding_generation > 0),
    state TEXT NOT NULL DEFAULT 'active'
        CHECK (state IN ('active', 'draining', 'released')),
    assigned_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,
    reason_code TEXT CHECK (reason_code IS NULL OR btrim(reason_code) <> ''),
    CONSTRAINT runtime_home_assignment_binding_workspace_fkey
        FOREIGN KEY (binding_id, workspace_id)
        REFERENCES runtime_binding(id, workspace_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_home_assignment_catalog_workspace_fkey
        FOREIGN KEY (catalog_id, workspace_id)
        REFERENCES credential_home_catalog(id, workspace_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_home_assignment_catalog_entry_fkey
        FOREIGN KEY (catalog_entry_id, catalog_id, catalog_generation, home_ref)
        REFERENCES credential_home_catalog_entry(id, catalog_id, generation, home_ref)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_home_assignment_snapshot_key
        UNIQUE (id, binding_id, home_ref, catalog_generation),
    CONSTRAINT runtime_home_assignment_release_check
        CHECK (
            (state = 'released' AND released_at IS NOT NULL)
            OR (state <> 'released' AND released_at IS NULL)
        )
);

-- Draining retains exclusivity until all active task references are gone.
CREATE UNIQUE INDEX runtime_home_assignment_active_binding_key
    ON runtime_home_assignment(binding_id)
    WHERE state IN ('active', 'draining');
CREATE UNIQUE INDEX runtime_home_assignment_active_home_key
    ON runtime_home_assignment(home_ref)
    WHERE state IN ('active', 'draining');
CREATE INDEX idx_runtime_binding_workspace
    ON runtime_binding(workspace_id, state, created_at, id);
CREATE INDEX idx_runtime_home_assignment_catalog
    ON runtime_home_assignment(catalog_id, catalog_generation, state, home_ref);

-- Cross-table CHECK constraints cannot express the mutually exclusive
-- transport rule. Enforce it before an assignment can become visible.
CREATE FUNCTION enforce_native_runtime_home_assignment()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    binding_transport TEXT;
    current_binding_generation BIGINT;
    binding_state TEXT;
    entry_state TEXT;
    entry_approved BOOLEAN;
    current_catalog_generation BIGINT;
    catalog_state TEXT;
BEGIN
    SELECT transport_binding, generation, state
      INTO binding_transport, current_binding_generation, binding_state
      FROM runtime_binding
      WHERE id = NEW.binding_id
      FOR KEY SHARE;

    IF binding_transport IS DISTINCT FROM 'native_credential_home' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'credential-home assignment requires native transport binding';
    END IF;

    IF binding_state IS DISTINCT FROM 'active' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'credential-home assignment requires an active runtime binding';
    END IF;

    IF NEW.binding_generation <> current_binding_generation THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'runtime binding generation conflict';
    END IF;

    SELECT e.state, e.approved, c.generation, c.state
      INTO entry_state, entry_approved, current_catalog_generation, catalog_state
      FROM credential_home_catalog_entry e
      JOIN credential_home_catalog c ON c.id = e.catalog_id
      WHERE e.id = NEW.catalog_entry_id;

    IF entry_state IS DISTINCT FROM 'healthy' OR entry_approved IS DISTINCT FROM true THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'credential-home assignment requires a healthy approved catalog entry';
    END IF;

    IF catalog_state IS DISTINCT FROM 'available'
       OR current_catalog_generation <> NEW.catalog_generation THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'credential-home catalog generation conflict';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER runtime_home_assignment_native_only
BEFORE INSERT OR UPDATE OF state ON runtime_home_assignment
FOR EACH ROW
WHEN (NEW.state IN ('active', 'draining'))
EXECUTE FUNCTION enforce_native_runtime_home_assignment();
