# Runbook: PostgreSQL Least Privilege Demotion & 4-Role Ownership Model (ORQ-60)

## Executive Summary
This runbook defines the operational procedure to eliminate application `SUPERUSER` privileges from PostgreSQL (`multica_transition`) and establish a fail-closed, least-privilege four-role database ownership and recovery model.

Discovered during the **ORQ-35** production security audit, application connection role `multica_transition` currently possesses `SUPERUSER`, `CREATEROLE`, `CREATEDB`, `BYPASSRLS`, and `REPLICATION` flags. This runbook details the zero-queue maintenance window execution plan, role separation, privilege inventory, credential provisioning contracts, verification steps, and non-SQL fail-safe rollback procedures.

---

## 1. Four-Role Ownership & Recovery Architecture

To separate administrative recovery, schema DDL operations, and runtime DML execution, the database security posture is split into four distinct role tiers:

| Role Name | Type | Login | Purpose & Scope | Security Attributes |
|:---|:---|:---|:---|:---|
| `multica_recovery` | Recovery / Emergency Authority | `LOGIN` (Peer-Mapped) | Proven recovery superuser authority for maintenance and emergency rollback. Reachable ONLY via ORQ-35 OS Peer Map (mapping container OS user `postgres` / socket user) or secret-backed handoff. | `SUPERUSER`, `CREATEROLE`, `CREATEDB`, `BYPASSRLS`, `REPLICATION` |
| `multica_owner` | Group / Schema Owner | `NOLOGIN` | Owns all schema objects across all object classes (tables, partitions, views, matviews, sequences, functions, procedures, types). | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |
| `multica_migrator` | Migration Runner | `NOLOGIN` (Pre-Provision) | Used by `cmd/migrate` during maintenance windows to execute DDL migrations. Granted `multica_owner`. | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |
| `multica_transition` / `multica_app` | Runtime Backend App | `LOGIN` | Used by `multica-server` at runtime. Granted minimal required DML (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, sequence `USAGE`). | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |

---

## 2. Content-Free Inventory of Required Object Privileges

### Database & Schema Privileges
- `CONNECT, TEMPORARY` on database `multica_transition` to `multica_app`, `multica_transition`, `multica_migrator`.
- `USAGE` on schema `public` to `multica_app`, `multica_transition`, `multica_migrator`.
- `CREATE` on schema `public` to `multica_owner`, `multica_migrator`.

### Object Class Level Privileges (Schema `public`)
- **Tables, Partitions, Views & MatViews**: `SELECT, INSERT, UPDATE, DELETE` to `multica_app`, `multica_transition`.
- **Sequences**: `USAGE, SELECT, UPDATE` to `multica_app`, `multica_transition`.
- **Functions & Procedures**: `EXECUTE` to `multica_app`, `multica_transition`.

### Default Privileges for Future Objects
All objects created in schema `public` by `multica_owner` or `multica_migrator` automatically grant:
- `SELECT, INSERT, UPDATE, DELETE` on tables to `multica_app`, `multica_transition`.
- `USAGE, SELECT, UPDATE` on sequences to `multica_app`, `multica_transition`.
- `EXECUTE` on functions to `multica_app`, `multica_transition`.

---

## 3. Command Line Hygiene & Credential Provisioning Contract

1. **Zero Secret Command Line Rule**: Never embed passwords or `<SECRET>` placeholders in command-line arguments or process invocations (`ps aux` visible).
2. **Environment Variable Handoff**: Pass connection strings and credentials exclusively via environment variables:
   - `TEST_DATABASE_URL`
   - `DATABASE_URL`
   - `MIGRATOR_DB_PASSWORD`
   - `APP_DB_PASSWORD`
3. **Metadata Registration**: Record credential rotation status on issue metadata using `multica issue metadata set` (metadata keys only, zero plaintext secret values).

---

## 4. Execution Order of Operations & Preflight Checklist

> [!IMPORTANT]
> To prevent transaction failure mid-flight, all privileged bootstrap steps (recovery role creation, owner setup, dynamic ownership transfer, DML grants, default privileges) MUST execute BEFORE demoting `multica_transition` / `multica_app`. Demotion occurs LAST.

### Preflight Verification Steps
1. **Zero Active Queue**: Confirm zero pending/running tasks in the platform.
2. **Deploy Safety Gate (ORQ-57)**: Verify that the safe Docker Compose wrapper is loaded with `/home/ec2-user/.config/multica-transition/dev.env`.
3. **Database Backup**: Execute a physical/logical snapshot of container `multica-dev-transition-postgres-1` database `multica_transition` prior to cutover.

