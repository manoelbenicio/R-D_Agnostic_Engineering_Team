# L2 Sidecar Deploy Plan

Status: PRE-DEPLOY REQUIRED

## 1. Scope

Deploy `prodex` AS-IS pinned as the near-term Rust L2 runtime plane under
Multica Go orchestration.

No real PROD deploy may start until `prod-rollout-runbook.md` is presented to
the owner and approval is recorded.

## 2. Preconditions

Required:

- prodex package version and commit pinned;
- package integrity/attestation captured;
- Postgres state backend configured;
- no shared SQLite backend;
- kill switch configured;
- Smart Context starts in shadow or canary;
- logs scrubber enabled;
- rollback command documented;
- profile homes on POSIX filesystem with 600 permissions where applicable;
- Go daemon build/test green;
- sidecar health/readiness green.

## 3. Pinning

Record:

```text
prodex_npm_package=@christiandoxa/prodex
prodex_version=0.246.0
prodex_commit=7750da9b6a5c91a6d429e18e6a4d422cab4bc144
```

Before deploy, verify latest selected artifact matches the intended pin.

## 4. Runtime Topology

```text
Multica Go daemon
  -> starts local prodex sidecar or prodex runtime command
  -> pushes policy/account refs
  -> ingests runtime events
  -> exposes aggregated metrics

prodex/Rust L2
  -> runtime proxy/gateway
  -> profile/session affinity
  -> Smart Context
  -> redeem
  -> structured events

Postgres
  -> durable shared state and audit
```

## 5. Deployment Modes

Initial mode:

```text
Smart Context: shadow or low canary
Auto redeem: disabled unless explicit owner approval
Gateway: enabled only for approved paths
Kill switch: enabled and tested
Rollback: codex raw launch path preserved
```

## 6. Required Environment Variables

Names may be adapted by implementation, but the runbook must record the final
names.

```text
PRODEX_HOME
PRODEX_RUNTIME_LOG_DIR
PRODEX_SMART_CONTEXT_SHADOW
PRODEX_SMART_CONTEXT_CANARY_PERCENT
PRODEX_GATEWAY_TOKEN
PRODEX_GATEWAY_POSTGRES_URL
MULTICA_L2_SIDECAR_TOKEN
MULTICA_L2_KILL_SWITCH_DEFAULT
```

Do not print values.

## 7. Deploy Gate

Deploy can start only if:

- owner approval is recorded;
- `prod-rollout-runbook.md` accepted;
- rollback tested in non-destructive path;
- kill switch smoke passed;
- redaction smoke passed;
- Postgres readiness passed;
- sidecar readiness passed.
