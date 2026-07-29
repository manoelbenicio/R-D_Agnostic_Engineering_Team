---
name: herdr-guide
description: "Teach humans how to set up, configure, and troubleshoot Herdr. Use when the user asks about Herdr concepts, installation, configuration, keybindings, agent detection, integrations, remote access, session state, or diagnosis. Do not use for programmatic control of Herdr — use herdr-control for that."
license: MIT
metadata:
  author: Kiro
  version: "1.0"
  source: "https://herdr.dev/docs/"
---

# herdr-guide — Teaching & Troubleshooting Skill

## Purpose

Help humans understand, install, configure, and diagnose Herdr. This skill is about **teaching a person**. For programmatic control of Herdr from inside a pane, use the `herdr-control` skill instead.

Canonical documentation: https://herdr.dev/docs/ — link humans there for depth. Verify any command you are unsure about against those pages instead of guessing.

---

## What Herdr Is

Herdr is a terminal workspace manager for AI coding agents. Like tmux, it is a multiplexer: a background server owns real terminal processes, and clients attach to render them. Panes keep running when the human detaches, closes the terminal, or disconnects SSH.

Unlike tmux, Herdr is:

- **Mouse-first** — the entire UI is clickable (panes, tabs, workspaces, split borders, right-click menus).
- **Agent-aware** — detects coding agents in panes and shows their state in a sidebar.
- **Scriptable** — a CLI and local socket API let scripts and agents drive it programmatically.

Herdr is NOT tmux. Never give tmux commands, `.tmux.conf` syntax, or tmux-specific advice for Herdr questions.

---

## Concept Model

Teach these in order:

| Concept | What it is |
|---------|------------|
| **Session** | A persistent background server namespace. `herdr` attaches to the default. Named sessions (`herdr --session work`) are fully separate; most people only need the default. |
| **Workspace** | Project-level container. One per repo/task. Owns tabs and panes. Sidebar rolls agent states up per workspace. |
| **Tab** | A layout inside a workspace. Separate views like `agents`, `logs`, `server`. |
| **Pane** | A real terminal. Splittable right or down. Survives client detach. |
| **Agent** | A process Herdr recognizes inside a pane. States: `working`, `blocked`, `done`, `idle`, `unknown`. |
| **Modes** | Terminal mode sends keys to focused pane. Prefix mode (`ctrl+b` then action key) sends one command to Herdr. Navigate mode is a persistent navigation surface. |

Full concepts: https://herdr.dev/docs/concepts/

---

## Installation

### Linux and macOS

```bash
curl -fsSL https://herdr.dev/install.sh | sh
herdr
```

