#!/usr/bin/env bash
# Smoke harness: measure Smart Context request/response size and token deltas.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="smart-context-measure"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-12}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
WORKSPACE_ID="${SMOKE_WORKSPACE_ID:-workspace-smoke}"
TASK_ID="${SMOKE_TASK_ID:-task-smart-context-measure}"
SESSION_ID="${SMOKE_SESSION_ID:-session-smart-context-measure}"
POLICY_ID="${SMOKE_POLICY_ID:-policy-smoke-shadow}"
WORKING_DIRECTORY="${SMOKE_WORKING_DIRECTORY:-/tmp/rpp-smoke-workspace}"
PROFILE_MAIN="${SMOKE_PROFILE_MAIN:-codex-smoke-main}"
PROFILE_BACKUP="${SMOKE_PROFILE_BACKUP:-codex-smoke-backup}"
REQUEST_KIB="${SMOKE_CONTEXT_KIB:-256}"
DRY_RUN=1

TMP_DIR=""
START_PAYLOAD_FILE=""
START_RESPONSE_FILE=""
PAYLOAD_FILE=""
RESPONSE_FILE=""
RUNTIME_PATH=""
RUNTIME_SESSION_ID=""

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: smart-context-measure.sh [--dry-run|--execute] [--base-url URL] [--context-kib N] [--session-id ID]

Creates a runtime session with /v1/session/start, then sends a large Responses
API payload through the returned /v1/runtime/proxy endpoint and emits JSON metrics:
REQUEST_BYTES, RESPONSE_BYTES, CONTEXT_TOKENS_BEFORE, CONTEXT_TOKENS_AFTER,
compression_ratio, tokens_saved, and fallback_triggered.

Execution is gated by SMOKE_ALLOW_EXECUTE=1 and, for SMOKE_TARGET_ENV=prod,
DEPLOY_OWNER_APPROVED=true. The response body is captured in a temporary file
for measurement, but the script prints only scrubbed metrics.
USAGE
}

