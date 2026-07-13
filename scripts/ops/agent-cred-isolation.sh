#!/usr/bin/env bash
# Source this file from ~/.bashrc. It assigns one credential slot to each
# stable Herdr terminal and never falls back to shared vendor homes.

if [[ -n "${AGENT_CRED_ISOLATION_SCRIPT_LOADED:-}" ]]; then
  return 0 2>/dev/null || exit 0
fi

agent_cred_isolation_now() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

agent_cred_isolation_die() {
  printf 'agent-cred-isolation: %s\n' "$*" >&2
  return 1
}

agent_cred_isolation_require() {
  command -v flock >/dev/null 2>&1 || { agent_cred_isolation_die 'flock is required'; return 1; }
  command -v python3 >/dev/null 2>&1 || { agent_cred_isolation_die 'python3 is required'; return 1; }
  command -v sha256sum >/dev/null 2>&1 || { agent_cred_isolation_die 'sha256sum is required'; return 1; }
  command -v cp >/dev/null 2>&1 || { agent_cred_isolation_die 'cp is required'; return 1; }
}

agent_cred_isolation_init_paths() {
  : "${AGENT_CRED_ISOLATION_HOST_HOME:=${HOME:?HOME is required}}"
  : "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME:=${XDG_DATA_HOME:-${AGENT_CRED_ISOLATION_HOST_HOME}/.local/share}}"
  : "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME:=${XDG_CONFIG_HOME:-${AGENT_CRED_ISOLATION_HOST_HOME}/.config}}"
  : "${AGENT_CRED_ISOLATION_ROOT:=${AGENT_CRED_ISOLATION_HOST_HOME}/.agent-cred-homes}"

  AGENT_CRED_ISOLATION_REGISTRY="${AGENT_CRED_ISOLATION_ROOT}/registry.json"
  AGENT_CRED_ISOLATION_LOCK="${AGENT_CRED_ISOLATION_ROOT}/registry.lock"
  export AGENT_CRED_ISOLATION_HOST_HOME
  export AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME
  export AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME
  export AGENT_CRED_ISOLATION_ROOT
  export AGENT_CRED_ISOLATION_REGISTRY
}

agent_cred_isolation_terminal_id_locked() {
  local pane_details terminal_id tty_name fallback_key tty_hash fallback_file fallback_uuid

  if [[ -n "${HERDR_PANE_ID:-}" ]] && command -v herdr >/dev/null 2>&1; then
    pane_details="$(herdr pane get "${HERDR_PANE_ID}" 2>/dev/null || true)"
    terminal_id="$(python3 -c '
import json
import sys
try:
    pane = json.load(sys.stdin)["result"]["pane"]
except (KeyError, TypeError, json.JSONDecodeError):
    raise SystemExit(0)
value = pane.get("terminal_id", "")
if isinstance(value, str) and value:
    print(value)
' <<<"${pane_details}")"
    if [[ -n "${terminal_id}" ]]; then
      printf 'herdr:%s\n' "${terminal_id}"
      return 0
    fi
  fi

  tty_name="$(tty 2>/dev/null || true)"
  if [[ -n "${tty_name}" && "${tty_name}" != "not a tty" ]]; then
    fallback_key="tty:${tty_name}"
  elif [[ -n "${HERDR_PANE_ID:-}" ]]; then
    # A pane id is not durable, but it is safer than a shared no-TTY bucket:
    # if Herdr lookup is unavailable, a recompact allocates a fresh slot rather
    # than ever falling back to a shared vendor home.
    fallback_key="pane:${HERDR_PANE_ID}"
  else
    fallback_key="process:${PPID}:${BASHPID}"
  fi
  tty_hash="$(printf '%s' "${fallback_key}" | sha256sum | awk '{print $1}')"
  fallback_file="${AGENT_CRED_ISOLATION_ROOT}/fallback-terminals/${tty_hash}.uuid"
  mkdir -p "${AGENT_CRED_ISOLATION_ROOT}/fallback-terminals" || return 1
  chmod 700 "${AGENT_CRED_ISOLATION_ROOT}" "${AGENT_CRED_ISOLATION_ROOT}/fallback-terminals" || return 1
  if [[ -r "${fallback_file}" ]]; then
    fallback_uuid="$(<"${fallback_file}")"
  else
    fallback_uuid="$(python3 -c 'import uuid; print(uuid.uuid4())')" || return 1
    (umask 077; printf '%s\n' "${fallback_uuid}" >"${fallback_file}") || return 1
  fi
  printf 'tty:%s:%s\n' "${tty_hash}" "${fallback_uuid}"
}

