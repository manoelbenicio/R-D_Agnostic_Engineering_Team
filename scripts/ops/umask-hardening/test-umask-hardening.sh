#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
installer="$here/install-umask-hardening.sh"
unit=orq37-test.service
pass=0 fail=0
sandbox="$(mktemp -d "${TMPDIR:-/tmp}/orq37-gate.XXXXXX")"
chmod 700 "$sandbox"
cleanup() { rm -rf -- "$sandbox"; }
trap cleanup EXIT

ok() { printf 'PASS %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf 'FAIL %s\n' "$1"; fail=$((fail + 1)); }
check() { local label="$1"; shift; if "$@" >/dev/null 2>&1; then ok "$label"; else bad "$label"; fi; }
refute() { local label="$1"; shift; if "$@" >/dev/null 2>&1; then bad "$label"; else ok "$label"; fi; }

root="$sandbox/root"
mkdir -m 700 "$root"
dropin="$root/.config/systemd/user/$unit.d/10-orq37-umask-hardening.conf"
fragment="$root/.config/orq37-umask-hardening.sh"

"$installer" --root "$root" --unit "$unit" --unit "$unit" >"$sandbox/dry"
check 'dry run writes nothing' test ! -e "$root/.config"
check 'dry run reports one deduplicated unit' test "$(grep -cF "$dropin" "$sandbox/dry")" = 1
check 'dry run renders UMask' grep -q '^  | UMask=0077$' "$sandbox/dry"

"$installer" --apply --root "$root" --unit "$unit" >"$sandbox/apply"
check 'apply creates drop-in' test -f "$dropin"
check 'drop-in mode is 0600' test "$(stat -c %a "$dropin")" = 600
check 'drop-in directory mode is 0700' test "$(stat -c %a "$(dirname "$dropin")")" = 700
check 'fragment mode is 0600' test "$(stat -c %a "$fragment")" = 600
check 'TMPDIR mode is 0700' test "$(stat -c %a "$root/.private-tmp")" = 700
check 'drop-in has UMask' grep -qx UMask=0077 "$dropin"
check 'drop-in preserves exact TMPDIR path' grep -qx "Environment=TMPDIR=$root/.private-tmp" "$dropin"
check 'apply states no service action' grep -q 'neither was invoked' "$sandbox/apply"

before="$(sha256sum "$dropin" "$fragment")"
"$installer" --apply --root "$root" --unit "$unit" >/dev/null
check 'second apply is byte-identical' test "$before" = "$(sha256sum "$dropin" "$fragment")"
# shellcheck disable=SC2016
check 'no predictable tmp remains' sh -c '! find "$1" \( -name "*.tmp" -o -name ".orq37.*" \) | grep -q .' sh "$root"
probe="$root/probe"; (umask 0077; : >"$probe")
check 'umask effect is 0600' test "$(stat -c %a "$probe")" = 600

cp "$dropin" "$sandbox/managed-dropin"
printf '%s\n' unmanaged >"$dropin"
refute 'apply refuses unmanaged drop-in' "$installer" --apply --root "$root" --unit "$unit"
refute 'rollback refuses unmanaged drop-in' "$installer" --rollback --root "$root" --unit "$unit"
check 'unmanaged drop-in remains intact' grep -qx unmanaged "$dropin"
mv "$sandbox/managed-dropin" "$dropin"
printf '%s\n' unmanaged >"$fragment"
refute 'apply refuses unmanaged fragment' "$installer" --apply --root "$root" --unit "$unit"
refute 'rollback refuses unmanaged fragment' "$installer" --rollback --root "$root" --unit "$unit"
check 'unmanaged fragment remains intact' grep -qx unmanaged "$fragment"
rm "$fragment"
"$installer" --apply --root "$root" --unit "$unit" >/dev/null

"$installer" --rollback --root "$root" --unit "$unit" >"$sandbox/rollback"
check 'rollback removes exact managed drop-in' test ! -e "$dropin"
check 'rollback removes exact managed fragment' test ! -e "$fragment"
check 'rollback preserves TMPDIR' test -d "$root/.private-tmp"
"$installer" --rollback --root "$root" --unit "$unit" >"$sandbox/rollback2"
check 'second rollback reports absence' grep -q ABSENT "$sandbox/rollback2"

refute 'rejects missing root value' "$installer" --root
refute 'rejects missing tmpdir value' "$installer" --tmpdir
refute 'rejects missing unit value' "$installer" --unit
refute 'rejects apply rollback conflict' "$installer" --apply --rollback --root "$root"
refute 'rejects duplicate mode' "$installer" --apply --apply --root "$root"
refute 'rejects unknown option' "$installer" --bogus
refute 'rejects non-service unit' "$installer" --root "$root" --unit timer.timer
refute 'rejects bare unit' "$installer" --root "$root" --unit daemon
refute 'rejects traversal unit' "$installer" --root "$root" --unit ../evil.service
refute 'rejects slash unit' "$installer" --root "$root" --unit evil/x.service
refute 'rejects newline unit' "$installer" --root "$root" --unit $'evil\n.service'
refute 'rejects system root' "$installer" --root /
refute 'rejects nonexistent root' "$installer" --root "$sandbox/missing"
ln -s "$root" "$sandbox/root-link"
refute 'rejects symlink root' "$installer" --root "$sandbox/root-link"

sibling="$sandbox/root-evil"; mkdir -m 700 "$sibling"
refute 'rejects prefix sibling TMPDIR' "$installer" --apply --root "$root" --tmpdir "$sibling/tmp" --unit "$unit"
refute 'rejects unsafe TMPDIR characters' "$installer" --apply --root "$root" --tmpdir "$root/space dir" --unit "$unit"
ln -s /tmp "$root/link"
refute 'rejects symlink TMPDIR component' "$installer" --apply --root "$root" --tmpdir "$root/link/private" --unit "$unit"
rm "$root/link"
printf x >"$root/not-dir"
refute 'rejects non-directory TMPDIR component' "$installer" --apply --root "$root" --tmpdir "$root/not-dir/private" --unit "$unit"
rm "$root/not-dir"

mkdir -p "$root/.config/systemd/user/$unit.d"
ln -s /etc/passwd "$dropin"
refute 'rejects symlink drop-in' "$installer" --apply --root "$root" --unit "$unit"
rm "$dropin"
mkdir "$dropin"
refute 'rejects directory target' "$installer" --apply --root "$root" --unit "$unit"
rmdir "$dropin"
ln -s /etc/shadow "$fragment"
refute 'rejects symlink fragment' "$installer" --apply --root "$root" --unit "$unit"
rm "$fragment"

# Existing final targets must have exactly one link. Each gate also proves
# refusal leaves both names and their shared inode untouched.
"$installer" --apply --root "$root" --unit "$unit" >/dev/null
ln "$dropin" "$root/dropin-hardlink"
reject_preserve_dropin_hardlink() {
  ! "$installer" --apply --root "$root" --unit "$unit" >/dev/null 2>&1 &&
    [ "$dropin" -ef "$root/dropin-hardlink" ] &&
    [ "$(stat -c %h "$dropin")" = 2 ]
}
check 'rejects and preserves hardlinked drop-in' reject_preserve_dropin_hardlink
rm "$root/dropin-hardlink"

ln "$fragment" "$root/fragment-hardlink"
reject_preserve_fragment_hardlink() {
  ! "$installer" --apply --root "$root" --unit "$unit" >/dev/null 2>&1 &&
    [ "$fragment" -ef "$root/fragment-hardlink" ] &&
    [ "$(stat -c %h "$fragment")" = 2 ]
}
check 'rejects and preserves hardlinked fragment' reject_preserve_fragment_hardlink
rm "$root/fragment-hardlink"
"$installer" --rollback --root "$root" --unit "$unit" >/dev/null

# Managed provenance includes the rendered TMPDIR. Rollback must replay the
# identical custom value; omission/mismatch fails closed and preserves files.
custom_tmpdir="$root/custom-tmp"
"$installer" --apply --root "$root" --tmpdir "$custom_tmpdir" --unit "$unit" >/dev/null
refute 'mismatched TMPDIR rollback fails closed' "$installer" --rollback --root "$root" --unit "$unit"
# shellcheck disable=SC2016
check 'mismatched TMPDIR rollback preserves managed files' sh -c '[ -f "$1" ] && [ -f "$2" ]' sh "$dropin" "$fragment"
"$installer" --rollback --root "$root" --tmpdir "$custom_tmpdir" --unit "$unit" >/dev/null
# shellcheck disable=SC2016
check 'matched TMPDIR rollback removes managed files' sh -c '[ ! -e "$1" ] && [ ! -e "$2" ]' sh "$dropin" "$fragment"

mkdir -p "$sandbox/bin"
# shellcheck disable=SC2016
printf '%s\n' '#!/usr/bin/env bash' 'printf invoked >"$ORQ37_SYSTEMCTL_LOG"' >"$sandbox/bin/systemctl"
chmod +x "$sandbox/bin/systemctl"
ORQ37_SYSTEMCTL_LOG="$sandbox/systemctl.log" PATH="$sandbox/bin:$PATH" "$installer" --apply --root "$root" --unit "$unit" >/dev/null
check 'stubbed systemctl was never invoked' test ! -e "$sandbox/systemctl.log"
{
  printf '%s\n' '[Unit]' 'Description=ORQ-37 harness verification' '[Service]' 'Type=oneshot' 'ExecStart=/bin/true'
  sed '1d' "$dropin"
} >"$root/verify.service"
check 'systemd-analyze accepts rendered settings' systemd-analyze verify "$root/verify.service"
refute 'installer never targets global bashrc' grep -E '(^|[[:space:]])(>|>>|install).*\/etc\/bashrc' "$installer"
# shellcheck disable=SC2016
check 'all writes stay under fake root' sh -c '! find "$1" -mindepth 1 -maxdepth 1 ! -name root ! -name root-evil ! -name root-link ! -name bin ! -name dry ! -name apply ! -name rollback ! -name rollback2 | grep -q .' sh "$sandbox"

if [ "$fail" -ne 0 ]; then printf 'GATES FAILED: %d (PASS %d)\n' "$fail" "$pass"; exit 1; fi
printf 'ALL GATES PASSED (%d/%d; zero skips)\n' "$pass" "$pass"
