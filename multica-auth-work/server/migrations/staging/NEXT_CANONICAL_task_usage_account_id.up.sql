-- ORQ-12: snapshot the provider account that produced each usage row.
--
-- Expected version once the registrar assigns one: 128 (127 is the highest
-- materialized migration in this lineage). That number is an EXPECTATION
-- recorded in prose, NOT a materialization: the filename still carries the
-- NEXT_CANONICAL_ placeholder, and only the registrar turns it into a number.
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

-- ORQ-12 review fix (2): the contract is the PRODUCING account, not "whichever
-- assignment existed when the first usage report arrived".
--
-- The account is therefore frozen on the TASK at claim/dispatch time, and the
-- usage row only copies it. Resolving assignments at report time was wrong: an
-- agent that rotates between dispatch and the report would have its spend filed
-- under the new account, and a re-report after a rotation could disagree with
-- the first one.
--
-- Nullable for the same reason account_id is: tasks claimed before this column
-- existed, and agents with no approved assignment, carry NULL. NULL means "the
-- producing account is unknown", never "the default account".
--
-- ON DELETE SET NULL: deleting an account must never delete queue history.
ALTER TABLE agent_task_queue
    ADD COLUMN IF NOT EXISTS credential_account_id UUID REFERENCES accounts(account_id) ON DELETE SET NULL;

COMMENT ON COLUMN agent_task_queue.credential_account_id IS
    'Provider account frozen on this task at claim/dispatch, resolved server-side from the approved assignment. Immutable once set. NULL = claimed before this column existed, or the agent had no approved assignment. Never supplied by the daemon.';

-- Only attributable tasks are ever grouped or filtered by account.
CREATE INDEX IF NOT EXISTS idx_agent_task_queue_credential_account
    ON agent_task_queue(credential_account_id)
    WHERE credential_account_id IS NOT NULL;

-- ORQ-12 review fix: one account, one agent.
--
-- The claim statement resolves the producing account by joining assignments to
-- approved_accounts. assignments.agent_id is already the primary key, so an
-- agent holds at most one assignment; what was NOT constrained is the other
-- direction. Two agents sharing one account makes per-account spend
-- unattributable to a worker and makes any future account->agent lookup
-- ambiguous, so the reverse direction is constrained here too.
--
-- PREFLIGHT, not a silent failure: creating the index on data that already
-- violates it would abort the migration with a bare "could not create unique
-- index" and no indication of which account is duplicated. The DO block fails
-- first, naming the offenders, so the operator can resolve them before retrying.
DO $$
DECLARE
    dupes TEXT;
BEGIN
    SELECT string_agg(account_id::text, ', ' ORDER BY account_id)
      INTO dupes
      FROM (
          SELECT account_id
          FROM assignments
          GROUP BY account_id
          HAVING count(*) > 1
      ) d;
    IF dupes IS NOT NULL THEN
        RAISE EXCEPTION
            'ORQ-12 preflight: accounts assigned to more than one agent: %. Resolve the duplicates before applying this migration.',
            dupes;
    END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_assignments_account
    ON assignments(account_id);
