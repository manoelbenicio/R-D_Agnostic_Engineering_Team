#!/usr/bin/env bash
# One-command rollback artifact for credential-account-home restoration (ORQ-23)
# Validates prior daemon references, 3 required runtimes (Codex, Antigravity, Kiro),
# and performs content-free identity reporting without reading credential secrets.
# Implements atomic file replacement, parent directory fsync, private roll-forward
# backup creation, explicit roll-forward restoration, EXIT recovery trap, systemd
# stop/start verification, mode 0700 posture, and fail-closed security guards.

set -euo pipefail

# Configuration
DAEMON_BIN_DIR="/home/ec2-user/.local/lib/multica/bin"
DAEMON_BIN_ACTIVE="${DAEMON_BIN_DIR}/multica-auth-credential-home-v1"
DAEMON_BIN_PREVIOUS="${DAEMON_BIN_DIR}/multica-auth-credential-home-v1.previous"
DAEMON_SERVICE_FILE="/home/ec2-user/.config/systemd/user/multica-daemon-orq2-credential.service"
DAEMON_SERVICE_NAME="multica-daemon-orq2-credential.service"

ROLLFORWARD_BACKUP_DIR="${DAEMON_BIN_DIR}/rollforward-backups"
ROLLFORWARD_LATEST_POINTER="${DAEMON_BIN_DIR}/multica-auth-credential-home-v1.rollforward.latest"

CODEX_BIN="/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex"
AGY_BIN="/home/ec2-user/.local/bin/agy"
KIRO_BIN="/home/ec2-user/.local/bin/kiro-cli"
CRED_HOMES_ROOT="/home/ec2-user/.agent-cred-homes"

MODE="${1:---dry-run}"
DISPATCH_CUTOVER="${GTL_CUTOVER_DISPATCH:-0}"

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

die() {
  log "ERROR: $*" >&2
  exit 1
}

# 1. Count active product tasks
count_active_tasks() {
  local count=0
  count="$(pgrep -f "agy.*--log-file|codex.*--log-file|kiro.*--log-file" 2>/dev/null | wc -l || true)"
  printf '%d' "${count}"
}

# 2. Content-free identity calculation
get_sha256() {
  local target_file="$1"
  if [[ -f "${target_file}" && ! -L "${target_file}" ]]; then
    sha256sum "${target_file}" | awk '{print $1}'
  else
    echo "FILE_MISSING_OR_SYMLINK"
  fi
}

# 3. Content-free runtime verification
verify_runtime() {
  local name="$1"
  local bin_path="$2"
  local real_path
  real_path="$(readlink -f "${bin_path}" 2>/dev/null || echo "${bin_path}")"
  if [[ -x "${bin_path}" && -f "${real_path}" && -x "${real_path}" ]]; then
    local version_info
    version_info="$("${bin_path}" --version 2>/dev/null | head -n 1 || echo "executable present")"
    log "Runtime check [PASS]: ${name} -> path=${bin_path} (${version_info})"
    return 0
  else
    log "Runtime check [FAIL]: ${name} -> path=${bin_path} (not found or not executable)"
    return 1
  fi
}

# 4. Fail-closed security validation checks
validate_security_environment() {
  local current_uid expected_uid
  current_uid="$(id -u)"
  expected_uid="$(stat -c '%u' "${DAEMON_BIN_DIR}" 2>/dev/null || echo "${current_uid}")"

  if [[ "${current_uid}" != "${expected_uid}" ]]; then
    die "FAIL-CLOSED: UID mismatch (running as UID ${current_uid}, expected UID ${expected_uid})."
  fi

  if [[ -L "${DAEMON_BIN_DIR}" ]]; then
    die "FAIL-CLOSED: Daemon binary directory ${DAEMON_BIN_DIR} is a symbolic link."
  fi

  if [[ -L "${DAEMON_BIN_ACTIVE}" ]]; then
    die "FAIL-CLOSED: Active daemon binary ${DAEMON_BIN_ACTIVE} is a symbolic link."
  fi

  if [[ -L "${DAEMON_BIN_PREVIOUS}" ]]; then
    die "FAIL-CLOSED: Previous daemon binary ${DAEMON_BIN_PREVIOUS} is a symbolic link."
  fi

  if [[ -L "${DAEMON_SERVICE_FILE}" ]]; then
    die "FAIL-CLOSED: Systemd service file ${DAEMON_SERVICE_FILE} is a symbolic link."
  fi
}

