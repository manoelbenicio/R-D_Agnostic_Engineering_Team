# R&D — Agnostic Engineering Team

Managed-agent platform built around one pinned transport binding per launch:

```text
                                  ┌─ [omniroute] ──────────────→ OmniRoute → model
Multica web / Kanban → Main Brain ┤
                                  └─ [native_credential_home] → approved native CLI
                              ↓
                 terminal/session persistence
```

The bindings are mutually exclusive. With `omniroute`, OmniRoute is the **sole inference
router and account/credential owner**; native homes are forbidden and an unavailable route
fails closed. With `native_credential_home`, R3 resolves one approved exclusive opaque home for
one existing logical runtime/agent before launch, Main Brain supplies only daemon-local isolated
home references required by the native CLI, and OmniRoute is not used. Main Brain never
fallbacks, translates, rotates, or retries between bindings or native homes. Neither mode uses a
global HOME or exposes raw paths, account identity, or credential data through product APIs,
events, logs, or evidence. Source credential homes are never copied, moved, deleted, truncated,
overwritten, sanitized, chmodded, or ownership-changed; only task-local non-source material may
be cleaned after active-reference checks.

## Product components

| Component | Source | Responsibility |
|---|---|---|
| Multica backend | `multica-auth-work/server` | Workspaces, projects, squads, Kanban issues/tasks, API and persistence |
| Multica web | `multica-auth-work/apps/web` | User-facing Kanban/project/squad interface |
| Main Brain daemon | `multica-auth-work/server/internal/daemon` | Task lifecycle, worktrees, terminals, cancellation, binding-specific transport admission |
| OmniRoute | externally deployed service | Sole inference router/account/credential owner for `omniroute`; unused by `native_credential_home` |
| Postgres | self-host Compose service | Durable product and control-plane state |

The frontend is the Next.js app under `multica-auth-work/`; there is no separate AgentVerse SPA in this repository.

## Run the self-host product stack

Docker runs the backend, frontend and Postgres. OmniRoute is deployed and operated separately and
must be reachable by the host daemon before `omniroute` work is admitted; it is not contacted
for `native_credential_home` work.

```bash
cd multica-auth-work
cp .env.example .env   # configure required product values out of band
docker compose -f docker-compose.selfhost.yml up -d
```

Open <http://localhost:3100>. The backend health endpoint is <http://127.0.0.1:8080/health>.

## Configure Main Brain

For an `omniroute` binding, use the neutral Agent Brain gateway configuration surface:

- `AGENT_BRAIN_GATEWAY_REQUIRED=true`
- `AGENT_BRAIN_GATEWAY_BASE_URL` (host/WSL default: `http://127.0.0.1:20128`)
- `AGENT_BRAIN_GATEWAY_SECRET_FILE` (restricted file reference; never commit its value)
- `AGENT_BRAIN_CLI_KIND`
- `AGENT_BRAIN_ROUTE_MODEL`
- `AGENT_BRAIN_TASK_CAPACITY_TIER`

Readiness must include the selected protocol/model. A missing secret reference, unavailable
gateway, or unsupported route prevents an `omniroute` launch; native mode is never used as a
fallback. A `native_credential_home` launch instead requires the approved R3 opaque assignment,
fresh catalog generation and health for the existing logical runtime/agent. It receives no
OmniRoute endpoint/secret and fails closed rather than selecting a different home or gateway.

## Verify the control plane (no inference)

```bash
docker compose -f multica-auth-work/docker-compose.selfhost.yml ps
curl -fsS 127.0.0.1:8080/health
openspec validate build-omniroute-agent-brain --strict
```

Do not use a live provider/inference request as a basic stack-health check. Create Kanban work
only after Main Brain reports the selected binding ready: approved OmniRoute deployment ready
for `omniroute`, or approved R3 assignment/catalog readiness for `native_credential_home`.

## Layout

```text
multica-auth-work/                  product backend, frontend, clients and daemon
openspec/changes/build-omniroute-agent-brain/  active Main Brain contract
docs/deploy/                       binding-specific fail-closed rollout and rollback runbooks
.deploy-control/                   preserved execution/evidence history
scripts/                            orchestration and non-inference validation tools
```
