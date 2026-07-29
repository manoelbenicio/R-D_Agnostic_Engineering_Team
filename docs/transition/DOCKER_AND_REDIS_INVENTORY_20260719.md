# Docker and Redis Infrastructure Inventory

Snapshot: 2026-07-19 23:27 America/Sao_Paulo / 2026-07-20 02:27 UTC

## 1. Scope and interpretation

This document inventories every Docker container visible on the host at the snapshot. Presence on the same Docker engine does not mean a container belongs to this repository.

Ownership classes:

- **Candidate-owned:** created from the new `/home/dataops-lab/R-D_Agnostic_Engineering_Team` checkout.
- **Legacy-project:** belongs to the old Multica worktree under `/mnt/c` and is retained for rollback.
- **External/shared:** belongs to another project such as AOP, HerdMaster, or Chatwoot.
- **Orphan/historical:** has no current Compose ownership label or is stopped/created-only.

Never stop, recreate, inspect secret values, delete volumes, or change configuration for an external/shared container as part of the Multica transition without its owner’s authorization.

## 2. Candidate Multica DEV project

Docker project: `multica-dev-transition`

Compose working directory: `/home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work`

Network: `multica-dev-transition_default`

| Container | Image | State | Host binding | Persistence | Health interpretation |
|---|---|---|---|---|---|
| `multica-dev-transition-frontend-1` | `multica-web:transition-6a2aba3` | Running | `127.0.0.1:13100→3000` | None | No Docker healthcheck; `/login` returned 200 |
| `multica-dev-transition-backend-1` | `multica-backend:transition-6a2aba3` | Running | `127.0.0.1:18080→8080` | `multica-dev-transition_backend_uploads:/app/data/uploads` | No Docker healthcheck; `/health` returned 200 |
| `multica-dev-transition-postgres-1` | `pgvector/pgvector:pg17` | Running | `127.0.0.1:15433→5432` | `multica-dev-transition_pgdata:/var/lib/postgresql/data` | Docker health `healthy`; migrations reached 126 |

Restart policy for all three services: `unless-stopped`.

### Image identities

| Image tag | Local immutable image ID |
|---|---|
| `multica-web:transition-6a2aba3` | `sha256:29e4bc52c351443234ac593eefecb4a712e205f5c9602b66fc6d5add39a8952d` |
| `multica-backend:transition-6a2aba3` | `sha256:c8ba7dc56be057c9bff5e076d6020d7d550cf4406a344729defc1eba94057dd0` |
| `pgvector/pgvector:pg17` at snapshot | `sha256:dd467f03ca5c5581222490e5217e48a262864ccb659be559f8491bbafdc97da0` |

The application images are local builds, not published registry artifacts. A fresh environment must rebuild them from tag `dev-deploy-20260719-candidate` or the verified canonical branch.

### Compose inputs

- Repository base: `docker-compose.selfhost.yml`
- Repository build override: `docker-compose.selfhost.build.yml`
- Owner-local image-name override: `/home/dataops-lab/.config/multica-transition/images.yml`
- Owner-local environment: `/home/dataops-lab/.config/multica-transition/dev.env`

Docker labels on containers created during the initial launch may retain the original `/tmp/multica-transition-20260719.images.yml` path as creation provenance. That temporary path is not the operational source of truth. Persistent copies exist under `/home/dataops-lab/.config/multica-transition` and must be used for future commands.

## 3. Legacy Multica rollback project

Docker project: `multica`

