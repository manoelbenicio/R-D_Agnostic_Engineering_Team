#!/usr/bin/env bash
# ORQ-37 / Wave B structural — user-scoped umask 0077 + private TMPDIR.
#
# Why this exists: /etc/bashrc sets `umask 002`, so every artifact created by
# the execution user is born world-readable and any containment has to be
# re-applied by hand. This installer hardens only what the execution user owns:
# a systemd *user* drop-in for the relevant units and an opt-in shell fragment.
# It never edits /etc/bashrc, never touches a system-scope unit, and never
# reads, moves or inspects a credential file.
#
# Default mode is a dry run: it prints exactly what would change and exits 0
# without writing. `--apply` writes; `--rollback` removes only what this script
# created. Both are idempotent.
#
# The target root is parameterizable so the test harness can exercise the real
# code path against a sandbox instead of the live home directory.
set -euo pipefail

readonly MANAGED_BEGIN='# >>> orq37 umask hardening (managed) >>>'
readonly MANAGED_END='# <<< orq37 umask hardening (managed) <<<'
readonly DROPIN_NAME='10-orq37-umask-hardening.conf'

usage() {
  cat <<'USAGE'
Usage: install-umask-hardening.sh [--apply|--rollback] [--root DIR] [--unit NAME]...

  --apply       write the drop-in and shell fragment (default: dry run)
  --rollback    remove only the artifacts this script created
  --root DIR    treat DIR as the user's home (default: $HOME); used by tests
  --unit NAME   systemd *user* unit to harden (repeatable). Default:
                multica-daemon-orq2-credential.service
  --tmpdir DIR  private TMPDIR to export (default: <root>/.private-tmp)

Exit codes: 0 ok, 2 usage, 3 refused (unsafe state).
USAGE
}

mode=dryrun
root="${HOME:-}"
tmpdir=""
units=()

while [ $# -gt 0 ]; do
  case "$1" in
    --apply) mode=apply; shift ;;
    --rollback) mode=rollback; shift ;;
    --root) root="${2:-}"; shift 2 ;;
    --tmpdir) tmpdir="${2:-}"; shift 2 ;;
    --unit) units+=("${2:-}"); shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'E_USAGE unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

