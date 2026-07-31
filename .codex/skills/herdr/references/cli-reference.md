# Herdr 0.7.5 CLI command map

This reference records the command surface observed from the installed `herdr
0.7.5` binary. Treat the installed binary as authoritative and rerun `herdr
--help` plus the relevant command group when the version changes.

Do not run bare `herdr` for discovery because it launches or attaches the TUI.
Do not probe a mutating nested command without `--help`.

## Top level

```text
herdr
herdr --session <name>
herdr --remote <ssh-target> [--session <name>]
herdr --no-session
herdr --remote-keybindings local|server
herdr --handoff
herdr --default-config
herdr --version
herdr status [server|client] [--json]
herdr update [--handoff]
herdr completion bash|elvish|fish|powershell|zsh
herdr completions bash|elvish|fish|powershell|zsh
herdr channel show
herdr channel set stable|preview
```

Bare `herdr` launches or attaches. Never launch it from an existing managed
pane. `--no-session` is the monolithic escape hatch. `completions` is an alias
for `completion`; both print the completion script to stdout.

## Server

```text
herdr server
herdr server stop
herdr server reload-config
herdr server agent-manifests [--json]
herdr server update-agent-manifests [--json]
herdr server reload-agent-manifests
```

`server stop` and manifest updates change live state. Require explicit user
intent. Never stop the server from an active session unless the user intends to
stop its pane processes.

## Agents

```text
herdr agent list
herdr agent get <target>
herdr agent read <target> [--source visible|recent|recent-unwrapped|detection] [--lines N] [--format text|ansi] [--ansi]
herdr agent send-keys <target> <key> [key ...]
herdr agent prompt <target> <text> [--wait] [--until STATUS]... [--timeout MS]
herdr agent rename <target> <name>|--clear
herdr agent focus <target>
herdr agent wait <target> [--until STATUS]... [--timeout MS]
herdr agent attach <target> [--takeover]
herdr agent start <name> --kind KIND --pane ID [--timeout MS] [-- <agent-args...>]
herdr agent explain <target> [--json|--format text|json] [--verbose]
herdr agent explain --file PATH --agent LABEL [--json|--format text|json] [--verbose]
```

Targets accept a unique live agent name or a pane ID that currently hosts an
agent. They do not accept terminal IDs or bare kind labels.

Installed kinds:

```text
pi claude codex gemini cursor devin agy cline omp mastracode opencode
copilot kimi kiro droid amp grok hermes kilo qodercli maki
```

`agent start` requires an existing shell pane at its interactive prompt. It does
not create layout. Its default startup timeout is 30000 ms and maximum is
300000 ms.

For `prompt --wait` and standalone `wait`, the default settled states are
`idle`, `done`, or `blocked`. Valid explicit states are `idle`, `working`,
`blocked`, `done`, and `unknown`.

## Panes

Inspection and geometry:

```text
herdr pane list [--workspace ID]
herdr pane current [--pane ID|--current]
herdr pane get <pane_id>
herdr pane layout [--pane ID|--current]
herdr pane process-info [--pane ID|--current]
herdr pane neighbor --direction left|right|up|down [--pane ID|--current]
herdr pane edges [--pane ID|--current]
herdr pane focus --direction left|right|up|down [--pane ID|--current]
herdr pane resize --direction left|right|up|down [--amount FLOAT] [--pane ID|--current]
herdr pane zoom [<pane_id>|--pane ID|--current] [--toggle|--on|--off]
```

Layout mutation:

```text
herdr pane rename <pane_id> <label>|--clear
herdr pane split [<pane_id>|--pane ID|--current] --direction right|down [--ratio FLOAT] [--cwd PATH] [--env KEY=VALUE] [--focus|--no-focus]
herdr pane swap --direction left|right|up|down [--pane ID|--current]
herdr pane swap --source-pane ID --target-pane ID
herdr pane move <pane_id> --tab <tab_id> --split right|down [--target-pane ID] [--ratio FLOAT] [--focus|--no-focus]
herdr pane move <pane_id> --new-tab [--workspace ID] [--label TEXT] [--focus|--no-focus]
herdr pane move <pane_id> --new-workspace [--label TEXT] [--tab-label TEXT] [--focus|--no-focus]
herdr pane close <pane_id>
```

Read, input, waits, and integration reporting:

```text
herdr pane read <pane_id> [--source visible|recent|recent-unwrapped|detection] [--lines N] [--format text|ansi] [--ansi] [--raw]
herdr pane send-text <pane_id> <text>
herdr pane send-keys <pane_id> <key> [key ...]
herdr pane run <pane_id> <command>
herdr pane wait-output <pane_id> (--match TEXT|--regex PATTERN) [--source visible|recent|recent-unwrapped] [--lines N] [--timeout MS] [--raw]
herdr pane report-agent <pane_id> --source ID --agent LABEL --state idle|working|blocked|unknown [--message TEXT] [--seq N] [--agent-session-id ID] [--agent-session-path PATH]
herdr pane report-agent-session <pane_id> --source ID --agent LABEL [--seq N] [--agent-session-id ID] [--agent-session-path PATH] [--session-start-source SOURCE]
herdr pane release-agent <pane_id> --source ID --agent LABEL [--seq N]
herdr pane report-metadata <pane_id> --source ID [--agent LABEL] [--applies-to-source ID] [--title TEXT|--clear-title] [--display-agent TEXT|--clear-display-agent] [--state-label STATUS=TEXT] [--clear-state-labels] [--token NAME=VALUE] [--clear-token NAME] [--seq N] [--ttl-ms N]
```

