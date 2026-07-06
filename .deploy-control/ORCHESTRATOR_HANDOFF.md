# ORCHESTRATOR HANDOFF — opus-4.8-orchestrator (Herdr fleet Tech-Lead/POC)

Operational playbook learned while managing the Rotation-Parity Polyglot fleet. Everything here is
verified against the real `herdr` binary + observed behavior. **Nothing invented.**

---

## 0. Preconditions & identity
- Only operate Herdr when `HERDR_ENV=1`. Your pane id is `$HERDR_PANE_ID` (e.g. `w3:pE`).
- Set your durable identity so agents can address you:
  `herdr agent rename "$HERDR_PANE_ID" opus-4.8-orchestrator`
- Board (source of truth) is on disk: `<clone>/.deploy-control/`. Pull from it; don't rely on messages.

## 1. Main responsibilities
1. **Coordinate** the fleet (dispatch work, keep no agent idle-with-pending).
2. **Enforce gates**: sign-in/out check-ins, disjoint file ownership, green-in-container before DONE.
3. **Validate every DONE yourself** — re-run the gate; never trust the agent's tail. Distinguish
   **empirical (real execution)** from **plan/dry-run**; only empirical counts as green.
4. **Honesty > speed**: live-gated ≠ validated. Reclassify plan-only "DONE" back to IN_PROGRESS.
5. **Never** run a real PROD deploy without the owner's explicit go + the safety pre-reqs proven.
6. Keep secrets out of logs/evidence; print only non-secret identifiers (e.g. account_id, not tokens).

## 2. Communication mechanics (THE critical lessons)
### You → agent (dispatch / instruct)
- **`herdr pane run <pane_id> "<text>"`** = sends text **AND presses Enter** → actually SUBMITS. Use this.
- **opencode & cline TUIs often DON'T accept pane-run's Enter** → after `pane run`, also send:
  `herdr pane send-keys <pane_id> Enter`
- opencode sometimes opens a model **picker** on first keystroke → send `Escape` (x2) then re-issue.

### Agent → you (reachback) — they cannot reach you with bare `agent send`
- **`herdr agent send <name> "<text>"` and `pane send-text` write literal text with NO Enter** → the
  text strands in the target's input buffer and is **never processed**. This was the #1 gotcha.
- The fix installed for the fleet: **`.deploy-control/ping-opus.sh`** — resolves your current pane id
  and uses `pane run` (text+Enter) so the message is actually delivered. Agents contact you via:
  `bash <clone>/.deploy-control/ping-opus.sh "[<AgentName>] <message>"`
- To press Enter on a pane by id (submit stranded text): `herdr pane send-keys <pane_id> Enter`
  (NOTE: `pane`/`send-keys`/`run` require a **pane_id**, NOT an agent name; only `agent *` accept names.)

### Telling agents what they can use
- Canonical docs on the board (broadcast + require ACK line in a check-in):
  - `.deploy-control/HERDR_COMMS_GUIDE.md` (commands, the Enter rule, ping-opus.sh)
  - `.deploy-control/STATUS_REPORTING_STANDARD.md` (check-in front-matter, cadence, ACK)
- ACK is pull-verified: agents append `ack: <Name> @ <UTC>  status: ACKNOWLEDGED` (or
  `herdr-comms-ack: ...`) to a check-in; you `grep` the board for it.

## 3. RE-READ ALL PANES (do this often — ids are NOT durable; agents crash/rename)
```bash
herdr pane list      # every pane: pane_id, tab_id, agent_status, agent(type), label, cwd  (JSON)
herdr agent list     # detected agents: herdr-name (from rename) + pane_id
herdr pane get <pane_id>     # one pane's full detail
herdr agent get <name>       # by durable name
```
- **IDs compact when panes close — re-read every time; never hardcode.** Parse from `pane list`.
- **Parsing gotcha:** pane ids contain a colon (`w3:pJ`). Do NOT `${x%%:*}`-split on `:` (gives `w3`).
  Use JSON parsing (python `json.load`) or split carefully.
- **Identity reconciliation** (names drift: herdr-name vs pane label vs self-id):
  - Compare `agent list` name ↔ `pane list` label. Fix drift from the orchestrator:
    `herdr agent rename <pane_id> "<Correct#Name>"`  (agent rename accepts a pane_id target)
  - Verified example: agy panes had label `Gemini#PRO#31` but I'd named one `Gemini#Flash35`;
    opencode panes had NO herdr-name (their self-rename failed) → I renamed them from here.
