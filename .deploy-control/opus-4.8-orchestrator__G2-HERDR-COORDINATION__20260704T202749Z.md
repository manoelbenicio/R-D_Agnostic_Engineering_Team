agent: opus-4.8-orchestrator
stream: G2-HERDR-COORDINATION
phase: G2
task: Herdr coordination smoke (discover / status-wait / notification / submit-with-Enter / reachback)
priority: P1
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:27:00Z
finished_at: 2026-07-04T20:27:49Z
depends_on: none
blockers: none
build_result: >
  green - Herdr coordination primitives FUNCTIONAL. pane list discovered 11 panes;
  `wait agent-status` rc=0; `notification show` rc=0; reachback proven via ping-opus.sh
  (8 herdr-comms-acks + 15 standard-acks on disk). CLI `wait` used; full events.subscribe
  socket stream not exercised (noted). No product code, no deploy.
evidence: .deploy-control/evidence/g2-herdr-coordination-smoke-20260704T202749Z.md
notes: >
  G2 acceptance gate (Herdr coordination operational) owned by orchestrator per Tech-Lead
  assignment. Live pane.agent_status_changed socket subscription available via socket API if required.
