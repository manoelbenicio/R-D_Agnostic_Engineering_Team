#!/usr/bin/env bash
# Smoke: read and validate rpp.l2.v1 newline-delimited runtime events.
# Contract reference: docs/contracts/l2-runtime-contract.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="event-stream-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
SESSION_ID="${SMOKE_SESSION_ID:-session-smoke}"
MIN_EVENTS="${SMOKE_MIN_EVENTS:-1}"
DRY_RUN=1

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: event-stream-smoke.sh [--dry-run|--execute] [--base-url URL] [--session-id ID] [--min-events N]

Reads NDJSON from /v1/events/stream and validates contract_version plus redaction.secrets_present=false.
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
      --min-events)
        shift
        [[ $# -gt 0 ]] || die "--min-events requires a value"
        MIN_EVENTS="$1"
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
  [[ "$MIN_EVENTS" =~ ^[0-9]+$ ]] || die "--min-events must be a non-negative integer"
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
  command -v python3 >/dev/null 2>&1 || die "python3 is required for NDJSON validation"
  [[ -n "${!TOKEN_ENV-}" ]] || die "bearer token env var is empty: $TOKEN_ENV"
}

stream_events() {
  local token="${!TOKEN_ENV}"
  curl -fsS --no-buffer --max-time "$TIMEOUT_SECONDS" \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/x-ndjson, application/json" \
    "${BASE_URL%/}/v1/events/stream?session_id=${SESSION_ID}"
}

validate_ndjson() {
  python3 -c '
import json
import sys

min_events = int(sys.argv[1])
count = 0
errors = []
for raw in sys.stdin:
    line = raw.strip()
    if not line:
        continue
    count += 1
    try:
        event = json.loads(line)
    except json.JSONDecodeError as exc:
        errors.append(f"line {count}: invalid JSON: {exc}")
        continue
    if event.get("contract_version") != "rpp.l2.v1":
        errors.append(f"line {count}: contract_version != rpp.l2.v1")
    redaction = event.get("redaction")
    if not isinstance(redaction, dict):
        errors.append(f"line {count}: redaction object is missing")
    elif redaction.get("secrets_present") is not False:
        errors.append(f"line {count}: redaction.secrets_present is not false")
if count < min_events:
    errors.append(f"expected at least {min_events} event(s), got {count}")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
print(f"validated_events={count}", file=sys.stderr)
  ' "$MIN_EVENTS"
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would GET ${BASE_URL%/}/v1/events/stream?session_id=${SESSION_ID}"
    log "DRY-RUN: would validate NDJSON contract_version=rpp.l2.v1 and redaction.secrets_present=false"
    exit 0
  fi

  stream_events | validate_ndjson
  log "PASS"
}

main "$@"
