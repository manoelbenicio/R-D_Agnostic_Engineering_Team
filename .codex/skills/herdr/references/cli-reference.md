# Herdr CLI command map

Read this file when the requested operation is not covered by the parent
skill's core pane workflows. The installed binary is authoritative: run
`herdr --help`, then the relevant non-mutating command group before acting. Use
the [canonical CLI reference](https://herdr.dev/docs/cli-reference/) for detail.

Do not run bare `herdr` for discovery because it launches or attaches the TUI.
Do not probe a mutating nested command by omitting arguments.

## Launch, update, status, and schema

```text
herdr --session <name>
herdr --remote <host> [--remote-keybindings local|server] [--handoff]
herdr --no-session
herdr --default-config
herdr update [--handoff]
herdr channel show
herdr channel set stable|preview
herdr --version
herdr status [server|client]
herdr api schema [--json|--output PATH]
herdr api snapshot
```

Bare `herdr` launches or attaches. Never launch it from an existing managed
pane. `--no-session` is a single-process escape hatch. Schema and snapshot are
the preferred bootstrap surfaces for protocol tooling.

Generate shell completions with `herdr completion <shell>`; `completions` is an
alias. Supported shells include Bash, Zsh, Fish, PowerShell, and Elvish. Do not
edit shell startup files unless the user asks.

## Server, notifications, and sessions

```text
herdr server
herdr server stop
herdr server reload-config
herdr server agent-manifests [--json]
herdr server update-agent-manifests [--json]
herdr server reload-agent-manifests
herdr notification show <title> [--body TEXT] [--position POSITION] [--sound SOUND]
herdr session list [--json]
herdr session attach <name>
herdr session stop <name> [--json]
herdr session delete <name> [--json]
```

`server stop`, session stop/delete, and update operations require explicit user
intent. Use `default` when explicitly targeting the default session.

## Workspaces, worktrees, and tabs

```text
herdr workspace list|get|focus|rename|close ...
herdr workspace create [--cwd PATH] [--label TEXT] [--env KEY=VALUE] [--focus|--no-focus]
herdr workspace report-metadata <id> --source ID [metadata options]

herdr worktree list [--workspace ID|--cwd PATH] [--json]
herdr worktree create [--workspace ID|--cwd PATH] [--branch NAME] [--base REF] [--path PATH] [--label TEXT] [--focus|--no-focus] [--json]
herdr worktree open [--workspace ID|--cwd PATH] (--path PATH|--branch NAME) [--label TEXT] [--focus|--no-focus] [--json]
herdr worktree remove --workspace ID [--force] [--json]

herdr tab list|create|get|focus|rename|close ...
```

Prefer `--no-focus` for background creation. `workspace close` removes Herdr
state only. `worktree remove` deletes the checkout with `git worktree remove`,
does not delete the branch, and is destructive; require clear authorization and
inspect dirty state first.

## Panes

Inspection and geometry:

```text
herdr pane list [--workspace ID]
herdr pane current [--pane ID|--current]
herdr pane get <pane_id>
herdr pane layout|process-info|edges [--pane ID|--current]
herdr pane neighbor --direction left|right|up|down [--pane ID|--current]
herdr pane focus --direction left|right|up|down [--pane ID|--current]
herdr pane resize --direction left|right|up|down [--amount FLOAT] [--pane ID|--current]
herdr pane zoom [<pane_id>|--pane ID|--current] [--toggle|--on|--off]
```

Mutation:

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

Read and input:

```text
herdr pane read <pane_id> [--source visible|recent|recent-unwrapped|detection] [--lines N] [--ansi]
herdr pane send-text <pane_id> <text>
herdr pane send-keys <pane_id> <key> [key ...]
herdr pane run <pane_id> <command>
```

Prefer `pane run` for text plus Enter. Use `--current` in the calling pane. An
omitted target can resolve to UI focus. Parse the new public ID after a move.

Custom hooks can use `pane report-agent` for semantic lifecycle state and
`pane report-metadata` for display-only presentation. Read
[socket-api.md](socket-api.md) for the distinction and schema constraints.

## Agents

```text
herdr agent list
herdr agent get|read|send|rename|focus|wait|attach <target> ...
herdr agent start <name> [--cwd PATH] [--workspace ID] [--tab ID] [--split right|down] [--env KEY=VALUE] [--focus|--no-focus] -- <argv...>
herdr agent explain <target> [--json|--verbose]
herdr agent explain --file PATH --agent LABEL [--json|--verbose]
```

Targets can be terminal IDs, unique names, detected or reported labels, or
legacy pane IDs. Use pane commands for ordinary terminals, commands, servers,
tests, and shells. `agent explain` diagnoses classification from the active
manifest cache; after a Herdr upgrade, ensure the server was restarted or handed
off before relying on newly added detection behavior.

## Direct terminal streams and waits

```text
herdr terminal attach <terminal_id> [--takeover]
herdr terminal session control <target> [--takeover] [--cols N] [--rows N]
herdr terminal session observe <target> [--cols N] [--rows N]
herdr terminal title set <title>
herdr terminal title clear
herdr wait output <pane_id> --match <text> [--source SOURCE] [--lines N] [--timeout MS] [--regex] [--raw]
herdr wait agent-status <pane_id> --status idle|working|blocked|done|unknown [--timeout MS]
```

Use observe for read-only frames. A controller owns input and resize authority;
`--takeover` replaces another controller, so require explicit need. Use output
waits for commands and servers and state waits for agents.

## Integrations and plugins

```text
herdr integration install|uninstall <agent>
herdr integration status [--outdated-only]

herdr plugin install <owner>/<repo>[/subdir...] [--ref REF] [--yes]
herdr plugin list [--plugin ID] [--json]
herdr plugin uninstall|enable|disable <plugin>
herdr plugin link <path> [--disabled]
herdr plugin unlink <plugin_id>
herdr plugin config-dir <plugin_id>
herdr plugin action list [--plugin ID]
herdr plugin action invoke <action_id> [--plugin ID]
herdr plugin log list [--plugin ID] [--limit N]
herdr plugin pane open --plugin ID --entrypoint ID [placement and target options]
herdr plugin pane focus|close <pane_id>
```

Inspect plugin trust and manifest warnings before install, link, or invocation.
Install, uninstall, link, unlink, enable, disable, and action invocation change
state or execute third-party code; obtain the authority implied by the user's
request and avoid `--yes` unless noninteractive confirmation is intended.

## Environment and output

Important injected variables are `HERDR_ENV`, `HERDR_SOCKET_PATH`,
`HERDR_WORKSPACE_ID`, `HERDR_TAB_ID`, and `HERDR_PANE_ID`. Herdr-managed values
remain authoritative over caller-provided launch environment.

Most commands return JSON. Treat all public IDs as opaque, inspect response
types and error codes, tolerate unknown fields, and refresh records after every
mutation.
