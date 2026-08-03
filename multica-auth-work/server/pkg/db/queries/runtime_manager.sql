-- Runtime Standards ---------------------------------------------------------

-- name: CreateRuntimeStandard :one
INSERT INTO runtime_standard (owner_id, name, description, request_id)
VALUES (@owner_id, @name, sqlc.narg(description), @request_id)
ON CONFLICT (owner_id, request_id) DO UPDATE
SET request_id = EXCLUDED.request_id
RETURNING *;

-- name: ListRuntimeStandardsByOwner :many
SELECT *
FROM runtime_standard
WHERE owner_id = @owner_id
ORDER BY created_at, id;

-- name: GetRuntimeStandardForOwner :one
SELECT *
FROM runtime_standard
WHERE id = @id AND owner_id = @owner_id;

-- name: LockRuntimeStandardForOwner :one
SELECT *
FROM runtime_standard
WHERE id = @id AND owner_id = @owner_id
FOR UPDATE;

-- name: GetRuntimeStandardVersionByRequestID :one
SELECT *
FROM runtime_standard_version
WHERE standard_id = @standard_id AND request_id = @request_id;

-- name: CreateRuntimeStandardVersion :one
INSERT INTO runtime_standard_version (
    standard_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason, request_id
)
VALUES (
    @standard_id, @version_number, @configuration, @configuration_digest,
    @apply_class, @created_by, @reason, @request_id
)
RETURNING *;

-- name: ListRuntimeStandardVersions :many
SELECT *
FROM runtime_standard_version
WHERE standard_id = @standard_id
ORDER BY version_number DESC;

-- name: ActivateRuntimeStandardVersion :one
UPDATE runtime_standard AS rs
SET active_version_id = @new_version_id, updated_at = now()
WHERE rs.id = @standard_id
  AND rs.active_version_id IS NOT DISTINCT FROM sqlc.narg(expected_active_version_id)::uuid
  AND EXISTS (
      SELECT 1 FROM runtime_standard_version AS rsv
      WHERE rsv.standard_id = @standard_id AND rsv.id = @new_version_id
  )
RETURNING *;

-- name: RecordRuntimeStandardActivation :one
INSERT INTO runtime_standard_activation (
    standard_id, previous_version_id, new_version_id, actor_id, request_id,
    correlation_id, reason, capability_digest
)
VALUES (
    @standard_id, sqlc.narg(previous_version_id), @new_version_id, @actor_id,
    @request_id, sqlc.narg(correlation_id), @reason, @capability_digest
)
RETURNING *;

-- name: GetRuntimeStandardActivationByRequestID :one
SELECT *
FROM runtime_standard_activation
WHERE standard_id = @standard_id AND request_id = @request_id;

-- Runtime Sessions and enrollment ------------------------------------------

-- name: CreateRuntimeSession :one
INSERT INTO runtime_session (
    owner_id, standard_id, name, provider, runtime_kind, created_by
)
VALUES (
    @owner_id, @standard_id, @name, @provider, @runtime_kind, @created_by
)
RETURNING *;

-- name: ListRuntimeSessionsByOwner :many
SELECT *
FROM runtime_session
WHERE owner_id = @owner_id
ORDER BY created_at, id;

-- name: GetRuntimeSessionForOwner :one
SELECT *
FROM runtime_session
WHERE id = @id AND owner_id = @owner_id;

-- name: CreateRuntimeSessionEnrollment :one
INSERT INTO runtime_session_enrollment (
    session_id, workspace_id, runtime_id, agent_id, enrolled_by
)
VALUES (
    @session_id, @workspace_id, @runtime_id, @agent_id, @enrolled_by
)
RETURNING *;

-- name: GetRuntimeSessionEnrollmentForWorkspace :one
SELECT *
FROM runtime_session_enrollment
WHERE workspace_id = @workspace_id AND session_id = @session_id;

