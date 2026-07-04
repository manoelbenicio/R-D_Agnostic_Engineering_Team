CREATE TABLE approved_accounts (
    approved_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    account_id     UUID NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    allowed        BOOLEAN NOT NULL DEFAULT true,
    worktype_scope TEXT
        CHECK (worktype_scope IN ('GENERAL', 'HEAVY', 'CHEAP', 'REVIEW')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, account_id)
);

CREATE INDEX idx_approved_accounts_tenant ON approved_accounts(tenant_id);
