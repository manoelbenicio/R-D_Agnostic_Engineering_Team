# HerdMaster — Single Sprint Delivery Plan

> **Sprint**: 1 (all features)  
> **Team**: 3× Codex Sr. SME · 2× Nemotron Ultra 550B · 1× Kiro (Opus 4.8)  
> **Mode**: 24/7 continuous  
> **Stack**: Python 3.12+ · SQLite · asyncio · textual/rich  
> **PRD Reference**: [PRD_HerdMaster_v1.0.md](./PRD_HerdMaster_v1.0.md)

---

## Scope — All Features, 1 Sprint

Every feature from the PRD is in-scope. No deferrals, no P2 backlog.

### Core Infrastructure
- [ ] SQLite database with full schema (agents, tasks, messages, health_events, projects, project_history)
- [ ] Agent Registry — sync from Herdr, track state + health + strengths + metrics
- [ ] Configuration system (TOML) with hot-reload
- [ ] Structured logging (structlog, JSON output)

### Herdr Integration
- [ ] Herdr Adapter — abstraction over CLI + Socket API
- [ ] `herdr agent list` parsing → agent registry sync
- [ ] `herdr pane read` → output capture
- [ ] `herdr pane send` → prompt injection (chunked for long prompts)
- [ ] `herdr agent wait` → state blocking
- [ ] Socket API event subscription (real-time state changes)

### Message Bus
- [ ] Unix domain socket server (asyncio)
- [ ] JSON-RPC 2.0 protocol
- [ ] Message types: task_assign, task_update, heartbeat, chat, alert, state_change
- [ ] Unicast (1:1), multicast (1:N), broadcast (1:all)
- [ ] Message persistence to SQLite (audit trail)
- [ ] Message TTL (auto-expire)
- [ ] Message acknowledgment (delivered, read, acted_upon)
- [ ] Pub/sub channels per agent + broadcast channel

### Task Queue & Dispatch
- [ ] Task CRUD with lifecycle states: queued → assigned → dispatched → in_progress → done/failed/timeout/cancelled
- [ ] Atomic task claiming (Compare-And-Swap via SQLite version field)
- [ ] Task dependencies (depends_on graph)
- [ ] Task priority levels: critical, high, normal, low
- [ ] Dispatch Injector — resolve pane ID, check idle, inject prompt, confirm
- [ ] Auto-reassign failed/timed-out tasks to next idle agent
- [ ] Task templates (reusable prompt patterns)

### Project Mode
- [ ] Project entity CRUD (name, scope, deadline, complexity_tier)
- [ ] Orchestrator analysis pipeline — inject scope, parse structured JSON output
- [ ] Squad Recommendation engine — agent capabilities + historical metrics → suggestion
- [ ] ETA Calculation — critical path depth × avg time × complexity multiplier / parallelism
- [ ] ETA presented as optimistic / expected / pessimistic range
- [ ] Human approval flow — accept / modify / override
- [ ] Project → Task auto-decomposition upon approval
- [ ] Project progress tracking (% complete, live ETA recalculation)
- [ ] Project templates (feature, bugfix, refactor, migration)
- [ ] Historical knowledge base for improved ETA accuracy

### ACL Engine
- [ ] Policy-based access control from TOML config
- [ ] Roles: orchestrator, worker, reviewer, observer
- [ ] Communication policies: who can send to whom
- [ ] Default deny, explicit allow
- [ ] Dynamic role changes at runtime

### Watchdog Engine (Tri-Layer)
- [ ] **Primary**: Herdr Socket API event subscription (real-time, <1s)
- [ ] **Secondary**: Periodic CLI polling (`herdr agent list` + `herdr pane read` diff, 15s)
- [ ] **Tertiary**: Terminal output hash comparison (frozen terminal detection, 30s)
- [ ] Health states: healthy → suspect → unhealthy → recovering
- [ ] Auto-recovery: kill hung process → respawn agent → replay last task
- [ ] Escalation: alert human after N consecutive recovery failures
- [ ] All health events logged with timestamps

