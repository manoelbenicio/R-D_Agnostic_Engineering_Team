# Database Backup and Restore Catalog

Snapshot: 2026-07-20 00:27 America/Sao_Paulo

Backup root:

```text
/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
```

The backup directory is outside Git, owned by `dataops-lab`, and mode 700. Every artifact and the checksum manifest are mode 600. Transfer the directory only through an owner-approved encrypted channel.

## 1. Coverage statement

The following repository and agent-work database stores are backed up:

- New candidate Multica PostgreSQL.
- Legacy rollback Multica PostgreSQL.
- OmniRoute SQLite state.
- Multica Grafana SQLite state.
- Kiro SQLite state/credential store.
- OpenCode SQLite agent/history state, including committed WAL content through SQLite online backup.
- Historical stopped P12 PostgreSQL data volume.
- Historical stopped P12 Redis data volume.
- Candidate and legacy uploads volumes.
- Complete OmniRoute data volume, including database, server state, logs, and existing internal backups.

External AOP, Chatwoot, HerdMaster, Docuseal, Hadoop, registry, and unrelated host databases are not represented as Multica-owned backups. They remain cataloged in `DOCKER_AND_REDIS_INVENTORY_20260719.md`. Their owners must authorize separate backups.

## 2. Artifact inventory

| Artifact | Type | Size at snapshot | Sensitivity | Restore verification |
|---|---|---:|---|---|
| `multica-candidate-postgres.dump` | PostgreSQL 17 custom dump | 231518 bytes | High: application/user/token hashes | Restored successfully into disposable PostgreSQL 17 |
| `multica-legacy-postgres.dump` | PostgreSQL 17 custom dump | 596053 bytes | High: application/user/token hashes | Restored successfully into disposable PostgreSQL 17 |
| `omniroute-storage.sqlite` | SQLite online backup | 3104768 bytes | Critical: route/account/client-key configuration may exist | In-memory restore + `quick_check=ok` |
| `multica-grafana.sqlite` | SQLite online backup | 1617920 bytes | High: Grafana users/session/config state | In-memory restore + `quick_check=ok` |
| `kiro-data.sqlite3` | SQLite online backup | 282624 bytes | Critical: Kiro authentication/session state | In-memory restore + `quick_check=ok` |
| `opencode.sqlite3` | SQLite online backup | 1965039616 bytes | Critical: agent history/config and potentially sensitive content | In-memory restore + `quick_check=ok` |
| `p12-postgres-volume.tar.gz` | Stopped PostgreSQL 17 raw volume | 7821758 bytes | High | Restored and started successfully in disposable pgvector/PostgreSQL 17 |
| `p12-redis-volume.tar.gz` | Stopped Redis 7 raw volume | 221 bytes | Potentially sensitive cache/state | Restored and started successfully in disposable Redis 7 |
| `omniroute-data-volume.tar.gz` | Full OmniRoute volume archive | 7073886 bytes | Critical: includes DB/server state/logs | Archive readable; 379 entries |
| `multica-candidate-uploads.tar.gz` | Candidate uploads archive | 87 bytes | Potential user content | Archive readable |
| `multica-legacy-uploads.tar.gz` | Legacy uploads archive | 87 bytes | Potential user content | Archive readable |
| `SHA256SUMS` | SHA-256 manifest | Owner-only | Fingerprints sensitive artifacts | `sha256sum -c` passed for every artifact |

The checksum values are intentionally kept in the owner-only manifest rather than copied into general documentation because several databases contain credentials or sensitive agent state.

## 3. Verification evidence

### PostgreSQL catalog and restore tests

| Backup | Archive catalog lines | Restored public tables | Restored migrations | Result |
|---|---:|---:|---:|---|
| Candidate | 498 | 68 | 163 | PASS |
| Legacy | 575 | 78 | 190 | PASS |
| Historical P12 raw volume | Raw volume | 67 | 161 | PASS |

Candidate and legacy dumps were restored with `pg_restore --no-owner` into separate databases inside a disposable `pgvector/pgvector:pg17` container. The verification container was removed afterward.

The stopped P12 volume was extracted into a disposable Docker volume, started using `pgvector/pgvector:pg17`, queried successfully, then destroyed. Source P12 containers/volumes were not modified.

### SQLite integrity and import tests

| Backup | Tables | `quick_check` | In-memory `.restore` |
|---|---:|---|---|
| Kiro | 7 | `ok` | PASS |
| OpenCode | 20 | `ok` | PASS |
| OmniRoute | 115 | `ok` | PASS |
| Grafana | 86 | `ok` | PASS |

SQLite backups were created with online backup APIs, not by copying a live main database/WAL pair:

