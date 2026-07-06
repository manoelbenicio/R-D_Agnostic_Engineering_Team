#!/usr/bin/env bash
# D2 smoke wrapper: run C1-C6 and S1-S5 against an already-running local
# rpp.l2.v1 sidecar, defaulting to http://127.0.0.1:43292.

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
SMOKE_DIR="${REPO_ROOT}/scripts/smoke"
REPORT_FILE="${D2_REPORT_FILE:-${SCRIPT_DIR}/W3-D2-smoke-rerun.md}"
DEFAULT_BASE_URL="http://127.0.0.1:43292"
BASE_URL="${1:-${L2_BASE_URL:-$DEFAULT_BASE_URL}}"
BASE_URL="${BASE_URL%/}"
TOKEN="${L2_BEARER_TOKEN:-d2-smoke-token}"
TIMESTAMP_UTC="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_ID="d2-${TIMESTAMP_UTC}-$$"
TMP_DIR="$(mktemp -d)"
FAILURES=0

cleanup() {
  rm -rf -- "$TMP_DIR"
}
trap cleanup EXIT

usage() {
  cat <<'USAGE'
Usage: D2-test-runner.sh [L2_BASE_URL]

Runs C1-C6 and S1-S5 smokes in sequence against an already-running local sidecar.
Default L2_BASE_URL is http://127.0.0.1:43292.

Environment:
  L2_BEARER_TOKEN      Bearer token expected by the sidecar. Defaults to d2-smoke-token.
  D2_REPORT_FILE      Report output path. Defaults to .deploy-control/evidence/W3-D2-smoke-rerun.md.
  SMOKE_TENANT_ID     Tenant used by scripts. Defaults to tenant-smoke.
USAGE
}

if [[ "${BASE_URL}" == "-h" || "${BASE_URL}" == "--help" ]]; then
  usage
  exit 0
fi

case "$BASE_URL" in
  http://127.0.0.1:43292 | http://localhost:43292 | http://[::1]:43292) ;;
  *)
    printf 'ERROR: D2 runner requires a loopback sidecar on port 43292, got: %s\n' "$BASE_URL" >&2
    exit 2
    ;;
esac

require_file() {
  local path="$1"
  [[ -x "$path" ]] || {
    printf 'ERROR: smoke script not executable: %s\n' "$path" >&2
    exit 2
  }
}

for smoke in \
  readyz-smoke.sh \
  state-backend-smoke.sh \
  policy-apply-smoke.sh \
  profile-fail-closed-smoke.sh \
  session-start-stop-smoke.sh \
  event-stream-smoke.sh \
  smart-context-measure.sh \
  kill-switch-smoke.sh; do
  require_file "${SMOKE_DIR}/${smoke}"
done

mkdir -p "$(dirname "$REPORT_FILE")"

{
  printf '# W3-D2 smoke rerun\n\n'
  printf -- '- timestamp_utc: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf -- '- runner: `.deploy-control/evidence/D2-test-runner.sh`\n'
  printf -- '- base_url: `%s`\n' "$BASE_URL"
  printf -- '- required_port: `43292`\n'
  printf -- '- target: already-running local sidecar\n'
  printf -- '- secrets_present: false\n\n'
  printf '## Plan\n\n'
  printf 'C1 readiness, C2 state backend, C3 session replay, C4 event stream replay, C5 Smart Context measurement, C6 fail-closed isolation, then S1-S5 smoke sequence.\n\n'
  printf '## Raw Output\n\n'
} >"$REPORT_FILE"

append_command() {
  printf 'Command:' >>"$REPORT_FILE"
  printf ' `%q' "$1" >>"$REPORT_FILE"
  shift
  for arg in "$@"; do
    printf ' %q' "$arg" >>"$REPORT_FILE"
  done
  printf '`\n\n' >>"$REPORT_FILE"
}