### Fallback Paths
- [ ] Herdr Socket API down → fall back to CLI polling
- [ ] Unix socket message bus down → fall back to file-based messaging
- [ ] SQLite failure → fall back to in-memory queue
- [ ] Graceful degradation: if HerdMaster crashes, Herdr continues unaffected

### Dashboard (TUI)
- [ ] Real-time agent grid: name, type, state, health, current task, uptime, last heartbeat
- [ ] Task list with lifecycle states and progress
- [ ] Project progress bars with live ETA
- [ ] Notification system: desktop/terminal alerts on escalations
- [ ] Metrics view: tasks/agent, avg completion time, failure rate

### Web Dashboard (Optional)
- [ ] FastAPI backend serving real-time data
- [ ] Vite + React frontend (localhost only)
- [ ] WebSocket stream for live updates

### CLI Interface
- [ ] `herdmaster start` / `stop` / `status`
- [ ] `herdmaster agents` — list all agents with state
- [ ] `herdmaster tasks` — list/create/cancel tasks
- [ ] `herdmaster projects` — list/create/approve projects
- [ ] `herdmaster metrics` — show KPIs
- [ ] `herdmaster config reload` — hot-reload

### Control API
- [ ] Unix socket / optional HTTP (localhost)
- [ ] Project endpoints: POST/GET/PATCH/DELETE /projects, /projects/:id/approve, /projects/:id/eta
- [ ] Task endpoints: POST/GET/PATCH/DELETE /tasks
- [ ] Agent endpoints: GET /agents, POST /agents/:id/message, /agents/:id/restart, GET /agents/:id/metrics
- [ ] Message endpoints: POST/GET /messages, WS /messages/stream
- [ ] System endpoints: GET /status, GET /metrics, POST /config/reload

### Security
- [ ] Agent identity from Herdr pane ID (kernel-enforced)
- [ ] Control API bound to localhost only
- [ ] Optional bearer token for network mode
- [ ] ACL enforcement on all message bus operations

### QA & Testing
- [ ] Unit tests (pytest + pytest-asyncio) — >80% coverage
- [ ] Integration tests with mock Herdr socket — >70% coverage
- [ ] E2E tests: full task lifecycle with real Herdr
- [ ] E2E tests: full project lifecycle (submit → analyze → approve → dispatch → complete)
- [ ] Chaos tests: agent crash, socket drop, DB corruption
- [ ] Load tests: 32 agents, 100 tasks (locust/pytest-benchmark)
- [ ] ACL tests: unauthorized message rejection
- [ ] Watchdog tests: stuck detection, auto-recovery, escalation
- [ ] Fallback tests: primary → secondary → tertiary degradation

### Packaging & Deployment
- [ ] `pip install` / `pipx install` ready
- [ ] PyInstaller single binary (optional)
- [ ] systemd user service unit file
- [ ] README with install + config + quickstart
- [ ] Config reference documentation
- [ ] API documentation
- [ ] Troubleshooting guide

---

## Team

| Agent | Role | Model |
|-------|------|-------|
| Codex #1 | Sr. SME R&D | Codex |
| Codex #2 | Sr. SME R&D | Codex |
| Codex #3 | Sr. SME R&D | Codex |
| Nemotron #1 | Sr. Coding Agent | Nemotron 3 Ultra 550B A55B |
| Nemotron #2 | Sr. Coding Agent | Nemotron 3 Ultra 550B A55B |
| Kiro | Architecture & Support | Opus 4.8 |

---

## Definition of Done

- [ ] All features above checked off
- [ ] All tests passing
- [ ] Documentation complete
- [ ] `pip install herdmaster` works
- [ ] Demo: create project → squad suggested → approved → tasks dispatched → agents monitored → watchdog fires on stuck agent → auto-recovery → project completed
