## Why

The product needs one deterministic transport binding for each managed-agent launch. Main Brain
owns Kanban task orchestration, workspaces, repositories/worktrees, process lifecycle,
cancellation, terminal/session persistence and result delivery. In the `omniroute` binding,
OmniRoute is the sole inference router and sole account/credential owner. In the
`native_credential_home` binding, R3 resolves one approved exclusive opaque home for one
existing logical runtime/agent and OmniRoute is not used. Ambiguous ownership, cross-mode
fallback, or direct exposure of credential homes would make admission and recovery unsafe.

## What Changes

- Admit exactly one pinned transport binding: `omniroute` or `native_credential_home`.
- For `omniroute`, require strict OmniRoute readiness; forbid native homes; keep the child
  provider-credentialless; and fail closed when OmniRoute, its selected model/protocol, or its
  approved route is unavailable.
- For `native_credential_home`, require R3 to resolve one approved exclusive opaque home for one
  existing logical runtime/agent before launch; pass only daemon-local isolated-home references
  needed by the native CLI; do not probe, contact, or use OmniRoute.
- Never fallback, translate, rotate, retry, or silently remap between transport bindings or
  native homes. Both modes fail closed.
- Preserve the Multica backend/web product, projects, squads, Kanban issues/tasks, Postgres,
  daemon lifecycle, worktrees, cancellation/watchdogs, terminal/session persistence,
  `runtimeenv`, `execenv` and the dynamic metadata-only controlled-root catalog contract.
- Forbid global HOME and raw path, account identity, or credential data in product APIs, events,
  logs, diagnostics, or evidence.
- Never copy, move, delete, truncate, sanitize, overwrite, chmod, or change ownership of a
  source credential home. Cleanup is limited to task-local non-source material after
  active-reference checks.
- Update deployment, rollback, health, observability and capacity contracts for both
  non-overlapping bindings.
- Make the product Kanban assignment API the only authority for starting agent work; one
  assignment/follow-up produces exactly one product task, while Herdr is supervision-only.

## Capabilities

### New Capabilities

- `agent-brain-runtime`: Main Brain orchestration from Kanban claim through one terminal result.
- `omniroute-agent-routing`: strict transport-binding admission and OmniRoute readiness.
- `credentialless-agent-execution`: binding-scoped child isolation, provider-secret exclusion,
  native isolated-home handling, and source-home preservation.
- `parallel-agent-capacity`: bounded admission, cancellation and tier evidence.
- `brain-cutover-operations`: safe rollout/rollback without changing the pinned transport.
- `end-to-end-observability`: metadata-only correlation across control, execution and result delivery.

## Impact

- Shared daemon files change surgically; product APIs, workspaces, project/squad/issue state and
  Postgres remain.
- A launch without one complete, valid transport plan is rejected before process creation.
- Rollback selects an earlier accepted configuration/revision inside the same binding or leaves
  admission closed; it never switches transport or mutates a source credential home.
- Live inference and source credential lifecycle remain outside basic repository validation.

## Reconciled local status

The accepted dual-binding design is implemented in the composed local source and reconciled with
credential-account-home REQ-24..35. Later ROOT OmniRoute-only wording is not promoted because it
would delete the accepted native binding. Kanban-only dispatch, fail-closed admission, controlled
child configuration, lifecycle preservation and metadata-only observability remain authoritative.
The local source is uncommitted and unpushed; this document authorizes no external action.
