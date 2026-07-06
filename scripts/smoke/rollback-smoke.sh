#!/usr/bin/env bash
# Smoke: validate rollback readiness without performing a live rollback.
# Contract reference: docs/deploy/rollback-runbook.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="rollback-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
RUNBOOK_FILE="${REPO_ROOT}/docs/deploy/rollback-runbook.md"
ROLLBACK_CMD="${REPO_ROOT}/scripts/deploy/rollback-to-raw-codex.sh"

DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: rollback-smoke.sh [--dry-run|--execute]

Dry-run is the default. It validates that the rollback runbook and dependent
smoke harnesses exist, then prints the planned F0-gated live rollback checks.

Execution is gated by SMOKE_ALLOW_EXECUTE=1 and DEPLOY_OWNER_APPROVED=true.
This script never starts, stops, restarts, or deploys prodex in dry-run mode.
USAGE
}

parse_args() {
  while (($#)); do
    case "$1" in
      --dry-run) DRY_RUN=1 ;;
      --execute) DRY_RUN=0 ;;
      -h | --help)
        usage
        exit 0
        ;;
      *) die "unknown argument: $1" ;;
    esac
    shift
  done
}

validate_common() {
  [[ -r "$RUNBOOK_FILE" ]] || die "rollback runbook not readable: $RUNBOOK_FILE"
  [[ -r "$ROLLBACK_CMD" ]] || die "rollback command not readable: $ROLLBACK_CMD"
  for required in \
    "Rollback Triggers" \
    "Rollback Controls" \
    "Rollback Steps" \
    "Rollback Success Criteria" \
    "Rollback Evidence"; do
    grep -F "$required" "$RUNBOOK_FILE" >/dev/null || die "runbook missing section: $required"
  done
  for smoke in \
    readyz-smoke.sh \
    kill-switch-smoke.sh \
    session-start-stop-smoke.sh \
    redaction-smoke.sh \
    state-backend-smoke.sh; do
    [[ -r "${SCRIPT_DIR}/${smoke}" ]] || die "dependent smoke harness missing: ${smoke}"
  done
}

require_execute_gate() {
  ((DRY_RUN == 1)) && return 0
  [[ "${SMOKE_ALLOW_EXECUTE:-0}" == "1" ]] || die "set SMOKE_ALLOW_EXECUTE=1 to execute"
  [[ "${DEPLOY_OWNER_APPROVED:-false}" == "true" ]] || die "execute blocked: DEPLOY_OWNER_APPROVED is not true"
}

write_fake_executable() {
  local path="$1"
  local label="$2"
  {
    printf '#!/usr/bin/env bash\n'
    printf 'printf "%%s\\n" %q\n' "${label}-smoke"
  } > "$path"
  chmod +x "$path"
}

assert_env_value() {
  local env_file="$1"
  local key="$2"
  local expected="$3"
  local got
  got="$(awk -F= -v key="$key" '$1 == key { value=$2 } END { print value }' "$env_file")"
  [[ "$got" == "$expected" ]] || die "${key}=${got}, want ${expected}"
}

execute_rollback_smoke() {
  local tmp_dir env_file fake_codex fake_prodex backup raw_output
  tmp_dir="$(mktemp -d)"
  env_file="${tmp_dir}/multica.env"
  fake_codex="${tmp_dir}/codex"
  fake_prodex="${tmp_dir}/prodex"
  write_fake_executable "$fake_codex" "codex"
  write_fake_executable "$fake_prodex" "prodex"

  {
    printf 'MULTICA_CODEX_PATH=%s\n' "$fake_prodex"
    printf 'MULTICA_PRODEX_ENABLED=1\n'
    printf 'MULTICA_PRODEX_PATH=%s\n' "$fake_prodex"
    printf 'MULTICA_PRODEX_VERSION=v0.246.0\n'
    printf 'MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144\n'
    printf 'PRODEX_HOME=%s\n' "${tmp_dir}/prodex-home"
    printf 'MULTICA_L2_ENABLED=1\n'
    printf 'MULTICA_L2_BASE_URL=http://127.0.0.1:43117\n'
    printf 'MULTICA_L2_BEARER_TOKEN=redacted-smoke-token\n'
  } > "$env_file"

  ROLLBACK_ALLOW_EXECUTE=1 ROLLBACK_TARGET_ENV=smoke \
    bash "$ROLLBACK_CMD" --env-file "$env_file" --codex-path "$fake_codex" --execute

  assert_env_value "$env_file" "MULTICA_CODEX_PATH" "$fake_codex"
  assert_env_value "$env_file" "MULTICA_PRODEX_ENABLED" "0"
  assert_env_value "$env_file" "MULTICA_L2_ENABLED" "0"
  if grep -Eq '^(MULTICA_PRODEX_PATH|PRODEX_HOME|MULTICA_L2_BASE_URL|MULTICA_L2_BEARER_TOKEN)=' "$env_file"; then
    die "rollback env still contains prodex/L2 routing keys"
  fi
  raw_output="$("$fake_codex" --version)"
  [[ "$raw_output" == "codex-smoke" ]] || die "raw codex smoke output=${raw_output}"

  backup="$(find "$tmp_dir" -name 'multica.env.rollback-*.bak' -print | sort | tail -n 1)"
  [[ -n "$backup" && -r "$backup" ]] || die "rollback backup not found"
  cp "$backup" "$env_file"
  assert_env_value "$env_file" "MULTICA_PRODEX_ENABLED" "1"
  assert_env_value "$env_file" "MULTICA_PRODEX_PATH" "$fake_prodex"
  rm -rf "$tmp_dir"
  log "PASS"
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: rollback runbook sections present"
    log "DRY-RUN: dependent smoke harnesses present"
    log "DRY-RUN: LIVE rollback execution remains F0-gated until DEPLOY_OWNER_APPROVED=true"
    log "DRY-RUN: would apply kill switches, verify sidecar health/readiness failure boundary, restore raw Codex path, run raw-session smoke, and preserve scrubbed evidence"
    exit 0
  fi

  execute_rollback_smoke
}

main "$@"
