---
name: herdr-control
description: "Control Herdr from inside a managed pane. Use when the user explicitly asks to use Herdr to manage panes, tabs, workspaces, agents, terminals, waits, plugins, worktrees, integrations, notifications, or layout. Requires HERDR_ENV=1. Do not use merely because a task could benefit from parallel work or delegation."
license: MIT
metadata:
  author: Kiro
  version: "1.0"
  source: "https://herdr.dev/docs/cli-reference/ + https://herdr.dev/docs/socket-api/"
---

# herdr-control — Operational Skill for Agents Inside Herdr

## Gate check

Before issuing any command, verify you are inside Herdr:

```bash
test "${HERDR_ENV:-}" = 1
```

If this fails, tell the user you are not running inside a Herdr-managed pane and stop. Do not inspect or control the focused session from outside.

When the check passes, the `herdr` binary in `PATH` talks to the running session over a local socket.

## Discover command syntax

The installed binary is the authority. Never invent flags. Start with:

```bash
herdr --help                 # top-level commands
herdr pane                   # pane subcommands
herdr workspace              # workspace subcommands
herdr tab                    # tab subcommands
herdr wait                   # wait subcommands
herdr agent                  # agent subcommands
herdr worktree               # worktree subcommands
herdr plugin                 # plugin subcommands
herdr integration            # integration subcommands
herdr notification           # notification subcommands
herdr terminal               # terminal subcommands
herdr server                 # server subcommands
herdr session                # session subcommands
```

Do NOT run bare `herdr` for discovery — it launches/attaches the TUI. Do NOT probe mutating commands by omitting arguments (`herdr workspace create` will execute with defaults).

## IDs and context

Public IDs: `w1` (workspace), `w1:t1` (tab), `w1:p1` (pane), `term_...` (terminal).  
IDs are opaque — never construct them from sidebar order or display numbers.

Herdr injects stable context into every managed pane:

```bash
echo "$HERDR_WORKSPACE_ID"  # e.g. w1
echo "$HERDR_TAB_ID"        # e.g. w1:t1
echo "$HERDR_PANE_ID"       # e.g. w1:p1
```

Prefer `--current` when targeting the calling pane. Omitting a target uses the UI-focused pane (which may not be yours).

---

## Workspaces

```bash
herdr workspace list
herdr workspace create [--cwd PATH] [--label TEXT] [--env KEY=VALUE] [--focus] [--no-focus]
herdr workspace get <workspace_id>
herdr workspace focus <workspace_id>
herdr workspace rename <workspace_id> <label>
herdr workspace report-metadata <workspace_id> --source ID [--token NAME=VALUE] [--clear-token NAME] [--seq N] [--ttl-ms N]
herdr workspace close <workspace_id>
```

Create without stealing focus:

```bash
herdr workspace create --cwd ~/project --label api --no-focus
```

## Worktrees (Git checkout-as-workspace)

```bash
herdr worktree list [--workspace ID | --cwd PATH] [--json]
herdr worktree create [--workspace ID | --cwd PATH] [--branch NAME] [--base REF] [--path PATH] [--label TEXT] [--focus] [--no-focus] [--json]
herdr worktree open [--workspace ID | --cwd PATH] (--path PATH | --branch NAME) [--label TEXT] [--focus] [--no-focus] [--json]
herdr worktree remove --workspace ID [--force] [--json]
```

- `create` creates a Git worktree checkout and opens it as a grouped workspace.
- If `--branch` names an existing local branch, it checks it out; otherwise creates from `--base` or HEAD.
- `workspace close` closes Herdr state only; `worktree remove` runs `git worktree remove` (never deletes the branch).
- Use `--force` when Git refuses a dirty checkout.

## Tabs

```bash
herdr tab list [--workspace <workspace_id>]
herdr tab create [--workspace <workspace_id>] [--cwd PATH] [--label TEXT] [--env KEY=VALUE] [--focus] [--no-focus]
herdr tab get <tab_id>
herdr tab focus <tab_id>
herdr tab rename <tab_id> <label>
herdr tab close <tab_id>
```

