#!/usr/bin/env bash
# Smoke: verify RegisterAccounts rejects an invalid/non-isolated profile home.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="profile-fail-closed-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
INVALID_PROFILE_HOME="${SMOKE_INVALID_PROFILE_HOME:-/tmp/rpp-smoke-outside-managed-root}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: profile-fail-closed-smoke.sh [--dry-run|--execute] [--base-url URL] [--invalid-profile-home PATH]

Sends an invalid profile reference to /v1/accounts/register. PASS means the runtime fails closed:
HTTP >= 400 or a 2xx response containing a non-empty rejected_profiles array.

Execution is gated by SMOKE_ALLOW_EXECUTE=1 and, for SMOKE_TARGET_ENV=prod, DEPLOY_OWNER_APPROVED=true.
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
      --invalid-profile-home)
        shift
        [[ $# -gt 0 ]] || die "--invalid-profile-home requires a value"
        INVALID_PROFILE_HOME="$1"
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
  [[ "$INVALID_PROFILE_HOME" = /* ]] || die "invalid profile home must be absolute"
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

payload() {
  python3 - "$TENANT_ID" "$INVALID_PROFILE_HOME" <<'PY'
import json
import sys

tenant_id, invalid_profile_home = sys.argv[1:3]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": "req_accounts_fail_closed_smoke",
    "tenant_id": tenant_id,
    "profiles": [{
        "profile_id": "invalid-outside-root-smoke",
        "provider": "codex",
        "profile_home": invalid_profile_home,
        "auth_mode": "oauth_profile",
        "status": "approved",
        "capability_ref": "codex.oauth_profile.v1"
    }]
}, separators=(",", ":")))
PY
}

post_accounts() {
  local token="${!TOKEN_ENV}"
  local tmp_body
  tmp_body="$(mktemp)"
  trap 'rm -f -- "$tmp_body"' RETURN

  local status
  status="$(curl -sS --max-time "$TIMEOUT_SECONDS" \
    -o "$tmp_body" \
    -w '%{http_code}' \
    -X POST "${BASE_URL%/}/v1/accounts/register" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @-)"

  printf '%s\n' "$status"
  cat -- "$tmp_body"
}

validate_fail_closed() {
  python3 -c '
import json
import sys

lines = sys.stdin.read().splitlines()
if not lines:
    print("missing HTTP status", file=sys.stderr)
    sys.exit(1)
try:
    status = int(lines[0])
except ValueError:
    print("invalid HTTP status", file=sys.stderr)
    sys.exit(1)
body = "\n".join(lines[1:]).strip()
if status >= 400:
    sys.exit(0)
if not body:
    print("2xx response without rejection body is fail-open", file=sys.stderr)
    sys.exit(1)
try:
    data = json.loads(body)
except json.JSONDecodeError as exc:
    print(f"2xx response body is not JSON: {exc}", file=sys.stderr)
    sys.exit(1)
if data.get("contract_version") != "rpp.l2.v1":
    print("contract_version != rpp.l2.v1", file=sys.stderr)
    sys.exit(1)
rejected = data.get("rejected_profiles")
if isinstance(rejected, list) and rejected:
    sys.exit(0)
print("invalid profile was not rejected", file=sys.stderr)
sys.exit(1)
'
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would POST ${BASE_URL%/}/v1/accounts/register with invalid profile_home=${INVALID_PROFILE_HOME}"
    log "DRY-RUN: PASS condition is HTTP >= 400 or rejected_profiles non-empty"
    exit 0
  fi

  payload | post_accounts | validate_fail_closed
  log "PASS"
}

main "$@"