-- name: SetRuntimeSessionEnrollmentState :one
UPDATE runtime_session_enrollment
SET state = @state,
    deactivated_at = CASE WHEN @state::text = 'inactive' THEN now() ELSE NULL END,
    updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id
RETURNING *;

-- Credential-home catalog ---------------------------------------------------

-- name: CreateCredentialHomeCatalog :one
INSERT INTO credential_home_catalog (workspace_id, daemon_id)
VALUES (@workspace_id, @daemon_id)
ON CONFLICT (workspace_id, daemon_id) DO NOTHING
RETURNING *;

-- name: GetCredentialHomeCatalogForWorkspace :one
SELECT *
FROM credential_home_catalog
WHERE id = @id AND workspace_id = @workspace_id;

-- name: LockCredentialHomeCatalog :one
SELECT *
FROM credential_home_catalog
WHERE id = @id AND workspace_id = @workspace_id
FOR UPDATE;

-- name: LoadCredentialHomeCatalogLifecycle :many
SELECT l.*
FROM credential_home_catalog_lifecycle l
JOIN credential_home_catalog c ON c.id = l.catalog_id
WHERE c.id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND l.provider = @provider
ORDER BY l.home_ref;

-- name: BeginCredentialHomeCatalogReconciliation :one
UPDATE credential_home_catalog
SET state = 'reconciling', updated_at = now()
WHERE id = @id
  AND workspace_id = @workspace_id
  AND generation = @expected_generation
  AND state <> 'reconciling'
RETURNING *;

-- name: AdvanceCredentialHomeCatalogLifecycleGeneration :one
-- Run after LockCredentialHomeCatalog and before saving lifecycle records in
-- the same transaction. The published generation remains unchanged until all
-- lifecycle rows have been durably written.
UPDATE credential_home_catalog
SET lifecycle_generation = sqlc.arg(lifecycle_generation),
    updated_at = now()