## Panes — Navigation & Layout

```bash
herdr pane list [--workspace <workspace_id>]
herdr pane current [--pane ID|--current]
herdr pane get <pane_id>
herdr pane layout [--pane ID|--current]
herdr pane process-info [--pane ID|--current]
herdr pane neighbor --direction left|right|up|down [--pane ID|--current]
herdr pane edges [--pane ID|--current]
herdr pane focus --direction left|right|up|down [--pane ID|--current]
herdr pane resize --direction left|right|up|down [--amount FLOAT] [--pane ID|--current]
herdr pane zoom [<pane_id>|--pane ID|--current] [--toggle|--on|--off]
herdr pane swap --direction left|right|up|down [--pane ID|--current]
herdr pane swap --source-pane ID --target-pane ID
herdr pane move <pane_id> --tab <tab_id> --split right|down [--target-pane ID] [--ratio FLOAT] [--focus|--no-focus]
herdr pane move <pane_id> --new-tab [--workspace ID] [--label TEXT] [--focus|--no-focus]
herdr pane move <pane_id> --new-workspace [--label TEXT] [--tab-label TEXT] [--focus|--no-focus]
```

## Panes — Split, Rename, Close

```bash
herdr pane split [<pane_id>|--pane ID|--current] --direction right|down [--ratio FLOAT] [--cwd PATH] [--env KEY=VALUE] [--focus] [--no-focus]
herdr pane rename <pane_id> <label>|--clear
herdr pane close <pane_id>
```

Split direction heuristic: check the calling pane's geometry first:

```bash
herdr pane layout --pane "$HERDR_PANE_ID"
```

Split a wide pane to the right, a narrow/tall pane down. Always use `--no-focus` for background work:

```bash
herdr pane split --current --direction right --no-focus
```

Read the new `pane_id` from the JSON response `result.pane.pane_id`.

## Panes — I/O (Read, Send, Run)

Read output:

```bash
herdr pane read <pane_id> [--source visible|recent|recent-unwrapped|detection] [--lines N]
herdr pane read <pane_id> --source visible --ansi
herdr pane read <pane_id> --source recent-unwrapped --lines 120
```

| Source | Use for |
|--------|---------|
| `visible` | Current viewport. UI feedback loops. |
| `recent` | Recent scrollback with soft wraps. |
| `recent-unwrapped` | Recent scrollback, no soft wraps. Best for logs/transcripts. |
| `detection` | Bottom-buffer snapshot for agent detection. |

Send input:

```bash
herdr pane send-text <pane_id> <text>
herdr pane send-keys <pane_id> <key> [key ...]
herdr pane run <pane_id> <command>
```

`pane run` sends text+Enter atomically. Prefer it over `send-text` + `send-keys enter`.

Key syntax: plain printable keys (`a`), special (`enter`, `tab`, `esc`, `backspace`, `left`, `right`, `up`, `down`), modifiers (`ctrl+c`, `alt+x`, `shift+tab`), function keys (`f1`), named punctuation (`minus`, `plus`, `backtick`). Legacy `C-c` accepted as `ctrl+c`.

## Panes — Agent State Reporting

Report agent state from custom hooks:

```bash
herdr pane report-agent <pane_id> \
  --source ID \
  --agent LABEL \
  --state idle|working|blocked|unknown \
  [--message TEXT] \
  [--seq N] \
  [--agent-session-id ID] \
  [--agent-session-path PATH]
```

Report display-only metadata (does NOT take over semantic state):

```bash
herdr pane report-metadata <pane_id> \
  --source ID \
  [--agent LABEL] \
  [--applies-to-source ID] \
  [--title TEXT|--clear-title] \
  [--display-agent TEXT|--clear-display-agent] \
  [--state-label STATUS=TEXT] \
  [--clear-state-labels] \
  [--token NAME=VALUE] \
  [--clear-token NAME] \
  [--seq N] \
  [--ttl-ms N]
```

