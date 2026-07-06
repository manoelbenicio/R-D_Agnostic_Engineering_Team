#!/usr/bin/env bash
# ping-opus.sh — reliable agent -> Tech-Lead (opus-4.8-orchestrator) message delivery.
#
# Usage:
#   bash /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/ping-opus.sh "[<YourAgentName>] <question | blocker | status | handoff>"
#
# WHY THIS EXISTS:
#   `herdr agent send <name>` writes literal text WITHOUT pressing Enter, so it strands
#   in the orchestrator's input buffer and is NEVER ingested. `herdr pane` commands
#   (run/send-keys) DO submit, but they require the pane id, not the agent name.
#   This helper resolves the orchestrator's CURRENT pane id (ids can compact) and uses
#   `herdr pane run` (text + Enter) so the message is actually delivered and read.
#
# Falls back to a Herdr notification (with sound) if the orchestrator pane can't be found.
set -uo pipefail

MSG="${1:-}"
if [ -z "$MSG" ]; then
  echo "usage: bash ping-opus.sh \"[<Agent>] <message>\"" >&2
  exit 2
fi

PANE="$(herdr agent list 2>/dev/null | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
    print(next((a["pane_id"] for a in d["result"]["agents"]
                if a.get("name") == "opus-4.8-orchestrator"), ""))
except Exception:
    print("")
' 2>/dev/null)"

if [ -z "$PANE" ]; then
  herdr notification show "opus-4.8-orchestrator unreachable" --body "$MSG" --sound request >/dev/null 2>&1 || true
  echo "opus pane not found -> sent notification fallback" >&2
  exit 1
fi

herdr pane run "$PANE" "$MSG"
echo "delivered to opus-4.8-orchestrator ($PANE)"
