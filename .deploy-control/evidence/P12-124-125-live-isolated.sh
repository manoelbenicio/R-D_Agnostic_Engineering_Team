#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="P12-124-125-live-isolated"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
OUT="${P12_EVIDENCE_OUT:-${SCRIPT_DIR}/P12-124-125-live-isolated.md}"
RUN_ID="${P12_RUN_ID:-p12-124-125-$(date -u +%Y%m%dT%H%M%SZ)}"
BASE_URL="${P12_BASE_URL:-${L2_BASE_URL:-${MULTICA_L2_BASE_URL:-}}}"
TOKEN="${P12_BEARER_TOKEN:-${L2_BEARER_TOKEN:-${MULTICA_L2_BEARER_TOKEN:-}}}"
TENANT_ID="${P12_TENANT_ID:-p12-live-killswitch}"
PROVIDER="${P12_PROVIDER:-codex}"
PROFILE_ID="${P12_PROFILE_ID:-p12-live-profile}"
FEATURE="${P12_KILL_FEATURE:-gateway}"
ROLLBACK_COMMAND="${P12_ROLLBACK_COMMAND:-}"
ALLOW_ROLLBACK="${P12_ALLOW_ROLLBACK:-0}"
TIMEOUT="${P12_TIMEOUT_SECONDS:-10}"
WORK_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

redact() {
  sed -E \
    -e 's/(Authorization: Bearer )[A-Za-z0-9._~+\/=-]+/\1<redacted>/g' \
    -e 's/(Bearer )[A-Za-z0-9._~+\/=-]+/\1<redacted>/g' \
    -e 's/(token[=:] ?)[^", ]+/\1<redacted>/Ig' \
    -e 's/(api[_-]?key[=:] ?)[^", ]+/\1<redacted>/Ig' \
    -e 's/sk-[A-Za-z0-9._-]+/sk-<redacted>/g'
}

append() {
  printf '%s\n' "$*" >> "$OUT"
}

append_code_file() {
  local label="$1"
  local file="$2"
  append ""
  append "### ${label}"
  append '```text'
  if [[ -s "$file" ]]; then
    redact < "$file" >> "$OUT"
  else
    append "<empty>"
  fi
  append '```'
}

json_value() {
  local file="$1"
  local expr="$2"
  python3 - "$file" "$expr" <<'PY' 2>/dev/null || true
import json
import sys

path, expr = sys.argv[1:3]
try:
    with open(path, "r", encoding="utf-8") as fh:
        data = json.load(fh)
except Exception:
    sys.exit(0)

cur = data
for part in expr.split("."):
    if not part:
        continue
    if isinstance(cur, dict):
        cur = cur.get(part)
    else:
        cur = None
        break
if cur is None:
    sys.exit(0)
print(cur)
PY
}

curl_capture() {
  local method="$1"
  local url="$2"
  local data_file="${3:-}"
  local label="$4"
  local body_file="${WORK_DIR}/${label}.body"
  local meta_file="${WORK_DIR}/${label}.meta"
  local headers=(-H "Accept: application/json")
  if [[ -n "$TOKEN" ]]; then
    headers+=(-H "Authorization: Bearer ${TOKEN}")
  fi
  if [[ -n "$data_file" ]]; then
    headers+=(-H "Content-Type: application/json")
  fi

  {
    printf '$ curl -sS --max-time %s -X %s %q' "$TIMEOUT" "$method" "$url"
    if [[ -n "$TOKEN" ]]; then
      printf ' -H %q' 'Authorization: Bearer <redacted>'
    fi
    if [[ -n "$data_file" ]]; then
      printf ' -H %q --data-binary @%s' 'Content-Type: application/json' "$(basename "$data_file")"
    fi
    printf '\n'
  } >> "$meta_file"

  local status
  if [[ -n "$data_file" ]]; then
    status="$(curl -sS --max-time "$TIMEOUT" -X "$method" "$url" "${headers[@]}" --data-binary @"$data_file" -o "$body_file" -w '%{http_code}' 2>>"$meta_file" || true)"
  else
    status="$(curl -sS --max-time "$TIMEOUT" -X "$method" "$url" "${headers[@]}" -o "$body_file" -w '%{http_code}' 2>>"$meta_file" || true)"
  fi
  printf 'http_status=%s\n' "${status:-curl_failed}" >> "$meta_file"

  append_code_file "${label} command/status" "$meta_file"
  append_code_file "${label} body" "$body_file"
  printf '%s\n' "${status:-curl_failed}" > "${WORK_DIR}/${label}.status"
}

