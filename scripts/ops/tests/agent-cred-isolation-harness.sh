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

cat >"${FAKE_BIN}/codex" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "${CODEX_HOME}"
SH
chmod +x "${FAKE_BIN}/codex"

cat >"${FAKE_BIN}/grok" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
[[ "${1:-}" == 'device-login' ]] || exit 64
if [[ -n "${GROK_FAKE_LOG:-}" ]]; then printf 'device-login\n' >>"${GROK_FAKE_LOG}"; fi
printf '%s\n' "${GROK_HOME}"
SH
chmod +x "${FAKE_BIN}/grok"

run_terminal() {
  local pane_id="$1"
  local command="$2"
  PATH="${FAKE_BIN}:${PATH}" \
  HOME="${HOST_HOME}" \
  HERDR_PANE_ID="${pane_id}" \
  AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}" \
  AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 \
  AGENT_CRED_ISOLATION_ENABLE_VENDOR_SLOTS=1 \
  GROK_FAKE_LOG="${GROK_FAKE_LOG:-}" \
  bash -c "source '${SCRIPT}'; ${command}"
}

capture_env() {
  local pane_id="$1"
  local output="$2"
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
IFS='|' read -r slot_b slot_name_b root_b codex_b cline_b home_b data_b config_b <"${TMP_DIR}/pane-b.env"
assert_equal "${slot_b}" '2'
assert_equal "${slot_name_b}" 'slot-02'
[[ "${root_a}" != "${root_b}" ]] || fail 'two terminals received the same slot root'
printf 'codex-account-B\n' >"${codex_b}/auth.json"
assert_content "${codex_a}/auth.json" 'codex-account-A'
assert_content "${codex_b}/auth.json" 'codex-account-B'
assert_content "${LEGACY_AUTH}" 'legacy-codex'

# Hard Codex lifecycle proof: every explicit login creates a new physical
# folder, while a subsequent normal launch in the same shell reuses that
# login. This rule does not consult Herdr.
lifecycle_login_1="${TMP_DIR}/lifecycle-login-1.out"
lifecycle_reuse_1="${TMP_DIR}/lifecycle-reuse-1.out"
lifecycle_login_2="${TMP_DIR}/lifecycle-login-2.out"
PATH="${FAKE_BIN}:${PATH}" \
HOME="${HOST_HOME}" \
AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
AGENT_CRED_ISOLATION_AUTOSTART=1 \
HERDR_PANE_ID='pane-a' \
LIFECYCLE_LOGIN_1="${lifecycle_login_1}" \
LIFECYCLE_REUSE_1="${lifecycle_reuse_1}" \
LIFECYCLE_LOGIN_2="${lifecycle_login_2}" \
bash -c "source '${SCRIPT}'; codex login >\"\${LIFECYCLE_LOGIN_1}\"; codex --probe >\"\${LIFECYCLE_REUSE_1}\"; codex login >\"\${LIFECYCLE_LOGIN_2}\""
login_home_1="$(<"${lifecycle_login_1}")"
reuse_home_1="$(<"${lifecycle_reuse_1}")"
login_home_2="$(<"${lifecycle_login_2}")"
assert_equal "${reuse_home_1}" "${login_home_1}"
[[ "${login_home_1}" != "${login_home_2}" ]] || fail 'two Codex logins reused one credential folder'
[[ "${login_home_1}" == "${STATE_ROOT}"/codex-logins/login-* ]] || fail 'first login escaped lifecycle root'
[[ "${login_home_2}" == "${STATE_ROOT}"/codex-logins/login-* ]] || fail 'second login escaped lifecycle root'
assert_dir "${login_home_1}"
assert_dir "${login_home_2}"
[[ ! -e "${login_home_1}/auth.json" ]] || fail 'new login folder inherited auth.json'
[[ ! -e "${login_home_2}/auth.json" ]] || fail 'new login folder inherited auth.json'

# Regression: a child closing an inherited lease FD must not unlock the
# parent's open-file description. Cleanup remains unable to delete the aged
# parent home until the parent exits.
holder_home_file="${TMP_DIR}/holder-home"
holder_ready="${TMP_DIR}/holder-ready"
holder_release="${TMP_DIR}/holder-release"
PATH="${FAKE_BIN}:${PATH}" HOME="${HOST_HOME}" AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 bash -c \
  "source '${SCRIPT}'; codex login >/dev/null; touch '${holder_ready}'; (agent_cred_isolation_release_codex_lease); while [[ ! -e '${holder_release}' ]]; do sleep 0.01; done" &
holder_pid=$!
while [[ ! -e "${holder_ready}" ]]; do sleep 0.01; done
holder_home="$(find "${STATE_ROOT}/codex-logins" -mindepth 1 -maxdepth 1 -type d -name 'login-*' -printf '%T@ %p\n' | sort -nr | awk 'NR==1 {sub(/^[^ ]+ /, ""); print}')"
[[ -n "${holder_home}" ]] || fail 'holder login home was not found'
touch -d '6 days ago' "${holder_home}"
AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" HOME="${HOST_HOME}" bash -c \
  "source '${SCRIPT}'; agent_cred_isolation_release_codex_lease"
run_terminal pane-a 'agent_cred_isolation_cleanup_codex_logins'
assert_dir "${holder_home}"
touch "${holder_release}"
wait "${holder_pid}"
run_terminal pane-a 'agent_cred_isolation_cleanup_codex_logins'
assert_dir "${holder_home}"

# Candidate replacement/symlink swap attempt: the original aged leased
# directory is moved out and replaced by a symlink before cleanup. Neither the
# replacement nor the moved candidate may be removed.
swap_home="${STATE_ROOT}/codex-logins/login-swap"
swap_moved="${TMP_DIR}/login-swap-moved"
swap_target="${TMP_DIR}/swap-target"
mkdir -p "${swap_target}"
mkdir "${swap_home}"
printf 'agent-cred-login-lease-v1\n' >"${swap_home}/.lease"
touch -d '6 days ago' "${swap_home}"
mv "${swap_home}" "${swap_moved}"
ln -s "${swap_target}" "${swap_home}"
run_terminal pane-a 'agent_cred_isolation_cleanup_codex_logins'
[[ -L "${swap_home}" && -d "${swap_moved}" ]] || fail 'candidate swap was not preserved'

# Retention proof: a 6-day credential folder is removed on the next login,
# while a 4-day folder remains.
expired_home="${STATE_ROOT}/codex-logins/login-expired"
recent_home="${STATE_ROOT}/codex-logins/login-recent"
live_home="${STATE_ROOT}/codex-logins/login-live"
legacy_home="${STATE_ROOT}/codex-logins/login-legacy"
corrupt_home="${STATE_ROOT}/codex-logins/login-corrupt"
symlink_target="${TMP_DIR}/symlink-target"
mkdir "${expired_home}" "${recent_home}" "${live_home}" "${legacy_home}" "${corrupt_home}" "${symlink_target}"
for home in "${expired_home}" "${recent_home}" "${live_home}"; do
  printf 'agent-cred-login-lease-v1\n' >"${home}/.lease"
  chmod 600 "${home}/.lease"
done
printf 'not-a-valid-lease\n' >"${corrupt_home}/.lease"
printf 'legacy\n' >"${legacy_home}/marker"
ln -s "${symlink_target}" "${STATE_ROOT}/codex-logins/login-symlink"
touch -d '6 days ago' "${expired_home}"
touch -d '4 days ago' "${recent_home}"
touch -d '6 days ago' "${live_home}"
touch -d '6 days ago' "${legacy_home}"
touch -d '6 days ago' "${corrupt_home}"

# A second shell holds the lease: cleanup must skip it. Owner death releases
# the lock, after which the same aged directory becomes removable.
exec {live_lease_fd}<"${live_home}/.lease"
flock -x "${live_lease_fd}"
run_terminal pane-a 'codex login >/dev/null'
assert_dir "${expired_home}"
assert_dir "${recent_home}"
assert_dir "${live_home}"

# GROK A-D: explicit physical profiles start without migrated auth state.
for grok_profile in A B C D; do
  run_terminal pane-a "agent_cred_isolation_grok_profile_init ${grok_profile}"
  grok_path="${STATE_ROOT}/grok/profiles/grok-${grok_profile,,}"
  assert_dir "${grok_path}"
  assert_equal "$(stat -c '%a' "${grok_path}")" '700'
  [[ -z "$(find "${grok_path}" -mindepth 1 -maxdepth 1 -print -quit)" ]] || fail "GROK ${grok_profile} profile did not start empty"
done
[[ "${STATE_ROOT}/grok/profiles/grok-a" != "${STATE_ROOT}/grok/profiles/grok-b" ]] || fail 'GROK A/B shared a profile'
for invalid_grok in '' E ../A A/B grok-a; do
  if run_terminal pane-a "agent_cred_isolation_grok_profile_init '${invalid_grok}'" >/dev/null 2>&1; then
    fail "invalid GROK profile accepted: ${invalid_grok}"
  fi
done

# Initialization is copy-free and idempotent: it does not overwrite existing
# data, and a pre-existing symlink profile fails closed.
printf 'status-sentinel-never-read\n' >"${STATE_ROOT}/grok/profiles/grok-a/sentinel-token-name"
run_terminal pane-a 'agent_cred_isolation_grok_profile_init A'
assert_content "${STATE_ROOT}/grok/profiles/grok-a/sentinel-token-name" 'status-sentinel-never-read'
grok_symlink_root="${TMP_DIR}/grok-symlink-state"
mkdir -p "${grok_symlink_root}/grok/profiles" "${grok_symlink_root}/grok/terminal-bindings" "${TMP_DIR}/grok-symlink-target"
ln -s "${TMP_DIR}/grok-symlink-target" "${grok_symlink_root}/grok/profiles/grok-a"
if PATH="${FAKE_BIN}:${PATH}" HOME="${HOST_HOME}" AGENT_CRED_ISOLATION_ROOT="${grok_symlink_root}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 bash -c "source '${SCRIPT}'; agent_cred_isolation_grok_profile_init A" >/dev/null 2>&1; then
  fail 'symlink GROK profile was accepted'
fi

# Parent owns A. An inherited child closes only its FD; another terminal still
# cannot acquire A. In parallel, terminal B can own profile B independently.
grok_a_ready="${TMP_DIR}/grok-a-ready"
grok_a_release="${TMP_DIR}/grok-a-release"
PATH="${FAKE_BIN}:${PATH}" HOME="${HOST_HOME}" HERDR_PANE_ID='pane-a' \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" AGENT_CRED_ISOLATION_AUTOSTART=1 \
  bash -c "source '${SCRIPT}'; agent_cred_isolation_grok_attach A; agent_cred_isolation_grok_assert_binding; (agent_cred_isolation_grok_release_lease); touch '${grok_a_ready}'; while [[ ! -e '${grok_a_release}' ]]; do sleep 0.01; done" &
grok_a_pid=$!
while [[ ! -e "${grok_a_ready}" ]]; do sleep 0.01; done
if run_terminal pane-b 'agent_cred_isolation_grok_attach A' >/dev/null 2>&1; then
  fail 'second GROK A writer was accepted'
fi
grok_b_ready="${TMP_DIR}/grok-b-ready"
grok_b_release="${TMP_DIR}/grok-b-release"
PATH="${FAKE_BIN}:${PATH}" HOME="${HOST_HOME}" HERDR_PANE_ID='pane-b' \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" AGENT_CRED_ISOLATION_AUTOSTART=1 \
  bash -c "source '${SCRIPT}'; agent_cred_isolation_grok_attach B; agent_cred_isolation_grok_assert_binding; touch '${grok_b_ready}'; while [[ ! -e '${grok_b_release}' ]]; do sleep 0.01; done" &
grok_b_pid=$!
while [[ ! -e "${grok_b_ready}" ]]; do sleep 0.01; done
touch "${grok_a_release}" "${grok_b_release}"
wait "${grok_a_pid}" "${grok_b_pid}"

# Inherited GROK_HOME alone is never ownership. Status is content-off and the
# fake device-login executable runs only after an explicit human-equivalent call.
if PATH="${FAKE_BIN}:${PATH}" HOME="${HOST_HOME}" HERDR_PANE_ID='pane-a' \
  GROK_HOME="${STATE_ROOT}/grok/profiles/grok-a" AGENT_CRED_GROK_PROFILE=A \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" AGENT_CRED_ISOLATION_AUTOSTART=1 \
  bash -c "source '${SCRIPT}'; agent_cred_isolation_grok_assert_binding" >/dev/null 2>&1; then
  fail 'inherited shared GROK_HOME was accepted'
fi
grok_status="$(run_terminal pane-a 'agent_cred_isolation_grok_status A')"
[[ "${grok_status}" != *'status-sentinel-never-read'* ]] || fail 'GROK status leaked sentinel content'
grok_fake_log="${TMP_DIR}/grok-device-login.log"
[[ ! -e "${grok_fake_log}" ]] || fail 'GROK device login ran automatically'
GROK_FAKE_LOG="${grok_fake_log}"
export GROK_FAKE_LOG
grok_device_home="$(run_terminal pane-a 'agent_cred_isolation_grok_device_login D')"
unset GROK_FAKE_LOG
assert_equal "${grok_device_home}" "${STATE_ROOT}/grok/profiles/grok-d"
assert_content "${grok_fake_log}" 'device-login'

# GROK cleanup is permanently report-only, including aged/symlink candidates.
touch -d '6 days ago' "${STATE_ROOT}/grok/profiles/grok-c"
run_terminal pane-a 'agent_cred_isolation_cleanup_grok_logins'
assert_dir "${STATE_ROOT}/grok/profiles/grok-c"
assert_content "${STATE_ROOT}/grok/profiles/grok-a/sentinel-token-name" 'status-sentinel-never-read'
assert_dir "${legacy_home}"
assert_dir "${corrupt_home}"
[[ -L "${STATE_ROOT}/codex-logins/login-symlink" ]] || fail 'symlink candidate was changed'
flock -u "${live_lease_fd}"
exec {live_lease_fd}>&-
run_terminal pane-a 'source "'"${SCRIPT}"'"; agent_cred_isolation_cleanup_codex_logins'
assert_dir "${live_home}"

# Existing-account recovery creates a new lifecycle folder without requiring a
# second OAuth login.
recovered_home="$(
  run_terminal pane-a 'codex_recover A; printf "%s\n" "${CODEX_HOME}"'
)"
[[ "${recovered_home}" == "${STATE_ROOT}"/codex-logins/login-* ]] || fail 'recovered account escaped lifecycle root'
assert_content "${recovered_home}/auth.json" 'codex-account-A'

