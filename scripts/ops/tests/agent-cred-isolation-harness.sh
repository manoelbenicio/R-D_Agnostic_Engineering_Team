#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT="${TEST_DIR}/../agent-cred-isolation.sh"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'agent-cred-isolation-harness: FAIL: %s\n' "$*" >&2
  exit 1
}

assert_equal() {
  [[ "$1" == "$2" ]] || fail "got '$1', want '$2'"
}

assert_file() {
  [[ -f "$1" ]] || fail "missing regular file: $1"
}

assert_dir() {
  [[ -d "$1" ]] || fail "missing directory: $1"
}

assert_not_symlink() {
  [[ ! -L "$1" ]] || fail "credential remained a symlink: $1"
}

assert_content() {
  assert_file "$1"
  assert_equal "$(<"$1")" "$2"
}

HOST_HOME="${TMP_DIR}/host-home"
STATE_ROOT="${TMP_DIR}/state"
FAKE_BIN="${TMP_DIR}/bin"
LEGACY_AUTH="${TMP_DIR}/legacy-auth.json"

mkdir -p \
  "${HOST_HOME}/.codex" \
  "${HOST_HOME}/.cline/data/settings" \
  "${HOST_HOME}/.gemini/antigravity-cli" \
  "${HOST_HOME}/.local/share/kiro-cli" \
  "${HOST_HOME}/.local/share/opencode" \
  "${HOST_HOME}/.local/share/glm" \
  "${HOST_HOME}/.config/opencode" \
  "${HOST_HOME}/.config/glm" \
  "${FAKE_BIN}"

printf 'legacy-codex\n' >"${LEGACY_AUTH}"
ln -s "${LEGACY_AUTH}" "${HOST_HOME}/.codex/auth.json"
printf 'legacy-codex-config\n' >"${HOST_HOME}/.codex/config.toml"
printf 'legacy-cline\n' >"${HOST_HOME}/.cline/data/settings/providers.json"
printf 'legacy-agy\n' >"${HOST_HOME}/.gemini/antigravity-cli/antigravity-oauth-token"
printf 'legacy-kiro\n' >"${HOST_HOME}/.local/share/kiro-cli/data.sqlite3"
printf 'legacy-kiro-wal\n' >"${HOST_HOME}/.local/share/kiro-cli/data.sqlite3-wal"
printf 'legacy-opencode\n' >"${HOST_HOME}/.local/share/opencode/auth.json"
printf 'legacy-opencode-config\n' >"${HOST_HOME}/.config/opencode/opencode.json"
printf 'legacy-glm\n' >"${HOST_HOME}/.local/share/glm/auth.json"
printf 'legacy-glm-config\n' >"${HOST_HOME}/.config/glm/config.json"

cat >"${FAKE_BIN}/herdr" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
[[ "$1 $2" == "pane get" ]]
case "$3" in
  pane-a|pane-a-recompact) terminal_id='terminal-A' ;;
  pane-b) terminal_id='terminal-B' ;;
  pane-concurrent-*) terminal_id="terminal-${3#pane-}" ;;
  *) exit 1 ;;
esac
printf '{"result":{"pane":{"terminal_id":"%s"}}}\n' "${terminal_id}"
SH
chmod +x "${FAKE_BIN}/herdr"

run_terminal() {
  local pane_id="$1"
  local command="$2"
  env -i \
  PATH="${FAKE_BIN}:/usr/bin:/bin" \
  HOME="${HOST_HOME}" \
  XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}" \
  AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  HERDR_PANE_ID="${pane_id}" \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 \
  bash -c "source '${SCRIPT}'; ${command}"
}

capture_env() {
  local pane_id="$1"
  local output="$2"
  # Variables intentionally expand inside the child shell started by run_terminal.
  # shellcheck disable=SC2016
  run_terminal "${pane_id}" \
    'printf "%s|%s|%s|%s|%s|%s|%s|%s\n" "$AGENT_CRED_ISOLATION_SLOT" "$AGENT_CRED_ISOLATION_SLOT_NAME" "$AGENT_CRED_ISOLATION_SLOT_ROOT" "$CODEX_HOME" "$CLINE_DATA_DIR" "$HOME" "$XDG_DATA_HOME" "$XDG_CONFIG_HOME"' \
    >"${output}"
}

