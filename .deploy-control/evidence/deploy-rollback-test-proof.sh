#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd -P)"
OUT="${SCRIPT_DIR}/deploy-rollback-test-proof.md"
SIDECAR="${REPO_ROOT}/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar"
ROLLBACK="${REPO_ROOT}/scripts/deploy/rollback-to-raw-codex.sh"
PRODEX_BIN="$(command -v prodex || true)"
CODEX_BIN="$(command -v codex || true)"
TOKEN="deploy-rollback-proof-token"
RUN_ID="deploy-rollback-$(date -u +%Y%m%dT%H%M%SZ)"
TMP_DIR=""
SIDECAR_PID=""
GATEWAY_PID=""
BASE_URL=""
GATEWAY_ADDR=""
FAILURES=0

redact() {
  sed -E \
    -e 's/(Authorization: Bearer )[A-Za-z0-9._~+\/=-]+/\1<redacted>/g' \
    -e 's/(Bearer )[A-Za-z0-9._~+\/=-]+/\1<redacted>/g' \
    -e 's/(TOKEN=)[^[:space:]]+/\1<redacted>/g' \
    -e 's/(token[=:] ?)[^", ]+/\1<redacted>/Ig' \
    -e 's/(api[_-]?key[=:] ?)[^", ]+/\1<redacted>/Ig' \
    -e 's/sk-[A-Za-z0-9._-]+/sk-<redacted>/g'
}

append() {
  printf '%s\n' "$*" >> "$OUT"
}

