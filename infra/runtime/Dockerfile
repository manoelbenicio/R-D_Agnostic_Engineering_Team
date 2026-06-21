# AgentVerse — Orchestration Runtime container (Cloud Run compatible)
#
# What this image is
#   A self-contained orchestration runtime that exposes the HTTP / WebSocket
#   surface AgentVerse talks to. Internally it wraps the upstream
#   `cli-agent-orchestrator` Python package (awslabs/cli-agent-orchestrator on
#   PyPI) and binds to the port supplied by Cloud Run via $PORT, with a
#   sensible local default of 8080.
#
# Cloud Run notes
#   * Cloud Run injects $PORT at runtime (8080 by default for managed). Our
#     entrypoint honours that.
#   * Knative-style graceful shutdown: SIGTERM is forwarded to the server.
#   * Stateless container: per-tenant state must live in a mounted volume or
#     external storage. Per master spec §13 / cloud-runtime-deployment, the
#     persistent state story is owned by the next change. This Dockerfile is
#     intentionally state-light.
#
# Local build
#   docker build -t agentverse-runtime:latest -f infra/runtime/Dockerfile .
#
# Local run (binds 8080)
#   docker run -d --rm --name agentverse-runtime -p 8080:8080 \
#     -e GO_CORE_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
#     -e GO_CORE_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
#     -e GO_CORE_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173 \
#     agentverse-runtime:latest
#
# Cloud Run deploy
#   See scripts/deploy-cloud.sh and infra/runtime/cloudbuild.yaml.

FROM python:3.12-slim AS base

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# tmux 3.3+ is required for upstream session isolation.
RUN apt-get update && apt-get install -y --no-install-recommends \
        tmux \
        git \
        curl \
        ca-certificates \
        procps \
        tini \
        unzip \
    && rm -rf /var/lib/apt/lists/*

# Install uv (fast Python tool installer).
RUN pip install --no-cache-dir "uv>=0.4,<0.6"

# Node.js 20 (required to bundle npm-distributed worker CLIs, e.g. Codex CLI).
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install the orchestration runtime engine.
RUN uv tool install --python 3.12 cli-agent-orchestrator
ENV PATH="/root/.local/bin:${PATH}"

# Optional worker CLI bundling (task 2.6 — USER DECISION, not chosen here).
#   A fresh deploy can only *execute* canvases if a worker CLI is on PATH.
#   This is intentionally left empty so no CLI is baked in by default; the
#   user picks which to bundle (e.g. Claude Code, Kiro CLI, etc.).
#
#   Pass a command to run via the build arg, e.g.:
#     docker build --build-arg WORKER_CLI="npm i -g @anthropic-ai/claude-code" ...
#     docker build --build-arg WORKER_CLI="uv tool install kiro-cli" ...
#   See docs/cloud-runtime-auth.md "Bundling a worker CLI" for the options
#   matrix and tradeoffs. Empty (default) = no CLI installed.
ARG WORKER_CLI=""
RUN if [ -n "$WORKER_CLI" ]; then \
        echo "Bundling worker CLI: $WORKER_CLI" && sh -c "$WORKER_CLI"; \
    else \
        echo "No WORKER_CLI provided — image ships without a worker CLI (task 2.6 deferred)."; \
    fi

# Antigravity CLI installs as 'agy' but GO Core detects 'gemini'. Create a symlink.
RUN if command -v agy >/dev/null 2>&1; then ln -sf "$(command -v agy)" /usr/local/bin/gemini; fi

# Upstream cli-agent-orchestrator hardcodes CORS_ORIGINS / ALLOWED_HOSTS in
# constants.py and ignores the GO_CORE_CORS_ORIGINS / GO_CORE_ALLOWED_HOSTS env vars.
# Patch constants.py to honour those env vars so the documented run-time
# overrides (and Cloud Run origins) actually take effect.
RUN CONST="$(find /root/.local/share/uv/tools/cli-agent-orchestrator -path '*/cli_agent_orchestrator/constants.py' | head -1)" \
    && grep -q 'AGENTVERSE_CORS_OVERRIDE' "$CONST" || cat >> "$CONST" <<'PY'

# --- AGENTVERSE_CORS_OVERRIDE ---
# Honour GO_CORE_CORS_ORIGINS / GO_CORE_ALLOWED_HOSTS env vars (upstream hardcodes these).
import os as _os
_cors_env = _os.environ.get('GO_CORE_CORS_ORIGINS')
if _cors_env:
    CORS_ORIGINS = [o.strip() for o in _cors_env.split(',') if o.strip()]
_hosts_env = _os.environ.get('GO_CORE_ALLOWED_HOSTS')
if _hosts_env:
    ALLOWED_HOSTS = [h.strip() for h in _hosts_env.split(',') if h.strip()]
PY

# Default CORS / host allow-list — overridable per environment.
# Cloud Run will receive its own origin via $CORS_ORIGINS env at deploy time.
ENV GO_CORE_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
    GO_CORE_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
    GO_CORE_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173

# State volume — sessions, logs, profiles. Cloud Run instances are
# ephemeral; mount a Cloud Storage bucket via Cloud Run gen2 volume mounts
# or use a writable in-memory tmpfs for stateless runs.
VOLUME ["/root/.go-core"]

# Default port for local dev. Cloud Run overrides via $PORT.
EXPOSE 8080

# Entrypoint — uses tini for proper PID 1 signal handling, and exec's into
# the runtime engine, binding to whatever port Cloud Run (or local) supplies.
COPY <<'SH' /usr/local/bin/runtime-entrypoint
#!/usr/bin/env bash
set -e

# Cloud Run injects $PORT (default 8080); local default is 8080.
RUNTIME_PORT="${PORT:-${GO_CORE_PORT:-8080}}"
RUNTIME_HOST="${GO_CORE_HOST:-0.0.0.0}"

# Prefer flag form when supported, fall back to env-driven mode.
if go-core-server --help 2>/dev/null | grep -q -- "--host"; then
    exec go-core-server --host "$RUNTIME_HOST" --port "$RUNTIME_PORT"
else
    exec env GO_CORE_HOST="$RUNTIME_HOST" GO_CORE_PORT="$RUNTIME_PORT" go-core-server
fi
SH
RUN chmod +x /usr/local/bin/runtime-entrypoint

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/runtime-entrypoint"]
