-- ORQ-13 phase 2: freeze the authoritative quote applied to each usage row.
-- NULL preserves historical and deliberately-unpriced rows without inventing
-- a price version. The pair is constrained to be both present or both absent.
ALTER TABLE task_usage
    ADD COLUMN IF NOT EXISTS price_version TEXT,
    ADD COLUMN IF NOT EXISTS computed_cost_usd DOUBLE PRECISION;

DO $$
BEGIN
    ALTER TABLE task_usage
        ADD CONSTRAINT task_usage_price_snapshot_consistent
        CHECK (
            (price_version IS NULL AND computed_cost_usd IS NULL)
            OR
            (price_version IS NOT NULL AND computed_cost_usd IS NOT NULL AND computed_cost_usd >= 0)
        );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

COMMENT ON COLUMN task_usage.price_version IS
    'Immutable pricing catalog version resolved from model, recorded thinking_level and task effective time.';
COMMENT ON COLUMN task_usage.computed_cost_usd IS
    'Cost computed from this row token counters and its authoritative price version. NULL means unpriced, never zero-price fallback.';
