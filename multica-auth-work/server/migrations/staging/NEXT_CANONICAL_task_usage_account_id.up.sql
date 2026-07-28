-- ORQ-12: snapshot the provider account that produced each usage row.
--
-- VERSION NUMBER IS DELIBERATELY ABSENT. The file lives under
-- migrations/staging/ with the placeholder name
-- NEXT_CANONICAL_task_usage_account_id, and it acquires a number only after the
-- central registrar scans for the next free version above 127
-- (127_task_usage_thinking_level is the highest materialized migration in this
-- lineage). internal/migrations.Files globs "<dir>/*.up.sql" NON-RECURSIVELY,
-- so nothing here is ever applied by `migrate up` while it stays staged.
--
-- NULLABLE on purpose, and additive only:
--   * legacy rows keep NULL: they predate the column and no account can be
--     attributed to them after the fact. NULL means "not attributable", never
--     "the default account".
--   * a row whose agent has no assignment also keeps NULL. Inventing an account
--     would misattribute spend, which is the exact failure this column exists
--     to prevent.
--   * ON DELETE SET NULL: deleting an account must not delete usage history nor
--     cascade into token counts. The spend happened; only its attribution is
--     lost.
--   * the UNIQUE (task_id, provider, model) key from migration 032 is left
--     untouched. A task runs under one account at a time, so the account is an
--     attribute of the row, not part of its identity. Adding it to the key
--     would let one task hold two rows for the same model and double-count.
--   * rollups (073/084/101/102) are intentionally NOT touched: this phase only
--     records the dimension.
ALTER TABLE task_usage
    ADD COLUMN IF NOT EXISTS account_id UUID REFERENCES accounts(account_id) ON DELETE SET NULL;

COMMENT ON COLUMN task_usage.account_id IS
    'Provider account that produced this usage, snapshotted at report time by resolving agent_task_queue.agent_id through assignments. NULL = not attributable (legacy row, or the agent had no assignment). Never inferred.';

-- Partial index: only attributable rows are ever grouped by account, and the
-- NULL-heavy legacy tail would otherwise bloat the index for no reader.
CREATE INDEX IF NOT EXISTS idx_task_usage_account
    ON task_usage(account_id)
    WHERE account_id IS NOT NULL;