# Pane A receives a fresh physical slot seeded from every supported vendor's
# current shared store. The Codex source is deliberately a symlink: migration
# must dereference it so refreshes cannot write back into the shared account.
capture_env pane-a "${TMP_DIR}/pane-a.env"
IFS='|' read -r slot_a slot_name_a root_a codex_a cline_a home_a data_a config_a <"${TMP_DIR}/pane-a.env"
assert_equal "${slot_a}" '1'
assert_equal "${slot_name_a}" 'slot-01'
assert_equal "${root_a}" "${STATE_ROOT}/slots/slot-01"
assert_not_symlink "${codex_a}/auth.json"
assert_content "${codex_a}/auth.json" 'legacy-codex'
assert_content "${cline_a}/data/settings/providers.json" 'legacy-cline'
assert_content "${home_a}/.gemini/antigravity-cli/antigravity-oauth-token" 'legacy-agy'
assert_content "${data_a}/kiro-cli/data.sqlite3" 'legacy-kiro'
assert_content "${data_a}/kiro-cli/data.sqlite3-wal" 'legacy-kiro-wal'
assert_content "${data_a}/opencode/auth.json" 'legacy-opencode'
assert_content "${config_a}/opencode/opencode.json" 'legacy-opencode-config'
assert_content "${data_a}/glm/auth.json" 'legacy-glm'
assert_content "${config_a}/glm/config.json" 'legacy-glm-config'

# Empirical same-vendor proof: two manual Codex logins write different token
# markers in two panes, and neither write changes the other's credential.
printf 'codex-account-A\n' >"${codex_a}/auth.json"
capture_env pane-b "${TMP_DIR}/pane-b.env"
IFS='|' read -r slot_b slot_name_b root_b codex_b cline_b home_b _data_b _config_b <"${TMP_DIR}/pane-b.env"
assert_equal "${slot_b}" '2'
assert_equal "${slot_name_b}" 'slot-02'
[[ "${root_a}" != "${root_b}" ]] || fail 'two terminals received the same slot root'
printf 'codex-account-B\n' >"${codex_b}/auth.json"
assert_content "${codex_a}/auth.json" 'codex-account-A'
assert_content "${codex_b}/auth.json" 'codex-account-B'
assert_content "${LEGACY_AUTH}" 'legacy-codex'

# Multi-vendor proof required by the operational gate: Cline and agy logins in
# each pane remain independent, alongside the two Codex accounts above.
printf 'cline-account-A\n' >"${cline_a}/data/settings/providers.json"
printf 'agy-account-A\n' >"${home_a}/.gemini/antigravity-cli/antigravity-oauth-token"
printf 'cline-account-B\n' >"${cline_b}/data/settings/providers.json"
printf 'agy-account-B\n' >"${home_b}/.gemini/antigravity-cli/antigravity-oauth-token"
assert_content "${cline_a}/data/settings/providers.json" 'cline-account-A'
assert_content "${cline_b}/data/settings/providers.json" 'cline-account-B'
assert_content "${home_a}/.gemini/antigravity-cli/antigravity-oauth-token" 'agy-account-A'
assert_content "${home_b}/.gemini/antigravity-cli/antigravity-oauth-token" 'agy-account-B'

# Recompacting pane ids must not change the stable terminal's slot or any live
# login. The fake Herdr returns terminal-A for the new pane id.
capture_env pane-a-recompact "${TMP_DIR}/pane-a-recompact.env"
IFS='|' read -r slot_a2 slot_name_a2 root_a2 codex_a2 cline_a2 home_a2 data_a2 config_a2 <"${TMP_DIR}/pane-a-recompact.env"
assert_equal "${slot_a2}" "${slot_a}"
assert_equal "${slot_name_a2}" "${slot_name_a}"
assert_equal "${root_a2}" "${root_a}"
assert_equal "${codex_a2}" "${codex_a}"
assert_equal "${cline_a2}" "${cline_a}"
assert_equal "${home_a2}" "${home_a}"
assert_equal "${data_a2}" "${data_a}"
assert_equal "${config_a2}" "${config_a}"
assert_content "${codex_a2}/auth.json" 'codex-account-A'
assert_content "${cline_a2}/data/settings/providers.json" 'cline-account-A'
assert_content "${home_a2}/.gemini/antigravity-cli/antigravity-oauth-token" 'agy-account-A'

