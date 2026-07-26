# Handover / Transition — MULTICA (Main Brain System)

**To:** New Squad (incoming agents & tech leads)
**From:** Outgoing engineering (dataops-lab environment)
**Date:** 2026-07-24
**Subject:** Full operational handover of **Multica** — the managed-agents orchestration platform ("main brain"), its infra, OpenSpec status, OmniRoute integration, access and runbook.
**Format:** `.md` living document — keep updated.

---

## 0. TL;DR (read first)
- **Multica is the main brain**: a self-hosted, vendor-neutral **"managed agents"** platform. You assign an issue to an AI agent like you would to a human; it picks up the work, writes code, reports blockers, updates status autonomously. Board + conversations + composable skills. Tagline: *"Your next 10 hires won't be human."*
- **Runs on orq1** (Docker Compose `multica-dev-transition`): web UI `:13100`, backend `:18080`, Postgres(pgvector) `:15433`, plus `multica-auth-fi`.
- **Repo:** `github.com/manoelbenicio/Agentic_Autonomous.git` (our work); upstream `github.com/multica-ai/multica`. Local source: `/mnt/c/VMs/Projetos/Automonous_Agentic/`.
- **OpenSpec (sovereign):** `/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/` — flagship change in progress: **rotation-parity-polyglot** (make Multica launch the `prodex` L2 Rust runtime instead of raw vendor CLIs).
- **OmniRoute** = the unified AI gateway (one endpoint for all LLM providers) that agents route model traffic through — runs on orq1 `:20128` (UI) / `:20129` (API).
- **Access:** via Tailscale, from WSL. Windows on this host is NOT on Tailscale (Netskope-blocked). UI ports are loopback-bound on orq1 → reach them via SSH tunnel.

---

## 1. What Multica is (product)
- Open-source **managed-agents** platform, **vendor-neutral, self-hosted**. Site: multica.ai.
- Turns coding agents into teammates: assign an issue → agent works autonomously, reports blockers, updates status.
- Board + conversations + **skills** that compose over time.
- **Supported agent CLIs:** Codex, Claude Code, GitHub Copilot CLI, OpenClaw, OpenCode, Hermes, Gemini, Pi, Cursor Agent, Kimi, **Kiro CLI**, Qoder CLI.
- **Squads:** routing layer — assign work to a squad led by an agent that delegates to the right member.
- Board columns (workspace `orq2-dev`): Backlog → Todo → In Progress → In Review → Done → Blocked. Cards `ORQ-N`.

## 2. Architecture (monorepo)
Go backend + JS frontend monorepo (pnpm workspaces + Turborepo).
```
server/            Go backend (Chi router, sqlc, gorilla/websocket) — module github.com/multica-ai/multica/server, Go 1.26.1
apps/web/          Next.js (App Router)  → the UI on :13100
apps/desktop/      Electron
packages/core/     headless business logic (Zustand, React Query, API client)
packages/ui/       atomic components (shadcn / Base UI)
packages/views/    shared pages/components
```
- **State:** React Query (server state) + **Postgres `pgvector/pgvector:pg17`** + **Redis**.
- **Architecture source of truth:** `CLAUDE.md` (repo root) — read first; `AGENTS.md` is a pointer; `Makefile`/`package.json`/`pnpm-workspace.yaml` = commands.

### 2.1 Key Go packages (`server/internal`)
| Package | Role |
|---|---|
| `handler` (143 files) | HTTP API (Chi) |
| `daemon` (49) | **launches & manages agents** (lifecycle, dispatch) — HOTSPOT |
| `daemon/execenv` (34) | **per-account/vendor isolation**: `CODEX_HOME` per task, isolated `HOME` (Antigravity/Kiro), copies `auth.json` per account |
| `rotation` (32) | account rotation (Go cold path) |
| `l2runtime` + `daemon/prodex.go` | client/launcher for **prodex** (the L2 Rust runtime) |
| `auth` (13) | credentials/tokens |
| `metrics`, `middleware`, `realtime`, `scheduler`, `storage` | infra |

## 3. Deployment (orq1) — running now
Docker Compose project **`multica-dev-transition`** on **orq1** (100.118.244.61, EC2 sa-east-1, internal ip-172-31-18-217):
| Container | Image | Port (loopback on orq1) |
|---|---|---|
| multica-dev-transition-frontend-1 | multica-web:transition-6a2aba3 | `127.0.0.1:13100 → 3000` (UI) |
| multica-dev-transition-backend-1 | multica-backend:t20obs-8f524054 | `127.0.0.1:18080 → 8080` (API) |
| multica-dev-transition-postgres-1 | pgvector/pgvector:pg17 | `127.0.0.1:15433 → 5432` (healthy, up 3d) |
| (host process) multica-auth-fi | — | `127.0.0.1:24318`, `127.0.0.1:19514` (auth) |
> All bound to **loopback on orq1** → to open the UI from another machine you need an SSH tunnel to `orq1:13100` (same pattern as OmniRoute, see §7).

