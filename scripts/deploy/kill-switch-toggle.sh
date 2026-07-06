#!/bin/bash
# Kill-switch toggle for Smart Context
# Usage: kill-switch-toggle.sh [enable|disable|status]

STORE="/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/kill-switch/smart_context.json"

case "${1:-status}" in
  enable)
    echo "{\"enabled\":true,\"scope\":\"global\",\"updated\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > "$STORE"
    echo "SMART_CONTEXT: ENABLED"
    cat "$STORE"
    ;;
  disable)
    echo "{\"enabled\":false,\"scope\":\"global\",\"updated\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > "$STORE"
    echo "SMART_CONTEXT: DISABLED (kill-switch active)"
    cat "$STORE"
    ;;
  status)
    if [ -f "$STORE" ]; then
      echo "KILL-SWITCH STATE:"
      cat "$STORE"
    else
      echo "NO KILL-SWITCH STORE"
    fi
    ;;
  *)
    echo "Usage: $0 [enable|disable|status]"
    exit 1
    ;;
esac
