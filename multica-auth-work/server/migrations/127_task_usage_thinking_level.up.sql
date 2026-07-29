-- ORQ-13 phase 1: persist the reasoning tier that produced each usage row.
--
-- NULLABLE on purpose, and additive only:
--   * historical rows keep NULL — they predate the field and we must not
--     invent a tier for them. NULL means "not declared", never "standard".
--   * no NOT NULL and no DEFAULT: a DEFAULT '' would make every legacy row
--     claim the base tier, which is exactly the false-success this column
--     exists to eliminate.
--   * the UNIQUE (task_id, provider, model) constraint from migration 032 is
--     deliberately left untouched. A task runs under one reasoning tier at a
--     time, so the tier is an attribute of the row, not part of its identity.
--     Adding it to the key would let one task accumulate two rows for the
--     same model and silently double-count tokens.
--   * no index: nothing queries by thinking_level in this phase. Rollups
--     (073/084/101/102) are intentionally NOT touched — cost stays derived
--     at read time.
ALTER TABLE task_usage
    ADD COLUMN IF NOT EXISTS thinking_level TEXT;

COMMENT ON COLUMN task_usage.thinking_level IS
    'Reasoning tier declared by the agent config for this task (e.g. high, low, thinking). NULL = not declared by the reporting daemon. Never inferred from the model name.';
