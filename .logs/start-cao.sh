#!/bin/bash
export CAO_CORS_ORIGINS="http://localhost:5173,http://localhost:4173"
export CAO_ALLOWED_HOSTS="127.0.0.1,localhost,0.0.0.0"
export CAO_WS_ALLOWED_CLIENTS="http://localhost:5173,http://localhost:4173"
export CAO_HOST="0.0.0.0"
export CAO_PORT="9889"

# Install tmux if missing
if ! command -v tmux &>/dev/null; then
    echo "[bootstrap] Installing tmux..."
    sudo apt-get update -qq && sudo apt-get install -y -qq tmux
fi

# Install uv if missing
if ! command -v uv &>/dev/null; then
    echo "[bootstrap] Installing uv..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
fi

# Install CAO if missing
if ! command -v cao-server &>/dev/null; then
    echo "[bootstrap] Installing cli-agent-orchestrator..."
    uv tool install cli-agent-orchestrator
    export PATH="$HOME/.local/bin:$PATH"
fi

echo "[bootstrap] Starting cao-server on :9889"
exec cao-server --host 0.0.0.0 --port 9889