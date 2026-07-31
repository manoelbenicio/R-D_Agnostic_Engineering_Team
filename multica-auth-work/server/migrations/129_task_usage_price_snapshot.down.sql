ALTER TABLE task_usage
    DROP CONSTRAINT IF EXISTS task_usage_price_snapshot_consistent,
    DROP COLUMN IF EXISTS computed_cost_usd,
    DROP COLUMN IF EXISTS price_version;
