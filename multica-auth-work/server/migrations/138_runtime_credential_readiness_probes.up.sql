-- ORQ-104 C1: immutable, pathless native credential-readiness probe
-- persistence. Migration 137 remains the sole identity, advisory-lock, and
-- rotation-fence authority.

CREATE TABLE runtime_credential_readiness_probe_request (
    probe_request_id UUID PRIMARY KEY,
    request_digest TEXT NOT NULL CHECK (request_digest ~ '^[0-9a-f]{64}$'),
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
    lifetime_id UUID NOT NULL,
    acquisition_request_id UUID NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    request_expires_at TIMESTAMPTZ NOT NULL
        DEFAULT transaction_timestamp() + interval '5 minutes',
    CONSTRAINT runtime_credential_readiness_probe_request_epoch_fkey
        FOREIGN KEY (task_id, home_epoch, lifetime_id, acquisition_request_id)
        REFERENCES runtime_task_home_epoch (
            task_id, home_epoch, lifetime_id, acquisition_request_id
        ) ON DELETE RESTRICT,
    CONSTRAINT runtime_credential_readiness_probe_request_window_check
        CHECK (request_expires_at = issued_at + interval '5 minutes')
);

CREATE INDEX runtime_credential_readiness_probe_request_identity
    ON runtime_credential_readiness_probe_request (
        task_id, home_epoch, runtime_binding_id, binding_generation,
        home_assignment_id, catalog_generation, daemon_boot_id
    );

CREATE TABLE runtime_credential_readiness_probe_result (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    probe_request_id UUID NOT NULL UNIQUE
        REFERENCES runtime_credential_readiness_probe_request(probe_request_id)
        ON DELETE RESTRICT,
    attestation_id UUID NOT NULL UNIQUE
        REFERENCES runtime_credential_readiness_attestation(id)
        ON DELETE RESTRICT,
    result_digest TEXT NOT NULL CHECK (result_digest ~ '^[0-9a-f]{64}$'),
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
    probe_observed_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
    expires_at TIMESTAMPTZ NOT NULL
        DEFAULT transaction_timestamp() + interval '5 minutes',
    CONSTRAINT runtime_credential_readiness_probe_result_state_reason_check
        CHECK (
            (state = 'ready' AND reason_code = 'ready')
            OR (state = 'unready' AND reason_code <> 'ready')
        ),
    CONSTRAINT runtime_credential_readiness_probe_result_window_check
        CHECK (expires_at = accepted_at + interval '5 minutes')
);

