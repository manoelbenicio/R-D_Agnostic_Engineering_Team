#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT="${TEST_DIR}/../agent-cred-isolation.sh"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf -- "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf 'agent-cred-isolation-harness: FAIL: %s\n' "$1" >&2
  exit 1
}

assert_equal() {
  [[ "$1" == "$2" ]] || fail 'value-mismatch'
}

assert_file() {
  [[ -f "$1" && ! -L "$1" ]] || fail 'expected-regular-file'
}

assert_dir() {
  [[ -d "$1" && ! -L "$1" ]] || fail 'expected-directory'
}

assert_not_symlink() {
  [[ ! -L "$1" ]] || fail 'unexpected-symlink'
}

assert_content() {
  assert_file "$1"
  assert_equal "$(<"$1")" "$2"
}

assert_sentinel_free() {
  local output_file="$1"
  if grep -Fq -- "${SENTINEL}" "${output_file}"; then
    fail 'sentinel-reached-captured-output'
  fi
}

chmod 700 "${TMP_DIR}"
FIXTURE_ROOT="${TMP_DIR}/fixture"
OUTSIDE_ROOT="${TMP_DIR}/synthetic-outside-root"
mkdir -p "${FIXTURE_ROOT}" "${OUTSIDE_ROOT}"
chmod 700 "${FIXTURE_ROOT}" "${OUTSIDE_ROOT}"
FIXTURE_ROOT="$(cd -- "${FIXTURE_ROOT}" && pwd -P)"

