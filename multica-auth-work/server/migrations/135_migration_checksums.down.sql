-- Never discard recorded source-integrity evidence. A pristine, never-used
-- checksum schema may be rolled back; once any migration was tracked, deploy
-- forward instead.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM schema_migrations
        WHERE up_checksum IS NOT NULL OR down_checksum IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'cannot roll back migration checksum tracking while evidence remains';
    END IF;
END
$$;

ALTER TABLE schema_migrations
    DROP CONSTRAINT schema_migrations_checksum_pair_check,
    DROP COLUMN down_checksum,
    DROP COLUMN up_checksum;
