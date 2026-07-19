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
export AGENT_CRED_ISOLATION_MIGRATE_LEGACY=1
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
  PATH="${FAKE_BIN}:${PATH}" \
  HOME="${HOST_HOME}" \
  HERDR_PANE_ID="${pane_id}" \
  AGENT_CRED_ISOLATION_ROOT="${STATE_ROOT}" \
  AGENT_CRED_ISOLATION_AUTOSTART=1 \
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
