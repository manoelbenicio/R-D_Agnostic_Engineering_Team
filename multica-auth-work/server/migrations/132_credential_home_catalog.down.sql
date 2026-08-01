-- A catalog may be removed only after later runtime-binding/task-snapshot
-- migrations have been rolled back and no metadata remains. This rollback
-- never touches a daemon-local credential home.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM credential_home_catalog_entry) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'cannot roll back credential-home catalog while catalog entries are retained';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS credential_home_catalog_entry_immutable ON credential_home_catalog_entry;
DROP TRIGGER IF EXISTS credential_home_catalog_generation_immutable ON credential_home_catalog_generation;
DROP TABLE credential_home_catalog_entry;
DROP TABLE credential_home_catalog_generation;
DROP TABLE credential_home_catalog;
