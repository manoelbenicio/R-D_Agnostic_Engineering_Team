# Fresh Environment Restart Runbook

This runbook reconstructs the repository and isolated DEV stack without rediscovering prior work or touching the legacy environment. Commands assume Linux/WSL, user `dataops-lab`, and repository location `/home/dataops-lab/R-D_Agnostic_Engineering_Team`. Adjust the home prefix consistently if the target user differs.

## 1. Preconditions

Required tools:

- Git with authenticated access to the remote repository.
- Docker Engine and Docker Compose v2.
- OpenSSL.
- Node.js 22 and pnpm 10.28.2 for host frontend validation.
- Go 1.26.1 or Docker access to `golang:1.26.1` for backend validation.
- OpenSpec CLI.
- Sufficient disk for source, package cache, Docker images, PostgreSQL data, and backups.

Owner inputs that may be required:

- GitHub access.
- Encrypted transfer of runtime backups if database/uploads continuity is required.
- Encrypted transfer or target issuance of the OmniRoute stable key.
- Explicit decision whether Redis is intentionally omitted or provisioned.
- Explicit authorization before any live provider authentication or credential mutation.

## 2. Clone and verify Git

```bash
cd /home/dataops-lab
git clone https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
cd R-D_Agnostic_Engineering_Team
git fetch --all --tags --prune
git switch --create integration/dev-transition-candidate-20260719 \
  --track origin/integration/dev-transition-candidate-20260719
git status --short --branch
```

Expected:

- branch tracks `origin/integration/dev-transition-candidate-20260719`;
- working tree is clean;
- no unexpected untracked source files.

Verify immutable anchors:

```bash
git cat-file -e b6571299b00c8e388abefe7ef9dcbcf8ac715d7f^{commit}
git cat-file -e 29056e5aaf52d1e10fbfa0745b69b66685febd54^{commit}
git cat-file -e 043e6415fdb7f3ecc7ebcabce076c6e1f970981c^{commit}
git cat-file -e 6a2aba3550aaf6b0468a37bfdf2f00c7faaae084^{commit}
git rev-parse dev-deploy-20260719-candidate^{}
git branch -r | sort
git tag -l 'dev-freeze-*' 'dev-deploy-*' | sort
```

Expected deployed tag resolution:

```text
6a2aba3550aaf6b0468a37bfdf2f00c7faaae084
```

## 3. Read transition authority before editing

```bash
sed -n '1,240p' docs/transition/README.md
sed -n '1,320p' docs/transition/DEV_RESTART_DOSSIER_20260719.md
sed -n '1,360p' docs/transition/DOCKER_AND_REDIS_INVENTORY_20260719.md
sed -n '1,320p' docs/transition/SECRETS_AND_ACCESS_REGISTER_20260719.md
```

Then read repository instructions:

```bash
cat multica-auth-work/AGENTS.md
cat multica-auth-work/CLAUDE.md
```

Do not start an agent team before this reading is complete.

## 4. Recount OpenSpec and planning state

```bash
openspec list
for file in openspec/changes/*/tasks.md; do
  change=${file#openspec/changes/}
  change=${change%/tasks.md}
  done_count=$(rg -c '^\s*- \[x\]' "$file" || true)
  open_count=$(rg -c '^\s*- \[ \]' "$file" || true)
  printf '%s complete=%s open=%s\n' "$change" "${done_count:-0}" "${open_count:-0}"
done
openspec validate build-omniroute-agent-brain --strict
openspec validate persist-prodex-runtime-integration --strict
```

Expected Agent Brain count at this snapshot: 53 complete, 43 open, 96 total.

If counts differ, inspect Git history and record why before changing any checkbox.

## 5. Choose source mode

### Reproduce the exact deployed binaries

```bash
git switch --detach dev-deploy-20260719-candidate
```

Use this for binary reproduction or rollback comparison.

### Continue engineering

```bash
git switch integration/dev-transition-candidate-20260719
git pull --ff-only
```

Use this for new work. Documentation commits after `6a2aba3` do not alter the deployed application source, but every new change must receive a new build identity.

## 6. Prepare owner-only runtime configuration

```bash
install -d -m 700 /home/dataops-lab/.config/multica-transition
umask 077
DB_PASSWORD="$(openssl rand -hex 24)"
JWT_SECRET="$(openssl rand -hex 48)"
cat > /home/dataops-lab/.config/multica-transition/dev.env <<EOF
POSTGRES_DB=multica_transition
POSTGRES_USER=multica_transition
POSTGRES_PASSWORD=$DB_PASSWORD
POSTGRES_PORT=15433
BACKEND_PORT=18080
FRONTEND_PORT=13100
JWT_SECRET=$JWT_SECRET
FRONTEND_ORIGIN=http://localhost:13100
GOOGLE_REDIRECT_URI=http://localhost:13100/auth/callback
MULTICA_APP_URL=http://localhost:13100
APP_ENV=development
MULTICA_LOCAL_AUTH_BYPASS=true
MULTICA_LOCAL_AUTH_EMAIL=owner@local.test
VERSION=dev-transition
COMMIT=6a2aba3
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
chmod 600 /home/dataops-lab/.config/multica-transition/dev.env
```