# 5. Validate binary regular file, owner, non-empty, executable mode 0700 posture, and hash
validate_binary_file() {
  local label="$1"
  local bin_path="$2"

  if [[ ! -f "${bin_path}" ]]; then
    die "FAIL-CLOSED: ${label} file does not exist at ${bin_path}."
  fi

  if [[ -L "${bin_path}" ]]; then
    die "FAIL-CLOSED: ${label} file ${bin_path} is an unsafe symbolic link."
  fi

  if [[ ! -s "${bin_path}" ]]; then
    die "FAIL-CLOSED: ${label} file ${bin_path} is empty (0 bytes / corrupt)."
  fi

  local file_uid
  file_uid="$(stat -c '%u' "${bin_path}")"
  if [[ "${file_uid}" != "$(id -u)" ]]; then
    die "FAIL-CLOSED: ${label} file ${bin_path} owner UID mismatch (${file_uid} != $(id -u))."
  fi

  if [[ ! -x "${bin_path}" ]]; then
    die "FAIL-CLOSED: ${label} file ${bin_path} is not executable."
  fi

  # Check private operational posture (0700)
  local file_mode
  file_mode="$(stat -c '%a' "${bin_path}")"
  if [[ "${file_mode}" != "700" ]]; then
    die "FAIL-CLOSED: ${label} file ${bin_path} mode is ${file_mode}, expected 700 (private operational posture)."
  fi

  local hash
  hash="$(get_sha256 "${bin_path}")"
  if [[ "${hash}" == "FILE_MISSING_OR_SYMLINK" || -z "${hash}" ]]; then
    die "FAIL-CLOSED: Failed to compute SHA256 checksum for ${label} file ${bin_path}."
  fi
}

# 6. Fsync helper for file and directory
fsync_file() {
  local target_file="$1"
  python3 -c "import os, sys; fd = os.open(sys.argv[1], os.O_RDWR); os.fsync(fd); os.close(fd)" "${target_file}"
}

fsync_dir() {
  local target_dir="$1"
  python3 -c "import os, sys; fd = os.open(sys.argv[1], os.O_RDONLY); os.fsync(fd); os.close(fd)" "${target_dir}"
}

# 7. Atomic binary replacement with EXIT recovery trap, systemd stop/start verification, and mode 0700
atomic_install_binary() {
  local src_bin="$1"
  local dest_bin="$2"

  validate_binary_file "Source" "${src_bin}"

  local dest_dir
  dest_dir="$(dirname "${dest_bin}")"
  local temp_bin="${dest_bin}.tmp.${BASHPID}_$(date +%s%N)"

  # Install EXIT recovery trap
  cleanup_atomic_install() {
    local exit_code=$?
    rm -f "${temp_bin:-}" 2>/dev/null || true
    if (( exit_code != 0 )); then
      log "EXIT RECOVERY TRAP: Atomic installation failed with exit code ${exit_code}."
      if command -v systemctl >/dev/null 2>&1; then
        if ! systemctl --user is-active --quiet "${DAEMON_SERVICE_NAME}" 2>/dev/null; then
          log "EXIT RECOVERY TRAP: Attempting emergency restart of ${DAEMON_SERVICE_NAME}..."
          systemctl --user restart "${DAEMON_SERVICE_NAME}" 2>/dev/null || true
        fi
      fi
    fi
  }
  trap cleanup_atomic_install EXIT

  log "Staging binary ${src_bin} -> ${temp_bin}"
  cp -p "${src_bin}" "${temp_bin}"
  chmod 0700 "${temp_bin}"

  validate_binary_file "Staged" "${temp_bin}"

  local src_hash temp_hash
  src_hash="$(get_sha256 "${src_bin}")"
  temp_hash="$(get_sha256 "${temp_bin}")"
  if [[ "${src_hash}" != "${temp_hash}" ]]; then
    die "FAIL-CLOSED: Staged binary SHA256 (${temp_hash}) does not match source (${src_hash})."
  fi

  fsync_file "${temp_bin}"

  # Systemd stop with error propagation and stop verification
  if command -v systemctl >/dev/null 2>&1; then
    log "Stopping daemon service ${DAEMON_SERVICE_NAME} prior to atomic swap..."
    systemctl --user stop "${DAEMON_SERVICE_NAME}"

    local stop_ok=0
    for _ in $(seq 1 5); do
      if ! systemctl --user is-active --quiet "${DAEMON_SERVICE_NAME}" 2>/dev/null; then
        stop_ok=1
        break
      fi
      sleep 1
    done

    if (( stop_ok != 1 )); then
      die "FAIL-CLOSED: Service ${DAEMON_SERVICE_NAME} failed to stop before atomic rename."
    fi
    log "Service ${DAEMON_SERVICE_NAME} verified STOPPED."
  else
    log "systemctl not available; daemon stop check skipped."
  fi

  log "Executing atomic rename: ${temp_bin} -> ${dest_bin}"
  mv -f "${temp_bin}" "${dest_bin}"
  chmod 0700 "${dest_bin}"
  fsync_dir "${dest_dir}"

  # Systemd restart with error propagation and active state verification
  if command -v systemctl >/dev/null 2>&1; then
    log "Reloading systemd daemon and restarting service ${DAEMON_SERVICE_NAME}..."
    systemctl --user daemon-reload
    systemctl --user restart "${DAEMON_SERVICE_NAME}"

    if ! systemctl --user is-active --quiet "${DAEMON_SERVICE_NAME}" 2>/dev/null; then
      die "FAIL-CLOSED: Service ${DAEMON_SERVICE_NAME} is not active after restart."
    fi
    log "Service ${DAEMON_SERVICE_NAME} verified ACTIVE."
  else
    log "systemctl not available; daemon restart skipped."
  fi

  # Clear EXIT trap on successful completion
  trap - EXIT
}

