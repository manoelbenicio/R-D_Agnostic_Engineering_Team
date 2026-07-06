#!/usr/bin/env bash
# P6 smoke suite: S1-S5 against the local rpp.l2.v1 sidecar surface, with
# direct verification of the pinned prodex binary under bin/prodex.

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="p6-s1-s5-prodex-smoke"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"

PRODEX_BIN="${PRODEX_BIN:-${REPO_ROOT}/bin/prodex}"
SIDECAR_BIN="${P6_SIDECAR_BIN:-${REPO_ROOT}/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar}"
BIND_ADDR="${P6_BIND_ADDR:-127.0.0.1:43117}"
BASE_URL="http://${BIND_ADDR}"
TENANT_ID="${SMOKE_TENANT_ID:-tenant-smoke}"
UNKNOWN_TENANT_ID="${SMOKE_UNKNOWN_TENANT_ID:-tenant-unknown}"
PROFILE_ID="${SMOKE_PROFILE_ID:-codex-smoke-main}"
BACKUP_PROFILE_ID="${SMOKE_BACKUP_PROFILE_ID:-codex-smoke-backup}"
POLICY_ID="${SMOKE_POLICY_ID:-policy-smoke-shadow}"
SESSION_ID="${SMOKE_SESSION_ID:-session-smoke-s4}"
KILL_SESSION_ID="${SMOKE_KILL_SESSION_ID:-session-smoke-s5}"
TIMESTAMP_UTC="$(date -u +%Y%m%dT%H%M%SZ)"
EVIDENCE_FILE="${P6_EVIDENCE_FILE:-${REPO_ROOT}/.deploy-control/evidence/p6-s1-s5-prodex-bin-${TIMESTAMP_UTC}.md}"
TMP_DIR="$(mktemp -d)"
TOKEN="p6-smoke-${TIMESTAMP_UTC}-$$"
SIDECAR_PID=""

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

cleanup() {
  if [[ -n "$SIDECAR_PID" ]] && kill -0 "$SIDECAR_PID" 2>/dev/null; then
    kill "$SIDECAR_PID" 2>/dev/null || true
    wait "$SIDECAR_PID" 2>/dev/null || true
  fi
  rm -rf -- "$TMP_DIR"
}
trap cleanup EXIT

record() {
  printf '%s\n' "$*" >>"$EVIDENCE_FILE"
}

curl_json() {
  local method="$1" path="$2" payload="${3:-}"
  if [[ -n "$payload" ]]; then
    curl -fsS --max-time 8 \
      -X "$method" "${BASE_URL}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary "$payload"
  else
    curl -fsS --max-time 8 \
      -X "$method" "${BASE_URL}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Accept: application/json"
  fi
}

post_status() {
  local path="$1" payload="$2" body_file="$3"
  curl -sS --max-time 8 \
    -o "$body_file" \
    -w '%{http_code}' \
    -X POST "${BASE_URL}${path}" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary "$payload"
}

json_assert() {
  python3 -c '
import json
import sys

mode = sys.argv[1]
data = json.load(sys.stdin)
errors = []

if mode == "healthz":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("status") != "alive":
        errors.append("status")
elif mode == "readyz":
    checks = {c.get("name"): c for c in data.get("checks", [])}
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("status") != "ready":
        errors.append("status")
    for name in ("shared_state_backend", "kill_switch", "runtime_proxy"):
        if checks.get(name, {}).get("status") != "pass":
            errors.append(f"check:{name}")
    backend = checks.get("shared_state_backend", {}).get("details", {}).get("backend_type")
    if backend != "postgres":
        errors.append("backend_type")
elif mode == "policy":
    policy_id, revision = sys.argv[2], int(sys.argv[3])
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("policy_id") != policy_id:
        errors.append("policy_id")
    if data.get("revision") != revision:
        errors.append("revision")
    if data.get("applied") is not True:
        errors.append("applied")
elif mode == "register":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("registered_profile_count") != 1:
        errors.append("registered_profile_count")
    if data.get("rejected_profiles") != []:
        errors.append("rejected_profiles")
elif mode == "reject":
    rejected = data.get("rejected_profiles")
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if not isinstance(rejected, list) or not rejected:
        errors.append("missing rejection")
elif mode == "start":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("router_owner") != "rust_l2":
        errors.append("router_owner")
    if not data.get("runtime_session_id"):
        errors.append("runtime_session_id")
    print(data.get("runtime_session_id", ""))
elif mode == "start_exact":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("router_owner") != "rust_l2":
        errors.append("router_owner")
    if data.get("smart_context_mode") != "exact":
        errors.append("smart_context_mode")
elif mode == "stop":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("stopped") is not True:
        errors.append("stopped")
elif mode == "kill":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("applied") is not True:
        errors.append("applied")
    if data.get("effective_at") != "next_request":
        errors.append("effective_at")
elif mode == "kill_status":
    if data.get("contract_version") != "rpp.l2.v1":
        errors.append("contract_version")
    if data.get("active") is not True:
        errors.append("active")
else:
    errors.append(f"unknown mode {mode}")

if errors:
    raise SystemExit("; ".join(errors))
' "$@"
}

