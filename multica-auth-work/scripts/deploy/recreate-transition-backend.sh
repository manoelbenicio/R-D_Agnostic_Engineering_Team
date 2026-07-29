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
ADMISSION_HELD=0
ADMISSION_PID=""
ADMISSION_IN_FD=""
ADMISSION_OUT_FD=""
ROLLBACK_STATE=""
ROLLBACK_OVERRIDE=""
ROLL_FORWARD_CONFIG=""
RENDERED_CONFIG=""

ENV_FILE="/home/ec2-user/.config/multica-transition/dev.env"
IMAGES_FILE="/home/ec2-user/.config/multica-transition/images.yml"
BACKEND_ENV_OVERRIDE="/home/ec2-user/.config/multica-transition/backend-env.override.yml"
STATE_DIR="/home/ec2-user/.config/multica-transition/rollback"
LAST_KNOWN_GOOD_FILE="/home/ec2-user/.config/multica-transition/last-known-good.yml"
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
  LAST_KNOWN_GOOD_FILE="$TEST_ROOT/last-known-good.yml"
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

start_admission_freeze() {
  local marker
  ((ADMISSION_HELD == 0)) || die "admission freeze is already held"

  coproc ADMISSION_DB {
    # shellcheck disable=SC2016 # $1/$2 must expand inside the container child.
    compose exec -T postgres sh -c \
      'IFS= read -r PGPASSWORD; export PGPASSWORD; exec psql -X --no-psqlrc -qAt -v ON_ERROR_STOP=1 -h postgres -U "$1" -d "$2"' \
      sh "$EXPECTED_DB_USER" "$EXPECTED_DB_NAME"
  }
  ADMISSION_PID="$ADMISSION_DB_PID"
  ADMISSION_OUT_FD="${ADMISSION_DB[0]}"
  ADMISSION_IN_FD="${ADMISSION_DB[1]}"

  # The password travels only over the private stdin pipe. It is exported by
  # the container child immediately before execing psql; it never appears in
  # docker/psql argv, Compose flags, logs, or wrapper output.
  if ! read_env_value POSTGRES_PASSWORD >&"$ADMISSION_IN_FD"; then
    kill "$ADMISSION_PID" 2>/dev/null || true
    wait "$ADMISSION_PID" 2>/dev/null || true
    die "cannot provide authenticated database child environment"
  fi
  cat >&"$ADMISSION_IN_FD" <<'SQL'
BEGIN;
SET lock_timeout = '15s';
LOCK TABLE agent_task_queue IN SHARE MODE;
SELECT CASE WHEN EXISTS (
  SELECT 1 FROM agent_task_queue
  WHERE status IN ('queued','dispatched','running','waiting_local_directory')
) THEN 'ACTIVE_QUEUE_PRESENT' ELSE 'ADMISSION_FREEZE_HELD' END;
SQL

  if ! IFS= read -r -t 20 marker <&"$ADMISSION_OUT_FD"; then
    kill "$ADMISSION_PID" 2>/dev/null || true
    wait "$ADMISSION_PID" 2>/dev/null || true
    die "authenticated admission freeze failed before lock confirmation"
  fi
  ADMISSION_HELD=1
  if [[ "$marker" == "ACTIVE_QUEUE_PRESENT" ]]; then
    release_admission_freeze rollback || true
    die "disruptive recreate blocked: active task queue is not zero"
  fi
  [[ "$marker" == "ADMISSION_FREEZE_HELD" ]] || {
    release_admission_freeze rollback || true
    die "admission freeze returned an invalid content-free marker"
  }
  log "admission freeze held: task queue writes blocked and active queue is zero"
}

release_admission_freeze() {
  local action="${1:-rollback}" sql="ROLLBACK" rc=0
  ((ADMISSION_HELD == 1)) || return 0
  [[ "$action" == "commit" ]] && sql="COMMIT"
  printf '%s;\n\\q\n' "$sql" 1>&"$ADMISSION_IN_FD" 2>/dev/null || rc=1
  wait "$ADMISSION_PID" 2>/dev/null || rc=1
  # The coprocess has exited after \q; Bash closes its pipe endpoints. Avoid
  # invoking the special `exec` builtin solely for fd closure because that can
  # terminate non-interactive shells on some Bash versions.
  ADMISSION_IN_FD=""
  ADMISSION_OUT_FD=""
  ADMISSION_HELD=0
  ADMISSION_PID=""
  log "admission freeze released ($action)"
  return "$rc"
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
  CANDIDATE_IMAGE="$(python3 - "$RENDERED_CONFIG" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["services"]["backend"]["image"])
PY
)" || die "cannot read candidate image from rendered config"
  [[ "$CANDIDATE_IMAGE" =~ ^[A-Za-z0-9./:_@-]+$ ]] || die "rendered candidate image reference is unsafe"
}

