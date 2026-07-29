# Herdr socket API

Read this reference when the task needs a raw protocol client, a long-lived
event subscription, custom agent or metadata reporting, plugin APIs, pane
graphics, or portable layout manipulation. Use the
[canonical socket API documentation](https://herdr.dev/docs/socket-api/) for
transport details and behavior not captured here.

## Choose the control layer

- Use the `herdr` agent skill for interactive operation from a managed pane.
- Prefer CLI wrappers for shell scripts, routine orchestration, and debugging.
- Use the raw socket API for direct request/response control or long-lived event
  subscriptions.

All layers share the same server control surface. Before any control operation,
retain the parent skill's `HERDR_ENV=1` safety check.

## Discover the installed protocol

Treat the installed binary's schema as authoritative:

```bash
herdr api schema
herdr api schema --json
herdr api schema --output herdr-api.schema.json
```

The JSON Schema covers requests, successful responses, errors, emitted events,
and subscription events. Generate or validate protocol code against it instead
of copying payload shapes from memory. Use `herdr api snapshot` for a JSON
bootstrap of the live `session.snapshot` response.

## Raw method inventory

Method names use dot notation. Confirm parameters in the installed schema.

| Area | Methods |
| --- | --- |
| Server | `ping`, `server.stop`, `server.reload_config`, `server.agent_manifests`, `server.reload_agent_manifests`, `server.live_handoff` |
| Notification | `notification.show` |
| Client | `client.window_title.set`, `client.window_title.clear` |
| Session | `session.snapshot` |
| Workspace | `workspace.create`, `workspace.list`, `workspace.get`, `workspace.focus`, `workspace.rename`, `workspace.move`, `workspace.report_metadata`, `workspace.close` |
| Worktree | `worktree.list`, `worktree.create`, `worktree.open`, `worktree.remove` |
| Tab | `tab.create`, `tab.list`, `tab.get`, `tab.focus`, `tab.rename`, `tab.move`, `tab.close` |
| Pane | `pane.split`, `pane.swap`, `pane.move`, `pane.zoom`, `pane.layout`, `pane.process_info`, `pane.neighbor`, `pane.edges`, `pane.focus`, `pane.focus_direction`, `pane.resize`, `pane.list`, `pane.current`, `pane.get`, `pane.rename`, `pane.send_text`, `pane.send_keys`, `pane.send_input`, `pane.read`, `pane.graphics.info`, `pane.graphics.set`, `pane.graphics.clear`, `pane.report_agent`, `pane.report_agent_session`, `pane.report_metadata`, `pane.clear_agent_authority`, `pane.release_agent`, `pane.close`, `pane.wait_for_output` |
| Popup | `popup.close` |
| Layout | `layout.export`, `layout.apply`, `layout.set_split_ratio` |
| Agent | `agent.list`, `agent.get`, `agent.read`, `agent.explain`, `agent.send_keys`, `agent.prompt`, `agent.wait`, `agent.rename`, `agent.focus`, `agent.start`, `agent.view.set`, `agent.view.clear` |
| Events | `events.subscribe`, `events.wait` |
| Integration | `integration.install`, `integration.uninstall` |
| Plugin | `plugin.link`, `plugin.list`, `plugin.unlink`, `plugin.enable`, `plugin.disable`, `plugin.action.list`, `plugin.action.invoke`, `plugin.log.list`, `plugin.pane.open`, `plugin.pane.focus`, `plugin.pane.close` |

This inventory was checked against Herdr 0.7.5 protocol 17. Refresh it from the
installed schema after an upgrade. Some CLI commands are convenience layers
over these methods.

## Bootstrap and keep state current

Use `session.snapshot` as a one-time bootstrap for clients with a local runtime
cache. It includes protocol metadata, focused resource IDs, workspace, tab,
pane, layout, and agent records. It is not a subscription.

After the snapshot:

1. Subscribe to the resource events the client needs.
2. Update local state from each event.
3. Request a new snapshot after reconnecting or suspected drift.
4. Use `worktree.list` for full repository worktree discovery; snapshot only
   includes attached worktree provenance on workspace records.

## Target panes safely

Public pane IDs such as `w1:p1` are opaque. Use an explicit ID or the caller's
stable `HERDR_PANE_ID`. Optional raw `pane_id` parameters otherwise target the
server's active focused pane. `pane.move` always requires its source pane ID.

`pane.current` accepts `caller_pane_id` and returns that pane; omitting it
returns the active focused pane. Continue to parse fresh IDs from mutation
responses, especially after cross-workspace pane moves.

`pane.send_keys` and `pane.send_input.keys` accept Herdr key-combo strings such
as `enter`, `esc`, `ctrl+h`, `alt+x`, `shift+tab`, `f1`, `minus`, and `plus`.
They do not accept `prefix+` binding notation.

## Manipulate layouts and panes

- Use `pane.layout`, `pane.neighbor`, and `pane.edges` to make geometry decisions
  from server-owned layout snapshots.
- Use `pane.process_info` for available shell PID, foreground process group,
  process argv, and cwd evidence.
- Use `pane.swap` for same-tab rearrangement. It preserves pane IDs, processes,
  layout shape, and ratios.
- Use `pane.move` for a different tab, a new tab, or a new workspace. A
  cross-workspace move preserves the terminal but assigns a new public pane ID.
- Use `pane.zoom` to toggle, enable, or disable tab zoom.
- Use `layout.export` for a portable BSP layout tree.
- Treat `layout.apply` as replacement, not live migration: it creates a fresh
  tab and restores structure, labels, cwd, environment, and optional commands,
  but not PTYs, scrollback, or running processes.
- Use `layout.set_split_ratio` with the schema-defined split path.

Process-launching methods accept an `env` object for the launched process only.
Herdr-managed variables such as `HERDR_SOCKET_PATH`, `HERDR_ENV`, and the public
workspace, tab, and pane IDs remain authoritative on conflicts.

## Report agents and presentation metadata

Use `pane.report_agent` for semantic lifecycle state. It affects waits,
notifications, and workspace rollups. Use `pane.report_agent_session` for a
native session reference that does not change semantic state.

Use `pane.report_metadata` or `workspace.report_metadata` for display-only
titles, displayed agent names, state labels, and token patches. Metadata must
not replace semantic state. Use the schema constraints for sources, token names,
text length, TTL, sequence handling, and per-resource limits.

Read `agent_session`, `foreground_cwd`, terminal titles, and metadata tokens from
fresh pane or agent records when present; treat optional fields as optional.

## Subscribe and wait

Use `events.subscribe` for a long-lived newline-delimited event stream and
`events.wait` or CLI waits for a bounded condition. The first subscription
response acknowledges the subscription; later lines are pushed events.

Use output waits for ordinary commands and servers. Use agent-state waits for
agents; these observe semantic state rather than generic command completion.
Inspect current state or output before waiting for a future transition.

`agent.wait` is server-owned and event-driven. It pins the resolved pane
occupant so a replacement cannot satisfy the wait. Prefer `agent.prompt` with
its optional wait object when a raw client must submit a prompt and wait
without a race between two requests.

## Project the Agents view

Use `agent.view.set` for one transient declarative projection of the built-in
Agents view. Filters support `all`, `any`, `not`, `eq`, `in`, and `exists`;
fields include status, workspace/tab/pane IDs, agent, seen,
`state_change_seq`, and metadata tokens. Sorts can use workspace/tab/pane
order, attention, status, agent, seen, state transition sequence, or tokens.

The projection changes presentation and navigation only. It does not change
`agent.list`, detection, notifications, or attention counts. Use
`agent.view.clear` to remove it. A plugin-owned view is removed when that
plugin is disabled, unlinked, or uninstalled.

## Work with plugins

Plugin actions, event hooks, terminal pane entrypoints, and link handlers come
from `herdr-plugin.toml`; runtime action registration is not part of v1.

- Require `min_herdr_version` and honor effective platform constraints.
- Inspect warnings after linking or listing; an unknown event name may warn
  without rejecting the link.
- Prefer qualified action IDs when names may collide.
- Treat injected plugin paths as discovery locations, not a managed storage API;
  plugins own their files, schemas, migrations, and cleanup.
- Use manifest-declared pane entrypoints. A popup is not a Herdr pane and does
  not participate in pane or agent APIs.
- Preserve Herdr-managed environment variables when adding process environment.

## Use graphics only when enabled

Pane graphics require `[experimental].kitty_graphics = true`; otherwise graphics
methods return `feature_disabled`. Protocol 17 exposes
`pane.graphics.info`, `pane.graphics.set`, and `pane.graphics.clear`. Use
`info` for cell pixel size, `set` for one image, and `clear` to remove it.
Some Herdr documentation also describes a dedicated
`pane.graphics.stream` transport for repeated frames; use it only when the
installed schema advertises it.

## Handle responses defensively

Every request carries an `id`. Successful replies contain a `result`; failures
contain an `error` with a code and message. Check the running protocol with
`ping` or `herdr status` before depending on recent behavior. Accept unknown
fields, branch on documented result types and error codes, and refresh state
after reconnecting.

Do not use raw `server.stop`, destructive close operations, worktree removal, or
plugin uninstall unless the user's request clearly authorizes that effect.