# Regression proof for the production failure: a new shell inherits the
# parent's loaded marker and CODEX_HOME. Even with Herdr lookup unavailable,
# the first Codex launch must replace that stale home with a new local folder.
inherited_probe="$(
  PATH="${FAKE_BIN}:${PATH}" \
  HOME="${HOST_HOME}" \
  CODEX_HOME="${login_home_2}" \
  XDG_DATA_HOME="${data_a}" \
  XDG_CONFIG_HOME="${config_a}" \
  AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}" \
  AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share" \
  AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config" \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_CODEX_LOGIN_ID="${login_home_2##*/}" \
  AGENT_CRED_CODEX_LEASE_SHELL_PID='1' \
  AGENT_CRED_ISOLATION_SCRIPT_LOADED=1 \
  AGENT_CRED_ISOLATION_AUTOSTART=1 \
  HERDR_PANE_ID='herdr-is-down' \
  bash -c "source '${SCRIPT}'; codex --probe"
)"
[[ "${inherited_probe}" == "${STATE_ROOT}"/codex-logins/login-* ]] || fail 'inherited shell escaped lifecycle root'
[[ "${inherited_probe}" != "${login_home_2}" ]] || fail 'inherited shell reused the parent credential folder'
assert_dir "${inherited_probe}"

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
allocated = [int(entry["slot"]) for entry in terminals.values()]
if len(set(allocated)) != len(allocated):
    raise SystemExit("registry assigned one slot to multiple terminals")
if len(slots) != len(terminals) or registry["next_slot"] != max(allocated) + 1:
    raise SystemExit("registry monotonic allocator metadata is inconsistent")
for number in range(1, 9):
    if f"herdr:terminal-concurrent-{number}" not in terminals:
        raise SystemExit(f"missing concurrent terminal {number}")
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

printf 'PASS: Codex no-delete leases, GROK A-D physical profiles/device-login isolation, 6-vendor migration, recompaction, fallback, and flock allocators\n'
