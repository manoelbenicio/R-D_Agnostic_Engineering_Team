-- Future-only migration source integrity. Existing schema_migrations rows stay
-- NULL/UNKNOWN: version 128 in particular has retrospective source and schema
-- evidence, but its historical execution cannot be cryptographically proven.
-- The migration runner records raw-file SHA-256 only for migrations it applies
-- while these columns exist and fails closed on any later mismatch.
--
-- Keep the runner-owned table declaration self-contained for schema tooling.
-- At runtime this is an exact no-op because cmd/migrate creates the same shape
-- before applying migration files.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE schema_migrations
    ADD COLUMN up_checksum TEXT,
    ADD COLUMN down_checksum TEXT,
    ADD CONSTRAINT schema_migrations_checksum_pair_check
        CHECK (
            (up_checksum IS NULL AND down_checksum IS NULL)
            OR
            (up_checksum ~ '^[0-9a-f]{64}$'
             AND down_checksum ~ '^[0-9a-f]{64}$')
        );

COMMENT ON COLUMN schema_migrations.up_checksum IS
    'Raw-file lowercase SHA-256 recorded at application time; NULL means historical execution identity is unknown.';
COMMENT ON COLUMN schema_migrations.down_checksum IS
    'Raw-file lowercase SHA-256 of the paired rollback file recorded at application time; NULL means historical execution identity is unknown.';