[ -n "$root" ] || { printf 'E_ROOT_REQUIRED\n' >&2; exit 2; }
[ -d "$root" ] || { printf 'E_ROOT_NOT_A_DIRECTORY %s\n' "$root" >&2; exit 3; }
[ ${#units[@]} -gt 0 ] || units=("multica-daemon-orq2-credential.service")
[ -n "$tmpdir" ] || tmpdir="$root/.private-tmp"

# Canonicalize root and validate ownership and system path restrictions
readonly canonical_root="$(realpath -m "$root" 2>/dev/null || readlink -f "$root" 2>/dev/null || echo "$root")"

# Refuse symlink root pointing to system paths
if [ -L "$root" ]; then
  target_root="$(realpath "$root" 2>/dev/null || echo "")"
  case "$target_root" in
    /|/etc|/etc/*|/usr|/usr/*|/var|/var/*|/tmp|/tmp/*|/sys|/sys/*|/proc|/proc/*|/dev|/dev/*|/boot|/boot/*|/run|/run/*|/lib|/lib*|/opt|/opt/*|/srv|/srv/*)
      printf 'E_ROOT_REFUSED %s (symlink target %s)\n' "$root" "$target_root" >&2
      exit 3
      ;;
  esac
fi

# Ownership check for root directory
current_uid="$(id -u)"
root_owner="$(stat -c '%u' "$root" 2>/dev/null || true)"
if [ -n "$root_owner" ] && [ "$root_owner" -ne "$current_uid" ]; then
  printf 'E_ROOT_OWNERSHIP_MISMATCH root owner %s != current uid %s\n' "$root_owner" "$current_uid" >&2
  exit 3
fi

# Refuse absolute-root or system paths outright (including canonicalized paths)
case "$canonical_root" in
  /|/etc|/etc/*|/usr|/usr/*|/var|/var/*|/tmp|/sys|/sys/*|/proc|/proc/*|/dev|/dev/*|/boot|/boot/*|/run|/run/*|/lib|/lib*|/opt|/opt/*|/srv|/srv/*)
    printf 'E_ROOT_REFUSED %s\n' "$canonical_root" >&2
    exit 3
    ;;
esac

# Validate unit names against path traversal, slashes, whitespace, and invalid characters
for unit in "${units[@]}"; do
  if [ -z "$unit" ] || [[ "$unit" =~ [/\\[:space:]] ]] || [[ "$unit" == *..* ]] || ! [[ "$unit" =~ ^[a-zA-Z0-9_.-]+$ ]]; then
    printf 'E_UNIT_INVALID %s\n' "$unit" >&2
    exit 3
  fi
done

# Canonicalize and validate tmpdir
readonly canonical_tmpdir="$(realpath -m "$tmpdir" 2>/dev/null || echo "$tmpdir")"
case "$canonical_tmpdir" in
  /|/etc|/etc/*|/usr|/usr/*|/var|/var/*|/tmp|/sys|/sys/*|/proc|/proc/*|/dev|/dev/*|/boot|/boot/*|/run|/run/*|/lib|/lib*|/opt|/opt/*|/srv|/srv/*)
    printf 'E_TMPDIR_REFUSED %s\n' "$canonical_tmpdir" >&2
    exit 3
    ;;
esac

if [[ "$canonical_tmpdir" != "$canonical_root"* ]]; then
  printf 'E_TMPDIR_OUTSIDE_ROOT %s outside %s\n' "$canonical_tmpdir" "$canonical_root" >&2
  exit 3
fi

if [ -L "$tmpdir" ]; then
  printf 'E_TMPDIR_SYMLINK %s\n' "$tmpdir" >&2
  exit 3
fi

if [ -e "$tmpdir" ]; then
  if [ ! -d "$tmpdir" ]; then
    printf 'E_TMPDIR_NOT_A_DIRECTORY %s\n' "$tmpdir" >&2
    exit 3
  fi
  tmpdir_owner="$(stat -c '%u' "$tmpdir" 2>/dev/null || true)"
  if [ -n "$tmpdir_owner" ] && [ "$tmpdir_owner" -ne "$current_uid" ]; then
    printf 'E_TMPDIR_OWNERSHIP_MISMATCH %s owner %s != current uid %s\n' "$tmpdir" "$tmpdir_owner" "$current_uid" >&2
    exit 3
  fi
fi

readonly systemd_user_dir="$root/.config/systemd/user"
readonly fragment="$root/.config/orq37-umask-hardening.sh"

assert_path_safe() {
  local p="$1"
  local canon
  canon="$(realpath -m "$p" 2>/dev/null || echo "$p")"
  if [[ "$canon" != "$canonical_root"* ]]; then
    printf 'E_PATH_TRAVERSAL %s outside %s\n' "$p" "$canonical_root" >&2
    exit 3
  fi
  if [ -L "$p" ]; then
    printf 'E_SYMLINK_REFUSED %s\n' "$p" >&2
    exit 3
  fi
}

emit() { printf '%s\n' "$*"; }

render_dropin() {
  cat <<EOF
[Service]
# ORQ-37 Wave B structural: files created by this unit must not be
# world-readable. 0077 clears group and other bits at creation time, which is
# what keeps a fresh credential artifact from starting life at 0664.
UMask=0077
# Keep scratch state out of the shared 1777 /tmp.
Environment=TMPDIR=$tmpdir
Environment=TMP=$tmpdir
EOF
}

render_fragment() {
  cat <<EOF
$MANAGED_BEGIN
# Source this from an interactive shell to inherit the hardened defaults:
#   [ -f "\$HOME/.config/orq37-umask-hardening.sh" ] && . "\$HOME/.config/orq37-umask-hardening.sh"
# It is deliberately NOT auto-installed into .bashrc: enabling it for a live
# session is a separate, explicit decision (host cutover).
umask 0077
export TMPDIR="$tmpdir"
export TMP="\$TMPDIR"
$MANAGED_END
EOF
}

plan_paths() {
  local unit
  for unit in "${units[@]}"; do
    emit "$systemd_user_dir/$unit.d/$DROPIN_NAME"
  done
  emit "$fragment"
  emit "$tmpdir"
}

do_apply() {
  local unit dir dropin_path
  assert_path_safe "$root/.config"
  assert_path_safe "$systemd_user_dir"
  install -d -m 700 "$tmpdir"
  chmod 700 "$tmpdir"
  install -d -m 700 "$root/.config"
  chmod 700 "$root/.config"
  install -d -m 700 "$systemd_user_dir"
  chmod 700 "$systemd_user_dir"

  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    dropin_path="$dir/$DROPIN_NAME"
    assert_path_safe "$dir"
    assert_path_safe "$dropin_path"
    assert_path_safe "$dropin_path.tmp"

    install -d -m 700 "$dir"
    chmod 700 "$dir"
    render_dropin >"$dropin_path.tmp"
    chmod 600 "$dropin_path.tmp"
    mv -f "$dropin_path.tmp" "$dropin_path"
    emit "APPLIED drop-in $dropin_path"
  done

  assert_path_safe "$fragment"
  assert_path_safe "$fragment.tmp"
  render_fragment >"$fragment.tmp"
  chmod 600 "$fragment.tmp"
  mv -f "$fragment.tmp" "$fragment"
  emit "APPLIED fragment $fragment"
  emit "NOTE daemon reload and unit restart are a SEPARATE authorized step; this script does not touch a running unit."
}

do_rollback() {
  local unit dir dropin_path
  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    dropin_path="$dir/$DROPIN_NAME"
    if [ -L "$dropin_path" ]; then
      printf 'E_SYMLINK_REFUSED %s\n' "$dropin_path" >&2
      exit 3
    fi
    if [ -f "$dropin_path" ]; then
      rm -f "$dropin_path"
      emit "ROLLED BACK drop-in $dropin_path"
      rmdir "$dir" 2>/dev/null && emit "REMOVED empty $dir" || true
    else
      emit "ABSENT drop-in $dropin_path"
    fi
  done

  if [ -L "$fragment" ]; then
    printf 'E_SYMLINK_REFUSED %s\n' "$fragment" >&2
    exit 3
  fi

  if [ -f "$fragment" ] && grep -qF "$MANAGED_BEGIN" "$fragment"; then
    rm -f "$fragment"
    emit "ROLLED BACK fragment $fragment"
  else
    emit "ABSENT or unmanaged fragment $fragment (left untouched)"
  fi
  emit "NOTE the private TMPDIR is left in place on purpose: it may hold task state, and removing data is not this script's job."
}

case "$mode" in
  dryrun)
    emit "DRY RUN — nothing written. Paths that --apply would create:"
    plan_paths
    emit "Drop-in content that would be written:"
    render_dropin | sed 's/^/  | /'
    ;;
  apply) do_apply ;;
  rollback) do_rollback ;;
esac

