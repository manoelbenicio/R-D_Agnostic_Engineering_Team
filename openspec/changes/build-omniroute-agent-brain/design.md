## Context

Main Brain is the host daemon that claims Multica tasks and preserves workspace,
repository/worktree, session, process, stream, cancellation and terminal-result semantics. Each
launch pins exactly one non-overlapping transport:

```text
project/squad/Kanban → Main Brain → [omniroute] → approved CLI → OmniRoute
                              └──→ [native_credential_home] → approved native CLI
                              └──→ terminal/session/result → Postgres → UI
```

For `omniroute`, OmniRoute exclusively owns routing, inference accounts, and credentials. For
`native_credential_home`, R3 exclusively resolves one approved opaque home for one existing
logical runtime/agent and OmniRoute is not used. Main Brain never translates or falls back
between these bindings.

## Goals / Non-Goals

**Goals**
- Preserve product/control-plane and daemon lifecycle behavior.
- Admit exactly one pinned `omniroute` or `native_credential_home` binding per launch.
- Keep OmniRoute readiness mandatory only for `omniroute`; keep native R3 home resolution,
  exclusivity, and health mandatory only for `native_credential_home`.
- Preserve controlled task homes, dynamic metadata-only catalog lifecycle, context/skills,
  worktrees, streaming, commit ledger, cancellation and terminal persistence.
- Provide bounded capacity and metadata-only observability without raw paths or account identity.

**Non-goals**
- Reimplement OmniRoute provider authentication, quota, account selection, retry or failover in
  Main Brain.
- Authenticate, provision, rotate, or mutate native source credential homes.
- Translate or fallback between transport bindings or native homes.
- Use a live inference request as routine repository validation.
- Rewrite product-neutral lifecycle code solely for naming.

## Decisions

### 1. One pinned transport binding

`TransportBinding` accepts exactly `omniroute` or `native_credential_home`; persisted or incoming
unknown values are rejected, never translated. `CLIKind` identifies the frontend executable.
For `omniroute`, `RouteModel` identifies an approved OmniRoute model and OmniRoute exclusively
owns provider/account selection and credentials. For `native_credential_home`, R3 resolves one
approved exclusive opaque home for one existing logical runtime/agent before launch; no
OmniRoute route or account lookup occurs.

### 2. Fail-closed admission

Main Brain first requires exactly one complete pinned binding. For `omniroute`, it validates its
restricted secret reference, gateway liveness/authenticated readiness, selected model, selected
protocol, and approved route before process launch. For `native_credential_home`, it validates
the R3 binding, existing logical runtime/agent, exclusive opaque assignment, catalog generation,
layout metadata, freshness, and health before process launch. Any missing, negative, stale,
ambiguous, conflicting, or unsupported state produces a bounded classification and no CLI
process. One binding is never used to recover the other.

### 3. Binding-scoped child boundary

Environment composition always starts from a sanitized allowlist, removes ambient provider
credentials and direct endpoint overrides, adds task/workspace context, validates custom
settings, and creates controlled per-task CLI configuration. In `omniroute`, trusted opaque
OmniRoute transport access is applied last and no native home reference is supplied. In
`native_credential_home`, no OmniRoute endpoint or secret is supplied; the daemon resolves the
pinned opaque `home_ref` internally and supplies only daemon-local isolated task-home references
required by the native CLI. Shared/source auth files are never copied. A global HOME, source raw
path, account identity, credential value, or provider response never enters product APIs,
events, logs, diagnostics, or evidence.

### 4. Preserve lifecycle ownership

Main Brain continues to own task slot admission, repo/worktree preparation, local-directory safety, process launch, watchdogs, cancellation, stream batching, commit ledger integration, session pinning, terminal result publication and resource release. Kanban/project/squad/issue and Postgres behavior is unchanged.

### 5. Recovery never changes transport

Each admitted session remains pinned to its transport binding. Its state is either
`NORMAL(<pinned binding>)` or `DEGRADED(none)` with new model admission closed. An OmniRoute
outage degrades only `omniroute`; stale/unhealthy/ambiguous R3 or catalog state degrades only
`native_credential_home`. Restore requires that same binding's strict readiness at a session
boundary. Repeated failure or operator action cannot promote, translate, rotate, or retry the
other binding or another native home.

### 6. Rollback preserves the invariant

Rollback holds new admissions and drains/cancels according to tool/commit safety. For
`omniroute`, it may select previous accepted Main Brain/OmniRoute revisions. For
`native_credential_home`, it may activate a previous accepted immutable configuration while
preserving the pinned exclusive opaque home and all source material. Rollback never switches
binding, resolves a replacement home, or copies, moves, deletes, truncates, sanitizes,
overwrites, chmods, or changes ownership of a source home. If the same binding cannot pass
strict readiness, admission remains closed.

### 7. Capacity and observability

Capacity tiers are explicit and evidence-based. Cancellation releases slots exactly once.
Metadata-only spans correlate ingress, queue, daemon admission, CLI process, the selected
transport hop (OmniRoute or native launch), terminal persistence and UI delivery without
prompts, tool payloads, repository content, credentials, raw paths or account identity.

### 8. Kanban is the dispatch authority

New executable work starts by assigning or following up on a product issue through the
supported Kanban API. Each logical activation must create exactly one `agent_task_queue` row.
Herdr may inspect panes and runtime state but must not launch, retry or duplicate product work.
The TL remains the sole priority, scope, acceptance and integration authority; the assigned
agent is the sole writer for its bounded branch/worktree/files.

A terminal task is an execution record, not proof that the issue is done. The TL reconciles
the result to `in_review`, `blocked` or `done` from evidence. Provider/auth/runtime failure is
reported as infrastructure failure and can be reassigned without misclassifying the code.

## Risks / Trade-offs

- **Selected transport is unavailable:** intentional fail-closed behavior; the other binding is
  not a fallback and non-model product/control-plane operations continue.
- **Compatibility input contains an old or ambiguous owner:** reject it with an actionable
  bounded error instead of translating or routing ambiguously.
- **Native catalog/source lifecycle:** metadata may be retired, but source homes are immutable;
  only task-local non-source material may be cleaned after active-reference checks.
- **Shared daemon file is large:** preserve existing lifecycle blocks and remove only obsolete
  ambiguous routing/credential branches; validate targeted packages.
- **No local Go toolchain:** use an available pinned build container if present; otherwise report
  the concrete validation blocker.

## Migration Plan

1. Delete only dedicated obsolete alternate-router packages, scripts and active plans; preserve
   the approved R3 `native_credential_home` authority and dynamic catalog.
2. Remove ambiguous startup/config/env/health/recovery consumers from shared daemon files.
3. Enforce exact binding identity, binding-specific admission and child environment, source-home
   preservation, and no cross-binding fallback.
4. Update tests, README, manifests and runbooks.
5. Run formatting, targeted unit tests/build, strict validation of both affected OpenSpecs, and
   a cross-authority residual scan without inference.
6. Before any future handoff/deploy, require a clean owned index/worktree, exact commit published
   to a configured non-main upstream with equal local/upstream SHA and 0/0 divergence, and
   deployment evidence pinned to that SHA. Any pending file or divergence fails closed.
