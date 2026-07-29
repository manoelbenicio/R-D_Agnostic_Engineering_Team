-- ============================================================================
-- Multica PostgreSQL Least Privilege Rollback Script (ORQ-60)
-- Target Database: multica_transition
-- Executed By: multica_recovery (proven recovery authority)
--
-- Objective:
-- Emergency rollback procedure to restore SUPERUSER status to multica_transition
-- and revert schema ownership using multica_recovery.
--
-- Zero-Secret Standard: No plaintext secrets in script or logs.
-- ============================================================================

BEGIN;

-- 1. Restore Administrative Privileges to multica_transition
ALTER ROLE multica_transition WITH SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION;

-- 2. Restore Schema Ownership to multica_transition
ALTER SCHEMA public OWNER TO multica_transition;

-- 3. Dynamically Reassign Objects Back to multica_transition
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO multica_transition', r.tablename);
    END LOOP;

    FOR r IN (SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public') LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO multica_transition', r.sequence_name);
    END LOOP;

    FOR r IN (SELECT table_name FROM information_schema.views WHERE table_schema = 'public') LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO multica_transition', r.table_name);
    END LOOP;

    FOR r IN (
        SELECT p.proname, pg_get_function_identity_arguments(p.oid) AS args
        FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'public'
    ) LOOP
        EXECUTE format('ALTER FUNCTION public.%I(%s) OWNER TO multica_transition', r.proname, r.args);
    END LOOP;
END $$;

COMMIT;
