---
name: teach-herdr
description: "Teach humans how to understand, install, configure, use, or troubleshoot Herdr, the terminal workspace manager for AI coding agents. Use for Herdr onboarding, concept explanations, keyboard or mouse guidance, agent-detection problems, integrations, configuration, logs, remote access, session persistence, and setup diagnosis."
---

# Teach Herdr

Guide the human from concepts to setup and diagnosis. Treat [the canonical Herdr documentation](https://herdr.dev/docs/) as authoritative. Verify any command, flag, keybinding, or configuration key not listed here against the relevant documentation page before presenting it.

## Explain the concept model

Teach the concepts in this order:

1. **Session** — a persistent background server namespace. `herdr` attaches to the default session. Named sessions such as `herdr session attach work` are separate runtime namespaces; most users only need the default.
2. **Workspace** — a project-level container for a repo, task, or investigation. It owns tabs and panes, and the sidebar summarizes agent states per workspace.
3. **Tab** — a layout within a workspace, useful for views such as `agents`, `logs`, and `server`.
4. **Pane** — a real, splittable terminal that survives client detach.
5. **Agent** — a process Herdr recognizes in a pane. States are `working`, `blocked`, `done`, `idle`, and `unknown`.
6. **Modes** — terminal mode sends keys to the focused pane; prefix mode sends one Herdr command after `ctrl+b`; navigate mode provides persistent navigation.

Describe Herdr as a mouse-first, agent-aware terminal multiplexer. A background server owns the terminal processes, while clients attach to render them. Panes continue running after terminal closure, detach, or SSH disconnect. Link to [Herdr concepts](https://herdr.dev/docs/concepts/) for depth.

## Install Herdr

For Linux and macOS, provide:

```bash
curl -fsSL https://herdr.dev/install.sh | sh
herdr
```

For the Windows preview beta, provide:

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"
herdr
```

Point Homebrew, mise, Nix, verification, and manual-download questions to [the install guide](https://herdr.dev/docs/install/). Use `herdr update` to update and `herdr --version` to print the installed version.

## Run the first-use walkthrough

First check whether the current agent is already inside Herdr:

```bash
test "${HERDR_ENV:-}" = 1
```

If the check passes, skip launching Herdr. Never tell the human to run bare `herdr` from the current pane because nested launches are blocked. Start at step 2 and use `$herdr` for control tasks when available.

Otherwise guide the human through:

1. Change into a project and run `herdr`. Explain that this attaches to the default background session and creates a workspace automatically. Mention the first-run onboarding flow.
2. Start a supported coding agent such as `claude` or `codex`. Explain that Herdr detects it and shows its state in the sidebar. Recommend `herdr integration install <agent>` when supported; link to [agents](https://herdr.dev/docs/agents/) and [integrations](https://herdr.dev/docs/integrations/).
3. Teach the mouse first: click panes and tabs to focus, drag split borders, right-click for menus, and drag-select to copy.
4. Split through the right-click menu, or use `prefix+v` for a right split and `prefix+minus` for a downward split. Create a tab with `prefix+c`.
5. Detach with `prefix+q` (`ctrl+b`, release, then `q`) or close the terminal. Explain that processes keep running and `herdr` reattaches later.
6. Stop the server only when the human intends to stop everything, using `herdr server stop`.

## Explain keyboard control

Emphasize that keybindings are optional because the mouse covers the UI.

- State that the default prefix is `ctrl+b`.
- Use `prefix+?` to show the active bindings.
- Recommend [the keyboard guide](https://herdr.dev/docs/keyboard/) for a vetted progression and prefix-free `ctrl+alt` setup.
- Explain that bindings, including the prefix, are configurable under `[keys]`.
- If a direct chord does nothing, explain that the OS or outer terminal may have consumed it before Herdr received it.

Do not improvise keybindings.

## Offer the operating skill

After setup, offer to install Herdr's operating skill so future agent sessions can control Herdr. Ask before writing global agent configuration. Use the [official upstream `SKILL.md`](https://raw.githubusercontent.com/ogulcancelik/herdr/master/SKILL.md) as the source of truth.

For harnesses supported by the open skills CLI, use:

```bash
npx skills add ogulcancelik/herdr --skill herdr -g
```

For an agent without a skill system, suggest adding the upstream skill contents to its global custom instructions.

## Explain configuration

- Use `~/.config/herdr/config.toml`; clarify that Herdr works without this file.
- Print the full defaults with `herdr --default-config`.
- Apply edits to a running server with `herdr server reload-config` or the global menu.
- Describe `[keys]`, `[theme]`, `[ui]`, `[terminal]`, and `[update]` as the main areas.
- Link to [the configuration reference](https://herdr.dev/docs/configuration/) for exact keys.

## Diagnose common problems

### Agent is missing or has the wrong state

Run `herdr agent list` to see Herdr's current view. Run `herdr agent explain <target> --json` to inspect the classification. Recommend `herdr integration install <name>` for authoritative state when supported and `herdr integration status` to inspect installed integrations. Use the [agents](https://herdr.dev/docs/agents/) and [integrations](https://herdr.dev/docs/integrations/) references.

### A keybinding does nothing

Explain that the outer terminal or desktop environment may own the chord. Use [the keyboard guide](https://herdr.dev/docs/keyboard/) to select a safe binding or free it in terminal settings.

### Startup or socket behavior looks wrong

Inspect these logs:

- `~/.config/herdr/herdr.log`
- `~/.config/herdr/herdr-client.log`
- `~/.config/herdr/herdr-server.log`

Use `herdr status`, `herdr status server`, and `herdr status client` to summarize runtime state.

### The human asks about remote use

Explain both supported approaches: SSH to the remote machine and run `herdr` there, or attach a thin local client with `herdr --remote <host>`. Point to [How to work](https://herdr.dev/docs/how-to-work/) for trade-offs.

### The human asks what survives

Use [the session-state documentation](https://herdr.dev/docs/session-state/) for detach, restart, and update behavior.

## Follow the guardrails

- Teach mouse interaction before keyboard shortcuts to users new to multiplexers.
- Do not give tmux commands, `.tmux.conf` syntax, or tmux configuration advice for Herdr.
- Do not invent CLI flags, keybindings, or configuration keys.
- Point automation and scripting questions to the [CLI reference](https://herdr.dev/docs/cli-reference/) and [socket API](https://herdr.dev/docs/socket-api/).
