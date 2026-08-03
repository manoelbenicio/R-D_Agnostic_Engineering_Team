-- ORQ-121 native rotation core. Callers execute each phase in one transaction
-- and follow the documented global class order. External/local work occurs
-- only between committed phases.

-- name: AcquireNativeRotationHomeLocks :many
SELECT home_ref,
       pg_advisory_xact_lock(
           multica_credential_home_advisory_key(home_ref)
       )::text AS lock_result
FROM (
    SELECT DISTINCT unnest(sqlc.arg(home_refs)::uuid[]) AS home_ref
) homes
ORDER BY home_ref::text;

-- name: LockNativeRotationTask :one
SELECT id, agent_id, runtime_id, status
FROM agent_task_queue
WHERE id = @task_id
  AND agent_id = @agent_id
  AND runtime_id = @runtime_id
FOR UPDATE;

-- name: LockNativeRotationBinding :one
SELECT *
FROM runtime_binding
WHERE id = @runtime_binding_id
  AND workspace_id = @workspace_id
  AND session_id = @runtime_session_id
  AND runtime_id = @runtime_id
  AND agent_id = @agent_id
  AND generation = @binding_generation
  AND transport_binding = 'native_credential_home'
  AND state = 'active'
FOR UPDATE;

-- name: LockNativeRotationAssignments :many
SELECT *
FROM runtime_home_assignment
WHERE id = ANY(sqlc.arg(assignment_ids)::uuid[])
ORDER BY id
FOR UPDATE;

-- name: LockNativeRotationCatalogs :many
SELECT *
FROM credential_home_catalog
WHERE id = ANY(sqlc.arg(catalog_ids)::uuid[])
ORDER BY id
FOR UPDATE;

-- name: LockNativeRotationEntries :many
SELECT *
FROM credential_home_catalog_entry
WHERE id = ANY(sqlc.arg(entry_ids)::uuid[])
ORDER BY id
FOR UPDATE;

-- name: LockNativeRotationLifecycle :many
SELECT *
FROM credential_home_catalog_lifecycle
WHERE (catalog_id, home_ref) IN (
    SELECT unnest(sqlc.arg(catalog_ids)::uuid[]),
           unnest(sqlc.arg(home_refs)::uuid[])
)
ORDER BY catalog_id, home_ref
FOR UPDATE;

-- name: LockNativeRotationEpochs :many
SELECT *
FROM runtime_task_home_epoch
WHERE task_id = @task_id
  AND home_epoch = ANY(sqlc.arg(home_epochs)::bigint[])
ORDER BY home_epoch
FOR UPDATE;

-- name: LockNativeRotationLifetimes :many
SELECT *
FROM runtime_home_lifetime
WHERE id = ANY(sqlc.arg(lifetime_ids)::uuid[])
ORDER BY id
FOR UPDATE;

-- name: GetNativeTaskHomeEpoch :one
SELECT *
FROM runtime_task_home_epoch
WHERE task_id = @task_id AND home_epoch = @home_epoch;

-- name: CreateNativeTargetAssignment :one
INSERT INTO runtime_home_assignment (
    id, binding_id, workspace_id, catalog_id, catalog_entry_id,
    catalog_generation, home_ref, binding_generation, state,
    assigned_by, reason_code
) VALUES (
    @id, @binding_id, @workspace_id, @catalog_id, @catalog_entry_id,
    @catalog_generation, @home_ref, @binding_generation, 'preparing',
    @assigned_by, @reason_code
)
RETURNING *;

-- name: CreateNativeTaskHomeEpoch :one
INSERT INTO runtime_task_home_epoch (
    task_id, home_epoch, workspace_id, agent_id, runtime_id,
    runtime_session_id, daemon_id, daemon_boot_id, provider,
    transport_binding, runtime_binding_id, binding_generation,
    home_assignment_id, catalog_id, catalog_entry_id, catalog_generation,
    home_ref, lifetime_id, acquisition_request_id
) VALUES (
    @task_id, @home_epoch, @workspace_id, @agent_id, @runtime_id,
    @runtime_session_id, @daemon_id, @daemon_boot_id, @provider,
    'native_credential_home', @runtime_binding_id, @binding_generation,
    @home_assignment_id, @catalog_id, @catalog_entry_id, @catalog_generation,
    @home_ref, @lifetime_id, @acquisition_request_id
)
RETURNING *;

