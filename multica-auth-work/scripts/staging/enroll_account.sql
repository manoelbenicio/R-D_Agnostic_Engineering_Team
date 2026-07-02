-- enroll_account.sql
-- Idempotent enrollment of one vendor account into the rotation pool, with its
-- active credential reference. Parameterized via psql variables (passed with
-- `psql -v name=value` and referenced with the safe string-literal syntax
-- `:'name'`). Intended to be streamed to the DB container, e.g.:
--   docker exec -i multica-postgres-1 psql -U multica -d multica \
--     -v vendor=... -v account_id=... ... < scripts/staging/enroll_account.sql
--
-- Schema (EXACT, from server/migrations/123_rotation.up.sql — do not invent):
--   accounts(account_id uuid pk default gen_random_uuid(), vendor text,
--     tenant_id uuid, priority int, home_dir text, config_dir text, status text
--     check(available|leased|exhausted|cooldown|degraded), tokens_per_window bigint,
--     tokens_used bigint, window_start timestamptz, cooldown_until timestamptz,
--     last_error text, created_at timestamptz, updated_at timestamptz)
--   credentials(credential_id uuid pk default gen_random_uuid(), account_id uuid
--     fk on delete cascade, vendor text, secret_ref text, format text,
--     created_at timestamptz, expires_at timestamptz)
--   unique partial index uq_credentials_active_account on
--     credentials(account_id) where expires_at is null
--
-- Required psql variables:
--   vendor, account_id, tenant_id, priority, home_dir, config_dir, status,
--   tokens_per_window, tokens_used, secret_ref, format
--
-- Safe defaults for a freshly enrolled REAL account are baked in here:
--   window_start = now(), cooldown_until = NULL, last_error = '',
--   status (passed, default 'available'), tokens_used (passed, default 0).
--   created_at is preserved across updates (NOT overwritten on conflict).
--
-- Re-runnable / idempotent: both statements are UPSERTs.
--   - accounts: ON CONFLICT (account_id) DO UPDATE (primary key)
--   - credentials: ON CONFLICT (account_id) WHERE expires_at IS NULL DO UPDATE
--     (the active-credential partial unique index) — credential_id/created_at
--     are preserved; only vendor/secret_ref/format/expires_at are refreshed.

\set ON_ERROR_STOP on

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
) VALUES (
    :'account_id'::uuid,
    :'vendor',
    :'tenant_id'::uuid,
    :'priority'::int,
    :'home_dir',
    :'config_dir',
    :'status',
    :'tokens_per_window'::bigint,
    :'tokens_used'::bigint,
    now(),
    NULL,
    '',
    now(),
    now()
)
ON CONFLICT (account_id) DO UPDATE
   SET vendor            = EXCLUDED.vendor,
       tenant_id         = EXCLUDED.tenant_id,
       priority          = EXCLUDED.priority,
       home_dir          = EXCLUDED.home_dir,
       config_dir        = EXCLUDED.config_dir,
       status            = EXCLUDED.status,
       tokens_per_window = EXCLUDED.tokens_per_window,
       tokens_used       = EXCLUDED.tokens_used,
       window_start      = now(),
       cooldown_until    = NULL,
       last_error        = '',
       updated_at        = now();
       -- created_at intentionally NOT overwritten: preserves original enrollment time.

INSERT INTO credentials (
    account_id,
    vendor,
    secret_ref,
    format,
    created_at,
    expires_at
) VALUES (
    :'account_id'::uuid,
    :'vendor',
    :'secret_ref',
    :'format',
    now(),
    NULL
)
ON CONFLICT (account_id) WHERE expires_at IS NULL DO UPDATE
   SET vendor     = EXCLUDED.vendor,
       secret_ref = EXCLUDED.secret_ref,
       format     = EXCLUDED.format,
       expires_at = NULL;
       -- credential_id (pk) and created_at preserved on update.

COMMIT;

-- Self-verifying snapshot of the enrolled account + its active credential.
-- Prints ids/paths/status only. Credential file CONTENTS are never selected.
SELECT a.account_id,
       a.vendor,
       a.priority,
       a.status,
       a.home_dir,
       a.config_dir,
       a.tokens_per_window,
       a.tokens_used,
       COALESCE(c.secret_ref, '') AS secret_ref,
       COALESCE(c.format, '')     AS credential_format
FROM accounts a
LEFT JOIN credentials c
       ON c.account_id = a.account_id
      AND c.expires_at IS NULL
WHERE a.account_id = :'account_id'::uuid;