run_step() {
  local label="$1"
  shift
  local raw_file="${TMP_DIR}/${label//[^A-Za-z0-9_.-]/_}.log"
  local started_at ended_at rc

  started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  {
    printf '### %s\n\n' "$label"
    printf -- '- started_at: %s\n' "$started_at"
  } >>"$REPORT_FILE"
  append_command "$@"

  set +e
  (
    cd "$REPO_ROOT"
    SMOKE_ALLOW_EXECUTE=1 \
    SMOKE_TARGET_ENV=local \
    L2_BASE_URL="$BASE_URL" \
    L2_BEARER_TOKEN="$TOKEN" \
    SMOKE_TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}" \
    SMOKE_POLICY_ID="${SMOKE_POLICY_ID:-policy-smoke-shadow}" \
    SMOKE_SESSION_ID="${SMOKE_SESSION_ID:-${RUN_ID}-${label//[^A-Za-z0-9]/-}}" \
    SMOKE_WORKING_DIRECTORY="${SMOKE_WORKING_DIRECTORY:-/tmp/rpp-smoke-workspace}" \
      "$@"
  ) >"$raw_file" 2>&1
  rc=$?
  set -e

  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  {
    printf -- '- finished_at: %s\n' "$ended_at"
    printf -- '- exit_code: %s\n\n' "$rc"
    printf '```text\n'
    cat "$raw_file"
    printf '\n```\n\n'
  } >>"$REPORT_FILE"

  if ((rc != 0)); then
    FAILURES=$((FAILURES + 1))
  fi
}

run_step "preflight-healthz" \
  bash -c 'curl -fsS --max-time 8 -H "Authorization: Bearer ${L2_BEARER_TOKEN}" "${L2_BASE_URL%/}/healthz"'

run_step "C1-readyz" \
  "${SMOKE_DIR}/readyz-smoke.sh" --execute --base-url "$BASE_URL" --timeout 8

run_step "C2-state-backend" \
  "${SMOKE_DIR}/state-backend-smoke.sh" --execute --base-url "$BASE_URL" --timeout 8

run_step "C3-session-start-stop" \
  "${SMOKE_DIR}/session-start-stop-smoke.sh" --execute --base-url "$BASE_URL" --session-id "${RUN_ID}-c3-session"

run_step "C4-event-stream" \
  "${SMOKE_DIR}/event-stream-smoke.sh" --execute --base-url "$BASE_URL" --session-id "${RUN_ID}-c3-session" --min-events 1

run_step "C5-smart-context-measure" \
  "${SMOKE_DIR}/smart-context-measure.sh" --execute --base-url "$BASE_URL" --context-kib 64 --session-id "${RUN_ID}-c5-smart-context" --timeout 12

run_step "C6-profile-fail-closed" \
  "${SMOKE_DIR}/profile-fail-closed-smoke.sh" --execute --base-url "$BASE_URL" --invalid-profile-home "/tmp/rpp-smoke-outside-managed-root"

run_step "S1-readyz" \
  "${SMOKE_DIR}/readyz-smoke.sh" --execute --base-url "$BASE_URL" --timeout 8

run_step "S2-policy-apply" \
  "${SMOKE_DIR}/policy-apply-smoke.sh" --execute --base-url "$BASE_URL" --tenant-id "${SMOKE_TENANT_ID:-tenant-smoke}"

run_step "S3-account-fail-closed" \
  "${SMOKE_DIR}/profile-fail-closed-smoke.sh" --execute --base-url "$BASE_URL" --invalid-profile-home "/tmp/rpp-smoke-outside-managed-root"

run_step "S4-session-start-stop" \
  "${SMOKE_DIR}/session-start-stop-smoke.sh" --execute --base-url "$BASE_URL" --session-id "${RUN_ID}-s4-session"

run_step "S5-kill-switch" \
  "${SMOKE_DIR}/kill-switch-smoke.sh" --execute --base-url "$BASE_URL" --feature smart_context

{
  printf '## Result\n\n'
  if ((FAILURES == 0)); then
    printf 'PASS - all D2 smoke steps exited 0.\n'
  else
    printf 'FAIL - %s D2 smoke step(s) exited non-zero.\n' "$FAILURES"
  fi
} >>"$REPORT_FILE"

printf 'D2 report: %s\n' "$REPORT_FILE"

if ((FAILURES != 0)); then
  exit 1
fi
