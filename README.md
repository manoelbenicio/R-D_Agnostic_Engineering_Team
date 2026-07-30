# R&D — Agnostic Engineering Team

Managed-agent platform built around one execution path:

```text
Multica web / Kanban → Main Brain (Agent Brain daemon) → OmniRoute → approved coding-agent CLI/model
                              ↓
                 terminal/session persistence
```

OmniRoute is the **only router owner**. Main Brain owns task admission, workspace/repository setup, process lifecycle, cancellation/watchdogs, message streaming, terminal results and capacity. It does not select provider accounts, inject provider-native credentials, or fall back to a native/provider router. If OmniRoute is unavailable or not ready, new model work fails closed.

## Product components

| Component | Source | Responsibility |
|---|---|---|
| Multica backend | `multica-auth-work/server` | Workspaces, projects, squads, Kanban issues/tasks, API and persistence |
| Multica web | `multica-auth-work/apps/web` | User-facing Kanban/project/squad interface |
| Main Brain daemon | `multica-auth-work/server/internal/daemon` | Task lifecycle, worktrees, terminals, cancellation, gateway admission |
| OmniRoute | externally deployed service | Sole model router and inference credential/account owner |
| Postgres | self-host Compose service | Durable product and control-plane state |

The frontend is the Next.js app under `multica-auth-work/`; there is no separate AgentVerse SPA in this repository.

## Run the self-host product stack

Docker runs the backend, frontend and Postgres. OmniRoute is deployed and operated separately and must be reachable by the host daemon before model work is admitted.

```bash
cd multica-auth-work
cp .env.example .env   # configure required product values out of band
docker compose -f docker-compose.selfhost.yml up -d
```

Open <http://localhost:3100>. The backend health endpoint is <http://127.0.0.1:8080/health>.

## Configure Main Brain

Use the neutral Agent Brain configuration surface:

- `AGENT_BRAIN_DEVELOPMENT_ENABLED=true`
- `AGENT_BRAIN_GATEWAY_REQUIRED=true` (valid only when `AGENT_BRAIN_DEVELOPMENT_ENABLED=true`)
- `AGENT_BRAIN_GATEWAY_BASE_URL` (host/WSL default: `http://127.0.0.1:20128`)
- `AGENT_BRAIN_GATEWAY_SECRET_FILE` (restricted file reference; never commit its value)
- `AGENT_BRAIN_CLI_KIND`
- `AGENT_BRAIN_ROUTE_MODEL`
- `AGENT_BRAIN_TASK_CAPACITY_TIER`

Readiness must include the selected protocol/model. A missing secret reference, unavailable gateway or unsupported route prevents launch; no alternate router is started.

## Verify the control plane (no inference)

```bash
docker compose -f multica-auth-work/docker-compose.selfhost.yml ps
curl -fsS 127.0.0.1:8080/health
openspec validate build-omniroute-agent-brain --strict
```

Do not use a live provider/inference request as a basic stack-health check. Create Kanban work only after Main Brain reports ready and the approved OmniRoute deployment is ready.

## Layout

```text
multica-auth-work/                  product backend, frontend, clients and daemon
openspec/changes/build-omniroute-agent-brain/  active Main Brain contract
docs/deploy/                       OmniRoute-only rollout and rollback runbooks
.deploy-control/                   preserved execution/evidence history
scripts/                            orchestration and non-inference validation tools
```
