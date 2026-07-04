# Shared State Backend - Postgres / Redis

Status: PRE-DEPLOY REQUIRED

## 1. Decision

Shared runtime and gateway state must use Postgres for durable state.

Redis may be used for ephemeral coordination, leases, queues, and short-lived
caches.

SQLite is forbidden for shared state.

## 2. Why SQLite Is Forbidden

SQLite/file state is acceptable only for single-node local state. This project
needs multi-agent, multi-session, gateway, ledger, approved accounts, and event
ingest paths. Prior small-scale SQLite locking already showed contention. The
new L2 state surface is larger and hotter.

## 3. Postgres Owns

- approved account registry;
- tenant/profile policy snapshots;
- gateway virtual key admin state if enabled;
- runtime event ledger;
- billing/spend/savings ledger;
- redeem attempt ledger;
- kill switch audit state;
- deployment approval records;
- migration history.

## 4. Redis May Own

- runtime leases;
- short-lived route health;
- rate/admission counters;
- event stream cursor cache;
- sidecar heartbeat cache;
- distributed lock with TTL.

Redis must not be the only source for durable audit.

## 5. Required Tables

Minimum durable tables:

```text
l2_runtime_policies
l2_approved_profiles
l2_runtime_sessions
l2_runtime_events
l2_redeem_attempts
l2_kill_switches
l2_deploy_approvals
l2_provider_capabilities
```

## 6. Migration Rules

- Every schema change has up/down migration.
- Migrations run before deploy.
- No destructive migration without owner approval.
- Rollback path must preserve audit rows.

## 7. Connection Rules

- Connection string comes from secret manager or env var.
- Never log connection string.
- Set statement timeout.
- Set connection pool max.
- Healthcheck must verify Postgres before readiness.

## 8. Readiness Gate

Sidecar readiness fails if:

- Postgres unavailable;
- migration version mismatch;
- configured backend is SQLite/file for shared state;
- Redis required but unavailable;
- kill switch state cannot be read.