### Step-by-Step Execution Command

```bash
docker exec -i multica-dev-transition-postgres-1 psql -U multica_transition -d multica_transition < scripts/ops/least_privilege_cutover.sql
```

---

## 5. Post-Cutover Audit & Verification

### Step 5.1: Execute Verification Script

```bash
docker exec -i multica-dev-transition-postgres-1 psql -U multica_transition -d multica_transition < scripts/ops/least_privilege_verification.sql
```

Expected Output:
- `violating_superuser_app_roles` = `0`
- `multica_recovery` = `rolsuper=true`, `rolcanlogin=true` (LOGIN / Peer-mapped only)
- `schema_owner` = `multica_owner`
- `non_owner_count` = `0` across all object classes (tables, views, matviews, sequences, functions, procedures, types)

### Step 5.2: Execute Go Security Test Suite

```bash
TEST_DATABASE_URL="${CONTAINER_APP_DB_URL}" \
PATH=/home/ec2-user/goroot/go/bin:$PATH \
go test ./multica-auth-work/server/cmd/migrate -run TestLeastPrivilege -v
```

---

## 6. Fail-Safe Rollback Procedure

If unexpected application permission errors occur post-cutover:

### Non-SQL Signal & Filesystem Recovery (ORQ-35 Alignment)
If authentication rules block connection, perform OS-level signal reload as container process owner:

```bash
docker exec multica-dev-transition-postgres-1 cp /var/lib/postgresql/data/pg_hba.conf.bak /var/lib/postgresql/data/pg_hba.conf
docker exec multica-dev-transition-postgres-1 pg_ctl reload -D /var/lib/postgresql/data
```

### SQL Privilege Rollback via OS Peer Map
To restore previous superuser status using the ORQ-35 OS peer-mapped recovery session:

```bash
docker exec -i -u postgres multica-dev-transition-postgres-1 psql -U multica_recovery -d multica_transition < scripts/ops/least_privilege_rollback.sql
```

### Peer Map & SCRAM Client Authentication (ORQ-35 / ORQ-60 Hardening)

To eliminate unauthenticated local/loopback `trust` bypasses while preserving recovery authority access, PostgreSQL client authentication is configured via `pg_hba.conf` and `pg_ident.conf`:

#### `pg_ident.conf` Configuration
```conf
# Map OS process owner 'postgres' to DB recovery authority role 'multica_recovery'
recovery_map    postgres                multica_recovery
```

#### `pg_hba.conf` Configuration
```conf
# 1. OS Peer Map Confinement for Recovery Authority (local unix socket only)
local   all             multica_recovery                          peer    map=recovery_map

# 2. Local Unix Socket Access for Application & Migration Roles (SCRAM required)
local   all             all                                       scram-sha-256

# 3. Loopback & Network Access (SCRAM required)
host    all             all               127.0.0.1/32            scram-sha-256
host    all             all               ::1/128                 scram-sha-256
host    all             all               all                     scram-sha-256
```

---

## 7. Coordination Matrix

| Gate / Dependency | Target State | Coordination Action |
|:---|:---|:---|
| **ORQ-35 (SCRAM & HBA)** | SCRAM-SHA-256 local/TCP auth & OS peer map enforcement | Synchronize `multica_recovery` NOLOGIN peer-mapping with `pg_hba.conf` and `pg_ident.conf` cutover. |
| **ORQ-57 (Deploy Gate)** | Mandatory env-file wrapper & fail-closed recreate | Ensure `multica_migrator` credentials and `DATABASE_URL` are supplied via the safe deploy wrapper. |
| **ORQ-58 (Canonical Rebuild)** | Canonical image rebuild & deployment | Validate backend binary operates under non-superuser `multica_transition` / `multica_app` credentials before release. |

---

## 8. Traceability & Authoritative Evidence Citations

- **Target Issue**: ORQ-35 (`3f73ff90-55a1-4c2f-a52d-d3735580ce7e`)
- **Authoritative Remote Branch**: `agent/gemini-3-6-flash-b/6b400612`
- **Implementation Base SHA (40-hex)**: `cde2e0898b18065658340d14bc3a71c3e354dcd2`
- **Verification Evidence**: Empirically proven via Go test suite `multica-auth-work/server/cmd/migrate/peer_hba_test.go` on PostgreSQL 17.10 cluster (`/usr/bin/postgres`).