-- name: CreateNativeHomeLifetime :one
INSERT INTO runtime_home_lifetime (
    id, acquisition_request_id, task_id, home_epoch, agent_id, runtime_id,
    runtime_session_id, runtime_binding_id, binding_generation,
    home_assignment_id, workspace_id, daemon_id, catalog_id,
    catalog_generation, home_ref, daemon_boot_id, provider,
    transport_binding, state
) VALUES (
    @id, @acquisition_request_id, @task_id, @home_epoch, @agent_id, @runtime_id,
    @runtime_session_id, @runtime_binding_id, @binding_generation,
    @home_assignment_id, @workspace_id, @daemon_id, @catalog_id,
    @catalog_generation, @home_ref, @daemon_boot_id, @provider,
    'native_credential_home', 'pending_local'
)
RETURNING *;

-- name: RecordNativeHomeLifetimeEvent :exec
INSERT INTO runtime_home_lifetime_event (
    lifetime_id, state_version, state, transition_request_id, reason_code
) VALUES (
    @lifetime_id, @state_version, @state, @transition_request_id, @reason_code
);

-- name: CreateNativeRotationOperation :one
INSERT INTO native_rotation_operation (
    id, operation_request_id, task_id, runtime_binding_id,
    current_home_epoch, current_home_ref, current_assignment_id,
    current_lifetime_id, target_home_epoch, target_home_ref,
    target_assignment_id, target_lifetime_id, state, reason_code
) VALUES (
    @id, @operation_request_id, @task_id, @runtime_binding_id,
    @current_home_epoch, @current_home_ref, @current_assignment_id,
    @current_lifetime_id, @target_home_epoch, @target_home_ref,
    @target_assignment_id, @target_lifetime_id, 'candidate_reserved',
    @reason_code
)
RETURNING *;

-- name: GetNativeRotationOperationByRequest :one
SELECT *
FROM native_rotation_operation
WHERE operation_request_id = @operation_request_id;

-- name: LockNativeRotationOperation :one
SELECT *
FROM native_rotation_operation
WHERE id = @id
FOR UPDATE;

-- name: GetNativeRotationOperation :one
SELECT *
FROM native_rotation_operation
WHERE id = @id;

-- name: RecordNativeRotationOperationEvent :exec
INSERT INTO native_rotation_operation_event (
    operation_id, state_version, state, transition_request_id, reason_code
) VALUES (
    @operation_id, @state_version, @state, @transition_request_id, @reason_code
);

-- name: AdvanceNativeRotationOperation :one
UPDATE native_rotation_operation
SET state = @next_state,
    state_version = state_version + 1,
    reason_code = @reason_code,
    next_attempt_at = sqlc.narg(next_attempt_at),
    updated_at = now()
WHERE id = @id
  AND state = @expected_state
  AND state_version = @expected_state_version
RETURNING *;

-- name: AdvanceNativeHomeLifetime :one
UPDATE runtime_home_lifetime
SET state = @next_state,
    state_version = state_version + 1,
    process_started_at = CASE
        WHEN @next_state::text = 'process_started'
        THEN COALESCE(process_started_at, now())
        ELSE process_started_at
    END,
    process_identity_digest = COALESCE(
        sqlc.narg(process_identity_digest), process_identity_digest
    ),
    released_at = CASE
        WHEN @next_state::text = 'released' THEN now()
        ELSE released_at
    END
WHERE id = @id
  AND state = @expected_state
  AND state_version = @expected_state_version
RETURNING *;

-- name: SwapNativeRotationAssignments :execrows
UPDATE runtime_home_assignment
SET state = CASE
        WHEN id = @current_assignment_id THEN 'draining'
        WHEN id = @target_assignment_id THEN 'active'
    END,
    reason_code = @reason_code
