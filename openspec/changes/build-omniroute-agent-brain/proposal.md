## Why

The product needs one deterministic managed-agent path. Main Brain already owns Kanban task orchestration, workspaces, repositories, process lifecycle, cancellation, terminal/session persistence and result delivery; OmniRoute must be the single owner of model routing and inference credentials. Multiple routers or direct-provider paths make readiness, recovery and failure ownership ambiguous.

## What Changes

- Make `omniroute` the only valid `RouterOwner` for model work.
- Fail closed when OmniRoute is unavailable, unauthenticated, missing the selected model/protocol, or lacks an approved route.
- Remove alternate runtime startup, configuration, health, recovery, credential-home selection and retry/account-selection paths.
- Preserve the Multica backend/web product, projects, squads, Kanban issues/tasks, Postgres, daemon lifecycle, worktrees, cancellation/watchdogs, terminal/session persistence, `runtimeenv`, `execenv` and gateway contracts.
- Keep child execution credentialless with respect to providers; only an opaque OmniRoute transport secret may cross the trusted boundary.
- Define recovery as `NORMAL(omniroute) → DEGRADED(none) → NORMAL(omniroute)`, only at session boundaries and only after readiness returns.
- Update deployment, rollback, health, observability and capacity contracts for this single path.
- Make the product Kanban assignment API the only authority for starting agent work; one
  assignment/follow-up produces exactly one product task, while Herdr is supervision-only.

## Capabilities

### New Capabilities

- `agent-brain-runtime`: Main Brain orchestration from Kanban claim through one terminal result.
- `omniroute-agent-routing`: strict readiness and single-router model admission.
- `credentialless-agent-execution`: provider-secret exclusion and controlled CLI configuration.
- `parallel-agent-capacity`: bounded admission, cancellation and tier evidence.
- `brain-cutover-operations`: safe rollout/rollback without alternate routing.
- `end-to-end-observability`: metadata-only correlation across control, execution and result delivery.

## Impact

- Shared daemon files change surgically; product APIs, workspaces, project/squad/issue state and Postgres remain.
- Model tasks without valid OmniRoute configuration are rejected before process launch.
- Rollback selects an earlier accepted Main Brain/OmniRoute revision or leaves admission closed.
- Live inference and provider credential lifecycle remain outside basic repository validation.

## Operational update — 2026-07-28

The Kanban-only dispatch rule is active and was exercised by ORQ-12: assignment created one
task, follow-up correction created one task only after the prior task was terminal, and no
Herdr execution was launched. A task being terminal does not close its issue; acceptance is
driven by the technical result and independent review.

The reasoning-admission correction exists on branch commit `f5660e9` but is not yet deployed.
Production behavior must not be described as fixed until integration, image rollout and a
controlled post-deploy smoke prove a non-empty supported level dispatches successfully.
