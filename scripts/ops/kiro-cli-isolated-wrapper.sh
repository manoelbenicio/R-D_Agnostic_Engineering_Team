#!/usr/bin/env bash
set -euo pipefail

# Mandatory credential-isolation boundary for manual Kiro CLI launches.
# This wrapper must occupy the normal kiro-cli PATH entry; the vendor binary
# is retained beside it as kiro-cli.real.

readonly KIRO_WRAPPER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly KIRO_WRAPPER_BIN="${KIRO_WRAPPER_REAL_BIN:-${KIRO_WRAPPER_DIR}/kiro-cli.real}"
readonly KIRO_WRAPPER_ISOLATION_SCRIPT="${KIRO_WRAPPER_ISOLATION_SCRIPT:-/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/ops/agent-cred-isolation.sh}"

fail() {
  printf 'kiro-cli: REFUSED unsafe credential launch: %s\n' "$*" >&2
  exit 78
}

[[ -x "${KIRO_WRAPPER_BIN}" ]] || fail "vendor binary missing at ${KIRO_WRAPPER_BIN}"
[[ -r "${KIRO_WRAPPER_ISOLATION_SCRIPT}" ]] || fail "isolation script missing"

export AGENT_CRED_ISOLATION_AUTOSTART=1
export AGENT_CRED_ISOLATION_MIGRATE_LEGACY=0
unset AGENT_CRED_ISOLATION_SCRIPT_LOADED
# shellcheck source=agent-cred-isolation.sh
source "${KIRO_WRAPPER_ISOLATION_SCRIPT}" || fail "slot allocation failed"

[[ -n "${AGENT_CRED_ISOLATION_SLOT_ROOT:-}" ]] || fail "slot root is unset"
[[ -n "${XDG_DATA_HOME:-}" ]] || fail "XDG_DATA_HOME is unset"
case "${XDG_DATA_HOME}" in
  "${AGENT_CRED_ISOLATION_SLOT_ROOT}/xdg-data") ;;
  *) fail "XDG_DATA_HOME is outside the allocated pane slot" ;;
esac

credential_store="${XDG_DATA_HOME}/kiro-cli/data.sqlite3"
[[ ! -L "${credential_store}" ]] || fail "credential store is a symlink"

# Refuse two live Kiro parents with the same writable XDG data home. Reading
# environment paths only; credential contents are never inspected.
for env_file in /proc/[0-9]*/environ; do
  [[ -r "${env_file}" ]] || continue
  other_pid="${env_file#/proc/}"
  other_pid="${other_pid%/environ}"
  [[ "${other_pid}" != "$$" ]] || continue
  cmdline="$(tr '\0' ' ' <"/proc/${other_pid}/cmdline" 2>/dev/null || true)"
  case "${cmdline}" in
    *kiro-cli*|*kiro-cli-chat*) ;;
    *) continue ;;
  esac
  other_xdg="$(tr '\0' '\n' <"${env_file}" 2>/dev/null | sed -n 's/^XDG_DATA_HOME=//p' | head -1 || true)"
  [[ "${other_xdg}" != "${XDG_DATA_HOME}" ]] || fail "slot already used by live process ${other_pid}"
done

exec "${KIRO_WRAPPER_BIN}" "$@"