Create the non-secret image override:

```bash
cat > /home/dataops-lab/.config/multica-transition/images.yml <<'EOF'
services:
  backend:
    image: multica-backend:transition-6a2aba3
  frontend:
    image: multica-web:transition-6a2aba3
EOF
chmod 600 /home/dataops-lab/.config/multica-transition/images.yml
```

Do not print `dev.env`. Validate only key names and modes:

```bash
stat -c '%a %U:%G %n' /home/dataops-lab/.config/multica-transition/{dev.env,images.yml}
awk -F= '/^[A-Za-z_][A-Za-z0-9_]*=/ {print $1}' \
  /home/dataops-lab/.config/multica-transition/dev.env | sort
```

If continuity of the existing database/JWT identity is required, do not regenerate. Securely transfer the existing `dev.env` instead.

## 7. Verify ports and Docker ownership

```bash
ss -ltn '( sport = :13100 or sport = :18080 or sport = :15433 )'
docker ps -a --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
```

The target ports must be free or intentionally assigned to the candidate project. Never reuse legacy ports 3100, 8080, or 5433 during blue/green validation.

## 8. Validate Compose graph

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  config --services

docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  config --images
```

Expected services: `postgres`, `backend`, `frontend`.

Expected candidate images:

- `multica-backend:transition-6a2aba3`
- `multica-web:transition-6a2aba3`
- `pgvector/pgvector:pg17`

## 9. Restore runtime state or start fresh

### Option A: fresh DEV database

Proceed directly to section 10. The backend entrypoint applies all migrations.

### Option B: restore transferred DEV database and uploads

Verify backup checksums first:

```bash
cd /home/dataops-lab/.local/share/multica-transition/backups
sha256sum -c SHA256SUMS-20260719T233100-0300
```

Start only PostgreSQL:

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  up -d postgres
```

Wait for `healthy`, copy the custom-format archive into the database container, then restore before starting the backend. `pg_restore` requires a seekable custom-format input; do not rely on `/dev/stdin`.

```bash
docker cp \
  /home/dataops-lab/.local/share/multica-transition/backups/postgres-20260719T233100-0300.dump \
  multica-dev-transition-postgres-1:/tmp/postgres-transition.dump
docker exec multica-dev-transition-postgres-1 \
  pg_restore -U multica_transition -d multica_transition \
  --clean --if-exists --no-owner /tmp/postgres-transition.dump
```

Restore uploads into the candidate volume only:

```bash
docker run --rm \
  -v multica-dev-transition_backend_uploads:/data \
  -v /home/dataops-lab/.local/share/multica-transition/backups:/backup:ro \
  alpine:3.21 \
  sh -c 'tar -xzf /backup/uploads-20260719T233100-0300.tar.gz -C /data'
```

For a production-sized database, test restore into a second disposable project/database first.

## 10. Build and start candidate stack

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  up -d --build
```

Do not omit `-p multica-dev-transition`; the base Compose declares `name: multica`, which could otherwise target the legacy project.

## 11. Verify candidate runtime

```bash
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  ps

curl -fsS http://127.0.0.1:18080/health
curl -fsS -o /dev/null -w 'frontend=%{http_code}\n' http://127.0.0.1:13100/login

docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  logs --tail=100 --no-color
```

Expected:

- PostgreSQL healthy.
- Backend HTTP 200 and `{"status":"ok"}`.
- Frontend HTTP 200.
- Migration output completes through the repository’s latest migration.
- Backend logs state in-memory realtime/local rate limiting until Redis is intentionally configured.
- No secret values appear in logs.

## 12. Redis decision and optional implementation

Do not point candidate Multica at `deploy-redis-1` merely because port 6379 exists.

Before adding Redis, record `REDIS-01` with:

- service owner;
- dedicated vs shared decision;
- image digest;
- ACL user and secret reference;
- persistent volume and backup;
- database/key-prefix isolation;
- healthcheck and restart behavior;
- outage/failover behavior;
- rotation owner;
- tests for realtime, liveness, cache, and rate limiting.

The current self-host Compose does not pass `REDIS_URL`; a reviewed Compose change is required. Keep secret-bearing URLs outside Git.

## 13. OmniRoute and host daemon readiness

### Verify OmniRoute without exposing credentials

```bash
docker ps --filter name=omniroute
curl -fsS http://127.0.0.1:20128/health || true
```

Use the installed health contract defined by source/evidence; do not assume an arbitrary endpoint if the first probe differs.

### Provision the Agent Brain secret reference

Required target path:

```text
/etc/agent-brain/secrets/omniroute-inference-key
```

Only the owner/security operator may populate its value. After authorized provisioning, verify ownership/mode without printing content.

### Host daemon environment names

- `AGENT_BRAIN_DEVELOPMENT_ENABLED`
- `AGENT_BRAIN_CONTROL_URL`
- `AGENT_BRAIN_GATEWAY_REQUIRED`
- `AGENT_BRAIN_GATEWAY_BASE_URL`
- `AGENT_BRAIN_GATEWAY_SECRET_FILE`
- `AGENT_BRAIN_GATEWAY_READINESS_POLICY`
- `AGENT_BRAIN_TASK_CAPACITY_TIER`
- `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED`

Do not enable gateway-required execution until the secret path, readiness, selected route/protocol, and credentialless child environment pass.

Prodex must remain default-OFF and cannot be active simultaneously with OmniRoute.

## 14. Re-run source validation after any code change

### Frontend

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
corepack enable
corepack prepare pnpm@10.28.2 --activate
pnpm install --frozen-lockfile
pnpm typecheck
pnpm build
```

