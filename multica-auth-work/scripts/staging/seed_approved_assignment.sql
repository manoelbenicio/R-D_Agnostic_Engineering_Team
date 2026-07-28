-- Idempotently seed one metadata-only approved account assignment.
--
-- Required psql variables:
--   workspace_id, agent_id, account_id, vendor, priority,
--   home_dir, config_dir, worktype_scope
--
-- This script never selects, inserts, updates, or deletes credentials.
-- It emits only fixed labels, IDs supplied by the operator, and row counts.

\set ON_ERROR_STOP on

\if :{?workspace_id}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_WORKSPACE_ID_REQUIRED'; END $$;
\endif
\if :{?agent_id}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_AGENT_ID_REQUIRED'; END $$;
\endif
\if :{?account_id}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_ACCOUNT_ID_REQUIRED'; END $$;
\endif
\if :{?vendor}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_VENDOR_REQUIRED'; END $$;
\endif
\if :{?priority}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_PRIORITY_REQUIRED'; END $$;
\endif
\if :{?home_dir}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_HOME_DIR_REQUIRED'; END $$;
\endif
\if :{?config_dir}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_CONFIG_DIR_REQUIRED'; END $$;
\endif
\if :{?worktype_scope}
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_WORKTYPE_SCOPE_REQUIRED'; END $$;
\endif

BEGIN;

SELECT pg_advisory_xact_lock(hashtextextended(:'account_id', 0));

SELECT CASE lower(btrim(:'vendor'))
           WHEN 'agy' THEN 'antigravity'
           WHEN 'antigravity' THEN 'antigravity'
           WHEN 'codex' THEN 'codex'
           WHEN 'kiro' THEN 'kiro'
           ELSE ''
       END AS canonical_vendor,
       lower(btrim(:'vendor')) IN ('agy', 'antigravity', 'codex', 'kiro')
         AS vendor_supported
\gset

\if :vendor_supported
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_VENDOR_UNSUPPORTED'; END $$;
\endif

SELECT EXISTS (
    SELECT 1 FROM workspace WHERE id = :'workspace_id'::uuid
) AS workspace_exists,
EXISTS (
    SELECT 1
      FROM agent
     WHERE id = :'agent_id'::uuid
       AND workspace_id = :'workspace_id'::uuid
) AS agent_exists,
EXISTS (
    SELECT 1
     FROM accounts
     WHERE account_id = :'account_id'::uuid
       AND (tenant_id <> :'workspace_id'::uuid OR vendor <> :'canonical_vendor')
) AS account_conflict,
EXISTS (
    SELECT 1
      FROM assignments
     WHERE agent_id = :'agent_id'::uuid
       AND account_id <> :'account_id'::uuid
) AS assignment_conflict,
EXISTS (
    SELECT 1
      FROM assignments
     WHERE account_id = :'account_id'::uuid
       AND agent_id <> :'agent_id'::uuid
) AS account_assignment_conflict,
EXISTS (
    SELECT 1
      FROM approved_accounts
     WHERE tenant_id = :'workspace_id'::uuid
       AND account_id = :'account_id'::uuid
       AND allowed = false
) AS approval_revoked,
(:'worktype_scope' = 'GENERAL') AS scope_supported
\gset

\if :workspace_exists
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_WORKSPACE_NOT_FOUND'; END $$;
\endif
\if :agent_exists
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_AGENT_WORKSPACE_MISMATCH'; END $$;
\endif
\if :account_conflict
  DO $$ BEGIN RAISE EXCEPTION 'E_ACCOUNT_ID_CONFLICT'; END $$;
\endif
\if :assignment_conflict
  DO $$ BEGIN RAISE EXCEPTION 'E_AGENT_ALREADY_ASSIGNED'; END $$;
\endif
\if :account_assignment_conflict
  DO $$ BEGIN RAISE EXCEPTION 'E_ACCOUNT_ALREADY_ASSIGNED'; END $$;
\endif
\if :approval_revoked
  DO $$ BEGIN RAISE EXCEPTION 'E_ACCOUNT_REVOKED'; END $$;
\endif
\if :scope_supported
\else
  DO $$ BEGIN RAISE EXCEPTION 'E_WORKTYPE_SCOPE_UNSUPPORTED'; END $$;
\endif

INSERT INTO accounts (
    account_id,
    vendor,
    tenant_id,
    priority,
    home_dir,
    config_dir,
    status
) VALUES (
    :'account_id'::uuid,
    :'canonical_vendor',
    :'workspace_id'::uuid,
    :'priority'::int,
    :'home_dir',
    :'config_dir',
    'available'
)
ON CONFLICT (account_id) DO UPDATE
   SET priority = EXCLUDED.priority,
       home_dir = EXCLUDED.home_dir,
       config_dir = EXCLUDED.config_dir,
       updated_at = now()
 WHERE (accounts.priority, accounts.home_dir, accounts.config_dir)
       IS DISTINCT FROM
       (EXCLUDED.priority, EXCLUDED.home_dir, EXCLUDED.config_dir);

INSERT INTO approved_accounts (
    tenant_id,
    account_id,
    allowed,
    worktype_scope
) VALUES (
    :'workspace_id'::uuid,
    :'account_id'::uuid,
    true,
    :'worktype_scope'
)
ON CONFLICT (tenant_id, account_id) DO UPDATE
   SET worktype_scope = EXCLUDED.worktype_scope
 WHERE approved_accounts.allowed = true
   AND approved_accounts.worktype_scope
       IS DISTINCT FROM
       EXCLUDED.worktype_scope;

INSERT INTO assignments (agent_id, account_id)
VALUES (:'agent_id'::uuid, :'account_id'::uuid)
ON CONFLICT (agent_id) DO NOTHING;

COMMIT;

SELECT 'OK_METADATA_ASSIGNMENT' AS result,
       (SELECT count(*) FROM accounts WHERE account_id = :'account_id'::uuid) AS accounts,
       (SELECT count(*) FROM approved_accounts
         WHERE tenant_id = :'workspace_id'::uuid
           AND account_id = :'account_id'::uuid
           AND allowed) AS approvals,
       (SELECT count(*) FROM assignments
         WHERE agent_id = :'agent_id'::uuid
           AND account_id = :'account_id'::uuid) AS assignments;
