#!/usr/bin/env bash
set -Eeuo pipefail

readonly REQUIRED_ROLLBACK_REVISION="88ca4f39"
readonly INSTALL_MODE="755"

usage() {
  cat >&2 <<'USAGE'
usage: cutover-orq66-daemon.sh \
  CANDIDATE CANDIDATE_SHA256 ROLLBACK_88CA ROLLBACK_SHA256 88ca4f39 \
  TARGET SERVICE HEALTH_COMMAND [HEALTH_ARGUMENT ...]

The candidate and known rollback artifact are staged in TARGET's directory,
verified, and published with same-filesystem atomic rename. The success path
restarts SERVICE exactly once, after candidate publication. Any post-install
failure atomically restores the verified 88ca artifact and restarts SERVICE.
Set ORQ66_SYSTEMCTL_BIN to inject a hermetic systemctl replacement for tests.
USAGE
  exit 64
}

[[ $# -ge 8 ]] || usage
candidate=$1
candidate_sha=$2
rollback_artifact=$3
rollback_sha=$4
rollback_revision=$5
target=$6
service=$7
shift 7
health_command=("$@")
systemctl_bin=${ORQ66_SYSTEMCTL_BIN:-systemctl}

target_dir=$(dirname -- "$target")
target_base=$(basename -- "$target")
candidate_stage=""
rollback_stage=""
installed=0
complete=0

fail() {
  printf 'cutover-orq66: %s\n' "$*" >&2
  exit 1
}

valid_sha256() {
  [[ $1 =~ ^[0-9a-f]{64}$ ]]
}

file_sha256() {
  sha256sum -- "$1" | awk '{print $1}'
}

file_mode() {
  stat -c '%a' -- "$1"
}

verify_artifact() {
  local path=$1 expected_sha=$2 label=$3
  [[ -f $path && ! -L $path ]] || {
    printf 'cutover-orq66: %s is not a physical regular file: %s\n' "$label" "$path" >&2
    return 1
  }
  [[ $(file_mode "$path") == "$INSTALL_MODE" ]] || {
    printf 'cutover-orq66: %s mode is not %s: %s\n' "$label" "$INSTALL_MODE" "$path" >&2
    return 1
  }
  [[ $(file_sha256 "$path") == "$expected_sha" ]] || {
    printf 'cutover-orq66: %s SHA-256 mismatch: %s\n' "$label" "$path" >&2
    return 1
  }
}

stage_artifact() {
  local source=$1 expected_sha=$2 label=$3
  local staged
  staged=$(mktemp -- "$target_dir/.${target_base}.${label}.XXXXXX") || return 1
  if ! install -m "$INSTALL_MODE" -- "$source" "$staged"; then
    rm -f -- "$staged"
    return 1
  fi
  if ! verify_artifact "$staged" "$expected_sha" "$label"; then
    rm -f -- "$staged"
    return 1
  fi
  printf '%s\n' "$staged"
}

cleanup_stages() {
  [[ -z $candidate_stage ]] || rm -f -- "$candidate_stage"
  [[ -z $rollback_stage ]] || rm -f -- "$rollback_stage"
}

rollback_after_failure() {
  local original_status=$1 rollback_status=0
  trap - EXIT
  set +e
  if [[ -n $rollback_stage && -f $rollback_stage ]]; then
    mv -fT -- "$rollback_stage" "$target" || rollback_status=1
    rollback_stage=""
    sync -f -- "$target_dir" || rollback_status=1
    verify_artifact "$target" "$rollback_sha" "rollback-$REQUIRED_ROLLBACK_REVISION" || rollback_status=1
    "$systemctl_bin" --user restart "$service" || rollback_status=1
  else
    printf 'cutover-orq66: verified rollback stage is unavailable\n' >&2
    rollback_status=1
  fi
  cleanup_stages
  if (( rollback_status != 0 )); then
    printf 'cutover-orq66: rollback failed after post-install error\n' >&2
    exit 70
  fi
  printf 'cutover-orq66: restored rollback revision %s after post-install failure\n' "$REQUIRED_ROLLBACK_REVISION" >&2
  exit "$original_status"
}

on_exit() {
  local status=$?
  if (( installed == 1 && complete == 0 )); then
    rollback_after_failure "$status"
  fi
  cleanup_stages
}
trap on_exit EXIT

[[ $rollback_revision == "$REQUIRED_ROLLBACK_REVISION" ]] || \
  fail "rollback revision must be $REQUIRED_ROLLBACK_REVISION"
valid_sha256 "$candidate_sha" || fail "candidate SHA-256 must be 64 lowercase hex characters"
valid_sha256 "$rollback_sha" || fail "rollback SHA-256 must be 64 lowercase hex characters"
[[ -d $target_dir && ! -L $target_dir ]] || fail "target directory must be a physical directory"
[[ ! -e $target || ( -f $target && ! -L $target ) ]] || fail "target must be absent or a physical regular file"
if [[ $systemctl_bin != */* ]]; then
  systemctl_bin=$(command -v -- "$systemctl_bin") || fail "systemctl command was not found"
fi
if [[ ${health_command[0]} != */* ]]; then
  health_command[0]=$(command -v -- "${health_command[0]}") || fail "health command was not found"
fi
[[ -x $systemctl_bin ]] || fail "systemctl command is not executable: $systemctl_bin"
[[ -x ${health_command[0]} ]] || fail "health command is not executable: ${health_command[0]}"
[[ -f $candidate && ! -L $candidate ]] || fail "candidate must be a physical regular file"
[[ -f $rollback_artifact && ! -L $rollback_artifact ]] || fail "rollback artifact must be a physical regular file"

rollback_stage=$(stage_artifact "$rollback_artifact" "$rollback_sha" "rollback-$REQUIRED_ROLLBACK_REVISION")
candidate_stage=$(stage_artifact "$candidate" "$candidate_sha" "candidate")

# Both paths are in target_dir, so this is a same-filesystem atomic rename.
mv -fT -- "$candidate_stage" "$target"
candidate_stage=""
installed=1
sync -f -- "$target_dir"
verify_artifact "$target" "$candidate_sha" "installed candidate"

# The normal cutover has exactly one restart, and it occurs only after the
# candidate's mode and full SHA-256 have been verified at the final path.
"$systemctl_bin" --user restart "$service"
"${health_command[@]}"

complete=1
rm -f -- "$rollback_stage"
rollback_stage=""
sync -f -- "$target_dir"
printf 'cutover-orq66: candidate installed and verified; rollback=%s\n' "$REQUIRED_ROLLBACK_REVISION"