validate_events() {
  local session_id="$1" min_events="$2" required_type="${3:-}"
  local events_file="${TMP_DIR}/events-${session_id}.ndjson"
  curl -fsS --max-time 8 \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Accept: application/x-ndjson, application/json" \
    "${BASE_URL}/v1/events/stream?session_id=${session_id}" >"$events_file"
  python3 - "$events_file" "$min_events" "$required_type" <<'PY'
import json
import sys

path = sys.argv[1]
min_events = int(sys.argv[2])
required_type = sys.argv[3]
events = []
for raw in open(path):
    raw = raw.strip()
    if raw:
        events.append(json.loads(raw))
if len(events) < min_events:
    raise SystemExit(f"expected at least {min_events} events, got {len(events)}")
for event in events:
    if event.get("contract_version") != "rpp.l2.v1":
        raise SystemExit("event contract_version")
    if event.get("secrets_present") is not False:
        raise SystemExit("event secrets_present")
if required_type and required_type not in {event.get("event_type") for event in events}:
    raise SystemExit(f"missing event_type {required_type}")
print(f"validated_events={len(events)}", file=sys.stderr)
PY
}

payload_policy() {
  local tenant="$1"
  python3 - "$tenant" "$POLICY_ID" <<'PY'
import json
import sys
tenant, policy_id = sys.argv[1:3]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_{policy_id}_{tenant}",
    "tenant_id": tenant,
    "policy_id": policy_id,
    "revision": 1,
    "allowed_providers": ["codex"],
    "allowed_profiles": ["codex-smoke-main", "codex-smoke-backup"],
    "budgets": {"max_requests_per_session": 3, "max_estimated_input_tokens_per_request": 180000},
    "smart_context": {"mode": "shadow", "canary_percent": 0, "exact_mode_allowed": True},
    "auto_redeem": {"enabled": False, "cooldown_seconds": 86400},
    "gateway": {"enabled": True, "adaptive_routing": "shadow"},
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

payload_accounts() {
  local tenant="$1" profile_home="$2" extra="${3:-}"
  python3 - "$tenant" "$PROFILE_ID" "$profile_home" "$extra" <<'PY'
import json
import sys
tenant, profile_id, profile_home, extra = sys.argv[1:5]
profile = {
    "profile_id": profile_id,
    "provider": "codex",
    "profile_home": profile_home,
    "auth_mode": "oauth_profile",
    "status": "approved",
    "capability_ref": "codex.oauth_profile.v1"
}
if extra == "raw_auth":
    profile["auth_json"] = {"token": "SHOULD_BE_REJECTED_NOT_A_REAL_SECRET"}
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_accounts_{profile_id}",
    "tenant_id": tenant,
    "profiles": [profile]
}, separators=(",", ":")))
PY
}

payload_start() {
  local session_id="$1"
  python3 - "$TENANT_ID" "$session_id" "$POLICY_ID" "$PROFILE_ID" "$BACKUP_PROFILE_ID" <<'PY'
import json
import sys
tenant, session_id, policy_id, profile_id, backup_profile_id = sys.argv[1:6]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_start_{session_id}",
    "tenant_id": tenant,
    "workspace_id": "workspace-smoke",
    "task_id": f"task-{session_id}",
    "session_id": session_id,
    "policy_id": policy_id,
    "requested_provider": "codex",
    "requested_model": "gpt-5",
    "working_directory": "/tmp/rpp-smoke-workspace",
    "profile_pool": [profile_id, backup_profile_id],
    "continuation": {"previous_response_id": None, "session_binding_hint": None}
}, separators=(",", ":")))
PY
}

payload_stop() {
  local session_id="$1" runtime_session_id="$2"
  python3 - "$TENANT_ID" "$session_id" "$runtime_session_id" <<'PY'
import json
import sys
tenant, session_id, runtime_session_id = sys.argv[1:4]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_stop_{session_id}",
    "tenant_id": tenant,
    "session_id": session_id,
    "runtime_session_id": runtime_session_id,
    "reason": "operator_requested"
}, separators=(",", ":")))
PY
}

