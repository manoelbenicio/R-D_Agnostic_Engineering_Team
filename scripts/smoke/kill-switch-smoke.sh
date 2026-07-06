#!/usr/bin/env bash
# Smoke: apply an rpp.l2.v1 kill switch for Smart Context.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="kill-switch-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
PROVIDER="${SMOKE_PROVIDER:-codex}"
PROFILE_ID="${SMOKE_PROFILE_ID:-codex-smoke-main}"
FEATURE="${SMOKE_KILL_FEATURE:-smart_context}"
EFFECTIVE_AT="${SMOKE_KILL_EFFECTIVE_AT:-next_request}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: kill-switch-smoke.sh [--dry-run|--execute] [--base-url URL] [--feature FEATURE]

Applies state=disabled for a feature key. Default feature is smart_context.
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
      --feature)
        shift
        [[ $# -gt 0 ]] || die "--feature requires a value"
        FEATURE="$1"
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
  case "$FEATURE" in
    runtime_proxy | gateway | smart_context | auto_redeem | provider_bridge) ;;
    *) die "invalid feature key for rpp.l2.v1: $FEATURE" ;;
  esac
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
  python3 - "$TENANT_ID" "$PROVIDER" "$PROFILE_ID" "$FEATURE" "$EFFECTIVE_AT" <<'PY'
import json
import sys

tenant_id, provider, profile_id, feature, effective_at = sys.argv[1:6]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_kill_{feature}_smoke",
    "tenant_id": tenant_id,
    "scope": {
        "provider": provider,
        "profile_id": profile_id,
        "session_id": None
    },
    "feature": feature,
    "state": "disabled",
    "reason": "operator_guardrail",
    "effective_at": effective_at
}, separators=(",", ":")))
PY
}

curl_json() {
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "${BASE_URL%/}/v1/killswitch/apply" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @-
}

validate_response() {
  python3 -c '
import json
import sys

requested_effective_at = sys.argv[1]
data = json.load(sys.stdin)
errors = []
if data.get("contract_version") != "rpp.l2.v1":
    errors.append("contract_version != rpp.l2.v1")
if data.get("applied") is not True:
    errors.append("applied is not true")
if data.get("effective_at") not in {requested_effective_at, "immediate", "next_request", "session_restart_required"}:
    errors.append("unexpected effective_at")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
  ' "$EFFECTIVE_AT"
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would POST ${BASE_URL%/}/v1/killswitch/apply feature=${FEATURE} state=disabled effective_at=${EFFECTIVE_AT}"
    exit 0
  fi

  payload | curl_json | validate_response
  log "PASS"
}

main "$@"
