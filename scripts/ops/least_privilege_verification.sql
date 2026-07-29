-- ============================================================================
-- Multica PostgreSQL Least Privilege Audit & Verification Script (ORQ-60)
-- Target Database: multica_transition
--
-- Objective:
-- Audit role privileges and confirm application roles lack SUPERUSER, CREATEROLE, CREATEDB, BYPASSRLS, REPLICATION.
-- Verify object class ownership and assert zero non-owner objects in public schema.
-- Content-free output (zero secret leakage).
-- ============================================================================

-- 1. Inspect Role Security Attributes
SELECT
    rolname,
    rolsuper,
    rolcreaterole,
    rolcreatedb,
    rolcanlogin,
    rolreplication,
    rolbypassrls
FROM pg_roles
WHERE rolname IN ('multica_transition', 'multica_app', 'multica_migrator', 'multica_owner', 'multica_recovery')
ORDER BY rolname;

-- 2. Assert Zero Superuser Application Roles
SELECT
    COUNT(*) AS violating_superuser_app_roles
FROM pg_roles
WHERE rolname IN ('multica_transition', 'multica_app')
  AND rolsuper = true;

-- 3. Assert Recovery Authority Security Profile (LOGIN Restricted via OS Peer Map)
SELECT
    rolname,
    rolsuper,
    rolcanlogin
FROM pg_roles
WHERE rolname = 'multica_recovery';

-- 4. Assert Schema Ownership
SELECT
    nspname AS schema_name,
    r.rolname AS schema_owner
FROM pg_namespace n
JOIN pg_roles r ON n.nspowner = r.oid
WHERE nspname = 'public';

-- 5. Inventory Non-Owner Objects across Object Classes in Schema public
SELECT
    'tables' AS object_class, COUNT(*) AS non_owner_count
FROM pg_tables WHERE schemaname = 'public' AND tableowner != 'multica_owner'
UNION ALL
SELECT
    'views' AS object_class, COUNT(*) AS non_owner_count
FROM pg_views WHERE schemaname = 'public' AND viewowner != 'multica_owner'
UNION ALL
SELECT
    'materialized_views' AS object_class, COUNT(*) AS non_owner_count
FROM pg_matviews WHERE schemaname = 'public' AND matviewowner != 'multica_owner'
UNION ALL
SELECT
    'sequences' AS object_class, COUNT(*) AS non_owner_count
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner
WHERE n.nspname = 'public' AND c.relkind = 'S' AND r.rolname != 'multica_owner'
UNION ALL
SELECT
    'functions_and_procedures' AS object_class, COUNT(*) AS non_owner_count
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
JOIN pg_roles ro ON p.proowner = ro.oid
WHERE n.nspname = 'public' AND ro.rolname != 'multica_owner'
UNION ALL
SELECT
    'standalone_user_types_and_domains' AS object_class, COUNT(*) AS non_owner_count
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
JOIN pg_roles ro ON t.typowner = ro.oid
WHERE n.nspname = 'public' AND ro.rolname != 'multica_owner' AND t.typtype IN ('c', 'd', 'e') AND t.typrelid = 0;
