#!/usr/bin/env bash
# Source this file from ~/.bashrc. It assigns one credential slot to each
# stable Herdr terminal and never falls back to shared vendor homes.

# Do not short-circuit when AGENT_CRED_ISOLATION_SCRIPT_LOADED is inherited.
# Herdr creates a new terminal from an existing pane environment, so a child
# shell can inherit the parent's slot variables even though it has a different
# terminal_id. Bootstrap is idempotent and must run in every interactive shell.

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

  if [[ "${AGENT_CRED_ISOLATION_MIGRATE_LEGACY:-0}" == "1" ]]; then
    # Migration is explicit because copying one shared credential store into
    # every new terminal would duplicate an identity across the fleet.
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex" "${slot_root}/codex" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.cline" "${slot_root}/cline" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.gemini/antigravity-cli" "${slot_root}/home/.gemini/antigravity-cli" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/kiro-cli" "${slot_root}/xdg-data/kiro-cli" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/opencode" "${slot_root}/xdg-data/opencode" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/opencode" "${slot_root}/xdg-config/opencode" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/glm" "${slot_root}/xdg-data/glm" || return 1
    agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/glm" "${slot_root}/xdg-config/glm" || return 1
  fi

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
  if [[ "${AGENT_CRED_ISOLATION_MIGRATE_LEGACY:-0}" == "1" ]]; then
    agent_cred_isolation_import_legacy_codex_homes_locked || {
      flock -u "${lock_fd}"
      exec {lock_fd}>&-
      return 1
    }
  fi
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

agent_cred_isolation_new_codex_login_home() {
  local login_root login_id login_home

  agent_cred_isolation_require || return 1
  agent_cred_isolation_init_paths
  agent_cred_isolation_release_codex_lease
  agent_cred_isolation_cleanup_codex_logins || return 1
  login_root="${AGENT_CRED_ISOLATION_ROOT}/codex-logins"
  login_id="login-$(date -u +%Y%m%dT%H%M%SZ)-$(python3 -c 'import uuid; print(uuid.uuid4())')"
  login_home="${login_root}/${login_id}"

  mkdir -p "${login_root}" || return 1
  chmod 700 "${AGENT_CRED_ISOLATION_ROOT}" "${login_root}" || return 1
  (umask 077; mkdir "${login_home}") || return 1

  # The lease is held by this shell for its whole login lifecycle.  flock is
  # owner-death safe; cleanup can therefore remove only an aged, unlocked
  # directory with a valid lease marker.
  local lease_path lease_fd
  lease_path="${login_home}/.lease"
  printf 'agent-cred-login-lease-v1\n' >"${lease_path}" || {
    rm -rf -- "${login_home}"
    return 1
  }
  chmod 600 "${lease_path}" || {
    rm -rf -- "${login_home}"
    return 1
  }
  exec {lease_fd}<>"${lease_path}" || {
    rm -rf -- "${login_home}"
    return 1
  }
  if ! flock -x "${lease_fd}"; then
    exec {lease_fd}>&-
    rm -rf -- "${login_home}"
    return 1
  fi
  AGENT_CRED_CODEX_LEASE_FD="${lease_fd}"
  AGENT_CRED_CODEX_LEASE_PATH="${lease_path}"
  AGENT_CRED_CODEX_LEASE_OWNER_BASHPID="${BASHPID}"

  # A login lifecycle starts without auth.json by definition. Configuration
  # may be copied, but credentials are never copied, linked, or shared.
  if [[ -f "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex/config.toml" ]]; then
    cp -p -- "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex/config.toml" \
      "${login_home}/config.toml" || {
      rm -rf -- "${login_home}"
      return 1
    }
    chmod 600 "${login_home}/config.toml" 2>/dev/null || true
  fi

  export CODEX_HOME="${login_home}"
  export AGENT_CRED_CODEX_LOGIN_ID="${login_id}"
  export AGENT_CRED_CODEX_LEASE_SHELL_PID="$$"
}