- Host `sqlite3 .backup` for Kiro and OpenCode.
- OmniRoute’s bundled `better-sqlite3.backup()` for OmniRoute.
- `sqlite3 .backup` against the Grafana named volume.

### P12 Redis test

- Raw stopped volume extracted successfully.
- Redis 7 started from the restored volume.
- `PING` returned `PONG`.
- `DBSIZE` returned 0 at this snapshot.
- Disposable verification container and volume were removed.

## 4. Verify after encrypted transfer

```bash
BACKUP_DIR=/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
stat -c '%a %U:%G %n' "$BACKUP_DIR" "$BACKUP_DIR"/*
(cd "$BACKUP_DIR" && sha256sum -c SHA256SUMS)
```

Expected modes:

- directory: 700;
- artifacts and `SHA256SUMS`: 600.

Do not print or publish the checksum manifest.

### Encrypted transport to the new environment

Use SSH/SFTP/rsync over an owner-approved encrypted connection. Substitute the real target host and user:

```bash
SOURCE=/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
TARGET_USER=<new-environment-user>
TARGET_HOST=<new-environment-host>
ssh "$TARGET_USER@$TARGET_HOST" \
  'install -d -m 700 ~/.local/share/multica-transition/backups/full-20260720T002750-0300'
rsync -a --chmod=F600,D700 \
  "$SOURCE/" \
  "$TARGET_USER@$TARGET_HOST:~/.local/share/multica-transition/backups/full-20260720T002750-0300/"
```

On the target, verify the owner-only checksum manifest before any import. Do not upload this backup set to GitHub because it contains application data, credentials, session state, logs, and agent history.

## 5. Restore candidate PostgreSQL

Prepare the isolated candidate PostgreSQL service first, without starting the backend:

```bash
cd /home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  up -d postgres
```

Wait for healthy, then restore from the seekable archive:

```bash
BACKUP_DIR=/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
docker cp "$BACKUP_DIR/multica-candidate-postgres.dump" \
  multica-dev-transition-postgres-1:/tmp/multica-candidate.dump
docker exec multica-dev-transition-postgres-1 \
  pg_restore -U multica_transition -d multica_transition \
  --clean --if-exists --no-owner /tmp/multica-candidate.dump
```

Then start backend/frontend. The backend entrypoint applies migrations newer than the restored snapshot.

Validation:

```bash
docker exec multica-dev-transition-postgres-1 \
  psql -U multica_transition -d multica_transition -Atc \
  "select count(*) from schema_migrations;"
curl -fsS http://127.0.0.1:18080/health
```

## 6. Restore legacy Multica PostgreSQL into an isolated project

Do not overwrite the currently running legacy database. Create a separate PostgreSQL 17 container or Compose project, copy `multica-legacy-postgres.dump`, and run:

```bash
pg_restore -U <target-user> -d <target-database> \
  --clean --if-exists --no-owner /path/multica-legacy-postgres.dump
```

Expected restored snapshot: 78 public tables and 190 migration rows.

Use a new target password. The dump does not require preserving the original role password when restored with `--no-owner`.

## 7. Restore candidate/legacy uploads

Candidate:

```bash
BACKUP_DIR=/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
docker run --rm \
  -v multica-dev-transition_backend_uploads:/data \
  -v "$BACKUP_DIR":/backup:ro \
  alpine:3.21 \
  sh -c 'tar -xzf /backup/multica-candidate-uploads.tar.gz -C /data'
```

Legacy archive uses `multica-legacy-uploads.tar.gz`. Restore it only into a new dedicated target volume unless rollback recovery explicitly requires the existing legacy volume.

## 8. Restore OmniRoute

### Preferred full-volume recovery

Stop the target OmniRoute service before restore. Create its volume and extract:

```bash
BACKUP_DIR=/home/dataops-lab/.local/share/multica-transition/backups/full-20260720T002750-0300
docker volume create omniroute-data
docker run --rm \
  -v omniroute-data:/data \
  -v "$BACKUP_DIR":/backup:ro \
  alpine:3.21 \
  sh -c 'tar -xzf /backup/omniroute-data-volume.tar.gz -C /data'
```

This archive may contain server configuration, client credentials, call logs, and historical backups. Treat it as critical secret material. Start only an approved digest-pinned OmniRoute image.

### SQLite-only recovery

Use `omniroute-storage.sqlite` when server configuration/logs should not be transferred. Place it as `/app/data/storage.sqlite`, owned by the container runtime UID/GID. Provision `server.env` and gateway keys separately through approved secret handling.

Before start:

```bash
sqlite3 /path/omniroute-storage.sqlite 'pragma quick_check;'
```

Expected: `ok`.

## 9. Restore Grafana

