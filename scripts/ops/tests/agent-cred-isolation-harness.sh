#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT="${TEST_DIR}/../agent-cred-isolation.sh"
TMP_DIR="$(mktemp -d)"

cleanup() {
  if [[ -n "${LIVE_PID:-}" ]]; then
    kill "${LIVE_PID}" 2>/dev/null || true
    wait "${LIVE_PID}" 2>/dev/null || true
  fi
  rm -rf -- "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'agent-cred-isolation-harness: FAIL: %s\n' "$1" >&2
  exit 1
}

assert_equal() {
  [[ "$1" == "$2" ]] || fail "value-mismatch: got=$1 want=$2"
}

assert_file() {
  [[ -f "$1" && ! -L "$1" ]] || fail "expected-regular-file:$1"
}

assert_dir() {
  [[ -d "$1" && ! -L "$1" ]] || fail "expected-directory:$1"
}

assert_absent() {
  [[ ! -e "$1" && ! -L "$1" ]] || fail "expected-absent:$1"
}

assert_content() {
  assert_file "$1"
  assert_equal "$(<"$1")" "$2"
}

slot_count() {
  local root="$1"
  find "${root}/slots" -mindepth 1 -maxdepth 1 -type d -name 'slot-*' -printf '.' | wc -c
}

chmod 700 "${TMP_DIR}"
FIXTURE_ROOT="${TMP_DIR}/fixture"
HOST_HOME="${FIXTURE_ROOT}/host-home"
HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share"
HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config"
STATE_ROOT="${FIXTURE_ROOT}/state"
mkdir -p \
  "${HOST_HOME}/.codex" \
  "${HOST_HOME}/.cline/data/settings" \
  "${HOST_HOME}/.gemini/antigravity-cli" \
  "${HOST_XDG_DATA_HOME}/kiro-cli" \
  "${HOST_XDG_DATA_HOME}/opencode" \
  "${HOST_XDG_DATA_HOME}/glm" \
  "${HOST_XDG_CONFIG_HOME}/opencode" \
  "${HOST_XDG_CONFIG_HOME}/glm"
chmod 700 "${FIXTURE_ROOT}" "${HOST_HOME}"

printf 'legacy-codex\n' >"${HOST_HOME}/.codex/auth.json"
printf 'legacy-codex-config\n' >"${HOST_HOME}/.codex/config.toml"
printf 'legacy-cline\n' >"${HOST_HOME}/.cline/data/settings/providers.json"
printf 'legacy-agy\n' >"${HOST_HOME}/.gemini/antigravity-cli/antigravity-oauth-token"
printf 'legacy-kiro\n' >"${HOST_XDG_DATA_HOME}/kiro-cli/data.sqlite3"
printf 'legacy-kiro-wal\n' >"${HOST_XDG_DATA_HOME}/kiro-cli/data.sqlite3-wal"
printf 'legacy-opencode\n' >"${HOST_XDG_DATA_HOME}/opencode/auth.json"
printf 'legacy-opencode-config\n' >"${HOST_XDG_CONFIG_HOME}/opencode/opencode.json"
printf 'legacy-glm\n' >"${HOST_XDG_DATA_HOME}/glm/auth.json"
printf 'legacy-glm-config\n' >"${HOST_XDG_CONFIG_HOME}/glm/config.json"

UUID_A='11111111-1111-4111-8111-111111111111'
UUID_B='22222222-2222-4222-8222-222222222222'
UUID_C='33333333-3333-4333-8333-333333333333'
FP_A='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
FP_B='bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
FP_C='cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'
FP_D='dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd'

run_binding() {
  local agent_id="$1"
  local fingerprint="$2"
  local command="$3"
  local state_root="${4:-${STATE_ROOT}}"
  local adopt_slot="${5:-}"
  local reconcile="${6:-0}"

  (
    unset AGENT_CRED_ISOLATION_SCRIPT_LOADED AGENT_CRED_ISOLATION_REGISTRY \
      AGENT_CRED_ISOLATION_IDENTITY AGENT_CRED_ISOLATION_SLOT \
      AGENT_CRED_ISOLATION_SLOT_NAME AGENT_CRED_ISOLATION_SLOT_ROOT \
      CODEX_HOME CLINE_DATA_DIR CLINE_SANDBOX CLINE_SANDBOX_DATA_DIR
    export HOME="${HOST_HOME}"
    export XDG_DATA_HOME="${HOST_XDG_DATA_HOME}"
    export XDG_CONFIG_HOME="${HOST_XDG_CONFIG_HOME}"
    export AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}"
    export AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_XDG_DATA_HOME}"
    export AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_XDG_CONFIG_HOME}"
    export AGENT_CRED_ISOLATION_ROOT="${state_root}"
    export AGENT_CRED_ISOLATION_AGENT_ID="${agent_id}"
    export AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT="${fingerprint}"
    export AGENT_CRED_ISOLATION_RECONCILE_STALE="${reconcile}"
    export AGENT_CRED_ISOLATION_TEST_ALLOW_UNREADABLE_PROC=1
    if [[ -n "${adopt_slot}" ]]; then
      export AGENT_CRED_ISOLATION_ADOPT_SLOT="${adopt_slot}"
    else
      unset AGENT_CRED_ISOLATION_ADOPT_SLOT
    fi
    export AGENT_CRED_ISOLATION_AUTOSTART=1
    source "${SCRIPT}" || exit $?
    eval "${command}"
  )
}

