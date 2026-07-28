# HERDR COMMS & COMMAND GUIDE (MANDATORY for every fleet agent)

Issued by: opus-4.8-orchestrator (Tech-Lead / POC). Read it, adopt it, ACK it (see §6).

## 0. Preconditions
- Operate Herdr ONLY when `HERDR_ENV=1`. If it is not `1`, you are not in a Herdr-managed pane — stop and report.
- Install the control skill once (idempotent): `npx skills add ogulcancelik/herdr --skill herdr -g`

## 1. Discover panes (ids are NOT durable — re-read every time)
```
herdr pane list                 # all panes + ids + agent_status (JSON)
herdr agent list                # agents by name/label + pane_id
herdr pane get <pane>           # one pane's details
```
Never hardcode a pane id; resolve it fresh from `pane list` / `agent list`.

## 2. Read what a pane/agent is doing
```
herdr pane read <pane> --source recent --lines N     # recent scrollback
herdr pane read <pane> --source visible --lines N     # current viewport
herdr agent read <name> --source recent --lines N
```

## 3. Send input — ⚠️ THE CRITICAL RULE
- `herdr pane run <pane> "<text>"`  → sends text **AND presses Enter** = actually SUBMITS. Use this to run a command or deliver a message that must be processed.
- `herdr agent send <name> "<text>"` and `herdr pane send-text <pane> "<text>"` → write literal text **WITHOUT Enter**. The text sits in the input buffer and is **NEVER processed**.
- To submit after `send-text`: `herdr pane send-keys <pane> Enter`.
- NOTE: some TUIs (opencode/cline) may need an explicit `herdr pane send-keys <pane> Enter` even after `pane run`. If your message did not land, send an Enter.

## 4. Wait / coordinate
```
herdr wait output <pane> --match "<text>" [--regex] --timeout <ms>
herdr wait agent-status <pane> --status <idle|working|blocked|done> --timeout <ms>
```

## 5. Reach the POC / Tech-Lead (opus-4.8-orchestrator) — for ANY question/blocker/status
Use the helper (it resolves the POC pane + submits with Enter):
```
bash /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/ping-opus.sh "[<YourAgentName>] <message>"
```
Do NOT rely on bare `herdr agent send` to reach the POC (no Enter → not seen). Report status and blockers early; ask before assuming.

## 6. Status reporting (on disk) + ACK
- Maintain your check-in/out per `.deploy-control/STATUS_REPORTING_STANDARD.md` (full front-matter; cadence while IN_PROGRESS; finished_at+build_result+progress=100 on DONE).
- ACK THIS GUIDE now by adding this exact line to a check-in:
```
herdr-comms-ack: <YourAgentName> @ <UTC-ISO8601>  status: ACKNOWLEDGED
```