# 8. Create private timestamped roll-forward backup before mutation
create_rollforward_backup() {
  validate_binary_file "Active Daemon" "${DAEMON_BIN_ACTIVE}"

  (umask 077; mkdir -p "${ROLLFORWARD_BACKUP_DIR}")
  chmod 0700 "${ROLLFORWARD_BACKUP_DIR}"

  if [[ -L "${ROLLFORWARD_BACKUP_DIR}" ]]; then
    die "FAIL-CLOSED: Roll-forward backup directory ${ROLLFORWARD_BACKUP_DIR} is an unsafe symlink."
  fi

  local timestamp backup_path
  timestamp="$(date -u +%Y%m%d_%H%M%S)"
  backup_path="${ROLLFORWARD_BACKUP_DIR}/multica-auth-credential-home-v1.rollforward.${timestamp}"

  log "Creating private roll-forward backup of active binary..."
  log "Backup destination: ${backup_path}"

  cp -p "${DAEMON_BIN_ACTIVE}" "${backup_path}"
  chmod 0700 "${backup_path}"

  validate_binary_file "Roll-forward Backup" "${backup_path}"
  fsync_file "${backup_path}"

  local active_hash backup_hash
  active_hash="$(get_sha256 "${DAEMON_BIN_ACTIVE}")"
  backup_hash="$(get_sha256 "${backup_path}")"
  if [[ "${active_hash}" != "${backup_hash}" ]]; then
    rm -f "${backup_path}"
    die "FAIL-CLOSED: Roll-forward backup SHA256 mismatch with active binary."
  fi

  # Atomically update pointer file
  local pointer_tmp="${ROLLFORWARD_LATEST_POINTER}.tmp.${BASHPID}"
  (umask 077; printf '%s\n' "${backup_path}" > "${pointer_tmp}")
  fsync_file "${pointer_tmp}"
  mv -f "${pointer_tmp}" "${ROLLFORWARD_LATEST_POINTER}"
  chmod 0600 "${ROLLFORWARD_LATEST_POINTER}"
  fsync_dir "${DAEMON_BIN_DIR}"

  log "Roll-forward backup created successfully: ${backup_path} (SHA256: ${backup_hash})"
}

# 9. Get latest valid roll-forward backup path
get_latest_rollforward_backup() {
  local backup_path=""

  if [[ -f "${ROLLFORWARD_LATEST_POINTER}" && ! -L "${ROLLFORWARD_LATEST_POINTER}" ]]; then
    backup_path="$(<"${ROLLFORWARD_LATEST_POINTER}")"
  fi

  if [[ -z "${backup_path}" || ! -f "${backup_path}" ]]; then
    if [[ -d "${ROLLFORWARD_BACKUP_DIR}" && ! -L "${ROLLFORWARD_BACKUP_DIR}" ]]; then
      backup_path="$(find "${ROLLFORWARD_BACKUP_DIR}" -maxdepth 1 -type f -name "multica-auth-credential-home-v1.rollforward.*" 2>/dev/null | sort -r | head -n 1 || true)"
    fi
  fi

  printf '%s' "${backup_path}"
}

