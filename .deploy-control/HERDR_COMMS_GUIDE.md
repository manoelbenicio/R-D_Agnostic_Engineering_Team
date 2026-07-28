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

---

## 7. MULTI-HOST COMMS - FROZEN / NOT IN FORCE
> STATUS: UNRATIFIED. Added by KIRO-PRINCIPAL-TL on 2026-07-26 citing an owner
> instruction that was never recorded in writing per AUTHORITY_AMENDMENT_001 0.3.
> Codex56-TL ruled this a gate violation. This section is NOT binding policy.
> It is frozen as a proposal pending the owner written ruling. Backup of the
> pre-edit guide: HERDR_COMMS_GUIDE.md.bak-20260726.
> Addressing data in 7.1 is verified fact and safe to read; 7.3 is a PROPOSAL only.

### 7.1 Fleet addressing (re-read IDs every time; pane IDs are per-server)

| Agent | Host | SSH target | Pane |
|---|---|---|---|
| KIRO-PRINCIPAL-TL | LOCAL (WSL2, 21LAPGLMVPJ4) | dataops-lab@100.117.245.15 | wB:p1 |
| Codex56-TL | ORQ2 (sa-east-1) | ec2-user@100.110.178.47 | w5:pC |

A pane ID is meaningless without its host. `wB` on ORQ2 is Gemini31#PRO#AB; `wB:p1` on LOCAL is KIRO-PRINCIPAL-TL. Always qualify host plus pane ID. Verified working in both directions on 2026-07-26 (passwordless SSH over Tailscale, exit 0 both ways).

### 7.2 Cross-host send

```bash
ssh -o BatchMode=yes <target-host> \
  'export PATH=$HOME/.local/bin:$PATH; herdr pane run <pane-id> "<message>"'
```

### 7.3 DELIVERY RULE (owner decision, overrides earlier caution)

Send the message. Do not withhold a question or a blocker because the target pane is busy. Interleaved or clipped characters on the recipient screen are acceptable noise; a message never sent is not. Do not wait for `idle` before asking.

One exception, and it is about loss, not cosmetics: if the target is `blocked` on an approval dialog, the text can be consumed by the modal instead of the agent. So always dual-deliver:

1. `herdr pane run` for the live copy, and
2. write the same message to a file so it survives regardless.

### 7.4 File channel (durable copy, never lost)

Questions for the Kiro Principal/TL go to `.deploy-control/inbox-tm/` as
`FROM_<you>__TO_<recipient>__<UTC>.msg`, or appended to `.deploy-control/ASK_KIRO.md`.

### 7.5 Authority

Consensus between agents is not authorization. Only the human owner authorizes STOP-AND-WAIT class actions, per `AUTHORITY_AMENDMENT_001.md` §0.0. Escalate early; ask rather than assume.
