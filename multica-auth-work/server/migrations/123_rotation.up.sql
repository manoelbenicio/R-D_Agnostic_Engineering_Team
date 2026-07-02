CREATE TABLE accounts (
    account_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor            TEXT NOT NULL,
    tenant_id         UUID NOT NULL,
    priority          INT NOT NULL DEFAULT 0,
    home_dir          TEXT NOT NULL DEFAULT '',
    config_dir        TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'available'
        CHECK (status IN ('available', 'leased', 'exhausted', 'cooldown', 'degraded')),
    tokens_per_window BIGINT NOT NULL DEFAULT 0,
    tokens_used       BIGINT NOT NULL DEFAULT 0,
    window_start      TIMESTAMPTZ,
    cooldown_until    TIMESTAMPTZ,
    last_error        TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_vendor_tenant ON accounts(vendor, tenant_id);
CREATE INDEX idx_accounts_select ON accounts(vendor, tenant_id, status, priority);
CREATE INDEX idx_accounts_cooldown_until ON accounts(cooldown_until);

CREATE TABLE credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    vendor        TEXT NOT NULL,
    secret_ref    TEXT NOT NULL,
    format        TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at    TIMESTAMPTZ
);

CREATE INDEX idx_credentials_account ON credentials(account_id);
CREATE UNIQUE INDEX uq_credentials_active_account ON credentials(account_id) WHERE expires_at IS NULL;

CREATE TABLE assignments (
    agent_id    UUID PRIMARY KEY,
    account_id  UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assignments_account ON assignments(account_id);

CREATE TABLE rotation_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID NOT NULL,
    from_account_id UUID REFERENCES accounts(account_id) ON DELETE SET NULL,
    to_account_id   UUID REFERENCES accounts(account_id) ON DELETE SET NULL,
    reason          TEXT NOT NULL
        CHECK (reason IN ('quota_exhausted_reactive', 'quota_forecast_proactive', 'login_failed', 'manual')),
    at              TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rotation_events_agent_at ON rotation_events(agent_id, at DESC);
CREATE INDEX idx_rotation_events_to_account ON rotation_events(to_account_id);
