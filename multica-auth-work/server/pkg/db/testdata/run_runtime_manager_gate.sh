#!/usr/bin/env bash
set -euo pipefail

# Mandatory self-contained real-PostgreSQL gate for the SPE-6 runtime-manager
# schema and generated methods. All state lives on /tmp and is removed on exit.

server_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
gate_tmp=$(mktemp -d /tmp/spe6-runtime-manager.XXXXXX)
gate_data="$gate_tmp/data"
gate_socket="$gate_tmp/socket"
gate_log="$gate_tmp/postgres.log"

export GOCACHE=${GOCACHE:-/tmp/multica-w6-p1-gocache}
export GOMODCACHE=${GOMODCACHE:-/tmp/multica-shared-gomodcache}

cleanup() {
    if [[ -s "$gate_data/postmaster.pid" ]]; then
        pg_ctl -D "$gate_data" -m immediate -w stop >/dev/null 2>&1 || true
    fi
    case "$gate_tmp" in
        /tmp/spe6-runtime-manager.*) rm -r -- "$gate_tmp" ;;
        *) printf 'refusing unsafe temporary cleanup\n' >&2; return 1 ;;
    esac
}
trap cleanup EXIT

mkdir -m 0700 "$gate_socket"
initdb -D "$gate_data" -A trust -U postgres --no-locale --encoding=UTF8 >/dev/null
pg_ctl -D "$gate_data" -l "$gate_log" \
    -o "-F -h '' -k $gate_socket" -w start >/dev/null

database_url() {
    printf 'postgres://postgres@/%s?host=%s&sslmode=disable' "$1" "$gate_socket"
}

createdb -h "$gate_socket" -U postgres runtime_manager_clean
clean_url=$(database_url runtime_manager_clean)

(
    cd "$server_root"
    DATABASE_URL="$clean_url" go run ./cmd/migrate up
)

# Targeted reversible 134 -> 132 rollback and 132 -> 134 upgrade. Migration
# 131 is an intentional irreversible boundary: its down file is tested
# separately as a required refusal and is never crossed by this gate. Protected
# account/runtime rows must remain byte-for-byte equal across the reversible
# Runtime Manager suffix.
(
cd "$server_root"
psql "$clean_url" -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO "user" (id, name, email)
VALUES ('61000000-0000-4000-8000-000000000001', 'rollback owner', 'rollback@example.invalid');
INSERT INTO workspace (id, name, slug)
VALUES ('61000000-0000-4000-8000-000000000002', 'rollback workspace', 'rollback-workspace');
INSERT INTO agent_runtime (
    id, workspace_id, daemon_id, name, runtime_mode, provider, owner_id
) VALUES (
    '61000000-0000-4000-8000-000000000003',
    '61000000-0000-4000-8000-000000000002', 'rollback-daemon',
    'rollback runtime', 'local', 'codex',
    '61000000-0000-4000-8000-000000000001'
);
INSERT INTO accounts (account_id, vendor, tenant_id)
VALUES (
    '61000000-0000-4000-8000-000000000004', 'codex',
    '61000000-0000-4000-8000-000000000002'
);
CREATE TEMP TABLE protected_before AS
SELECT (SELECT to_jsonb(a) FROM accounts a
        WHERE account_id = '61000000-0000-4000-8000-000000000004') AS account_row,
       (SELECT to_jsonb(r) FROM agent_runtime r
        WHERE id = '61000000-0000-4000-8000-000000000003') AS runtime_row;
\ir migrations/134_runtime_configuration_snapshots.down.sql
DELETE FROM schema_migrations WHERE version = '134_runtime_configuration_snapshots';
\ir migrations/133_runtime_bindings.down.sql
DELETE FROM schema_migrations WHERE version = '133_runtime_bindings';
\ir migrations/132_credential_home_catalog.down.sql
DELETE FROM schema_migrations WHERE version = '132_credential_home_catalog';
\ir migrations/132_credential_home_catalog.up.sql
INSERT INTO schema_migrations(version) VALUES ('132_credential_home_catalog');
\ir migrations/133_runtime_bindings.up.sql
INSERT INTO schema_migrations(version) VALUES ('133_runtime_bindings');
\ir migrations/134_runtime_configuration_snapshots.up.sql
INSERT INTO schema_migrations(version) VALUES ('134_runtime_configuration_snapshots');
DO $$
BEGIN
    IF (SELECT account_row FROM protected_before) IS DISTINCT FROM
       (SELECT to_jsonb(a) FROM accounts a
        WHERE account_id = '61000000-0000-4000-8000-000000000004')
       OR (SELECT runtime_row FROM protected_before) IS DISTINCT FROM
       (SELECT to_jsonb(r) FROM agent_runtime r
        WHERE id = '61000000-0000-4000-8000-000000000003') THEN
        RAISE EXCEPTION 'protected rows changed across runtime-manager down/up';
    END IF;
END
$$;
SQL
)

migration_131_down_log="$gate_tmp/migration-131-down.log"
if (
    cd "$server_root"
    psql "$clean_url" -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
        -f migrations/131_runtime_sessions.down.sql
) >"$migration_131_down_log" 2>&1; then
    printf 'migration 131 down unexpectedly succeeded\n' >&2
    exit 1
fi
if ! grep -Fq '55000' "$migration_131_down_log" ||
   ! grep -Fq 'migration 131 is non-destructive and cannot be rolled down' "$migration_131_down_log"; then
    cat "$migration_131_down_log" >&2
    printf 'migration 131 down did not return the required irreversible-boundary refusal\n' >&2
    exit 1
fi
printf 'runtime_manager_migration_131_irreversible=PASS\n'

(
    cd "$server_root"
    SPE6_RUNTIME_MANAGER_DATABASE_URL="$clean_url" \
        go test ./pkg/db/generated -run '^TestRuntimeManagerReservationPrimitives$' -count=1
)

printf 'runtime_manager_postgresql_gate=PASS\n'