append_code() {
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

free_port() {
  python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

json_get() {
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

assert_eq() {
  local label="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" == "$want" ]]; then
    append "- PASS ${label}: ${got}"
  else
    append "- FAIL ${label}: got '${got}', want '${want}'"
    FAILURES=$((FAILURES + 1))
  fi
}

assert_ne() {
  local label="$1"
  local got="$2"
  local bad="$3"
  if [[ "$got" != "$bad" && -n "$got" ]]; then
    append "- PASS ${label}: ${got}"
  else
    append "- FAIL ${label}: got '${got}', disallowed '${bad}'"
    FAILURES=$((FAILURES + 1))
  fi
}

cleanup() {
  if [[ -n "$SIDECAR_PID" ]] && kill -0 "$SIDECAR_PID" 2>/dev/null; then
    kill "$SIDECAR_PID" 2>/dev/null || true
    wait "$SIDECAR_PID" 2>/dev/null || true
  fi
  if [[ -n "$GATEWAY_PID" ]] && kill -0 "$GATEWAY_PID" 2>/dev/null; then
    kill "$GATEWAY_PID" 2>/dev/null || true
    wait "$GATEWAY_PID" 2>/dev/null || true
  fi
  [[ -n "$TMP_DIR" ]] && rm -rf "$TMP_DIR"
}

init() {
  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  cat > "$OUT" <<EOF
# Deploy-Rollback Spec Test Proof

- timestamp_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- runner: Codex#5.5#B
- task: prove deploy-rollback kill-switch and rollback by test
- spec: openspec/changes/rotation-parity-polyglot/specs/deploy-rollback/spec.md
- sidecar: ${SIDECAR}
- prodex_bin: ${PRODEX_BIN:-<missing>}
- codex_bin: ${CODEX_BIN:-<missing>}
- temp_root: ${TMP_DIR}
- secrets_present: false

## Provenance

\`\`\`text
$ hostname
$(hostname)

$ git rev-parse --short HEAD
$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || printf unknown)

$ prodex --version
$("$PRODEX_BIN" --version 2>/dev/null || true)

$ codex --version
$("$CODEX_BIN" --version 2>/dev/null || true)

$ sha256sum multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
$(sha256sum "$SIDECAR" 2>/dev/null || true)
\`\`\`
EOF
}

preflight() {
  append ""
  append "## Preflight"
  [[ -x "$SIDECAR" ]] || { append "- FAIL sidecar executable missing: ${SIDECAR}"; FAILURES=$((FAILURES + 1)); }
  [[ -x "$PRODEX_BIN" ]] || { append "- FAIL prodex binary missing"; FAILURES=$((FAILURES + 1)); }
  [[ -x "$CODEX_BIN" ]] || { append "- FAIL codex binary missing"; FAILURES=$((FAILURES + 1)); }
  [[ -r "$ROLLBACK" ]] || { append "- FAIL rollback script missing: ${ROLLBACK}"; FAILURES=$((FAILURES + 1)); }

  local prodex_version
  prodex_version="$("$PRODEX_BIN" --version 2>/dev/null || true)"
  assert_eq "pinned prodex version" "$prodex_version" "prodex 0.246.0"
}

start_sidecar() {
  local sidecar_port gateway_port log_file
  sidecar_port="$(free_port)"
  gateway_port="$(free_port)"
  BASE_URL="http://127.0.0.1:${sidecar_port}"
  GATEWAY_ADDR="127.0.0.1:${gateway_port}"
  log_file="${TMP_DIR}/sidecar.log"

  append ""
  append "## Isolated Sidecar Start"
  append '```text'
  {
    printf '$ PRODEX_HOME=<tmp>/prodex-home CODEX_HOME=<tmp>/codex-home MULTICA_PRODEX_PATH=%q PRODEX_GATEWAY_LISTEN=%q MULTICA_L2_BEARER_TOKEN=<redacted> %q %q\n' \
      "$PRODEX_BIN" "$GATEWAY_ADDR" "$SIDECAR" "127.0.0.1:${sidecar_port}"
  } >> "$OUT"
  append '```'

  PRODEX_HOME="${TMP_DIR}/prodex-home" \
  CODEX_HOME="${TMP_DIR}/codex-home" \
  MULTICA_PRODEX_PATH="$PRODEX_BIN" \
  PRODEX_GATEWAY_LISTEN="$GATEWAY_ADDR" \
  PRODEX_GATEWAY_UPSTREAM_BASE_URL="http://127.0.0.1:9" \
  MULTICA_L2_BEARER_TOKEN="$TOKEN" \
  "$SIDECAR" "127.0.0.1:${sidecar_port}" >"$log_file" 2>&1 &
  SIDECAR_PID="$!"

  for _ in $(seq 1 80); do
    if curl -fsS --max-time 1 -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/healthz" >"${TMP_DIR}/healthz.json" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done

  curl -sS --max-time 2 -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/readyz" >"${TMP_DIR}/readyz.json" 2>/dev/null || true
  append_code "healthz after start" "${TMP_DIR}/healthz.json"
  append_code "readyz observation after start" "${TMP_DIR}/readyz.json"
  GATEWAY_PID="$(json_get "${TMP_DIR}/readyz.json" "checks.2.details.pid")"
  assert_eq "healthz status" "$(json_get "${TMP_DIR}/healthz.json" "status")" "alive"
}

session_payload() {
  local tenant="$1"
  local provider="$2"
  local profile="$3"
  local session="$4"
  python3 - "$tenant" "$provider" "$profile" "$session" <<'PY'
import json
import sys
tenant, provider, profile, session = sys.argv[1:5]
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req-{session}",
    "tenant_id": tenant,
    "workspace_id": "workspace-deploy-rollback-proof",
    "task_id": f"task-{session}",
    "session_id": session,
    "policy_id": "policy-deploy-rollback-proof",
    "requested_provider": provider,
    "requested_model": "gpt-5",
    "working_directory": "/tmp/deploy-rollback-proof-workspace",
    "profile_pool": [profile],
}, separators=(",", ":")))
PY
}

kill_payload() {
  local tenant="$1"
  local provider="$2"
  local profile="$3"
  local feature="$4"
  local state="$5"
  local scope_kind="$6"
  python3 - "$tenant" "$provider" "$profile" "$feature" "$state" "$scope_kind" "$RUN_ID" <<'PY'
import json
import sys
tenant, provider, profile, feature, state, scope_kind, run_id = sys.argv[1:8]
scope = {}
if scope_kind in {"provider", "profile"}:
    scope["provider"] = provider
if scope_kind == "profile":
    scope["profile_id"] = profile
print(json.dumps({
    "contract_version": "rpp.l2.v1",
    "request_id": f"req-{run_id}-{scope_kind}-{feature}-{state}",
    "tenant_id": tenant,
    "scope": scope,
    "feature": feature,
    "state": state,
    "reason": "deploy_rollback_spec_test",
    "effective_at": "immediate",
}, separators=(",", ":")))
PY
}

