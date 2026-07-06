#!/usr/bin/env bash
# Roll back Multica runtime launch config from prodex/L2 to raw Codex.

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="rollback-to-raw-codex"
ENV_FILE=""
CODEX_PATH="${MULTICA_CODEX_PATH:-codex}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: rollback-to-raw-codex.sh --env-file FILE --codex-path PATH [--dry-run|--execute]

Writes a rollback env file in one command:
  MULTICA_CODEX_PATH=<raw codex executable>
  MULTICA_PRODEX_ENABLED=0
  MULTICA_L2_ENABLED=0

Execution is gated by ROLLBACK_ALLOW_EXECUTE=1. For ROLLBACK_TARGET_ENV=prod,
DEPLOY_OWNER_APPROVED=true is also required.
USAGE
}

parse_args() {
  while (($#)); do
    case "$1" in
      --env-file)
        shift
        [[ $# -gt 0 ]] || die "--env-file requires a value"
        ENV_FILE="$1"
        ;;
      --codex-path)
        shift
        [[ $# -gt 0 ]] || die "--codex-path requires a value"
        CODEX_PATH="$1"
        ;;
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

require_execute_gate() {
  ((DRY_RUN == 1)) && return 0
  [[ "${ROLLBACK_ALLOW_EXECUTE:-0}" == "1" ]] || die "set ROLLBACK_ALLOW_EXECUTE=1 to execute"
  if [[ "${ROLLBACK_TARGET_ENV:-prod}" == "prod" && "${DEPLOY_OWNER_APPROVED:-false}" != "true" ]]; then
    die "prod execute blocked: DEPLOY_OWNER_APPROVED is not true"
  fi
}

main() {
  parse_args "$@"
  [[ -n "$ENV_FILE" ]] || die "--env-file is required"
  [[ -f "$ENV_FILE" ]] || die "env file not found: $ENV_FILE"
  command -v awk >/dev/null 2>&1 || die "awk is required"
  if [[ "$CODEX_PATH" != */* ]]; then
    CODEX_PATH="$(command -v "$CODEX_PATH" || true)"
  fi
  [[ -n "$CODEX_PATH" && -x "$CODEX_PATH" ]] || die "codex executable not found or not executable: ${CODEX_PATH:-empty}"
  require_execute_gate

  local rollback_id backup tmp
  rollback_id="rollback-$(date -u +%Y%m%dT%H%M%SZ)"
  backup="${ENV_FILE}.${rollback_id}.bak"

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would back up ${ENV_FILE} to ${backup}"
    log "DRY-RUN: would set MULTICA_CODEX_PATH=${CODEX_PATH}, MULTICA_PRODEX_ENABLED=0, MULTICA_L2_ENABLED=0"
    exit 0
  fi

  cp "$ENV_FILE" "$backup"
  tmp="$(mktemp)"
  awk -F= '
    $1=="MULTICA_CODEX_PATH" { next }
    $1=="MULTICA_PRODEX_ENABLED" { next }
    $1=="MULTICA_PRODEX_PATH" { next }
    $1=="MULTICA_PRODEX_VERSION" { next }
    $1=="MULTICA_PRODEX_COMMIT" { next }
    $1=="MULTICA_PRODEX_SMART_CONTEXT_SHADOW" { next }
    $1=="MULTICA_PRODEX_SMART_CONTEXT_CANARY_PERCENT" { next }
    $1=="MULTICA_PRODEX_KILL_SWITCH_DEFAULT_ON" { next }
    $1=="PRODEX_HOME" { next }
    $1=="MULTICA_L2_ENABLED" { next }
    $1=="MULTICA_L2_BASE_URL" { next }
    $1=="MULTICA_L2_BEARER_TOKEN" { next }
    $1=="MULTICA_L2_SIDECAR_ARGS" { next }
    $1=="MULTICA_ROLLBACK_ID" { next }
    { print }
  ' "$ENV_FILE" > "$tmp"
  {
    printf 'MULTICA_CODEX_PATH=%s\n' "$CODEX_PATH"
    printf 'MULTICA_PRODEX_ENABLED=0\n'
    printf 'MULTICA_L2_ENABLED=0\n'
    printf 'MULTICA_ROLLBACK_ID=%s\n' "$rollback_id"
  } >> "$tmp"
  mv "$tmp" "$ENV_FILE"
  log "PASS rollback_id=${rollback_id} backup=${backup}"
}

main "$@"
