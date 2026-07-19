#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf -- "${TMP}"' EXIT

mkdir -p "${TMP}/bin" "${TMP}/host"
cat >"${TMP}/bin/kiro-real" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "${XDG_DATA_HOME}"
EOF
chmod 700 "${TMP}/bin/kiro-real"

output="$({
  HOME="${TMP}/host" \
  AGENT_CRED_ISOLATION_HOST_HOME="${TMP}/host" \
  AGENT_CRED_ISOLATION_ROOT="${TMP}/slots" \
  KIRO_WRAPPER_REAL_BIN="${TMP}/bin/kiro-real" \
  KIRO_WRAPPER_ISOLATION_SCRIPT="${ROOT}/scripts/ops/agent-cred-isolation.sh" \
  HERDR_PANE_ID="test-pane-a" \
  bash "${ROOT}/scripts/ops/kiro-cli-isolated-wrapper.sh"
} 2>&1)"

case "${output}" in
  "${TMP}/slots/slots/slot-"*/xdg-data) ;;
  *) printf 'FAIL: wrapper did not export an isolated XDG_DATA_HOME: %s\n' "${output}" >&2; exit 1 ;;
esac

printf 'PASS: Kiro wrapper enforces a per-terminal XDG_DATA_HOME\n'