curl_json() {
  local method="$1"
  local path="$2"
  local body_file="${3:-}"
  local label="$4"
  local out_body="${TMP_DIR}/${label}.body"
  local out_meta="${TMP_DIR}/${label}.meta"
  local status

  {
    printf '$ curl -sS --max-time 8 -X %s %q -H %q' "$method" "${BASE_URL%/}${path}" "Authorization: Bearer <redacted>"
    [[ -n "$body_file" ]] && printf ' -H %q --data-binary @%s' "Content-Type: application/json" "$(basename "$body_file")"
    printf '\n'
  } > "$out_meta"

  if [[ -n "$body_file" ]]; then
    status="$(curl -sS --max-time 8 -X "$method" "${BASE_URL%/}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      --data-binary @"$body_file" \
      -o "$out_body" -w '%{http_code}' 2>>"$out_meta" || true)"
  else
    status="$(curl -sS --max-time 8 -X "$method" "${BASE_URL%/}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Accept: application/json" \
      -o "$out_body" -w '%{http_code}' 2>>"$out_meta" || true)"
  fi
  printf 'http_status=%s\n' "${status:-curl_failed}" >> "$out_meta"

  append_code "${label} command/status" "$out_meta"
  append_code "${label} body" "$out_body"
  printf '%s' "${status:-curl_failed}" > "${TMP_DIR}/${label}.status"
}

post_session() {
  local tenant="$1"
  local provider="$2"
  local profile="$3"
  local session="$4"
  local label="$5"
  local payload="${TMP_DIR}/${label}.json"
  session_payload "$tenant" "$provider" "$profile" "$session" > "$payload"
  curl_json POST "/v1/session/start" "$payload" "$label"
}

post_kill() {
  local tenant="$1"
  local provider="$2"
  local profile="$3"
  local feature="$4"
  local state="$5"
  local scope_kind="$6"
  local label="$7"
  local payload="${TMP_DIR}/${label}.json"
  kill_payload "$tenant" "$provider" "$profile" "$feature" "$state" "$scope_kind" > "$payload"
  curl_json POST "/v1/killswitch/apply" "$payload" "$label"
}

get_status() {
  local tenant="$1"
  local provider="$2"
  local profile="$3"
  local feature="$4"
  local label="$5"
  local query="/v1/killswitch/status?tenant_id=${tenant}&feature=${feature}"
  [[ -n "$provider" ]] && query="${query}&provider=${provider}"
  [[ -n "$profile" ]] && query="${query}&profile_id=${profile}"
  curl_json GET "$query" "" "$label"
}

