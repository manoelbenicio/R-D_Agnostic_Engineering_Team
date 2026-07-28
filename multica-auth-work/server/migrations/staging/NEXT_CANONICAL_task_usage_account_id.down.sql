-- Reverse of NEXT_CANONICAL_task_usage_account_id.up.sql.
--
-- Dropping the column discards only the attribution this migration introduced.
-- Every pre-existing column, the UNIQUE (task_id, provider, model) constraint
-- and all token counts are untouched, so rollups survive the rollback intact.
-- IF EXISTS keeps the down step idempotent.
DROP INDEX IF EXISTS idx_task_usage_account;

ALTER TABLE task_usage
    DROP COLUMN IF EXISTS account_id;

DROP INDEX IF EXISTS idx_agent_task_queue_credential_account;

ALTER TABLE agent_task_queue
    DROP COLUMN IF EXISTS credential_account_id;

-- Down symmetry for the uniqueness constraint. Dropping it restores the prior
-- freedom to assign one account to several agents; no row is touched, so the
-- rollback cannot lose an assignment.
DROP INDEX IF EXISTS uq_assignments_account;