WHERE (id = @current_assignment_id AND state = 'active')
   OR (id = @target_assignment_id AND state = 'preparing');

-- name: ReleaseNativeRotationAssignment :one
UPDATE runtime_home_assignment
SET state = 'released', released_at = now(), reason_code = @reason_code
WHERE id = @id
  AND state IN ('preparing', 'draining')
RETURNING *;

-- name: CreateNativeRetirementAttempt :one
INSERT INTO native_rotation_retirement_attempt (
    id, retirement_request_id, operation_id, task_id, home_epoch,
    lifetime_id, home_ref, daemon_id, daemon_boot_id, runtime_session_id,
    runtime_binding_id, binding_generation, scheduler_owner, attempt_number,
    state, lease_expires_at, next_attempt_at
) VALUES (
    @id, @retirement_request_id, @operation_id, @task_id, @home_epoch,
    @lifetime_id, @home_ref, @daemon_id, @daemon_boot_id, @runtime_session_id,
    @runtime_binding_id, @binding_generation, @scheduler_owner,
    @attempt_number, 'scheduled', @lease_expires_at, @next_attempt_at
)
RETURNING *;

-- name: LockNativeRetirementAttempt :one
SELECT *
FROM native_rotation_retirement_attempt
WHERE id = @id
FOR UPDATE;

-- name: GetNativeRetirementAttempt :one
SELECT *
FROM native_rotation_retirement_attempt
WHERE id = @id;

-- name: CountNativeRetirementAttempts :one
SELECT count(*)::smallint
FROM native_rotation_retirement_attempt
WHERE operation_id = @operation_id;

-- name: AdvanceNativeRetirementAttempt :one
UPDATE native_rotation_retirement_attempt
SET state = @next_state,
    state_version = state_version + 1,
    scheduler_owner = @scheduler_owner,
    lease_expires_at = sqlc.narg(lease_expires_at),
    channel_binding_digest = COALESCE(
        sqlc.narg(channel_binding_digest), channel_binding_digest
    ),
    request_body_digest = COALESCE(
        sqlc.narg(request_body_digest), request_body_digest
    ),
    result_code = sqlc.narg(result_code),
    next_attempt_at = sqlc.narg(next_attempt_at),
    updated_at = now()
WHERE id = @id
  AND state = @expected_state
  AND state_version = @expected_state_version
RETURNING *;

-- name: RecordNativeRetirementAttemptEvent :exec
INSERT INTO native_rotation_retirement_attempt_event (
    attempt_id, state_version, state, transition_request_id, reason_code
) VALUES (
    @attempt_id, @state_version, @state, @transition_request_id, @reason_code
);

-- name: ListDueNativeRetirementOperations :many
SELECT *
FROM native_rotation_operation
WHERE state IN (
        'committed_retirement_pending',
        'aborted_candidate_retirement_pending'
    )
  AND (next_attempt_at IS NULL OR next_attempt_at <= @now)
ORDER BY next_attempt_at NULLS FIRST, id
LIMIT @batch_size;

-- name: ListDueNativeRetirementAttempts :many
SELECT *
FROM native_rotation_retirement_attempt
WHERE state IN ('scheduled', 'retryable_failure')
  AND (next_attempt_at IS NULL OR next_attempt_at <= @now)
ORDER BY next_attempt_at NULLS FIRST, id
LIMIT @batch_size;

-- name: NativeRotationFenceExists :one
SELECT EXISTS (
    SELECT 1
    FROM native_rotation_operation o
    WHERE (
        o.current_lifetime_id = sqlc.narg(lifetime_id)
        OR o.target_lifetime_id = sqlc.narg(lifetime_id)
        OR o.current_home_ref = sqlc.narg(home_ref)
        OR o.target_home_ref = sqlc.narg(home_ref)
    )
      AND o.state NOT IN ('committed_retired', 'aborted_candidate_retired')
) AS fenced;
