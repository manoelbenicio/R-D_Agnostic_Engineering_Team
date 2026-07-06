# prodex Account Enrollment Runbook

Status: OPERATOR READY - LIVE EXECUTION OWNER-GATED

This runbook enrolls approved account profiles into the prodex-backed rotation pool without copying raw credential material into logs, events, tickets, or evidence. It covers the existing `scripts/staging/enroll_account.sh` flow and the production controls required before the same pattern is used for real tenants.

## 1. Hard Gates

Before enrolling any real profile:

- owner approval for the tenant/provider/profile scope is recorded;
- `PRODEX_HOME` and per-profile credential homes are on a real POSIX filesystem such as ext4/xfs, not drvfs, 9p, CIFS, or another permission-emulating mount;
- credential files validate as mode `0600`, credential directories as `0700`;
- Postgres is reachable and migrations `123_rotation` and `124_approved_accounts` are applied;
- the profile is approved in the control plane and has a non-secret alias;
- redaction is enabled for shell transcript capture and evidence;
- kill-switch and rollback references are known before the account can receive traffic.

Never paste OAuth tokens, API keys, cookies, `auth.json`, sqlite credential files, database URLs, Redis URLs, bearer tokens, prompts, or raw provider payloads into evidence.

## 2. Supported Vendors

The current enrollment helper accepts:

```text
codex        source credential file, stored as auth.json
kiro         source credential file, stored as kiro-cli/data.sqlite3
opus         source credential file, stored as kiro-cli/data.sqlite3
antigravity  source credential directory, stored as .gemini/antigravity-cli
```

Vendor capability and disabled-by-default decisions remain governed by `docs/vendors/vendor-capability-matrix.md`.

## 3. Dry-Run Preflight

Run from repo root:

```bash
test -x multica-auth-work/scripts/staging/enroll_account.sh
test -f multica-auth-work/scripts/staging/enroll_account.sql
find multica-auth-work/server/migrations -maxdepth 1 -name '*.up.sql' | wc -l
find multica-auth-work/server/migrations -maxdepth 1 -name '*.down.sql' | wc -l
```

For the credential source path selected by the owner:

```bash
stat -f -c '%T %m' <source-credential-path>
stat -c '%a %n' <source-credential-path>
```

Abort if the filesystem is not POSIX-safe or the permission mode cannot be enforced.

## 4. Enrollment Procedure

Use a non-secret alias. The script derives a deterministic UUID from the alias unless the alias is already a UUID, so reruns are idempotent.

```bash
cd multica-auth-work

ENROLL_TENANT_ID=<tenant-uuid> \
ENROLL_CREDS_EXT4_BASE=/home/dataops-lab/multica-auth-creds \
bash scripts/staging/enroll_account.sh \
  codex \
  <profile-alias> \
  <priority-int> \
  <source-credential-path> \
  <tokens-per-window>
```

Expected behavior:

- source credential contents are copied into an isolated ext4-backed home;
- files are chmodded to `0600`, directories to `0700`;
- `scripts/staging/creds/<alias>` is a symlink to the ext4-backed home;
- `accounts` and active `credentials` rows are upserted in Postgres;
- output prints IDs, paths, status, mode, and sizes only.

## 5. Post-Enrollment Validation

Validate only scrubbed metadata:

```bash
docker exec -i multica-postgres-1 psql -U multica -d multica -v ON_ERROR_STOP=1 <<'SQL'
SELECT account_id, vendor, priority, status, home_dir, config_dir, tokens_per_window, tokens_used
FROM accounts
WHERE account_id = '<account-uuid>';

SELECT account_id, vendor, format, expires_at IS NULL AS active
FROM credentials
WHERE account_id = '<account-uuid>';
SQL
```

Then validate filesystem permissions through the canonical path:

```bash
stat -c '%a %n' multica-auth-work/scripts/staging/creds/<profile-alias>/<relative-credential-path>
```

Pass criteria:

- one approved account row exists for the tenant/provider/profile;
- exactly one active credential reference exists;
- no raw secret was printed;
- credential mode is `0600` for files;
- account status is `available` unless owner requested a safer initial state such as `degraded`;
- kill switch can disable this profile before its first production session.

## 6. Failure Handling

If enrollment fails, do not retry with modified secret paths until the cause is known. For partial rows, either rerun the same alias after fixing the filesystem/DB issue or expire the active credential reference with a scrubbed audit note. If any credential content was printed, treat it as a secret exposure and rotate the underlying provider credential before reuse.

## 7. Evidence Fields

Record under `.deploy-control/evidence/`:

```text
timestamp_utc
operator
tenant_id_hash_or_alias
provider
profile_alias
account_id
priority
credential_format
home_dir
filesystem_type
credential_mode
postgres_account_row_present
postgres_active_credential_present
kill_switch_reference
rollback_reference
secrets_present=false
remaining_risks
```