payload_session_start() {
  local session_id="$1"
  python3 - "$TENANT_ID" "$PROVIDER" "$PROFILE_ID" "$session_id" "$RUN_ID" <<'PY'
import json
import sys

tenant_id, provider, profile_id, session_id, run_id = sys.argv[1:6]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req-{run_id}-{session_id}",
    "tenant_id": tenant_id,
    "workspace_id": "workspace-p12-live",
    "task_id": f"task-{session_id}",
    "session_id": session_id,
    "requested_provider": provider,
    "provider": provider,
    "profile_pool": [profile_id],
    "task": {
        "task_id": f"task-{session_id}",
        "workspace_id": "workspace-p12-live"
    }
}, separators=(",", ":")))
PY
}

payload_runtime() {
  python3 - <<'PY'
import json

print(json.dumps({
    "request_id": "req-p12-live-runtime",
    "body": {
        "model": "gpt-4.1",
        "input": [{"role": "user", "content": "P12 live routing probe. Return one short word."}],
        "instructions": "respond with ok",
        "max_output_tokens": 1
    }
}, separators=(",", ":")))
PY
}

payload_kill() {
  local state="$1"
  python3 - "$TENANT_ID" "$PROVIDER" "$PROFILE_ID" "$FEATURE" "$state" "$RUN_ID" <<'PY'
import json
import sys

tenant_id, provider, profile_id, feature, state, run_id = sys.argv[1:7]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req-{run_id}-killswitch-{feature}-{state}",
    "tenant_id": tenant_id,
    "scope": {
        "provider": provider,
        "profile_id": profile_id
    },
    "feature": feature,
    "state": state,
    "reason": "p12_live_test",
    "effective_at": "immediate"
}, separators=(",", ":")))
PY
}

discover_base_url() {
  [[ -n "$BASE_URL" ]] && return 0
  local candidate
  for candidate in http://127.0.0.1:43293 http://127.0.0.1:43292 http://127.0.0.1:43117; do
    local code
    code="$(curl -sS --max-time 1 -H "Authorization: Bearer ${TOKEN:-missing}" "${candidate}/readyz" -o /dev/null -w '%{http_code}' 2>/dev/null || true)"
    if [[ "$code" != "000" ]]; then
      BASE_URL="$candidate"
      return 0
    fi
  done
}