read_backend_image_override() {
  local file="$1"
  python3 - "$file" <<'PY'
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
services_indent = backend_indent = None
values = []
for raw in path.read_text(encoding="utf-8").splitlines():
    clean = raw.split("#", 1)[0].rstrip()
    if not clean.strip():
        continue
    indent = len(clean) - len(clean.lstrip(" "))
    text = clean.strip()
    if text == "services:":
        services_indent = indent
        backend_indent = None
        continue
    if services_indent is not None and indent > services_indent and text == "backend:":
        backend_indent = indent
        continue
    if backend_indent is not None and indent <= backend_indent:
        backend_indent = None
    if backend_indent is not None and indent > backend_indent:
        match = re.fullmatch(r"image:\s*(\S.*?)\s*", text)
        if match:
            value = match.group(1)
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                value = value[1:-1]
            values.append(value)
if len(values) != 1:
    raise SystemExit("backend image override must appear exactly once")
print(values[0])
PY
}

write_last_known_good() {
  local image_ref="$1"
  [[ "$image_ref" =~ ^[A-Za-z0-9./:_@-]+$ ]] || die "unsafe image reference for last-known-good config"
  python3 - "$LAST_KNOWN_GOOD_FILE" "$image_ref" <<'PY'
import os, pathlib, sys, tempfile
path = pathlib.Path(sys.argv[1])
image = sys.argv[2]
path.parent.mkdir(parents=True, exist_ok=True)
payload = f"services:\n  backend:\n    image: {image}\n"
fd, tmp_name = tempfile.mkstemp(prefix=".last-known-good.", dir=path.parent)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as out:
        out.write(payload)
        out.flush()
        os.fsync(out.fileno())
    os.chmod(tmp_name, 0o600)
    os.replace(tmp_name, path)
    dir_fd = os.open(path.parent, os.O_DIRECTORY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
except Exception:
    try:
        os.unlink(tmp_name)
    except FileNotFoundError:
        pass
    raise
PY
}

pin_images_file_image() {
  local image_ref="$1"
  [[ "$image_ref" =~ ^[A-Za-z0-9./:_@-]+$ ]] || return 1
  python3 - "$IMAGES_FILE" "$image_ref" <<'PY'
import os, pathlib, re, stat, sys, tempfile
path = pathlib.Path(sys.argv[1])
image = sys.argv[2]
lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
services_indent = backend_indent = None
indexes = []
for index, raw in enumerate(lines):
    clean = raw.split("#", 1)[0].rstrip()
    if not clean.strip():
        continue
    indent = len(clean) - len(clean.lstrip(" "))
    text = clean.strip()
    if text == "services:":
        services_indent = indent
        backend_indent = None
        continue
    if services_indent is not None and indent > services_indent and text == "backend:":
        backend_indent = indent
        continue
    if backend_indent is not None and indent <= backend_indent:
        backend_indent = None
    if backend_indent is not None and indent > backend_indent and re.match(r"image:\s*", text):
        indexes.append((index, len(raw) - len(raw.lstrip(" "))))
if len(indexes) != 1:
    raise SystemExit("images.yml must contain exactly one services.backend.image")
index, indent = indexes[0]
ending = "\n" if lines[index].endswith("\n") else ""
lines[index] = " " * indent + "image: " + image + ending
mode = stat.S_IMODE(path.stat().st_mode)
fd, tmp_name = tempfile.mkstemp(prefix=".images.", dir=path.parent)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as out:
        out.writelines(lines)
        out.flush()
        os.fsync(out.fileno())
    os.chmod(tmp_name, mode)
    os.replace(tmp_name, path)
    dir_fd = os.open(path.parent, os.O_DIRECTORY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
except Exception:
    try:
        os.unlink(tmp_name)
    except FileNotFoundError:
        pass
    raise
PY
}

validate_last_known_good() {
  local durable_image
  if [[ ! -f "$LAST_KNOWN_GOOD_FILE" ]]; then
    log "last-known-good config absent; execute would initialize it from the running image"
    return 0
  fi
  durable_image="$(read_backend_image_override "$LAST_KNOWN_GOOD_FILE")" || die "last-known-good config is invalid"
  [[ "$durable_image" == "$PREVIOUS_IMAGE_REF" ]] || die "running backend image differs from durable last-known-good config"
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

assert_deployed_candidate() {
  local container deployed_ref
  container="$(compose ps -q backend)" || die "cannot identify recreated backend container"
  [[ -n "$container" ]] || die "recreated backend container is not running"
  deployed_ref="$(docker inspect --format '{{.Config.Image}}' "$container")" || die "cannot inspect recreated backend image"
  [[ "$deployed_ref" == "$CANDIDATE_IMAGE" ]] || die "healthy backend is not running the rendered candidate image"
}

write_rollback_state() {
  local stamp durable_image
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  mkdir -p "$STATE_DIR"
  chmod 700 "$STATE_DIR"
  ROLLBACK_STATE="$STATE_DIR/backend.${stamp}.state"
  ROLLBACK_OVERRIDE="$STATE_DIR/backend.${stamp}.rollback.yml"
  ROLL_FORWARD_CONFIG="$STATE_DIR/images.yml.${stamp}.roll-forward"

  # Archive the requested candidate for an explicit future roll-forward, but
  # keep the operational last-known-good file pinned to the running image.
  cp -p "$IMAGES_FILE" "$ROLL_FORWARD_CONFIG"
  if [[ ! -f "$LAST_KNOWN_GOOD_FILE" ]]; then
    write_last_known_good "$PREVIOUS_IMAGE_REF"
  fi
  durable_image="$(read_backend_image_override "$LAST_KNOWN_GOOD_FILE")" || die "last-known-good config is invalid"
  [[ "$durable_image" == "$PREVIOUS_IMAGE_REF" ]] || die "last-known-good config does not match the running image"
  cp -p "$LAST_KNOWN_GOOD_FILE" "$ROLLBACK_OVERRIDE"

  cat >"$ROLLBACK_STATE" <<EOF
project=$PROJECT_NAME
service=backend
previous_image_ref=$PREVIOUS_IMAGE_REF
previous_image_id=$PREVIOUS_IMAGE_ID
candidate_image_ref=$CANDIDATE_IMAGE
env_file=$ENV_FILE
last_known_good_config=$LAST_KNOWN_GOOD_FILE
roll_forward_config=$ROLL_FORWARD_CONFIG
rollback_override=$ROLLBACK_OVERRIDE
created_at=$stamp
EOF
  chmod 600 "$ROLL_FORWARD_CONFIG" "$ROLLBACK_OVERRIDE" "$ROLLBACK_STATE" "$LAST_KNOWN_GOOD_FILE"
}

rollback() {
  ((ROLLBACK_RUNNING == 0)) || return 1
  ROLLBACK_RUNNING=1
  log "deployment check failed; starting automatic rollback"
  if [[ -z "$ROLLBACK_OVERRIDE" || ! -f "$ROLLBACK_OVERRIDE" || ! -f "$LAST_KNOWN_GOOD_FILE" ]]; then
    log "ROLLBACK FAILED: durable last-known-good files are unavailable"
    return 1
  fi
  # Atomically repoint both durable LKG and the normal operational images.yml
  # before recreating. Any later ordinary Compose call therefore remains on
  # the known-good image; the failed candidate survives only in the explicit
  # roll-forward archive.
  if ! write_last_known_good "$PREVIOUS_IMAGE_REF"; then
    log "ROLLBACK FAILED: could not restore durable last-known-good config"
    return 1
  fi
  if ! pin_images_file_image "$PREVIOUS_IMAGE_REF"; then
    log "ROLLBACK FAILED: could not atomically restore images.yml"
    return 1
  fi
  if [[ "$(read_backend_image_override "$IMAGES_FILE" 2>/dev/null || true)" != "$PREVIOUS_IMAGE_REF" ]]; then
    log "ROLLBACK FAILED: images.yml did not verify as last-known-good"
    return 1
  fi
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
  if ((ADMISSION_HELD == 1)); then
    release_admission_freeze rollback || true
  fi
  exit "$rc"
}
trap on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

main() {
  parse_args "$@"
  reject_shell_overrides
  validate_files_and_identity
  render_and_assert_config

  local api_me_baseline
  api_me_baseline="$(curl_status "${BASE_URL}/api/me" 2>/dev/null || true)"
  [[ "$api_me_baseline" == "200" || "$api_me_baseline" == "401" ]] || die "preflight /api/me did not return the expected reachable/authenticated status"
  capture_current_image
  validate_last_known_good

  if ((DRY_RUN == 0)); then
    [[ "${DEPLOY_ALLOW_EXECUTE:-0}" == "1" ]] || die "execution blocked: set DEPLOY_ALLOW_EXECUTE=1 after an approved deploy window opens"
  fi

  # Queue-zero is checked only after the table lock is held. The same
  # transaction remains open through recreate, post-checks, or rollback.
  start_admission_freeze

  if ((DRY_RUN == 1)); then
    release_admission_freeze rollback || die "dry-run could not confirm admission freeze release"
    log "DRY-RUN PASS: live config identity, authenticated queue-zero under admission freeze, env-file label, pinned image, durable rollback posture, and /api/me baseline verified"
    log "DRY-RUN: no recreate or deployment mutation was performed"
    exit 0
  fi

  write_rollback_state
  MUTATED=1
  compose up -d --force-recreate --no-deps backend >/dev/null
  wait_for_status /health 200 60 || die "post-recreate /health failed"
  wait_for_status /readyz 200 30 || die "post-recreate /readyz failed"
  wait_for_status /api/me "$api_me_baseline" 10 || die "post-recreate /api/me status changed"
  assert_deployed_candidate

  # Promote while the critical section is still held. If promotion or release
  # fails, rollback rewrites both durable LKG and images.yml to the previous
  # image before releasing admission.
  write_last_known_good "$CANDIDATE_IMAGE"
  release_admission_freeze commit || die "could not confirm admission freeze release"
  printf 'deploy_status=healthy\nlast_known_good_promoted=true\n' >>"$ROLLBACK_STATE"
  MUTATED=0
  log "PASS: backend-only forced recreate is healthy; durable last-known-good is $CANDIDATE_IMAGE"
}

main "$@"
