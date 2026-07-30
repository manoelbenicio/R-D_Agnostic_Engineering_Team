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

The reasoning-admission correction derived from `f5660e9` is integrated in the current
backend image. A production Kiro canary with `thinking_level=high` reached ACP, emitted
messages and executed tools without `thinking_not_approved`; this proves the supported
non-empty admission path for Kiro. Codex and Kiro agents read back `high`, while AGY agents
remain `NULL` because their reasoning tier is embedded in the model identifier.

## Verified deployment update — 2026-07-30

- Backend revision `edd7b932`, image `f9e6b777`, was observed healthy and ready with
  `RestartCount=0`; migration 129 was validated and rollback evidence was preserved.
- Daemon revision `f20b3e`, binary `f40ab0`, was observed active with `RestartCount=0`; its
  rollback evidence was preserved.
- ORQ-13 is deployed, but its observed post-deploy row still has `thinking_level`,
  `price_version`, and `cost` as `NULL`; accounting completion is not claimed.
- ORQ-41 currently proves only that a `documentation_only` activation enqueued zero task and
  one explicit assignment enqueued exactly one task.
- ORQ-54 still lacks the focused routing/resume smoke. ORQ-69 still waits for ORQ-74, which is
  blocked exactly on slots `162` and `163`.
- Queue depth was zero at cutover only; it is not asserted as a timeless production state.