# Fail-safe default: if Herdr lookup fails in a non-TTY shell, each unmatched
# pane gets a private slot rather than any shared vendor home.
capture_env fallback-pane-1 "${TMP_DIR}/fallback-1.env"
capture_env fallback-pane-2 "${TMP_DIR}/fallback-2.env"
IFS='|' read -r _ _ fallback_root_1 _ _ _ _ _ <"${TMP_DIR}/fallback-1.env"
IFS='|' read -r _ _ fallback_root_2 _ _ _ _ _ <"${TMP_DIR}/fallback-2.env"
[[ "${fallback_root_1}" != "${fallback_root_2}" ]] || fail 'fallback panes shared a credential slot'
[[ "${fallback_root_1}" == "${STATE_ROOT}"/slots/* ]] || fail 'fallback pane escaped isolated root'
[[ "${fallback_root_2}" == "${STATE_ROOT}"/slots/* ]] || fail 'fallback pane escaped isolated root'

# Exercise the flock-protected allocator under real process concurrency.
pids=()
for number in 1 2 3 4 5 6 7 8; do
  capture_env "pane-concurrent-${number}" "${TMP_DIR}/concurrent-${number}.env" &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done

python3 - "${STATE_ROOT}/registry.json" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as source:
    registry = json.load(source)

terminals = registry["terminals"]
slots = registry["slots"]
if len(terminals) != 12:
    raise SystemExit(f"expected 12 terminal mappings, got {len(terminals)}")
allocated = [int(entry["slot"]) for entry in terminals.values()]
if len(set(allocated)) != len(allocated):
    raise SystemExit("registry assigned one slot to multiple terminals")
if len(slots) != 12 or registry["next_slot"] != 13:
    raise SystemExit("registry monotonic allocator metadata is inconsistent")
for terminal_id, entry in terminals.items():
    owner = slots[str(entry["slot"])]["terminal_id"]
    if owner != terminal_id:
        raise SystemExit(f"slot ownership mismatch for {terminal_id}")
PY

status="$(run_terminal pane-a-recompact "'${SCRIPT}' doctor status")"
[[ "${status}" == *'pane=pane-a-recompact terminal=herdr:terminal-A slot=1'* ]] || fail 'doctor did not report the stable terminal slot'
for vendor in codex cline agy kiro opencode glm; do
  [[ "${status}" == *"vendor=${vendor} account=slot-01 state=on"* ]] || fail "doctor did not report ${vendor} on slot-01"
done

# -----------------------------------------------------------------------------
# ORQ-23 Rollback Artifact & Isolation Precedence Tests
# -----------------------------------------------------------------------------

# Test HERDR_PANE_ID fallback key precedence over tty/process when Herdr pane lookup returns empty
capture_env_no_herdr() {
  local pane_id="$1"
  local output="$2"
  env -i \
  PATH="/usr/bin:/bin" \
  HOME="${HOST_HOME}" \
  XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}" \
  AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  HERDR_PANE_ID="${pane_id}" \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 \
  bash -c "source '${SCRIPT}'; printf '%s|%s\n' \"\$AGENT_CRED_ISOLATION_SLOT\" \"\$AGENT_CRED_ISOLATION_SLOT_NAME\"" \
  >"${output}"
}

capture_env_no_herdr "pane-herdr-fallback-1" "${TMP_DIR}/herdr-fallback-1.env"
capture_env_no_herdr "pane-herdr-fallback-2" "${TMP_DIR}/herdr-fallback-2.env"
IFS='|' read -r h_slot1 _h_name1 <"${TMP_DIR}/herdr-fallback-1.env"
IFS='|' read -r h_slot2 _h_name2 <"${TMP_DIR}/herdr-fallback-2.env"
[[ "${h_slot1}" != "${h_slot2}" ]] || fail 'two different HERDR_PANE_ID fallbacks shared a slot'

ROLLBACK_SCRIPT="${TEST_DIR}/../rollback-cred-account-home.sh"
assert_file "${ROLLBACK_SCRIPT}"
assert_not_symlink "${ROLLBACK_SCRIPT}"

# Every rollback invocation below runs with an empty environment, a controlled
# PATH, synthetic paths under TEST_ROOT, and captured stdout/stderr. No command
# can resolve the real daemon binary, service unit, credential home, or runtime.
TEST_ROOT="${TMP_DIR}/rollback-test-root"
SYNTHETIC_SENTINEL_TOKEN="SK-SYNTHETIC-SENTINEL-KEY-DO-NOT-LEAK-998877"
SYNTHETIC_SECRET_SHAPED="sk-proj-syntheticsecretkey1234567890"
mkdir -p "${TEST_ROOT}/home"
chmod 700 "${TEST_ROOT}" "${TEST_ROOT}/home"

create_rollback_fixture() {
  local root="$1"
  mkdir -p "${root}/bin" "${root}/credential-homes/slots/slot-01/codex"
  chmod 700 \
    "${root}" \
    "${root}/bin" \
    "${root}/credential-homes" \
    "${root}/credential-homes/slots" \
    "${root}/credential-homes/slots/slot-01" \
    "${root}/credential-homes/slots/slot-01/codex"
  printf '#!/usr/bin/env bash\nexit 0\n# synthetic active binary\n' >"${root}/bin/active-daemon"
  printf '#!/usr/bin/env bash\nexit 0\n# synthetic previous binary\n' >"${root}/bin/previous-daemon"
  chmod 700 "${root}/bin/active-daemon" "${root}/bin/previous-daemon"
  printf '[Service]\nExecStart=%s\n' "${root}/bin/active-daemon" >"${root}/synthetic-daemon.service"
  chmod 600 "${root}/synthetic-daemon.service"
}

SUCCESS_ROOT="${TEST_ROOT}/success"
create_rollback_fixture "${SUCCESS_ROOT}"
printf '{"token":"%s","secret":"%s"}\n' \
  "${SYNTHETIC_SENTINEL_TOKEN}" \
  "${SYNTHETIC_SECRET_SHAPED}" \
  >"${SUCCESS_ROOT}/credential-homes/slots/slot-01/codex/auth.json"
chmod 600 "${SUCCESS_ROOT}/credential-homes/slots/slot-01/codex/auth.json"
cp "${SUCCESS_ROOT}/bin/active-daemon" "${SUCCESS_ROOT}/expected-active-daemon"

SYNTHETIC_STUBS_DIR="${TEST_ROOT}/stubs-success"
mkdir -p "${SYNTHETIC_STUBS_DIR}"
chmod 700 "${SYNTHETIC_STUBS_DIR}"

cat >"${SYNTHETIC_STUBS_DIR}/codex" <<'SH'
#!/usr/bin/env bash
[[ "${1:-}" == "--version" ]] && printf '%s\n' 'codex-cli synthetic'
SH
cat >"${SYNTHETIC_STUBS_DIR}/agy" <<'SH'
#!/usr/bin/env bash
[[ "${1:-}" == "--version" ]] && printf '%s\n' 'agy synthetic'
SH
cat >"${SYNTHETIC_STUBS_DIR}/kiro-cli" <<'SH'
#!/usr/bin/env bash
[[ "${1:-}" == "--version" ]] && printf '%s\n' 'kiro-cli synthetic'
SH
cat >"${SYNTHETIC_STUBS_DIR}/pgrep" <<'SH'
#!/usr/bin/env bash
exit 1
SH
cat >"${SYNTHETIC_STUBS_DIR}/curl" <<'SH'
#!/usr/bin/env bash
printf '%s\n' '{"status":"ok"}'
SH
cat >"${SYNTHETIC_STUBS_DIR}/systemctl" <<'SH'
#!/usr/bin/env bash
state_file="${SYSTEMCTL_STATE_FILE:?missing synthetic state file}"
case "${1:-} ${2:-}" in
  '--user stop')
    printf '%s\n' inactive >"${state_file}"
    ;;
  '--user daemon-reload')
    ;;
  '--user restart')
    printf '%s\n' active >"${state_file}"
    ;;
  '--user is-active')
    [[ "$(<"${state_file}")" == active ]]
    ;;
esac
SH
chmod 700 "${SYNTHETIC_STUBS_DIR}"/*

CONTROLLED_PATH="${SYNTHETIC_STUBS_DIR}:/usr/bin:/bin"
SYSTEMCTL_STATE_FILE="${TEST_ROOT}/systemctl-success.state"
printf '%s\n' active >"${SYSTEMCTL_STATE_FILE}"

COMMON_ROLLBACK_ENV=(
  "HOME=${TEST_ROOT}/home"
  "PATH=${CONTROLLED_PATH}"
  'LC_ALL=C'
  'TZ=UTC'
  "DAEMON_BIN_DIR=${SUCCESS_ROOT}/bin"
  "DAEMON_BIN_ACTIVE=${SUCCESS_ROOT}/bin/active-daemon"
  "DAEMON_BIN_PREVIOUS=${SUCCESS_ROOT}/bin/previous-daemon"
  "DAEMON_SERVICE_FILE=${SUCCESS_ROOT}/synthetic-daemon.service"
  'DAEMON_SERVICE_NAME=synthetic-daemon.service'
  "ROLLFORWARD_BACKUP_DIR=${SUCCESS_ROOT}/bin/rollforward-backups"
  "ROLLFORWARD_LATEST_POINTER=${SUCCESS_ROOT}/bin/rollforward.latest"
  "CRED_HOMES_ROOT=${SUCCESS_ROOT}/credential-homes"
  "CODEX_BIN=${SYNTHETIC_STUBS_DIR}/codex"
  "AGY_BIN=${SYNTHETIC_STUBS_DIR}/agy"
  "KIRO_BIN=${SYNTHETIC_STUBS_DIR}/kiro-cli"
  "ADDITIONAL_PROTECTED_LIVE_ROOT=${TEST_ROOT}"
  "SYSTEMCTL_STATE_FILE=${SYSTEMCTL_STATE_FILE}"
)

captured_anti_leak_logs="${TEST_ROOT}/captured-rollback.log"
: >"${captured_anti_leak_logs}"

run_rollback_capture() {
  local expected_status="$1"
  local label="$2"
  local output_file="$3"
  shift 3
  local actual_status

  set +e
  "$@" >"${output_file}" 2>&1
  actual_status=$?
  set -e

  printf '\n--- %s (exit=%d) ---\n' "${label}" "${actual_status}" >>"${captured_anti_leak_logs}"
  cat "${output_file}" >>"${captured_anti_leak_logs}"
  if (( actual_status != expected_status )); then
    fail "${label} exited ${actual_status}, expected ${expected_status}; captured output retained at ${output_file}"
  fi
}

# Protected synthetic roots model the hard-coded real-path ALLOW_LIVE gate
# without ever passing a real path to the rollback artifact.
guard_output="${TEST_ROOT}/guard-refusal.log"
run_rollback_capture 1 'ALLOW_LIVE guard refusal' "${guard_output}" \
  env -i "${COMMON_ROLLBACK_ENV[@]}" FORCE_ROLLBACK=1 \
  "${ROLLBACK_SCRIPT}" --live
if ! grep -q 'require explicit ALLOW_LIVE=1' "${guard_output}"; then
  fail 'protected live path did not report the ALLOW_LIVE=1 requirement'
fi

# Dry-run, successful rollback, and successful roll-forward are all synthetic.
dry_run_output="${TEST_ROOT}/dry-run.log"
run_rollback_capture 0 'synthetic dry-run' "${dry_run_output}" \
  env -i "${COMMON_ROLLBACK_ENV[@]}" \
  "${ROLLBACK_SCRIPT}" --dry-run
grep -q '=== DRY-RUN VERIFICATION RESULT: PASS ===' "${dry_run_output}" || \
  fail 'synthetic rollback dry-run did not report PASS'

live_output="${TEST_ROOT}/live.log"
run_rollback_capture 0 'synthetic live rollback' "${live_output}" \
  env -i "${COMMON_ROLLBACK_ENV[@]}" ALLOW_LIVE=1 FORCE_ROLLBACK=1 \
  "${ROLLBACK_SCRIPT}" --live
cmp -s "${SUCCESS_ROOT}/bin/active-daemon" "${SUCCESS_ROOT}/bin/previous-daemon" || \
  fail 'synthetic live rollback did not install the prior binary'

rollforward_output="${TEST_ROOT}/roll-forward.log"
run_rollback_capture 0 'synthetic live roll-forward' "${rollforward_output}" \
  env -i "${COMMON_ROLLBACK_ENV[@]}" ALLOW_LIVE=1 FORCE_ROLLFORWARD=1 \
  "${ROLLBACK_SCRIPT}" --roll-forward
cmp -s "${SUCCESS_ROOT}/bin/active-daemon" "${SUCCESS_ROOT}/expected-active-daemon" || \
  fail 'synthetic roll-forward did not restore the original active binary'

# Missing backup must fail with exactly 1 and preserve the real exit status.
MISSING_ROOT="${TEST_ROOT}/missing-backup"
create_rollback_fixture "${MISSING_ROOT}"
missing_output="${TEST_ROOT}/missing-backup.log"
run_rollback_capture 1 'missing roll-forward backup' "${missing_output}" \
  env -i \
  "HOME=${TEST_ROOT}/home" "PATH=${CONTROLLED_PATH}" LC_ALL=C TZ=UTC \
  "DAEMON_BIN_DIR=${MISSING_ROOT}/bin" \
  "DAEMON_BIN_ACTIVE=${MISSING_ROOT}/bin/active-daemon" \
  "DAEMON_BIN_PREVIOUS=${MISSING_ROOT}/bin/previous-daemon" \
  "DAEMON_SERVICE_FILE=${MISSING_ROOT}/synthetic-daemon.service" \
  DAEMON_SERVICE_NAME=synthetic-daemon.service \
  "ROLLFORWARD_BACKUP_DIR=${MISSING_ROOT}/bin/rollforward-backups" \
  "ROLLFORWARD_LATEST_POINTER=${MISSING_ROOT}/bin/rollforward.latest" \
  "CRED_HOMES_ROOT=${MISSING_ROOT}/credential-homes" \
  "CODEX_BIN=${SYNTHETIC_STUBS_DIR}/codex" \
  "AGY_BIN=${SYNTHETIC_STUBS_DIR}/agy" \
  "KIRO_BIN=${SYNTHETIC_STUBS_DIR}/kiro-cli" \
  "ADDITIONAL_PROTECTED_LIVE_ROOT=${TEST_ROOT}" \
  "SYSTEMCTL_STATE_FILE=${SYSTEMCTL_STATE_FILE}" \
  ALLOW_LIVE=1 FORCE_ROLLFORWARD=1 \
  "${ROLLBACK_SCRIPT}" --roll-forward
grep -q 'No roll-forward backup file recorded' "${missing_output}" || \
  fail 'missing backup path did not fail closed'

# Stop failure propagates the stub's exact status (9).
STOP_FAIL_ROOT="${TEST_ROOT}/stop-failure"
STOP_FAIL_STUBS="${TEST_ROOT}/stubs-stop-failure"
create_rollback_fixture "${STOP_FAIL_ROOT}"
mkdir -p "${STOP_FAIL_STUBS}"
cp "${SYNTHETIC_STUBS_DIR}"/{pgrep,curl,codex,agy,kiro-cli} "${STOP_FAIL_STUBS}/"
cat >"${STOP_FAIL_STUBS}/systemctl" <<'SH'
#!/usr/bin/env bash
case "${1:-} ${2:-}" in
  '--user stop') exit 9 ;;
  '--user is-active') exit 0 ;;
  *) exit 0 ;;
esac
SH
chmod 700 "${STOP_FAIL_STUBS}"/*
stop_fail_output="${TEST_ROOT}/stop-failure.log"
run_rollback_capture 9 'systemctl stop failure' "${stop_fail_output}" \
  env -i \
  "HOME=${TEST_ROOT}/home" "PATH=${STOP_FAIL_STUBS}:/usr/bin:/bin" LC_ALL=C TZ=UTC \
  "DAEMON_BIN_DIR=${STOP_FAIL_ROOT}/bin" \
  "DAEMON_BIN_ACTIVE=${STOP_FAIL_ROOT}/bin/active-daemon" \
  "DAEMON_BIN_PREVIOUS=${STOP_FAIL_ROOT}/bin/previous-daemon" \
  "DAEMON_SERVICE_FILE=${STOP_FAIL_ROOT}/synthetic-daemon.service" \
  DAEMON_SERVICE_NAME=synthetic-daemon.service \
  "ROLLFORWARD_BACKUP_DIR=${STOP_FAIL_ROOT}/bin/rollforward-backups" \
  "ROLLFORWARD_LATEST_POINTER=${STOP_FAIL_ROOT}/bin/rollforward.latest" \
  "CRED_HOMES_ROOT=${STOP_FAIL_ROOT}/credential-homes" \
  "CODEX_BIN=${STOP_FAIL_STUBS}/codex" \
  "AGY_BIN=${STOP_FAIL_STUBS}/agy" \
  "KIRO_BIN=${STOP_FAIL_STUBS}/kiro-cli" \
  "ADDITIONAL_PROTECTED_LIVE_ROOT=${TEST_ROOT}" \
  ALLOW_LIVE=1 FORCE_ROLLBACK=1 \
  "${ROLLBACK_SCRIPT}" --live

# A restart that never becomes active must fail with exactly 1.
RESTART_FAIL_ROOT="${TEST_ROOT}/restart-failure"
RESTART_FAIL_STUBS="${TEST_ROOT}/stubs-restart-failure"
create_rollback_fixture "${RESTART_FAIL_ROOT}"
mkdir -p "${RESTART_FAIL_STUBS}"
cp "${SYNTHETIC_STUBS_DIR}"/{pgrep,curl,codex,agy,kiro-cli} "${RESTART_FAIL_STUBS}/"
cat >"${RESTART_FAIL_STUBS}/systemctl" <<'SH'
#!/usr/bin/env bash
case "${1:-} ${2:-}" in
  '--user stop'|'--user daemon-reload'|'--user restart') exit 0 ;;
  '--user is-active') exit 1 ;;
  *) exit 0 ;;
esac
SH
chmod 700 "${RESTART_FAIL_STUBS}"/*
restart_fail_output="${TEST_ROOT}/restart-failure.log"
run_rollback_capture 1 'systemctl restart inactive failure' "${restart_fail_output}" \
  env -i \
  "HOME=${TEST_ROOT}/home" "PATH=${RESTART_FAIL_STUBS}:/usr/bin:/bin" LC_ALL=C TZ=UTC \
  "DAEMON_BIN_DIR=${RESTART_FAIL_ROOT}/bin" \
  "DAEMON_BIN_ACTIVE=${RESTART_FAIL_ROOT}/bin/active-daemon" \
  "DAEMON_BIN_PREVIOUS=${RESTART_FAIL_ROOT}/bin/previous-daemon" \
  "DAEMON_SERVICE_FILE=${RESTART_FAIL_ROOT}/synthetic-daemon.service" \
  DAEMON_SERVICE_NAME=synthetic-daemon.service \
  "ROLLFORWARD_BACKUP_DIR=${RESTART_FAIL_ROOT}/bin/rollforward-backups" \
  "ROLLFORWARD_LATEST_POINTER=${RESTART_FAIL_ROOT}/bin/rollforward.latest" \
  "CRED_HOMES_ROOT=${RESTART_FAIL_ROOT}/credential-homes" \
  "CODEX_BIN=${RESTART_FAIL_STUBS}/codex" \
  "AGY_BIN=${RESTART_FAIL_STUBS}/agy" \
  "KIRO_BIN=${RESTART_FAIL_STUBS}/kiro-cli" \
  "ADDITIONAL_PROTECTED_LIVE_ROOT=${TEST_ROOT}" \
  ALLOW_LIVE=1 FORCE_ROLLBACK=1 \
  "${ROLLBACK_SCRIPT}" --live

# Captured output from success and every error path must remain content-free.
if grep -F -q "${SYNTHETIC_SENTINEL_TOKEN}" "${captured_anti_leak_logs}"; then
  fail 'ANTI-LEAK REGRESSION: output contained the synthetic sentinel token'
fi
if grep -F -q "${SYNTHETIC_SECRET_SHAPED}" "${captured_anti_leak_logs}"; then
  fail 'ANTI-LEAK REGRESSION: output contained synthetic secret-shaped material'
fi
if grep -E -q 'sk-[a-zA-Z0-9_-]{10,}' "${captured_anti_leak_logs}"; then
  fail 'ANTI-LEAK REGRESSION: output contained a secret-shaped pattern'
fi
if grep -F -q '/home/ec2-user/.local/lib' "${captured_anti_leak_logs}" || \
  grep -F -q '/home/ec2-user/.config/systemd' "${captured_anti_leak_logs}"; then
  fail 'HERMETICITY REGRESSION: rollback output referenced a real daemon or service path'
fi

printf 'PASS: 6-vendor migration, isolated dual login, HERDR_PANE_ID fallback precedence, hermetic TEST_ROOT rollback/dry-run/roll-forward, anti-leak sentinel regression, exact failure statuses, mode 0700 posture, and flock allocator\n'