capture_env() {
  local agent_id="$1"
  local fingerprint="$2"
  local output="$3"
  local state_root="${4:-${STATE_ROOT}}"
  local adopt_slot="${5:-}"
  local reconcile="${6:-0}"
  run_binding "${agent_id}" "${fingerprint}" \
    'printf "%s|%s|%s|%s|%s|%s|%s|%s|%s\n" "$AGENT_CRED_ISOLATION_SLOT" "$AGENT_CRED_ISOLATION_SLOT_NAME" "$AGENT_CRED_ISOLATION_SLOT_ROOT" "$AGENT_CRED_ISOLATION_IDENTITY" "$CODEX_HOME" "$CLINE_DATA_DIR" "$HOME" "$XDG_DATA_HOME" "$XDG_CONFIG_HOME"' \
    "${state_root}" "${adopt_slot}" "${reconcile}" >"${output}"
}

# First stable binding receives one physical home seeded from synthetic stores.
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/a.env"
IFS='|' read -r slot_a slot_name_a root_a identity_a codex_a cline_a home_a data_a config_a <"${TMP_DIR}/a.env"
assert_equal "${slot_a}" '1'
assert_equal "${slot_name_a}" 'slot-01'
assert_equal "${root_a}" "${STATE_ROOT}/slots/slot-01"
[[ "${identity_a}" == sha256:* ]] || fail 'identity-key-format'
assert_content "${codex_a}/auth.json" 'legacy-codex'
assert_content "${cline_a}/data/settings/providers.json" 'legacy-cline'
assert_content "${home_a}/.gemini/antigravity-cli/antigravity-oauth-token" 'legacy-agy'
assert_content "${data_a}/kiro-cli/data.sqlite3" 'legacy-kiro'
assert_content "${data_a}/kiro-cli/data.sqlite3-wal" 'legacy-kiro-wal'
assert_content "${data_a}/opencode/auth.json" 'legacy-opencode'
assert_content "${config_a}/opencode/opencode.json" 'legacy-opencode-config'
assert_content "${data_a}/glm/auth.json" 'legacy-glm'
assert_content "${config_a}/glm/config.json" 'legacy-glm-config'

# Restart/relogin keeps the same binding, folder, and live state.
printf 'account-A-marker\n' >"${codex_a}/auth.json"
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/a-restart.env"
IFS='|' read -r slot_a2 slot_name_a2 root_a2 identity_a2 codex_a2 _ _ _ _ <"${TMP_DIR}/a-restart.env"
assert_equal "${slot_a2}" "${slot_a}"
assert_equal "${slot_name_a2}" "${slot_name_a}"
assert_equal "${root_a2}" "${root_a}"
assert_equal "${identity_a2}" "${identity_a}"
assert_equal "${codex_a2}" "${codex_a}"
assert_content "${codex_a2}/auth.json" 'account-A-marker'
assert_equal "$(slot_count "${STATE_ROOT}")" '1'

# A different stable binding gets a distinct home and cannot overwrite A.
capture_env "${UUID_B}" "${FP_B}" "${TMP_DIR}/b.env"
IFS='|' read -r slot_b slot_name_b root_b identity_b codex_b _ _ _ _ <"${TMP_DIR}/b.env"
assert_equal "${slot_b}" '2'
assert_equal "${slot_name_b}" 'slot-02'
[[ "${identity_b}" != "${identity_a}" ]] || fail 'different-bindings-shared-identity'
printf 'account-B-marker\n' >"${codex_b}/auth.json"
assert_content "${codex_a}/auth.json" 'account-A-marker'
assert_content "${codex_b}/auth.json" 'account-B-marker'
assert_content "${HOST_HOME}/.codex/auth.json" 'legacy-codex'

