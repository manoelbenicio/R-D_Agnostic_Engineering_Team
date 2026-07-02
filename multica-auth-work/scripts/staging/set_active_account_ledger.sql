\set ON_ERROR_STOP on

\if :{?workspace_id}
\else
  \echo 'missing required psql variable: workspace_id'
  \quit 1
\endif

\if :{?agent_id}
\else
  \echo 'missing required psql variable: agent_id'
  \quit 1
\endif

BEGIN;

UPDATE accounts
   SET tenant_id = :'workspace_id'::uuid,
       status = 'available',
       tokens_per_window = 1000000,
       tokens_used = 960000,
       window_start = now(),
       cooldown_until = NULL,
       updated_at = now()
 WHERE vendor = 'codex'
   AND account_id = '10000000-0000-4000-8000-000000000001'::uuid;

UPDATE accounts
   SET tenant_id = :'workspace_id'::uuid,
       status = 'available',
       tokens_per_window = 1000000,
       tokens_used = 100000,
       window_start = now(),
       cooldown_until = NULL,
       updated_at = now()
 WHERE vendor = 'codex'
   AND account_id = '10000000-0000-4000-8000-000000000002'::uuid;

INSERT INTO assignments (agent_id, account_id, assigned_at)
SELECT :'agent_id'::uuid, '10000000-0000-4000-8000-000000000001'::uuid, now()
 WHERE EXISTS (
       SELECT 1
         FROM agent
        WHERE id = :'agent_id'::uuid
          AND workspace_id = :'workspace_id'::uuid
 )
ON CONFLICT (agent_id) DO UPDATE
   SET account_id = EXCLUDED.account_id,
       assigned_at = EXCLUDED.assigned_at;

COMMIT;

SELECT a.account_id,
       a.tenant_id,
       a.priority,
       a.status,
       a.tokens_used,
       a.tokens_per_window,
       round((a.tokens_used::numeric / NULLIF(a.tokens_per_window, 0)) * 100, 2) AS pct_used,
       a.window_start,
       ass.agent_id AS active_agent_id
  FROM accounts a
  LEFT JOIN assignments ass ON ass.account_id = a.account_id
 WHERE a.vendor = 'codex'
   AND a.account_id IN (
       '10000000-0000-4000-8000-000000000001'::uuid,
       '10000000-0000-4000-8000-000000000002'::uuid
   )
 ORDER BY a.priority;
