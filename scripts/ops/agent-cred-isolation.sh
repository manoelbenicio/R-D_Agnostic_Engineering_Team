# Source this file from ~/.bashrc. It assigns one credential home to each
# explicit stable agent UUID + provider-subscription fingerprint binding.
# Missing or invalid identity fails closed; terminal, pane, PID, TTY, and random
# identities are intentionally unsupported.

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

agent_cred_isolation_identity() {
  python3 - "${AGENT_CRED_ISOLATION_AGENT_ID:-}" \
    "${AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT:-}" <<'PY'
import hashlib
import re
import sys
import uuid

agent_id, fingerprint = sys.argv[1:]
if not agent_id:
    raise SystemExit("AGENT_CRED_ISOLATION_AGENT_ID is required")
if not fingerprint:
    raise SystemExit("AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT is required")
try:
    parsed = uuid.UUID(agent_id)
except (ValueError, AttributeError):
    raise SystemExit("AGENT_CRED_ISOLATION_AGENT_ID must be a canonical UUID")
if parsed.int == 0 or str(parsed) != agent_id:
    raise SystemExit("AGENT_CRED_ISOLATION_AGENT_ID must be a canonical non-zero UUID")
if re.fullmatch(r"[0-9a-f]{64}", fingerprint) is None:
    raise SystemExit("AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT must be 64 lowercase hex characters")
digest = hashlib.sha256(("agent-credential-binding-v2\0" + agent_id + "\0" + fingerprint).encode()).hexdigest()
print("sha256:" + digest)
PY
}

# Return a comma-separated set of slot numbers referenced by OS processes.
# Production reconciliation requires euid 0 so every UID can be inspected;
# unreadable processes fail closed. The isolated harness has a /tmp-only mode
# that scans accessible same-UID synthetic processes without weakening the
# production path.
agent_cred_isolation_referenced_slots_locked() {
  python3 - "${AGENT_CRED_ISOLATION_ROOT}" <<'PY'
import os
import re
import sys

root = os.path.realpath(sys.argv[1])
slots_root = os.path.join(root, "slots")
pattern = re.compile(re.escape(slots_root) + r"/slot-([0-9]+)(?:/|$)")
referenced = set()
our_uid = os.geteuid()
test_allow_unreadable = os.environ.get("AGENT_CRED_ISOLATION_TEST_ALLOW_UNREADABLE_PROC") == "1"
if test_allow_unreadable and not (root == "/tmp" or root.startswith("/tmp/")):
    raise SystemExit("test proc-audit override is restricted to /tmp roots")
if not test_allow_unreadable and our_uid != 0:
    raise SystemExit("stale reconciliation requires root for a complete OS-process reference audit")

def collect(value):
    if not value:
        return
    value = value.removesuffix(" (deleted)")
    for match in pattern.finditer(value):
        referenced.add(int(match.group(1)))

for name in os.listdir("/proc"):
    if not name.isdigit():
        continue
    base = os.path.join("/proc", name)
    try:
        with open(os.path.join(base, "status"), "r", encoding="utf-8", errors="replace") as source:
            uid_line = next((line for line in source if line.startswith("Uid:")), "")
        if not uid_line:
            continue
        process_uid = int(uid_line.split()[1])
        if test_allow_unreadable and process_uid != our_uid:
            continue
    except (FileNotFoundError, ProcessLookupError):
        continue
    except (OSError, ValueError) as error:
        if test_allow_unreadable:
            continue
        raise SystemExit(f"cannot inspect process {name}: {error}")

    try:
        with open(os.path.join(base, "environ"), "rb") as source:
            for entry in source.read().split(b"\0"):
                if b"=" in entry:
                    collect(entry.split(b"=", 1)[1].decode("utf-8", "surrogateescape"))
        for link_name in ("cwd", "root"):
            try:
                collect(os.readlink(os.path.join(base, link_name)))
            except (FileNotFoundError, ProcessLookupError):
                pass
        fd_root = os.path.join(base, "fd")
        try:
            fd_names = os.listdir(fd_root)
        except (FileNotFoundError, ProcessLookupError):
            fd_names = []
        for fd_name in fd_names:
            try:
                collect(os.readlink(os.path.join(fd_root, fd_name)))
            except (FileNotFoundError, ProcessLookupError):
                pass
    except (FileNotFoundError, ProcessLookupError):
        continue
    except PermissionError as error:
        if test_allow_unreadable:
            continue
        raise SystemExit(f"cannot inspect process {name}: {error}")

print(",".join(str(slot) for slot in sorted(referenced)))
PY
}

