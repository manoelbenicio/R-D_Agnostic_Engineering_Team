#!/usr/bin/env bash
# ORQ-37 gates for install-umask-hardening.sh.
#
# Every assertion runs against a sandbox root in a private temp directory, so
# the live home directory and the running unit are never touched. No credential
# file is read, created or inspected.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
installer="$here/install-umask-hardening.sh"
unit="orq37-test.service"
fail=0

sandbox="$(mktemp -d "${TMPDIR:-/tmp}/orq37-gate.XXXXXX")"
chmod 700 "$sandbox"
cleanup() { rm -rf "$sandbox"; }
trap cleanup EXIT

check() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    printf 'PASS %s\n' "$label"
  else
    printf 'FAIL %s\n' "$label"
    fail=$((fail + 1))
  fi
}

refute() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    printf 'FAIL %s (command unexpectedly succeeded)\n' "$label"
    fail=$((fail + 1))
  else
    printf 'PASS %s\n' "$label"
  fi
}

dropin="$sandbox/.config/systemd/user/$unit.d/10-orq37-umask-hardening.conf"
fragment="$sandbox/.config/orq37-umask-hardening.sh"

# 1. Dry run is the default and must not write anything.
"$installer" --root "$sandbox" --unit "$unit" >"$sandbox/dryrun.out"
check "dry run writes no drop-in" [ ! -e "$dropin" ]
check "dry run writes no fragment" [ ! -e "$fragment" ]
check "dry run reports the planned drop-in path" grep -qF "$dropin" "$sandbox/dryrun.out"
check "dry run shows UMask=0077" grep -qE '^\s+\| UMask=0077$' "$sandbox/dryrun.out"

# 2. Apply writes both artifacts with private modes and the right content.
"$installer" --apply --root "$sandbox" --unit "$unit" >"$sandbox/apply.out"
check "apply creates the drop-in" [ -f "$dropin" ]
check "drop-in is 0600" [ "$(stat -c '%a' "$dropin")" = "600" ]
check "drop-in directory is 0700" [ "$(stat -c '%a' "$(dirname "$dropin")")" = "700" ]
check "drop-in sets UMask=0077" grep -qx 'UMask=0077' "$dropin"
check "drop-in points TMPDIR at the private dir" grep -qx "Environment=TMPDIR=$sandbox/.private-tmp" "$dropin"
check "private TMPDIR exists and is 0700" [ "$(stat -c '%a' "$sandbox/.private-tmp")" = "700" ]
check "fragment is 0600" [ "$(stat -c '%a' "$fragment")" = "600" ]
check "fragment sets umask 0077" grep -qx 'umask 0077' "$fragment"
check "apply refuses to claim it restarted anything" grep -qF "SEPARATE authorized step" "$sandbox/apply.out"

# 3. Apply is idempotent: a second run leaves an identical drop-in.
before="$(sha256sum "$dropin" | cut -d' ' -f1)"
"$installer" --apply --root "$sandbox" --unit "$unit" >/dev/null
after="$(sha256sum "$dropin" | cut -d' ' -f1)"
check "second apply is byte-identical" [ "$before" = "$after" ]

# 4. The hardened umask actually produces 0600 files (behaviour, not just text).
probe="$sandbox/umask-probe"
( umask 0077; : >"$probe" )
check "umask 0077 yields a 0600 file" [ "$(stat -c '%a' "$probe")" = "600" ]

# 5. Rollback removes exactly what was created and keeps the data directory.
"$installer" --rollback --root "$sandbox" --unit "$unit" >"$sandbox/rollback.out"
check "rollback removes the drop-in" [ ! -e "$dropin" ]
check "rollback removes the fragment" [ ! -e "$fragment" ]
check "rollback preserves the private TMPDIR" [ -d "$sandbox/.private-tmp" ]
check "rollback states why the TMPDIR stays" grep -qF "left in place on purpose" "$sandbox/rollback.out"

# 6. Rollback on a clean tree is safe and reports absence.
"$installer" --rollback --root "$sandbox" --unit "$unit" >"$sandbox/rollback2.out"
check "second rollback is safe" grep -qF "ABSENT" "$sandbox/rollback2.out"

# 7. Refusals: system roots and a missing root must fail closed.
refute "refuses --root /etc" "$installer" --apply --root /etc --unit "$unit"
refute "refuses --root /" "$installer" --apply --root / --unit "$unit"
refute "refuses a nonexistent root" "$installer" --apply --root "$sandbox/does-not-exist" --unit "$unit"
refute "refuses an unknown flag" "$installer" --bogus

# 8. The installer must never mention a global shell file.
refute "never references /etc/bashrc as a target" grep -n 'install.*\/etc\/bashrc\|>>\s*\/etc\/bashrc' "$installer"

if [ "$fail" -ne 0 ]; then
  printf 'GATES FAILED: %d\n' "$fail"
  exit 1
fi
printf 'ALL GATES PASSED\n'