`pane wait-output` searches the existing snapshot before polling. `--match` is
literal; `--regex` is a Rust regular expression. Without `--timeout`, it waits
indefinitely. For reads, `detection` is the plain-text bottom-buffer snapshot
used by agent detection. `recent-unwrapped` is usually best for logs.

## Workspaces

```text
herdr workspace list
herdr workspace create [--cwd PATH] [--label TEXT] [--env KEY=VALUE] [--focus|--no-focus]
herdr workspace get <workspace_id>
herdr workspace focus <workspace_id>
herdr workspace rename <workspace_id> <label>
herdr workspace report-metadata <workspace_id> --source ID [--token NAME=VALUE] [--clear-token NAME] [--seq N] [--ttl-ms N]
herdr workspace close <workspace_id>
```

`workspace create` is valid with defaults and mutates layout. Do not use it as
a help probe.

## Tabs

```text
herdr tab list [--workspace ID]
herdr tab create [--workspace ID] [--cwd PATH] [--label TEXT] [--env KEY=VALUE] [--focus|--no-focus]
herdr tab get <tab_id>
herdr tab focus <tab_id>
herdr tab rename <tab_id> <label>
herdr tab close <tab_id>
```

## Worktrees

```text
herdr worktree list [--workspace ID|--cwd PATH] [--json]
herdr worktree create [--workspace ID|--cwd PATH] [--branch NAME] [--base REF] [--path PATH] [--label TEXT] [--focus|--no-focus] [--json]
herdr worktree open [--workspace ID|--cwd PATH] (--path PATH|--branch NAME) [--label TEXT] [--focus|--no-focus] [--json]
herdr worktree remove --workspace ID [--force] [--json]
```

`worktree remove` deletes a checkout through Git and is destructive. Resolve
the exact target and inspect dirty state before using it.

## Direct terminal streams

```text
herdr terminal attach <terminal_id> [--takeover]
herdr terminal session control <target> [--takeover] [--cols N] [--rows N]
herdr terminal session observe <target> [--cols N] [--rows N]
herdr terminal title set <title>
herdr terminal title clear
```

Detach from direct attach with `ctrl+b q`; send a literal `ctrl+b` with
`ctrl+b ctrl+b`. `--takeover` replaces another controller, so use it only with
explicit need.

## Notifications

```text
herdr notification show <title> [--body TEXT] [--position top-left|top-right|bottom-left|bottom-right] [--sound none|done|request]
```

## Integrations

```text
herdr integration install pi|omp|claude|codex|copilot|devin|droid|kimi|opencode|kilo|hermes|qodercli|cursor|mastracode
herdr integration uninstall pi|omp|claude|codex|copilot|devin|droid|kimi|opencode|kilo|hermes|qodercli|cursor|mastracode
herdr integration status [--outdated-only]
```

Install and uninstall change agent integration state.

## Sessions

```text
herdr session list [--json]
herdr session attach <name>
herdr session stop <name> [--json]
herdr session delete <name> [--json]
```

Use `default` as the name to target the default session for stop. Stop and
delete require explicit user intent.

## Plugins

```text
herdr plugin install <owner>/<repo>[/subdir...] [--ref REF] [--yes]
herdr plugin uninstall <plugin_id|owner/repo[/subdir...]>
herdr plugin link <path> [--disabled]
herdr plugin unlink <plugin_id>
herdr plugin enable <plugin_id>
herdr plugin disable <plugin_id>
herdr plugin list [--plugin ID] [--json]
herdr plugin config-dir <plugin_id>
herdr plugin action list [--plugin ID]
herdr plugin action invoke <action_id> [--plugin ID]
herdr plugin log list [--plugin ID] [--limit N]
herdr plugin pane open --plugin ID --entrypoint ID [--placement overlay|popup|split|tab|zoomed] [--width SIZE] [--height SIZE] [--workspace ID] [--target-pane ID] [--direction right|down] [--cwd PATH] [--env KEY=VALUE] [--focus|--no-focus]
herdr plugin pane focus <pane_id>
herdr plugin pane close <pane_id>
```

GitHub installs use `owner/repo[/subdir...]`. Local development uses `link`;
`unlink` leaves source files in place, while uninstalling a managed GitHub
plugin removes its managed checkout. Plugins require `min_herdr_version`.
Inspect link/list warnings and effective platform support before invoking an
action or opening a pane.

Plugin actions and panes are manifest-declared. Popup terminals are modal and
do not receive a pane ID or participate in pane/agent APIs. Use
`plugin config-dir` for a stable user-editable configuration location; plugin
state and migrations remain plugin-owned.

## API, configuration, and update channel

```text
herdr api snapshot
herdr api schema [--json|--output PATH]
herdr config check
herdr config reset-keys
herdr channel show
herdr channel set stable|preview
```

`config reset-keys` backs up `config.toml` and removes custom keybindings.
`channel set` changes update configuration.

## Environment and exits

Important variables include `HERDR_CONFIG_PATH`, `HERDR_SESSION`,
`HERDR_SOCKET_PATH`, `HERDR_ENV`, `HERDR_WORKSPACE_ID`, `HERDR_TAB_ID`,
`HERDR_PANE_ID`, `HERDR_LOG`, and `HERDR_DISABLE_SOUND`. Treat Herdr-managed
context variables as authoritative.

Most successful control commands return JSON. Server errors are JSON on stderr
with exit status 1. CLI syntax errors exit with status 2. Treat IDs as opaque
and refresh records after every mutation.