run_dry_run() {
  local active_tasks
  active_tasks="$(count_active_tasks)"

  validate_security_environment

  log "=== ONE-COMMAND ROLLBACK DRY-RUN VERIFICATION (ORQ-23) ==="
  log "Mode: DRY-RUN (Content-Free Inspection)"
  log "Active Product Tasks Count: ${active_tasks}"
  log "GTL Cutover Dispatch Flag: ${DISPATCH_CUTOVER}"
  log "F3 Cost Gate Status: BLOCKED on ORQ-13/14 (not claimed)"

  log "--- Checking Daemon Binary References ---"
  log "Active Daemon Binary: ${DAEMON_BIN_ACTIVE} (SHA256: $(get_sha256 "${DAEMON_BIN_ACTIVE}"))"
  log "Prior Daemon Binary:  ${DAEMON_BIN_PREVIOUS} (SHA256: $(get_sha256 "${DAEMON_BIN_PREVIOUS}"))"

  validate_binary_file "Prior Daemon" "${DAEMON_BIN_PREVIOUS}"

  log "--- Checking Systemd Service Reference ---"
  log "Systemd Unit File: ${DAEMON_SERVICE_FILE} (SHA256: $(get_sha256 "${DAEMON_SERVICE_FILE}"))"
  if [[ ! -f "${DAEMON_SERVICE_FILE}" || -L "${DAEMON_SERVICE_FILE}" ]]; then
    die "Systemd service file ${DAEMON_SERVICE_FILE} does not exist or is a symlink!"
  fi

  log "--- Checking Roll-Forward Backup Status ---"
  local latest_backup
  latest_backup="$(get_latest_rollforward_backup)"
  if [[ -n "${latest_backup}" && -f "${latest_backup}" ]]; then
    log "Latest Roll-Forward Backup: ${latest_backup} (SHA256: $(get_sha256 "${latest_backup}"))"
  else
    log "Latest Roll-Forward Backup: None recorded yet (will be created automatically on live rollback)"
  fi

  log "--- Verifying Required Runtimes (Content-Free) ---"
  local runtime_pass=0
  verify_runtime "Codex" "${CODEX_BIN}" && ((runtime_pass++)) || true
  verify_runtime "Antigravity" "${AGY_BIN}" && ((runtime_pass++)) || true
  verify_runtime "Kiro" "${KIRO_BIN}" && ((runtime_pass++)) || true

  if (( runtime_pass < 3 )); then
    die "Runtime verification failed: only ${runtime_pass}/3 runtimes passed."
  fi

  log "--- Checking Credential Homes Root ---"
  if [[ -d "${CRED_HOMES_ROOT}" && ! -L "${CRED_HOMES_ROOT}" ]]; then
    log "Credential Homes Root: ${CRED_HOMES_ROOT} (Directory Present, Mode: $(stat -c '%a' "${CRED_HOMES_ROOT}"))"
  else
    log "Credential Homes Root: ${CRED_HOMES_ROOT} (Will be created on demand)"
  fi

  log "=== DRY-RUN VERIFICATION RESULT: PASS ==="
  if (( active_tasks > 0 )); then
    log "SAFEGUARD: Active product tasks (${active_tasks}) are currently running."
    log "Destructive live operations MUST NOT be executed until active tasks = 0 and GTL dispatches cutover."
  else
    log "Active product tasks count is 0. System is ready for GTL cutover dispatch when requested."
  fi

  return 0
}