test_kill_switches() {
  local tenant="tenant-proof"
  local provider="codex"
  local other_provider="anthropic"
  local profile="profile-proof"
  local other_profile="profile-other"

  append ""
  append "## Kill-Switch Tests"

  append ""
  append "### 1. Tenant-scope smart_context disables Smart Context"
  post_session "$tenant" "$provider" "$profile" "baseline-smart-context" "ks1-baseline-session"
  assert_eq "baseline smart_context HTTP" "$(cat "${TMP_DIR}/ks1-baseline-session.status")" "200"
  assert_eq "baseline smart_context mode" "$(json_get "${TMP_DIR}/ks1-baseline-session.body" "smart_context_mode")" "proxy_rewrite"

  post_kill "$tenant" "" "" "smart_context" "disabled" "tenant" "ks1-disable-smart-context"
  assert_eq "tenant smart_context disable applied" "$(json_get "${TMP_DIR}/ks1-disable-smart-context.body" "applied")" "True"
  get_status "$tenant" "$provider" "$profile" "smart_context" "ks1-status-disabled"
  assert_eq "tenant smart_context active" "$(json_get "${TMP_DIR}/ks1-status-disabled.body" "active")" "True"
  post_session "$tenant" "$provider" "$profile" "after-smart-context-disabled" "ks1-after-disable-session"
  assert_eq "after tenant smart_context HTTP" "$(cat "${TMP_DIR}/ks1-after-disable-session.status")" "200"
  assert_eq "after tenant smart_context mode" "$(json_get "${TMP_DIR}/ks1-after-disable-session.body" "smart_context_mode")" "exact"

  post_kill "$tenant" "" "" "smart_context" "enabled" "tenant" "ks1-enable-smart-context"
  get_status "$tenant" "$provider" "$profile" "smart_context" "ks1-status-enabled"
  assert_eq "tenant smart_context inactive after enable" "$(json_get "${TMP_DIR}/ks1-status-enabled.body" "active")" "False"
  post_session "$tenant" "$provider" "$profile" "after-smart-context-enabled" "ks1-after-enable-session"
  assert_eq "after tenant smart_context restore HTTP" "$(cat "${TMP_DIR}/ks1-after-enable-session.status")" "200"
  assert_eq "after tenant smart_context restore mode" "$(json_get "${TMP_DIR}/ks1-after-enable-session.body" "smart_context_mode")" "proxy_rewrite"

  append ""
  append "### 2. Provider-scope gateway disables routing"
  post_session "$tenant" "$provider" "$profile" "baseline-gateway" "ks2-baseline-session"
  assert_eq "baseline gateway HTTP" "$(cat "${TMP_DIR}/ks2-baseline-session.status")" "200"
  post_kill "$tenant" "$provider" "" "gateway" "disabled" "provider" "ks2-disable-gateway"
  get_status "$tenant" "$provider" "$profile" "gateway" "ks2-status-disabled"
  assert_eq "provider gateway active" "$(json_get "${TMP_DIR}/ks2-status-disabled.body" "active")" "True"
  post_session "$tenant" "$provider" "$profile" "after-gateway-disabled" "ks2-after-disable-session"
  assert_eq "provider gateway blocked HTTP" "$(cat "${TMP_DIR}/ks2-after-disable-session.status")" "423"
  post_session "$tenant" "$other_provider" "$profile" "other-provider-not-blocked" "ks2-other-provider-session"
  assert_eq "other provider unaffected HTTP" "$(cat "${TMP_DIR}/ks2-other-provider-session.status")" "200"
  post_kill "$tenant" "$provider" "" "gateway" "enabled" "provider" "ks2-enable-gateway"
  get_status "$tenant" "$provider" "$profile" "gateway" "ks2-status-enabled"
  assert_eq "provider gateway inactive after enable" "$(json_get "${TMP_DIR}/ks2-status-enabled.body" "active")" "False"
  post_session "$tenant" "$provider" "$profile" "after-gateway-enabled" "ks2-after-enable-session"
  assert_eq "provider gateway restored HTTP" "$(cat "${TMP_DIR}/ks2-after-enable-session.status")" "200"

  append ""
  append "### 3. Profile-scope auto_redeem disables auto-redeem state"
  post_kill "$tenant" "$provider" "$profile" "auto_redeem" "disabled" "profile" "ks3-disable-auto-redeem"
  assert_eq "profile auto_redeem disable applied" "$(json_get "${TMP_DIR}/ks3-disable-auto-redeem.body" "applied")" "True"
  get_status "$tenant" "$provider" "$profile" "auto_redeem" "ks3-status-disabled"
  assert_eq "profile auto_redeem active" "$(json_get "${TMP_DIR}/ks3-status-disabled.body" "active")" "True"
  get_status "$tenant" "$provider" "$other_profile" "auto_redeem" "ks3-status-other-profile"
  assert_eq "other profile auto_redeem unaffected" "$(json_get "${TMP_DIR}/ks3-status-other-profile.body" "active")" "False"
  post_kill "$tenant" "$provider" "$profile" "auto_redeem" "enabled" "profile" "ks3-enable-auto-redeem"
  get_status "$tenant" "$provider" "$profile" "auto_redeem" "ks3-status-enabled"
  assert_eq "profile auto_redeem inactive after enable" "$(json_get "${TMP_DIR}/ks3-status-enabled.body" "active")" "False"

  append ""
  append "Auto-redeem implementation note: this sidecar exposes auto_redeem as kill-switch state and event only; session_block currently checks runtime_proxy/gateway/provider_bridge, while smart_context changes session mode. The test therefore verifies auto_redeem disable/restore through the status API and profile scope isolation."
}