The backup contains Grafana’s SQLite database only. Dashboards and provisioning remain in the repository/observability configuration. Secrets must be regenerated or securely transferred separately.

For a new named volume:

```bash
docker volume create multica-observability_grafana_data
docker run --rm \
  -v multica-observability_grafana_data:/data \
  -v /path/to/backup:/backup:ro \
  alpine:3.21 \
  sh -c 'cp /backup/multica-grafana.sqlite /data/grafana.db && chown 472:0 /data/grafana.db'
```

Rotate/reissue the Grafana admin password rather than copying the permissive legacy `/mnt/c` secret file.

## 10. Restore Kiro SQLite

`kiro-data.sqlite3` is a credential/session store. Only the owner may authorize this restore.

1. Stop all Kiro processes using the target `XDG_DATA_HOME`.
2. Preserve any existing target database separately.
3. Install the backup at `<XDG_DATA_HOME>/kiro-cli/data.sqlite3`.
4. Set parent directory mode 700 and file mode 600.
5. Run `sqlite3 ... 'pragma quick_check;'` without querying authentication data.
6. Owner validates login/session state.

Do not merge two Kiro databases and do not copy this file into Git or general cloud storage.

## 11. Restore OpenCode SQLite

`opencode.sqlite3` is almost 2 GB and can contain detailed agent/session history and sensitive content.

1. Stop all OpenCode processes.
2. Verify at least 5 GB free space for restore and subsequent WAL growth.
3. Install as `$XDG_DATA_HOME/opencode/opencode.db` or the target runtime’s documented data path.
4. Set directory 700 and database 600.
5. Do not separately restore old `-wal`/`-shm`; the online backup already incorporates committed WAL state.
6. Run `sqlite3 opencode.db 'pragma quick_check;'`.

Expected table count at snapshot: 20.

## 12. Restore stopped P12 PostgreSQL/Redis

These are raw-volume archives, not application-level logical exports.

### PostgreSQL

- Required engine: PostgreSQL 17 with pgvector; source reported `PG_VERSION=17.10`.
- Source user/database: `multica` / `multica`.
- Extract archive into a new volume mounted at `/var/lib/postgresql/data`.
- Start `pgvector/pgvector:pg17` without reinitializing the volume.
- Do not change data ownership during extraction.

The restore test produced 67 public tables and 161 migration rows.

### Redis

- Required engine: Redis 7 Alpine-compatible.
- Extract archive into a new volume mounted at `/data`.
- Start Redis with `--dir /data` and the target authentication policy.
- Snapshot contained zero keys, but the archive is retained for complete provenance.

## 13. Create a new full backup

Use online logical/SQLite backup APIs. Never copy a live SQLite main file without accounting for WAL.

Minimum sequence:

1. `pg_dump -Fc` candidate and legacy PostgreSQL.
2. `sqlite3 .backup` Kiro, OpenCode, and Grafana.
3. OmniRoute `better-sqlite3.backup()` or a service-supported online backup.
4. Archive uploads and required volumes.
5. Set directory 700/files 600.
6. Create owner-only SHA-256 manifest.
7. Run PostgreSQL catalog/restore tests.
8. Run SQLite `quick_check` and `.restore` tests.
9. Record counts/results, never data values.

## 14. External database boundary

Not backed by this operation:

- AOP PostgreSQL and authenticated Redis.
- Chatwoot PostgreSQL/Redis; Chatwoot PostgreSQL was in a restart loop.
- HerdMaster historical PostgreSQL and observability stores.
- Docuseal storage.
- Hadoop namenode/datanode state.
- Docker registry data.

These are separate projects on the same Docker host. Backing them up requires their own application-consistent procedures, credentials, retention requirements, and owner approval. Their absence from this Multica backup set is explicit, not accidental.

## 15. Recovery acceptance checklist

- [x] Every Multica/Agent Brain/OmniRoute/agent SQLite or PostgreSQL store discovered was classified.
- [x] Candidate and legacy PostgreSQL logical dumps created.
- [x] Candidate and legacy PostgreSQL dumps restored successfully in disposable PostgreSQL 17.
- [x] Kiro, OpenCode, OmniRoute, and Grafana SQLite online backups created.
- [x] All SQLite backups passed `quick_check` and in-memory `.restore`.
- [x] P12 PostgreSQL and Redis raw volumes restored successfully in disposable containers.
- [x] Uploads and full OmniRoute volume archived.
- [x] All artifacts owner-only and checksum verified.
- [ ] Backup directory transferred to the new host through an approved encrypted channel.
- [ ] Restore repeated on the actual new host.
- [ ] Owner validates credential/session-bearing Kiro/OmniRoute state after authorized restore.
