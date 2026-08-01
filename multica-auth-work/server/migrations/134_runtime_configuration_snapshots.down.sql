DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM runtime_task_snapshot rts
        JOIN agent_task_queue atq ON atq.id = rts.task_id
        WHERE atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'cannot roll back runtime configuration snapshots while active task references remain';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS runtime_task_snapshot_immutable ON runtime_task_snapshot;
DROP TRIGGER IF EXISTS runtime_task_snapshot_release_immutable ON runtime_task_snapshot_release;
DROP TRIGGER IF EXISTS runtime_task_snapshot_requires_reservation ON runtime_task_snapshot;
DROP TRIGGER IF EXISTS runtime_task_snapshot_release_terminal_only ON runtime_task_snapshot_release;
DROP TRIGGER IF EXISTS runtime_configuration_acknowledgement_immutable ON runtime_configuration_acknowledgement;
DROP TRIGGER IF EXISTS runtime_configuration_activation_immutable ON runtime_configuration_activation;
DROP TRIGGER IF EXISTS runtime_configuration_version_immutable ON runtime_configuration_version;
DROP TABLE runtime_task_snapshot_release;
DROP TABLE runtime_task_snapshot;
DROP FUNCTION enforce_runtime_task_snapshot_terminal_release();
DROP FUNCTION enforce_runtime_task_snapshot_reservation();
ALTER TABLE agent_task_queue
    DROP CONSTRAINT agent_task_queue_runtime_snapshot_projection_key;
DROP TABLE runtime_configuration_acknowledgement;
DROP TABLE runtime_configuration_activation;
ALTER TABLE runtime_binding
    DROP CONSTRAINT runtime_binding_active_configuration_digest_check,
    DROP CONSTRAINT runtime_binding_active_configuration_fkey,
    DROP COLUMN effective_configuration_digest,
    DROP COLUMN active_configuration_version_id;
DROP TABLE runtime_configuration_version;
