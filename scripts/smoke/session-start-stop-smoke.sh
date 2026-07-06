#!/usr/bin/env bash
# Smoke: start then stop an rpp.l2.v1 runtime session.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="session-start-stop-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
WORKSPACE_ID="${SMOKE_WORKSPACE_ID:-workspace-smoke}"
TASK_ID="${SMOKE_TASK_ID:-task-smoke}"
SESSION_ID="${SMOKE_SESSION_ID:-session-smoke}"
POLICY_ID="${SMOKE_POLICY_ID:-policy-smoke-shadow}"
WORKING_DIRECTORY="${SMOKE_WORKING_DIRECTORY:-/tmp/rpp-smoke-workspace}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: session-start-stop-smoke.sh [--dry-run|--execute] [--base-url URL] [--session-id ID]

Validates StartSession returns router_owner=rust_l2, then sends StopSession.
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
      --session-id)
        shift
        [[ $# -gt 0 ]] || die "--session-id requires a value"
        SESSION_ID="$1"
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

start_payload() {
  python3 - "$TENANT_ID" "$WORKSPACE_ID" "$TASK_ID" "$SESSION_ID" "$POLICY_ID" "$WORKING_DIRECTORY" <<'PY'
import json
import sys

tenant_id, workspace_id, task_id, session_id, policy_id, working_directory = sys.argv[1:7]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_start_{session_id}",
    "tenant_id": tenant_id,
    "workspace_id": workspace_id,
    "task_id": task_id,
    "session_id": session_id,
    "policy_id": policy_id,
    "requested_provider": "codex",
    "requested_model": "gpt-5",
    "working_directory": working_directory,
    "profile_pool": ["codex-smoke-main", "codex-smoke-backup"],
    "continuation": {
        "previous_response_id": None,
        "session_binding_hint": None
    }
}, separators=(",", ":")))
PY
}

stop_payload() {
  local runtime_session_id="$1"
  python3 - "$TENANT_ID" "$SESSION_ID" "$runtime_session_id" <<'PY'
import json
import sys

tenant_id, session_id, runtime_session_id = sys.argv[1:4]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_stop_{session_id}",
    "tenant_id": tenant_id,
    "session_id": session_id,
    "runtime_session_id": runtime_session_id,
    "reason": "operator_requested"
}, separators=(",", ":")))
PY
}

post_json() {
  local path="$1"
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "${BASE_URL%/}${path}" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @-
}

extract_runtime_session_id() {
  python3 -c '
import json
import sys

data = json.load(sys.stdin)
errors = []
if data.get("contract_version") != "rpp.l2.v1":
    errors.append("contract_version != rpp.l2.v1")
if data.get("router_owner") != "rust_l2":
    errors.append("router_owner != rust_l2")
runtime_session_id = data.get("runtime_session_id")
if not runtime_session_id:
    errors.append("runtime_session_id missing")
if not data.get("event_stream_url"):
    errors.append("event_stream_url missing")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
print(runtime_session_id)
'
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would POST ${BASE_URL%/}/v1/session/start and validate router_owner=rust_l2"
    log "DRY-RUN: would POST ${BASE_URL%/}/v1/session/stop with reason=operator_requested"
    exit 0
  fi

  runtime_session_id="$(start_payload | post_json /v1/session/start | extract_runtime_session_id)"
  stop_payload "$runtime_session_id" | post_json /v1/session/stop >/dev/null
  log "PASS"
}

main "$@"