# Missing and malformed stable identity fail before creating any allocation root.
expect_identity_failure() {
  local label="$1"
  local agent_id="$2"
  local fingerprint="$3"
  local failure_root="${FIXTURE_ROOT}/failure-${label}"
  local rc
  set +e
  run_binding "${agent_id}" "${fingerprint}" ':' "${failure_root}" \
    >"${TMP_DIR}/${label}.stdout" 2>"${TMP_DIR}/${label}.stderr"
  rc=$?
  set -e
  (( rc != 0 )) || fail "identity-failure-accepted:${label}"
  assert_absent "${failure_root}"
}
expect_identity_failure missing-agent '' "${FP_A}"
expect_identity_failure missing-fingerprint "${UUID_A}" ''
expect_identity_failure malformed-agent 'not-a-uuid' "${FP_A}"
expect_identity_failure nil-agent '00000000-0000-0000-0000-000000000000' "${FP_A}"
expect_identity_failure uppercase-fingerprint "${UUID_A}" "${FP_A^^}"
expect_identity_failure short-fingerprint "${UUID_A}" 'abcd'

# Real process concurrency for one stable binding converges to one slot/folder.
pids=()
for number in 1 2 3 4 5 6 7 8; do
  capture_env "${UUID_C}" "${FP_C}" "${TMP_DIR}/concurrent-${number}.env" &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done
for number in 1 2 3 4 5 6 7 8; do
  IFS='|' read -r concurrent_slot _ concurrent_root _ _ _ _ _ _ <"${TMP_DIR}/concurrent-${number}.env"
  assert_equal "${concurrent_slot}" '3'
  assert_equal "${concurrent_root}" "${STATE_ROOT}/slots/slot-03"
done
assert_equal "$(slot_count "${STATE_ROOT}")" '3'

python3 - "${STATE_ROOT}/registry.json" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as source:
    registry = json.load(source)
if registry.get("version") != 2:
    raise SystemExit("registry version is not 2")
if len(registry["bindings"]) != 3 or len(registry["slots"]) != 3:
    raise SystemExit("expected exactly three stable bindings and slots")
if sorted(int(value) for value in registry["slots"]) != [1, 2, 3]:
    raise SystemExit("unexpected slot set")
for key, entry in registry["bindings"].items():
    if entry["identity"] != key or registry["slots"][str(entry["slot"])]["identity"] != key:
        raise SystemExit("registry ownership mismatch")
PY

# Missing physical folders for non-current bindings are stale metadata only and
# are pruned without allocating an extra folder.
python3 - "${STATE_ROOT}/registry.json" <<'PY'
import json
import os
import sys
path = sys.argv[1]
with open(path, "r", encoding="utf-8") as source:
    registry = json.load(source)
key = "sha256:" + "9" * 64
registry["bindings"][key] = {
    "identity": key,
    "agent_id": "99999999-9999-4999-8999-999999999999",
    "subscription_fingerprint": "9" * 64,
    "slot": 99,
    "first_seen": "synthetic",
    "last_seen": "synthetic",
}
registry["slots"]["99"] = {"identity": key, "created_at": "synthetic"}
tmp = path + ".test"
with open(tmp, "w", encoding="utf-8") as target:
    json.dump(registry, target)
os.replace(tmp, path)
PY
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/stale-metadata.env"
python3 - "${STATE_ROOT}/registry.json" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as source:
    registry = json.load(source)
if "99" in registry["slots"] or any(entry["slot"] == 99 for entry in registry["bindings"].values()):
    raise SystemExit("stale missing-folder metadata was not pruned")
PY
assert_equal "$(slot_count "${STATE_ROOT}")" '3'

# An unowned physical folder requires explicit reconciliation. A live process
# reference blocks deletion; after it exits, reconciliation removes only stale 77.
mkdir -p "${STATE_ROOT}/slots/slot-77/home"
printf 'stale-folder-marker\n' >"${STATE_ROOT}/slots/slot-77/synthetic-marker"
env HOME="${STATE_ROOT}/slots/slot-77/home" \
  AGENT_CRED_ISOLATION_SLOT_ROOT="${STATE_ROOT}/slots/slot-77" \
  sleep 30 &
LIVE_PID=$!
sleep 0.1
set +e
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/live-reconcile.env" "${STATE_ROOT}" '' 1 \
  >"${TMP_DIR}/live-reconcile.stdout" 2>"${TMP_DIR}/live-reconcile.stderr"
rc=$?
set -e
(( rc != 0 )) || fail 'live-referenced-stale-slot-was-reconciled'
assert_dir "${STATE_ROOT}/slots/slot-77"
assert_content "${STATE_ROOT}/slots/slot-77/synthetic-marker" 'stale-folder-marker'
kill "${LIVE_PID}"
wait "${LIVE_PID}" 2>/dev/null || true
unset LIVE_PID
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/reconcile.env" "${STATE_ROOT}" '' 1
assert_absent "${STATE_ROOT}/slots/slot-77"
assert_content "${codex_a}/auth.json" 'account-A-marker'
assert_equal "$(slot_count "${STATE_ROOT}")" '3'