STATUS: `idle`, `working`, `blocked`, `done`, `unknown`.

---

## Agents

```bash
herdr agent list
herdr agent get <target>
herdr agent read <target> [--source recent-unwrapped] [--lines N]
herdr agent explain <target> [--json]
herdr agent send <target> <text>
herdr agent rename <target> <label>
herdr agent focus <target>
herdr agent start <target>
```

Agent status semantics:

| Status | Meaning |
|--------|---------|
| `idle` | Waiting, result considered seen. |
| `done` | Finished, result not yet seen (background tab). |
| `working` | Actively processing. |
| `blocked` | Needs input. |
| `unknown` | No detected/integrated agent yet. |

`idle` and `done` are both "completed" — the difference is attention/visibility state.

## Starting an agent — full workflow

```bash
# 1. Pick split direction from layout
herdr pane layout --pane "$HERDR_PANE_ID"

# 2. Split without stealing focus
herdr pane split --current --direction right --no-focus
# → read pane_id from JSON: result.pane.pane_id

# 3. Label and launch the agent interactively
herdr pane rename <new-pane-id> "reviewer"
herdr pane run <new-pane-id> "claude"

# 4. Wait for agent ready
herdr wait agent-status <new-pane-id> --status idle --timeout 30000

# 5. Submit the task
herdr pane run <new-pane-id> "Review the current diff and report actionable findings."

# 6. Wait for completion and read results
herdr wait agent-status <new-pane-id> --status working --timeout 30000
herdr wait agent-status <new-pane-id> --status done --timeout 120000
herdr pane read <new-pane-id> --source recent-unwrapped --lines 120
```

Agent executables: `claude` (Claude Code), `codex` (Codex), `pi`, `opencode` (OpenCode), `omp` (OMP).

Do NOT pass the task as argv by default. Do NOT add non-interactive flags. Only change the normal interactive launch when the user explicitly requests it.

If the user is watching that tab, completion reports `idle` instead of `done`. Always treat either as "completed" when inspecting.

## Running ordinary commands in a sibling pane

```bash
# Split
herdr pane split --current --direction right --no-focus
# → read pane_id

# Run and wait
herdr pane run <new-pane-id> "just test"
herdr wait output <new-pane-id> --match "test result" --timeout 120000
herdr pane read <new-pane-id> --source recent-unwrapped --lines 120
```

---

## Waits

```bash
herdr wait output <pane_id> --match <text> [--source visible|recent|recent-unwrapped] [--lines N] [--timeout MS] [--regex] [--raw]
herdr wait agent-status <pane_id> --status <idle|working|blocked|done|unknown> [--timeout MS]
```

- `wait output` — for commands and servers.
- `wait agent-status` — for coding agents.
- A wait timeout exits with status `1`.
- Inspect current output BEFORE waiting for future output.

---

## Layout (Export/Apply)

Export the current tab layout as a portable BSP tree:

```bash
herdr pane layout [--pane ID|--current]
```

Raw socket methods for full layout control:

- `layout.export` — returns portable tab layout tree.
- `layout.apply` — creates a fresh tab from a declarative tree (structure, labels, cwd, env, commands; NOT live PTYs or scrollback).
- `layout.set_split_ratio` — updates an existing split ratio.

---

## Notifications

```bash
herdr notification show <title> [--body TEXT] [--position top-left|top-right|bottom-left|bottom-right] [--sound none|done|request]
```

- `--position` only affects in-app Herdr toasts.
- `--sound` defaults to `none`; `done` and `request` play sounds only when the notification is shown.

---

## Integrations

```bash
herdr integration install <name>
herdr integration uninstall <name>
herdr integration status [--outdated-only]
```

Supported: `pi`, `omp`, `claude`, `codex`, `copilot`, `devin`, `droid`, `kimi`, `opencode`, `kilo`, `hermes`, `qodercli`, `cursor`, `mastracode`.

