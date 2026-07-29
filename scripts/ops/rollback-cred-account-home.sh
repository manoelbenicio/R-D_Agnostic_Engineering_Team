#!/usr/bin/env bash
# One-command rollback artifact for credential-account-home restoration (ORQ-23)
# Validates prior daemon references, 3 required runtimes (Codex, Antigravity, Kiro),
# and performs content-free identity reporting without reading credential secrets.

set -euo pipefail

# Configuration
DAEMON_BIN_DIR="/home/ec2-user/.local/lib/multica/bin"
DAEMON_BIN_ACTIVE="${DAEMON_BIN_DIR}/multica-auth-credential-home-v1"
DAEMON_BIN_PREVIOUS="${DAEMON_BIN_DIR}/multica-auth-credential-home-v1.previous"
DAEMON_SERVICE_FILE="/home/ec2-user/.config/systemd/user/multica-daemon-orq2-credential.service"
DAEMON_SERVICE_NAME="multica-daemon-orq2-credential.service"

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
  # Count active task execution processes under multica daemon
  count="$(pgrep -f "agy.*--log-file|codex.*--log-file|kiro.*--log-file" 2>/dev/null | wc -l || true)"
  printf '%d' "${count}"
}

# 2. Content-free runtime verification
verify_runtime() {
  local name="$1"
  local bin_path="$2"
  if [[ -x "${bin_path}" ]]; then
    local version_info
    version_info="$("${bin_path}" --version 2>/dev/null | head -n 1 || echo "executable present")"
    log "Runtime check [PASS]: ${name} -> path=${bin_path} (${version_info})"
    return 0
  else
    log "Runtime check [FAIL]: ${name} -> path=${bin_path} (not found or not executable)"
    return 1
  fi
}

# 3. Content-free identity calculation
get_sha256() {
  local target_file="$1"
  if [[ -f "${target_file}" ]]; then
    sha256sum "${target_file}" | awk '{print $1}'
  else
    echo "FILE_MISSING"
  fi
}

run_dry_run() {
  local active_tasks
  active_tasks="$(count_active_tasks)"

  log "=== ONE-COMMAND ROLLBACK DRY-RUN VERIFICATION (ORQ-23) ==="
  log "Mode: DRY-RUN (Content-Free Inspection)"
  log "Active Product Tasks Count: ${active_tasks}"
  log "GTL Cutover Dispatch Flag: ${DISPATCH_CUTOVER}"
  log "F3 Cost Gate Status: BLOCKED on ORQ-13/14 (not claimed)"

  log "--- Checking Daemon Binary References ---"
  log "Active Daemon Binary: ${DAEMON_BIN_ACTIVE} (SHA256: $(get_sha256 "${DAEMON_BIN_ACTIVE}"))"
  log "Prior Daemon Binary:  ${DAEMON_BIN_PREVIOUS} (SHA256: $(get_sha256 "${DAEMON_BIN_PREVIOUS}"))"

  if [[ ! -f "${DAEMON_BIN_PREVIOUS}" ]]; then
    die "Prior daemon binary ${DAEMON_BIN_PREVIOUS} does not exist!"
  fi

  log "--- Checking Systemd Service Reference ---"
  log "Systemd Unit File: ${DAEMON_SERVICE_FILE} (SHA256: $(get_sha256 "${DAEMON_SERVICE_FILE}"))"
  if [[ ! -f "${DAEMON_SERVICE_FILE}" ]]; then
    die "Systemd service file ${DAEMON_SERVICE_FILE} does not exist!"
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
  if [[ -d "${CRED_HOMES_ROOT}" ]]; then
    log "Credential Homes Root: ${CRED_HOMES_ROOT} (Directory Present, Mode: $(stat -c '%a' "${CRED_HOMES_ROOT}"))"
  else
    log "Credential Homes Root: ${CRED_HOMES_ROOT} (Will be created on demand)"
  fi

  log "=== DRY-RUN VERIFICATION RESULT: PASS ==="
  if (( active_tasks > 0 )); then
    log "SAFEGUARD: Active product tasks (${active_tasks}) are currently running."
    log "Destructive live rollback MUST NOT be executed until active tasks = 0 and GTL dispatches cutover."
  else
    log "Active product tasks count is 0. System is ready for GTL cutover dispatch when requested."
  fi

  return 0
}

run_live_rollback() {
  local active_tasks
  active_tasks="$(count_active_tasks)"

  log "=== ONE-COMMAND ROLLBACK LIVE EXECUTION (ORQ-23) ==="
  log "Active Product Tasks Count: ${active_tasks}"
  log "GTL Cutover Dispatch Flag: ${DISPATCH_CUTOVER}"

  if (( active_tasks > 0 )); then
    die "LIVE ROLLBACK REFUSED: active product tasks (${active_tasks}) remain running. Live rollback requires active task count = 0."
  fi

  if [[ "${DISPATCH_CUTOVER}" != "1" && "${FORCE_ROLLBACK:-0}" != "1" ]]; then
    die "LIVE ROLLBACK REFUSED: GTL cutover has not been dispatched. Set GTL_CUTOVER_DISPATCH=1 or FORCE_ROLLBACK=1 to proceed."
  fi

  log "--- Executing Prior Daemon Binary Restoration ---"
  log "Restoring ${DAEMON_BIN_PREVIOUS} -> ${DAEMON_BIN_ACTIVE}"
  cp -p "${DAEMON_BIN_PREVIOUS}" "${DAEMON_BIN_ACTIVE}"
  chmod 0755 "${DAEMON_BIN_ACTIVE}"

  log "--- Reloading and Restarting Daemon Service ---"
  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload || true
    systemctl --user restart "${DAEMON_SERVICE_NAME}" || true
    log "Daemon service ${DAEMON_SERVICE_NAME} restarted."
  else
    log "systemctl not available; daemon restart skipped."
  fi

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
  return 0
}

case "${MODE}" in
  --dry-run|-n|dry-run)
    run_dry_run
    ;;
  --live|--execute|live)
    run_live_rollback
    ;;
  *)
    log "Usage: $0 [--dry-run | --live]"
    exit 2
    ;;
esac
