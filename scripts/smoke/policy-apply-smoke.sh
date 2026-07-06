#!/usr/bin/env bash
# Smoke: apply a minimal rpp.l2.v1 policy envelope in shadow mode.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="policy-apply-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
POLICY_ID="${SMOKE_POLICY_ID:-policy-smoke-shadow}"
REVISION="${SMOKE_POLICY_REVISION:-1}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: policy-apply-smoke.sh [--dry-run|--execute] [--base-url URL] [--tenant-id ID]

Applies a safe policy shape: Smart Context shadow, canary 0, auto-redeem disabled.
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
      --tenant-id)
        shift
        [[ $# -gt 0 ]] || die "--tenant-id requires a value"
        TENANT_ID="$1"
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
  [[ "$REVISION" =~ ^[1-9][0-9]*$ ]] || die "SMOKE_POLICY_REVISION must be a positive integer"
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
  python3 - "$TENANT_ID" "$POLICY_ID" "$REVISION" <<'PY'
import json
import sys

tenant_id, policy_id, revision = sys.argv[1], sys.argv[2], int(sys.argv[3])
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_{policy_id}_smoke",
    "tenant_id": tenant_id,
    "policy_id": policy_id,
    "revision": revision,
    "allowed_providers": ["codex"],
    "allowed_profiles": ["codex-smoke-main", "codex-smoke-backup"],
    "budgets": {
        "max_requests_per_session": 3,
        "max_estimated_input_tokens_per_request": 180000,
        "max_redeem_attempts_per_profile_per_day": 0
    },
    "smart_context": {
        "mode": "shadow",
        "canary_percent": 0,
        "exact_mode_allowed": True
    },
    "auto_redeem": {
        "enabled": False,
        "cooldown_seconds": 86400
    },
    "gateway": {
        "enabled": True,
        "adaptive_routing": "shadow"
    },
    "provider_capabilities": [{
        "provider": "codex",
        "launch_mode": "native_cli",
        "auth_mode": "oauth_profile",
        "quota_mode": "codex_usage",
        "rotation_mode": "profile_pool",
        "continuation_mode": "response_id",
        "smart_context_mode": "proxy_rewrite",
        "reset_claim_mode": "codex_redeem",
        "validation_status": "verified"
    }],
    "kill_switches": []
}, separators=(",", ":")))
PY
}

curl_json() {
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "${BASE_URL%/}/v1/policy/apply" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @-
}

validate_response() {
  python3 -c '
import json
import sys

policy_id = sys.argv[1]
revision = int(sys.argv[2])
data = json.load(sys.stdin)
errors = []
if data.get("contract_version") != "rpp.l2.v1":
    errors.append("contract_version != rpp.l2.v1")
if data.get("policy_id") != policy_id:
    errors.append("policy_id mismatch")
if data.get("revision") != revision:
    errors.append("revision mismatch")
if data.get("applied") is not True:
    errors.append("applied is not true")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
  ' "$POLICY_ID" "$REVISION"
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would POST ${BASE_URL%/}/v1/policy/apply"
    log "DRY-RUN: policy uses Smart Context shadow, canary 0, auto-redeem disabled"
    exit 0
  fi

  payload | curl_json | validate_response
  log "PASS"
}

main "$@"
