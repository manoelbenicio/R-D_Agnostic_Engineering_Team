---
# Check-in: GLM#52#CLINE#A — Herdr comms guide adoption + ACK
# Created per HERDR_COMMS_GUIDE.md §6 (MANDATORY herdr-comms-ack) +
# STATUS_REPORTING_STANDARD.md front-matter. This is a SEPARATE check-in from
# the frozen DONE ownership-audit check-in (GLM-52-CLINE-A__OWNERSHIP-AUDIT__...),
# which is NOT modified, per Tech-Lead standing order (DONE stays frozen; new
# directives get a new check-in). No product code or deploy touched.

agent: GLM#52#CLINE#A
stream: HERDR-COMMS
phase: COMMS
task: Read + adopt .deploy-control/HERDR_COMMS_GUIDE.md; record mandatory herdr-comms-ack line
priority: P1
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:16:01Z
finished_at: 2026-07-04T20:16:01Z
depends_on: none
blockers: none
build_result: green | HERDR_COMMS_GUIDE.md read and adopted. Key rules internalized: operate Herdr ONLY if HERDR_ENV=1; pane ids NOT durable (re-read via herdr pane list each time, never hardcode); CRITICAL — 'herdr pane run <pane> "<text>"' submits text+ENTER (processed), whereas 'herdr agent send'/'herdr pane send-text' write text WITHOUT Enter (stuck/unprocessed) — to submit after send-text use 'herdr pane send-keys <pane> Enter'; reach POC only via ping-opus.sh (which resolves pane + submits with Enter). No product code or deploy touched.
notes:
  - ACK line recorded below per guide §6.
  - Comms protocol reaffirmed: ALL reachback to opus-4.8-orchestrator via bash .deploy-control/ping-opus.sh "[GLM#52#CLINE#A] <msg>" only; never bare 'herdr agent send'.
  - Ownership-audit check-in remains frozen DONE/progress=100 (not modified by this ACK).
---

herdr-comms-ack: GLM#52#CLINE#A @ 2026-07-04T20:16:01Z  status: ACKNOWLEDGED