agent_cred_isolation_release_codex_lease() {
  if [[ "${AGENT_CRED_CODEX_LEASE_FD:-}" =~ ^[0-9]+$ ]]; then
    # Never unlock an inherited open-file description; close only this FD.
    eval "exec ${AGENT_CRED_CODEX_LEASE_FD}>&-" 2>/dev/null || true
  fi
  unset AGENT_CRED_CODEX_LEASE_FD AGENT_CRED_CODEX_LEASE_PATH AGENT_CRED_CODEX_LEASE_OWNER_BASHPID
}

agent_cred_isolation_cleanup_codex_logins() {
  local login_root stale_count

  agent_cred_isolation_init_paths
  login_root="${AGENT_CRED_ISOLATION_ROOT}/codex-logins"
  [[ -d "${login_root}" ]] || return 0
  # Automatic deletion is disabled. The 120-hour threshold is report-only
  # guidance until a separately audited descriptor-relative helper exists.
  # Do not inspect lease contents and do not delete any candidate.
  stale_count="$(find "${login_root}" -mindepth 1 -maxdepth 1 -type d \
    -name 'login-*' -mmin +7200 -print 2>/dev/null | wc -l)"
  : "${stale_count}"
  return 0
}

agent_cred_isolation_codex_binding_owned_by_shell() {
  [[ -n "${AGENT_CRED_ISOLATION_ROOT:-}" ]] || return 1
  [[ "${AGENT_CRED_CODEX_LEASE_SHELL_PID:-}" == "$$" ]] || return 1
  [[ "${CODEX_HOME:-}" == "${AGENT_CRED_ISOLATION_ROOT}/codex-logins/login-"* ]] || return 1
  [[ -d "${CODEX_HOME}" && ! -L "${CODEX_HOME}" ]] || return 1
  [[ ! -L "${CODEX_HOME}/auth.json" ]] || return 1
}

agent_cred_isolation_assert_codex_binding() {
  if ! agent_cred_isolation_codex_binding_owned_by_shell; then
    agent_cred_isolation_die "unsafe Codex binding: this shell does not own a private login folder"
    return 1
  fi
}

codex_recover() {
  local account="${1:-}" source_slot source_home

  account="${account^^}"
  case "${account}" in
    A) source_slot='slot-01' ;;
    B) source_slot='slot-02' ;;
    C) source_slot='slot-03' ;;
    D) source_slot='slot-04' ;;
    E) source_slot='slot-05' ;;
    F) source_slot='slot-13' ;;
    *)
      agent_cred_isolation_die 'usage: codex_recover A|B|C|D|E|F'
      return 2
      ;;
  esac

  agent_cred_isolation_init_paths
  source_home="${AGENT_CRED_ISOLATION_ROOT}/slots/${source_slot}/codex"
  if [[ ! -f "${source_home}/auth.json" || -L "${source_home}/auth.json" ]]; then
    agent_cred_isolation_die "saved account ${account} is unavailable or unsafe"
    return 1
  fi
  agent_cred_isolation_new_codex_login_home || return 1
  cp -p -- "${source_home}/auth.json" "${CODEX_HOME}/auth.json" || return 1
  chmod 600 "${CODEX_HOME}/auth.json" 2>/dev/null || true
  if [[ -f "${source_home}/config.toml" ]]; then
    cp -p -- "${source_home}/config.toml" "${CODEX_HOME}/config.toml" || return 1
    chmod 600 "${CODEX_HOME}/config.toml" 2>/dev/null || true
  fi
  agent_cred_isolation_assert_codex_binding
}

agent_cred_isolation_run_codex() {
  local argument force_new=0

  # The safety boundary is entirely local. Herdr identity, availability, and
  # registry state are deliberately not consulted here.
  agent_cred_isolation_require || return 70
  agent_cred_isolation_init_paths
  for argument in "$@"; do
    if [[ "${argument}" == "login" ]]; then
      force_new=1
      break
    fi
  done

  if (( force_new )) || ! agent_cred_isolation_codex_binding_owned_by_shell; then
    if ! agent_cred_isolation_new_codex_login_home; then
      agent_cred_isolation_die "refusing to start Codex because a new private login folder could not be created"
      return 70
    fi
  fi
  if ! agent_cred_isolation_assert_codex_binding; then
    agent_cred_isolation_die "refusing to start Codex with an unverified credential home"
    return 70
  fi
  touch "${CODEX_HOME}" || return 70
  command codex "$@"
}

