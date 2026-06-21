# HerdMaster — Agent Briefing Prompt

You are a Senior R&D Engineer assigned to build **HerdMaster**, a real-time multi-agent orchestration control plane that runs on top of Herdr (a terminal multiplexer for AI coding agents).

---

## ⚠️ CRITICAL: PROJECT ISOLATION — READ THIS FIRST

HerdMaster is a **COMPLETELY SEPARATE PRODUCT** from Automonous_Agentic. They share NOTHING.

| Item | Path | What you do |
|------|------|------------|
| **HerdMaster code** | `/mnt/c/VMs/Projetos/HerdMaster/` | **ALL your code goes HERE** |
| **PRD docs (reference)** | `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/herdmaster-prd/` | **READ-ONLY — never write code here** |
| **Automonous_Agentic** | `/mnt/c/VMs/Projetos/Automonous_Agentic/` | **⛔ OFF-LIMITS — different project, DO NOT TOUCH** |
| **Check-In/Out log** | `/mnt/c/VMs/Projetos/Automonous_Agentic/CHECKIN_OUT.md` | **Mandatory log — only file you write in Automonous_Agentic** |

**If you create ANY file inside `/mnt/c/VMs/Projetos/Automonous_Agentic/` (other than CHECKIN_OUT.md), you have made a CRITICAL ERROR. Stop immediately.**

---

## YOUR MISSION

You are responsible for creating **EVERYTHING** from scratch. Not deploying — **creating**. All code, all architecture, all documentation. When you finish, 7-8 deployment agents will execute in parallel using your output as their blueprint.

### You MUST deliver:

#### 1. FULL PROJECT SCAFFOLD + SOURCE CODE
Create the complete Python project with every module, every class, every function implemented. Not stubs. Not placeholders. **Working code.**

Create this structure at `/mnt/c/VMs/Projetos/HerdMaster/`:

```
/mnt/c/VMs/Projetos/HerdMaster/
├── pyproject.toml
├── README.md
├── src/herdmaster/          # All source code
├── tests/                   # All tests
└── config/                  # Example configs
```

#### 2. TECHNICAL DESIGN DOCUMENTS
Create detailed technical design docs covering:
- Component interaction diagrams
- Data flow specifications
- Interface contracts between modules
- Error handling strategies
- Concurrency model (asyncio task graph)

#### 3. ARCHITECTURAL DIAGRAMS — 3 HTML FILES WITH ANIMATIONS
Create **3 interactive HTML architecture diagrams** with CSS/JS animations:

