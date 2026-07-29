#!/usr/bin/env bash
# Real backend/PostgreSQL integration proof for the ORQ-57 admission lock.
# Creates only isolated temporary containers and never joins the product stack.

set -Eeuo pipefail
IFS=$'\n\t'
umask 077

SCRIPT_NAME="recreate-transition-integration"
CANDIDATE_IMAGE="${CANDIDATE_IMAGE:?CANDIDATE_IMAGE is required}"
ROLLBACK_IMAGE="${ROLLBACK_IMAGE:?ROLLBACK_IMAGE is required}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-pgvector/pgvector:pg17}"
[[ "${ORQ57_INTEGRATION_ALLOW:-0}" == "1" ]] || {
  echo "[$SCRIPT_NAME] ERROR: set ORQ57_INTEGRATION_ALLOW=1 for isolated integration containers" >&2
  exit 1
}

for ref in "$CANDIDATE_IMAGE" "$ROLLBACK_IMAGE" "$POSTGRES_IMAGE"; do
  [[ "$ref" =~ ^[A-Za-z0-9./:_@-]+$ ]] || {
    echo "[$SCRIPT_NAME] ERROR: unsafe image reference" >&2
    exit 1
  }
done

SUFFIX="${$}-$(date -u +%s)"
NETWORK="orq57-lock-${SUFFIX}"
POSTGRES_CONTAINER="orq57-pg-${SUFFIX}"
BOOTSTRAP_CONTAINER="orq57-bootstrap-${SUFFIX}"
CANDIDATE_CONTAINER="orq57-candidate-${SUFFIX}"
ROLLBACK_CONTAINER="orq57-rollback-${SUFFIX}"
DB_USER="orq57_integration"
DB_NAME="orq57_integration"
TMP_DIR="$(mktemp -d)"
LOCK_HELD=0
LOCK_PID=""
LOCK_IN_FD=""
LOCK_OUT_FD=""

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*"; }
die() { log "ERROR: $*" >&2; exit 1; }

release_lock() {
  ((LOCK_HELD == 1)) || return 0
  printf 'ROLLBACK;\n\\q\n' 1>&"$LOCK_IN_FD" 2>/dev/null || true
  wait "$LOCK_PID" 2>/dev/null || true
  LOCK_HELD=0
  LOCK_PID=""
  log "TRACE lock_released=PASS"
}

cleanup() {
  local rc=$?
  set +e
  release_lock
  docker rm -fv "$BOOTSTRAP_CONTAINER" "$CANDIDATE_CONTAINER" "$ROLLBACK_CONTAINER" "$POSTGRES_CONTAINER" >/dev/null 2>&1
  docker network rm "$NETWORK" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

command -v docker >/dev/null 2>&1 || die "docker is required"
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"
docker version >/dev/null 2>&1 || die "docker daemon is unavailable"
docker image inspect "$CANDIDATE_IMAGE" >/dev/null 2>&1 || die "candidate image is not present locally"
docker image inspect "$ROLLBACK_IMAGE" >/dev/null 2>&1 || die "rollback image is not present locally"
docker image inspect "$POSTGRES_IMAGE" >/dev/null 2>&1 || die "PostgreSQL image is not present locally"

DB_PASSWORD="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(32))
PY
)"
JWT_SECRET_VALUE="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(48))
PY
)"

cat >"$TMP_DIR/postgres.env" <<EOF
POSTGRES_USER=$DB_USER
POSTGRES_DB=$DB_NAME
POSTGRES_PASSWORD=$DB_PASSWORD
EOF
cat >"$TMP_DIR/backend.env" <<EOF
DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@postgres:5432/$DB_NAME?sslmode=disable
PORT=8080
APP_ENV=development
JWT_SECRET=$JWT_SECRET_VALUE
FRONTEND_ORIGIN=http://127.0.0.1
MULTICA_APP_URL=http://127.0.0.1
ALLOW_SIGNUP=false
EOF
chmod 600 "$TMP_DIR/postgres.env" "$TMP_DIR/backend.env"

docker network create "$NETWORK" >/dev/null
docker run -d --pull never \
  --name "$POSTGRES_CONTAINER" \
  --network "$NETWORK" \
  --network-alias postgres \
  --env-file "$TMP_DIR/postgres.env" \
  "$POSTGRES_IMAGE" >/dev/null

for _ in $(seq 1 60); do
  if docker exec "$POSTGRES_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
    log "TRACE postgres_ready=PASS"
    break
  fi
  sleep 1
done
docker exec "$POSTGRES_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1 || die "ephemeral PostgreSQL did not become ready"