init_evidence() {
  cat > "$OUT" <<EOF
# P12 Tasks 12.4/12.5 Live Isolated Evidence

- task_ids: 12.4 kill-switch LIVE, 12.5 rollback LIVE
- timestamp_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- run_id: ${RUN_ID}
- runner: Codex#5.5#B
- host: $(hostname)
- cwd: ${REPO_ROOT}
- plan_ref: .planning/phases/12-prod-deploy/PLAN.md
- evidence_contract: .planning/EVIDENCE_CONTRACT.md
- secrets_present: false

## Provenance Commands

\`\`\`text
$ date -u +%Y-%m-%dT%H:%M:%SZ
$(date -u +%Y-%m-%dT%H:%M:%SZ)

$ hostname
$(hostname)

$ git rev-parse --short HEAD
$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || printf 'unknown')
\`\`\`
EOF
}

record_environment() {
  append ""
  append "## Deployed Gateway Discovery"
  append ""
  append "- P12_BASE_URL_supplied: $([[ -n "${P12_BASE_URL:-}" ]] && printf true || printf false)"
  append "- resolved_base_url: ${BASE_URL:-<none>}"
  append "- bearer_token_supplied: $([[ -n "$TOKEN" ]] && printf true || printf false)"
  append "- rollback_command_supplied: $([[ -n "$ROLLBACK_COMMAND" ]] && printf true || printf false)"
  append ""
  append "### Process/Port Snapshot"
  append '```text'
  {
    printf '$ ps -eo pid,etimes,cmd | rg -i "prodex|sidecar|gateway"\n'
    ps -eo pid,etimes,cmd | rg -i 'prodex|sidecar|gateway' | rg -v 'rg -i' || true
    printf '\n$ ss -ltnp | rg "43117|43292|43293|43291|prodex|sidecar|gateway"\n'
    ss -ltnp 2>/dev/null | rg '43117|43292|43293|43291|prodex|sidecar|gateway' || true
    printf '\n$ docker ps --format ... | rg "prodex|sidecar|gateway|multica|deploy-"\n'
    docker ps --format '{{.Names}}\t{{.Image}}\t{{.Ports}}\t{{.Status}}' 2>/dev/null | rg 'prodex|sidecar|gateway|multica|deploy-' || true
  } | redact >> "$OUT"
  append '```'

  case "${BASE_URL:-}" in
    http://127.0.0.1:*|http://localhost:*|http://[::1]:*)
      append ""
      append "> [!CAUTION]"
      append "> INVALID_PROVENANCE: resolved base URL is loopback (${BASE_URL}), which EVIDENCE_CONTRACT Rule 1 rejects for PROD live evidence."
      ;;
    "")
      append ""
      append "> [!CAUTION]"
      append "> BLOCKED: no deployed gateway base URL found or supplied. Set P12_BASE_URL to the deployed PROD gateway/sidecar endpoint."
      ;;
  esac
}

run_killswitch_live() {
  append ""
  append "## 12.4 Kill-Switch LIVE"
  if [[ -z "${BASE_URL:-}" ]]; then
    append ""
    append "Result: BLOCKED before request; no gateway endpoint."
    return 1
  fi

  local session_payload runtime_payload kill_payload status
  session_payload="${WORK_DIR}/session-baseline.json"
  runtime_payload="${WORK_DIR}/runtime.json"
  kill_payload="${WORK_DIR}/kill.json"

  payload_session_start "${RUN_ID}-baseline" > "$session_payload"
  curl_capture POST "${BASE_URL%/}/v1/session/start" "$session_payload" "12.4-baseline-session-start"
  local baseline_body="${WORK_DIR}/12.4-baseline-session-start.body"
  local runtime_endpoint
  runtime_endpoint="$(json_value "$baseline_body" "runtime_endpoint")"
  append ""
  append "- baseline_runtime_endpoint: ${runtime_endpoint:-<none>}"
  append "- baseline_router_owner: $(json_value "$baseline_body" "router_owner")"
  append "- baseline_runtime_session_id: $(json_value "$baseline_body" "runtime_session_id")"

  if [[ -n "$runtime_endpoint" ]]; then
    payload_runtime > "$runtime_payload"
    curl_capture POST "$runtime_endpoint" "$runtime_payload" "12.4-baseline-runtime-proxy"
  fi

  payload_kill disabled > "$kill_payload"
  curl_capture POST "${BASE_URL%/}/v1/killswitch/apply" "$kill_payload" "12.4-killswitch-disable"
  status="$(cat "${WORK_DIR}/12.4-killswitch-disable.status")"
  append ""
  append "- disable_http_status: ${status}"
  append "- disable_applied: $(json_value "${WORK_DIR}/12.4-killswitch-disable.body" "applied")"

  payload_session_start "${RUN_ID}-blocked-after-kill" > "$session_payload"
  curl_capture POST "${BASE_URL%/}/v1/session/start" "$session_payload" "12.4-after-disable-session-start"
  append ""
  append "- after_disable_http_status: $(cat "${WORK_DIR}/12.4-after-disable-session-start.status")"
  append "- after_disable_router_owner: $(json_value "${WORK_DIR}/12.4-after-disable-session-start.body" "router_owner")"

  payload_kill enabled > "$kill_payload"
  curl_capture POST "${BASE_URL%/}/v1/killswitch/apply" "$kill_payload" "12.4-killswitch-enable"
  append ""
  append "- enable_http_status: $(cat "${WORK_DIR}/12.4-killswitch-enable.status")"
  append "- enable_applied: $(json_value "${WORK_DIR}/12.4-killswitch-enable.body" "applied")"

  payload_session_start "${RUN_ID}-resumed-after-kill" > "$session_payload"
  curl_capture POST "${BASE_URL%/}/v1/session/start" "$session_payload" "12.4-after-enable-session-start"
  append ""
  append "- after_enable_http_status: $(cat "${WORK_DIR}/12.4-after-enable-session-start.status")"
  append "- after_enable_router_owner: $(json_value "${WORK_DIR}/12.4-after-enable-session-start.body" "router_owner")"
}

