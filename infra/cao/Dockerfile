# AgentVerse — CAO runtime container
#
# Purpose
#   Run the CLI Agent Orchestrator (CAO) from awslabs/cli-agent-orchestrator
#   as a self-contained service exposing the HTTP/WebSocket surface AgentVerse
#   talks to (default port 9889).
#
# Notes
#   * CAO is a Python/uv package on PyPI: `cli-agent-orchestrator`.
#     The `cao-server` entrypoint serves the same HTTP+WS API the SPA expects.
#   * tmux 3.3+ is required for CAO's session isolation. Debian bookworm
#     (python:3.12-slim base) ships tmux 3.3a, which satisfies the minimum.
#   * To actually deploy real agents you would also install the CLI providers
#     (Claude Code, Kiro CLI, Codex CLI, Gemini CLI, etc.) inside this image.
#     For the AgentVerse v1 health-check + UI walkthrough that is not
#     required: the SPA renders correctly with CAO running but no providers.
#
# Build
#   docker build -t agentverse-cao:latest -f infra/cao/Dockerfile .
#
# Run
#   docker run -d --rm --name agentverse-cao -p 9889:9889 \
#     -e CAO_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
#     -e CAO_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
#     -e CAO_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173 \
#     agentverse-cao:latest

FROM python:3.12-slim AS base

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# tmux 3.3+ is mandatory; git + curl + ca-certs are needed for CAO and uv.
RUN apt-get update && apt-get install -y --no-install-recommends \
        tmux \
        git \
        curl \
        ca-certificates \
        procps \
    && rm -rf /var/lib/apt/lists/*

# Install uv (fast Python package + tool manager). Pin to a known-good version.
RUN pip install --no-cache-dir "uv>=0.4,<0.6"

# Install CAO globally as a uv tool. This pulls cli-agent-orchestrator and
# exposes the `cao` and `cao-server` binaries.
RUN uv tool install --python 3.12 cli-agent-orchestrator
ENV PATH="/root/.local/bin:${PATH}"

# Quick sanity check on the entrypoints during the build (fails fast if the
# package was renamed upstream).
RUN cao --version || true && which cao-server

# Upstream cli-agent-orchestrator hardcodes CORS_ORIGINS / ALLOWED_HOSTS in
# constants.py and ignores the CAO_CORS_ORIGINS / CAO_ALLOWED_HOSTS env vars.
# Patch constants.py to honour those env vars (see docs/cao-cors.md).
RUN CONST="$(find /root/.local/share/uv/tools/cli-agent-orchestrator -path '*/cli_agent_orchestrator/constants.py' | head -1)" \
    && grep -q 'AGENTVERSE_CORS_OVERRIDE' "$CONST" || cat >> "$CONST" <<'PY'

# --- AGENTVERSE_CORS_OVERRIDE ---
# Honour CAO_CORS_ORIGINS / CAO_ALLOWED_HOSTS env vars (upstream hardcodes these).
import os as _os
_cors_env = _os.environ.get('CAO_CORS_ORIGINS')
if _cors_env:
    CORS_ORIGINS = [o.strip() for o in _cors_env.split(',') if o.strip()]
_hosts_env = _os.environ.get('CAO_ALLOWED_HOSTS')
if _hosts_env:
    ALLOWED_HOSTS = [h.strip() for h in _hosts_env.split(',') if h.strip()]
PY

# Add AgentVerse auth/session discovery endpoints expected by the SPA.
COPY infra/cao/auth_routes.py /tmp/agentverse-auth-routes.py
RUN API_DIR="$(find /root/.local/share/uv/tools/cli-agent-orchestrator -path '*/cli_agent_orchestrator/api' -type d | head -1)" \
    && cp /tmp/agentverse-auth-routes.py "$API_DIR/auth_routes.py" \
    && MAIN="$API_DIR/main.py" \
    && if ! grep -q 'from cli_agent_orchestrator.api.auth_routes import auth_router' "$MAIN"; then \
        sed -i '/^from cli_agent_orchestrator.utils.terminal import/a from cli_agent_orchestrator.api.auth_routes import auth_router' "$MAIN"; \
    fi \
    && if ! grep -q 'app.include_router(auth_router)' "$MAIN"; then \
        sed -i '/^# Security: DNS Rebinding Protection/i app.include_router(auth_router)\n' "$MAIN"; \
    fi

EXPOSE 9889

# Runtime CORS / host allow-lists per docs/cao-cors.md. Override at run time.
ENV CAO_CORS_ORIGINS=http://localhost:5173,http://localhost:4173 \
    CAO_ALLOWED_HOSTS=127.0.0.1,localhost,0.0.0.0 \
    CAO_WS_ALLOWED_CLIENTS=http://localhost:5173,http://localhost:4173

# CAO state (sessions, logs, profiles) lives under ~/.cao/ — mount this volume
# to persist between container restarts.
VOLUME ["/root/.cao"]

# Default to listening on all interfaces inside the container so the host can
# port-forward 9889. cao-server reads --host / --port; if the build of
# cli-agent-orchestrator does not honour these flags, the entrypoint script
# below falls back to environment variables CAO_HOST / CAO_PORT.
ENV CAO_HOST=0.0.0.0 \
    CAO_PORT=9889

ENTRYPOINT ["/usr/local/bin/cao-entrypoint"]

# Tiny entrypoint that tries flags first, then env, then the default.
COPY <<'SH' /usr/local/bin/cao-entrypoint
#!/usr/bin/env bash
set -e
if cao-server --help 2>/dev/null | grep -q -- "--host"; then
    exec cao-server --host "${CAO_HOST:-0.0.0.0}" --port "${CAO_PORT:-9889}"
else
    # Older versions only read env vars
    exec env CAO_HOST="${CAO_HOST:-0.0.0.0}" CAO_PORT="${CAO_PORT:-9889}" cao-server
fi
SH
RUN chmod +x /usr/local/bin/cao-entrypoint