# Hard local lifecycle guard for manual Codex launches. Every explicit login
# gets a fresh physical folder. A new shell also gets a fresh folder on its
# first Codex launch, so inherited environments cannot share credentials.
# `command codex` above bypasses this function and resolves the real CLI.
codex() {
  agent_cred_isolation_run_codex "$@"
}

agent_cred_isolation_grok_profile_name() {
  case "${1:-}" in
    A|B|C|D) printf '%s\n' "$1" ;;
    *) agent_cred_isolation_die 'GROK profile must be exactly A, B, C, or D'; return 2 ;;
  esac
}

agent_cred_isolation_grok_init_paths() {
  local path
  agent_cred_isolation_init_paths
  for path in "${AGENT_CRED_ISOLATION_ROOT}" \
    "${AGENT_CRED_ISOLATION_ROOT}/grok" \
    "${AGENT_CRED_ISOLATION_ROOT}/grok/profiles" \
    "${AGENT_CRED_ISOLATION_ROOT}/grok/terminal-bindings"; do
    [[ ! -L "${path}" ]] || { agent_cred_isolation_die "unsafe GROK symlink path"; return 1; }
    mkdir -p -- "${path}" || return 1
    chmod 700 "${path}" || return 1
  done
}

agent_cred_isolation_grok_profile_path() {
  local profile
  profile="$(agent_cred_isolation_grok_profile_name "${1:-}")" || return
  printf '%s/grok/profiles/grok-%s\n' "${AGENT_CRED_ISOLATION_ROOT}" "${profile,,}"
}

agent_cred_isolation_validate_grok_profile() {
  local profile="$1" candidate="$2" expected profiles_real candidate_real
  profile="$(agent_cred_isolation_grok_profile_name "${profile}")" || return
  expected="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  [[ "${candidate}" == "${expected}" && -d "${candidate}" && ! -L "${candidate}" ]] || return 1
  [[ ! -L "${AGENT_CRED_ISOLATION_ROOT}/grok" && ! -L "${AGENT_CRED_ISOLATION_ROOT}/grok/profiles" ]] || return 1
  profiles_real="$(readlink -f -- "${AGENT_CRED_ISOLATION_ROOT}/grok/profiles")" || return 1
  candidate_real="$(readlink -f -- "${candidate}")" || return 1
  [[ "${candidate_real}" == "${profiles_real}/grok-${profile,,}" ]]
}

agent_cred_isolation_grok_profile_init() {
  local profile profile_path
  profile="$(agent_cred_isolation_grok_profile_name "${1:-}")" || return
  agent_cred_isolation_grok_init_paths || return 1
  profile_path="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  if [[ -e "${profile_path}" || -L "${profile_path}" ]]; then
    agent_cred_isolation_validate_grok_profile "${profile}" "${profile_path}" || {
      agent_cred_isolation_die 'existing GROK profile is not a safe physical directory'
      return 1
    }
    return 0
  fi
  (umask 077; mkdir -- "${profile_path}") || return 1
  chmod 700 "${profile_path}" || return 1
  agent_cred_isolation_validate_grok_profile "${profile}" "${profile_path}"
}

agent_cred_isolation_grok_release_lease() {
  if [[ "${AGENT_CRED_GROK_LEASE_FD:-}" =~ ^[0-9]+$ ]]; then
    # Close only. Never explicitly unlock an inherited open-file description.
    eval "exec ${AGENT_CRED_GROK_LEASE_FD}>&-" 2>/dev/null || true
  fi
  unset AGENT_CRED_GROK_LEASE_FD AGENT_CRED_GROK_LEASE_OWNER_BASHPID
}

