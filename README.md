# R&D — Agnostic Engineering Team (Multica + prodex)

Managed-agents platform. **Multica** (Go control plane / L4) launches **prodex**
(Rust data plane / L2, pinned `v0.246.0`) on the hot path — pre-commit rotation,
affinity, Smart Context / token-saver, reset-claim. Program name:
**Rotation-Parity Polyglot (RPP)**.

> The user-facing frontend is the **Multica web app** (Next.js), lives under
> [`multica-auth-work/`](./multica-auth-work). There is **no separate SPA** in
> this repo — the former AgentVerse SPA was removed (it belonged to another
> project). See `LEGACY_ARCHIVE_REFERENCE.md`.

## Layers

```
L4 (cold)  Multica — Go server (control plane, sessions/workspaces/kanban/auth)
L2 (hot)   prodex — Rust sidecar (rotation, Smart Context, reset-claim), contract rpp.l2.v1
Frontend   Multica web (Next.js)  ·  mobile (Expo)  ·  desktop (Electron)
Data       Postgres (pgvector)
```

Authoritative sources: `openspec/changes/rotation-parity-polyglot/`,
`.planning/`, `Diligencias/`, `docs/rotation-parity-polyglot/`.

## Components & where they run (verified local topology)

| Component | Source | Local (Docker) | Health check |
|-----------|--------|----------------|--------------|
| Multica backend (Go, L4) | `multica-auth-work/server` | container `multica-backend-1` → `127.0.0.1:8080` | `curl 127.0.0.1:8080/health` → `{"status":"ok"}` |
| Multica web (frontend) | `multica-auth-work/apps/web` | container `multica-frontend-1` → `127.0.0.1:3100` (internal 3000) | open http://localhost:3100 |
| Postgres (pgvector pg17) | image | container `multica-postgres-1` | `docker inspect` → `healthy` |
| prodex (Rust, L2) | `bin/prodex` (`multica-auth-work/prodex-sidecar`) | spawned per-session by the backend | `bin/prodex --version` → `prodex 0.246.0` |

## Run the stack (Docker required)

Docker **is** required — the backend, frontend, and Postgres run as containers.

```bash
cd multica-auth-work
cp .env.example .env          # edit JWT_SECRET at minimum
docker compose -f docker-compose.selfhost.yml up -d
```

Then open the **real app**: **http://localhost:3100** (not :5173 — that was the
removed SPA).

## Verify everything is 100% up

```bash
# 1) Containers up + Postgres healthy
docker compose -f multica-auth-work/docker-compose.selfhost.yml ps

# 2) Go control plane healthy
curl -s 127.0.0.1:8080/health          # expect {"status":"ok"}

# 3) Frontend serving
curl -s -o /dev/null -w '%{http_code}\n' 127.0.0.1:3100   # expect 200

# 4) Rust data plane binary present & correct pin
bin/prodex --version                    # expect prodex 0.246.0
```

All four green ⇒ control plane (Go), data plane (Rust), DB, and frontend are
healthy. Then you can create tasks on the kanban and assign agents from the
Multica web UI.

## Program state / dashboard (RPP planning)

```bash
openspec validate rotation-parity-polyglot
python3 scripts/dashboard/plan_dashboard.py --once --ascii
```

## Layout

```
multica-auth-work/   # THE PRODUCT — Go backend, Next.js web, mobile, desktop, prodex-sidecar
bin/prodex           # built Rust L2 binary (v0.246.0)
openspec/changes/    # rotation-parity-polyglot, rotation-router, agent-credential-isolation
docs/                # RPP/prodex/Multica architecture, contracts, deploy runbooks
.planning/           # GSD planning (PROJECT/REQUIREMENTS/ROADMAP/STATE/RCA)
Diligencias/         # charter, context, dependency/crate/env matrices, phases
.deploy-control/     # fleet board, check-ins, evidence
scripts/{smoke,deploy,dashboard}/  # RPP smoke tests, rollback/kill-switch, dashboards
```