run_live_rollback() {
  local active_tasks
  active_tasks="$(count_active_tasks)"

  validate_security_environment

  log "=== ONE-COMMAND ROLLBACK LIVE EXECUTION (ORQ-23) ==="
  log "Active Product Tasks Count: ${active_tasks}"
  log "GTL Cutover Dispatch Flag: ${DISPATCH_CUTOVER}"

  if (( active_tasks > 0 )); then
    die "LIVE ROLLBACK REFUSED: active product tasks (${active_tasks}) remain running. Live rollback requires active task count = 0."
  fi

  if [[ "${DISPATCH_CUTOVER}" != "1" && "${FORCE_ROLLBACK:-0}" != "1" ]]; then
    die "LIVE ROLLBACK REFUSED: GTL cutover has not been dispatched. Set GTL_CUTOVER_DISPATCH=1 or FORCE_ROLLBACK=1 to proceed."
  fi

  # 1. Create timestamped roll-forward backup of active binary before mutation
  create_rollforward_backup

  # 2. Perform atomic installation of prior daemon binary
  log "--- Executing Prior Daemon Binary Restoration ---"
  log "Restoring ${DAEMON_BIN_PREVIOUS} -> ${DAEMON_BIN_ACTIVE}"
  atomic_install_binary "${DAEMON_BIN_PREVIOUS}" "${DAEMON_BIN_ACTIVE}"

  log "--- Verifying Daemon Health Endpoint ---"
  local health_ok=0
  for _ in $(seq 1 10); do
    if curl -s http://127.0.0.1:18080/health | grep -q '"status":"ok"'; then
      health_ok=1
      break
    fi
    sleep 1
  done

  if (( health_ok == 1 )); then
    log "Daemon health check [PASS]: http://127.0.0.1:18080/health responded OK."
  else
    die "Daemon health check [FAIL]: http://127.0.0.1:18080/health did not report OK."
  fi

  log "--- Re-verifying Runtimes ---"
  verify_runtime "Codex" "${CODEX_BIN}" || die "Codex runtime verification failed after rollback."
  verify_runtime "Antigravity" "${AGY_BIN}" || die "Antigravity runtime verification failed after rollback."
  verify_runtime "Kiro" "${KIRO_BIN}" || die "Kiro runtime verification failed after rollback."

  log "=== LIVE ROLLBACK COMPLETE (RESTORED TO PRIOR DAEMON BINARY) ==="
  log "Restored Binary SHA256: $(get_sha256 "${DAEMON_BIN_ACTIVE}")"
  log "Roll-Forward Backup Available at: $(get_latest_rollforward_backup)"
  return 0
}

run_live_rollforward() {
  local active_tasks
  active_tasks="$(count_active_tasks)"

  validate_security_environment

  log "=== EXPLICIT ROLL-FORWARD RESTORATION EXECUTION (ORQ-23) ==="
  log "Active Product Tasks Count: ${active_tasks}"
  log "GTL Cutover Dispatch Flag: ${DISPATCH_CUTOVER}"

  if (( active_tasks > 0 )); then
    die "LIVE ROLL-FORWARD REFUSED: active product tasks (${active_tasks}) remain running. Requires active task count = 0."
  fi

  if [[ "${DISPATCH_CUTOVER}" != "1" && "${FORCE_ROLLFORWARD:-0}" != "1" && "${FORCE_ROLLBACK:-0}" != "1" ]]; then
    die "LIVE ROLL-FORWARD REFUSED: GTL cutover has not been dispatched. Set GTL_CUTOVER_DISPATCH=1 or FORCE_ROLLFORWARD=1 to proceed."
  fi

  local backup_path
  backup_path="$(get_latest_rollforward_backup)"

  if [[ -z "${backup_path}" ]]; then
    die "LIVE ROLL-FORWARD REFUSED: No roll-forward backup file recorded."
  fi

  log "Target Roll-Forward Backup: ${backup_path}"
  validate_binary_file "Roll-forward Backup" "${backup_path}"

  log "--- Executing Roll-Forward Binary Restoration ---"
  log "Restoring ${backup_path} -> ${DAEMON_BIN_ACTIVE}"
  atomic_install_binary "${backup_path}" "${DAEMON_BIN_ACTIVE}"

  log "--- Verifying Daemon Health Endpoint ---"
  local health_ok=0
  for _ in $(seq 1 10); do
    if curl -s http://127.0.0.1:18080/health | grep -q '"status":"ok"'; then
      health_ok=1
      break
    fi
    sleep 1
  done

  if (( health_ok == 1 )); then
    log "Daemon health check [PASS]: http://127.0.0.1:18080/health responded OK."
  else
    die "Daemon health check [FAIL]: http://127.0.0.1:18080/health did not report OK."
  fi

  log "--- Re-verifying Runtimes ---"
  verify_runtime "Codex" "${CODEX_BIN}" || die "Codex runtime verification failed after roll-forward."
  verify_runtime "Antigravity" "${AGY_BIN}" || die "Antigravity runtime verification failed after roll-forward."
  verify_runtime "Kiro" "${KIRO_BIN}" || die "Kiro runtime verification failed after roll-forward."

  log "=== LIVE ROLL-FORWARD COMPLETE (RESTORED FROM BACKUP) ==="
  log "Restored Binary SHA256: $(get_sha256 "${DAEMON_BIN_ACTIVE}")"
  return 0
}

case "${MODE}" in
  --dry-run|-n|dry-run)
    run_dry_run
    ;;
  --live|--execute|live)
    run_live_rollback
    ;;
  --roll-forward|--rollforward|roll-forward|rollforward)
    run_live_rollforward
    ;;
  *)
    log "Usage: $0 [--dry-run | --live | --roll-forward]"
    exit 2
    ;;
esac
