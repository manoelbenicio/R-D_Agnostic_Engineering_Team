# P0 Fleet control protocol

updated: 2026-07-21
owner: Principal Orchestrator
supervisor: Opus48-Kiro
cadence_seconds: 600
state_root: `.deploy-control/p0/`

## Purpose

Provide a single namespaced source of operational truth for the Main Brain P0 without replaying the historical RPP board. The live process state comes from local Herdr; assignment state and evidence come from this directory.

This protocol specializes `.deploy-control/STATUS_REPORTING_STANDARD.md` for P0. Its 10-minute cadence overrides the historical 30-minute cadence only for active P0 assignments.

## State semantics

- `UNASSIGNED`: no obligation to work or heartbeat; an idle pane is healthy standby.
- `IN_PROGRESS`: agent must be Herdr `working` and write a heartbeat at least every 10 minutes.
- `BLOCKED`: agent must record the concrete blocker, owner and required next action; no speculative work.
- `DONE`: checkout contains evidence and focused validation; no heartbeat obligation.
- `FAILED`: checkout records the failure and evidence; Principal decides reassignment.

The policy applies to assigned agents, not every physical pane. Keeping an unassigned pane idle is required when no disjoint useful work exists; manufacturing work to keep it busy is prohibited.

## Disk artifacts

- `control.json`: authorization, supervisor and assignment registry.
- `checkins/*.json`: one immutable-identity record per assignment, updated through state transitions.
- `events.jsonl`: append-only transition journal.
- `monitor.jsonl`: append-only 10-minute fleet snapshots.
- `kiro-audit.jsonl`: bounded independent Kiro findings and escalations.
- `handoffs/`: lane-to-W1/Principal implementation handoffs.
- `evidence/`: namespaced operational evidence; authoritative technical evidence remains under `.planning/agent-brain-v3/evidence/` when accepted.

Do not write secrets, credential values, prompt bodies, provider payloads, account identities or user content into any artifact.

## Required commands

Resolve current pane identity first:

```bash
herdr pane current --current
```

Check in before any edit:

```bash
python3 scripts/orchestration/p0_control.py check-in \
  --agent '<current pane label>' \
  --pane-id "$HERDR_PANE_ID" \
  --lane A1 \
  --task 5.6 \
  --activity 'Cline providers.json shared contract' \
  --files multica-auth-work/server/internal/daemon/runtimeenv/cline.go \
          multica-auth-work/server/internal/daemon/runtimeenv/cline_test.go
```

Heartbeat on every material state change and no later than 10 minutes:

```bash
python3 scripts/orchestration/p0_control.py heartbeat \
  --agent '<current pane label>' --task 5.6 --progress 35 \
  --activity 'materialization test added; running focused package tests'
```

Block immediately rather than guessing:

```bash
python3 scripts/orchestration/p0_control.py block \
  --agent '<current pane label>' --task 5.6 \
  --blocker 'Exact GLM RouteModel absent; OmniRoute registry owner must publish it'
```

Check out only with evidence and focused validation:

```bash
python3 scripts/orchestration/p0_control.py check-out \
  --agent '<current pane label>' --task 5.6 \
  --evidence .planning/agent-brain-v3/evidence/<real-artifact>.md \
  --validation 'go test ./internal/daemon/runtimeenv: PASS' \
  --summary 'Shared Cline contract complete; live run not executed'
```

Supervisor sweep:

```bash
python3 scripts/orchestration/p0_control.py monitor --once
```

## Monitor decisions

A sweep is RED when any active assignment has overlapping locks, missing check-in, heartbeat older than 15 minutes, missing Herdr pane, or `IN_PROGRESS` while Herdr is not `working`. It is AMBER for a recorded blocker awaiting an external owner. It is GREEN when all active assignments have current locks/heartbeats and the expected Herdr state.

Kiro reads each snapshot produced by the dedicated `P0-10m-Monitor` pane and sends facts only to the Principal Orchestrator. If that snapshot is older than 11 minutes, Kiro performs one fallback `monitor --once` and escalates the monitor failure. It does not message lanes directly, invent assignments, perform broad review, or launch live acceptance. The Principal independently samples material claims and owns all accept/reject/reassign decisions.

## Live-run token

A live execution is allowed only when `control.json` contains a route-family assignment with `live_run.authorized=true`, exact RouteModel, integrated commit/build/config provenance and no prior accepted run for the same family/build/scenario. The agent records the run ID before launch and the terminal evidence after completion. `5.x`, `8.1` and `8.2` reuse that same run.

No token is issued for Antigravity when A5 proves equivalence. No Main Brain token is ever issued for OmniRoute tasks `8.5–8.7`.

## Escalation and final authority

Kiro is the right-hand supervisor, not the final authority. The Principal Orchestrator remains accountable, independently verifies high-impact claims, resolves ownership, authorizes live runs and edits authoritative OpenSpec/GSD. No agent commits or pushes unless the user explicitly requests it.


## Mandatory FLEET_SATURATED gate

Before any Principal or Opus48-Kiro product-code action, the monitor/auditor must prove all of:

1. every P0 task and supporting lane has an explicit owner in `control.json`;
2. every detected eligible worker agent has a real, disjoint assignment and check-in;
3. each assigned worker is Herdr `working`, or `blocked` with a concrete blocker and owner;
4. each assignment recorded a lane-specific tool preflight;
5. active file locks have zero intersection;
6. W1 integration and live-run serialization remain intact.

An eligible idle/unknown worker makes the gate RED. A pane with no agent process is inventory, not an
eligible worker. Busywork, duplicate QA/review, broad regression and repeated live runs never count.
Principal and Opus48-Kiro are management/supervision-only and do not write product code while any
real task remains assignable to a worker, even after the gate first turns GREEN.