agent_cred_isolation_grok_acquire_lease() {
  local profile="$1" profile_path lease_path lease_fd
  profile_path="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  agent_cred_isolation_validate_grok_profile "${profile}" "${profile_path}" || return 1
  lease_path="${profile_path}/.writer.lease"
  [[ ! -L "${lease_path}" ]] || return 1
  if [[ ! -e "${lease_path}" ]]; then
    (umask 077; : >"${lease_path}") || return 1
    chmod 600 "${lease_path}" || return 1
  fi
  [[ -f "${lease_path}" && ! -L "${lease_path}" ]] || return 1
  exec {lease_fd}<>"${lease_path}" || return 1
  if ! flock -n -x "${lease_fd}"; then
    exec {lease_fd}>&-
    agent_cred_isolation_die "GROK profile ${profile} already has a writer"
    return 1
  fi
  AGENT_CRED_GROK_LEASE_FD="${lease_fd}"
  AGENT_CRED_GROK_LEASE_OWNER_BASHPID="${BASHPID}"
}

agent_cred_isolation_grok_terminal_binding_locked() {
  local terminal_id="$1" profile="${2:-}" binding_hash binding_path binding_tmp
  binding_hash="$(printf '%s' "${terminal_id}" | sha256sum | awk '{print $1}')" || return 1
  binding_path="${AGENT_CRED_ISOLATION_ROOT}/grok/terminal-bindings/${binding_hash}"
  if [[ -n "${profile}" ]]; then
    binding_tmp="${binding_path}.tmp.${BASHPID}"
    (umask 077; printf '%s\n' "${profile}" >"${binding_tmp}") || return 1
    chmod 600 "${binding_tmp}" || return 1
    mv -- "${binding_tmp}" "${binding_path}" || return 1
  else
    [[ -f "${binding_path}" && ! -L "${binding_path}" ]] || return 1
    <"${binding_path}" read -r profile || return 1
    agent_cred_isolation_grok_profile_name "${profile}" >/dev/null || return 1
    printf '%s\n' "${profile}"
  fi
}

agent_cred_isolation_grok_attach() {
  local profile profile_path lock_fd terminal_id
  profile="$(agent_cred_isolation_grok_profile_name "${1:-}")" || return
  agent_cred_isolation_grok_profile_init "${profile}" || return 1
  agent_cred_isolation_grok_release_lease
  exec {lock_fd}>"${AGENT_CRED_ISOLATION_LOCK}" || return 1
  flock -x "${lock_fd}" || { exec {lock_fd}>&-; return 1; }
  terminal_id="$(agent_cred_isolation_terminal_id_locked)" || { exec {lock_fd}>&-; return 1; }
  agent_cred_isolation_grok_terminal_binding_locked "${terminal_id}" "${profile}" || { exec {lock_fd}>&-; return 1; }
  exec {lock_fd}>&-
  agent_cred_isolation_grok_acquire_lease "${profile}" || return 1
  profile_path="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  export GROK_HOME="${profile_path}"
  export AGENT_CRED_GROK_PROFILE="${profile}"
  export AGENT_CRED_GROK_TERMINAL_ID="${terminal_id}"
}

agent_cred_isolation_grok_assert_binding() {
  local profile profile_path lock_fd terminal_id bound
  profile="$(agent_cred_isolation_grok_profile_name "${AGENT_CRED_GROK_PROFILE:-}")" || return 1
  [[ "${AGENT_CRED_GROK_LEASE_OWNER_BASHPID:-}" == "${BASHPID}" ]] || return 1
  [[ "${AGENT_CRED_GROK_LEASE_FD:-}" =~ ^[0-9]+$ ]] || return 1
  profile_path="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  [[ "${GROK_HOME:-}" == "${profile_path}" ]] || return 1
  agent_cred_isolation_validate_grok_profile "${profile}" "${profile_path}" || return 1
  exec {lock_fd}>"${AGENT_CRED_ISOLATION_LOCK}" || return 1
  flock -x "${lock_fd}" || { exec {lock_fd}>&-; return 1; }
  terminal_id="$(agent_cred_isolation_terminal_id_locked)" || { exec {lock_fd}>&-; return 1; }
  bound="$(agent_cred_isolation_grok_terminal_binding_locked "${terminal_id}")" || { exec {lock_fd}>&-; return 1; }
  exec {lock_fd}>&-
  [[ "${terminal_id}" == "${AGENT_CRED_GROK_TERMINAL_ID:-}" && "${bound}" == "${profile}" ]]
}