require_fixture_directory() {
  local label="$1"
  local path="$2"
  local relative component cursor canonical found_symlink

  [[ -n "${path}" && "${path}" == /* ]] || fail "path-contract:${label}:not-absolute"
  case "/${path#/}/" in
    *'/../'*) fail "path-contract:${label}:traversal" ;;
  esac
  [[ "${path}" == "${FIXTURE_ROOT}" || "${path}" == "${FIXTURE_ROOT}/"* ]] || \
    fail "path-contract:${label}:outside-fixture"
  [[ -d "${path}" && ! -L "${path}" ]] || fail "path-contract:${label}:absent-or-symlink"

  relative="${path#"${FIXTURE_ROOT}"}"
  relative="${relative#/}"
  cursor="${FIXTURE_ROOT}"
  local IFS='/'
  read -r -a components <<<"${relative}"
  for component in "${components[@]}"; do
    [[ -n "${component}" ]] || continue
    cursor="${cursor}/${component}"
    [[ ! -L "${cursor}" ]] || fail "path-contract:${label}:symlink-component"
  done

  canonical="$(cd -- "${path}" && pwd -P)" || fail "path-contract:${label}:canonicalize"
  [[ "${canonical}" == "${path}" ]] || fail "path-contract:${label}:non-canonical"
  found_symlink="$(find -P "${path}" -type l -print -quit 2>/dev/null)" || \
    fail "path-contract:${label}:scan"
  [[ -z "${found_symlink}" ]] || fail "path-contract:${label}:nested-symlink"
}

HOST_HOME="${FIXTURE_ROOT}/host-home"
HOST_XDG_DATA_HOME="${HOST_HOME}/.local/share"
HOST_XDG_CONFIG_HOME="${HOST_HOME}/.config"
STATE_ROOT="${FIXTURE_ROOT}/state"
FAKE_BIN="${FIXTURE_ROOT}/bin"
LEGACY_AUTH="${HOST_HOME}/.codex/auth.json"
SENTINEL='ORQ64_SYNTHETIC_SENTINEL_MUST_NOT_REACH_OUTPUT_7f921c'

mkdir -p \
  "${HOST_HOME}/.codex" \
  "${HOST_HOME}/.cline/data/settings" \
  "${HOST_HOME}/.gemini/antigravity-cli" \
  "${HOST_XDG_DATA_HOME}/kiro-cli" \
  "${HOST_XDG_DATA_HOME}/opencode" \
  "${HOST_XDG_DATA_HOME}/glm" \
  "${HOST_XDG_CONFIG_HOME}/opencode" \
  "${HOST_XDG_CONFIG_HOME}/glm" \
  "${STATE_ROOT}/slots" \
  "${FAKE_BIN}"

validate_runtime_roots() {
  require_fixture_directory login-home "${HOME}"
  require_fixture_directory host-home "${AGENT_CRED_ISOLATION_HOST_HOME}"
  require_fixture_directory host-xdg-data "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}"
  require_fixture_directory host-xdg-config "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}"
  require_fixture_directory state-root "${AGENT_CRED_ISOLATION_ROOT}"
  require_fixture_directory destination-slots "${AGENT_CRED_ISOLATION_ROOT}/slots"
  require_fixture_directory source-codex "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex"
  require_fixture_directory source-cline "${AGENT_CRED_ISOLATION_HOST_HOME}/.cline"
  require_fixture_directory source-agy "${AGENT_CRED_ISOLATION_HOST_HOME}/.gemini/antigravity-cli"
  require_fixture_directory source-kiro "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/kiro-cli"
  require_fixture_directory source-opencode-data "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/opencode"
  require_fixture_directory source-opencode-config "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/opencode"
  require_fixture_directory source-glm-data "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/glm"
  require_fixture_directory source-glm-config "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/glm"
}

HOME="${HOST_HOME}" \
AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}" \
AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_XDG_DATA_HOME}" \
AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_XDG_CONFIG_HOME}" \
AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  validate_runtime_roots

expect_path_rejected() {
  local label="$1"
  local path="$2"
  local stdout_file="${TMP_DIR}/guard-${label}.stdout"
  local stderr_file="${TMP_DIR}/guard-${label}.stderr"
  local rc

  set +e
  (require_fixture_directory "${label}" "${path}") >"${stdout_file}" 2>"${stderr_file}"
  rc=$?
  set -e
  (( rc != 0 )) || fail "path-contract:${label}:accepted"
  assert_sentinel_free "${stdout_file}"
  assert_sentinel_free "${stderr_file}"
}

GUARD_CASE_ROOT="${FIXTURE_ROOT}/guard-cases"
mkdir -p "${GUARD_CASE_ROOT}/valid" "${GUARD_CASE_ROOT}/symlink-target"
ln -s "${GUARD_CASE_ROOT}/symlink-target" "${GUARD_CASE_ROOT}/symlink-root"
expect_path_rejected symlink "${GUARD_CASE_ROOT}/symlink-root"
expect_path_rejected traversal "${GUARD_CASE_ROOT}/valid/../valid"
expect_path_rejected absent "${GUARD_CASE_ROOT}/absent"
expect_path_rejected inherited "${OUTSIDE_ROOT}"

printf 'legacy-codex\n' >"${LEGACY_AUTH}"
printf 'legacy-codex-config\n' >"${HOST_HOME}/.codex/config.toml"
printf 'legacy-cline\n' >"${HOST_HOME}/.cline/data/settings/providers.json"
printf 'legacy-agy\n' >"${HOST_HOME}/.gemini/antigravity-cli/antigravity-oauth-token"
printf 'legacy-kiro\n' >"${HOST_XDG_DATA_HOME}/kiro-cli/data.sqlite3"
printf 'legacy-kiro-wal\n' >"${HOST_XDG_DATA_HOME}/kiro-cli/data.sqlite3-wal"
printf 'legacy-opencode\n' >"${HOST_XDG_DATA_HOME}/opencode/auth.json"
printf 'legacy-opencode-config\n' >"${HOST_XDG_CONFIG_HOME}/opencode/opencode.json"
printf 'legacy-glm\n' >"${HOST_XDG_DATA_HOME}/glm/auth.json"
printf 'legacy-glm-config\n' >"${HOST_XDG_CONFIG_HOME}/glm/config.json"

SENTINEL_FILE="${FIXTURE_ROOT}/synthetic-sentinel-credential.json"
printf '%s\n' "${SENTINEL}" >"${SENTINEL_FILE}"
for scenario in match actual-mismatch expected-mismatch; do
  set +e
  case "${scenario}" in
    match)
      (assert_content "${SENTINEL_FILE}" "${SENTINEL}") \
        >"${TMP_DIR}/${scenario}.stdout" 2>"${TMP_DIR}/${scenario}.stderr"
      rc=$?
      ;;
    actual-mismatch)
      (assert_content "${SENTINEL_FILE}" 'different-synthetic-value') \
        >"${TMP_DIR}/${scenario}.stdout" 2>"${TMP_DIR}/${scenario}.stderr"
      rc=$?
      ;;
    expected-mismatch)
      printf 'different-synthetic-value\n' >"${SENTINEL_FILE}"
      (assert_content "${SENTINEL_FILE}" "${SENTINEL}") \
        >"${TMP_DIR}/${scenario}.stdout" 2>"${TMP_DIR}/${scenario}.stderr"
      rc=$?
      ;;
  esac
  set -e
  if [[ "${scenario}" == match ]]; then
    (( rc == 0 )) || fail 'sentinel-match-failed'
  else
    (( rc != 0 )) || fail 'sentinel-mismatch-accepted'
  fi
  assert_sentinel_free "${TMP_DIR}/${scenario}.stdout"
  assert_sentinel_free "${TMP_DIR}/${scenario}.stderr"
done
rm -f -- "${SENTINEL_FILE}"

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

  (
    unset AGENT_CRED_ISOLATION_SCRIPT_LOADED AGENT_CRED_ISOLATION_REGISTRY \
      AGENT_CRED_ISOLATION_TERMINAL_ID AGENT_CRED_ISOLATION_SLOT \
      AGENT_CRED_ISOLATION_SLOT_NAME AGENT_CRED_ISOLATION_SLOT_ROOT \
      CODEX_HOME CLINE_DATA_DIR CLINE_SANDBOX CLINE_SANDBOX_DATA_DIR
    export PATH="${FAKE_BIN}:${PATH}"
    export HOME="${HOST_HOME}"
    export XDG_DATA_HOME="${HOST_XDG_DATA_HOME}"
    export XDG_CONFIG_HOME="${HOST_XDG_CONFIG_HOME}"
    export HERDR_PANE_ID="${pane_id}"
    export AGENT_CRED_ISOLATION_HOST_HOME="${HOST_HOME}"
    export AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="${HOST_XDG_DATA_HOME}"
    export AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="${HOST_XDG_CONFIG_HOME}"
    export AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}"
    export AGENT_CRED_ISOLATION_AUTOSTART=1
    validate_runtime_roots
    source "${SCRIPT}"
    eval "${command}"
  )
}

capture_env() {
  local pane_id="$1"
  local output="$2"
  run_terminal "${pane_id}" \
    'printf "%s|%s|%s|%s|%s|%s|%s|%s\n" "$AGENT_CRED_ISOLATION_SLOT" "$AGENT_CRED_ISOLATION_SLOT_NAME" "$AGENT_CRED_ISOLATION_SLOT_ROOT" "$CODEX_HOME" "$CLINE_DATA_DIR" "$HOME" "$XDG_DATA_HOME" "$XDG_CONFIG_HOME"' \
    >"${output}"
}

# Pane A receives a fresh physical slot seeded only from validated synthetic
# stores beneath this harness's private fixture root.
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

printf 'PASS: 6-vendor migration, isolated dual login, Cline+agy, recompaction, fail-safe, and flock allocator\n'