WHERE id = sqlc.arg(catalog_id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND generation = sqlc.arg(expected_published_generation)
  AND lifecycle_generation = sqlc.arg(expected_lifecycle_generation)
  AND sqlc.arg(lifecycle_generation)::bigint = sqlc.arg(expected_lifecycle_generation)::bigint + 1
RETURNING *;

-- name: SaveCredentialHomeCatalogLifecycleRecord :one
INSERT INTO credential_home_catalog_lifecycle (
    catalog_id, home_ref, name_ref, provider, state, reason_code, active_refs,
    generation, updated_at, retention_deadline
)
SELECT
    c.id,
    sqlc.arg(home_ref),
    NULLIF(sqlc.arg(name_ref)::text, ''),
    sqlc.arg(provider),
    sqlc.arg(lifecycle_state),
    NULLIF(sqlc.arg(reason_code)::text, ''),
    sqlc.arg(active_refs),
    sqlc.arg(lifecycle_generation),
    sqlc.arg(updated_at),
    sqlc.arg(retention_deadline)
FROM credential_home_catalog c
WHERE c.id = sqlc.arg(catalog_id)
  AND c.workspace_id = sqlc.arg(workspace_id)
  AND c.state = 'reconciling'
  AND c.lifecycle_generation = sqlc.arg(lifecycle_generation)
ON CONFLICT (catalog_id, home_ref) DO UPDATE
SET name_ref = EXCLUDED.name_ref,
    provider = EXCLUDED.provider,
    state = EXCLUDED.state,
    reason_code = EXCLUDED.reason_code,
    active_refs = EXCLUDED.active_refs,
    generation = EXCLUDED.generation,
    updated_at = EXCLUDED.updated_at,
    retention_deadline = EXCLUDED.retention_deadline
WHERE credential_home_catalog_lifecycle.generation <= EXCLUDED.generation
RETURNING *;

-- name: CreateCredentialHomeCatalogGeneration :one
INSERT INTO credential_home_catalog_generation (
    catalog_id, previous_generation, generation, scan_kind, counters,
    catalog_digest, started_at, published_at
)
VALUES (
    @catalog_id, @previous_generation, @generation, @scan_kind,
    convert_from(@counters::bytea, 'UTF8')::jsonb,
    @catalog_digest, @started_at, @published_at
)
RETURNING *;

-- name: CreateCredentialHomeCatalogEntry :one
INSERT INTO credential_home_catalog_entry (
    generation_id, catalog_id, generation, home_ref, name_ref, provider,
    approved, state, reason_code, active_refs, first_seen_at, last_seen_at,
    last_full_scan_at, health_watermark, missing_watermark, ttl_nanoseconds,
    retention_deadline
)
VALUES (
    @generation_id, @catalog_id, @generation, @home_ref,
    NULLIF(sqlc.arg(name_ref)::text, ''), @provider, @approved, @state,
    NULLIF(sqlc.arg(reason_code)::text, ''), @active_refs, @first_seen_at,
    @last_seen_at, @last_full_scan_at, sqlc.narg(health_watermark),
    sqlc.narg(missing_watermark), @ttl_nanoseconds, @retention_deadline
)
RETURNING *;

-- name: PublishCredentialHomeCatalogGeneration :one
UPDATE credential_home_catalog AS chc
SET generation = @generation,
    state = 'available',
    watermark = @watermark,
    updated_at = now()
WHERE chc.id = @catalog_id
  AND chc.generation = @previous_generation
  AND chc.lifecycle_generation = @generation
  AND EXISTS (
      SELECT 1 FROM credential_home_catalog_generation AS chcg
      WHERE chcg.catalog_id = @catalog_id
        AND chcg.previous_generation = @previous_generation
        AND chcg.generation = @generation
  )
RETURNING *;

-- name: GetPublishedCredentialHomeCatalogGeneration :one
SELECT g.*, c.watermark
FROM credential_home_catalog_generation g
JOIN credential_home_catalog c ON c.id = g.catalog_id
WHERE c.id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND c.generation = @generation
  AND g.generation = @generation;

-- name: ListCredentialHomeCatalogGenerationEntries :many
SELECT e.*
FROM credential_home_catalog_entry e
JOIN credential_home_catalog c ON c.id = e.catalog_id
WHERE e.catalog_id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND e.generation = @generation
ORDER BY e.home_ref;

-- name: ListCredentialHomeCatalogGenerationLifecycle :many
SELECT l.*
FROM credential_home_catalog_lifecycle l
JOIN credential_home_catalog c ON c.id = l.catalog_id
WHERE l.catalog_id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND l.generation = @generation
ORDER BY l.home_ref;

-- name: ListCurrentCredentialHomeCatalogEntries :many
SELECT e.*
FROM credential_home_catalog_entry e
JOIN credential_home_catalog c
  ON c.id = e.catalog_id AND c.generation = e.generation
WHERE c.workspace_id = @workspace_id
  AND c.id = @catalog_id
ORDER BY e.home_ref;

-- name: SelectHealthyApprovedCredentialHomeForUpdate :one
SELECT e.*
FROM credential_home_catalog_entry e
JOIN credential_home_catalog c
  ON c.id = e.catalog_id AND c.generation = e.generation
WHERE c.id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND c.generation = @expected_catalog_generation
  AND c.state = 'available'
  AND e.provider = @provider
  AND e.approved = true
  AND e.state = 'healthy'
  AND e.health_watermark >= @health_fresh_after
  AND e.retention_deadline > now()
  AND NOT EXISTS (
      SELECT 1 FROM runtime_home_assignment a
      WHERE a.home_ref = e.home_ref AND a.state IN ('active', 'draining')
  )
  AND NOT EXISTS (
      SELECT 1 FROM native_rotation_operation o
      WHERE (o.current_home_ref = e.home_ref OR o.target_home_ref = e.home_ref)
        AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
  )
ORDER BY e.home_ref
LIMIT 1
FOR UPDATE OF e SKIP LOCKED;

-- name: GetHealthyApprovedCredentialHomeForUpdate :one
SELECT e.*
FROM credential_home_catalog_entry e
JOIN credential_home_catalog c
  ON c.id = e.catalog_id AND c.generation = e.generation
WHERE c.id = @catalog_id
  AND c.workspace_id = @workspace_id
  AND c.generation = @expected_catalog_generation
  AND c.state = 'available'
  AND e.home_ref = @home_ref
  AND e.provider = @provider
  AND e.approved = true
  AND e.state = 'healthy'
  AND e.health_watermark >= @health_fresh_after
  AND e.retention_deadline > now()
  AND NOT EXISTS (
      SELECT 1 FROM runtime_home_assignment a
      WHERE a.home_ref = e.home_ref AND a.state IN ('active', 'draining')
  )
  AND NOT EXISTS (
      SELECT 1 FROM native_rotation_operation o
      WHERE (o.current_home_ref = e.home_ref OR o.target_home_ref = e.home_ref)
        AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
  )
FOR UPDATE OF e;

-- Runtime bindings and exclusive assignments -------------------------------

-- name: CreateRuntimeBinding :one
INSERT INTO runtime_binding (
    enrollment_id, session_id, workspace_id, runtime_id, agent_id,
    transport_binding, max_concurrent_tasks, created_by
)
VALUES (
    @enrollment_id, @session_id, @workspace_id, @runtime_id, @agent_id,
    @transport_binding, @max_concurrent_tasks, @created_by
)
RETURNING *;

-- name: ListRuntimeBindingsForWorkspace :many
SELECT *
FROM runtime_binding
WHERE workspace_id = @workspace_id
ORDER BY created_at, id;

-- name: GetRuntimeBindingForWorkspace :one
SELECT *
FROM runtime_binding
WHERE id = @id AND workspace_id = @workspace_id;

-- name: LockRuntimeBinding :one
SELECT *
FROM runtime_binding
WHERE id = @id AND workspace_id = @workspace_id
FOR UPDATE;

-- name: FenceRuntimeBindingGeneration :one
UPDATE runtime_binding
SET generation = generation + 1, updated_at = now()
WHERE id = @id
  AND workspace_id = @workspace_id
  AND generation = @expected_binding_generation
  AND state = 'active'
RETURNING *;

-- name: SetRuntimeBindingConcurrencyLimit :one
UPDATE runtime_binding
SET max_concurrent_tasks = @max_concurrent_tasks,
    generation = generation + 1,
    updated_at = now()
WHERE id = @id
  AND workspace_id = @workspace_id
  AND generation = @expected_binding_generation
  AND @max_concurrent_tasks::integer > 0
  AND active_task_count <= @max_concurrent_tasks
RETURNING *;

-- name: CreateRuntimeHomeAssignment :one
INSERT INTO runtime_home_assignment (
    binding_id, workspace_id, catalog_id, catalog_entry_id,
    catalog_generation, home_ref, binding_generation, assigned_by,
    reason_code
)
VALUES (
    @binding_id, @workspace_id, @catalog_id, @catalog_entry_id,
    @catalog_generation, @home_ref, @binding_generation, @assigned_by,
    sqlc.narg(reason_code)
)
RETURNING *;

-- name: GetActiveRuntimeHomeAssignment :one
SELECT *
FROM runtime_home_assignment
WHERE binding_id = @binding_id AND state IN ('active', 'draining');

-- name: GetActiveRuntimeHomeAssignmentForUpdate :one
SELECT *
FROM runtime_home_assignment
WHERE binding_id = @binding_id AND state IN ('active', 'draining')
FOR UPDATE;

-- name: ReserveNativeRuntimeBindingCapacity :one
UPDATE runtime_binding AS b
SET active_task_count = b.active_task_count + 1,
    updated_at = now()
WHERE b.id = @binding_id
  AND b.workspace_id = @workspace_id
  AND b.generation = @expected_binding_generation
  AND b.transport_binding = 'native_credential_home'
  AND b.state = 'active'
  AND b.active_task_count < b.max_concurrent_tasks
  AND EXISTS (
      SELECT 1
      FROM runtime_home_assignment a
      JOIN credential_home_catalog c ON c.id = a.catalog_id
      JOIN credential_home_catalog_entry e
        ON e.id = a.catalog_entry_id
       AND e.catalog_id = a.catalog_id
       AND e.generation = a.catalog_generation
       AND e.home_ref = a.home_ref
      WHERE a.id = @home_assignment_id
        AND a.binding_id = b.id
        AND a.workspace_id = b.workspace_id
        AND a.catalog_generation = @expected_catalog_generation
        AND a.state = 'active'
        AND c.generation = @expected_catalog_generation
        AND c.state = 'available'
        AND e.approved = true
        AND e.state = 'healthy'
        AND e.health_watermark >= @health_fresh_after
        AND e.retention_deadline > now()
        AND NOT EXISTS (
            SELECT 1 FROM native_rotation_operation o
            WHERE (
                o.current_lifetime_id IN (
                    SELECT l.id FROM runtime_home_lifetime l
                    WHERE l.home_assignment_id = a.id
                )
                OR o.target_lifetime_id IN (
                    SELECT l.id FROM runtime_home_lifetime l
                    WHERE l.home_assignment_id = a.id
                )
                OR o.current_home_ref = a.home_ref
                OR o.target_home_ref = a.home_ref
            )
              AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
        )
  )
RETURNING b.*;

-- name: DrainRuntimeHomeAssignment :one
UPDATE runtime_home_assignment
SET state = 'draining', reason_code = sqlc.narg(reason_code)
WHERE id = @id AND binding_id = @binding_id AND state = 'active'
RETURNING *;

-- name: ReleaseRuntimeHomeAssignment :one
UPDATE runtime_home_assignment
SET state = 'released', released_at = now(), reason_code = sqlc.narg(reason_code)
WHERE runtime_home_assignment.id = @id
  AND runtime_home_assignment.binding_id = @binding_id
  AND runtime_home_assignment.state = 'draining'
  AND NOT EXISTS (
      SELECT 1 FROM native_rotation_operation o
      WHERE (
          o.current_home_ref = runtime_home_assignment.home_ref
          OR o.target_home_ref = runtime_home_assignment.home_ref
      )
        AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
  )
  AND NOT EXISTS (
      SELECT 1
      FROM runtime_task_snapshot s
      LEFT JOIN runtime_task_snapshot_release r ON r.task_id = s.task_id
      WHERE s.home_assignment_id = runtime_home_assignment.id
        AND r.task_id IS NULL
  )
RETURNING *;

-- Runtime configuration and task snapshots ---------------------------------

-- name: GetRuntimeConfigurationVersionByRequestID :one
SELECT *
FROM runtime_configuration_version
WHERE binding_id = @binding_id AND request_id = @request_id;

-- name: CreateRuntimeConfigurationVersion :one
INSERT INTO runtime_configuration_version (
    binding_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason, request_id
)
VALUES (
    @binding_id, @version_number, @configuration, @configuration_digest,
    @apply_class, @created_by, @reason, @request_id
)
RETURNING *;

-- name: ListRuntimeConfigurationVersions :many
SELECT *
FROM runtime_configuration_version
WHERE binding_id = @binding_id
ORDER BY version_number DESC;

-- name: ActivateRuntimeConfigurationVersion :one
UPDATE runtime_binding AS rb
SET active_configuration_version_id = @new_version_id,
    effective_configuration_digest = @effective_configuration_digest,
    generation = rb.generation + 1,
    updated_at = now()
WHERE rb.id = @binding_id
  AND rb.workspace_id = @workspace_id
  AND rb.generation = @expected_binding_generation
  AND rb.active_configuration_version_id IS NOT DISTINCT FROM sqlc.narg(expected_active_version_id)::uuid
  AND EXISTS (
      SELECT 1 FROM runtime_configuration_version AS rcv
      WHERE rcv.binding_id = @binding_id AND rcv.id = @new_version_id
  )
RETURNING *;

-- name: RecordRuntimeConfigurationActivation :one
INSERT INTO runtime_configuration_activation (
    binding_id, previous_version_id, new_version_id, binding_generation,
    effective_configuration_digest, capability_digest, apply_class, actor_id,
    request_id, correlation_id, reason
)
VALUES (
    @binding_id, sqlc.narg(previous_version_id), @new_version_id,
    @binding_generation, @effective_configuration_digest, @capability_digest,
    @apply_class, @actor_id, @request_id, sqlc.narg(correlation_id), @reason
)
RETURNING *;

-- name: GetRuntimeConfigurationActivationByRequestID :one
SELECT *
FROM runtime_configuration_activation
WHERE binding_id = @binding_id AND request_id = @request_id;

-- name: RecordRuntimeConfigurationAcknowledgement :one
INSERT INTO runtime_configuration_acknowledgement (
    binding_id, configuration_version_id, binding_generation, daemon_id,
    status, applied_configuration_digest, reason_code
)
VALUES (
    @binding_id, @configuration_version_id, @binding_generation, @daemon_id,
    @status, sqlc.narg(applied_configuration_digest), sqlc.narg(reason_code)
)
RETURNING *;

-- name: CreateRuntimeTaskSnapshot :one
-- The caller first locks the task with LockAgentTaskForRuntimeSnapshot in the
-- same transaction. An exact retry returns the immutable existing snapshot;
-- a duplicate task with different pinned identity returns no row, which the
-- store maps to its duplicate-task conflict and rolls back (including the
-- preceding capacity increment).
WITH inserted AS (
    INSERT INTO runtime_task_snapshot (
        task_id, runtime_session_id, runtime_id, agent_id, workspace_id,
        runtime_standard_version_id, runtime_configuration_version_id,
        effective_configuration_digest, runtime_binding_id, binding_generation,
        transport_binding, home_assignment_id, home_ref, catalog_generation,
        capability_digest
    )
    VALUES (
        @task_id, @runtime_session_id, @runtime_id, @agent_id, @workspace_id,
        @runtime_standard_version_id, @runtime_configuration_version_id,
        @effective_configuration_digest, @runtime_binding_id, @binding_generation,
        @transport_binding, sqlc.narg(home_assignment_id), sqlc.narg(home_ref),
        sqlc.narg(catalog_generation), @capability_digest
    )
    ON CONFLICT (task_id) DO NOTHING
    RETURNING *
)
SELECT * FROM inserted
UNION ALL
SELECT s.*
FROM runtime_task_snapshot s
WHERE s.task_id = @task_id
  AND s.runtime_session_id = @runtime_session_id
  AND s.runtime_id = @runtime_id
  AND s.agent_id = @agent_id
  AND s.workspace_id = @workspace_id
  AND s.runtime_standard_version_id = @runtime_standard_version_id
  AND s.runtime_configuration_version_id = @runtime_configuration_version_id
  AND s.effective_configuration_digest = @effective_configuration_digest
  AND s.runtime_binding_id = @runtime_binding_id
  AND s.binding_generation = @binding_generation
  AND s.transport_binding = @transport_binding
  AND s.home_assignment_id IS NOT DISTINCT FROM sqlc.narg(home_assignment_id)::uuid
  AND s.home_ref IS NOT DISTINCT FROM sqlc.narg(home_ref)::uuid
  AND s.catalog_generation IS NOT DISTINCT FROM sqlc.narg(catalog_generation)::bigint
  AND s.capability_digest = @capability_digest
  AND NOT EXISTS (SELECT 1 FROM inserted)
LIMIT 1;

-- name: GetRuntimeTaskSnapshot :one
SELECT *
FROM runtime_task_snapshot
WHERE task_id = @task_id;

-- name: LockAgentTaskForRuntimeSnapshot :one
SELECT id, agent_id, runtime_id, status
FROM agent_task_queue
WHERE id = @task_id
  AND agent_id = @agent_id
  AND runtime_id = @runtime_id
FOR UPDATE;

-- name: ReleaseRuntimeTaskSnapshotCapacity :one
WITH released AS (
    INSERT INTO runtime_task_snapshot_release (
        task_id, runtime_binding_id, released_by, reason_code
    )
    SELECT s.task_id, s.runtime_binding_id, sqlc.narg(released_by), @reason_code
    FROM runtime_task_snapshot s
    JOIN agent_task_queue t ON t.id = s.task_id
    WHERE s.task_id = @task_id
      AND s.runtime_binding_id = @runtime_binding_id
      AND t.status NOT IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
      AND NOT EXISTS (
          SELECT 1 FROM native_rotation_operation o
          WHERE (
              o.current_home_ref = s.home_ref
              OR o.target_home_ref = s.home_ref
          )
            AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
      )
    ON CONFLICT (task_id) DO NOTHING
    RETURNING runtime_binding_id
)
UPDATE runtime_binding AS b
SET active_task_count = b.active_task_count - 1,
    updated_at = now()
FROM released r
WHERE b.id = r.runtime_binding_id
  AND b.active_task_count > 0
RETURNING b.*;

-- name: GetRuntimeTaskSnapshotRelease :one
SELECT *
FROM runtime_task_snapshot_release
WHERE task_id = @task_id;

-- name: CreateCredentialReadinessProbeRequest :one
INSERT INTO runtime_credential_readiness_probe_request (
    probe_request_id, request_digest, task_id, home_epoch, workspace_id,
    agent_id, runtime_id, runtime_session_id, daemon_id, daemon_boot_id,
    provider, transport_binding, runtime_binding_id, binding_generation,
    home_assignment_id, catalog_id, catalog_entry_id, catalog_generation,
    home_ref, lifetime_id, acquisition_request_id
)
SELECT
    @probe_request_id, @request_digest, e.task_id, e.home_epoch,
    e.workspace_id, e.agent_id, e.runtime_id, e.runtime_session_id,
    e.daemon_id, e.daemon_boot_id, e.provider, e.transport_binding,
    e.runtime_binding_id, e.binding_generation, e.home_assignment_id,
    e.catalog_id, e.catalog_entry_id, e.catalog_generation, e.home_ref,
    e.lifetime_id, e.acquisition_request_id
FROM runtime_task_home_epoch e
WHERE e.task_id = @task_id
  AND e.home_epoch = @home_epoch
  AND e.daemon_boot_id = @daemon_boot_id
  AND e.runtime_session_id = @runtime_session_id
  AND e.runtime_binding_id = @runtime_binding_id
  AND e.binding_generation = @binding_generation
  AND e.home_assignment_id = @home_assignment_id
  AND e.catalog_generation = @catalog_generation
ON CONFLICT (probe_request_id) DO NOTHING
RETURNING *;

-- name: AcceptOrReplayCredentialReadinessResult :one
WITH request_identity AS MATERIALIZED (
    SELECT r.*
    FROM runtime_credential_readiness_probe_request r
    WHERE r.probe_request_id = @probe_request_id
),
home_lock AS MATERIALIZED (
    SELECT pg_advisory_xact_lock(
        multica_credential_home_advisory_key(r.home_ref)
    )
    FROM request_identity r
),
binding_lock AS MATERIALIZED (
    SELECT b.id
    FROM runtime_binding b, request_identity r, home_lock
    WHERE b.id = r.runtime_binding_id
    FOR UPDATE
),
assignment_lock AS MATERIALIZED (
    SELECT a.id
    FROM runtime_home_assignment a, request_identity r, binding_lock
    WHERE a.id = r.home_assignment_id
    FOR UPDATE
),
catalog_lock AS MATERIALIZED (
    SELECT c.id
    FROM credential_home_catalog c, request_identity r, assignment_lock
    WHERE c.id = r.catalog_id
    FOR UPDATE
),
entry_lock AS MATERIALIZED (
    SELECT e.id
    FROM credential_home_catalog_entry e, request_identity r, catalog_lock
    WHERE e.id = r.catalog_entry_id
    FOR UPDATE
),
locked_request AS MATERIALIZED (
    SELECT r.*
    FROM runtime_credential_readiness_probe_request r, entry_lock
    WHERE r.probe_request_id = @probe_request_id
    FOR UPDATE
),
inserted_attestation AS (
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
        r.provider, r.daemon_id, r.daemon_boot_id, @state, @reason_code,
        @probe_observed_at, transaction_timestamp() + interval '5 minutes'
    FROM locked_request r
    WHERE r.request_digest = @request_digest
      AND @probe_observed_at >= r.issued_at
      AND @probe_observed_at <= transaction_timestamp()
      AND transaction_timestamp() <= r.request_expires_at
    ON CONFLICT (probe_request_id) DO NOTHING
    RETURNING *
),
inserted_result AS (
    INSERT INTO runtime_credential_readiness_probe_result (
        probe_request_id, attestation_id, result_digest, state, reason_code,
        probe_observed_at
    )
    SELECT
        a.probe_request_id, a.id, @result_digest, a.state, a.reason_code,
        a.observed_at
    FROM inserted_attestation a
    RETURNING *
)
SELECT * FROM inserted_result
UNION ALL
SELECT existing.*
FROM runtime_credential_readiness_probe_result existing
JOIN runtime_credential_readiness_probe_request r
  ON r.probe_request_id = existing.probe_request_id
WHERE existing.probe_request_id = @probe_request_id
  AND r.request_digest = @request_digest
  AND existing.result_digest = @result_digest
  AND existing.state = @state
  AND existing.reason_code = @reason_code
  AND existing.probe_observed_at = @probe_observed_at
  AND NOT EXISTS (SELECT 1 FROM inserted_result)
LIMIT 1;

-- name: GetCurrentCredentialReadinessAttestation :one
SELECT a.*
FROM runtime_credential_readiness_attestation a
JOIN runtime_credential_readiness_probe_result r
  ON r.attestation_id = a.id
JOIN runtime_credential_readiness_probe_request q
  ON q.probe_request_id = r.probe_request_id
JOIN runtime_task_home_epoch e
  ON e.task_id = q.task_id
 AND e.home_epoch = q.home_epoch
JOIN runtime_binding b ON b.id = e.runtime_binding_id
JOIN runtime_home_assignment h ON h.id = e.home_assignment_id
JOIN credential_home_catalog c ON c.id = e.catalog_id
JOIN credential_home_catalog_entry ce ON ce.id = e.catalog_entry_id
JOIN runtime_home_lifetime l ON l.id = e.lifetime_id
WHERE q.task_id = @task_id
  AND q.home_epoch = @home_epoch
  AND q.runtime_session_id = @runtime_session_id
  AND q.daemon_boot_id = @daemon_boot_id
  AND q.runtime_binding_id = @runtime_binding_id
  AND q.binding_generation = @binding_generation
  AND q.home_assignment_id = @home_assignment_id
  AND q.catalog_generation = @catalog_generation
  AND b.generation = q.binding_generation
  AND b.state = 'active'
  AND b.transport_binding = 'native_credential_home'
  AND h.state = 'active'
  AND c.generation = q.catalog_generation
  AND c.state = 'available'
  AND ce.generation = q.catalog_generation
  AND ce.state = 'healthy'
  AND ce.approved
  AND l.state = 'process_started'
  AND a.expires_at > transaction_timestamp()
  AND NOT EXISTS (
      SELECT 1 FROM native_rotation_operation o
      WHERE o.task_id = q.task_id
        AND (o.current_home_ref = q.home_ref OR o.target_home_ref = q.home_ref)
        AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
  )
ORDER BY r.accepted_at DESC, a.created_at DESC, a.id DESC
LIMIT 1;