agent_cred_isolation_allocate_slot_locked() {
  local identity_key="$1"
  local referenced_slots="$2"
  local now
  now="$(agent_cred_isolation_now)"

  python3 - "${AGENT_CRED_ISOLATION_REGISTRY}" "${AGENT_CRED_ISOLATION_ROOT}" \
    "${identity_key}" "${AGENT_CRED_ISOLATION_AGENT_ID}" \
    "${AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT}" "${now}" \
    "${referenced_slots}" "${AGENT_CRED_ISOLATION_ADOPT_SLOT:-}" \
    "${AGENT_CRED_ISOLATION_RECONCILE_STALE:-0}" <<'PY'
import json
import os
import re
import shutil
import sys

(
    registry_path,
    root,
    identity_key,
    agent_id,
    fingerprint,
    now,
    referenced_text,
    adopt_text,
    reconcile_text,
) = sys.argv[1:]
slots_root = os.path.join(root, "slots")
referenced = {int(value) for value in referenced_text.split(",") if value}
reconcile = reconcile_text == "1"
if reconcile_text not in ("0", "1"):
    raise SystemExit("AGENT_CRED_ISOLATION_RECONCILE_STALE must be 0 or 1")

adopt_slot = None
if adopt_text:
    if re.fullmatch(r"[1-9][0-9]*", adopt_text) is None:
        raise SystemExit("AGENT_CRED_ISOLATION_ADOPT_SLOT must be a positive integer")
    adopt_slot = int(adopt_text)

physical = {}
for name in os.listdir(slots_root):
    match = re.fullmatch(r"slot-([0-9]+)", name)
    if match is None:
        raise SystemExit(f"unexpected entry under slots root: {name}")
    slot = int(match.group(1))
    path = os.path.join(slots_root, name)
    if os.path.islink(path) or not os.path.isdir(path):
        raise SystemExit(f"slot-{slot:02d} is not a physical directory")
    physical[slot] = path

registry = None
if os.path.exists(registry_path):
    with open(registry_path, "r", encoding="utf-8") as source:
        registry = json.load(source)

# Legacy v1 contains terminal identities and cannot be trusted as stable. It may
# be converted only by explicitly adopting the sole surviving physical slot.
if registry is not None and registry.get("version") == 1:
    try:
        legacy_next = int(registry.get("next_slot", 1))
    except (TypeError, ValueError):
        raise SystemExit("legacy registry has an invalid next_slot")
    if physical:
        if adopt_slot is None or set(physical) != {adopt_slot}:
            raise SystemExit("legacy registry requires explicit adoption of its sole physical slot")
    registry = {
        "version": 2,
        "next_slot": max(1, legacy_next),
        "bindings": {},
        "slots": {},
    }
elif registry is None:
    registry = {"version": 2, "next_slot": 1, "bindings": {}, "slots": {}}

if (
    registry.get("version") != 2
    or not isinstance(registry.get("bindings"), dict)
    or not isinstance(registry.get("slots"), dict)
):
    raise SystemExit("registry.json has an unsupported schema")
try:
    next_slot = int(registry.get("next_slot", 1))
except (TypeError, ValueError):
    raise SystemExit("registry.json has an invalid next_slot")
if next_slot < 1:
    raise SystemExit("registry.json has an invalid next_slot")

bindings = registry["bindings"]
slots = registry["slots"]

# Validate the complete bidirectional registry before changing anything.
for key, entry in list(bindings.items()):
    if not isinstance(key, str) or not isinstance(entry, dict):
        raise SystemExit("registry binding is invalid")
    try:
        slot = int(entry["slot"])
    except (KeyError, TypeError, ValueError):
        raise SystemExit("registry binding has an invalid slot")
    if slot < 1 or entry.get("identity") != key:
        raise SystemExit("registry binding identity mismatch")
    owner = slots.get(str(slot))
    if not isinstance(owner, dict) or owner.get("identity") != key:
        raise SystemExit("registry slot ownership mismatch")
for slot_text, entry in slots.items():
    try:
        slot = int(slot_text)
    except (TypeError, ValueError):
        raise SystemExit("registry slot key is invalid")
    if slot < 1 or not isinstance(entry, dict):
        raise SystemExit("registry slot entry is invalid")
    key = entry.get("identity")
    if key not in bindings or int(bindings[key]["slot"]) != slot:
        raise SystemExit("registry slot reverse ownership mismatch")

# A missing current home is credential loss/corruption and must never trigger a
# replacement allocation. Missing homes for other bindings are stale metadata
# and can be safely removed because no physical credential folder exists.
for key, entry in list(bindings.items()):
    slot = int(entry["slot"])
    if slot not in physical:
        if key == identity_key:
            raise SystemExit("current stable binding points to a missing physical slot")
        del bindings[key]
        del slots[str(slot)]

# Explicitly adopt the sole existing unowned home. This is the safe migration
# path for ORQ2 slot-185; its contents and path are neither moved nor overwritten.
if identity_key not in bindings and adopt_slot is not None:
    if adopt_slot not in physical:
        raise SystemExit("requested adoption slot does not exist")
    if str(adopt_slot) in slots:
        raise SystemExit("requested adoption slot is already owned")
    if any(entry.get("agent_id") == agent_id for entry in bindings.values()):
        raise SystemExit("agent already has a different stable binding")
    bindings[identity_key] = {
        "identity": identity_key,
        "agent_id": agent_id,
        "subscription_fingerprint": fingerprint,
        "slot": adopt_slot,
        "first_seen": now,
        "last_seen": now,
    }
    slots[str(adopt_slot)] = {"identity": identity_key, "created_at": now}
    next_slot = max(next_slot, adopt_slot + 1)

# Unknown physical homes are never silently treated as stale. Reconciliation is
# explicit and still refuses any slot referenced by a live same-UID process.
owned_slots = {int(value) for value in slots}
orphans = sorted(set(physical) - owned_slots)
if orphans:
    active = sorted(set(orphans) & referenced)
    if active:
        raise SystemExit("refusing to reconcile live unowned slot(s): " + ",".join(map(str, active)))
    if not reconcile:
        raise SystemExit("unowned physical slot(s) require explicit stale reconciliation")
    for slot in orphans:
        shutil.rmtree(physical[slot])
        del physical[slot]

# One agent UUID may have only one active subscription binding. A subscription
# transition is explicit: stale reconciliation removes the old unreferenced home
# before any new folder is allocated, so no historical physical folder remains.
other_agent_keys = [
    key for key, entry in bindings.items()
    if key != identity_key and entry.get("agent_id") == agent_id
]
if other_agent_keys:
    old_slots = sorted(int(bindings[key]["slot"]) for key in other_agent_keys)
    active = sorted(set(old_slots) & referenced)
    if active:
        raise SystemExit("refusing to replace live agent binding slot(s): " + ",".join(map(str, active)))
    if not reconcile:
        raise SystemExit("agent subscription changed; explicit stale reconciliation is required")
    for key in other_agent_keys:
        slot = int(bindings[key]["slot"])
        path = os.path.join(slots_root, f"slot-{slot:02d}")
        if os.path.lexists(path):
            if os.path.islink(path) or not os.path.isdir(path):
                raise SystemExit(f"refusing to remove non-directory slot-{slot:02d}")
            shutil.rmtree(path)
        del bindings[key]
        del slots[str(slot)]
        physical.pop(slot, None)

entry = bindings.get(identity_key)
if entry is None:
    used = set(physical) | {int(value) for value in slots}
    slot = next_slot
    while slot in used:
        slot += 1
    next_slot = slot + 1
    entry = {
        "identity": identity_key,
        "agent_id": agent_id,
        "subscription_fingerprint": fingerprint,
        "slot": slot,
        "first_seen": now,
        "last_seen": now,
    }
    bindings[identity_key] = entry
    slots[str(slot)] = {"identity": identity_key, "created_at": now}
else:
    slot = int(entry["slot"])
    if entry.get("agent_id") != agent_id or entry.get("subscription_fingerprint") != fingerprint:
        raise SystemExit("stable binding metadata mismatch")
    if slot not in physical:
        raise SystemExit("stable binding physical slot is missing")
    entry["last_seen"] = now

registry["next_slot"] = next_slot
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
    if ! cp -aL -- "${source}" "${destination}"; then
      rm -rf -- "${destination}"
      return 1
    fi
  else
    mkdir -p "${destination}" || return 1
  fi
  chmod 700 "${destination}" 2>/dev/null || true
}