run_rollback_live() {
  append ""
  append "## 12.5 Rollback LIVE"
  if [[ -z "$ROLLBACK_COMMAND" ]]; then
    append ""
    append "Result: BLOCKED. P12_ROLLBACK_COMMAND was not supplied, so there is no owner-approved one-command rollback to execute."
    return 1
  fi
  if [[ "$ALLOW_ROLLBACK" != "1" ]]; then
    append ""
    append "Result: BLOCKED. P12_ALLOW_ROLLBACK=1 was not supplied; refusing destructive rollback command."
    append ""
    append "Configured one-command rollback (redacted):"
    append '```text'
    printf '%s\n' "$ROLLBACK_COMMAND" | redact >> "$OUT"
    append '```'
    return 1
  fi

  append ""
  append "### Rollback command"
  append '```text'
  printf '$ %s\n' "$ROLLBACK_COMMAND" | redact >> "$OUT"
  append '```'

  local rb_out="${WORK_DIR}/rollback.out"
  set +e
  bash -lc "$ROLLBACK_COMMAND" >"$rb_out" 2>&1
  local rb_status=$?
  set -e
  append_code_file "rollback output" "$rb_out"
  append ""
  append "- rollback_exit_code: ${rb_status}"

  if [[ -n "${BASE_URL:-}" ]]; then
    curl_capture GET "${BASE_URL%/}/readyz" "" "12.5-after-rollback-readyz"
  fi

  local raw_check="${P12_RAW_CODEX_CHECK_COMMAND:-codex --version}"
  local raw_out="${WORK_DIR}/raw-codex.out"
  append ""
  append "### Raw Codex check command"
  append '```text'
  printf '$ %s\n' "$raw_check" | redact >> "$OUT"
  append '```'
  set +e
  bash -lc "$raw_check" >"$raw_out" 2>&1
  local raw_status=$?
  set -e
  append_code_file "raw codex check output" "$raw_out"
  append ""
  append "- raw_codex_check_exit_code: ${raw_status}"
}

summarize() {
  append ""
  append "## Verdict"
  if [[ -z "${BASE_URL:-}" ]]; then
    append ""
    append "BLOCKED: no deployed gateway endpoint was available in this isolated environment."
  elif [[ "${BASE_URL}" == http://127.0.0.1:* || "${BASE_URL}" == http://localhost:* || "${BASE_URL}" == http://[::1]:* ]]; then
    append ""
    append "INVALID under EVIDENCE_CONTRACT Rule 1 for PROD: endpoint is loopback (${BASE_URL}). Captured commands/responses are retained for audit, but cannot close P12 LIVE."
  elif [[ -z "$ROLLBACK_COMMAND" || "$ALLOW_ROLLBACK" != "1" ]]; then
    append ""
    append "BLOCKED: kill-switch probes may be captured above, but rollback LIVE cannot execute without P12_ROLLBACK_COMMAND and P12_ALLOW_ROLLBACK=1."
  else
    append ""
    append "Review raw HTTP statuses above. PASS requires before route succeeds, kill-switch stops routing, remove resumes routing, rollback command exits 0, and raw Codex check succeeds."
  fi
}

main() {
  command -v curl >/dev/null 2>&1 || { echo "curl required" >&2; exit 2; }
  command -v python3 >/dev/null 2>&1 || { echo "python3 required" >&2; exit 2; }
  command -v rg >/dev/null 2>&1 || true
  init_evidence
  discover_base_url
  record_environment
  run_killswitch_live || true
  run_rollback_live || true
  summarize
  printf '%s\n' "$OUT"
}

main "$@"
