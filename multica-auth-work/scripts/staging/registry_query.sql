-- registry_query.sql
-- Per-tenant account governance registry (design.md §9: Approved vs All).
-- For each account in a tenant, shows its approval state plus vendor/status/priority.
-- LEFT JOIN so accounts with no approved_accounts row still appear (approved = false).
--
-- Run:
--   docker exec -i multica-postgres-1 psql -U multica -d multica \
--     < scripts/staging/registry_query.sql

SELECT
    a.tenant_id,
    a.account_id,
    a.vendor,
    a.status,
    a.priority,
    COALESCE(aa.allowed, false)  AS approved,
    aa.worktype_scope,
    aa.created_at                 AS approved_at
FROM accounts AS a
LEFT JOIN approved_accounts AS aa
    ON aa.account_id = a.account_id
   AND aa.tenant_id   = a.tenant_id
ORDER BY a.tenant_id, approved DESC, a.priority DESC, a.vendor, a.account_id;