agent_cred_isolation_grok_status() {
  local profile profile_path lease_path lease_fd state='available'
  profile="$(agent_cred_isolation_grok_profile_name "${1:-${AGENT_CRED_GROK_PROFILE:-}}")" || return
  agent_cred_isolation_grok_init_paths || return 1
  profile_path="$(agent_cred_isolation_grok_profile_path "${profile}")" || return
  if ! agent_cred_isolation_validate_grok_profile "${profile}" "${profile_path}"; then
    printf 'grok_profile=%s state=missing-or-unsafe\n' "${profile}"
    return 0
  fi
  lease_path="${profile_path}/.writer.lease"
  if [[ -f "${lease_path}" && ! -L "${lease_path}" ]]; then
    exec {lease_fd}<>"${lease_path}" || return 1
    if flock -n -x "${lease_fd}"; then
      state='available'
    else
      state='busy'
    fi
    exec {lease_fd}>&-
  fi
  if agent_cred_isolation_grok_assert_binding 2>/dev/null; then state='owned'; fi
  printf 'grok_profile=%s state=%s path=%s\n' "${profile}" "${state}" "${profile_path}"
}

agent_cred_isolation_grok_device_login() {
  local profile
  profile="$(agent_cred_isolation_grok_profile_name "${1:-}")" || return
  agent_cred_isolation_grok_attach "${profile}" || return 1
  agent_cred_isolation_grok_assert_binding || {
    agent_cred_isolation_die 'unsafe GROK binding; device login refused'
    return 70
  }
  command grok device-login
}

agent_cred_isolation_cleanup_grok_logins() {
  # Report-only policy: no GROK profile or login candidate is ever deleted,
  # moved, quarantined, or approved for deletion by this script.
  return 0
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
  source scripts/ops/agent-cred-isolation.sh
  agent_cred_isolation_grok_profile_init A|B|C|D
  agent_cred_isolation_grok_attach A|B|C|D
  agent_cred_isolation_grok_status A|B|C|D
  agent_cred_isolation_grok_device_login A|B|C|D

The sourceable form allocates a stable Herdr terminal slot and exports:
CODEX_HOME; XDG_DATA_HOME/XDG_CONFIG_HOME (Kiro, OpenCode, GLM);
CLINE_DATA_DIR/CLINE_SANDBOX_DATA_DIR (Cline); and HOME (Antigravity/agy).
Legacy credential copying is disabled by default. Set
AGENT_CRED_ISOLATION_MIGRATE_LEGACY=1 only for an explicitly authorized,
one-time migration.
USAGE
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-status}" in
    migrate) agent_cred_isolation_bootstrap ;;
    status) agent_cred_isolation_status ;;
    grok-profile-init) agent_cred_isolation_grok_profile_init "${2:-}" ;;
    grok-profile-status) agent_cred_isolation_grok_status "${2:-}" ;;
    grok-device-login) agent_cred_isolation_grok_device_login "${2:-}" ;;
    doctor)
      [[ "${2:-status}" == "status" ]] || { agent_cred_isolation_usage; exit 2; }
      agent_cred_isolation_status
      ;;
    -h|--help|help) agent_cred_isolation_usage ;;
    *) agent_cred_isolation_usage; exit 2 ;;
  esac
elif [[ "$-" == *i* || "${AGENT_CRED_ISOLATION_AUTOSTART:-0}" == "1" ]]; then
  # Codex lifecycle protection is local and loads immediately. The legacy
  # multi-vendor/Herdr slot bootstrap is opt-in so shell startup never waits
  # for Herdr or performs a credential-tree migration.
  agent_cred_isolation_require || return 1
  agent_cred_isolation_init_paths
  if [[ "${AGENT_CRED_ISOLATION_ENABLE_VENDOR_SLOTS:-0}" == "1" ]]; then
    agent_cred_isolation_bootstrap || return 1
  fi
  AGENT_CRED_ISOLATION_SCRIPT_LOADED=1
fi