cleanup() {
  if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
    rm -f "$START_PAYLOAD_FILE" "$START_RESPONSE_FILE" "$PAYLOAD_FILE" "$RESPONSE_FILE"
    rmdir "$TMP_DIR" 2>/dev/null || true
  fi
}
trap cleanup EXIT

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
      --context-kib)
        shift
        [[ $# -gt 0 ]] || die "--context-kib requires a value"
        REQUEST_KIB="$1"
        ;;
      --session-id)
        shift
        [[ $# -gt 0 ]] || die "--session-id requires a value"
        SESSION_ID="$1"
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
  [[ "$REQUEST_KIB" =~ ^[1-9][0-9]*$ ]] || die "--context-kib must be a positive integer"
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
  command -v python3 >/dev/null 2>&1 || die "python3 is required for JSON metrics"
  command -v wc >/dev/null 2>&1 || die "wc is required"
  [[ -n "${!TOKEN_ENV-}" ]] || die "bearer token env var is empty: $TOKEN_ENV"
}

make_start_payload() {
  python3 - "$TENANT_ID" "$WORKSPACE_ID" "$TASK_ID" "$SESSION_ID" "$POLICY_ID" \
    "$WORKING_DIRECTORY" "$PROFILE_MAIN" "$PROFILE_BACKUP" <<'PY'
import json
import sys

(
    tenant_id,
    workspace_id,
    task_id,
    session_id,
    policy_id,
    working_directory,
    profile_main,
    profile_backup,
) = sys.argv[1:9]

payload = {
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_measure_start_{session_id}",
    "tenant_id": tenant_id,
    "workspace_id": workspace_id,
    "task_id": task_id,
    "session_id": session_id,
    "policy_id": policy_id,
    "requested_provider": "codex",
    "requested_model": "gpt-5",
    "working_directory": working_directory,
    "profile_pool": [profile_main, profile_backup],
    "continuation": {
        "previous_response_id": None,
        "session_binding_hint": None,
    },
}
print(json.dumps(payload, separators=(",", ":")))
PY
}

make_payload() {
  python3 - "$TENANT_ID" "$SESSION_ID" "$REQUEST_KIB" <<'PY'
import json
import math
import sys

tenant_id, session_id, request_kib = sys.argv[1:4]

target_bytes = int(request_kib) * 1024
unit = (
    "SMART_CONTEXT_MEASUREMENT_BLOCK\n"
    "retain identifiers: tenant_id workspace_id session_id previous_response_id tool_call_id\n"
    "compress repetitive diagnostics and repeated build output, but preserve JSON/control fields exactly.\n"
)
context = (unit * math.ceil(target_bytes / len(unit)))[:target_bytes]

payload = {
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_measure_runtime_{session_id}",
    "tenant_id": tenant_id,
    "session_id": session_id,
    "runtime_request_id": f"rt_req_measure_{session_id}",
    "body": {
        "model": "gpt-4.1",
        "instructions": (
            "system prompt: preserve identifiers exactly, keep JSON/control fields exact, "
            "and answer with a short confirmation."
        ),
        "input": [
            {
                "role": "user",
                "content": context,
            }
        ],
        "max_output_tokens": 1,
    },
}
print(json.dumps(payload, separators=(",", ":")))
PY
}

post_start_session() {
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "${BASE_URL%/}/v1/session/start" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @"$START_PAYLOAD_FILE" \
    -o "$START_RESPONSE_FILE"
}

runtime_session_id_from_start_response() {
  python3 - "$START_RESPONSE_FILE" "$BASE_URL" "$SESSION_ID" <<'PY'
import json
import sys
from urllib.parse import parse_qs, urlsplit

response_path, base_url, session_id = sys.argv[1:4]
with open(response_path, "rb") as f:
    response = json.loads(f.read())
endpoint = response.get("runtime_endpoint")
if isinstance(endpoint, str) and endpoint:
    parsed = urlsplit(endpoint)
    values = parse_qs(parsed.query).get("session_id") or []
    if values and values[0]:
        print(values[0])
        raise SystemExit(0)
response_session_id = response.get("session_id")
if isinstance(response_session_id, str) and response_session_id:
    print(response_session_id)
else:
    print(session_id)
PY
}

post_runtime_proxy() {
  local token="${!TOKEN_ENV}"
  curl -fsS --max-time "$TIMEOUT_SECONDS" \
    -X POST "${BASE_URL%/}${RUNTIME_PATH}" \
    -H "Authorization: Bearer ${token}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @"$PAYLOAD_FILE" \
    -o "$RESPONSE_FILE"
}

emit_metrics() {
  local request_bytes="$1"
  local response_bytes="$2"
  python3 - "$PAYLOAD_FILE" "$RESPONSE_FILE" "$request_bytes" "$response_bytes" "$BASE_URL" "$SESSION_ID" "$RUNTIME_PATH" <<'PY'
import json
import math
import sys

payload_path, response_path, request_bytes, response_bytes, base_url, session_id, runtime_path = sys.argv[1:8]
request_bytes = int(request_bytes)
response_bytes = int(response_bytes)

with open(payload_path, "rb") as f:
    payload = json.loads(f.read())
with open(response_path, "rb") as f:
    response_raw = f.read()

try:
    response = json.loads(response_raw)
except json.JSONDecodeError as exc:
    raise SystemExit(f"response is not JSON: {exc}")

def get_path(obj, dotted):
    cur = obj
    for part in dotted.split("."):
        if not isinstance(cur, dict) or part not in cur:
            return None
        cur = cur[part]
    return cur

def first_int(obj, paths):
    for path in paths:
        value = get_path(obj, path)
        if isinstance(value, bool):
            continue
        if isinstance(value, int):
            return value, path
        if isinstance(value, float) and value.is_integer():
            return int(value), path
        if isinstance(value, str) and value.isdigit():
            return int(value), path
    return None, None

def first_bool(obj, paths):
    for path in paths:
        value = get_path(obj, path)
        if isinstance(value, bool):
            return value, path
        if isinstance(value, str) and value.lower() in {"true", "false"}:
            return value.lower() == "true", path
    return None, None

before_paths = [
    "context_tokens_before",
    "smart_context.context_tokens_before",
    "smart_context.tokens_before",
    "smart_context.estimated_input_tokens_before",
    "smart_context.input_tokens_before_estimate",
    "metrics.context_tokens_before",
    "metrics.smart_context_tokens_before",
    "usage.context_tokens_before",
]
after_paths = [
    "context_tokens_after",
    "smart_context.context_tokens_after",
    "smart_context.tokens_after",
    "smart_context.estimated_input_tokens_after",
    "smart_context.input_tokens_after_observed_or_estimate",
    "smart_context.compressed_tokens",
    "metrics.context_tokens_after",
    "metrics.smart_context_tokens_after",
    "usage.context_tokens_after",
]
fallback_paths = [
    "fallback_triggered",
    "smart_context.fallback_triggered",
    "smart_context.exact_fallback_triggered",
    "metrics.fallback_triggered",
]

local_before = math.ceil(request_bytes / 4)

context_tokens_before, before_source = first_int(response, before_paths)
if context_tokens_before is None:
    context_tokens_before = local_before
    before_source = "local_request_bytes_estimate"

context_tokens_after, after_source = first_int(response, after_paths)
smart_context_mode = response.get("smart_context_mode") or get_path(response, "smart_context.mode")
if context_tokens_after is None:
    context_tokens_after = context_tokens_before
    if smart_context_mode == "exact":
        after_source = "inferred_exact_pass_through"
    elif smart_context_mode == "shadow":
        after_source = "inferred_shadow_no_active_rewrite"
    else:
        after_source = "inferred_no_response_token_metric"

fallback_triggered, fallback_source = first_bool(response, fallback_paths)
if fallback_triggered is None:
    fallback_triggered = smart_context_mode == "exact"
    fallback_source = "inferred_from_smart_context_mode" if smart_context_mode else "default_false_no_response_field"

if context_tokens_after > context_tokens_before:
    tokens_saved = 0
else:
    tokens_saved = context_tokens_before - context_tokens_after

compression_ratio = (
    round(context_tokens_after / context_tokens_before, 6)
    if context_tokens_before > 0 else None
)

errors = []
if response.get("contract_version") != "rpp.l2.v1":
    errors.append("contract_version != rpp.l2.v1")
if response.get("router_owner") != "rust_l2":
    errors.append("router_owner != rust_l2")
if not response.get("runtime_session_id"):
    errors.append("runtime_session_id missing")
if errors:
    raise SystemExit("; ".join(errors))

print(json.dumps({
    "REQUEST_BYTES": request_bytes,
    "RESPONSE_BYTES": response_bytes,
    "CONTEXT_TOKENS_BEFORE": context_tokens_before,
    "CONTEXT_TOKENS_AFTER": context_tokens_after,
    "compression_ratio": compression_ratio,
    "tokens_saved": tokens_saved,
    "fallback_triggered": fallback_triggered,
    "metric_sources": {
        "CONTEXT_TOKENS_BEFORE": before_source,
        "CONTEXT_TOKENS_AFTER": after_source,
        "fallback_triggered": fallback_source,
    },
    "response_summary": {
        "contract_version": response.get("contract_version"),
        "router_owner": response.get("router_owner"),
        "runtime_session_id_present": bool(response.get("runtime_session_id")),
        "event_stream_url_present": bool(response.get("event_stream_url")),
        "smart_context_mode": smart_context_mode,
    },
    "target": {
        "base_url": base_url,
        "path": runtime_path,
        "session_id": session_id,
    },
}, sort_keys=True))
PY
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would POST lifecycle payload to ${BASE_URL%/}/v1/session/start"
    log "DRY-RUN: would POST Responses API runtime payload to returned /v1/runtime/proxy endpoint"
    log "DRY-RUN: would emit REQUEST_BYTES, RESPONSE_BYTES, CONTEXT_TOKENS_BEFORE, CONTEXT_TOKENS_AFTER, compression_ratio, tokens_saved, fallback_triggered"
    exit 0
  fi

  TMP_DIR="$(mktemp -d)"
  START_PAYLOAD_FILE="${TMP_DIR}/smart-context-session-start.json"
  START_RESPONSE_FILE="${TMP_DIR}/smart-context-session-start-response.json"
  PAYLOAD_FILE="${TMP_DIR}/smart-context-request.json"
  RESPONSE_FILE="${TMP_DIR}/smart-context-response.json"

  make_start_payload >"$START_PAYLOAD_FILE"
  post_start_session
  RUNTIME_SESSION_ID="$(runtime_session_id_from_start_response)"
  RUNTIME_PATH="/v1/runtime/proxy?session_id=${RUNTIME_SESSION_ID}"

  make_payload >"$PAYLOAD_FILE"
  post_runtime_proxy

  request_bytes="$(wc -c <"$PAYLOAD_FILE" | tr -d '[:space:]')"
  response_bytes="$(wc -c <"$RESPONSE_FILE" | tr -d '[:space:]')"
  emit_metrics "$request_bytes" "$response_bytes"
}

main "$@"
