-- ============================================================================
-- Multica PostgreSQL Least Privilege Cutover Script (ORQ-60)
-- Target Database: multica_transition
-- Initial Authorized Session Role: multica_transition
--
-- Objective:
-- 1. Create separate NOLOGIN roles for recovery, owner, migrator, and app.
-- 2. Perform all privileged schema/object ownership transfers and DML grants FIRST while authorized.
-- 3. Inventory and reassign all object classes (tables, partitions, views, matviews, sequences, functions, procedures, standalone types).
-- 4. Keep recovery, migrator, and app roles NOLOGIN until independently provisioned (ORQ-35 peer map / SCRAM).
-- 5. Demote application role multica_transition LAST as the final statement.
--
-- Zero-Secret Standard: Zero plaintext secrets embedded in script, logs, or command argv.
-- ============================================================================

BEGIN;

-- 1. Create NOLOGIN Recovery Authority (Reachable ONLY via ORQ-35 OS Peer Map or Secret Handoff)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_recovery') THEN
        CREATE ROLE multica_recovery WITH NOLOGIN SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION;
    END IF;
END $$;

-- 2. Create Decoupled Role Hierarchy (Owner, Migrator, App as NOLOGIN)
DO $$
BEGIN
    -- Owner Role (nologin role that owns database objects)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_owner') THEN
        CREATE ROLE multica_owner WITH NOLOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;

    -- Migrator Role (nologin until provisioned in maintenance window)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_migrator') THEN
        CREATE ROLE multica_migrator WITH NOLOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;

    -- App Role (nologin until provisioned in maintenance window)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_app') THEN
        CREATE ROLE multica_app WITH NOLOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;
END $$;

-- Assign current executing user admin option on multica_owner so it can grant it to migrator
GRANT multica_owner TO CURRENT_USER WITH ADMIN OPTION;

-- Assign Migrator to Owner role hierarchy so it can execute DDL as multica_owner
GRANT multica_owner TO multica_migrator;

-- Revoke multica_owner from current executing user so application role does not retain ownership inheritance
REVOKE multica_owner FROM CURRENT_USER;

-- 3. Perform Privileged Schema & Comprehensive Object Class Ownership Transfer
ALTER SCHEMA public OWNER TO multica_owner;
REVOKE CREATE ON SCHEMA public FROM PUBLIC, multica_transition, multica_app;

-- Dynamically reassign ownership of ALL public schema object classes to multica_owner
DO $$
DECLARE
    rec RECORD;
BEGIN
    -- Tables & Partitions
    FOR rec IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tableowner != 'multica_owner') LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO multica_owner', rec.tablename);
    END LOOP;

    -- Views
    FOR rec IN (SELECT viewname FROM pg_views WHERE schemaname = 'public' AND viewowner != 'multica_owner') LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO multica_owner', rec.viewname);
    END LOOP;

    -- Materialized Views
    FOR rec IN (SELECT matviewname FROM pg_matviews WHERE schemaname = 'public' AND matviewowner != 'multica_owner') LOOP
        EXECUTE format('ALTER MATERIALIZED VIEW public.%I OWNER TO multica_owner', rec.matviewname);
    END LOOP;

    -- Sequences
    FOR rec IN (
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_roles ro ON ro.oid = c.relowner
        WHERE n.nspname = 'public' AND c.relkind = 'S' AND ro.rolname != 'multica_owner'
    ) LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO multica_owner', rec.relname);
    END LOOP;

    -- Functions & Procedures
    FOR rec IN (
        SELECT p.proname, pg_get_function_identity_arguments(p.oid) AS args,
               CASE WHEN p.prokind = 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END AS object_type
        FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        JOIN pg_roles ro ON p.proowner = ro.oid
        WHERE n.nspname = 'public' AND ro.rolname != 'multica_owner'
    ) LOOP
        EXECUTE format('ALTER %s public.%I(%s) OWNER TO multica_owner', rec.object_type, rec.proname, rec.args);
    END LOOP;

    -- Standalone User-Defined Types / Domains (excluding table row types where typrelid != 0)
    FOR rec IN (
        SELECT t.typname
        FROM pg_type t
        JOIN pg_namespace n ON t.typnamespace = n.oid
        JOIN pg_roles ro ON t.typowner = ro.oid
        WHERE n.nspname = 'public' AND ro.rolname != 'multica_owner' AND t.typtype IN ('c', 'd', 'e') AND t.typrelid = 0
    ) LOOP
        EXECUTE format('ALTER TYPE public.%I OWNER TO multica_owner', rec.typname);
    END LOOP;
END $$;

-- 4. Grant Database & Schema Level Access to App, Migrator, and Transition Roles
GRANT CONNECT, TEMPORARY ON DATABASE multica_transition TO multica_app, multica_transition, multica_migrator;
GRANT USAGE ON SCHEMA public TO multica_app, multica_transition, multica_migrator;
GRANT CREATE ON SCHEMA public TO multica_owner, multica_migrator;

-- 5. Grant Explicit DML Privileges on Existing Objects to Application Roles
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO multica_app, multica_transition;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO multica_app, multica_transition;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO multica_app, multica_transition;

-- 6. Configure Default Privileges for Future Objects Created by Owner or Migrator
ALTER DEFAULT PRIVILEGES FOR ROLE multica_owner IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO multica_app, multica_transition;

ALTER DEFAULT PRIVILEGES FOR ROLE multica_owner IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO multica_app, multica_transition;

ALTER DEFAULT PRIVILEGES FOR ROLE multica_owner IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO multica_app, multica_transition;

ALTER DEFAULT PRIVILEGES FOR ROLE multica_migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO multica_app, multica_transition;

ALTER DEFAULT PRIVILEGES FOR ROLE multica_migrator IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO multica_app, multica_transition;

ALTER DEFAULT PRIVILEGES FOR ROLE multica_migrator IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO multica_app, multica_transition;

-- 7. Demote Application Roles LAST as the Final Statement
ALTER ROLE multica_app NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
ALTER ROLE multica_transition NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;

COMMIT;
