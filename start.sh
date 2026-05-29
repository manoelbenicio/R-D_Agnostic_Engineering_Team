st#!/usr/bin/env bash
#
# AgentVerse system bootstrap — single entrypoint to bring the platform up.
#
# Usage:
#   ./start.sh                         # default: production preview (built dist/)
#   ./start.sh agentic_system          # alias of default — matches the user vocabulary
#   ./start.sh dev                     # vite dev server with HMR (real CAO required)
#   ./start.sh stop                    # stop everything we started
#   ./start.sh status                  # report what's listening on which port
#   ./start.sh --help                  # print help
#
# What it does, in order:
#   1. Sanity-check Node ≥ 20.10 and npm ≥ 10.2.
#   2. Verify dependencies are installed; run `npm ci` if `node_modules/` is absent.
#   3. Build the production bundle if `dist/` is stale or missing.
#   4. Probe the CAO server at $VITE_CAO_BASE_URL (default http://127.0.0.1:9889).
#      - If reachable: continue.
#      - If unreachable: print exactly what to do, but still start AgentVerse so the
#        UI shell, settings, and design surfaces are usable. The Health page will
#        show CAO offline; canvas deploys / terminals / flows will fail until CAO
#        is up — that is the documented v1 behaviour (master spec §13).
#   5. Start `vite preview` (or `vite dev`) detached so it survives this script.
#   6. Tail the log briefly and confirm the URL responds with HTTP 200.
#   7. Print the URLs and the stop command.
#
# CAO is an external dependency by design (master spec §13; cloud-runtime-deployment
# is post-launch). This script CANNOT install CAO. If you have a CAO Docker image
# or local install path, set $CAO_DOCKER_IMAGE or $CAO_START_CMD and the script will
# try to start it for you.

set -euo pipefail

#############################################
# Config
#############################################
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT="$SCRIPT_DIR"

# Allow `./scripts/bootstrap.sh` to find the repo root too
if [[ -d "$SCRIPT_DIR/../package.json" || -f "$SCRIPT_DIR/../package.json" ]]; then
  REPO_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
fi

cd "$REPO_ROOT"

# Default env, overridable from the caller's shell
: "${VITE_CAO_BASE_URL:=http://127.0.0.1:9889}"
: "${PREVIEW_PORT:=4173}"
: "${DEV_PORT:=5173}"
: "${LOG_DIR:=/tmp}"
: "${CAO_DOCKER_IMAGE:=agentverse-runtime:latest}"   # auto-built from infra/runtime/Dockerfile if missing
: "${CAO_DOCKERFILE:=infra/runtime/Dockerfile}"
: "${CAO_START_CMD:=}"      # e.g. "uv run cao serve --port 9889"
: "${CLOUD_RUNTIME_URL:=}"  # set after `./start.sh cloud-deploy`; used by `./start.sh cloud`
: "${FIREBASE_HOSTING_URL:=}"
: "${OPEN_BROWSER:=auto}"   # auto | yes | no

PREVIEW_LOG="$LOG_DIR/agentverse-preview.log"
DEV_LOG="$LOG_DIR/agentverse-dev.log"
CAO_LOG="$LOG_DIR/agentverse-cao.log"

#############################################
# Pretty output
#############################################
if [[ -t 1 ]]; then
  C_BOLD="\033[1m"; C_DIM="\033[2m"; C_GRN="\033[32m"; C_YEL="\033[33m"; C_RED="\033[31m"; C_CYA="\033[36m"; C_OFF="\033[0m"
else
  C_BOLD=""; C_DIM=""; C_GRN=""; C_YEL=""; C_RED=""; C_CYA=""; C_OFF=""
fi

step() { printf "${C_CYA}${C_BOLD}»${C_OFF} %s\n" "$*"; }
ok()   { printf "  ${C_GRN}✓${C_OFF} %s\n" "$*"; }
warn() { printf "  ${C_YEL}!${C_OFF} %s\n" "$*"; }
err()  { printf "  ${C_RED}✗${C_OFF} %s\n" "$*" >&2; }
hr()   { printf "${C_DIM}%s${C_OFF}\n" "────────────────────────────────────────────────────────────"; }