agent_cred_isolation_migrate_slot() {
  local slot_root="$1"

  # Copy native vendor stores once. Existing slot contents are the source of
  # truth and are never overwritten, including an explicitly adopted slot-185.
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.codex" "${slot_root}/codex" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.cline" "${slot_root}/cline" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_HOME}/.gemini/antigravity-cli" "${slot_root}/home/.gemini/antigravity-cli" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/kiro-cli" "${slot_root}/xdg-data/kiro-cli" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME}/opencode" "${slot_root}/xdg-data/opencode" || return 1
  agent_cred_isolation_copy_dir_once "${AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME}/opencode" "${slot_root}/xdg-config/opencode" || return 1
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

agent_cred_isolation_export_slot() {
  local identity_key="$1"
  local slot="$2"
  local slot_name slot_root
  printf -v slot_name 'slot-%02d' "${slot}"
  slot_root="${AGENT_CRED_ISOLATION_ROOT}/slots/${slot_name}"

  agent_cred_isolation_migrate_slot "${slot_root}" || return 1

  export AGENT_CRED_ISOLATION_IDENTITY="${identity_key}"
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
  local lock_fd identity_key referenced_slots slot

  agent_cred_isolation_require || return 1
  agent_cred_isolation_init_paths
  # Validate stable identity before creating any registry, lock, or slot path.
  identity_key="$(agent_cred_isolation_identity)" || return 1

  mkdir -p "${AGENT_CRED_ISOLATION_ROOT}/slots" || return 1
  chmod 700 "${AGENT_CRED_ISOLATION_ROOT}" "${AGENT_CRED_ISOLATION_ROOT}/slots" || return 1

  exec {lock_fd}>"${AGENT_CRED_ISOLATION_LOCK}" || return 1
  flock -x "${lock_fd}" || {
    exec {lock_fd}>&-
    return 1
  }
  if [[ "${AGENT_CRED_ISOLATION_RECONCILE_STALE:-0}" == "1" ]]; then
    referenced_slots="$(agent_cred_isolation_referenced_slots_locked)" || {
      flock -u "${lock_fd}"
      exec {lock_fd}>&-
      return 1
    }
  else
    referenced_slots=""
  fi
  slot="$(agent_cred_isolation_allocate_slot_locked "${identity_key}" "${referenced_slots}")" || {
    flock -u "${lock_fd}"
    exec {lock_fd}>&-
    return 1
  }
  agent_cred_isolation_export_slot "${identity_key}" "${slot}" || {
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
  printf 'agent=%s identity=%s slot=%s root=%s\n' \
    "${AGENT_CRED_ISOLATION_AGENT_ID}" \
    "${AGENT_CRED_ISOLATION_IDENTITY}" \
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
Required environment:
  AGENT_CRED_ISOLATION_AGENT_ID=<canonical non-zero UUID>
  AGENT_CRED_ISOLATION_SUBSCRIPTION_FINGERPRINT=<64 lowercase sha256 hex>

Usage:
  source scripts/ops/agent-cred-isolation.sh
  scripts/ops/agent-cred-isolation.sh migrate
  scripts/ops/agent-cred-isolation.sh status
  scripts/ops/agent-cred-isolation.sh reconcile

`reconcile` explicitly removes only unreferenced stale homes while holding the
registry lock. AGENT_CRED_ISOLATION_ADOPT_SLOT=<n> explicitly binds a sole legacy
physical home (for example slot-185) without moving or overwriting it.
USAGE
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-status}" in
    migrate) agent_cred_isolation_bootstrap ;;
    status) agent_cred_isolation_status ;;
    reconcile)
      AGENT_CRED_ISOLATION_RECONCILE_STALE=1 agent_cred_isolation_bootstrap
      ;;
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