Compose working directory: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work`

Network: `multica_default`

| Container | Image | State | Host binding | Persistence | Notes |
|---|---|---|---|---|---|
| `multica-frontend-1` | `multica-web:dev` | Running | `127.0.0.1:3100→3000` | None | Legacy frontend; `/login` returned 200 |
| `multica-backend-1` | `multica-backend:dev` | Running | `127.0.0.1:8080→8080` | `multica_backend_uploads` | Legacy backend; `/health` returned 200 |
| `multica-postgres-1` | `pgvector/pgvector:pg17` | Running/healthy | `127.0.0.1:5433→5432` | `multica_pgdata` | Legacy writable DB |

Restart policy: `unless-stopped`. Do not connect candidate code to this writable database.

## 4. OmniRoute

| Attribute | Observed value |
|---|---|
| Container | `omniroute` |
| Ownership | Standalone container; no Compose labels |
| Configured image | `diegosouzapw/omniroute:latest` |
| Snapshot image ID | `sha256:badb560971fdc23c2fb84b3e8695116239ff215b4cca4b07076201a8efae7f0d` |
| State | Running; Docker health `healthy` |
| Host binding | `0.0.0.0:20128→20128` |
| Networks | `bridge`, `multica_default` |
| Persistence | `omniroute-data:/app/data` |
| Restart | `unless-stopped` |
| Visible non-secret env keys | `OMNIROUTE_MEMORY_MB`, `OMNIROUTE_MIGRATIONS_DIR` |

Critical observations:

- The image tag is floating. `PD-02` requires an approved digest before cutover.
- Port 20128 is bound to all host interfaces, unlike the loopback-only application ports. Review firewall/exposure before any non-local environment.
- The container is not attached to `multica-dev-transition_default`.
- Host-side Agent Brain uses `http://127.0.0.1:20128` by default and can reach the current container.
- A containerized daemon would require either a shared Docker network or an explicitly approved host-gateway topology.
- The stable OmniRoute credential is not exposed in the container environment inventory shown here. Do not infer credential readiness from health alone.

## 5. Legacy Multica observability project

Docker project: `multica-observability`