agent_cred_isolation_allocate_slot_locked() {
  local terminal_id="$1"
  local now
  now="$(agent_cred_isolation_now)"
  python3 - "${AGENT_CRED_ISOLATION_REGISTRY}" "${terminal_id}" "${now}" <<'PY'
import json
import os
import sys

registry_path, terminal_id, now = sys.argv[1:]
registry = {"version": 1, "next_slot": 1, "terminals": {}, "slots": {}}
if os.path.exists(registry_path):
    with open(registry_path, "r", encoding="utf-8") as source:
        registry = json.load(source)

if (
    registry.get("version") != 1
    or not isinstance(registry.get("terminals"), dict)
    or not isinstance(registry.get("slots"), dict)
):
    raise SystemExit("registry.json has an unsupported schema")

try:
    next_slot = int(registry.get("next_slot", 1))
except (TypeError, ValueError):
    raise SystemExit("registry.json has an invalid next_slot")
if next_slot < 1:
    raise SystemExit("registry.json has an invalid next_slot")

entry = registry["terminals"].get(terminal_id)
if entry is None:
    slot = next_slot
    while str(slot) in registry["slots"]:
        slot += 1
    registry["next_slot"] = slot + 1
    entry = {"slot": slot, "first_seen": now, "last_seen": now}
    registry["terminals"][terminal_id] = entry
    registry["slots"][str(slot)] = {"terminal_id": terminal_id, "created_at": now}
else:
    try:
        slot = int(entry["slot"])
    except (KeyError, TypeError, ValueError):
        raise SystemExit("registry terminal has an invalid slot")
    if slot < 1:
        raise SystemExit("registry terminal has an invalid slot")
    entry["last_seen"] = now
    owner = registry["slots"].get(str(slot), {}).get("terminal_id")
    if owner != terminal_id:
        raise SystemExit("registry slot ownership mismatch")

tmp_path = registry_path + ".tmp." + str(os.getpid())
with open(tmp_path, "w", encoding="utf-8") as target:
    json.dump(registry, target, indent=2, sort_keys=True)
    target.write("\n")
os.chmod(tmp_path, 0o600)
os.replace(tmp_path, registry_path)
print(slot)
PY
}

agent_cred_isolation_copy_dir_once() {
  local source="$1"
  local destination="$2"

  if [[ -e "${destination}" || -L "${destination}" ]]; then
    return 0
  fi
  mkdir -p "$(dirname -- "${destination}")" || return 1
  if [[ -e "${source}" || -L "${source}" ]]; then
    # Dereference legacy symlinks. The destination must be a physical copy so
    # a vendor token refresh cannot write back into a shared source home.
    if ! cp -aL -- "${source}" "${destination}"; then
      rm -rf -- "${destination}"
      return 1
    fi
  else
    mkdir -p "${destination}" || return 1
  fi
  chmod 700 "${destination}" 2>/dev/null || true
}

agent_cred_isolation_copy_file_once() {
  local source="$1"
  local destination="$2"

  if [[ -e "${destination}" || -L "${destination}" ]]; then
    return 0
  fi
  mkdir -p "$(dirname -- "${destination}")" || return 1
  if [[ -f "${source}" ]]; then
    if ! cp -p -- "${source}" "${destination}"; then
      rm -f -- "${destination}"
      return 1
    fi
    chmod 600 "${destination}" 2>/dev/null || true
  fi
}