#############################################
# Helpers
#############################################
ver_ge() {
  # ver_ge "20.10.0" "20.10" -> 0 (true)
  [[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" == "$2" ]]
}

probe() {
  # probe URL TIMEOUT — prints HTTP code or 000
  local url="$1"; local timeout="${2:-3}"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "$timeout" "$url" 2>/dev/null) || true
  printf '%s' "${code:-000}"
}

is_listening() {
  # is_listening PORT
  local port="$1"
  ss -tln 2>/dev/null | awk '{print $4}' | grep -qE ":$port\$"
}

spawn_detached() {
  # spawn_detached LOGFILE CMD ARGS...
  local log="$1"; shift
  ( setsid nohup "$@" </dev/null >"$log" 2>&1 & ) 2>/dev/null
}

cleanup_pid_pattern() {
  # cleanup_pid_pattern PATTERN
  local pat="$1"
  local pids
  pids=$(pgrep -f "$pat" 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    echo "$pids" | xargs -r kill 2>/dev/null || true
    sleep 1
    pids=$(pgrep -f "$pat" 2>/dev/null || true)
    [[ -n "$pids" ]] && echo "$pids" | xargs -r kill -9 2>/dev/null || true
  fi
}

open_url() {
  local url="$1"
  case "$OPEN_BROWSER" in
    no) return 0 ;;
    yes|auto)
      if command -v wslview >/dev/null 2>&1; then wslview "$url" 2>/dev/null || true
      elif command -v xdg-open >/dev/null 2>&1; then xdg-open "$url" 2>/dev/null || true
      elif command -v open >/dev/null 2>&1; then open "$url" 2>/dev/null || true
      elif command -v cmd.exe >/dev/null 2>&1; then cmd.exe /c start "" "$url" 2>/dev/null || true
      fi
      ;;
  esac
}