run_psql() {
  local sql="$1"
  {
    printf '%s\n' "$DB_PASSWORD"
    printf '%s\n' "$sql"
  } | docker exec -i "$POSTGRES_CONTAINER" sh -c \
    'IFS= read -r PGPASSWORD; export PGPASSWORD; exec psql -X --no-psqlrc -qAt -v ON_ERROR_STOP=1 -h postgres -U "$1" -d "$2"' \
    sh "$DB_USER" "$DB_NAME"
}

start_backend() {
  local name="$1" image="$2"
  docker run -d --pull never \
    --name "$name" \
    --network "$NETWORK" \
    --env-file "$TMP_DIR/backend.env" \
    -p 127.0.0.1::8080 \
    "$image" >/dev/null
}

backend_port() {
  local name="$1" mapping
  mapping="$(docker port "$name" 8080/tcp | head -n1)"
  [[ "$mapping" =~ ^127\.0\.0\.1:([0-9]+)$ ]] || return 1
  printf '%s\n' "${BASH_REMATCH[1]}"
}

wait_backend() {
  local name="$1" path="$2" attempts="${3:-120}" port status
  port="$(backend_port "$name")" || return 1
  for _ in $(seq 1 "$attempts"); do
    status="$(curl --silent --show-error --output /dev/null --max-time 2 --write-out '%{http_code}' "http://127.0.0.1:${port}${path}" 2>/dev/null || true)"
    [[ "$status" == "200" ]] && return 0
    if [[ "$(docker inspect "$name" --format '{{.State.Running}}' 2>/dev/null || true)" != "true" ]]; then
      return 1
    fi
    sleep 1
  done
  return 1
}

# Bootstrap the real schema using the actual candidate entrypoint/migrator.
start_backend "$BOOTSTRAP_CONTAINER" "$CANDIDATE_IMAGE"
wait_backend "$BOOTSTRAP_CONTAINER" /readyz 180 || die "bootstrap backend did not reach readiness"
log "TRACE bootstrap_ready=PASS"
docker rm -f "$BOOTSTRAP_CONTAINER" >/dev/null

# Prove authenticated bridge-path SQL before acquiring the lock.
[[ "$(run_psql 'SELECT 1;')" == "1" ]] || die "authenticated bridge-path SQL preflight failed"
log "TRACE authenticated_bridge_sql=PASS"

coproc LOCK_DB {
  docker exec -i "$POSTGRES_CONTAINER" sh -c \
    'IFS= read -r PGPASSWORD; export PGPASSWORD; exec psql -X --no-psqlrc -qAt -v ON_ERROR_STOP=1 -h postgres -U "$1" -d "$2"' \
    sh "$DB_USER" "$DB_NAME"
}
LOCK_PID="$LOCK_DB_PID"
LOCK_OUT_FD="${LOCK_DB[0]}"
LOCK_IN_FD="${LOCK_DB[1]}"
printf '%s\n' "$DB_PASSWORD" 1>&"$LOCK_IN_FD"
cat 1>&"$LOCK_IN_FD" <<'SQL'
BEGIN;
SET lock_timeout = '15s';
LOCK TABLE agent_task_queue IN SHARE MODE;
SELECT 'INTEGRATION_LOCK_HELD';
SQL
IFS= read -r -t 20 marker <&"$LOCK_OUT_FD" || die "lock holder did not confirm"
[[ "$marker" == "INTEGRATION_LOCK_HELD" ]] || die "lock holder returned an invalid marker"
LOCK_HELD=1
log "TRACE admission_lock_held=PASS"

# UPDATE acquires ROW EXCLUSIVE even when no rows match; it must time out.
if run_psql "SET lock_timeout = '1s'; UPDATE agent_task_queue SET status = status WHERE false;" >/dev/null 2>&1; then
  die "queue writer unexpectedly passed while SHARE lock was held"
fi
log "TRACE queue_writer_blocked=PASS"

# Candidate startup includes the actual entrypoint and migrate-up path.
start_backend "$CANDIDATE_CONTAINER" "$CANDIDATE_IMAGE"
wait_backend "$CANDIDATE_CONTAINER" /readyz 180 || die "candidate backend did not reach /readyz under admission lock"
log "TRACE candidate_ready_under_lock=PASS"
docker rm -f "$CANDIDATE_CONTAINER" >/dev/null

# The rollback image must also start and become healthy under the same lock.
start_backend "$ROLLBACK_CONTAINER" "$ROLLBACK_IMAGE"
wait_backend "$ROLLBACK_CONTAINER" /health 180 || die "rollback backend did not reach /health under admission lock"
log "TRACE rollback_health_under_lock=PASS"
docker rm -f "$ROLLBACK_CONTAINER" >/dev/null

release_lock
run_psql "SET lock_timeout = '2s'; UPDATE agent_task_queue SET status = status WHERE false;" >/dev/null 
log "TRACE queue_writer_after_release=PASS"
log "PASS isolated actual-backend lock/startup/rollback integration"
