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

# Refuse absolute-root or system paths outright: this tool is user-scoped by
# contract and must never be pointed at /etc or /.
case "$root" in
  /|/etc|/etc/*|/usr|/usr/*|/var|/var/*) printf 'E_ROOT_REFUSED %s\n' "$root" >&2; exit 3 ;;
esac

readonly systemd_user_dir="$root/.config/systemd/user"
readonly fragment="$root/.config/orq37-umask-hardening.sh"

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
  local unit dir
  install -d -m 700 "$tmpdir"
  install -d -m 700 "$root/.config"
  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    install -d -m 700 "$dir"
    render_dropin >"$dir/$DROPIN_NAME.tmp"
    chmod 600 "$dir/$DROPIN_NAME.tmp"
    mv -f "$dir/$DROPIN_NAME.tmp" "$dir/$DROPIN_NAME"
    emit "APPLIED drop-in $dir/$DROPIN_NAME"
  done
  render_fragment >"$fragment.tmp"
  chmod 600 "$fragment.tmp"
  mv -f "$fragment.tmp" "$fragment"
  emit "APPLIED fragment $fragment"
  emit "NOTE daemon reload and unit restart are a SEPARATE authorized step; this script does not touch a running unit."
}

do_rollback() {
  local unit dir
  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    if [ -f "$dir/$DROPIN_NAME" ]; then
      rm -f "$dir/$DROPIN_NAME"
      emit "ROLLED BACK drop-in $dir/$DROPIN_NAME"
      rmdir "$dir" 2>/dev/null && emit "REMOVED empty $dir" || true
    else
      emit "ABSENT drop-in $dir/$DROPIN_NAME"
    fi
  done
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