## 4. OmniRoute integration (unified AI gateway)
- **OmniRoute** = "route any LLM through one endpoint" — the model/credential gateway agents use. Runs on **orq1**: UI `:20128`, OpenAI-compatible API `:20129`. Repo `github.com/diegosouzapw/OmniRoute`; local `/mnt/c/VMs/Projetos/Omini_Router`.
- **Role in the brain:** Multica's `daemon/execenv` isolates each agent by account/vendor; agents' **LLM traffic is routed through OmniRoute** (one endpoint, multi-provider), and the `rotation` package + OmniRoute handle **account rotation** across providers. Agents do not use raw API keys — OmniRoute brokers auth.
- **Status:** OmniRoute healthy on orq1 (official image `diegosouzapw/omniroute:latest`) after a broken custom build (`3.8.48-affinity-fix2`, blank UI) was reverted; broken container kept as `omniroute-broken-3.8.48-affinity-fix2` for rollback. Local uncommitted `docker-compose*.yml` changes exist (review before commit/push).

## 5. OpenSpec status (SOVEREIGN) — `/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/`
Active changes and task progress (as of 2026-07-24):
| Change | Progress | Notes |
|---|---|---|
| **rotation-parity-polyglot** | **30/66** | Flagship: Multica launches `prodex` (L2 Rust runtime) instead of raw vendor CLIs; polyglot parity |
| milestone-1-canvas-deploy-run | 184/186 | Near-complete |
| cloud-runtime-deployment | 36/38 | Near-complete |
| design-system-indra-alignment | 23/25 | Near-complete (has SEV0 pack: EVIDENCE_LOG, RCA, RELEASE_READINESS, TEST_STRATEGY) |
| finops-tier2-token-parsing | 12/15 | |
| validation-proxy | 8/10 | |
| tech-debt-schema-version-shared | 9/10 | |
| tech-debt-react-refresh-cleanups | 11/12 | |
| tech-debt-keystore-validator-coverage | 18/18 | ✅ done |
| tech-debt-smoke-voice-real-flow | 12/12 | ✅ done |
| tech-debt-voice-coverage-gap | 13/13 | ✅ done |
| tech-debt-voice-event-bus | 15/15 | ✅ done |
| **agent-credential-isolation** | **0/21** | NOT started |
| **dev-env-reliability** | **0/16** | NOT started |
| **prod-readiness-critical-fixes** | **0/7** | NOT started |
| archived | — | `2026-07-04-rotation-router` |
> Note: an identical OpenSpec tree also exists under `RD_Agnostic_Engineering_Team/RD_Agnostic_Engineering_Team/openspec/` (likely a mirror/worktree) — confirm which is canonical before editing.

## 6. Repos, source & docs
| Item | Location |
|---|---|
| Work repo (git) | `https://github.com/manoelbenicio/Agentic_Autonomous.git` (branch `main`) |
| Upstream | `https://github.com/multica-ai/multica` |
| Local source | `/mnt/c/VMs/Projetos/Automonous_Agentic/multica-src` |
| Auth work | `/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work` (git remote → Agentic_Autonomous) |
| OpenSpec | `/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/` |
| Product context (READ FIRST) | `/mnt/c/VMs/Projetos/Automonous_Agentic/Diligencias/00_CONTEXTO_MULTICA.md` |
| Multica reference docs | `/mnt/c/VMs/Projetos/Automonous_Agentic/docs/multica-reference` |
| New-features mapping | `/mnt/c/VMs/Projetos/Automonous_Agentic/Mapeamento_New_Features/02_multica` |
| Local config/creds | `/home/dataops-lab/.multica`, `/home/dataops-lab/multica-auth-creds`, `/home/dataops-lab/multica_workspaces_staging` |
| OmniRoute repo (local) | `/mnt/c/VMs/Projetos/Omini_Router` (remote diegosouzapw/OmniRoute) |
| Architecture source of truth | `CLAUDE.md` at repo root |

