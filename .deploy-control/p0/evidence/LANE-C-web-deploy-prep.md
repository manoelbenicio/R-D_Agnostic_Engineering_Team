# LANE C — web deploy prep (orq1 frontend only) — BLOCKED on container runtime

- agent: `Codex56#A` · pane: `w7:p3` · task: `WEB-DEPLOY-PREP-FRONTEND`
- scope: **frontend service ONLY** (`multica-dev-transition-frontend-1` on orq1). NEVER backend/daemon/postgres/omniroute. No source edits.
- deploy freeze: build+tag + record rollback, then STOP and await LEAD go before recreate.

> STATUS: **BLOCKED** — pane `w7:p3` has no container runtime, so the image cannot be built/tagged here
> and the frontend cannot be recreated from this pane. No build/deploy attempted; no source edited.

## Environment (verified read-only)
- `docker`, `podman`, `nerdctl`, `docker-compose` → **all ABSENT**; no `DOCKER_HOST`/`DOCKER_CONTEXT`.
- `ssh` present, but orq1 is a remote host and the dispatch provided no orq1 ssh target/credentials.
- Root disk healthy (24G free, 52%).

## Build context (confirmed for whoever has the runtime)
- Web image build: `multica-auth-work/Dockerfile.web` (node:22-alpine, pnpm 10.28.2 multi-stage).
- Build service: `docker-compose.selfhost.build.yml` → `frontend` (`image: multica-web:dev`, `context: .`, `dockerfile: Dockerfile.web`).
  - Equivalent: `cd multica-auth-work && docker build -f Dockerfile.web -t multica-web:<new-tag> .`
- orq1 runtime service: `docker-compose.selfhost.yml` → `frontend`
  (`image: ${MULTICA_WEB_IMAGE:-ghcr.io/multica-ai/multica-web}:${MULTICA_IMAGE_TAG:-latest}`,
  ports `127.0.0.1:${FRONTEND_PORT:-3000}:3000`). orq1 maps `FRONTEND_PORT=13100` → verify `127.0.0.1:13100=200`.
- Compose project: `multica-dev-transition`; container `multica-dev-transition-frontend-1`.
- Frontend-only recreate (on go): `docker compose up -d --no-deps frontend`.
- **Rollback ref NOT capturable here** (needs `docker inspect multica-dev-transition-frontend-1` on orq1
  to record its current image id/tag before recreate).

## Blocker / owner / next action
- **Blocker:** no container runtime + no orq1 build/deploy access on pane `w7:p3`.
- **Owner:** LEAD (`w5:p1`) + orq1 host owner.
- **Next action (either):** (1) provision `docker` CLI + `DOCKER_HOST`/context to orq1 on this pane, or
  (2) execute on a pane that has the runtime + orq1 access. Then: build+tag `multica-web:<tag>`, capture
  the current frontend image ref for rollback, and report `READY: <new tag>, rollback=<old tag>`. The
  deploy freeze (await LEAD go) still applies before the frontend recreate.

## Non-claims
- No image built, no tag created, no container recreated, no deploy. No source edited (frontend fix +
  CreateChatSessionRequest export already in source per LEAD; tsc/eslint EXIT=0). Frontend-only scope
  held; backend/daemon/postgres/omniroute untouched. `ssh` not used (no orq1 coordinates provided).
