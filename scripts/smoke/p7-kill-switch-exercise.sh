#!/usr/bin/env bash
# P7 exercise: prove kill-switch scope and feature behavior against a real
# local rpp.l2.v1 sidecar process.

set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_NAME="p7-kill-switch-exercise"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
SIDECAR="${P7_SIDECAR_PATH:-${REPO_ROOT}/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar}"
TOKEN="${P7_L2_BEARER_TOKEN:-p7-smoke-token}"
TENANT_ID="${P7_TENANT_ID:-tenant-p7-smoke}"
PROVIDER="${P7_PROVIDER:-codex}"
PROFILE_ID="${P7_PROFILE_ID:-codex-smoke-main}"
TMP_DIR=""
SIDECAR_PID=""

log() { printf '[%s] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

cleanup() {
  if [[ -n "$SIDECAR_PID" ]] && kill -0 "$SIDECAR_PID" 2>/dev/null; then
    kill "$SIDECAR_PID" 2>/dev/null || true
    wait "$SIDECAR_PID" 2>/dev/null || true
  fi
  [[ -n "$TMP_DIR" ]] && rm -rf "$TMP_DIR"
}

free_port() {
  python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

wait_ready() {
  local base_url="$1"
  for _ in $(seq 1 50); do
    if curl -fsS --max-time 1 -H "Authorization: Bearer ${TOKEN}" "${base_url}/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

post_kill() {
  local base_url="$1"
  local feature="$2"
  local state="$3"
  local effective_at="$4"
  local scope_kind="$5"
  python3 - "$TENANT_ID" "$PROVIDER" "$PROFILE_ID" "$feature" "$state" "$effective_at" "$scope_kind" <<'PY' |
import json
import sys

tenant_id, provider, profile_id, feature, state, effective_at, scope_kind = sys.argv[1:8]
scope = {}
if scope_kind in {"provider", "profile"}:
    scope["provider"] = provider
if scope_kind == "profile":
    scope["profile_id"] = profile_id
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_p7_{scope_kind}_{feature}_{state}",
    "tenant_id": tenant_id,
    "scope": scope,
    "feature": feature,
    "state": state,
    "reason": "p7_smoke",
    "effective_at": effective_at,
}, separators=(",", ":")))
PY
  curl -fsS --max-time 5 \
    -X POST "${base_url}/v1/killswitch/apply" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    --data-binary @- |
  python3 -c '
import json
import sys
data = json.load(sys.stdin)
if data.get("contract_version") != "rpp.l2.v1" or data.get("applied") is not True:
    raise SystemExit(f"unexpected kill-switch response: {data}")
'
}

assert_status() {
  local base_url="$1"
  local feature="$2"
  local scope_kind="$3"
  local expected="$4"
  local query="tenant_id=${TENANT_ID}&feature=${feature}"
  if [[ "$scope_kind" == "provider" || "$scope_kind" == "profile" ]]; then
    query="${query}&provider=${PROVIDER}"
  fi
  if [[ "$scope_kind" == "profile" ]]; then
    query="${query}&profile_id=${PROFILE_ID}"
  fi
  curl -fsS --max-time 5 \
    -H "Authorization: Bearer ${TOKEN}" \
    "${base_url}/v1/killswitch/status?${query}" |
  python3 -c '
import json
import sys
expected = sys.argv[1] == "true"
data = json.load(sys.stdin)
if data.get("contract_version") != "rpp.l2.v1" or data.get("active") is not expected:
    raise SystemExit(f"kill-switch active={data.get('active')} want {expected}: {data}")
  ' "$expected"
}

session_payload() {
  local session_id="$1"
  python3 - "$TENANT_ID" "$PROVIDER" "$PROFILE_ID" "$session_id" <<'PY'
import json
import sys
tenant_id, provider, profile_id, session_id = sys.argv[1:5]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req_start_{session_id}",
    "tenant_id": tenant_id,
    "workspace_id": "workspace-p7-smoke",
    "task_id": f"task-{session_id}",
    "session_id": session_id,
    "policy_id": "policy-p7-smoke",
    "requested_provider": provider,
    "requested_model": "gpt-5",
    "working_directory": "/tmp/rpp-smoke-workspace",
    "profile_pool": [profile_id],
}, separators=(",", ":")))
PY
}

assert_session() {
  local base_url="$1"
  local session_id="$2"
  local expected="$3"
  local body_file="${TMP_DIR}/${session_id}.json"
  local status
  status="$(session_payload "$session_id" |
    curl -sS --max-time 5 \
      -X POST "${base_url}/v1/session/start" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary @- \
      -o "$body_file" \
      -w '%{http_code}')"
  if [[ "$expected" == "blocked" ]]; then
    [[ "$status" == "423" ]] || die "session ${session_id} status=${status}, want 423; body=$(cat "$body_file")"
    return 0
  fi
  [[ "$status" == "200" ]] || die "session ${session_id} status=${status}, want 200; body=$(cat "$body_file")"
  python3 - "$body_file" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
if data.get("contract_version") != "rpp.l2.v1" or data.get("router_owner") != "rust_l2":
    raise SystemExit(f"unexpected session response: {data}")
PY
}

main() {
  command -v curl >/dev/null 2>&1 || die "curl is required"
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  [[ -x "$SIDECAR" ]] || die "sidecar not executable: $SIDECAR"

  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  local port base_url
  port="$(free_port)"
  base_url="http://127.0.0.1:${port}"
  MULTICA_L2_BEARER_TOKEN="$TOKEN" "$SIDECAR" "127.0.0.1:${port}" >"${TMP_DIR}/sidecar.log" 2>&1 &
  SIDECAR_PID="$!"
  wait_ready "$base_url" || die "sidecar did not become ready; log=$(cat "${TMP_DIR}/sidecar.log")"

  post_kill "$base_url" smart_context disabled next_request tenant
  assert_status "$base_url" smart_context tenant true
  post_kill "$base_url" smart_context enabled next_request tenant
  assert_status "$base_url" smart_context tenant false

  post_kill "$base_url" gateway disabled immediate provider
  assert_status "$base_url" gateway provider true
  assert_session "$base_url" session-provider-blocked blocked
  post_kill "$base_url" gateway enabled immediate provider
  assert_status "$base_url" gateway provider false
  assert_session "$base_url" session-provider-resumed pass

  post_kill "$base_url" runtime_proxy disabled immediate profile
  assert_status "$base_url" runtime_proxy profile true
  assert_session "$base_url" session-profile-blocked blocked
  post_kill "$base_url" runtime_proxy enabled immediate profile
  assert_status "$base_url" runtime_proxy profile false
  assert_session "$base_url" session-profile-resumed pass

  post_kill "$base_url" auto_redeem disabled immediate tenant
  assert_status "$base_url" auto_redeem tenant true
  post_kill "$base_url" auto_redeem enabled immediate tenant
  assert_status "$base_url" auto_redeem tenant false

  log "PASS tenant/provider/profile scopes; smart_context/gateway/auto_redeem features; disable and resume behavior"
}

main "$@"