#############################################
# Subcommands
#############################################
cmd_help() {
  cat <<EOF
AgentVerse bootstrap

Usage: ./start.sh [COMMAND]

Commands:
  agentic_system    (default) Build if needed, probe runtime, start production preview
  prod              Alias of agentic_system
  local             Same as default — explicit local-mode (Docker runtime)
  dev               Start vite dev server (HMR; runtime required)
  cloud-deploy      Deploy runtime + SPA to GCP (Cloud Run + Firebase Hosting)
  cloud             Open the deployed cloud SPA in the browser
  stop              Stop everything this script started (preview + dev + runtime container)
  status            Report what is currently listening on AgentVerse + runtime ports
  --help, -h        This help

Environment overrides:
  VITE_CAO_BASE_URL    Runtime endpoint to probe       (default http://127.0.0.1:9889)
  PREVIEW_PORT         Production preview port         (default 4173)
  DEV_PORT             Vite dev server port            (default 5173)
  CAO_DOCKER_IMAGE     Image to start runtime          (default agentverse-runtime:latest)
  CAO_DOCKERFILE       Source Dockerfile if rebuilding (default infra/runtime/Dockerfile)
  CAO_START_CMD        Shell command to start runtime  (alternative to docker)
  CLOUD_RUNTIME_URL    Cloud Run URL for `cloud` cmd   (set after cloud-deploy)
  FIREBASE_HOSTING_URL SPA URL on Firebase Hosting     (default https://\${PROJECT_ID}.web.app)
  OPEN_BROWSER         auto | yes | no                 (default auto)

Modes
  Local mode (default): runtime is the local Docker container.
  Cloud mode: runtime + SPA live in GCP. Run \`./start.sh cloud-deploy\` once
  with your gcloud + firebase CLIs authenticated, then \`./start.sh cloud\` to
  open the deployed URL. The SPA can also be flipped between modes at runtime
  via Settings → General → Runtime Base URL.

Examples:
  ./start.sh
  ./start.sh dev
  CAO_DOCKER_IMAGE=agentverse-runtime:dev ./start.sh
  PROJECT_ID=my-gcp-project ./start.sh cloud-deploy
  ./start.sh cloud
  ./start.sh stop
EOF
}

check_node() {
  step "Checking Node + npm"
  command -v node >/dev/null 2>&1 || { err "node not found"; exit 2; }
  command -v npm  >/dev/null 2>&1 || { err "npm not found";  exit 2; }
  local nv; nv="$(node -v | sed 's/^v//')"
  local mv; mv="$(npm -v)"
  if ! ver_ge "$nv" "20.10.0"; then err "Node $nv < 20.10 required"; exit 2; fi
  if ! ver_ge "$mv" "10.2.0";  then err "npm $mv < 10.2 required";  exit 2; fi
  ok "node $nv · npm $mv"
}

check_deps() {
  step "Checking dependencies"
  if [[ ! -d node_modules ]]; then
    warn "node_modules/ missing — running npm ci"
    npm ci
  fi
  ok "dependencies installed"
}

build_if_stale() {
  step "Checking production bundle"
  local need_build=0
  if [[ ! -f dist/index.html ]]; then
    need_build=1
  else
    # Rebuild if any source file is newer than the built index.html
    if find src public index.html vite.config.ts package.json .env.production -type f -newer dist/index.html 2>/dev/null | grep -q .; then
      need_build=1
    fi
  fi

  if [[ "$need_build" == 1 ]]; then
    step "Building production bundle (npm run build)"
    npm run build
    ok "build complete"
  else
    ok "dist/ is up-to-date"
  fi

  # Belt-and-braces: never ship the MSW worker in production
  if [[ -f dist/mockServiceWorker.js ]]; then
    rm -f dist/mockServiceWorker.js
    warn "stripped dist/mockServiceWorker.js (no mock infrastructure in production)"
  fi
}

start_cao_if_configured() {
  step "Probing CAO at $VITE_CAO_BASE_URL"
  local code; code=$(probe "$VITE_CAO_BASE_URL/health" 3)
  if [[ "$code" == "200" ]]; then
    ok "CAO is reachable (HTTP $code)"
    return 0
  fi

  warn "CAO is NOT reachable (HTTP $code)"

  if [[ -n "$CAO_DOCKER_IMAGE" ]]; then
    step "Starting CAO from Docker image: $CAO_DOCKER_IMAGE"
    if ! command -v docker >/dev/null 2>&1; then
      err "docker not found but CAO_DOCKER_IMAGE is set"
      return 1
    fi
    # Auto-build the image from infra/cao/Dockerfile if it isn't already present
    if ! docker image inspect "$CAO_DOCKER_IMAGE" >/dev/null 2>&1; then
      if [[ -f "$CAO_DOCKERFILE" ]]; then
        step "Image $CAO_DOCKER_IMAGE not found locally — building from $CAO_DOCKERFILE"
        docker build -t "$CAO_DOCKER_IMAGE" -f "$CAO_DOCKERFILE" . >>"$CAO_LOG" 2>&1 || {
          err "docker build failed (see $CAO_LOG)"; return 1; }
        ok "image built"
      else
        err "image $CAO_DOCKER_IMAGE not present and no Dockerfile at $CAO_DOCKERFILE"
        return 1
      fi
    fi
    docker rm -f agentverse-runtime agentverse-cao >/dev/null 2>&1 || true
    docker run -d --name agentverse-runtime \
      -p 9889:9889 \
      -e CAO_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
      -e CAO_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
      -e CAO_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173 \
      -v agentverse-runtime-state:/root/.cao \
      "$CAO_DOCKER_IMAGE" >>"$CAO_LOG" 2>&1 || { err "docker run failed (see $CAO_LOG)"; return 1; }
    sleep 3
    local i
    for i in $(seq 1 30); do
      code=$(probe "$VITE_CAO_BASE_URL/health" 2)
      if [[ "$code" == "200" ]]; then ok "CAO is up (HTTP 200)"; return 0; fi
      sleep 1
    done
    err "CAO did not become healthy within 30s — check $CAO_LOG"
    return 1
  fi

  if [[ -n "$CAO_START_CMD" ]]; then
    step "Starting CAO via CAO_START_CMD"
    spawn_detached "$CAO_LOG" bash -lc "$CAO_START_CMD"
    local i
    for i in $(seq 1 30); do
      code=$(probe "$VITE_CAO_BASE_URL/health" 2)
      if [[ "$code" == "200" ]]; then ok "CAO is up (HTTP 200)"; return 0; fi
      sleep 1
    done
    err "CAO did not become healthy within 30s — check $CAO_LOG"
    return 1
  fi

  cat <<EOF
  ${C_YEL}!${C_OFF} CAO is required for canvas deploys, terminal streaming, and flows.
    Per master spec §13 it is an EXTERNAL service. To start it, do ONE of:

      1) Set CAO_DOCKER_IMAGE if you have a CAO image:
           CAO_DOCKER_IMAGE=cao-server:latest ./start.sh

      2) Set CAO_START_CMD to a local command (e.g. uv/pip-installed CAO):
           CAO_START_CMD="uv run cao serve --port 9889" ./start.sh

      3) Start CAO yourself in another terminal at $VITE_CAO_BASE_URL.

    The SPA will start anyway and the Health page will show CAO offline.
    See docs/cao-cors.md for the env vars CAO needs to allow this SPA.

EOF
  return 0
}

start_preview() {
  step "Starting production preview on :$PREVIEW_PORT"
  if is_listening "$PREVIEW_PORT"; then
    warn "port $PREVIEW_PORT already in use — assuming a preview is already running"
  else
    spawn_detached "$PREVIEW_LOG" npm run preview
    local i
    for i in $(seq 1 15); do
      sleep 1
      if [[ "$(probe "http://localhost:$PREVIEW_PORT/" 2)" == "200" ]]; then
        ok "preview responding"
        break
      fi
    done
  fi
  if [[ "$(probe "http://localhost:$PREVIEW_PORT/" 2)" != "200" ]]; then
    err "preview did not come up — check $PREVIEW_LOG"
    return 1
  fi
}

start_dev() {
  step "Starting vite dev server on :$DEV_PORT"
  if is_listening "$DEV_PORT"; then
    warn "port $DEV_PORT already in use — assuming dev server is already running"
  else
    spawn_detached "$DEV_LOG" npm run dev
    local i
    for i in $(seq 1 15); do
      sleep 1
      if [[ "$(probe "http://localhost:$DEV_PORT/" 2)" == "200" ]]; then
        ok "dev server responding"
        break
      fi
    done
  fi
  if [[ "$(probe "http://localhost:$DEV_PORT/" 2)" != "200" ]]; then
    err "dev server did not come up — check $DEV_LOG"
    return 1
  fi
}

cmd_status() {
  hr
  step "AgentVerse system status"
  hr
  printf "  preview (%s) : HTTP %s\n" "$PREVIEW_PORT" "$(probe "http://localhost:$PREVIEW_PORT/" 2)"
  printf "  dev     (%s) : HTTP %s\n" "$DEV_PORT" "$(probe "http://localhost:$DEV_PORT/" 2)"
  printf "  CAO     (%s) : HTTP %s\n" "${VITE_CAO_BASE_URL##*/}" "$(probe "$VITE_CAO_BASE_URL/health" 2)"
  hr
  step "Processes"
  pgrep -af "vite preview" 2>/dev/null | sed 's/^/  preview  /' || true
  pgrep -af "vite dev|vite\$"   2>/dev/null | sed 's/^/  dev      /' || true
  if command -v docker >/dev/null 2>&1; then
    docker ps --filter name=agentverse-cao --format '  cao      {{.ID}} {{.Image}} {{.Status}}' 2>/dev/null || true
  fi
  hr
}

cmd_stop() {
  step "Stopping everything"
  cleanup_pid_pattern "vite preview"
  cleanup_pid_pattern "vite dev"
  if command -v docker >/dev/null 2>&1; then
    docker rm -f agentverse-cao >/dev/null 2>&1 || true
  fi
  ok "stopped"
}

cmd_up_prod() {
  hr
  step "Bootstrapping AgentVerse — production"
  hr
  check_node
  check_deps
  build_if_stale
  start_cao_if_configured || true
  start_preview
  hr
  ok "AgentVerse production preview is live"
  printf "    URL          : ${C_BOLD}http://localhost:%s${C_OFF}\n" "$PREVIEW_PORT"
  printf "    CAO endpoint : %s (HTTP %s)\n" "$VITE_CAO_BASE_URL" "$(probe "$VITE_CAO_BASE_URL/health" 2)"
  printf "    Logs         : %s\n" "$PREVIEW_LOG"
  printf "    Stop         : ./start.sh stop\n"
  hr
  open_url "http://localhost:$PREVIEW_PORT"
}

cmd_up_dev() {
  hr
  step "Bootstrapping AgentVerse — dev (HMR)"
  hr
  check_node
  check_deps
  start_cao_if_configured || true
  start_dev
  hr
  ok "AgentVerse dev server is live"
  printf "    URL          : ${C_BOLD}http://localhost:%s${C_OFF}\n" "$DEV_PORT"
  printf "    CAO endpoint : %s (HTTP %s)\n" "$VITE_CAO_BASE_URL" "$(probe "$VITE_CAO_BASE_URL/health" 2)"
  printf "    Logs         : %s\n" "$DEV_LOG"
  printf "    Stop         : ./start.sh stop\n"
  hr
  open_url "http://localhost:$DEV_PORT"
}

cmd_cloud_deploy() {
  hr
  step "Bootstrapping AgentVerse — cloud deploy"
  hr
  if [[ ! -x scripts/deploy-cloud.sh ]]; then
    err "scripts/deploy-cloud.sh missing or not executable"
    exit 2
  fi
  exec ./scripts/deploy-cloud.sh
}

cmd_cloud_open() {
  hr
  step "Opening deployed AgentVerse cloud SPA"
  hr
  local url="${FIREBASE_HOSTING_URL:-}"
  if [[ -z "$url" && -n "${PROJECT_ID:-}" ]]; then
    url="https://${PROJECT_ID}.web.app"
  fi
  if [[ -z "$url" && -f .env.production.local ]]; then
    # Best-effort: derive from cloud-deploy output if present
    url="$(grep -oE 'https://[^=]*\.web\.app' .env.production.local | head -n1)"
  fi
  if [[ -z "$url" ]]; then
    err "no cloud URL known. Run ./start.sh cloud-deploy first, or set FIREBASE_HOSTING_URL."
    exit 2
  fi
  ok "URL: $url"
  open_url "$url"
}

#############################################
# Dispatch
#############################################
case "${1:-agentic_system}" in
  -h|--help|help)         cmd_help ;;
  agentic_system|prod|local|"") cmd_up_prod ;;
  dev)                    cmd_up_dev ;;
  cloud-deploy)           cmd_cloud_deploy ;;
  cloud)                  cmd_cloud_open ;;
  stop)                   cmd_stop ;;
  status)                 cmd_status ;;
  *)
    err "unknown command: $1"
    cmd_help
    exit 64
    ;;
esac
