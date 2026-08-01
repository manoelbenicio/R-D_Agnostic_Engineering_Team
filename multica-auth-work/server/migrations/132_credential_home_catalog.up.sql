-- SPE-6: pathless, generation-pinned credential-home catalog projection.
--
-- Discovery and filesystem identity remain daemon-local. The control plane
-- stores only opaque UUID references and lifecycle metadata. In particular,
-- this schema has no source-home location or operating-system identity field.

CREATE TABLE credential_home_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    daemon_id TEXT NOT NULL CHECK (btrim(daemon_id) <> ''),
    generation BIGINT NOT NULL DEFAULT 0 CHECK (generation >= 0),
    state TEXT NOT NULL DEFAULT 'available'
        CHECK (state IN ('available', 'reconciling', 'unavailable')),
    watermark TEXT NOT NULL DEFAULT 'normal'
        CHECK (watermark IN ('normal', 'low', 'high', 'critical')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT credential_home_catalog_workspace_daemon_key
        UNIQUE (workspace_id, daemon_id),
    CONSTRAINT credential_home_catalog_id_workspace_key
        UNIQUE (id, workspace_id),
    CONSTRAINT credential_home_catalog_id_generation_key
        UNIQUE (id, generation)
);

CREATE TABLE credential_home_catalog_generation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id UUID NOT NULL REFERENCES credential_home_catalog(id) ON DELETE RESTRICT,
    previous_generation BIGINT NOT NULL CHECK (previous_generation >= 0),
    generation BIGINT NOT NULL CHECK (generation > 0),
    scan_kind TEXT NOT NULL
        CHECK (scan_kind IN ('startup', 'periodic', 'hint_loss', 'overflow', 'watcher_restart', 'requested')),
    counters JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(counters) = 'object'),
    catalog_digest TEXT NOT NULL CHECK (catalog_digest ~ '^[0-9a-f]{64}$'),
    started_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT credential_home_catalog_generation_sequence_check
        CHECK (previous_generation < generation),
    CONSTRAINT credential_home_catalog_generation_catalog_key
        UNIQUE (catalog_id, generation),
    CONSTRAINT credential_home_catalog_generation_id_catalog_generation_key
        UNIQUE (id, catalog_id, generation)
);

CREATE TABLE credential_home_catalog_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id UUID NOT NULL,
    catalog_id UUID NOT NULL,
    generation BIGINT NOT NULL,
    home_ref UUID NOT NULL,
    provider TEXT NOT NULL CHECK (btrim(provider) <> ''),
    approved BOOLEAN NOT NULL DEFAULT false,
    state TEXT NOT NULL
        CHECK (state IN ('candidate', 'healthy', 'degraded', 'quarantined', 'missing', 'draining', 'retired')),
    reason_code TEXT CHECK (reason_code IS NULL OR btrim(reason_code) <> ''),
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    last_full_scan_at TIMESTAMPTZ NOT NULL,
    health_watermark TIMESTAMPTZ,
    missing_watermark TIMESTAMPTZ,
    ttl INTERVAL NOT NULL CHECK (ttl > interval '0 seconds'),
    retention_deadline TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT credential_home_catalog_entry_generation_fkey
        FOREIGN KEY (generation_id, catalog_id, generation)
        REFERENCES credential_home_catalog_generation(id, catalog_id, generation)
        ON DELETE RESTRICT,
    CONSTRAINT credential_home_catalog_entry_ref_key
        UNIQUE (catalog_id, generation, home_ref),
    CONSTRAINT credential_home_catalog_entry_id_catalog_generation_ref_key
        UNIQUE (id, catalog_id, generation, home_ref),
    CONSTRAINT credential_home_catalog_entry_time_check
        CHECK (
            first_seen_at <= last_seen_at
            AND last_seen_at <= last_full_scan_at
            AND retention_deadline > first_seen_at
        ),
    CONSTRAINT credential_home_catalog_entry_missing_check
        CHECK (
            (state IN ('missing', 'draining', 'retired') AND missing_watermark IS NOT NULL)
            OR (state NOT IN ('missing', 'draining', 'retired'))
        )
);

CREATE INDEX idx_credential_home_catalog_workspace
    ON credential_home_catalog(workspace_id, state, updated_at DESC, id);
CREATE INDEX idx_credential_home_catalog_generation_catalog
    ON credential_home_catalog_generation(catalog_id, generation DESC);
CREATE INDEX idx_credential_home_catalog_entry_current
    ON credential_home_catalog_entry(catalog_id, generation, state, home_ref);
CREATE INDEX idx_credential_home_catalog_entry_ref
    ON credential_home_catalog_entry(home_ref, generation DESC);

-- Published generations and their entries are evidence. A later scan appends
-- another complete generation; it never rewrites a previously published one.
CREATE TRIGGER credential_home_catalog_generation_immutable
BEFORE UPDATE OR DELETE ON credential_home_catalog_generation
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();

CREATE TRIGGER credential_home_catalog_entry_immutable
BEFORE UPDATE OR DELETE ON credential_home_catalog_entry
FOR EACH ROW EXECUTE FUNCTION reject_runtime_manager_immutable_mutation();
