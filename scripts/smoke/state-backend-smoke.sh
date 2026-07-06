#!/usr/bin/env bash
# Smoke: assert the shared state backend is Postgres and FAIL if a shared
# SQLite backend is detected.
# Contract reference: docs/contracts/l2-runtime-contract.md (readyz §111: "no shared SQLite backend selected")

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="state-backend-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: state-backend-smoke.sh [--dry-run|--execute] [--base-url URL] [--timeout SECONDS]

Asserts the shared state backend is Postgres and FAILS if a shared SQLite backend is detected.

Dry-run is the default and prints the planned loopback request only.

Execute gates:
  SMOKE_ALLOW_EXECUTE=1 must be set.
  If SMOKE_TARGET_ENV=prod, DEPLOY_OWNER_APPROVED=true must also be set.
  L2_BEARER_TOKEN, or the env var named by L2_BEARER_TOKEN_ENV, must contain the bearer token.

This script never starts/restarts prodex and never performs deploy actions.
USAGE
}

parse_args() {
  while (($#)); do
    case "$1" in
      --dry-run) DRY_RUN=1 ;;
      --execute) DRY_RUN=0 ;;
      --base-url)
        shift
        [[ $# -gt 0 ]] || die "--base-url requires a value"
        BASE_URL="$1"
        ;;
      --timeout)
        shift
        [[ $# -gt 0 ]] || die "--timeout requires a value"
        TIMEOUT_SECONDS="$1"
        ;;
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
  [[ -r "$CONTRACT_FILE" ]] || die "contract not readable: $CONTRACT_FILE"
  [[ "$TIMEOUT_SECONDS" =~ ^[1-9][0-9]*$ ]] || die "--timeout must be a positive integer"
  case "$BASE_URL" in
    http://127.0.0.1:* | http://localhost:* | http://[::1]:*) ;;
    *) die "refusing non-loopback L2_BASE_URL: $BASE_URL" ;;
  esac
}

require_execute_gate() {
  ((DRY_RUN == 1)) && return 0
  [[ "${SMOKE_ALLOW_EXECUTE:-0}" == "1" ]] || die "set SMOKE_ALLOW_EXECUTE=1 to execute"
  if [[ "${SMOKE_TARGET_ENV:-prod}" == "prod" && "${DEPLOY_OWNER_APPROVED:-false}" != "true" ]]; then
    die "prod execute blocked: DEPLOY_OWNER_APPROVED is not true"
  fi
  command -v curl >/dev/null 2>&1 || die "curl is required"
  command -v python3 >/dev/null 2>&1 || die "python3 is required for JSON validation"
  [[ -n "${!TOKEN_ENV-}" ]] || die "bearer token env var is empty: $TOKEN_ENV"
}

curl_json() {
  local url="${BASE_URL%/}/readyz"
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/json" \
    "$url"
}

validate_state_backend() {
  python3 -c '
import json
import sys

data = json.load(sys.stdin)
errors = []

if data.get("contract_version") != "rpp.l2.v1":
    errors.append("contract_version != rpp.l2.v1")

if data.get("status") != "ready":
    errors.append("status != ready")

checks = data.get("checks")
if not isinstance(checks, list) or not checks:
    errors.append("checks must be a non-empty list")
else:
    failed = [c for c in checks if c.get("status") != "pass"]
    if failed:
        errors.append("one or more readiness checks did not pass")

    # Find shared_state_backend check
    backend_check = None
    for c in checks:
        if c.get("name") == "shared_state_backend":
            backend_check = c
            break

    if not backend_check:
        errors.append("shared_state_backend check missing")
    else:
        if backend_check.get("status") != "pass":
            errors.append("shared_state_backend check did not pass")

        details = backend_check.get("details", {})
        backend_type = details.get("backend_type") or details.get("type")

        if not backend_type:
            errors.append("shared_state_backend details missing backend_type")
        elif backend_type.lower() == "sqlite":
            errors.append("FORBIDDEN: shared SQLite backend detected")
        elif backend_type.lower() != "postgres" and backend_type.lower() != "postgresql":
            errors.append(f"unexpected backend_type: {backend_type} (expected postgres)")

if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)

print(f"backend_type={backend_type}", file=sys.stderr)
'
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would GET ${BASE_URL%/}/readyz"
    log "DRY-RUN: would validate contract_version=rpp.l2.v1, status=ready"
    log "DRY-RUN: would assert shared_state_backend check present and passing"
    log "DRY-RUN: would assert backend_type is postgres (not sqlite)"
    exit 0
  fi

  curl_json | validate_state_backend
  log "PASS"
}

main "$@"