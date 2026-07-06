#!/usr/bin/env bash
# Smoke: assert no secret-like tokens appear in logs/events/evidence and that
# events with secrets_present=true are rejected.
# Contract references:
#   - docs/contracts/l2-runtime-contract.md
#   - docs/security/secrets-redaction-policy.md
#   - docs/security/audit-event-taxonomy.md

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="redaction-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
CONTRACT_FILE="${REPO_ROOT}/docs/contracts/l2-runtime-contract.md"
SECRETS_POLICY_FILE="${REPO_ROOT}/docs/security/secrets-redaction-policy.md"
AUDIT_TAXONOMY_FILE="${REPO_ROOT}/docs/security/audit-event-taxonomy.md"

BASE_URL="${L2_BASE_URL:-http://127.0.0.1:43117}"
TOKEN_ENV="${L2_BEARER_TOKEN_ENV:-L2_BEARER_TOKEN}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-8}"
SESSION_ID="${SMOKE_SESSION_ID:-session-smoke}"
DRY_RUN=1

# Test secret markers from secrets-redaction-policy.md §5
TEST_MARKERS=(
  "sk-test-secret"
  "Bearer test-secret"
  "postgres://user:pass@example/db"
  "redis://:pass@example:6379"
)

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

usage() {
  cat <<'USAGE'
Usage: redaction-smoke.sh [--dry-run|--execute] [--base-url URL] [--session-id ID] [--timeout SECONDS] [--logs-dir PATH] [--events-file PATH] [--evidence-dir PATH]

Validates that no secret-like tokens appear in logs, runtime events, or evidence files.
Also asserts that any event with secrets_present=true is rejected.

Dry-run is the default and prints planned checks only.

Execute gates:
  SMOKE_ALLOW_EXECUTE=1 must be set.
  If SMOKE_TARGET_ENV=prod, DEPLOY_OWNER_APPROVED=true must also be set.
  L2_BEARER_TOKEN (or env var named by L2_BEARER_TOKEN_ENV) must contain the bearer token.

This script never starts/restarts prodex and never performs deploy actions.
USAGE
}

