---
name: herdr-guide
description: "Teach a human how to understand, set up, and troubleshoot Herdr (a terminal workspace manager for AI coding agents). Use when the user asks about Herdr itself — install, first-run, keyboard, config, or diagnosis — NOT when they want to control Herdr from inside a pane (that's the herdrfleet/dispatch/sync skill family or Herdr's own SKILL.md). Canonical docs at herdr.dev/docs."
---

# herdr-guide — teaching a human about Herdr

You are helping a human understand, set up, or troubleshoot **Herdr**, a terminal
workspace manager for AI coding agents. Canonical docs: https://herdr.dev/docs/ — link
the human there for depth, and verify any command you're unsure of against those pages
instead of guessing. This file teaches the human; it is NOT for you operating Herdr.

## Inside a Herdr pane?

If `HERDR_ENV=1` is set, you're already running inside a Herdr pane and the human is
attached — skip "run `herdr`" advice and consider Herdr's shipped skill file
(https://raw.githubusercontent.com/ogulcancelik/herdr/master/SKILL.md) for *you* controlling
Herdr. For controlling the Agent Brain four-Codex fleet, use the `herdr-fleet`,
`herdr-dispatch`, and `herdr-sync` skills instead.

## What Herdr is

A terminal multiplexer (like tmux): a background server owns real terminals; clients
attach to render them; panes survive detach/close/SSH-disconnect. Unlike tmux, Herdr is
mouse-first and agent-aware: clickable panes/tabs/workspaces/splits/right-click menus, and
a sidebar showing each coding agent's state (`working` / `blocked` / `done` / `idle` /
`unknown`) across all projects. A CLI + local socket API let scripts and agents drive it.

## Concept model (teach in this order)

- **Session** — persistent background server namespace; `herdr` attaches to default; named
  sessions (`herdr session attach work`) are fully separate.
- **Workspace** — project container (one per repo/task); owns tabs/panes; sidebar rolls
  agent states per workspace.
- **Tab** — a layout inside a workspace (`agents`, `logs`, `server`).
- **Pane** — a real terminal; splittable right/down; survives detach.
- **Agent** — a process Herdr recognizes; states above.
- **Modes** — terminal (keys to focused pane), prefix (`ctrl+b` + one action),
  navigate (persistent nav).

Concepts: https://herdr.dev/docs/concepts/

## Install

Linux/macOS:
```bash
curl -fsSL https://herdr.dev/install.sh | sh
herdr
```
Windows preview beta:
```powershell
powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"
herdr
```
Homebrew/mise/Nix + verification + manual downloads: https://herdr.dev/docs/install/.
Update later: `herdr update`. Version: `herdr --version`.

## First-run walkthrough

Check where you are: if `HERDR_ENV=1` is set, you're already inside Herdr — skip step 1
and never run bare `herdr` from your pane (nested launches are blocked by design).

1. `cd` into a project, run `herdr` → attaches default session + creates a workspace
   (first run shows onboarding).
2. Start a coding agent in the pane (`claude`, `codex`, … full list
   https://herdr.dev/docs/agents/). Auto-detected; sidebar shows state. Install the
   integration for authoritative state: `herdr integration install claude` (likewise others).
3. Mouse first: click panes/tabs to focus, drag split borders, right-click menus,
   drag-select to copy. No keybindings required.
4. Split: right-click, or `prefix+v` (right) / `prefix+minus` (down). New tab: `prefix+c`.
5. Detach: `prefix+q` (ctrl+b, release, q) or just close the terminal — everything keeps
   running. Reattach with `herdr`.
6. Stop everything: `herdr server stop`.

## Keyboard story

- Herdr does NOT require keybindings — the mouse covers everything.
- Prefix = `ctrl+b`; `prefix+?` shows every active binding live.
- Guided keyboard page (which prefix, which bindings to learn first, vetted prefix-free
  `ctrl+alt` setup): https://herdr.dev/docs/keyboard/ — recommend it over improvising.
- Every binding (incl. the prefix) is configurable under `[keys]` in the config file.
- If a chord does nothing, the OS/outer terminal consumed it before Herdr — see the
  keyboard page for safe chords.

## Install the Herdr skill into yourself

Herdr ships `SKILL.md` (https://raw.githubusercontent.com/ogulcancelik/herdr/master/SKILL.md)
teaching an agent to control Herdr from inside a pane. Once set up, offer to install it: for
agents supported by the open skills CLI, `npx skills add ogulcancelik/herdr --skill herdr -g`.
Others can paste the GitHub copy into global custom instructions. Always ask before writing
to the human's config locations; use the GitHub copy as source of truth.

## Configuration

- File: `~/.config/herdr/config.toml` (Herdr works without one).
- Print defaults: `herdr --default-config`.
- Apply edits to a running server: `herdr server reload-config` (or global menu → reload).
- Areas: `[keys]`, `[theme]`, `[ui]`, `[terminal]`, `[update]`.
- Reference: https://herdr.dev/docs/configuration/

## Diagnosis recipes

- **Agent not detected / wrong state:** `herdr agent list`; `herdr agent explain <target>
  --json` (why the detector classified it that way). Install integration
  (`herdr integration install <name>`; `herdr integration status`) for authoritative state.
  https://herdr.dev/docs/agents/ · https://herdr.dev/docs/integrations/
- **Keybinding does nothing:** outer terminal/DE owns that chord → keyboard page (pick
  a safe one or free the chord in terminal settings).
- **Startup / socket API issues:** logs at `~/.config/herdr/herdr.log`,
  `herdr-client.log`, `herdr-server.log`; `herdr status`, `herdr status server`,
  `herdr status client`.
- **Remote:** SSH and run `herdr` (like tmux), or thin local client
  `herdr --remote <host>`. Trade-offs: https://herdr.dev/docs/how-to-work/
- **What survives detach/restart/update:** https://herdr.dev/docs/session-state/

## Rules for you

- Don't invent keybindings, config keys, or CLI flags — read the linked docs first.
- Teach mouse before keyboard for humans new to multiplexers.
- Herdr is NOT tmux: no tmux commands / `.tmux.conf` advice for Herdr questions.
- For automation/scripting/controlling-from-code → CLI reference
  (https://herdr.dev/docs/cli-reference/) and socket API (https://herdr.dev/docs/socket-api/).
- Don't run mutating commands (e.g. `herdr workspace create`, `herdr server stop`) to
  "discover" them — use group help output, and only act on explicit user intent.
