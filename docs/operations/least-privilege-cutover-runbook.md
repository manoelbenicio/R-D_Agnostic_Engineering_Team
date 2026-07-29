# Runbook: PostgreSQL Least Privilege Demotion & 4-Role Ownership Model (ORQ-60)

## Executive Summary
This runbook defines the operational procedure to eliminate application `SUPERUSER` privileges from PostgreSQL (`multica_transition`) and establish a fail-closed, least-privilege four-role database ownership and recovery model.

Discovered during the **ORQ-35** production security audit, application connection role `multica_transition` currently possesses `SUPERUSER`, `CREATEROLE`, `CREATEDB`, `BYPASSRLS`, and `REPLICATION` flags. This runbook details the zero-queue maintenance window execution plan, role separation, privilege inventory, credential provisioning contracts, verification steps, and non-SQL fail-safe rollback procedures.

---

## 1. Four-Role Ownership & Recovery Architecture

To separate administrative recovery, schema DDL operations, and runtime DML execution, the database security posture is split into four distinct role tiers:

| Role Name | Type | Login | Purpose & Scope | Security Attributes |
|:---|:---|:---|:---|:---|
| `multica_recovery` | Recovery / Emergency Authority | `LOGIN` | Proven recovery superuser authority for maintenance, schema repairs, and emergency rollback. | `SUPERUSER`, `CREATEROLE`, `CREATEDB`, `BYPASSRLS`, `REPLICATION` |
| `multica_owner` | Group / Schema Owner | `NOLOGIN` | Owns all schema objects (tables, sequences, views, functions, types). | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |
| `multica_migrator` | Migration Runner | `LOGIN` | Used by `cmd/migrate` during maintenance windows to execute DDL migrations. Member of `multica_owner`. | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |
| `multica_transition` / `multica_app` | Runtime Backend App | `LOGIN` | Used by `multica-server` at runtime. Granted minimal required DML (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, sequence `USAGE`). | `NOSUPERUSER`, `NOCREATEROLE`, `NOCREATEDB`, `NOBYPASSRLS`, `NOREPLICATION` |

---

## 2. Content-Free Inventory of Required Object Privileges

### Database & Schema Privileges
- `CONNECT, TEMPORARY` on database `multica_transition` to `multica_app`, `multica_transition`, `multica_migrator`.
- `USAGE` on schema `public` to `multica_app`, `multica_transition`, `multica_migrator`.
- `CREATE` on schema `public` to `multica_owner`, `multica_migrator`.

### Object Level Privileges (Schema `public`)
- **Tables & Views**: `SELECT, INSERT, UPDATE, DELETE` to `multica_app`, `multica_transition`.
- **Sequences**: `USAGE, SELECT, UPDATE` to `multica_app`, `multica_transition`.
- **Functions & Routines**: `EXECUTE` to `multica_app`, `multica_transition`.

### Default Privileges for Future Objects
All objects created in schema `public` by `multica_owner` or `multica_migrator` automatically grant:
- `SELECT, INSERT, UPDATE, DELETE` on tables to `multica_app`, `multica_transition`.
- `USAGE, SELECT, UPDATE` on sequences to `multica_app`, `multica_transition`.
- `EXECUTE` on functions to `multica_app`, `multica_transition`.

---

## 3. Credential Provisioning Contract (LOGIN Roles)

`LOGIN` roles (`multica_migrator`, `multica_app`, `multica_recovery`) must be provisioned without exposing plaintext secrets in repository files or log streams.

1. **Environment Handoff**: Pass dynamic environment variables during maintenance window:
   - `MIGRATOR_DB_PASSWORD`
   - `APP_DB_PASSWORD`
   - `RECOVERY_DB_PASSWORD`
2. **Metadata Key Registration**: Pin high-signal credential rotation state to issue metadata using `multica issue metadata set` (metadata-only references, zero plaintext values).

---

## 4. Execution Order of Operations & Preflight Checklist

> [!IMPORTANT]
> To prevent transaction failure mid-flight, all privileged bootstrap steps (recovery role creation, owner setup, ownership transfer, DML grants, default privileges) MUST execute BEFORE demoting `multica_transition` / `multica_app`. Demotion occurs LAST.

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
- `valid_recovery_authority_roles` = `1` (`multica_recovery`)
- `schema_owner` = `multica_owner`

### Step 5.2: Execute Go Security Test Suite

```bash
TEST_DATABASE_URL="postgres://multica_transition:<SECRET>@127.0.0.1:15433/multica_transition?sslmode=disable" \
PATH=/home/ec2-user/goroot/go/bin:$PATH go test ./cmd/migrate -run TestLeastPrivilege -v
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

### SQL Privilege Rollback via `multica_recovery`
To restore previous superuser status using the proven recovery authority:

```bash
docker exec -i multica-dev-transition-postgres-1 psql -U multica_recovery -d multica_transition < scripts/ops/least_privilege_rollback.sql
```

---

## 7. Coordination Matrix

| Gate / Dependency | Target State | Coordination Action |
|:---|:---|:---|
| **ORQ-35 (SCRAM & HBA)** | SCRAM-SHA-256 local/TCP auth enforcement | Synchronize least-privilege role demotion with non-trust SCRAM authentication rules in `pg_hba.conf`. |
| **ORQ-57 (Deploy Gate)** | Mandatory env-file wrapper & fail-closed recreate | Ensure `multica_migrator` credentials and `DATABASE_URL` are supplied via the safe deploy wrapper. |
| **ORQ-58 (Canonical Rebuild)** | Canonical image rebuild & deployment | Validate backend binary operates under non-superuser `multica_transition` / `multica_app` credentials before release. |
