## Context

Main Brain is the host daemon that claims Multica tasks and preserves workspace, repository/worktree, session, process, stream, cancellation and terminal-result semantics. OmniRoute is an externally deployed gateway. The target operational flow is:

```text
project/squad/Kanban issue → queued task → Main Brain → approved CLI → OmniRoute
                                      ↘ terminal/session/result → Postgres → UI
```

No other component may own routing for a model request.

## Goals / Non-Goals

**Goals**
- Preserve product/control-plane and daemon lifecycle behavior.
- Make OmniRoute readiness a mandatory pre-launch gate.
- Remove provider-account selection, provider auth copying, direct-provider endpoints and alternate routing.
- Preserve controlled task homes, context/skills, worktrees, streaming, commit ledger, cancellation and terminal persistence.
- Provide bounded capacity and metadata-only observability.

**Non-goals**
- Reimplement provider authentication, quota, account selection, retry or failover in Main Brain.
- Use a live inference request as routine repository validation.
- Rewrite product-neutral lifecycle code solely for naming.

## Decisions

### 1. One router identity

`RouterOwner` accepts only `omniroute`. Persisted or incoming alternate values are rejected, not translated into an executable path. `CLIKind` identifies the frontend executable and `RouteModel` identifies the approved OmniRoute model; neither selects a provider account.

### 2. Fail-closed admission

Main Brain validates its configuration, restricted secret reference, gateway liveness/authenticated readiness, selected model and selected protocol before process launch. Any missing or negative state produces a bounded admission classification and no CLI process.

### 3. Credentialless child boundary

Environment composition starts from a sanitized allowlist, removes provider credentials and direct endpoints, adds task/workspace context, validates custom settings, creates controlled per-task CLI configuration, and applies trusted OmniRoute values last. Shared provider auth files are never copied into a task home.

### 4. Preserve lifecycle ownership

Main Brain continues to own task slot admission, repo/worktree preparation, local-directory safety, process launch, watchdogs, cancellation, stream batching, commit ledger integration, session pinning, terminal result publication and resource release. Kanban/project/squad/issue and Postgres behavior is unchanged.

### 5. Recovery has no alternate router

The state machine has only:

- `NORMAL`, router owner `omniroute`;
- `DEGRADED`, router owner `none`, model admission closed.

A gateway outage moves to `DEGRADED` at a session boundary. Only strict OmniRoute readiness permits return to `NORMAL`. Repeated outage or operator action cannot promote another router.

### 6. Rollback preserves the invariant

Rollback holds new admissions, drains/cancels according to tool/commit safety, selects the previous accepted OmniRoute and Main Brain revisions, and reopens only after readiness. If readiness cannot be restored, admission remains closed.

### 7. Capacity and observability

Capacity tiers are explicit and evidence-based. Cancellation releases slots exactly once. Metadata-only spans correlate ingress, queue, daemon admission, CLI process, OmniRoute, terminal persistence and UI delivery without prompts, tool payloads, repository content, credentials or account identity.

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

- **OmniRoute outage blocks new model work:** intentional fail-closed behavior; non-model product/control-plane operations continue.
- **Compatibility input contains an old owner:** reject it with an actionable bounded error instead of routing ambiguously.
- **Shared daemon file is large:** preserve existing lifecycle blocks and remove only routing/credential branches; validate targeted packages.
- **No local Go toolchain:** use an available pinned build container if present; otherwise report the concrete validation blocker.

## Migration Plan

1. Delete dedicated alternate-runtime packages, scripts and active plans.
2. Remove startup/config/env/health/recovery and task-time consumers from shared daemon files.
3. Enforce OmniRoute-only identity, admission, child environment and recovery.
4. Update tests, README, manifests and runbooks.
5. Run formatting, targeted unit tests/build and strict OpenSpec validation without inference.
6. Roll out only after restricted-secret metadata and strict readiness checks pass.
