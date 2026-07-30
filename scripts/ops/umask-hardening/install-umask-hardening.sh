#!/usr/bin/env bash
# Install a user-scoped UMask/TMPDIR drop-in without touching services.
set -euo pipefail

readonly MANAGED_BEGIN='# >>> orq37 umask hardening (managed) >>>'
readonly MANAGED_END='# <<< orq37 umask hardening (managed) <<<'
readonly DROPIN_NAME='10-orq37-umask-hardening.conf'

usage() {
  printf '%s\n' 'Usage: install-umask-hardening.sh [--apply|--rollback] [--root DIR] [--tmpdir DIR] [--unit NAME]...'
}

die_usage() { printf 'E_USAGE %s\n' "$1" >&2; usage >&2; exit 2; }
die_refused() { printf '%s\n' "$1" >&2; exit 3; }
need_value() { [ "$#" -ge 2 ] && [ -n "$2" ] && [[ "$2" != --* ]] || die_usage "$1 requires a value"; }

mode=dryrun
mode_seen=0
root="${HOME:-}"
tmpdir=
units=()
while [ "$#" -gt 0 ]; do
  case "$1" in
    --apply|--rollback)
      [ "$mode_seen" -eq 0 ] || die_usage 'choose exactly one of --apply and --rollback'
      mode_seen=1
      [ "$1" = --apply ] && mode=apply || mode=rollback
      shift
      ;;
    --root|--tmpdir|--unit)
      need_value "$@"
      case "$1" in
        --root) root="$2" ;;
        --tmpdir) tmpdir="$2" ;;
        --unit) units+=("$2") ;;
      esac
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) die_usage "unknown argument: $1" ;;
  esac
done

[ -n "$root" ] || die_usage '--root is empty'
[ -e "$root" ] || die_refused "E_ROOT_MISSING $root"
[ ! -L "$root" ] || die_refused "E_ROOT_SYMLINK $root"
[ -d "$root" ] || die_refused "E_ROOT_NOT_DIRECTORY $root"
canonical_root="$(realpath -e -- "$root")" || die_refused "E_ROOT_INVALID $root"
[ "$canonical_root" != / ] || die_refused 'E_ROOT_REFUSED /'
caller_uid="$(id -u)"
[ "$(stat -c %u -- "$canonical_root")" = "$caller_uid" ] || die_refused "E_ROOT_NOT_OWNED $canonical_root"

[ "${#units[@]}" -gt 0 ] || units=('multica-daemon-orq2-credential.service')
deduped=()
for unit in "${units[@]}"; do
  [[ "$unit" =~ ^[A-Za-z0-9][A-Za-z0-9_.@-]*\.service$ ]] || die_refused "E_UNIT_INVALID $unit"
  seen=0
  for prior in "${deduped[@]}"; do [ "$prior" = "$unit" ] && seen=1; done
  [ "$seen" -eq 1 ] || deduped+=("$unit")
done
units=("${deduped[@]}")

