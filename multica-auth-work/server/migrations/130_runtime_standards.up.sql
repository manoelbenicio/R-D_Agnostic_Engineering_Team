-- SPE-6: owner-global Runtime Standards and immutable policy versions.
--
-- This migration is additive and pathless. Runtime configuration documents may
-- contain only product policy values and opaque references; no source-home path
-- or credential material belongs in this schema.

CREATE TABLE runtime_standard (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    description TEXT,
    active_version_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_standard_owner_name_key UNIQUE (owner_id, name),
    CONSTRAINT runtime_standard_id_owner_key UNIQUE (id, owner_id)
);

CREATE TABLE runtime_standard_version (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_id UUID NOT NULL REFERENCES runtime_standard(id) ON DELETE RESTRICT,
    version_number BIGINT NOT NULL CHECK (version_number > 0),
    configuration JSONB NOT NULL CHECK (jsonb_typeof(configuration) = 'object'),
    configuration_digest TEXT NOT NULL
        CHECK (configuration_digest ~ '^[0-9a-f]{64}$'),
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_standard_version_number_key
        UNIQUE (standard_id, version_number),
    CONSTRAINT runtime_standard_version_standard_id_key
        UNIQUE (standard_id, id)
);

-- The composite FK makes it impossible to point a standard at a version owned
-- by another standard. NULL means that no version has been activated yet.
ALTER TABLE runtime_standard
    ADD CONSTRAINT runtime_standard_active_version_fkey
    FOREIGN KEY (id, active_version_id)
    REFERENCES runtime_standard_version(standard_id, id)
    ON DELETE RESTRICT;

CREATE TABLE runtime_standard_activation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_id UUID NOT NULL REFERENCES runtime_standard(id) ON DELETE RESTRICT,
    previous_version_id UUID,
    new_version_id UUID NOT NULL,
    actor_id UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    request_id TEXT NOT NULL CHECK (btrim(request_id) <> ''),
    correlation_id TEXT CHECK (correlation_id IS NULL OR btrim(correlation_id) <> ''),
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    capability_digest TEXT NOT NULL
        CHECK (capability_digest ~ '^[0-9a-f]{64}$'),
    activated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_standard_activation_request_key
        UNIQUE (standard_id, request_id),
    CONSTRAINT runtime_standard_activation_previous_version_fkey
        FOREIGN KEY (standard_id, previous_version_id)
        REFERENCES runtime_standard_version(standard_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_standard_activation_new_version_fkey
        FOREIGN KEY (standard_id, new_version_id)
        REFERENCES runtime_standard_version(standard_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_standard_activation_version_change_check
        CHECK (previous_version_id IS NULL OR previous_version_id <> new_version_id)
);

CREATE INDEX idx_runtime_standard_owner
    ON runtime_standard(owner_id, created_at, id);
CREATE INDEX idx_runtime_standard_version_standard
    ON runtime_standard_version(standard_id, version_number DESC);
CREATE INDEX idx_runtime_standard_activation_standard
    ON runtime_standard_activation(standard_id, activated_at DESC, id);

-- Versions and activation audit rows are append-only. Rollback changes the
-- active pointer and appends an activation record; it never rewrites history.
CREATE FUNCTION reject_runtime_manager_immutable_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = format('%s rows are immutable; create or activate another version instead', TG_TABLE_NAME);
END;
$$;

CREATE TRIGGER runtime_standard_version_immutable
BEFORE UPDATE OR DELETE ON runtime_standard_version
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER runtime_standard_activation_immutable
BEFORE UPDATE OR DELETE ON runtime_standard_activation
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();