test_rollback() {
  append ""
  append "## One-Command Rollback Test"

  local env_file="${TMP_DIR}/multica-runtime.env"
  local prodex_home="${TMP_DIR}/isolated-prodex-home"
  local codex_home="${TMP_DIR}/isolated-codex-home"
  mkdir -p "$prodex_home/profiles/profile-proof" "$codex_home"
  chmod 700 "$prodex_home" "$codex_home" || true

  cat > "$env_file" <<EOF
MULTICA_CODEX_PATH=${PRODEX_BIN}
MULTICA_PRODEX_ENABLED=1
MULTICA_PRODEX_PATH=${PRODEX_BIN}
MULTICA_PRODEX_VERSION=v0.246.0
MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144
PRODEX_HOME=${prodex_home}
CODEX_HOME=${codex_home}
MULTICA_L2_ENABLED=1
MULTICA_L2_BASE_URL=${BASE_URL}
MULTICA_L2_BEARER_TOKEN=${TOKEN}
MULTICA_L2_SIDECAR_ARGS=--isolated-profile profile-proof
EOF

  append_code "rollback before env" "$env_file"
  append ""
  append "Before rollback behavior:"
  append '```text'
  printf '$ "$MULTICA_CODEX_PATH" --version\n' >> "$OUT"
  "$PRODEX_BIN" --version | redact >> "$OUT"
  append '```'

  local rollback_out="${TMP_DIR}/rollback.out"
  append ""
  append "One command executed:"
  append '```text'
  printf '$ ROLLBACK_ALLOW_EXECUTE=1 ROLLBACK_TARGET_ENV=smoke bash %q --env-file %q --codex-path %q --execute\n' "$ROLLBACK" "$env_file" "$CODEX_BIN" | redact >> "$OUT"
  append '```'
  ROLLBACK_ALLOW_EXECUTE=1 ROLLBACK_TARGET_ENV=smoke \
    bash "$ROLLBACK" --env-file "$env_file" --codex-path "$CODEX_BIN" --execute >"$rollback_out" 2>&1
  append_code "rollback command output" "$rollback_out"
  append_code "rollback after env" "$env_file"

  local raw_out="${TMP_DIR}/raw-codex.out"
  "$CODEX_BIN" --version > "$raw_out" 2>&1
  append_code "raw codex behavior after rollback" "$raw_out"

  assert_eq "rollback MULTICA_CODEX_PATH" "$(awk -F= '$1=="MULTICA_CODEX_PATH"{print $2}' "$env_file")" "$CODEX_BIN"
  assert_eq "rollback MULTICA_PRODEX_ENABLED" "$(awk -F= '$1=="MULTICA_PRODEX_ENABLED"{print $2}' "$env_file")" "0"
  assert_eq "rollback MULTICA_L2_ENABLED" "$(awk -F= '$1=="MULTICA_L2_ENABLED"{print $2}' "$env_file")" "0"
  if grep -Eq '^(MULTICA_PRODEX_PATH|PRODEX_HOME|MULTICA_L2_BASE_URL|MULTICA_L2_BEARER_TOKEN)=' "$env_file"; then
    append "- FAIL rollback removed prodex/L2 keys: keys still present"
    FAILURES=$((FAILURES + 1))
  else
    append "- PASS rollback removed prodex/L2 routing keys"
  fi
  assert_ne "raw codex version after rollback" "$(cat "$raw_out")" "prodex 0.246.0"
}

main() {
  init
  preflight
  if ((FAILURES == 0)); then
    start_sidecar
    test_kill_switches
    test_rollback
  fi

  append ""
  append "## Verdict"
  if ((FAILURES == 0)); then
    append "PASS. Kill-switch and rollback behavior were proven by executable tests with before/after captures."
  else
    append "FAIL. ${FAILURES} assertion(s) failed."
  fi
  append ""
  append "## Scrub Check"
  append '```text'
  if grep -RniE 'sk-|bearer|api[_-]?key|token=' "$OUT" | grep -Ev '<redacted>|\\<redacted\\>' >/tmp/deploy-rollback-scrub.$$ 2>/dev/null; then
    cat /tmp/deploy-rollback-scrub.$$ >> "$OUT"
    rm -f /tmp/deploy-rollback-scrub.$$
    exit 1
  else
    printf '0 matches\n' >> "$OUT"
  fi
  append '```'

  if ((FAILURES == 0)); then
    exit 0
  fi
  exit 1
}

main "$@"