agent_cred_isolation_migrate_slot() {
  local slot_root="$1"

  # Copy the native vendor stores once, preserving live OAuth state, SQLite
  # sidecars, and vendor-specific metadata. Never overwrite an initialized
  # slot: after migration its copy is the source of truth for that terminal.
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex" "${slot_root}/codex" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.cline" "${slot_root}/cline" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.gemini/antigravity-cli" "${slot_root}/home/.gemini/antigravity-cli" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/kiro-cli" "${slot_root}/xdg-data/kiro-cli" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/opencode" "${slot_root}/xdg-data/opencode" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/opencode" "${slot_root}/xdg-config/opencode" || return 1

  # Keep compatibility with hosts that enrolled GLM into an explicit `glm`
  # XDG directory. Fleet GLM agents that run through OpenCode use the copied
  # opencode directories above; both layouts remain physically per-slot.
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/glm" "${slot_root}/xdg-data/glm" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/glm" "${slot_root}/xdg-config/glm" || return 1

  mkdir -p "${slot_root}/codex" "${slot_root}/cline" \
    "${slot_root}/home/.gemini/antigravity-cli" "${slot_root}/xdg-data/kiro-cli" \
    "${slot_root}/xdg-data/opencode" "${slot_root}/xdg-config/opencode" \
    "${slot_root}/cline-sandbox" || return 1
  chmod 700 "${slot_root}" "${slot_root}/codex" "${slot_root}/cline" \
    "${slot_root}/home" "${slot_root}/home/.gemini" \
    "${slot_root}/home/.gemini/antigravity-cli" "${slot_root}/xdg-data" \
    "${slot_root}/xdg-config" "${slot_root}/cline-sandbox" || return 1
}

agent_cred_isolation_import_legacy_codex_homes_locked() {
  local legacy_home legacy_name legacy_slot legacy_root previous_nullglob
  previous_nullglob="$(shopt -p nullglob || true)"
  shopt -s nullglob
  for legacy_home in "${AGENT_CRED_ISOLATION_HOST_HOME}"/.codex-*; do
    [[ -d "${legacy_home}" ]] || continue
    legacy_name="${legacy_home##*/}"
    legacy_slot="$(agent_cred_isolation_allocate_slot_locked "legacy:codex:${legacy_name}")" || {
      eval "${previous_nullglob}"
      return 1
    }
    printf -v legacy_root '%s/slots/slot-%02d' "${AGENT_CRED_ISOLATION_ROOT}" "${legacy_slot}"
    mkdir -p "${legacy_root}/codex" || {
      eval "${previous_nullglob}"
      return 1
    }
    agent_cred_isolation_copy_file_once "${legacy_home}/auth.json" "${legacy_root}/codex/auth.json" || {
      eval "${previous_nullglob}"
      return 1
    }
    agent_cred_isolation_copy_file_once "${legacy_home}/config.toml" "${legacy_root}/codex/config.toml" || {
      eval "${previous_nullglob}"
      return 1
    }
    chmod 700 "${legacy_root}" "${legacy_root}/codex" || {
      eval "${previous_nullglob}"
      return 1
    }
  done
  eval "${previous_nullglob}"
}

agent_cred_isolation_export_slot() {
  local terminal_id="$1"
  local slot="$2"
  local slot_name slot_root
  printf -v slot_name 'slot-%02d' "${slot}"
  slot_root="${AGENT_CRED_ISOLATION_ROOT}/slots/${slot_name}"

  agent_cred_isolation_migrate_slot "${slot_root}" || return 1

  export AGENT_CRED_ISOLATION_TERMINAL_ID="${terminal_id}"
  export AGENT_CRED_ISOLATION_SLOT="${slot}"
  export AGENT_CRED_ISOLATION_SLOT_NAME="${slot_name}"
  export AGENT_CRED_ISOLATION_SLOT_ROOT="${slot_root}"
  export CODEX_HOME="${slot_root}/codex"
  export XDG_DATA_HOME="${slot_root}/xdg-data"
  export XDG_CONFIG_HOME="${slot_root}/xdg-config"
  export CLINE_DATA_DIR="${slot_root}/cline"
  export CLINE_SANDBOX="1"
  export CLINE_SANDBOX_DATA_DIR="${slot_root}/cline-sandbox"
  export HOME="${slot_root}/home"
}

