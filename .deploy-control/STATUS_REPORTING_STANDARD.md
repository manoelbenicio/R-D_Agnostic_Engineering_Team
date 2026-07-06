# STATUS REPORTING STANDARD (MANDATORY — non-optional)

Issued by: Principal Architect (Opus 4.8, Tech-Lead) via opus-4.8-orchestrator.
Applies to: EVERY agent in the fleet, for the whole project. No silent agents.

## 1. Check-in front-matter (MANDATORY in every `.deploy-control/<AGENT>__<STREAM>__<UTC>.md`)

Use `none` when a field is N/A. Fields:

```
agent:         # exact name, e.g. Codex#5.5#A
stream:        # e.g. RPP-CONTRACT
phase:         # e.g. F1
task:          # short description
priority:      # P0 | P1 | P2
status:        # IN_PROGRESS | BLOCKED | DONE
progress:      # 0-100, honest; DONE = 100
eta:           # e.g. 2h | 45m | done
started_at:    # UTC ISO8601
finished_at:   # UTC; MANDATORY if status == DONE
depends_on:    # [streams] | none
blockers:      # text | none; MANDATORY if status == BLOCKED
build_result:  # green|red + 1-line summary; MANDATORY if status == DONE
notes:         # short
```

## 2. ACK (MANDATORY, within 15 minutes of receipt)

Each agent creates/updates a check-in containing exactly this line, confirming receipt + compliance:

```
ack: <AgentName> @ <UTC-ISO8601>  status: ACKNOWLEDGED
```

Anyone who does not sign the ACK line = NON-COMPLIANT.

## 3. Cadence

- Update the check-in on every state change, and at minimum every 30 minutes while `IN_PROGRESS`.
- On completion: set `finished_at` + `build_result` + `progress: 100` + `status: DONE`.
- Cadence applies ONLY while `status: IN_PROGRESS`. A `DONE` agent has no periodic-update obligation — stand by.
- A NEW task/stream = a NEW check-in file (`<AGENT>__<STREAM>__<UTC>.md`). Do NOT reopen a `DONE` check-in; open a fresh one when re-tasked.

## 4. Reachback

Contact the Tech-Lead/POC first for any question/doubt/blocker/status via:
`bash /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/ping-opus.sh "[<AgentName>] <msg>"`
Do NOT use bare `herdr agent send` (it strands without Enter).