payload_kill() {
  python3 - "$TENANT_ID" <<'PY'
import json
import sys
tenant = sys.argv[1]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": "req_kill_smart_context_tenant_smoke",
    "tenant_id": tenant,
    "scope": {},
    "feature": "smart_context",
    "state": "disabled",
    "reason": "operator_guardrail",
    "effective_at": "next_request"
}, separators=(",", ":")))
PY
}

ensure_no_secret_keys() {
  python3 - "$@" <<'PY'
import json
import sys
for path in sys.argv[1:]:
    data = json.load(open(path))
    stack = [data]
    while stack:
        value = stack.pop()
        if isinstance(value, dict):
            for key, nested in value.items():
                if key.lower() in {"api_key", "access_token", "refresh_token", "bearer_token", "cookie", "cookies", "auth_json", "auth", "raw_auth"}:
                    raise SystemExit(f"forbidden key present: {key}")
                stack.append(nested)
        elif isinstance(value, list):
            stack.extend(value)
PY
}

main() {
  [[ -x "$PRODEX_BIN" ]] || die "prodex binary not executable: $PRODEX_BIN"
  [[ -x "$SIDECAR_BIN" ]] || die "sidecar binary not executable: $SIDECAR_BIN"
  command -v curl >/dev/null 2>&1 || die "curl is required"
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  if ss -ltn "( sport = :${BIND_ADDR##*:} )" | grep -q LISTEN; then
    die "port already in use: ${BIND_ADDR}"
  fi

  mkdir -p "$(dirname "$EVIDENCE_FILE")"
  : >"$EVIDENCE_FILE"
  record "# P6 S1-S5 prodex/bin Smoke Evidence"
  record ""
  record "- timestamp_utc: ${TIMESTAMP_UTC}"
  record "- prodex_bin: ${PRODEX_BIN}"
  record "- sidecar_bin: ${SIDECAR_BIN}"
  record "- base_url: ${BASE_URL}"
  record "- tenant_id: ${TENANT_ID}"
  record "- secrets_present: false"
  record ""

  local version
  version="$("$PRODEX_BIN" --version)"
  [[ "$version" == "prodex 0.246.0" ]] || die "unexpected prodex version: $version"
  PRODEX_HOME="${TMP_DIR}/prodex-home" "$PRODEX_BIN" doctor --runtime --json >"${TMP_DIR}/prodex-doctor.json"
  PRODEX_HOME="${TMP_DIR}/prodex-home" "$PRODEX_BIN" capability list --json >"${TMP_DIR}/prodex-capabilities.json"
  python3 - "${TMP_DIR}/prodex-doctor.json" "${TMP_DIR}/prodex-capabilities.json" <<'PY'
import json
import sys
doctor = json.load(open(sys.argv[1]))
caps = json.load(open(sys.argv[2]))
if doctor.get("runtime_logs", {}).get("format") != "text":
    raise SystemExit("doctor runtime_logs format missing")
if not any(c.get("name") == "smart-context" and c.get("status") == "built-in" for c in caps):
    raise SystemExit("smart-context capability missing")
PY
  record "## Direct bin/prodex"
  record ""
  record "- version: ${version}"
  record "- doctor --runtime --json: PASS"
  record "- capability list --json includes smart-context built-in: PASS"
  record ""

  mkdir -p /tmp/rpp-smoke/profiles/"$PROFILE_ID" /tmp/rpp-smoke/profiles/"$BACKUP_PROFILE_ID" /tmp/rpp-smoke-workspace
  chmod 700 /tmp/rpp-smoke /tmp/rpp-smoke/profiles/"$PROFILE_ID" /tmp/rpp-smoke/profiles/"$BACKUP_PROFILE_ID" /tmp/rpp-smoke-workspace 2>/dev/null || true

  MULTICA_L2_BEARER_TOKEN="$TOKEN" \
  MULTICA_L2_ALLOWED_TENANTS="$TENANT_ID" \
  MULTICA_PRODEX_PATH="$PRODEX_BIN" \
  MULTICA_PRODEX_VERSION="0.246.0" \
  MULTICA_PRODEX_COMMIT="7750da9b" \
  PRODEX_HOME="${TMP_DIR}/prodex-home" \
    "$SIDECAR_BIN" "$BIND_ADDR" >"${TMP_DIR}/sidecar.log" 2>&1 &
  SIDECAR_PID="$!"
  for _ in $(seq 1 50); do
    if ! kill -0 "$SIDECAR_PID" 2>/dev/null; then
      die "sidecar exited before readiness; see ${TMP_DIR}/sidecar.log"
    fi
    if curl -fsS --max-time 1 -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/healthz" >/dev/null 2>&1; then
      break
    fi
    sleep 0.1
  done
  curl -fsS --max-time 2 -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/healthz" >/dev/null

  record "## S1 readiness"
  curl_json GET /healthz | json_assert healthz
  curl_json GET /readyz | json_assert readyz
  SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BEARER_TOKEN="$TOKEN" L2_BASE_URL="$BASE_URL" "${REPO_ROOT}/scripts/smoke/readyz-smoke.sh" --execute
  SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=local L2_BEARER_TOKEN="$TOKEN" L2_BASE_URL="$BASE_URL" "${REPO_ROOT}/scripts/smoke/state-backend-smoke.sh" --execute
  record "- healthz OK; readyz OK; shared_state_backend=postgres; kill_switch readable: PASS"
  record ""

  record "## S2 policy apply"
  local policy_payload unknown_payload unknown_body unknown_status
  policy_payload="$(payload_policy "$TENANT_ID")"
  printf '%s' "$policy_payload" >"${TMP_DIR}/policy.json"
  ensure_no_secret_keys "${TMP_DIR}/policy.json"
  curl_json POST /v1/policy/apply "$policy_payload" | json_assert policy "$POLICY_ID" 1
  unknown_payload="$(payload_policy "$UNKNOWN_TENANT_ID")"
  unknown_body="${TMP_DIR}/unknown-policy.json"
  unknown_status="$(post_status /v1/policy/apply "$unknown_payload" "$unknown_body")"
  [[ "$unknown_status" =~ ^(4|5)[0-9][0-9]$ ]] || die "unknown tenant policy status=$unknown_status"
  record "- valid policy accepted; unknown tenant rejected with HTTP ${unknown_status}; payload secret-key scan clean: PASS"
  record ""

  record "## S3 account register"
  local account_payload missing_payload raw_auth_payload reject_body reject_status
  account_payload="$(payload_accounts "$TENANT_ID" "/tmp/rpp-smoke/profiles/${PROFILE_ID}")"
  printf '%s' "$account_payload" >"${TMP_DIR}/accounts-valid.json"
  ensure_no_secret_keys "${TMP_DIR}/accounts-valid.json"
  curl_json POST /v1/accounts/register "$account_payload" | json_assert register
  missing_payload="$(payload_accounts "$TENANT_ID" "/tmp/rpp-smoke/missing-home")"
  curl_json POST /v1/accounts/register "$missing_payload" | json_assert reject
  raw_auth_payload="$(payload_accounts "$TENANT_ID" "/tmp/rpp-smoke/profiles/${PROFILE_ID}" raw_auth)"
  reject_body="${TMP_DIR}/raw-auth-register.json"
  reject_status="$(post_status /v1/accounts/register "$raw_auth_payload" "$reject_body")"
  if [[ "$reject_status" =~ ^2 ]]; then
    json_assert reject <"$reject_body"
  else
    [[ "$reject_status" =~ ^(4|5)[0-9][0-9]$ ]] || die "raw-auth status=$reject_status"
  fi
  record "- valid profile refs registered; missing home rejected; raw auth payload rejected: PASS"
  record ""

  record "## S4 session start/stop"
  local start_response runtime_session_id stop_payload_body
  start_response="$(curl_json POST /v1/session/start "$(payload_start "$SESSION_ID")")"
  runtime_session_id="$(printf '%s' "$start_response" | json_assert start)"
  stop_payload_body="$(payload_stop "$SESSION_ID" "$runtime_session_id")"
  curl_json POST /v1/session/stop "$stop_payload_body" | json_assert stop
  curl_json POST /v1/session/stop "$stop_payload_body" | json_assert stop
  validate_events "$SESSION_ID" 2 session_started
  record "- start returned router_owner=rust_l2; stop idempotent; event stream emitted scrubbed events: PASS"
  record ""

  record "## S5 kill switch"
  local kill_payload kill_status_response kill_start_response
  kill_payload="$(payload_kill)"
  curl_json POST /v1/killswitch/apply "$kill_payload" | json_assert kill
  kill_status_response="$(curl_json GET "/v1/killswitch/status?tenant_id=${TENANT_ID}&feature=smart_context&provider=codex&profile_id=${PROFILE_ID}")"
  printf '%s' "$kill_status_response" | json_assert kill_status
  kill_start_response="$(curl_json POST /v1/session/start "$(payload_start "$KILL_SESSION_ID")")"
  printf '%s' "$kill_start_response" | json_assert start_exact
  validate_events "$TENANT_ID" 1 kill_switch_applied
  record "- tenant smart_context kill-switch active; next session reports smart_context_mode=exact; kill-switch event emitted: PASS"
  record ""

  record "## Result"
  record ""
  record "S1-S5 PASS against local rpp.l2.v1 sidecar, with direct bin/prodex version, doctor, and capability verification."
  log "PASS evidence=${EVIDENCE_FILE}"
}

main "$@"
