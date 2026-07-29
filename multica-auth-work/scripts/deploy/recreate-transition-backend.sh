#!/usr/bin/env bash
# Approved fail-closed backend recreate path for the multica-dev-transition stack.

set -Eeuo pipefail
IFS=$'\n\t'
umask 077

SCRIPT_NAME="recreate-transition-backend"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DRY_RUN=1
MUTATED=0
ROLLBACK_RUNNING=0
ROLLBACK_BACKUP=""
ROLLBACK_STATE=""
ROLLBACK_OVERRIDE=""
RENDERED_CONFIG=""

ENV_FILE="/home/ec2-user/.config/multica-transition/dev.env"
IMAGES_FILE="/home/ec2-user/.config/multica-transition/images.yml"
BACKEND_ENV_OVERRIDE="/home/ec2-user/.config/multica-transition/backend-env.override.yml"
STATE_DIR="/home/ec2-user/.config/multica-transition/rollback"
PROJECT_NAME="multica-dev-transition"
EXPECTED_DB_USER="multica_transition"
EXPECTED_DB_NAME="multica_transition"
EXPECTED_BACKEND_PORT="18080"
BASE_URL="http://127.0.0.1:${EXPECTED_BACKEND_PORT}"

# Tests may redirect all mutable/config paths under /tmp. Test mode still uses
# command stubs and never permits a production path.
if [[ "${DEPLOY_WRAPPER_TEST_MODE:-0}" == "1" ]]; then
  TEST_ROOT="${DEPLOY_WRAPPER_TEST_ROOT:?DEPLOY_WRAPPER_TEST_ROOT is required in test mode}"
  TEST_ROOT="$(realpath -m "$TEST_ROOT")"
  [[ "$TEST_ROOT" == /tmp/* ]] || { echo "[$SCRIPT_NAME] ERROR: test root must be under /tmp" >&2; exit 1; }
  ENV_FILE="$TEST_ROOT/dev.env"
  IMAGES_FILE="$TEST_ROOT/images.yml"
  BACKEND_ENV_OVERRIDE="$TEST_ROOT/backend-env.override.yml"
  STATE_DIR="$TEST_ROOT/state"
fi

GUARD_FILE="$ROOT_DIR/deploy/multica-transition.guard.yml"
COMPOSE_FILES=(
  "$ROOT_DIR/docker-compose.selfhost.yml"
  "$ROOT_DIR/docker-compose.selfhost.build.yml"
  "$IMAGES_FILE"
  "$BACKEND_ENV_OVERRIDE"
  "$GUARD_FILE"
)

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: recreate-transition-backend.sh [--dry-run|--execute]

This is the only approved backend deploy/recreate path for
multica-dev-transition. Paths, project, service, and recreate flags are fixed.
Execution additionally requires DEPLOY_ALLOW_EXECUTE=1.
USAGE
}

parse_args() {
  while (($#)); do
    case "$1" in
      --dry-run) DRY_RUN=1 ;;
      --execute) DRY_RUN=0 ;;
      -h|--help) usage; exit 0 ;;
      *) die "unsupported argument: $1" ;;
    esac
    shift
  done
}

reject_shell_overrides() {
  local name
  while IFS= read -r name; do
    case "$name" in
      JWT_SECRET|POSTGRES_*|BACKEND_PORT|FRONTEND_ORIGIN|MULTICA_APP_URL|MULTICA_BACKEND_IMAGE|MULTICA_IMAGE_TAG|COMPOSE_FILE|COMPOSE_PROJECT_NAME|MULTICA_DEPLOY_ENV_FILE_LABEL)
        die "operator shell override is prohibited: $name (put the approved value in $ENV_FILE)"
        ;;
    esac
  done < <(compgen -e)
}

read_env_value() {
  local key="$1"
  python3 - "$ENV_FILE" "$key" <<'PY'
import pathlib, re, sys
path, wanted = pathlib.Path(sys.argv[1]), sys.argv[2]
values = []
for raw in path.read_text(encoding="utf-8").splitlines():
    line = raw.strip()
    if not line or line.startswith("#"):
        continue
    if line.startswith("export "):
        line = line[7:].lstrip()
    match = re.match(r"([A-Za-z_][A-Za-z0-9_]*)=(.*)$", line)
    if match and match.group(1) == wanted:
        value = match.group(2).strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
            value = value[1:-1]
        values.append(value)
if len(values) != 1 or not values[0]:
    raise SystemExit(f"{wanted} must appear exactly once and be non-empty")
print(values[0])
PY
}

compose() {
  local args=(docker compose --env-file "$ENV_FILE" -p "$PROJECT_NAME") file
  for file in "${COMPOSE_FILES[@]}"; do args+=( -f "$file" ); done
  MULTICA_DEPLOY_ENV_FILE_LABEL="$ENV_FILE" "${args[@]}" "$@"
}

compose_rollback() {
  local args=(docker compose --env-file "$ENV_FILE" -p "$PROJECT_NAME") file
  for file in "${COMPOSE_FILES[@]}"; do args+=( -f "$file" ); done
  args+=( -f "$ROLLBACK_OVERRIDE" )
  MULTICA_DEPLOY_ENV_FILE_LABEL="$ENV_FILE" "${args[@]}" "$@"
}

validate_files_and_identity() {
  local file
  command -v docker >/dev/null 2>&1 || die "docker is required"
  command -v curl >/dev/null 2>&1 || die "curl is required"
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  docker compose version >/dev/null 2>&1 || die "Docker Compose v2 is required"
  [[ -f "$ENV_FILE" && -r "$ENV_FILE" ]] || die "mandatory env file is missing or unreadable: $ENV_FILE"
  for file in "$IMAGES_FILE" "$BACKEND_ENV_OVERRIDE" "$GUARD_FILE"; do
    [[ -f "$file" && -r "$file" ]] || die "required compose file is missing or unreadable: $file"
  done

  local db_user db_name backend_port origin
  db_user="$(read_env_value POSTGRES_USER)" || die "cannot read POSTGRES_USER from mandatory env file"
  db_name="$(read_env_value POSTGRES_DB)" || die "cannot read POSTGRES_DB from mandatory env file"
  backend_port="$(read_env_value BACKEND_PORT)" || die "cannot read BACKEND_PORT from mandatory env file"
  origin="$(read_env_value FRONTEND_ORIGIN)" || die "cannot read FRONTEND_ORIGIN from mandatory env file"
  [[ "$db_user" == "$EXPECTED_DB_USER" ]] || die "env-file database user does not match the transition deployment identity"
  [[ "$db_name" == "$EXPECTED_DB_NAME" ]] || die "env-file database name does not match the transition deployment identity"
  [[ "$backend_port" == "$EXPECTED_BACKEND_PORT" ]] || die "env-file backend port must be $EXPECTED_BACKEND_PORT"
  [[ "$origin" =~ ^https?://[^/[:space:]]+(:[0-9]+)?$ ]] || die "FRONTEND_ORIGIN must be one absolute HTTP(S) origin"
  [[ "$origin" != "http://localhost:3000" ]] || die "transition deployment must not use the upstream fallback origin"
  EXPECTED_ORIGIN="$origin"
}

render_and_assert_config() {
  local render_err
  RENDERED_CONFIG="$(mktemp)"
  render_err="$(mktemp)"
  if ! compose config --format json >"$RENDERED_CONFIG" 2>"$render_err"; then
    rm -f "$render_err"
    die "compose render failed (rendered content withheld)"
  fi
  rm -f "$render_err"

  python3 - "$RENDERED_CONFIG" "$EXPECTED_DB_USER" "$EXPECTED_DB_NAME" "$EXPECTED_BACKEND_PORT" "$EXPECTED_ORIGIN" "$ENV_FILE" <<'PY'
import json, sys
from urllib.parse import urlparse

path, db_user, db_name, host_port, origin, env_file = sys.argv[1:]
try:
    data = json.load(open(path, encoding="utf-8"))
    services = data["services"]
    backend = services["backend"]
    postgres = services["postgres"]
    benv = backend["environment"]
    penv = postgres["environment"]
except Exception:
    raise SystemExit("rendered config assertion failed: required service shape")

def fail(category):
    raise SystemExit("rendered config assertion failed: " + category)

if str(penv.get("POSTGRES_USER", "")) != db_user or str(penv.get("POSTGRES_DB", "")) != db_name:
    fail("database identity")
parsed = urlparse(str(benv.get("DATABASE_URL", "")))
if parsed.scheme not in ("postgres", "postgresql") or parsed.username != db_user or parsed.hostname != "postgres" or parsed.port != 5432 or parsed.path.lstrip("/") != db_name:
    fail("backend database identity")
ports = backend.get("ports") or []
if not any(str(p.get("published")) == host_port and str(p.get("target")) == "8080" and str(p.get("host_ip")) == "127.0.0.1" for p in ports if isinstance(p, dict)):
    fail("backend host port")
if str(benv.get("FRONTEND_ORIGIN", "")) != origin:
    fail("frontend origin")
labels = backend.get("labels") or {}
if labels.get("com.multica.deploy.approved-wrapper") != "scripts/deploy/recreate-transition-backend.sh":
    fail("approved-wrapper label")
if labels.get("com.multica.deploy.env-file") != env_file:
    fail("env-file label")
image = str(backend.get("image", ""))
if not image or image.endswith(":latest"):
    fail("pinned backend image")
PY
}

assert_queue_zero() {
  local count
  count="$(compose exec -T postgres psql -U "$EXPECTED_DB_USER" -d "$EXPECTED_DB_NAME" -Atqc \
    "SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','dispatched','running','waiting_local_directory')")" \
    || die "active task queue preflight failed"
  count="${count//$'\r'/}"
  count="${count//$'\n'/}"
  [[ "$count" =~ ^[0-9]+$ ]] || die "active task queue preflight returned an invalid count"
  [[ "$count" == "0" ]] || die "disruptive recreate blocked: active task queue is not zero"
}

curl_status() {
  curl --silent --show-error --output /dev/null --max-time 5 --write-out '%{http_code}' "$1"
}

wait_for_status() {
  local path="$1" expected="$2" attempts="${3:-30}" status i
  if [[ "${DEPLOY_WRAPPER_TEST_MODE:-0}" == "1" && "$attempts" -gt 2 ]]; then attempts=2; fi
  for ((i=1; i<=attempts; i++)); do
    status="$(curl_status "${BASE_URL}${path}" 2>/dev/null || true)"
    [[ "$status" == "$expected" ]] && return 0
    [[ "${DEPLOY_WRAPPER_TEST_MODE:-0}" == "1" ]] || sleep 1
  done
  return 1
}

capture_current_image() {
  local container record
  container="$(compose ps -q backend)" || die "cannot identify current backend container"
  [[ -n "$container" ]] || die "backend container is not running"
  record="$(docker inspect --format '{{.Config.Image}}|{{.Image}}' "$container")" || die "cannot inspect current backend image"
  PREVIOUS_IMAGE_REF="${record%%|*}"
  PREVIOUS_IMAGE_ID="${record#*|}"
  [[ -n "$PREVIOUS_IMAGE_REF" && -n "$PREVIOUS_IMAGE_ID" && "$PREVIOUS_IMAGE_REF" != "$record" ]] || die "current backend image metadata is incomplete"
  [[ "$PREVIOUS_IMAGE_REF" =~ ^[A-Za-z0-9./:_@-]+$ ]] || die "current backend image reference is unsafe for rollback state"
  docker image inspect "$PREVIOUS_IMAGE_REF" >/dev/null 2>&1 || die "rollback image is not present locally"
}

write_rollback_state() {
  local stamp
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  mkdir -p "$STATE_DIR"
  chmod 700 "$STATE_DIR"
  ROLLBACK_BACKUP="$STATE_DIR/images.yml.${stamp}.bak"
  ROLLBACK_STATE="$STATE_DIR/backend.${stamp}.state"
  ROLLBACK_OVERRIDE="$STATE_DIR/backend.${stamp}.rollback.yml"
  cp -p "$IMAGES_FILE" "$ROLLBACK_BACKUP"
  cat >"$ROLLBACK_OVERRIDE" <<EOF
services:
  backend:
    image: $PREVIOUS_IMAGE_REF
EOF
  cat >"$ROLLBACK_STATE" <<EOF
project=$PROJECT_NAME
service=backend
previous_image_ref=$PREVIOUS_IMAGE_REF
previous_image_id=$PREVIOUS_IMAGE_ID
env_file=$ENV_FILE
images_backup=$ROLLBACK_BACKUP
rollback_override=$ROLLBACK_OVERRIDE
created_at=$stamp
EOF
  chmod 600 "$ROLLBACK_BACKUP" "$ROLLBACK_OVERRIDE" "$ROLLBACK_STATE"
}

rollback() {
  ((ROLLBACK_RUNNING == 0)) || return 1
  ROLLBACK_RUNNING=1
  log "deployment check failed; starting automatic rollback"
  if [[ -z "$ROLLBACK_BACKUP" || ! -f "$ROLLBACK_BACKUP" || -z "$ROLLBACK_OVERRIDE" || ! -f "$ROLLBACK_OVERRIDE" ]]; then
    log "ROLLBACK FAILED: preserved rollback files are unavailable"
    return 1
  fi
  cp -p "$ROLLBACK_BACKUP" "$IMAGES_FILE"
  if ! compose_rollback up -d --force-recreate --no-deps backend >/dev/null; then
    log "ROLLBACK FAILED: backend recreate failed"
    return 1
  fi
  if ! wait_for_status /health 200 30; then
    log "ROLLBACK FAILED: restored backend did not become healthy"
    return 1
  fi
  printf 'rollback_status=completed\n' >>"$ROLLBACK_STATE"
  log "automatic rollback completed using preserved image configuration"
  MUTATED=0
  return 0
}

on_exit() {
  local rc=$?
  [[ -z "$RENDERED_CONFIG" ]] || rm -f "$RENDERED_CONFIG"
  if ((rc != 0 && MUTATED == 1)); then
    rollback || true
  fi
  exit "$rc"
}
trap on_exit EXIT

main() {
  parse_args "$@"
  reject_shell_overrides
  validate_files_and_identity
  render_and_assert_config
  assert_queue_zero

  local api_me_baseline
  api_me_baseline="$(curl_status "${BASE_URL}/api/me" 2>/dev/null || true)"
  [[ "$api_me_baseline" == "200" || "$api_me_baseline" == "401" ]] || die "preflight /api/me did not return the expected reachable/authenticated status"
  capture_current_image

  if ((DRY_RUN == 1)); then
    log "DRY-RUN PASS: config identity, env-file label, pinned image, queue-zero, rollback image, and /api/me baseline verified"
    log "DRY-RUN: would preserve images.yml, force-recreate backend only, verify health/readiness and /api/me, then auto-rollback on failure"
    exit 0
  fi

  [[ "${DEPLOY_ALLOW_EXECUTE:-0}" == "1" ]] || die "execution blocked: set DEPLOY_ALLOW_EXECUTE=1 after an approved deploy window opens"
  write_rollback_state
  MUTATED=1
  compose up -d --force-recreate --no-deps backend >/dev/null
  wait_for_status /health 200 60 || die "post-recreate /health failed"
  wait_for_status /readyz 200 30 || die "post-recreate /readyz failed"
  wait_for_status /api/me "$api_me_baseline" 10 || die "post-recreate /api/me status changed"
  printf 'deploy_status=healthy\n' >>"$ROLLBACK_STATE"
  MUTATED=0
  log "PASS: backend-only forced recreate is healthy; rollback state preserved at $ROLLBACK_STATE"
}

main "$@"