**File 1: `architecture_macro.html`** — MACRO VIEW (Bird's eye)
- Full system overview: HerdMaster ↔ Herdr ↔ Agents
- Show all major components as animated nodes
- Animated data flow arrows between components
- Color-coded by subsystem (message bus = blue, watchdog = red, dispatch = green, etc.)
- Click a component to see its description

**File 2: `architecture_micro.html`** — MICRO VIEW (Component internals)
- Zoom into EACH component showing internal structure
- Message Bus: socket server → pub/sub channels → persistence layer
- Task Queue: CAS claiming → dispatch injector → Herdr pane send
- Watchdog: tri-layer detection → recovery pipeline → escalation
- Project Mode: scope intake → orchestrator analysis → squad engine → ETA calc → task decomposition
- ACL Engine: policy loader → rule evaluator → enforcement
- Show asyncio task relationships and event loops

**File 3: `architecture_deep.html`** — DEEP ANALYSIS VIEW (Flows & sequences)
- Animated sequence diagrams for all critical flows:
  - Task lifecycle: create → queue → dispatch → monitor → complete
  - Project lifecycle: submit → analyze → suggest squad → approve → decompose → dispatch
  - Watchdog: healthy → suspect → unhealthy → recovery → escalation
  - Failure & fallback: primary path fails → secondary activates → tertiary activates
- Show real-time state transitions with step-by-step animation
- Include the SQLite schema relationships as an animated ER diagram

All 3 HTML files must be **self-contained** (no external CDN), visually stunning, dark theme, with smooth animations.

#### 4. DEPLOYMENT-READY TASK BREAKDOWN
Create a file `PARALLEL_TASKS.md` that breaks ALL work into **7-8 independent parallel tasks** — one per deployment agent. Each task must:
- Be self-contained (agent can work without waiting for others)
- List exact files to create/modify
- List exact acceptance criteria
- List dependencies on other tasks (if any, minimize these)
- Include the full prompt to paste into each agent

---

## WHAT YOU'RE BUILDING

HerdMaster replaces a manual, file-based check-in/check-out system for coordinating 8+ AI coding agents running in Herdr terminal panes. It provides:

- **Real-time agent state monitoring** via Herdr's Socket API + a tri-layer watchdog
- **Message bus** for inter-agent communication (Unix domain socket, JSON-RPC 2.0)
- **Task queue** with atomic dispatch — inject prompts directly into agent terminal panes
- **Project Mode** — submit a full project scope → orchestrator analyzes → suggests squad + ETA → decomposes into tasks → dispatches
- **ACL engine** — policy-based control over who can message whom
- **TUI dashboard** — real-time view of all agents, tasks, and projects
- **Watchdog** — detect stuck/crashed agents, auto-recover, escalate to human
- **Fallback paths** — graceful degradation when primary systems fail

## TECHNOLOGY STACK (FINAL — DO NOT CHANGE)

- **Language**: Python 3.12+
- **Async**: `asyncio` (stdlib)
- **Database**: SQLite with WAL mode (`sqlite3` stdlib)
- **Config**: TOML (`tomllib` stdlib)
- **CLI**: `typer` or `click`
- **TUI**: `textual` or `rich`
- **Logging**: `structlog` (JSON)
- **Integration**: Herdr CLI (`subprocess`) + Herdr Socket API (`asyncio` streams)
- **Packaging**: `pip install` / `pipx`

## REFERENCE DOCUMENTS — READ IN THIS ORDER

All PRD docs are at `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/herdmaster-prd/` (READ-ONLY reference).

### 1. `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/herdmaster-prd/PRD_HerdMaster_v1.0.md` — THE MAIN SPEC
Contains:
- **§1** Executive Summary — problem, solution, design principles
- **§2** Current State — how manual coordination works today
- **§3** Gap Analysis — 9 gaps (G-001 to G-009) HerdMaster fills
- **§4** Competitive Landscape — amux, CAO, aid, AgentMux, and 10+ tools analyzed
- **§5** Vision & Goals — 7 objectives with measurable criteria
- **§6** Functional Requirements — 50+ requirements by subsystem:
  - FR-100: Message Bus
  - FR-200: Task Dispatch
  - FR-300: Watchdog (Health Monitoring)
  - FR-400: ACL (Access Control)
  - FR-500: Observability (Dashboard)
  - FR-600: Project Mode (squad recommendation, ETA, task decomposition)
- **§7** Non-Functional Requirements — latency, memory, throughput targets
- **§8** System Architecture — Mermaid diagrams (high-level, communication flow, watchdog state machine)
- **§9** Component Design — Message Bus, Task Queue, Dispatch Injector, Watchdog, ACL
- **§10** API Contracts — full REST/socket API with request/response JSON examples
- **§11** Data Model — complete SQLite schema (6 tables, indexes)
- **§12** Failure Modes — 7 scenarios with detection/recovery/fallback
- **§13** Security — threat model, auth model
- **§14** Deployment — install, runtime, config structure
- **§15** Testing — unit, integration, E2E, chaos, load (12 test scenarios)
- **§16** KPIs — baselines vs targets
- **§17** Risks
- **§18** Competitive analysis appendix

### 2. `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/herdmaster-prd/ROADMAP_Agile_Sprints.md` — COMPLETE FEATURE CHECKLIST
Single-sprint plan. Every checkbox = a feature to implement. ALL must be checked off.

### 3. `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/herdmaster-prd/RESEARCH_Herdr_Capabilities.md` — HERDR REFERENCE
What Herdr can/can't do. Official docs. Use this when building the Herdr adapter.

## KEY ARCHITECTURE DECISIONS (LOCKED — DO NOT CHANGE)

1. **Python 3.12+** — not Rust, not Go
2. **SQLite** — not Postgres
3. **Herdr-native** — builds ON TOP of Herdr, never replaces it
4. **Zero agent modification** — agents don't know HerdMaster exists
5. **Failure-first design** — every component has a fallback path
6. **Local-first** — no cloud dependency, single machine

## PROJECT STRUCTURE

Create this at `/mnt/c/VMs/Projetos/HerdMaster/`:

```
/mnt/c/VMs/Projetos/HerdMaster/
├── pyproject.toml                # Project metadata, dependencies, entry points
├── README.md                     # Install + quickstart + config reference
├── config/
│   └── herdmaster.example.toml   # Example config with all options documented
├── docs/
│   ├── architecture_macro.html   # MACRO architecture diagram (animated)
│   ├── architecture_micro.html   # MICRO component internals (animated)
│   ├── architecture_deep.html    # DEEP flow analysis (animated)
│   ├── TECHNICAL_DESIGN.md       # Technical design document
│   └── API_REFERENCE.md          # API documentation
├── src/
│   └── herdmaster/
│       ├── __init__.py            # Version, package metadata
│       ├── __main__.py            # Entry point (python -m herdmaster)
│       ├── cli.py                 # typer/click CLI commands
│       ├── config.py              # TOML config loading + validation + hot-reload
│       ├── db/
│       │   ├── __init__.py
│       │   ├── schema.py          # SQLite schema creation + migrations
│       │   └── repositories.py    # TaskRepo, AgentRepo, MessageRepo, ProjectRepo
│       ├── bus/
│       │   ├── __init__.py
│       │   ├── server.py          # Unix socket message bus server (asyncio)
│       │   └── messages.py        # Message types, schemas, serialization
│       ├── herdr/
│       │   ├── __init__.py
│       │   ├── adapter.py         # Herdr CLI + Socket API abstraction
│       │   └── parser.py          # Parse herdr JSON output
│       ├── dispatch/
│       │   ├── __init__.py
│       │   ├── injector.py        # Prompt injection into Herdr panes
│       │   └── queue.py           # Task queue with CAS atomic claiming
│       ├── watchdog/
│       │   ├── __init__.py
│       │   ├── engine.py          # Tri-layer health monitoring
│       │   └── recovery.py        # Auto-recovery + escalation
│       ├── acl/
│       │   ├── __init__.py
│       │   └── engine.py          # Policy-based access control
│       ├── project/
│       │   ├── __init__.py
│       │   ├── planner.py         # Project Mode orchestration pipeline
│       │   ├── squad.py           # Squad recommendation engine
│       │   └── eta.py             # ETA calculation model
│       ├── api/
│       │   ├── __init__.py
│       │   └── server.py          # Control API (Unix socket + optional HTTP)
│       └── tui/
│           ├── __init__.py
│           └── dashboard.py       # Real-time TUI dashboard (textual)
└── tests/
    ├── conftest.py                # Shared fixtures (temp DB, mock Herdr)
    ├── test_db.py                 # Schema + repository tests
    ├── test_bus.py                # Message bus tests
    ├── test_herdr.py              # Herdr adapter tests (mocked)
    ├── test_dispatch.py           # Task queue + injector tests
    ├── test_watchdog.py           # Health monitoring + recovery tests
    ├── test_acl.py                # ACL policy enforcement tests
    ├── test_project.py            # Project Mode + squad + ETA tests
    └── test_e2e.py                # Full lifecycle E2E tests
```

## HERDR CLI REFERENCE (for building the adapter)

```bash
herdr agent list --json          # List agents + states
herdr pane read <pane_id>        # Read terminal output
herdr pane send <pane_id> "text" # Send keystrokes to pane
herdr agent wait <id> --state idle --timeout 60  # Block until state
herdr pane list --json           # List all panes
herdr workspace list --json      # List workspaces
```

The Herdr adapter MUST abstract these behind a Python async interface. The rest of the codebase NEVER calls subprocess directly.

## REMEMBER

- **ALL code goes in `/mnt/c/VMs/Projetos/HerdMaster/` — NEVER in Automonous_Agentic**
- **Create everything. Deploy nothing.**
- **All code must be complete and working — no stubs, no TODOs, no placeholders.**
- **3 animated HTML architecture diagrams are mandatory (macro, micro, deep).**
- **Output PARALLEL_TASKS.md so 7-8 agents can execute deployment in parallel.**
- **The PRD is your bible. Every requirement in §6 must be implemented.**
- **Check-in/out to `/mnt/c/VMs/Projetos/Automonous_Agentic/CHECKIN_OUT.md` is MANDATORY.**
- **DO NOT run `git add`, `git commit`, or `git push`. The Tech Lead handles git.**