[ -n "$tmpdir" ] || tmpdir="$canonical_root/.private-tmp"
case "$tmpdir" in /*) ;; *) die_refused "E_TMPDIR_NOT_ABSOLUTE $tmpdir" ;; esac
canonical_tmpdir="$(realpath -m -- "$tmpdir")"
case "$canonical_tmpdir" in "$canonical_root"/*) ;; *) die_refused "E_TMPDIR_OUTSIDE_ROOT $canonical_tmpdir" ;; esac
[[ "$canonical_tmpdir" =~ ^/[A-Za-z0-9_./-]+$ ]] || die_refused "E_TMPDIR_UNSAFE_CHARACTERS $canonical_tmpdir"
tmpdir="$canonical_tmpdir"

assert_existing_component_chain() {
  local target="$1" rel current component
  case "$target" in "$canonical_root"|"$canonical_root"/*) ;; *) die_refused "E_PATH_OUTSIDE_ROOT $target" ;; esac
  rel="${target#"$canonical_root"}"
  current="$canonical_root"
  IFS=/ read -r -a components <<<"${rel#/}"
  for component in "${components[@]}"; do
    [ -n "$component" ] || continue
    current="$current/$component"
    [ -e "$current" ] || [ -L "$current" ] || break
    [ ! -L "$current" ] || die_refused "E_COMPONENT_SYMLINK $current"
    [ -d "$current" ] || die_refused "E_COMPONENT_NOT_DIRECTORY $current"
    [ "$(stat -c %u -- "$current")" = "$caller_uid" ] || die_refused "E_COMPONENT_NOT_OWNED $current"
  done
}

assert_target_type() {
  local target="$1"
  [ ! -L "$target" ] || die_refused "E_TARGET_SYMLINK $target"
  if [ -e "$target" ]; then
    [ -f "$target" ] || die_refused "E_TARGET_NOT_REGULAR $target"
    [ "$(stat -c %u -- "$target")" = "$caller_uid" ] || die_refused "E_TARGET_NOT_OWNED $target"
  fi
}

systemd_user_dir="$canonical_root/.config/systemd/user"
fragment="$canonical_root/.config/orq37-umask-hardening.sh"
temps=()
cleanup() { local p; for p in "${temps[@]}"; do rm -f -- "$p"; done; }
trap cleanup EXIT HUP INT TERM

render_dropin() {
  printf '%s\n' '[Service]' \
    '# ORQ-37 managed user hardening.' \
    'UMask=0077' \
    "Environment=TMPDIR=$tmpdir" \
    "Environment=TMP=$tmpdir"
}

render_fragment() {
  # TMPDIR must remain literal in the generated fragment.
  # shellcheck disable=SC2016
  printf '%s\n' "$MANAGED_BEGIN" \
    '# Opt-in only; this installer never edits shell startup files.' \
    'umask 0077' \
    "export TMPDIR=$(printf '%q' "$tmpdir")" \
    'export TMP="$TMPDIR"' "$MANAGED_END"
}

expected_matches() {
  local target="$1" renderer="$2" expected
  expected="$(mktemp "$(dirname "$target")/.orq37.expected.XXXXXX")"
  temps+=("$expected")
  chmod 600 "$expected"
  "$renderer" >"$expected"
  cmp -s -- "$expected" "$target"
}

atomic_install() {
  local target="$1" renderer="$2" tmp
  assert_target_type "$target"
  if [ -e "$target" ] && ! expected_matches "$target" "$renderer"; then
    die_refused "E_UNMANAGED_TARGET $target"
  fi
  tmp="$(mktemp "$(dirname "$target")/.orq37.install.XXXXXX")"
  temps+=("$tmp")
  chmod 600 "$tmp"
  "$renderer" >"$tmp"
  mv -- "$tmp" "$target"
}

verify_dropin() {
  local dropin="$1" verify_unit
  verify_unit="$(mktemp "$(dirname "$dropin")/.orq37-verify-XXXXXX.service")"
  temps+=("$verify_unit")
  chmod 600 "$verify_unit"
  { printf '%s\n' '[Unit]' 'Description=ORQ-37 verification' '[Service]' 'Type=oneshot' 'ExecStart=/bin/true'; render_dropin | sed '1d'; } >"$verify_unit"
  systemd-analyze verify "$verify_unit" >/dev/null 2>&1 || die_refused "E_SYSTEMD_VERIFY $dropin"
}

ensure_dir() {
  local dir="$1"
  assert_existing_component_chain "$dir"
  install -d -m 700 -- "$dir"
  [ ! -L "$dir" ] && [ -d "$dir" ] || die_refused "E_DIRECTORY_UNSAFE $dir"
  [ "$(stat -c %u -- "$dir")" = "$caller_uid" ] || die_refused "E_DIRECTORY_NOT_OWNED $dir"
  chmod 700 -- "$dir"
}

do_apply() {
  local unit dir dropin
  ensure_dir "$tmpdir"
  ensure_dir "$canonical_root/.config"
  ensure_dir "$canonical_root/.config/systemd"
  ensure_dir "$systemd_user_dir"
  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    ensure_dir "$dir"
    dropin="$dir/$DROPIN_NAME"
    verify_dropin "$dropin"
    atomic_install "$dropin" render_dropin
    printf 'APPLIED drop-in %s\n' "$dropin"
  done
  atomic_install "$fragment" render_fragment
  printf 'APPLIED fragment %s\n' "$fragment"
  printf '%s\n' 'NOTE daemon reload and restart are separate authorized steps; neither was invoked.'
}

remove_if_managed() {
  local target="$1" renderer="$2"
  assert_existing_component_chain "$(dirname "$target")"
  assert_target_type "$target"
  if [ ! -e "$target" ]; then printf 'ABSENT %s\n' "$target"; return; fi
  expected_matches "$target" "$renderer" || die_refused "E_UNMANAGED_TARGET $target"
  rm -- "$target"
  printf 'ROLLED BACK %s\n' "$target"
}

do_rollback() {
  local unit dir dropin
  for unit in "${units[@]}"; do
    dir="$systemd_user_dir/$unit.d"
    dropin="$dir/$DROPIN_NAME"
    remove_if_managed "$dropin" render_dropin
    rmdir -- "$dir" 2>/dev/null || true
  done
  remove_if_managed "$fragment" render_fragment
  printf '%s\n' 'NOTE the private TMPDIR is left in place on purpose; it may hold task state.'
}

case "$mode" in
  dryrun)
    printf '%s\n' 'DRY RUN — nothing written.'
    for unit in "${units[@]}"; do printf '%s\n' "$systemd_user_dir/$unit.d/$DROPIN_NAME"; done
    printf '%s\n' "$fragment" "$tmpdir" 'Drop-in content:'
    render_dropin | sed 's/^/  | /'
    ;;
  apply) do_apply ;;
  rollback) do_rollback ;;
esac