### Backend using Docker Go 1.26.1

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker run --rm --init \
  -v "$PWD":/src \
  -v /home/dataops-lab/go/pkg/mod:/go/pkg/mod \
  -w /src/server \
  -e GOPROXY=off -e GOSUMDB=off \
  golang:1.26.1 \
  sh -c 'go test ./internal/daemon && go vet ./internal/daemon/brain ./internal/daemon/gateway ./internal/daemon/runtimeenv ./internal/daemon/observability/e2e ./pkg/agent'
```

If the module cache is absent on a fresh host, permit dependency download according to the environment network policy, then rerun offline for reproducibility.

## 15. Agent orchestration restart protocol

Current transition snapshot intentionally has no subordinate agents running. Start new agents only after repository, OpenSpec, Docker, secrets, and health reconciliation.

For each agent, record before dispatch:

- stable agent name;
- exact OpenSpec task ID;
- branch/worktree;
- owned files/directories;
- prohibited files/hotspots;
- dependencies;
- evidence IDs and acceptance command;
- credential/network/live permissions;
- stop conditions;
- expected check-in cadence.

Mandatory rules:

- One task branch/worktree per independent lane.
- No overlapping file ownership.
- One serial integrator for central daemon/config/health/command hotspots.
- No agent may authorize or perform login/logout/rotation/session mutation.
- No live-provider work until owner/security gates permit it.
- Every completed task must include commit SHA, tests actually run, evidence path, non-claims, and push confirmation.
- Never restore stale panes as authority; disk/Git/OpenSpec are authoritative.

## 16. Create a new runtime backup

```bash
umask 077
BACKUP_DIR=/home/dataops-lab/.local/share/multica-transition/backups
install -d -m 700 "$BACKUP_DIR"
STAMP="$(date '+%Y%m%dT%H%M%S%z')"
docker exec multica-dev-transition-postgres-1 \
  pg_dump -U multica_transition -d multica_transition -Fc > \
  "$BACKUP_DIR/postgres-$STAMP.dump"
docker run --rm \
  -v multica-dev-transition_backend_uploads:/data:ro \
  alpine:3.21 \
  tar -czf - -C /data . > "$BACKUP_DIR/uploads-$STAMP.tar.gz"
chmod 600 "$BACKUP_DIR/postgres-$STAMP.dump" "$BACKUP_DIR/uploads-$STAMP.tar.gz"
sha256sum "$BACKUP_DIR/postgres-$STAMP.dump" "$BACKUP_DIR/uploads-$STAMP.tar.gz" > \
  "$BACKUP_DIR/SHA256SUMS-$STAMP"
chmod 600 "$BACKUP_DIR/SHA256SUMS-$STAMP"
```

Transfer backups only through an owner-approved encrypted channel.

## 17. Rollback

### Application rollback while retaining candidate data

The legacy stack remains available on ports 3100/8080/5433. Route users back to the legacy frontend/reverse proxy target, then stop candidate containers without deleting volumes:

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  stop
```

Do not use `down --volumes`.

### Source rollback/reproduction

```bash
git switch --detach dev-deploy-20260719-candidate
```

Rebuild the immutable deployment snapshot with the same candidate image names.

### Recovery from bad integration

Use the documented `dev-freeze-*` tags and `backup/dev-transition-*` refs to compare or recover. Never force-reset the shared candidate branch without owner approval. Create a new recovery branch from the required immutable commit.

## 18. Completion report required after restart

The restart operator must commit/push a scrubbed report containing:

- target host/environment identity without secrets;
- repository branch and exact commit;
- verified immutable refs;
- OpenSpec counts;
- toolchain versions;
- Compose project/services/images;
- port/network/volume map;
- secret path/mode checks only;
- backup checksum verification;
- migration result;
- backend/frontend health;
- Redis decision and status;
- OmniRoute/daemon status;
- active agent/task ownership;
- unresolved decisions/risks;
- explicit non-claims.

Only then resume implementation.
