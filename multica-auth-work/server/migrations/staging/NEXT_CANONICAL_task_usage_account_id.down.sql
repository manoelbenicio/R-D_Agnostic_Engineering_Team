-- Reverse of NEXT_CANONICAL_task_usage_account_id.up.sql.
--
-- Dropping the column discards only the attribution this migration introduced.
-- Every pre-existing column, the UNIQUE (task_id, provider, model) constraint
-- and all token counts are untouched, so rollups survive the rollback intact.
-- IF EXISTS keeps the down step idempotent.
DROP INDEX IF EXISTS idx_task_usage_account;

ALTER TABLE task_usage
    DROP COLUMN IF EXISTS account_id;
