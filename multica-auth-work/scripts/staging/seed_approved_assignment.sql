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
  \echo 'E_WORKSPACE_ID_REQUIRED'
  \quit 1
\endif
\if :{?agent_id}
\else
  \echo 'E_AGENT_ID_REQUIRED'
  \quit 1
\endif
\if :{?account_id}
\else
  \echo 'E_ACCOUNT_ID_REQUIRED'
  \quit 1
\endif
\if :{?vendor}
\else
  \echo 'E_VENDOR_REQUIRED'
  \quit 1
\endif
\if :{?priority}
\else
  \echo 'E_PRIORITY_REQUIRED'
  \quit 1
\endif
\if :{?home_dir}
\else
  \echo 'E_HOME_DIR_REQUIRED'
  \quit 1
\endif
\if :{?config_dir}
\else
  \echo 'E_CONFIG_DIR_REQUIRED'
  \quit 1
\endif
\if :{?worktype_scope}
\else
  \echo 'E_WORKTYPE_SCOPE_REQUIRED'
  \quit 1
\endif

BEGIN;

SELECT pg_advisory_xact_lock(hashtextextended(:'account_id', 0));

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
       AND (tenant_id <> :'workspace_id'::uuid OR vendor <> :'vendor')
) AS account_conflict,
EXISTS (
    SELECT 1
      FROM assignments
     WHERE agent_id = :'agent_id'::uuid
       AND account_id <> :'account_id'::uuid
) AS assignment_conflict
\gset

\if :workspace_exists
\else
  \echo 'E_WORKSPACE_NOT_FOUND'
  \quit 1
\endif
\if :agent_exists
\else
  \echo 'E_AGENT_WORKSPACE_MISMATCH'
  \quit 1
\endif
\if :account_conflict
  \echo 'E_ACCOUNT_ID_CONFLICT'
  \quit 1
\endif
\if :assignment_conflict
  \echo 'E_AGENT_ALREADY_ASSIGNED'
  \quit 1
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
    :'vendor',
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
   SET allowed = true,
       worktype_scope = EXCLUDED.worktype_scope
 WHERE (approved_accounts.allowed, approved_accounts.worktype_scope)
       IS DISTINCT FROM
       (EXCLUDED.allowed, EXCLUDED.worktype_scope);

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
