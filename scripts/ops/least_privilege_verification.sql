-- ============================================================================
-- Multica PostgreSQL Least Privilege Audit & Verification Script (ORQ-60)
-- Target Database: multica_transition
--
-- Objective:
-- Audit role privileges and confirm application roles lack SUPERUSER, CREATEROLE, CREATEDB, BYPASSRLS, REPLICATION.
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

-- 3. Assert Recovery Authority Exists
SELECT 
    COUNT(*) AS valid_recovery_authority_roles
FROM pg_roles
WHERE rolname = 'multica_recovery'
  AND rolsuper = true;

-- 4. Assert Schema Ownership
SELECT 
    nspname AS schema_name,
    r.rolname AS schema_owner
FROM pg_namespace n
JOIN pg_roles r ON n.nspowner = r.oid
WHERE nspname = 'public';
