-- Runtime Standards ---------------------------------------------------------

-- name: CreateRuntimeStandard :one
INSERT INTO runtime_standard (owner_id, name, description)
VALUES (@owner_id, @name, sqlc.narg(description))
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

-- name: CreateRuntimeStandardVersion :one
INSERT INTO runtime_standard_version (
    standard_id, version_number, configuration, configuration_digest,
    created_by, reason
)
VALUES (
    @standard_id, @version_number, @configuration, @configuration_digest,
    @created_by, @reason
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

-- name: BeginCredentialHomeCatalogReconciliation :one
UPDATE credential_home_catalog
SET state = 'reconciling', updated_at = now()
WHERE id = @id
  AND workspace_id = @workspace_id
  AND generation = @expected_generation
  AND state <> 'reconciling'
RETURNING *;

-- name: CreateCredentialHomeCatalogGeneration :one
INSERT INTO credential_home_catalog_generation (
    catalog_id, previous_generation, generation, scan_kind, counters,
    catalog_digest, started_at
)
VALUES (
    @catalog_id, @previous_generation, @generation, @scan_kind, @counters,
    @catalog_digest, @started_at
)
RETURNING *;

-- name: CreateCredentialHomeCatalogEntry :one
INSERT INTO credential_home_catalog_entry (
    generation_id, catalog_id, generation, home_ref, provider, approved, state,
    reason_code, first_seen_at, last_seen_at, last_full_scan_at,
    health_watermark, missing_watermark, ttl, retention_deadline
)
VALUES (
    @generation_id, @catalog_id, @generation, @home_ref, @provider, @approved, @state,
    sqlc.narg(reason_code), @first_seen_at, @last_seen_at, @last_full_scan_at,
    sqlc.narg(health_watermark), sqlc.narg(missing_watermark), @ttl,
    @retention_deadline
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
  AND EXISTS (
      SELECT 1 FROM credential_home_catalog_generation AS chcg
      WHERE chcg.catalog_id = @catalog_id
        AND chcg.previous_generation = @previous_generation
        AND chcg.generation = @generation
  )
RETURNING *;

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
WHERE id = @id AND binding_id = @binding_id AND state = 'draining'
  AND NOT EXISTS (
      SELECT 1
      FROM runtime_task_snapshot s
      LEFT JOIN runtime_task_snapshot_release r ON r.task_id = s.task_id
      WHERE s.home_assignment_id = runtime_home_assignment.id
        AND r.task_id IS NULL
  )
RETURNING *;

-- Runtime configuration and task snapshots ---------------------------------

-- name: CreateRuntimeConfigurationVersion :one
INSERT INTO runtime_configuration_version (
    binding_id, version_number, configuration, configuration_digest,
    apply_class, created_by, reason
)
VALUES (
    @binding_id, @version_number, @configuration, @configuration_digest,
    @apply_class, @created_by, @reason
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
