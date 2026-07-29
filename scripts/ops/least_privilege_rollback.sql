-- ============================================================================
-- Multica PostgreSQL Least Privilege Rollback Script (ORQ-60)
-- Target Database: multica_transition
-- Executed By: multica_recovery (or superuser session via ORQ-35 OS Peer Map)
--
-- Objective:
-- Emergency rollback procedure to restore SUPERUSER status to multica_transition
-- and revert schema/object ownership.
--
-- Zero-Secret Standard: Zero plaintext secrets embedded in script, logs, or command argv.
-- ============================================================================

BEGIN;

-- 1. Restore Administrative Privileges to multica_transition
ALTER ROLE multica_transition WITH SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION;

-- 2. Restore Schema Ownership to multica_transition
ALTER SCHEMA public OWNER TO multica_transition;

-- 3. Dynamically Reassign Ownership of All Object Classes Back to multica_transition
DO $$
DECLARE
    rec RECORD;
BEGIN
    -- Tables & Partitions
    FOR rec IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO multica_transition', rec.tablename);
    END LOOP;

    -- Views
    FOR rec IN (SELECT viewname FROM pg_views WHERE schemaname = 'public') LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO multica_transition', rec.viewname);
    END LOOP;

    -- Materialized Views
    FOR rec IN (SELECT matviewname FROM pg_matviews WHERE schemaname = 'public') LOOP
        EXECUTE format('ALTER MATERIALIZED VIEW public.%I OWNER TO multica_transition', rec.matviewname);
    END LOOP;

    -- Sequences
    FOR rec IN (
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relkind = 'S'
    ) LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO multica_transition', rec.relname);
    END LOOP;

    -- Functions & Procedures
    FOR rec IN (
        SELECT p.proname, pg_get_function_identity_arguments(p.oid) AS args,
               CASE WHEN p.prokind = 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END AS object_type
        FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'public'
    ) LOOP
        EXECUTE format('ALTER %s public.%I(%s) OWNER TO multica_transition', rec.object_type, rec.proname, rec.args);
    END LOOP;

    -- Standalone User-Defined Types / Domains (excluding table row types where typrelid != 0)
    FOR rec IN (
        SELECT t.typname
        FROM pg_type t
        JOIN pg_namespace n ON t.typnamespace = n.oid
        WHERE n.nspname = 'public' AND t.typtype IN ('c', 'd', 'e') AND t.typrelid = 0
    ) LOOP
        EXECUTE format('ALTER TYPE public.%I OWNER TO multica_transition', rec.typname);
    END LOOP;
END $$;

COMMIT;