Compose path: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/deploy/observability/docker-compose.yml`

Network: `multica-observability`

| Container | Image | State | Host binding | Persistence/config |
|---|---|---|---|---|
| `multica-grafana` | `grafana/grafana-oss:latest` | Running | `0.0.0.0:3005→3000` | `multica-observability_grafana_data`; bind-mounted provisioning/dashboards; password secret file |
| `multica-postgres-exporter` | `quay.io/prometheuscommunity/postgres-exporter:latest` | Running | `0.0.0.0:9187→9187` | Bind-mounted entrypoint and DB user/password secret files |
| `multica-prometheus` | `prom/prometheus:latest` | Running | `0.0.0.0:9090→9090` | `multica-observability_prometheus_data`; bind-mounted config/alerts |
| `multica-alertmanager` | `prom/alertmanager:latest` | Running | `0.0.0.0:9093→9093` | `multica-observability_alertmanager_data`; bind-mounted config |

All use `unless-stopped`; none had a Docker healthcheck. All four configured images use floating tags.

Security finding: the three bind-mounted secret files under the legacy `/mnt/c` tree were observed from Linux with mode 777. Do not copy them into the new repository. Recreate secrets in Linux owner-only storage and rotate as required before promoting observability.

## 6. Redis inventory and decision

There are three Redis-related containers on the host.

### 6.1 Active shared AOP Redis

| Attribute | Observed value |
|---|---|
| Container | `deploy-redis-1` |
| Owner/project | External AOP Compose project `deploy` |
| Compose path | `/mnt/c/VMs/Projects/AOP/deploy/docker-compose.yml` |
| Image | `redis/redis-stack-server:latest` |
| State | Running; Docker health `healthy` |
| Binding | `127.0.0.1:6379→6379` |
| Network | `deploy_aop_net` |
| Restart | `unless-stopped` |
| Authentication | Required; unauthenticated `PING` returned `NOAUTH Authentication required` |
| ACL username | Default Redis ACL user unless AOP configuration says otherwise |
| Password source | `REDIS_PASSWORD` in `/mnt/c/VMs/Projects/AOP/deploy/.env` |
| Persistence | No Docker mount was present in container inspection |

The actual password is intentionally not recorded here. The `.env` file was observed as mode 777 from Linux. This is an external secret and a security concern, not a value to copy into Multica.

Risks of reusing this Redis for Multica:

- It belongs to AOP, not this repository.
- Its lifecycle and password rotation are controlled elsewhere.
- No Docker data volume was visible; recreation can discard Redis state.
- The image is not digest-pinned.
- Shared keys/failure domains can couple AOP and Multica.
- Database-number separation alone is weaker than dedicated service ownership and key-prefix isolation.

### 6.2 Historical P12 Redis

| Attribute | Value |
|---|---|
| Container | `p12-prod-redis` |
| Image | `redis:7-alpine` |
| State | Exited successfully approximately 12 days before snapshot |
| Network | `bridge` |
| Persistence | Anonymous Docker volume mounted at `/data` |
| Compose ownership | None recorded |

Do not assume this is safe to resume. Its source configuration, credentials, and relationship to current code are not established.

### 6.3 Chatwoot Redis

| Attribute | Value |
|---|---|
| Container | `chatwoot-redis-1` |
| Image | `redis:alpine` |
| State | Created, not running |
| Owner | External `chatwoot` Compose project |
| Persistence | `chatwoot_redis_data:/data` |

This service is unrelated to Multica and must not be reused.

### 6.4 Multica candidate Redis behavior

The candidate backend container does not contain a `REDIS_URL` environment variable. Startup logs explicitly reported:

- realtime uses the in-memory hub in single-node mode;
- Redis-backed rate limiting is unavailable;
- a bounded local fallback is enabled.

The backend source supports Redis for realtime relays, liveness, caches, webhook limiting, and related state. The self-host Compose file does not currently pass `REDIS_URL` into the backend service.

Required decision `REDIS-01`:

1. **Documented single-node DEV:** keep `REDIS_URL` unset and accept non-distributed behavior; or
2. **Dedicated Multica Redis:** add an authenticated, persistent, health-checked service and wire a secret-bearing `REDIS_URL`; or
3. **Approved external Redis:** only after explicit AOP/Multica owner agreement, dedicated logical isolation/key prefixes, persistence/backup, credential rotation, availability ownership, and test evidence.

Recommended TO-BE: option 2 for any multi-node or production-like environment.

## 7. External AOP project

Compose project: `deploy`

Working directory: `/mnt/c/VMs/Projects/AOP/deploy`

Network: `deploy_aop_net`, except nginx uses host networking.

| Container | Image | State | Ports | Persistence |
|---|---|---|---|---|
| `deploy-nginx-1` | `nginx:latest` | Running | Host network | Bind-mounted templates |
| `deploy-redis-1` | `redis/redis-stack-server:latest` | Running/healthy | `127.0.0.1:6379` | No mount observed |
| `deploy-postgres-1` | Local image ID/tag `be400b50812a` | Running/healthy | `127.0.0.1:5432` | `deploy_aop_postgres_data` |
| `deploy-registry-1` | `registry:2` | Running | `127.0.0.1:5000` | Anonymous volume at `/var/lib/registry` |
| `deploy-namenode-1` | Hadoop 3.2.1 image | Running/healthy | `127.0.0.1:8020`, `127.0.0.1:9870` | `deploy_aop_hdfs_namenode` |
| `deploy-datanode-1` | Hadoop 3.2.1 image | Running/healthy | Internal `9864` | `deploy_aop_hdfs_datanode` |

AOP PostgreSQL identity observed: user `aop_dev`, database `aop`. Its password remains in the external AOP `.env` and is not a Multica credential.

## 8. External HerdMaster observability project

Compose project label: `observability`

Working directory: `/mnt/c/VMs/Projects/HerdMaster/deploy/observability`

Running services use host networking.

| Container | Image | State | Important mounts |
|---|---|---|---|
| `herdmaster-remediation` | `python:3.12-slim` | Running | `/home/dataops-lab/.config/herdmaster`, owner token, remediation source |
| `herdmaster-prometheus` | `prom/prometheus:latest` | Running | Generated config, alert rules, token, anonymous data volume |
| `herdmaster-grafana` | `grafana/grafana:latest` | Running | Generated dashboards and datasource config |
| `herdmaster-alertmanager` | `prom/alertmanager:latest` | Running | Generated config, anonymous data volume |
| `herdmaster-blackbox` | `prom/blackbox-exporter:latest` | Running | Blackbox configuration |
| `herdmaster-postgres` | `postgres:16` | Exited approximately three weeks | Anonymous data volume |

These containers are external to the repository transition.

## 9. External Chatwoot project

Compose path: `/mnt/c/VMs/Projects/Chatwoot/docker-compose.yaml`

| Container | Image | State | Persistence |
|---|---|---|---|
| `chatwoot-sidekiq-1` | `chatwoot/chatwoot:latest` | Created | `chatwoot_storage_data` |
| `chatwoot-rails-1` | `chatwoot/chatwoot:latest` | Created | `chatwoot_storage_data` |
| `chatwoot-redis-1` | `redis:alpine` | Created | `chatwoot_redis_data` |
| `chatwoot-base-1` | `chatwoot/chatwoot:latest` | Exited | `chatwoot_storage_data` |
| `chatwoot-postgres-1` | `pgvector/pgvector:pg16` | Restart loop | `chatwoot_postgres_data` |

The Chatwoot PostgreSQL restart loop is an existing external issue. Do not fix it as part of the Multica transition unless explicitly assigned.

## 10. Other historical/orphan containers

| Container | Image | State | Persistence/notes |
|---|---|---|---|
| `p12-prod-postgres` | Local image ID/tag `be400b50812a` | Exited approximately 12 days | Anonymous PostgreSQL volume |
| `docuseal` | `docuseal/docuseal` | Exited approximately two weeks | Bind mount `/mnt/c/VMs/Projects/Skills:/data` |

No current Compose ownership label was present.

## 11. Ports and collision map

| Port | Binding | Owner |
|---:|---|---|
| 13100 | loopback | Candidate frontend |
| 18080 | loopback | Candidate backend |
| 15433 | loopback | Candidate PostgreSQL |
| 3100 | loopback | Legacy frontend |
| 8080 | loopback | Legacy backend |
| 5433 | loopback | Legacy PostgreSQL |
| 20128 | all interfaces | Standalone OmniRoute |
| 3005 | all interfaces | Legacy Grafana |
| 9187 | all interfaces | Legacy PostgreSQL exporter |
| 9090 | all interfaces | Legacy Prometheus |
| 9093 | all interfaces | Legacy Alertmanager |
| 6379 | loopback | External AOP Redis |
| 5432 | loopback | External AOP PostgreSQL |
| 5000 | loopback | External AOP registry |
| 8020/9870 | loopback | External AOP Hadoop namenode |

All-interface bindings require firewall/exposure review in the target environment.

## 12. Operational verification commands

These commands reveal no secret values.

```bash
docker ps -a --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
docker network ls
docker volume ls
curl -fsS http://127.0.0.1:18080/health
curl -fsS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:13100/login
docker logs multica-dev-transition-backend-1 2>&1 | tail -100
```

To prove Redis authentication without revealing the password:

```bash
docker exec deploy-redis-1 redis-cli ping
```

Expected result for the external AOP Redis without credentials: `NOAUTH Authentication required`.

## 13. Restart and deletion safety

- Candidate restart commands are in `FRESH_ENV_RESTART_RUNBOOK.md`.
- Do not use `docker system prune`, `docker volume prune`, or project-wide destructive cleanup during transition.
- Do not use `down --volumes` for candidate, legacy Multica, AOP, Chatwoot, or observability projects without owner authorization and verified backups.
- Do not stop the legacy Multica stack until the owner accepts the candidate observation window.
- Do not assume `restart: unless-stopped` reconstructs missing bind-mounted configuration or secret files.