### Windows (preview beta)

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"
herdr
```

### Other methods

Homebrew, mise, Nix, verification, and manual downloads: https://herdr.dev/docs/install/

### Update and version

```bash
herdr update              # download and install from configured channel
herdr --version           # print version
herdr channel show        # stable or preview
herdr channel set preview # opt into preview builds
herdr channel set stable  # return to stable
```

---

## First-Run Walkthrough

**Important:** If you are running inside a Herdr pane (`HERDR_ENV=1` is set), the human is already attached. Skip step 1 and never tell them to run `herdr` from your pane — Herdr blocks nested launches by design.

### The sequence

1. **Launch:** `cd` into a project and run `herdr`. It launches/attaches to the default session and creates a workspace automatically. First run shows onboarding.

2. **Start an agent:** Run `claude`, `codex`, or any supported agent in the pane. Herdr detects it automatically; the sidebar shows its state. Installing the matching integration improves detection:
   ```bash
   herdr integration install claude
   ```

3. **Show the mouse first:** Click panes/tabs to focus, drag split borders, right-click for menus, drag-select to copy. No keybindings required.

4. **Split panes:** Right-click menu, or `prefix+v` (right) / `prefix+minus` (down). New tab: `prefix+c`.

5. **Detach:** `prefix+q` (press `ctrl+b`, release, press `q`) or simply close the terminal window. Everything keeps running.

6. **Reattach:** Run `herdr` again.

7. **Stop everything:** `herdr server stop` (actually terminates pane processes).

---

## The Keyboard Story

**Frame this clearly for new users:** Herdr does NOT require learning keybindings. The mouse covers everything.

When the human wants keyboard control:

- **Prefix key:** `ctrl+b` by default. Press it, release, then press the action key.
- **Show all bindings:** `prefix+?` (live in the running session).
- **Every binding is configurable** under `[keys]` in the config file.
- **If a chord does nothing:** The OS or outer terminal consumed it before Herdr could see it.

Guided keyboard page (what to learn first, vetted prefix-free `ctrl+alt` setup): https://herdr.dev/docs/keyboard/

Recommend that page over improvising bindings.

---

## Configuration

- **Config file:** `~/.config/herdr/config.toml` (Herdr works without one).
- **Print defaults:** `herdr --default-config`
- **Reload live:** `herdr server reload-config` (or global menu → reload config).

Main config areas:

| Section | Controls |
|---------|----------|
| `[keys]` | Keybindings |
| `[theme]` | Colors and themes |
| `[ui]` | Sidebar, toast, and UI behavior |
| `[terminal]` | Shell defaults |
| `[update]` | Channel and update behavior |
| `[experimental]` | Feature flags (e.g. `kitty_graphics`) |

Full reference: https://herdr.dev/docs/configuration/

---

## Agent Detection & Integrations

Herdr identifies agents via two mechanisms:

1. **Screen detection** — reads the terminal buffer and matches patterns (default, works without setup).
2. **Integrations** — installed hooks that give Herdr authoritative state from the agent's own protocol.

### Manage integrations

```bash
herdr integration install claude    # install integration
herdr integration uninstall claude  # remove integration
herdr integration status            # show all, with outdated info
herdr integration status --outdated-only
```

Supported agents: `pi`, `omp`, `claude`, `codex`, `copilot`, `devin`, `droid`, `kimi`, `opencode`, `kilo`, `hermes`, `qodercli`, `cursor`, `mastracode`.

### Diagnose detection

```bash
herdr agent list                    # what Herdr currently sees
herdr agent explain <target> --json # why the detector classified this pane
herdr server agent-manifests        # active manifest sources and versions
herdr server reload-agent-manifests # reload after local override edits
```

`agent explain` shows: final state, manifest source/version, matched rule, evaluated rule evidence, skip-state reason, idle fallback reason, and `screen_detection_skip_reason` when an integration has authority.

Full docs: https://herdr.dev/docs/agents/ and https://herdr.dev/docs/integrations/

---

## Remote Access

Two approaches:

| Method | How | Best for |
|--------|-----|----------|
| SSH + `herdr` on remote | SSH to the machine, run `herdr` there | Full multiplexer on server, like tmux over SSH |
| `herdr --remote <host>` | Thin local client, rendering happens locally, processes stay remote | Low-latency UI with local keybindings |

```bash
herdr --remote workbox                        # remote with local keybindings
herdr --remote workbox --remote-keybindings server  # use server's bindings
herdr --remote workbox --handoff              # live handoff to new binary
```

Trade-offs and setup: https://herdr.dev/docs/how-to-work/

---

## Session State & Persistence

What survives a detach:
- All pane processes, scrollback, and agent states.

What survives a server restart:
- Workspace/tab/pane structure, labels, cwd.
- NOT live PTYs, scrollback, or running processes (they terminate on server stop).

What survives an update with `--handoff`:
- Everything — live processes transfer to the new server binary.

Full details: https://herdr.dev/docs/session-state/

---

## Diagnosis Recipes

### Agent not detected or wrong state

```bash
herdr agent list                    # see what Herdr sees
herdr agent explain <target> --json # why it was classified this way
herdr integration install <name>    # install authoritative integration
herdr integration status            # check integration health
```

### Keybinding does nothing

The outer terminal or OS consumed the chord before Herdr saw it. Point human to https://herdr.dev/docs/keyboard/ to pick a safe chord or free it in their terminal settings.

### Startup or socket errors

```bash
herdr status
herdr status server
herdr status client
```

Logs:
- `~/.config/herdr/herdr.log`
- `~/.config/herdr/herdr-client.log`
- `~/.config/herdr/herdr-server.log`

Enable debug logging: `HERDR_LOG=herdr=debug herdr`

### UI or rendering issues

- Check terminal emulator compatibility (modern emulators: WezTerm, Ghostty, Kitty, Alacritty, iTerm2, Windows Terminal).
- Try `herdr --default-config` to rule out config issues.
- Check `herdr --version` is current.

### Plugin issues

```bash
herdr plugin list --json            # see all plugins, warnings
herdr plugin log list --plugin ID   # recent command logs
herdr plugin config-dir <plugin_id> # find config location
```

Plugin link/load warnings appear in `plugin.list` responses (e.g. missing manifests, unknown event names).

### Socket API issues

```bash
herdr api schema                    # verify protocol summary
herdr api snapshot                  # live session bootstrap
herdr status server                 # server health
```

---

## Installing the Herdr Skill Into Agents

Herdr ships `SKILL.md` — an instruction file that teaches a coding agent to control Herdr from inside a pane.

### For agents with a skills CLI

```bash
npx skills add ogulcancelik/herdr --skill herdr -g
```

### For agents without a skill system

Paste the content from https://raw.githubusercontent.com/ogulcancelik/herdr/master/SKILL.md into global custom instructions.

### For this project

The `herdr-control` skill at `.kiro/skills/herdr-control/SKILL.md` already provides Herdr operational knowledge to agents working in this repo.

Always ask the human before writing to their config locations.

---

## Quick Reference Links

| Topic | URL |
|-------|-----|
| All docs | https://herdr.dev/docs/ |
| Concepts | https://herdr.dev/docs/concepts/ |
| Install | https://herdr.dev/docs/install/ |
| Keyboard | https://herdr.dev/docs/keyboard/ |
| Configuration | https://herdr.dev/docs/configuration/ |
| Agents | https://herdr.dev/docs/agents/ |
| Integrations | https://herdr.dev/docs/integrations/ |
| CLI reference | https://herdr.dev/docs/cli-reference/ |
| Socket API | https://herdr.dev/docs/socket-api/ |
| Session state | https://herdr.dev/docs/session-state/ |
| Remote/SSH | https://herdr.dev/docs/how-to-work/ |

---

## Rules for This Skill

1. **Do not invent** keybindings, config keys, or CLI flags. Verify against linked docs if unsure.
2. **Teach mouse before keyboard** for humans new to multiplexers.
3. **Herdr is not tmux.** Never give tmux commands or `.tmux.conf` advice.
4. **Link canonical docs** for depth rather than reproducing entire pages.
5. **For automation/scripting**, point to CLI reference and socket API docs or recommend the `herdr-control` skill.
6. **Ask before writing** to the human's config files or agent settings.
7. **If running inside Herdr** (`HERDR_ENV=1`), never tell the human to launch `herdr` — they are already attached.
