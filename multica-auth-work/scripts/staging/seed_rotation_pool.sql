-- Staging rotation pool seed for Codex.
-- Re-runnable: upserts two stable account rows and their active credential refs.
-- Human aliases:
--   stg-codex-a = 10000000-0000-4000-8000-000000000001, priority 1, 96% ledger used
--   stg-codex-b = 10000000-0000-4000-8000-000000000002, priority 2, spare account
--
-- tenant_id is a stable staging tenant/workspace id for rotation smoke tests.
-- Credential material is not stored here; secret_ref points to the isolated
-- staging auth.json location seeded on disk before running this SQL.

BEGIN;

INSERT INTO accounts (
    account_id,
    vendor,
    tenant_id,
    priority,
    home_dir,
    config_dir,
    status,
    tokens_per_window,
    tokens_used,
    window_start,
    cooldown_until,
    last_error,
    created_at,
    updated_at
) VALUES
    (
        '10000000-0000-4000-8000-000000000001',
        'codex',
        '20000000-0000-4000-8000-000000000001',
        1,
        '/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-a',
        '/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-a',
        'available',
        1000000,
        960000,
        now() - interval '4 hours 45 minutes',
        NULL,
        '',
        now(),
        now()
    ),
    (
        '10000000-0000-4000-8000-000000000002',
        'codex',
        '20000000-0000-4000-8000-000000000001',
        2,
        '/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-b',
        '/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-b',
        'available',
        1000000,
        100000,
        now() - interval '1 hour',
        NULL,
        '',
        now(),
        now()
    )
ON CONFLICT (account_id) DO UPDATE
   SET vendor = EXCLUDED.vendor,
       tenant_id = EXCLUDED.tenant_id,
       priority = EXCLUDED.priority,
       home_dir = EXCLUDED.home_dir,
       config_dir = EXCLUDED.config_dir,
       status = EXCLUDED.status,
       tokens_per_window = EXCLUDED.tokens_per_window,
       tokens_used = EXCLUDED.tokens_used,
       window_start = EXCLUDED.window_start,
       cooldown_until = EXCLUDED.cooldown_until,
       last_error = EXCLUDED.last_error,
       updated_at = now();

INSERT INTO credentials (
    credential_id,
    account_id,
    vendor,
    secret_ref,
    format,
    created_at,
    expires_at
) VALUES
    (
        '30000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        'codex',
        'file:///mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-a/auth.json',
        'codex_auth_json_ref',
        now(),
        NULL
    ),
    (
        '30000000-0000-4000-8000-000000000002',
        '10000000-0000-4000-8000-000000000002',
        'codex',
        'file:///mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/scripts/staging/creds/codex-b/auth.json',
        'codex_auth_json_ref',
        now(),
        NULL
    )
ON CONFLICT (account_id) WHERE expires_at IS NULL DO UPDATE
   SET vendor = EXCLUDED.vendor,
       secret_ref = EXCLUDED.secret_ref,
       format = EXCLUDED.format,
       expires_at = NULL;

COMMIT;
