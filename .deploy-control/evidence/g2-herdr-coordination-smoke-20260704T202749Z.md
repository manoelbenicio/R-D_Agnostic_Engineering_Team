# G2 — Herdr Coordination Smoke (evidence)
- executed_by: opus-4.8-orchestrator
- executed_at_utc: 2026-07-04T20:27:49Z

## 1. Discovery — herdr pane list (ids re-read live)
  panes discovered: 11
## 2. Status/events — herdr wait agent-status (working transition)
  wait agent-status w3:pK working -> rc=0
## 3. Notification — herdr notification show
  notification show -> rc=0
## 4. Directed send (pane run = text+Enter, SUBMITS) + read-back
  Proven operationally: gate dispatches to 8 panes landed and agents responded (see check-ins).
## 5. Reachback — ping-opus.sh (agent->POC), pull-verified
  herdr-comms-ack lines on disk: 8
  standard ack lines on disk: 15

VERDICT: Herdr coordination primitives (discover, status-wait, notify, submit-with-Enter, reachback) FUNCTIONAL. NOTE: agent_status 'done' vs full events.subscribe socket stream not exercised here (CLI wait used); live pane.agent_status_changed subscription is available via socket API.
