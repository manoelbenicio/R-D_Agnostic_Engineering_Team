---
agent: Codex#5.5#B
stream: HERDR-COMMS-ACK
phase: training
task: Read and acknowledge Herdr comms guide
priority: P0
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:15:28Z
finished_at: 2026-07-04T20:15:28Z
depends_on: .deploy-control/HERDR_COMMS_GUIDE.md
blockers: none
build_result: green - Read and adopted Herdr comms guide; acknowledgement recorded.
notes: Herdr operations require HERDR_ENV=1; pane ids are not durable; use ping-opus.sh for POC reachback.
herdr-comms-ack: Codex#5.5#B @ 2026-07-04T20:15:28Z  status: ACKNOWLEDGED
---

## Adopted Rules

- Operate Herdr only when `HERDR_ENV=1`.
- Re-read pane/agent ids before use; ids are not durable.
- Use `herdr pane run <pane> "<text>"` when text must be submitted with Enter.
- `herdr agent send` and `herdr pane send-text` do not press Enter; follow with `herdr pane send-keys <pane> Enter` when using send-text.
- Reach POC/status/blockers through `.deploy-control/ping-opus.sh`.