# A subscription change for the same agent cannot allocate a second folder. A
# live reference to the established home blocks even explicit reconciliation.
env HOME="${root_a}/home" AGENT_CRED_ISOLATION_SLOT_ROOT="${root_a}" sleep 30 &
LIVE_PID=$!
sleep 0.1
before_count="$(slot_count "${STATE_ROOT}")"
set +e
capture_env "${UUID_A}" "${FP_D}" "${TMP_DIR}/changed-subscription.env" "${STATE_ROOT}" '' 1 \
  >"${TMP_DIR}/changed-subscription.stdout" 2>"${TMP_DIR}/changed-subscription.stderr"
rc=$?
set -e
(( rc != 0 )) || fail 'live-active-binding-was-replaced'
assert_equal "$(slot_count "${STATE_ROOT}")" "${before_count}"
assert_content "${codex_a}/auth.json" 'account-A-marker'
kill "${LIVE_PID}"
wait "${LIVE_PID}" 2>/dev/null || true
unset LIVE_PID

# Explicit migration of the sole legacy physical slot preserves slot-185 and its
# contents, discards terminal metadata, and creates no historical folder.
ADOPT_ROOT="${FIXTURE_ROOT}/adopt-state"
mkdir -p "${ADOPT_ROOT}/slots/slot-185"
printf 'preserve-slot-185\n' >"${ADOPT_ROOT}/slots/slot-185/synthetic-preserve-marker"
cat >"${ADOPT_ROOT}/registry.json" <<'JSON'
{
  "version": 1,
  "next_slot": 186,
  "terminals": {
    "herdr:obsolete-terminal": {"slot": 185, "first_seen": "old", "last_seen": "old"}
  },
  "slots": {
    "185": {"terminal_id": "herdr:obsolete-terminal", "created_at": "old"}
  }
}
JSON
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/adopt.env" "${ADOPT_ROOT}" '185'
IFS='|' read -r adopt_slot adopt_name adopt_home adopt_identity _ _ _ _ _ <"${TMP_DIR}/adopt.env"
assert_equal "${adopt_slot}" '185'
assert_equal "${adopt_name}" 'slot-185'
assert_equal "${adopt_home}" "${ADOPT_ROOT}/slots/slot-185"
[[ "${adopt_identity}" == sha256:* ]] || fail 'adopted-identity-format'
assert_content "${ADOPT_ROOT}/slots/slot-185/synthetic-preserve-marker" 'preserve-slot-185'
assert_equal "$(slot_count "${ADOPT_ROOT}")" '1'
capture_env "${UUID_A}" "${FP_A}" "${TMP_DIR}/adopt-restart.env" "${ADOPT_ROOT}"
IFS='|' read -r adopt_slot_2 _ adopt_home_2 _ _ _ _ _ _ <"${TMP_DIR}/adopt-restart.env"
assert_equal "${adopt_slot_2}" '185'
assert_equal "${adopt_home_2}" "${ADOPT_ROOT}/slots/slot-185"
assert_content "${ADOPT_ROOT}/slots/slot-185/synthetic-preserve-marker" 'preserve-slot-185'

python3 - "${ADOPT_ROOT}/registry.json" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as source:
    registry = json.load(source)
if registry.get("version") != 2:
    raise SystemExit("legacy adoption did not produce registry v2")
if set(registry["slots"]) != {"185"} or len(registry["bindings"]) != 1:
    raise SystemExit("legacy adoption retained historical mappings")
text = json.dumps(registry)
for forbidden in ("terminal", "herdr", "pane", "tty", "process"):
    if forbidden in text.lower():
        raise SystemExit(f"legacy identity term retained: {forbidden}")
PY

status="$(run_binding "${UUID_A}" "${FP_A}" "'${SCRIPT}' doctor status")"
[[ "${status}" == *"agent=${UUID_A} identity=sha256:"*" slot=1"* ]] || fail 'doctor-stable-binding-status'
[[ "${status}" != *'terminal='* && "${status}" != *'pane='* && "${status}" != *'herdr'* ]] || fail 'doctor-reported-terminal-identity'
for vendor in codex cline agy kiro opencode glm; do
  [[ "${status}" == *"vendor=${vendor} account=slot-01 state=on"* ]] || fail "doctor-vendor-state:${vendor}"
done

# Source no longer contains any forbidden fallback mechanism.
if grep -Eiq 'HERDR_PANE_ID|command -v herdr|herdr pane|get.*terminal_id|tty 2>|BASHPID|PPID|uuid4|fallback-terminals' "${SCRIPT}"; then
  fail 'forbidden-terminal-or-random-fallback-remains'
fi

printf 'PASS: stable agent/subscription identity, fail-closed validation, restart/relogin, concurrency, stale reconciliation, live-reference protection, and slot-185 adoption\n'
