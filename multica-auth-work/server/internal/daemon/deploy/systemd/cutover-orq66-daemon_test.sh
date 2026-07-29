#!/usr/bin/env bash
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
cutover="$script_dir/cutover-orq66-daemon.sh"
unit="$script_dir/multica-daemon-orq2-credential.service"

tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT

fail() {
  printf 'cutover-orq66-test: %s\n' "$*" >&2
  exit 1
}

assert_eq() {
  local got=$1 want=$2 label=$3
  [[ $got == "$want" ]] || fail "$label: got $got, want $want"
}

sha() {
  sha256sum -- "$1" | awk '{print $1}'
}

assert_unit_allowlist() {
  local name=$1 value=$2
  local line="Environment=MULTICA_CREDENTIAL_SLOT_ALLOWLIST_${name}=${value}"
  assert_eq "$(grep -Fxc -- "$line" "$unit")" "1" "$name allowlist exact-line count"
  assert_eq "$(grep -Ec -- "^Environment=MULTICA_CREDENTIAL_SLOT_ALLOWLIST_${name}=" "$unit")" "1" "$name allowlist total count"
}

assert_unit_allowlist ANTIGRAVITY 162,163,168,169
assert_unit_allowlist KIRO 139,140,143,149
assert_unit_allowlist CODEX 152,170

candidate="$tmp/candidate"
rollback="$tmp/rollback-88ca4f39"
target_dir="$tmp/install"
target="$target_dir/multica"
log="$tmp/events.log"
mkdir -m 700 -- "$target_dir"
printf '#!/bin/sh\nprintf candidate\n' >"$candidate"
printf '#!/bin/sh\nprintf rollback-88ca4f39\n' >"$rollback"
printf '#!/bin/sh\nprintf pre-cutover\n' >"$target"
chmod 755 "$candidate" "$rollback" "$target"
candidate_sha=$(sha "$candidate")
rollback_sha=$(sha "$rollback")

fake_systemctl="$tmp/fake-systemctl"
cat >"$fake_systemctl" <<'FAKE_SYSTEMCTL'
#!/usr/bin/env bash
set -u
actual=$(sha256sum -- "$ORQ66_TEST_TARGET" | awk '{print $1}')
printf 'restart %s %s\n' "$actual" "$*" >>"$ORQ66_TEST_LOG"
if [[ ${ORQ66_TEST_FAIL_CANDIDATE_RESTART:-0} == 1 && $actual == "$ORQ66_TEST_CANDIDATE_SHA" ]]; then
  exit 42
fi
FAKE_SYSTEMCTL
chmod 755 "$fake_systemctl"

fake_health="$tmp/fake-health"
cat >"$fake_health" <<'FAKE_HEALTH'
#!/usr/bin/env bash
set -u
actual=$(sha256sum -- "$ORQ66_TEST_TARGET" | awk '{print $1}')
printf 'health %s %s\n' "$actual" "$*" >>"$ORQ66_TEST_LOG"
[[ ${ORQ66_TEST_FAIL_HEALTH:-0} != 1 ]]
FAKE_HEALTH
chmod 755 "$fake_health"

run_cutover() {
  ORQ66_SYSTEMCTL_BIN="$fake_systemctl" \
  ORQ66_TEST_TARGET="$target" \
  ORQ66_TEST_LOG="$log" \
  ORQ66_TEST_CANDIDATE_SHA="$candidate_sha" \
    "$cutover" "$candidate" "$candidate_sha" "$rollback" "$rollback_sha" \
    88ca4f39 "$target" multica-daemon-orq2-credential.service "$fake_health" ready
}

# Successful cutover publishes and verifies the candidate before exactly one restart.
: >"$log"
run_cutover
assert_eq "$(sha "$target")" "$candidate_sha" "successful target hash"
assert_eq "$(stat -c '%a' -- "$target")" "755" "successful target mode"
assert_eq "$(grep -c '^restart ' "$log")" "1" "successful restart count"
assert_eq "$(sed -n '1p' "$log")" "restart $candidate_sha --user restart multica-daemon-orq2-credential.service" "restart after candidate install"
assert_eq "$(sed -n '2p' "$log")" "health $candidate_sha ready" "health after candidate restart"

# A restart failure is post-install: the verified 88ca artifact is restored
# atomically, checked at the target, and restarted once for rollback recovery.
printf '#!/bin/sh\nprintf pre-cutover\n' >"$target"
chmod 755 "$target"
: >"$log"
if ORQ66_TEST_FAIL_CANDIDATE_RESTART=1 run_cutover; then
  fail "candidate restart failure unexpectedly succeeded"
fi
assert_eq "$(sha "$target")" "$rollback_sha" "restart-failure rollback hash"
assert_eq "$(stat -c '%a' -- "$target")" "755" "restart-failure rollback mode"
assert_eq "$(grep -c '^restart ' "$log")" "2" "restart-failure restart count"
assert_eq "$(sed -n '1p' "$log")" "restart $candidate_sha --user restart multica-daemon-orq2-credential.service" "failed candidate restart ordering"
assert_eq "$(sed -n '2p' "$log")" "restart $rollback_sha --user restart multica-daemon-orq2-credential.service" "rollback restart ordering"

# A readiness failure after a successful candidate restart follows the same
# verified rollback path and leaves the known 88ca bytes at the final target.
printf '#!/bin/sh\nprintf pre-cutover\n' >"$target"
chmod 755 "$target"
: >"$log"
if ORQ66_TEST_FAIL_HEALTH=1 run_cutover; then
  fail "candidate health failure unexpectedly succeeded"
fi
assert_eq "$(sha "$target")" "$rollback_sha" "health-failure rollback hash"
assert_eq "$(grep -c '^restart ' "$log")" "2" "health-failure restart count"
assert_eq "$(sed -n '2p' "$log")" "health $candidate_sha ready" "candidate health ordering"
assert_eq "$(sed -n '3p' "$log")" "restart $rollback_sha --user restart multica-daemon-orq2-credential.service" "health rollback ordering"

# Exact hashes are preconditions. A mismatch must not publish or restart.
printf '#!/bin/sh\nprintf untouched\n' >"$target"
chmod 755 "$target"
untouched_sha=$(sha "$target")
: >"$log"
bad_sha=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
if ORQ66_SYSTEMCTL_BIN="$fake_systemctl" \
  ORQ66_TEST_TARGET="$target" ORQ66_TEST_LOG="$log" \
  ORQ66_TEST_CANDIDATE_SHA="$candidate_sha" \
  "$cutover" "$candidate" "$bad_sha" "$rollback" "$rollback_sha" \
  88ca4f39 "$target" multica-daemon-orq2-credential.service "$fake_health" ready; then
  fail "candidate hash mismatch unexpectedly succeeded"
fi
assert_eq "$(sha "$target")" "$untouched_sha" "pre-install mismatch target hash"
assert_eq "$(grep -c '^restart ' "$log" || true)" "0" "pre-install mismatch restart count"

printf 'cutover-orq66-test: PASS\n'