Installing an integration gives Herdr authoritative agent state instead of screen detection.

---

## Plugins

Install and manage:

```bash
herdr plugin install <owner>/<repo>[/subdir...] [--ref REF] [--yes]
herdr plugin list [--plugin ID] [--json]
herdr plugin uninstall <plugin_id|owner/repo[/subdir...]>
herdr plugin enable <plugin_id>
herdr plugin disable <plugin_id>
```

Local development:

```bash
herdr plugin link <path> [--disabled]
herdr plugin unlink <plugin_id>
herdr plugin config-dir <plugin_id>
```

Actions:

```bash
herdr plugin action list [--plugin ID]
herdr plugin action invoke <action_id> [--plugin ID]
```

Logs:

```bash
herdr plugin log list [--plugin ID] [--limit N]
```

Managed terminal panes:

```bash
herdr plugin pane open --plugin ID --entrypoint ID [--placement overlay|popup|split|tab|zoomed] [--width SIZE] [--height SIZE] [--workspace ID] [--target-pane PANE] [--direction right|down] [--cwd PATH] [--env KEY=VALUE] [--focus|--no-focus]
herdr plugin pane focus <pane_id>
herdr plugin pane close <pane_id>
```

Plugin environment variables injected by Herdr: `HERDR_SOCKET_PATH`, `HERDR_BIN_PATH`, `HERDR_ENV`, `HERDR_PLUGIN_ID`, `HERDR_PLUGIN_ROOT`, `HERDR_PLUGIN_CONFIG_DIR`, `HERDR_PLUGIN_STATE_DIR`, `HERDR_PLUGIN_CONTEXT_JSON`, `HERDR_WORKSPACE_ID`, `HERDR_TAB_ID`, `HERDR_PANE_ID`, `HERDR_PLUGIN_ACTION_ID` (actions), `HERDR_PLUGIN_EVENT`/`HERDR_PLUGIN_EVENT_JSON` (hooks), `HERDR_PLUGIN_ENTRYPOINT_ID` (panes).

---

## Terminal (Direct Attach & Control)

```bash
herdr terminal attach <terminal_id> [--takeover]
herdr terminal session control <target> [--takeover] [--cols N] [--rows N]
herdr terminal session observe <target> [--cols N] [--rows N]
herdr terminal title set <title>
herdr terminal title clear
```

- `control` — writable live terminal stream (one controller at a time; `--takeover` replaces).
- `observe` — read-only live stream (multiple observers allowed).
- Detach from direct attach with `ctrl+b q`. Send literal `ctrl+b` with `ctrl+b ctrl+b`.

---

## Sessions

```bash
herdr session list [--json]
herdr session attach <name>
herdr session stop <name> [--json]
herdr session delete <name> [--json]
```

Use `default` to target the default session explicitly.

---

## Server

```bash
herdr server stop
herdr server reload-config
herdr server agent-manifests [--json]
herdr server update-agent-manifests [--json]
herdr server reload-agent-manifests
```

---

## Socket API (Raw Methods)

When the CLI is insufficient, use the raw socket (newline-delimited JSON):

```bash
# Socket path resolution:
# 1. explicit --session <name>
# 2. HERDR_SOCKET_PATH
# 3. HERDR_SESSION=<name>
# 4. ~/.config/herdr/herdr.sock (default)
```

Full raw method reference by area:

