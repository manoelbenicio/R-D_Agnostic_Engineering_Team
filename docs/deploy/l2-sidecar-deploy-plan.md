# L2 Sidecar Deploy Plan

Status: DRAFT FOR OWNER REVIEW - PROD DEPLOY NO-GO

This plan describes how Multica Go will launch pinned `prodex` AS-IS as the near-term Rust L2 runtime plane. It is not approval to deploy. Real PROD deploy remains blocked until the owner has reviewed this package and `deploy_owner_approved: true` is recorded in `.deploy-control/evidence/status-board.md`.

## 1. Scope

Near-term topology:

```text
Multica Go daemon (L4 control plane, cold)
  - owns tenants, approved accounts, policy, budgets, lifecycle, aggregated metrics
  - launches/stops prodex process or sidecar
  - pushes desired state only
  - ingests events as observability/ledger only

prodex AS-IS (Rust L2 runtime plane, hot)
  - owns runtime proxy/gateway, profile/session affinity, pre-commit fallback
  - owns Smart Context shadow/canary/live mechanics
  - owns guarded redeem/reset-claim attempts
  - emits scrubbed runtime events

Postgres
  - durable shared state, policy snapshots, approved profiles, event ledger, kill switch audit

Redis, if enabled
  - ephemeral leases, cursors, counters, heartbeats; never the durable audit source
```

The single-router invariant is mandatory: once a session is started through prodex, Go must not invoke legacy Go rotation routing for that session.

## 2. Artifact Pinning

The rollout record must include the exact selected artifact:

```text
prodex_package=@christiandoxa/prodex
prodex_version=<owner-approved-version>
prodex_commit=<owner-approved-commit>
prodex_artifact_sha256=<sha256>
sbom_ref=<path-or-attestation-id>
attestation_ref=<path-or-attestation-id>
```

Before owner approval, verify and record:

- package source and license attribution;
- commit matches the reviewed fork map or AS-IS source;
- artifact SHA256 matches the reviewed artifact;
- SBOM, dependency audit, and gitleaks output are scrubbed and attached;
- no install step fetches an unpinned `latest` artifact in PROD.

## 3. Filesystem and Credential Invariants

Credential and profile material must never live on the 9p mount. The only allowed PROD credential/profile root is a real Linux filesystem such as ext4.

Required checks:

```text
PRODEX_HOME filesystem type is ext4 or approved Linux fs
CODEX_HOME per profile filesystem type is ext4 or approved Linux fs
auth/profile files mode is 0600
credential/profile directories mode is 0700
owner uid/gid matches the daemon runtime user
paths are not under /mnt/c, /mnt/wsl, /mnt/9p, or other shared host mounts
```

Fail closed if any profile home, `CODEX_HOME`, OAuth store, `auth.json`, cookie store, token cache, or prodex profile directory resolves onto 9p or has group/world-readable credential files.

## 4. State Backend

Postgres is required for shared runtime state. SQLite/file state is prohibited for shared product/runtime state.

Postgres durable state owns:

- approved account registry;
- tenant/profile policy snapshots;
- runtime sessions;
- runtime event ledger;
- spend/savings ledger;
- redeem attempt ledger;
- kill switch audit;
- deployment approval records;
- migration history.

Redis may be used only for ephemeral coordination. Redis outage handling must not lose durable audit rows.

Readiness fails closed when:

- Postgres is unavailable;
- migration version mismatches the reviewed deployment;
- any shared backend points to SQLite/file state;
- required Redis is unavailable;
- kill switch state cannot be read.

## 5. Required Environment Inventory

Record final names and source of each setting before approval. Do not print values.

```text
PRODEX_HOME
PRODEX_RUNTIME_LOG_DIR
PRODEX_SMART_CONTEXT_SHADOW=1
PRODEX_SMART_CONTEXT_CANARY_PERCENT=0
PRODEX_AUTO_REDEEM_ENABLED=0
PRODEX_GATEWAY_ENABLED=<approved-paths-only>
PRODEX_GATEWAY_POSTGRES_URL=<secret-manager-ref-or-env-name>
PRODEX_REDIS_URL=<secret-manager-ref-or-env-name-if-used>
MULTICA_L2_MODE=prodex_as_is
MULTICA_L2_SIDECAR_TOKEN=<ephemeral-generated-per-start>
MULTICA_L2_KILL_SWITCH_DEFAULT=enabled
MULTICA_L2_EVENT_STREAM_REQUIRED=1
MULTICA_L2_LOG_REDACTION=1
MULTICA_RAW_CODEX_ROLLBACK_ENABLED=1
```

Secret values must come from the approved secret boundary and must not appear in logs, traces, evidence, shell history, check-ins, screenshots, or runbook command output.

## 6. Startup Sequence

After the owner gate opens, Go startup must perform this sequence and stop on first failure:

1. Generate an ephemeral sidecar bearer token.
2. Validate `PRODEX_HOME`, `CODEX_HOME` profile roots, ownership, permissions, and filesystem type.
3. Verify prodex artifact pin, checksum, and attestation.
4. Verify Postgres migrations and reject shared SQLite.
5. Start prodex process/sidecar through the approved Multica launch path.
6. Check liveness (`/healthz` or AS-IS equivalent).
7. Check readiness (`/readyz` or AS-IS equivalent).
8. Apply policy with Smart Context shadow and auto-redeem disabled.
9. Register approved account/profile references only; never send secrets.
10. Open runtime event stream.
11. Confirm kill switch store is reachable.
12. Mark sidecar ready for controlled session smoke only.

## 7. Runtime Modes

Initial PROD mode:

```text
Smart Context: shadow
Canary percent: 0
Auto redeem: disabled
Gateway: enabled only for approved Codex paths
Runtime events: required
Redaction: required
Kill switch: enabled globally, tenant, provider, profile, and session scoped
```

Canary mode requires separate owner approval after clean shadow evidence:

```text
PRODEX_SMART_CONTEXT_SHADOW=0
PRODEX_SMART_CONTEXT_CANARY_PERCENT=1
```

Live Smart Context requires shadow evidence, canary evidence, QA sign-off, owner approval, and a successful kill switch test.

## 8. Deploy Gate Checklist

Real deploy can start only when all are true:

- owner has reviewed this deploy plan, rollout runbook, rollback runbook, and metrics/alerts;
- `.deploy-control/evidence/status-board.md` records `deploy_owner_approved: true`;
- artifact hash, prodex version/commit, rollback command, kill switch command, accepted risk, owner, and timestamp are recorded;
- Postgres migration and readiness checks pass;
- no shared SQLite/file state is configured;
- ext4 credential/profile invariant passes;
- redaction smoke passes with fake secret markers;
- kill switch smoke passes in a non-destructive path;
- raw Codex rollback path is preserved;
- event ingest can write durable audit rows;
- Go integration build/test gate has been satisfied by the owning stream.

## 9. Non-Goals

- No architecture change without ADR.
- No Go implementation of Smart Context or in-flight routing.
- No FFI and no subprocess-per-request target design.
- No real PROD deploy from this document alone.
