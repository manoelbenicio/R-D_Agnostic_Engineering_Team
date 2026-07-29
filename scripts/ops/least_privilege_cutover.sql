-- ============================================================================
-- Multica PostgreSQL Least Privilege Cutover Script (ORQ-60)
-- Target Database: multica_transition
-- Initial Authorized User: multica_transition
--
-- Objective:
-- 1. Create separate recovery, owner, migrator, and app roles.
-- 2. Perform all privileged schema/object ownership transfers and DML grant statements FIRST while authorized.
-- 3. Provision login credentials via dynamic environment variables (no plaintext secrets).
-- 4. Demote application role (multica_transition / multica_app) LAST as the final statement.
--
-- Zero-Secret Standard: No plaintext passwords in code or logs.
-- ============================================================================

BEGIN;

-- 1. Create Recovery / Administrative Authority FIRST (proven separate recovery role)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_recovery') THEN
        CREATE ROLE multica_recovery WITH LOGIN SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION;
    END IF;
END $$;

-- 2. Create Decoupled Role Hierarchy (Owner, Migrator, App)
DO $$
BEGIN
    -- Owner Role (nologin role that owns database objects)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_owner') THEN
        CREATE ROLE multica_owner WITH NOLOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;

    -- Migrator Role (login role used for running DDL migrations during maintenance windows)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_migrator') THEN
        CREATE ROLE multica_migrator WITH LOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;

    -- App Role (login role for backend runtime if separated from multica_transition)
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'multica_app') THEN
        CREATE ROLE multica_app WITH LOGIN NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS NOREPLICATION;
    END IF;
END $$;

-- Assign Migrator to Owner role hierarchy so it can execute DDL as multica_owner
GRANT multica_owner TO multica_migrator;

-- 3. Perform Privileged Schema & Object Ownership Transfer
ALTER SCHEMA public OWNER TO multica_owner;

-- Dynamically reassign ownership of existing public schema objects to multica_owner
DO $$
DECLARE
    r RECORD;
BEGIN
    -- Tables
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tableowner != 'multica_owner') LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO multica_owner', r.tablename);
    END LOOP;

    -- Sequences
    FOR r IN (SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public') LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO multica_owner', r.sequence_name);
    END LOOP;

    -- Views
    FOR r IN (SELECT table_name FROM information_schema.views WHERE table_schema = 'public') LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO multica_owner', r.table_name);
    END LOOP;

    -- Routines / Functions
    FOR r IN (
        SELECT p.proname, pg_get_function_identity_arguments(p.oid) AS args
        FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'public'
    ) LOOP
        EXECUTE format('ALTER FUNCTION public.%I(%s) OWNER TO multica_owner', r.proname, r.args);
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