## 7. Access & network
- **Tailnet:** `tail96e2c0.ts.net` · account **cloud.labs.brazil@gmail.com** · MagicDNS on.
- **orq1** = 100.118.244.61 (public 15.228.84.165) — runs Multica + OmniRoute.
- **orq2** = 100.110.178.47 (public 15.229.146.145) — root EBS expanded 24→48 GiB.
- This workstation: WSL node `wsl-dataops-labs` 100.117.245.15. **Windows here is NOT on Tailscale** (Netskope hijacks `*.tailscale.com` DNS → 403; no IPv6 route). Use WSL, or a Tailscale machine (e.g., `msi-laptop-1`).
- **SSH:** `ssh orq1` / `ssh orq2` (user `ec2-user`, Tailscale SSH, no key). PowerShell functions `orq1`/`orq2` proxy to WSL.

### 7.1 Open Multica UI (loopback-bound on orq1)
```
# From WSL, tunnel orq1:13100 to your machine:
ssh -N -L 13100:127.0.0.1:13100 orq1
# then open in browser:  http://localhost:13100    (workspace: orq2-dev)
```
(OmniRoute already has a persistent tunnel: WSL systemd `omniroute-orq1.service` → `http://localhost:20130`. Backend/DB tunnels analogous: `-L 18080:127.0.0.1:18080 orq1`, `-L 15433:127.0.0.1:15433 orq1`.)

## 8. Runbook
```
# health of Multica stack on orq1
ssh orq1 'docker ps --format "{{.Names}} | {{.Status}}" | grep multica'
# restart a component
ssh orq1 'docker restart multica-dev-transition-backend-1'
ssh orq1 'docker restart multica-dev-transition-frontend-1'
# postgres (pgvector) shell
ssh orq1 'docker exec -it multica-dev-transition-postgres-1 psql -U postgres'
# logs
ssh orq1 'docker logs --tail=200 multica-dev-transition-backend-1'
```

## 9. AS IS → TO BE
- **AS IS:** Multica dev-transition stack running on orq1 (web/back/pgvector + auth); milestone-1 (canvas deploy/run) ~99% and cloud-runtime ~95% complete; several tech-debt changes closed; OmniRoute integrated and healthy; access standardized via Tailscale/WSL.
- **TO BE:** finish **rotation-parity-polyglot** (prodex L2 Rust runtime launch + polyglot parity, currently 30/66); start the three not-started changes (**agent-credential-isolation 0/21**, **dev-env-reliability 0/16**, **prod-readiness-critical-fixes 0/7**); close finops-tier2, validation-proxy, design-system-indra SEV0; commit/push OmniRoute compose fixes; move stack from `-dev-transition` to a hardened prod deployment.

## 10. PENDING / OPEN ITEMS (for new squad)
1. **Finish rotation-parity-polyglot** (30/66) — the flagship; enables `prodex` (L2 Rust) launch path.
2. **Start** agent-credential-isolation (0/21), dev-env-reliability (0/16), prod-readiness-critical-fixes (0/7).
3. **Confirm canonical OpenSpec tree** (Automonous_Agentic vs RD_Agnostic_Engineering_Team mirror).
4. **OmniRoute:** review/commit uncommitted `docker-compose*.yml`; decide on the lost affinity patch (rebuild without breaking UI); clean broken images on orq1.
5. **Reconcile OpenSpec × source × remote** for the merge (was the open ask).
6. **Promote from dev-transition to prod** (images are `transition-*`/`t20obs-*` dev tags).
7. **AWS Agent Toolkit** setup paused at Step 3 (region + `aws login`).

## 11. Gotchas
- Multica/OmniRoute UIs are **loopback-bound on orq1** → always via tunnel from non-orq1 machines.
- **Windows here can't reach `100.x`** (no Tailscale). Use WSL or a Tailscale machine.
- Blank OmniRoute page = stale **PWA/service-worker cache** → use incognito.
- Secrets (`INITIAL_PASSWORD`, `OMNIROUTE_API_KEY`, Redis pwd, Multica auth creds) live in container env / `.env` / `~/.multica` — never commit/echo.
- `ssh` is case-sensitive (lowercase).

---
*Start at §0 → §1 (product) → §7 (access) → §8 (runbook). Escalate §10.*


---

## 12. Infrastructure Inventory (Docker / Containers / Tailscale / AWS)