CREATE FUNCTION enforce_runtime_credential_readiness_probe_request()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    epoch_row runtime_task_home_epoch%ROWTYPE;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'credential-readiness probe requests are immutable';
    END IF;

    NEW.issued_at := transaction_timestamp();
    NEW.request_expires_at := transaction_timestamp() + interval '5 minutes';

    PERFORM pg_advisory_xact_lock(
        multica_credential_home_advisory_key(NEW.home_ref)
    );

    SELECT e.* INTO epoch_row
    FROM runtime_task_home_epoch e
    WHERE e.task_id = NEW.task_id
      AND e.home_epoch = NEW.home_epoch
    FOR UPDATE;

    IF NOT FOUND
       OR ROW(
            epoch_row.workspace_id, epoch_row.agent_id, epoch_row.runtime_id,
            epoch_row.runtime_session_id, epoch_row.daemon_id,
            epoch_row.daemon_boot_id, epoch_row.provider,
            epoch_row.transport_binding, epoch_row.runtime_binding_id,
            epoch_row.binding_generation, epoch_row.home_assignment_id,
            epoch_row.catalog_id, epoch_row.catalog_entry_id,
            epoch_row.catalog_generation, epoch_row.home_ref,
            epoch_row.lifetime_id, epoch_row.acquisition_request_id
       ) IS DISTINCT FROM ROW(
            NEW.workspace_id, NEW.agent_id, NEW.runtime_id,
            NEW.runtime_session_id, NEW.daemon_id, NEW.daemon_boot_id,
            NEW.provider, NEW.transport_binding, NEW.runtime_binding_id,
            NEW.binding_generation, NEW.home_assignment_id, NEW.catalog_id,
            NEW.catalog_entry_id, NEW.catalog_generation, NEW.home_ref,
            NEW.lifetime_id, NEW.acquisition_request_id
       )
       OR NOT EXISTS (
            SELECT 1
            FROM runtime_binding b
            JOIN runtime_home_assignment a
              ON a.id = NEW.home_assignment_id
             AND a.binding_id = b.id
            JOIN credential_home_catalog c
              ON c.id = NEW.catalog_id
             AND c.workspace_id = NEW.workspace_id
            JOIN credential_home_catalog_entry ce
              ON ce.id = NEW.catalog_entry_id
             AND ce.catalog_id = c.id
            JOIN runtime_home_lifetime l
              ON l.id = NEW.lifetime_id
             AND l.task_id = NEW.task_id
             AND l.home_epoch = NEW.home_epoch
            WHERE b.id = NEW.runtime_binding_id
              AND b.workspace_id = NEW.workspace_id
              AND b.session_id = NEW.runtime_session_id
              AND b.runtime_id = NEW.runtime_id
              AND b.agent_id = NEW.agent_id
              AND b.generation = NEW.binding_generation
              AND b.transport_binding = 'native_credential_home'
              AND b.state = 'active'
              AND a.workspace_id = NEW.workspace_id
              AND a.catalog_id = NEW.catalog_id
              AND a.catalog_entry_id = NEW.catalog_entry_id
              AND a.home_ref = NEW.home_ref
              AND a.catalog_generation = NEW.catalog_generation
              AND a.binding_generation = NEW.binding_generation
              AND a.state = 'active'
              AND c.daemon_id = NEW.daemon_id
              AND c.generation = NEW.catalog_generation
              AND c.state = 'available'
              AND ce.generation = NEW.catalog_generation
              AND ce.home_ref = NEW.home_ref
              AND ce.provider = NEW.provider
              AND ce.state = 'healthy'
              AND ce.approved
              AND l.state = 'process_started'
       )
       OR EXISTS (
            SELECT 1
            FROM native_rotation_operation o
            WHERE o.task_id = NEW.task_id
              AND (
                    o.current_home_ref = NEW.home_ref
                    OR o.target_home_ref = NEW.home_ref
              )
              AND o.state NOT IN (
                    'committed_retired', 'aborted_candidate_retired'
              )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'probe request identity is not current native authority';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER runtime_credential_readiness_probe_request_guard
BEFORE INSERT OR UPDATE OR DELETE
ON runtime_credential_readiness_probe_request
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_credential_readiness_probe_request();

CREATE FUNCTION enforce_runtime_credential_readiness_probe_result()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    request_row runtime_credential_readiness_probe_request%ROWTYPE;
    attestation_row runtime_credential_readiness_attestation%ROWTYPE;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'credential-readiness probe results are immutable';
    END IF;

    SELECT r.* INTO request_row
    FROM runtime_credential_readiness_probe_request r
    WHERE r.probe_request_id = NEW.probe_request_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '23503',
            MESSAGE = 'probe result requires its immutable request';
    END IF;

    PERFORM pg_advisory_xact_lock(
        multica_credential_home_advisory_key(request_row.home_ref)
    );

    SELECT r.* INTO request_row
    FROM runtime_credential_readiness_probe_request r
    WHERE r.probe_request_id = NEW.probe_request_id
    FOR UPDATE;

    NEW.accepted_at := transaction_timestamp();
    NEW.expires_at := transaction_timestamp() + interval '5 minutes';

    SELECT a.* INTO attestation_row
    FROM runtime_credential_readiness_attestation a
    WHERE a.id = NEW.attestation_id
      AND a.probe_request_id = NEW.probe_request_id;

    IF NOT FOUND
       OR NEW.probe_observed_at < request_row.issued_at
       OR NEW.probe_observed_at > transaction_timestamp()
       OR transaction_timestamp() > request_row.request_expires_at
       OR ROW(
            attestation_row.workspace_id, attestation_row.agent_id,
            attestation_row.runtime_id, attestation_row.runtime_session_id,
            attestation_row.runtime_binding_id,
            attestation_row.binding_generation,
            attestation_row.home_assignment_id, attestation_row.catalog_id,
            attestation_row.catalog_generation, attestation_row.home_ref,
            attestation_row.provider, attestation_row.daemon_id,
            attestation_row.daemon_boot_id, attestation_row.state,
            attestation_row.reason_code, attestation_row.observed_at,
            attestation_row.expires_at
       ) IS DISTINCT FROM ROW(
            request_row.workspace_id, request_row.agent_id,
            request_row.runtime_id, request_row.runtime_session_id,
            request_row.runtime_binding_id, request_row.binding_generation,
            request_row.home_assignment_id, request_row.catalog_id,
            request_row.catalog_generation, request_row.home_ref,
            request_row.provider, request_row.daemon_id,
            request_row.daemon_boot_id, NEW.state, NEW.reason_code,
            NEW.probe_observed_at, NEW.expires_at
       )
       OR NOT EXISTS (
            SELECT 1
            FROM runtime_task_home_epoch e
            JOIN runtime_binding b ON b.id = e.runtime_binding_id
            JOIN runtime_home_assignment a ON a.id = e.home_assignment_id
            JOIN credential_home_catalog c ON c.id = e.catalog_id
            JOIN credential_home_catalog_entry ce ON ce.id = e.catalog_entry_id
            JOIN runtime_home_lifetime l ON l.id = e.lifetime_id
            WHERE e.task_id = request_row.task_id
              AND e.home_epoch = request_row.home_epoch
              AND e.daemon_boot_id = request_row.daemon_boot_id
              AND b.generation = request_row.binding_generation
              AND b.state = 'active'
              AND b.transport_binding = 'native_credential_home'
              AND a.id = request_row.home_assignment_id
              AND a.state = 'active'
              AND c.generation = request_row.catalog_generation
              AND c.state = 'available'
              AND ce.generation = request_row.catalog_generation
              AND ce.state = 'healthy'
              AND ce.approved
              AND l.state = 'process_started'
       )
       OR EXISTS (
            SELECT 1 FROM native_rotation_operation o
            WHERE o.task_id = request_row.task_id
              AND (
                    o.current_home_ref = request_row.home_ref
                    OR o.target_home_ref = request_row.home_ref
              )
              AND o.state NOT IN (
                    'committed_retired', 'aborted_candidate_retired'
              )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'probe result is not current bounded native authority';
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER runtime_credential_readiness_probe_result_guard
BEFORE INSERT OR UPDATE OR DELETE
ON runtime_credential_readiness_probe_result
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_credential_readiness_probe_result();