| Area | Methods |
|------|---------|
| Server | `ping`, `server.stop`, `server.reload_config`, `server.agent_manifests`, `server.reload_agent_manifests` |
| Notification | `notification.show` |
| Client | `client.window_title.set`, `client.window_title.clear` |
| Session | `session.snapshot` |
| Workspace | `workspace.create`, `workspace.list`, `workspace.get`, `workspace.focus`, `workspace.rename`, `workspace.move`, `workspace.report_metadata`, `workspace.close` |
| Worktree | `worktree.list`, `worktree.create`, `worktree.open`, `worktree.remove` |
| Tab | `tab.create`, `tab.list`, `tab.get`, `tab.focus`, `tab.rename`, `tab.move`, `tab.close` |
| Pane | `pane.split`, `pane.swap`, `pane.move`, `pane.zoom`, `pane.layout`, `pane.process_info`, `pane.neighbor`, `pane.edges`, `pane.focus_direction`, `pane.resize`, `pane.list`, `pane.current`, `pane.get`, `pane.rename`, `pane.send_text`, `pane.send_keys`, `pane.send_input`, `pane.read`, `pane.graphics.info`, `pane.graphics.set`, `pane.graphics.clear`, `pane.graphics.stream`, `pane.report_agent`, `pane.report_agent_session`, `pane.report_metadata`, `pane.clear_agent_authority`, `pane.release_agent`, `pane.close`, `pane.wait_for_output` |
| Popup | `popup.close` |
| Layout | `layout.export`, `layout.apply`, `layout.set_split_ratio` |
| Agent | `agent.list`, `agent.get`, `agent.read`, `agent.explain`, `agent.send`, `agent.rename`, `agent.focus`, `agent.start` |
| Events | `events.subscribe`, `events.wait` |
| Integrations | `integration.install`, `integration.uninstall` |
| Plugins | `plugin.link`, `plugin.list`, `plugin.unlink`, `plugin.enable`, `plugin.disable`, `plugin.action.list`, `plugin.action.invoke`, `plugin.log.list`, `plugin.pane.open`, `plugin.pane.focus`, `plugin.pane.close` |

Request format:

```json
{"id":"req_1","method":"ping","params":{}}
```

Response:

```json
{"id":"req_1","result":{"type":"pong"}}
```

Error:

```json
{"id":"req_1","error":{"code":"not_found","message":"pane not found"}}
```

Get a bootstrap snapshot:

```bash
herdr api snapshot   # prints live session.snapshot as JSON
herdr api schema     # protocol schema summary
herdr api schema --json  # full JSON Schema document
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `HERDR_ENV` | `1` inside Herdr-managed pane processes. |
| `HERDR_PANE_ID` | Public pane id for the running pane. |
| `HERDR_TAB_ID` | Public tab id. |
| `HERDR_WORKSPACE_ID` | Public workspace id. |
| `HERDR_SOCKET_PATH` | Low-level socket path override. |
| `HERDR_SESSION` | Select a named session for CLI commands. |
| `HERDR_CONFIG_PATH` | Override config file path. |
| `HERDR_LOG` | Log filter (e.g. `HERDR_LOG=herdr=debug`). |
| `HERDR_DISABLE_SOUND` | Disable sound even when enabled. |
| `HERDR_BIN_PATH` | Path to herdr binary (plugins). |
| `HERDR_PLUGIN_ID` | Plugin id (inside plugin commands). |
| `HERDR_PLUGIN_ROOT` | Plugin root directory. |
| `HERDR_PLUGIN_CONFIG_DIR` | Plugin config directory. |
| `HERDR_PLUGIN_STATE_DIR` | Plugin state directory. |
| `HERDR_PLUGIN_CONTEXT_JSON` | Invocation context JSON. |

---

## Safety & Coordination Rules

1. **Always** `--no-focus` for background work unless the user asks to switch context.
2. **Always** use `--current` or an explicit ID. Never rely on another client's focused pane.
3. **Parse IDs from JSON responses.** Never derive from sidebar order or examples.
4. **Inspect before waiting.** Read current output first, then wait for expected state.
5. **Never close** workspaces, tabs, panes, or sessions you did not create (unless explicitly asked).
6. **Never run** `herdr server stop` from an active session unless the user explicitly intends to stop.
7. **Never kill** the main Herdr process. Use named test sessions for experiments.
8. **Never run bare `herdr`** — it launches/attaches the TUI.
9. **Never probe** a mutating command by omitting arguments.
10. Closed IDs are not reused. A moved pane gets a new ID. Always re-read responses after mutations.