- **Read what an agent is doing:**
  `herdr pane read <pane_id> --source recent|visible|recent-unwrapped --lines N`
- **Wait for state:**
  `herdr wait agent-status <pane_id> --status <idle|working|blocked|done|unknown> --timeout <ms>`
  (NOTE: `herdr agent wait` supports idle|working|blocked|unknown but **NOT `done`**; use
  `herdr wait agent-status` for `done`.)

## 4. Agent crash / restart handling (codex especially)
- **Exit a codex TUI to shell:** send `/quit` (via `pane run`). Then the shell prompt returns.
- **Relaunch codex (correct cwd + isolated home):**
  `herdr pane run <pane_id> 'cd <clone> && export CODEX_HOME=$HOME/.codex-<id> && codex'`
- **CWD drift:** after auth relaunches, panes land in the wrong repo (e.g. `/mnt/c/VMs/Projects/AOP`).
  Always `cd <clone>` before relaunch; verify with `pane get … | cwd`.
- **Single shared account (verified: 1 account `d3982266`):** running multiple codex sessions
  concurrently on ONE account → `HTTP 401` and eventually **refresh-token REVOKED**. Mitigations:
  (a) run ≤1 codex actively at a time, or (b) enroll DISTINCT accounts per `CODEX_HOME`.
  Account/profile management belongs to **prodex** (`prodex profile add/import`, `prodex login`) —
  do NOT build custom login tooling.
- **Persisted CODEX_HOME per pane** (survives restart) — appended to `~/.bashrc`:
  ```bash
  case "${HERDR_PANE_ID:-}" in
    w3:pJ) export CODEX_HOME="$HOME/.codex-a" ;;  # A
    w3:pM) export CODEX_HOME="$HOME/.codex-b" ;;  # B
    w3:pK) export CODEX_HOME="$HOME/.codex-c" ;;  # C
    w3:p9) export CODEX_HOME="$HOME/.codex-d" ;;  # D
  esac
  ```

## 5. Dispatch discipline (every task)
- Before editing files, the agent creates a check-in `<AGENT>__<STREAM>__<UTC>.md` with FULL
  front-matter (agent/stream/phase/task/priority/status/progress/eta/started_at/finished_at/
  depends_on/blockers/build_result/notes). Update while IN_PROGRESS; on done set finished_at +
  build_result + progress=100. New task = NEW check-in (don't reopen a DONE one).
- Give absolute paths (repo is on a 9p mount; credentials/profiles must be ext4, mode 600).
- Disjoint file ownership; hotspots (`daemon.go`/`config.go`) = single-owner serial with explicit
  HOTSPOT LOCK lines.

## 6. Validation (never trust the tail)
- Go container gate:
  `docker run --rm -v <clone>/multica-auth-work:/src -v gomodcache:/go/pkg/mod -w /src/server \`
  `  golang:1.26-alpine sh -c 'apk add --no-cache git >/dev/null 2>&1 && go build ./... && \`
  `  go vet ./internal/... && go test ./internal/rotation/ ./internal/l2runtime/'`
  (Use a persistent `gomodcache` volume; transient proxy blips → just retry. Known pre-existing
  failure `TestValidateLocalPath` is an env/symlink issue, unrelated.)
- Publish evidence to `.deploy-control/evidence/`; link it from the check-in.

## 7. Monitoring
- You are turn-based — there is no background loop. Re-sweep each turn: `pane list` statuses +
  `grep` the board for check-in status/ACK lines + read stragglers. Re-task idle-with-pending
  immediately. If no input, take the next pending item.
- Reliable fleet→you channel = the on-disk board (+ F8's status-board), not pushed messages.

## 8. Handy one-liners
```bash
# fleet status table (colon-safe JSON parse)
herdr pane list | python3 -c 'import sys,json;[print(f"{p[\"pane_id\"]:7} {p.get(\"agent_status\",\"?\"):8} {p.get(\"agent\",\"-\"):9} {p.get(\"label\",\"-\")}") for p in json.load(sys.stdin)["result"]["panes"]]'
# ACK roster sweep
grep -rhE "(^|[^-])ack:.*ACKNOWLEDGED" .deploy-control/*.md | grep -v '<'
# gate check-in statuses
for f in .deploy-control/*RPP*.md; do echo "[$(grep -m1 -i '^status:' "$f"|sed 's/.*: *//')] $(basename "$f")"; done
```

_Authored by opus-4.8-orchestrator, 2026-07-04._