agent_cred_isolation_bootstrap() {
  local lock_fd terminal_id slot

  agent_cred_isolation_require || return 1
  agent_cred_isolation_init_paths
  mkdir -p "${AGENT_CRED_ISOLATION_ROOT}/slots" || return 1
  chmod 700 "${AGENT_CRED_ISOLATION_ROOT}" "${AGENT_CRED_ISOLATION_ROOT}/slots" || return 1

  exec {lock_fd}>"${AGENT_CRED_ISOLATION_LOCK}" || return 1
  flock -x "${lock_fd}" || {
    exec {lock_fd}>&-
    return 1
  }
  agent_cred_isolation_import_legacy_codex_homes_locked || {
    flock -u "${lock_fd}"
    exec {lock_fd}>&-
    return 1
  }
  terminal_id="$(agent_cred_isolation_terminal_id_locked)" || {
    flock -u "${lock_fd}"
    exec {lock_fd}>&-
    return 1
  }
  slot="$(agent_cred_isolation_allocate_slot_locked "${terminal_id}")" || {
    flock -u "${lock_fd}"
    exec {lock_fd}>&-
    return 1
  }
  agent_cred_isolation_export_slot "${terminal_id}" "${slot}" || {
    flock -u "${lock_fd}"
    exec {lock_fd}>&-
    return 1
  }
  flock -u "${lock_fd}"
  exec {lock_fd}>&-
}

agent_cred_isolation_vendor_state() {
  local vendor="$1"
  local path="$2"
  local account="${AGENT_CRED_ISOLATION_SLOT_NAME:-slot-unknown}"
  local state="off"

  case "${vendor}" in
    codex) [[ -f "${path}/auth.json" ]] && state="on" ;;
    cline) [[ -f "${path}/data/settings/providers.json" || -f "${path}/settings/providers.json" ]] && state="on" ;;
    agy) [[ -f "${path}/.gemini/antigravity-cli/antigravity-oauth-token" ]] && state="on" ;;
    kiro) [[ -f "${path}/kiro-cli/data.sqlite3" ]] && state="on" ;;
    opencode) [[ -f "${path}/opencode/auth.json" ]] && state="on" ;;
    glm) [[ -f "${path}/glm/auth.json" || -f "${path}/opencode/auth.json" ]] && state="on" ;;
  esac
  printf 'vendor=%s account=%s state=%s path=%s\n' "${vendor}" "${account}" "${state}" "${path}"
}

agent_cred_isolation_status() {
  agent_cred_isolation_bootstrap || return 1
  printf 'pane=%s terminal=%s slot=%s root=%s\n' \
    "${HERDR_PANE_ID:-fallback}" \
    "${AGENT_CRED_ISOLATION_TERMINAL_ID}" \
    "${AGENT_CRED_ISOLATION_SLOT}" \
    "${AGENT_CRED_ISOLATION_SLOT_ROOT}"
  agent_cred_isolation_vendor_state codex "${CODEX_HOME}"
  agent_cred_isolation_vendor_state cline "${CLINE_DATA_DIR}"
  agent_cred_isolation_vendor_state agy "${HOME}"
  agent_cred_isolation_vendor_state kiro "${XDG_DATA_HOME}"
  agent_cred_isolation_vendor_state opencode "${XDG_DATA_HOME}"
  agent_cred_isolation_vendor_state glm "${XDG_DATA_HOME}"
}

agent_cred_isolation_usage() {
  cat <<'USAGE'
Usage:
  source scripts/ops/agent-cred-isolation.sh
  scripts/ops/agent-cred-isolation.sh migrate
  scripts/ops/agent-cred-isolation.sh status
  scripts/ops/agent-cred-isolation.sh doctor status

The sourceable form allocates a stable Herdr terminal slot and exports:
CODEX_HOME; XDG_DATA_HOME/XDG_CONFIG_HOME (Kiro, OpenCode, GLM);
CLINE_DATA_DIR/CLINE_SANDBOX_DATA_DIR (Cline); and HOME (Antigravity/agy).
USAGE
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-status}" in
    migrate) agent_cred_isolation_bootstrap ;;
    status) agent_cred_isolation_status ;;
    doctor)
      [[ "${2:-status}" == "status" ]] || { agent_cred_isolation_usage; exit 2; }
      agent_cred_isolation_status
      ;;
    -h|--help|help) agent_cred_isolation_usage ;;
    *) agent_cred_isolation_usage; exit 2 ;;
  esac
elif [[ "$-" == *i* || "${AGENT_CRED_ISOLATION_AUTOSTART:-0}" == "1" ]]; then
  if agent_cred_isolation_bootstrap; then
    AGENT_CRED_ISOLATION_SCRIPT_LOADED=1
  else
    unset AGENT_CRED_ISOLATION_SCRIPT_LOADED
    return 1
  fi
fi