### 12.1 AWS EC2 (region `sa-east-1`, AZ sa-east-1b)
| Tailnet | Instance ID | Type | Private (internal) | Public | Root EBS |
|---|---|---|---|---|---|
| **orq1** (100.118.244.61) | `i-0d9d441dd364039f9` | m7i-flex.large | ip-172-31-18-217 | 15.228.84.165 | gp3 24 GiB (~71%), `vol-024324f534a37e072` |
| **orq2** (100.110.178.47) | `i-0af937456e125143d` | m7i-flex.xlarge | ip-172-31-30-9 | 15.229.146.145 | gp3 **48 GiB** (expanded from 24), `vol-04089fc818225fbcc` |
- **orq1** = the compute host: runs **all Docker containers** (Multica + OmniRoute). Docker Engine present, `ec2-user` in docker group.
- **orq2** = **no container runtime** (no docker/podman/nerdctl on PATH) — high tailnet traffic (rx ~67 MB); treat as a native worker / secondary node. New squad: confirm orq2's exact role (likely runs agents/prodex natively or is a DR/parity node).
- Access: SSM-managed (both online); SSH via Tailscale as `ec2-user`.

### 12.2 orq1 — Docker containers (full)
| Container | Image | Status | Port |
|---|---|---|---|
| `multica-dev-transition-frontend-1` | multica-web:transition-6a2aba3 | Up 11h | 127.0.0.1:13100→3000 (UI) |
| `multica-dev-transition-backend-1` | multica-backend:t20obs-8f524054 | Up 10h | 127.0.0.1:18080→8080 (API) |
| `multica-dev-transition-postgres-1` | pgvector/pgvector:pg17 | Up 3d (healthy) | 127.0.0.1:15433→5432 (DB) |
| `omniroute` | diegosouzapw/omniroute:latest | Up (healthy) | **100.118.244.61:20128**→20128 |
| `omniroute-broken-3.8.48-affinity-fix2` | omniroute:3.8.48-affinity-fix2 | Exited | — (rollback keep) |
| `omniroute-canary` | omniroute:3.8.48-affinity-fix2 | Exited | — (cleanup candidate) |
| `omniroute-affinity-FAILED-20260721T042218Z` | omniroute:3.8.48-affinity-fix | Exited (143) | — (cleanup candidate) |
| `omniroute-prev-20260721T044004Z` | diegosouzapw/omniroute | Exited | — (cleanup candidate) |
> Note the **bind difference**: Multica containers bind to `127.0.0.1` (loopback → need tunnel); OmniRoute binds to the **Tailscale IP** `100.118.244.61` (reachable directly from any tailnet machine).

### 12.3 orq1 — Docker volumes (persistent data)
| Volume | Used by |
|---|---|
| `multica-dev-transition_pgdata` | Multica Postgres (current stack) |
| `multica-dev-transition_backend_uploads` | Multica backend uploads (current) |
| `multica_pgdata` | Multica Postgres (older/prior stack) |
| `multica_backend_uploads` | Multica backend uploads (prior) |
| `omniroute-data` | OmniRoute (current) |
| `omniroute-canary-data` | OmniRoute canary (stale) |
> Data lives in these volumes — **do not `docker volume prune`** without backup. `_pgdata` = the pgvector database.

### 12.4 Tailscale nodes (tailnet `tail96e2c0.ts.net`)
| Node | IP | OS | Status |
|---|---|---|---|
| wsl-dataops-labs | 100.117.245.15 | linux (WSL) | this workstation / jump |
| orq1 | 100.118.244.61 | linux | **active**, direct 15.228.84.165:41641 — Multica+OmniRoute host |
| orq2 | 100.110.178.47 | linux | **active**, direct 15.229.146.145:41641 |
| msi-laptop-1 | 100.112.85.91 | windows | **active**, direct 192.168.1.3:41641 (has Tailscale) |
| manoelneto-laptop | 100.98.214.121 | linux | active |
| ec2-jump-box | 100.94.211.42 | linux | idle; **offers exit node** |
| lenovo-lab / wsl-lenovo-lab / msi-laptop | 100.120.203.49 / 100.104.184.20 / 100.122.21.119 | linux | offline |
| manoelneto-laptop-1 | 100.86.110.121 | windows | offline |
| a07-de-manoel / poco-x8-pro-max / x8promax | 100.79.199.50 / 100.116.153.53 / 100.80.41.51 | android/linux | offline |
| ipad-air-5th-gen-wifi | 100.71.184.65 | iOS | offline |
- All connections are **direct WireGuard** (hole-punched) — no relay for active nodes. Nearest DERP: São Paulo (~8 ms) as fallback.
- **This Windows host has NO Tailscale** (Netskope blocks `*.tailscale.com` → 403; no IPv6). Reach servers via WSL or via `msi-laptop-1`.