parse_args() {
  LOGS_DIR="${LOGS_DIR:-${REPO_ROOT}/logs}"
  EVENTS_FILE="${EVENTS_FILE:-}"
  EVIDENCE_DIR="${EVIDENCE_DIR:-${REPO_ROOT}/evidence}"

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
      --timeout)
        shift
        [[ $# -gt 0 ]] || die "--timeout requires a value"
        TIMEOUT_SECONDS="$1"
        ;;
      --logs-dir)
        shift
        [[ $# -gt 0 ]] || die "--logs-dir requires a value"
        LOGS_DIR="$1"
        ;;
      --events-file)
        shift
        [[ $# -gt 0 ]] || die "--events-file requires a value"
        EVENTS_FILE="$1"
        ;;
      --evidence-dir)
        shift
        [[ $# -gt 0 ]] || die "--evidence-dir requires a value"
        EVIDENCE_DIR="$1"
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
  [[ -r "$SECRETS_POLICY_FILE" ]] || die "secrets policy not readable: $SECRETS_POLICY_FILE"
  [[ -r "$AUDIT_TAXONOMY_FILE" ]] || die "audit taxonomy not readable: $AUDIT_TAXONOMY_FILE"
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
  command -v python3 >/dev/null 2>&1 || die "python3 is required for validation"
  command -v grep >/dev/null 2>&1 || die "grep is required"
  [[ -n "${!TOKEN_ENV-}" ]] || die "bearer token env var is empty: $TOKEN_ENV"
}

# Check a file/directory for secret markers
check_for_secrets() {
  local path="$1"
  local label="$2"
  local errors=()

  if [[ ! -e "$path" ]]; then
    log "SKIP $label: path does not exist: $path"
    return 0
  fi

  log "Checking $label: $path"
  for marker in "${TEST_MARKERS[@]}"; do
    if grep -r -F "$marker" "$path" >/dev/null 2>&1; then
      errors+=("found raw secret marker '$marker' in $label")
    fi
  done

  # Also check for common secret patterns (bearer tokens, api keys, db urls, redis urls)
  local patterns=(
    'Bearer [A-Za-z0-9_\-]{20,}'
    'sk-[A-Za-z0-9]{20,}'
    'postgres://[^:]+:[^@]+@'
    'redis://:[^@]+@'
    'api[_-]?key["\s:=]+[A-Za-z0-9_\-]{20,}'
    'secret["\s:=]+[A-Za-z0-9_\-]{20,}'
    'password["\s:=]+[^"\s]{8,}'
    'token["\s:=]+[A-Za-z0-9_\-]{20,}'
  )

  for pattern in "${patterns[@]}"; do
    if grep -r -E "$pattern" "$path" >/dev/null 2>&1; then
      errors+=("found secret-like pattern '$pattern' in $label")
    fi
  done

  if ((${#errors[@]} > 0)); then
    for err in "${errors[@]}"; do
      log "FAIL: $err"
    done
    return 1
  fi

  log "PASS: $label clean"
  return 0
}

# Validate event stream for secrets_present=false
validate_event_stream() {
  local token="${!TOKEN_ENV}"
  local url="${BASE_URL%/}/v1/events/stream?session_id=${SESSION_ID}"

  log "Fetching event stream from $url"
  curl -fsS --no-buffer --max-time "$TIMEOUT_SECONDS" \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/x-ndjson, application/json" \
    "$url" |
  python3 -c '
import json
import sys

errors = []
count = 0
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
    if event.get("secrets_present") is not False:
        got = event.get("secrets_present")
        errors.append(f"line {count}: secrets_present is not false (got {got})")
if count == 0:
    errors.append("no events received")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
print(f"validated_events={count}", file=sys.stderr)
'
}

main() {
  parse_args "$@"
  validate_common
  require_execute_gate

  if ((DRY_RUN == 1)); then
    log "DRY-RUN: would check logs dir: $LOGS_DIR"
    log "DRY-RUN: would check events file: ${EVENTS_FILE:-'<from /v1/events/stream>'}"
    log "DRY-RUN: would check evidence dir: $EVIDENCE_DIR"
    log "DRY-RUN: would validate secret markers: ${TEST_MARKERS[*]}"
    log "DRY-RUN: would assert secrets_present=false on event stream"
    log "DRY-RUN: would reject events with secrets_present=true"
    exit 0
  fi

  local failed=0

  # Check logs directory
  if ! check_for_secrets "$LOGS_DIR" "logs"; then
    failed=1
  fi

  # Check evidence directory
  if ! check_for_secrets "$EVIDENCE_DIR" "evidence"; then
    failed=1
  fi

  # Check events file if provided, otherwise stream from sidecar
  if [[ -n "$EVENTS_FILE" ]]; then
    if [[ ! -r "$EVENTS_FILE" ]]; then
      die "events file not readable: $EVENTS_FILE"
    fi
    log "Checking events file: $EVENTS_FILE"
    if ! python3 -c '
import json
import sys

errors = []
count = 0
with open(sys.argv[1]) as f:
    for raw in f:
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
        if event.get("secrets_present") is not False:
            got = event.get("secrets_present")
            errors.append(f"line {count}: secrets_present is not false (got {got})")
if count == 0:
    errors.append("no events in file")
if errors:
    print("; ".join(errors), file=sys.stderr)
    sys.exit(1)
print(f"validated_events={count}", file=sys.stderr)
' "$EVENTS_FILE"; then
      failed=1
    fi
  else
    if ! validate_event_stream; then
      failed=1
    fi
  fi

  if ((failed == 1)); then
    die "redaction checks failed"
  fi

  log "PASS"
}

main "$@"
