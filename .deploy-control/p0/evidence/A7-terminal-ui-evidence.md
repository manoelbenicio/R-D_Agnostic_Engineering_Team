# A7-EVIDENCE — Terminal UI Delivery/Persistence Evidence

agent: Agy-P0-A7
lane: A7-EVIDENCE
task: P0-TERMINAL-UI-EVIDENCE
pane: wB:p1
timestamp: 2026-07-21T22:51:00Z
status: BLOCKED

## Preflight

| Tool    | Version  | Verified                                          |
|---------|----------|---------------------------------------------------|
| git     | 2.50.1   | `git --version` → `git version 2.50.1`            |
| python3 | 3.9.25   | `python3 --version` → `Python 3.9.25`             |
| rg      | 15.2.0   | `rg --version` → `ripgrep 15.2.0 (rev e89fff89ac)`|
| Node    | v22.23.1 | `node --version` → `v22.23.1`                     |
| pnpm    | 10.28.2  | Corepack download confirmed; `pnpm --version` → `10.28.2` |

Check-in: `.deploy-control/p0/checkins/Agy-P0-A7__P0-TERMINAL-UI-EVIDENCE__20260721T225107Z.json`

## Purpose

Capture real Kanban→terminal UI delivery and persistence evidence from the
authorized integrated route runs (GLM-5.2, Kimi-K2.7, Opus48). This lane
does NOT initiate or duplicate any live run. It observes and records the
frontend/UI leg of evidence from runs authorized and executed by their
respective owners.

## Concrete dependency

The frontend components that render terminal results are confirmed present and
wired to real API endpoints (verified in predecessor task P0-PROD-INTEGRITY-FE):

- `packages/views/issues/components/execution-log-section.tsx` — streams live
  stdout/stderr via WebSocket and `GET /api/tasks/{taskId}/logs`
- `packages/views/issues/components/issue-detail.tsx` — renders task status,
  result persistence, and terminal state
- `packages/views/autopilots/components/autopilot-detail-page.tsx` — displays
  run status, timestamps, failure reasons from real server data
- `apps/mobile/app/(app)/[workspace]/issue/[id]/runs.tsx` — agent run sheet
  with active/past tasks from `GET /api/issues/{id}/active-tasks` and
  `GET /api/issues/{id}/tasks`

These components use real HTTP/WS endpoints exclusively. No synthetic data or
fake-success paths exist (proven in `A7-production-integrity-frontend.md`).

**However**: no integrated build with the pending route changes (GLM-5.2,
Kimi-K2.7, Opus48) has been produced yet, and no authorized live run has been
executed. Without a real run, there is no terminal result to observe in the UI,
and therefore no evidence to capture.

## BLOCKED

**Blocker**: Integrated build and authorized GLM/Opus48 run evidence do not
exist yet.

**Owner**: W1 (integrator) + GLM/Opus48 live-run owners + Principal
Orchestrator.

**Next action**: When an authorized integrated run occurs, A7-EVIDENCE captures
the real Kanban-to-terminal UI delivery/persistence evidence from that same run:
1. Confirm the terminal result is persisted and rendered in the issue detail UI
2. Confirm execution logs stream correctly via WebSocket
3. Confirm task status transitions display accurately (queued → running → completed/failed)
4. Confirm cancel/cleanup is reflected in the UI if exercised
5. Record screenshots/logs/artifact paths as evidence

No source edits, broad QA, or synthetic execution will be performed.

## Predecessor evidence

- `A7-production-integrity-frontend.md` — `NO_REACHABLE_RESIDUAL` verdict
- Check-in: `.deploy-control/p0/checkins/Agy-P0-A7__P0-PROD-INTEGRITY-FE__20260721T224233Z.json`

## Sequencing note

The predecessor task (P0-PROD-INTEGRITY-FE) launched reads and subagents
concurrently with its check-in command rather than strictly waiting for disk
confirmation first. The check-in did execute with pane-id `wB:p1` (via shell
default) and completed before any writes, but the ordering was not sequentially
strict as required by protocol. The audit findings themselves are valid — all
scans ran against the real codebase with confirmed tools. This successor task
follows strict sequencing: check-in confirmed on disk before any work.
