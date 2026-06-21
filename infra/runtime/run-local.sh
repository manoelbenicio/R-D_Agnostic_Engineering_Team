#!/usr/bin/env bash
# Run the AgentVerse orchestration runtime LOCALLY in Docker, with the worker
# CLIs bundled and the host's existing CLI credentials mounted in (so the
# container reuses the same logins as your terminal/IDE — no headless tokens).
#
#   bash infra/runtime/run-local.sh build   # build the image (one-off / on change)
#   bash infra/runtime/run-local.sh up       # run the runtime on :8080
#   bash infra/runtime/run-local.sh down     # stop + remove the container
#   bash infra/runtime/run-local.sh logs     # follow logs
#
# After `up`, start the SPA in another shell:  npm run dev   (http://localhost:5173)
set -euo pipefail

IMAGE="agentverse-runtime:local"
NAME="agentverse-runtime"
PORT="${GO_CORE_PORT:-8080}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Worker CLIs bundled into the image (USER DECISION):
#   Codex CLI (npm) + Kiro CLI (official script) + Antigravity CLI (covers Gemini).
#   NOTE: `uv tool install kiro-cli` does NOT work — kiro-cli is not on PyPI;
#   the official install script is the headless-friendly method.
WORKER_CLI="npm install --global @openai/codex \
  && curl -fsSL https://cli.kiro.dev/install | bash \
  && curl -fsSL https://antigravity.google/cli/install.sh | bash"

# Host credential dirs mounted so the container reuses your existing CLI
# logins (same auth as your terminal/IDE — no headless tokens needed).
#
# IMPORTANT: the runtime engine writes its DB / logs under /root/.aws
# (cli-agent-orchestrator hardcodes that path), so /root/.aws itself MUST be
# writable. We therefore mount only the AWS *credentials/config files*
# read-only, and let the rest of /root/.aws be a writable container layer.
# The other CLI logins (.codex/.kiro/.gemini) are read-only.
CRED_MOUNTS=()
for d in .codex .kiro .gemini; do
  if [ -d "$HOME/$d" ]; then
    CRED_MOUNTS+=( -v "$HOME/$d:/root/$d:ro" )
  fi
done
# AWS: mount just the credential + config files read-only (not the whole dir),
# so the engine can still create /root/.aws/cli-agent-orchestrator at runtime.
for f in credentials config; do
  if [ -f "$HOME/.aws/$f" ]; then
    CRED_MOUNTS+=( -v "$HOME/.aws/$f:/root/.aws/$f:ro" )
  fi
done
# Persist runtime state/logs across restarts in a named volume.
CRED_MOUNTS+=( -v "agentverse-runtime-state:/root/.aws/cli-agent-orchestrator" )

build() {
  echo "Building $IMAGE (bundling Codex + Kiro + Antigravity)…"
  docker build -t "$IMAGE" -f "$ROOT/infra/runtime/Dockerfile" \
    --build-arg WORKER_CLI="$WORKER_CLI" "$ROOT"
}

up() {
  docker rm -f "$NAME" >/dev/null 2>&1 || true
  echo "Starting $NAME on :$PORT (mounting ${#CRED_MOUNTS[@]} credential paths)…"
  docker run -d --rm --name "$NAME" -p "$PORT:8080" \
    -e GO_CORE_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
    -e GO_CORE_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
    -e GO_CORE_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173 \
    "${CRED_MOUNTS[@]}" \
    "$IMAGE"
  echo "Runtime up. Health: curl http://localhost:$PORT/health"
  echo "Now run the SPA:  npm run dev   →  http://localhost:5173"
}

down() { docker rm -f "$NAME" >/dev/null 2>&1 && echo "stopped" || echo "not running"; }
logs() { docker logs -f "$NAME"; }

case "${1:-up}" in
  build) build ;;
  up)    up ;;
  down)  down ;;
  logs)  logs ;;
  *) echo "usage: $0 {build|up|down|logs}"; exit 1 ;;
esac
