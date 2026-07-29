---
name: herdr-fleet
description: "Launch, inspect, route, and stop the Agent Brain four-Codex fleet inside Herdr panes. Use only when running inside Herdr (HERDR_ENV=1) and the user explicitly asks to manage Codex 1-4 agents or check-in/out against the GSD v3 AGENT_LEDGER. Delegation-only guardrails for the TL."
---

# herdr-fleet — four-Codex fleet control inside Herdr

This skill operates the **four Codex agent streams** of the Agent Brain program inside
Herdr (a terminal multiplexer + runtime for coding agents). It is the execution surface
for the four-stream topology defined in
`C:\VMs\Projects\RD_Agnostic_Engineering_Team\.planning\agent-brain-v3\` (see
`AGENT_LEDGER.md`, `FILE_OWNERSHIP.md`, `ROADMAP.md`).

## Preconditions (fail closed)

1. Herdr runtime gate — run first and stop if it fails:

   ```bash
   test "${HERDR_ENV:-}" = 1
   ```
   If it fails: tell the user you are not running inside Herdr and stop. Do not control
   the session from outside Herdr.

2. Authorization gate — implementation/Wave activity is blocked until
   `OMNIROUTE_ARCHITECT_RESPONSE.md` §7.1 = `AUTORIZADO`. Fleet *launch for real work*
   is only allowed after §7.1 = AUTORIZADO and G1 freeze is complete. Until then this
   skill may still **inspect** the live fleet (read-only) and record check-in/out, but
   must NOT dispatch production task work. If unsure whether work is authorized, stop
   and surface the question rather than starting an agent on a real task.

3. Delegation-only — the operator of this skill is the TL (Claude/GLM-5.2). The TL
   plans, assigns, and validates; the TL does NOT write product code and does NOT
   alter production. The four Codex agents execute.

## Learn the actual CLI first

The installed `herdr` binary is the authority. Do not invent subcommands or flags.

```bash
herdr --help
herdr pane
herdr workspace
herdr tab
herdr wait
```
- Do NOT run bare `herdr` (launches/attaches the TUI).
- Do NOT omit args on a mutating nested command (e.g. `herdr workspace create`) to
  "discover" it — it will execute. Use the group output above.
- Most control commands print JSON. Parse IDs/state from JSON, never predict them.

## Fleet topology → Herdr panes

Map the four streams to panes (labels below). Default to the CURRENT workspace/tab and
current cwd unless the user requests a different topology.

| Stream | Agent executable | Pane label | Sole hot files (must not collide) |
|---|---|---|---|
| Codex 1 — lead integrator / Brain core | `codex` | `codex1-brain` | daemon.go, config.go, health.go, cmd_daemon.go, go.mod, execenv(execenv.go+codex_home.go), models.go |
| Codex 2 — OmniRoute gateway | `codex` | `codex2-gateway` | gateway/**, protocol fixtures, route-policy types |
| Codex 3 — runtime/CLI security | `codex` | `codex3-runtime` | runtimeenv/**, adapters claude/codex/kimi/nim/antigravity, sanitizer |
| Codex 4 — ops/paridade/evidence | `codex` | `codex4-ops` | deploy/**, observability, evidence harness, runbooks |

> All four agents can be the same executable (`codex`); they are differentiated by their
> assigned task stream and locked files, not by binary choice. Other supported
> executables: `claude`, `pi`, `opencode`, `omp` (use `herdr integration status` to check
> which are integrated for authoritative state).

## Start an agent stream (interactively)

1. Inspect the calling pane rectangle to pick split direction (wide → right, narrow/tall → down):

   ```bash
   herdr pane layout --pane "$HERDR_PANE_ID"
   herdr pane split --current --direction right --no-focus
   ```
   Replace `right` with `down` when the layout calls for it. Avoid repeated same-direction
   splits that create unusably narrow panes. Keep the TL's focus on the calling pane
   (`--no-focus`).

2. Read `result.pane.pane_id` from the JSON. Label it, then start the agent with only its
   normal executable (interactive TUI). Do NOT pass the task as an argv prompt and do NOT
   add non-interactive flags unless the user explicitly requests a different launch mode.

   ```bash
   herdr pane rename <pane-id> "codex2-gateway"
   herdr pane run <pane-id> "codex"
   ```

3. Wait for the agent to reach `idle` before submitting a task:

   ```bash
   herdr wait agent-status <pane-id> --status idle --timeout 30000
   ```
   A `blocked` agent needs input; an `unknown` pane may not yet be a detected/integrated
   agent.

## Inspect the fleet (read-only, always allowed)

```bash
herdr workspace list
herdr pane list --workspace "$HERDR_WORKSPACE_ID"
herdr pane current --current
herdr pane get <pane-id>          # shows agent, agent_status, session metadata
herdr agent list                   # if available
herdr agent explain <target> --json   # why detector classified the pane that way
```
- `idle` = waiting and result seen; `done` = finished and result NOT seen. Treat both as
  "completed" when inspecting `pane get`.
- Closed pane/tab IDs are never reused and never retarget; re-read IDs after mutations.

## Stop / close a stream (cautious)

- Do NOT close workspaces, tabs, panes, or sessions you did not create, unless the user
  explicitly asks.
- Never run `herdr server stop` from an active session unless the user explicitly intends
  to stop the server and all pane processes.
- Never kill the main Herdr process. Use named test sessions for isolated experiments.
- To stop a single agent stream: prefer letting the agent complete, or close only the pane
  you created, after confirming the owner's check-out is recorded in AGENT_LEDGER.

## Check-in / out against AGENT_LEDGER (binding)

Every fleet action that begins or ends work on a stream MUST update
`.planning/agent-brain-v3/AGENT_LEDGER.md` (the GSD check-in/out ledger), per
`EVIDENCE_CONTRACT.md` (Rule 2) and legacy Golden Rule 1. Track per stream:

- agent, stream, task-ID (must reference a PLAN.md task-ID), start/end UTC,
  status (IN_PROGRESS/DONE/BLOCKED), progress %, `files_locked` (disjoint — see
  FILE_OWNERSHIP; two agents must not lock the same hotspot), evidence ID, notes.

Hard rules:
- No two agents edit the same hotspot concurrently. Codex 1 alone owns
  daemon.go/config.go/health.go/cmd_daemon.go/go.mod. Agents 2-4 create new packages and
  must NOT wire into the central daemon (Codex 1 wires in G3).
- A task reaches an agent only if it is a task-ID in a PLAN.md on disk.
- No secrets: never print/copy/log/screenshot the OmniRoute key, provider creds, cookies,
  prompts, repo content, or tool payloads.

## What this skill must NOT do

- Start production task work before §7.1 = AUTORIZADO and G1 freeze complete.
- Edit product code (the TL is delegation-only).
- Add non-interactive flags or pass the task as an argv prompt by default.
- Derive pane IDs from sidebar order or from examples — parse them from JSON.
- Touch secrets. No file under `multica-auth-work/prodex-sidecar/` or secret refs is to
  be read/printed.
- Run `herdr server stop` or kill the main process unprompted.
- Reset, discard, stash, or let multiple agents edit the preserved
  `persist-prodex-runtime-integration` baseline. PD-01 assigns its hotspots exclusively to
  Codex1 after the G1 lock; all other streams remain read-only there.

## Reference

- Herdr concepts / CLI: https://herdr.dev/docs/cli-reference/
- Agent Brain governance: `.planning/agent-brain-v3/` (AGENT_LEDGER, FILE_OWNERSHIP,
  ROADMAP, EVIDENCE_CONTRACT).
- The authoritative agent-facing Herdr control file ships inside a Herdr pane; this skill
  is the Agent-Brain-governed wrapper around those commands.
