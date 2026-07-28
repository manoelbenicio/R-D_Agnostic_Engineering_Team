-- Reverse of 127_task_usage_thinking_level.up.sql.
--
-- Dropping the column discards only data that this migration introduced;
-- every pre-existing column and the UNIQUE (task_id, provider, model)
-- constraint are untouched, so token counts and rollups survive the
-- rollback intact. IF EXISTS keeps the down step idempotent.
ALTER TABLE task_usage
    DROP COLUMN IF EXISTS thinking_level;
