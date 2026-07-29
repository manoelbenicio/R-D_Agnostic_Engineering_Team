# ORQ-21 → ORQ-12 — immutable producing-account snapshot handoff

- Author/owner: `Codex56#B`
- Date: `2026-07-28`
- Reviewed ORQ-21 commit: `182b7b3644c860f3149dee75292dc5dfe8ea6b6a`
- Reviewed ORQ-12 commit: `e7c6a5c` in
  `/home/ec2-user/workspace/worktrees/orq12-account-id`
- Mode: read-only analysis of ORQ-12. No ORQ-12 file, migration, generated output, DB, daemon,
  runtime, remote, or board state was changed.

## Verdict

**ORQ-21 resolves the correct backend-owned account identity, but does not yet freeze enough
identity for ORQ-12.**

`credentialregistry.Resolver.Resolve` returns the approved `accounts.account_id` after checking the
agent/workspace tenant binding, provider, approval, status, paths, and worktype scope. The daemon
does not choose this value. However, commit `182b7b3` sends only `home_dir` plus a required boolean
and persists no account identity on `agent_task_queue`.

ORQ-12 commit `e7c6a5c` derives `task_usage.account_id` from the *current*
`assignments.account_id` when usage is reported. If assignment A is used to start a task, then the
agent is reassigned to B before its first usage report, the query attributes A's usage to B.
`COALESCE` protects only after a first non-NULL `task_usage` row exists; it does not close the
pre-first-report race.

Therefore ORQ-12 must not integrate its live-assignment lookup as an immutable snapshot.

## Required backend-owned snapshot

The safe value is the UUID already returned as `Assignment.AccountID`. It is metadata, not a
credential, secret reference, token, email, home path, or daemon assertion.

The canonical snapshot must be:

1. selected from the approved account/assignment chain by the backend;
2. written to the claimed `agent_task_queue` row as part of the same claim transaction/statement;
3. immutable after the queued → dispatched transition;
4. unavailable as a caller-supplied field on the usage endpoint; and
5. copied server-side from the task snapshot into `task_usage.account_id`.

Sending an account ID in daemon JSON and accepting it back is explicitly insufficient. Performing a
second live assignment lookup at usage time is also insufficient.

## Exact handoff and file ownership

### Schema/SQLC handoff — serialized LANE-DB

The schema owner/Registrar must allocate, without inventing a number, one column:

```text
agent_task_queue.credential_account_id UUID NULL
```

The FK/delete policy needs a durable ruling. `ON DELETE RESTRICT` is the fail-closed default because
it preserves the historical identity; `ON DELETE SET NULL` weakens immutability. No column currently
exists, and reusing `context` or `trigger_summary` would be an unsafe hidden schema.

Exact shared files:

- the canonically reserved migration UP/DOWN selected by the Registrar;
- `multica-auth-work/server/pkg/db/generated/models.go`;
- `multica-auth-work/server/pkg/db/generated/agent.sql.go`;
- later, ORQ-12's own `task_usage` migration/query/generated files consume the snapshot.

This document does not assign or materialize a migration number and does not edit ORQ-12's staged
files.

### ORQ-21 follow-up ownership

The smallest ORQ-21 code follow-up is:

- `multica-auth-work/server/pkg/db/queries/agent.sql` — extend the atomic
  `ClaimAgentTask`/reclaim contract so covered providers set and return the approved account UUID in
  the same statement that changes status to `dispatched`; a missing/mismatched/revoked assignment
  returns no claim.
- `multica-auth-work/server/pkg/db/generated/agent.sql.go` — generated only by the exclusive
  LANE-DB owner.
- `multica-auth-work/server/internal/service/task.go` — propagate the claim result; do not accept an
  account identifier from daemon or HTTP input.
- `multica-auth-work/server/internal/credentialregistry/resolver.go` — retain reusable policy
  validation or move its exact predicates into the claim query, with one canonical implementation.
- `multica-auth-work/server/internal/handler/daemon.go` — use the frozen task snapshot for the
  approved home resolution/response, before any daemon payload is emitted.
- ORQ-21 focused DB tests in
  `multica-auth-work/server/internal/handler/orq21_credential_assignment_test.go` and/or a new
  query-level DB test.

The dispatch event currently occurs inside `TaskService.ClaimTask`, before the handler's separate
resolver call. Thus merely adding an UPDATE after `ClaimTaskForRuntime` would leave a race and is not
the accepted handoff. The snapshot belongs in `ClaimAgentTask` itself (and must be preserved by
`ReclaimStaleDispatchedTaskForRuntime`).

### ORQ-12 consumption

After the above is integrated, ORQ-12 should replace:

```text
agent_task_queue.agent_id → assignments.account_id
```

with:

```text
agent_task_queue.credential_account_id
```

inside `UpsertTaskUsage`. `UpsertTaskUsageParams` remains account-ID-free, so the daemon cannot
select attribution. ORQ-12's `COALESCE` then protects corrections/reports without consulting mutable
assignment state.

## Executable acceptance tests

1. Claim with approved A; atomically assert the task row snapshots A before dispatch is returned.
2. Reassign the agent to B before the first usage report; usage remains attributed to A.
3. Revoke A after claim; the already claimed task retains A, while the next claim fails or snapshots
   another explicitly approved account according to the approved rotation contract.
4. Cross-tenant, provider mismatch, NULL worktype scope, unavailable account, or missing approval
   returns no dispatched claim and no snapshot.
5. A stale dispatched-task redelivery returns the original snapshot and never re-resolves B.
6. Usage JSON containing an invented `account_id` is rejected as unknown or ignored by a strict
   request contract and can never affect storage.
7. Account deletion obeys the selected immutable FK policy.

## Sequencing

`182b7b3` can be reviewed for its present metadata-assignment contract, but ORQ-12 must remain
blocked from integration on its live-assignment attribution. The queue-snapshot follow-up requires
an explicit LANE-DB/Registrar handoff because its query/generated/migration files are shared. No
writer should duplicate those files while ORQ-12 owns them.

## GTM R3 ownership addendum

The later GTM ruling supersedes the tentative “ORQ-21 follow-up ownership” list above:

- ORQ-21 owns canonical `accounts.vendor` writes in
  `multica-auth-work/scripts/staging/seed_approved_assignment.sql` and exact comparison in
  `multica-auth-work/server/internal/credentialregistry/resolver.go`. Commit
  `aac222715c46dd3000b00c0df3ba43e83b94a73f` implements that boundary.
- ORQ-12/LANE-DB exclusively owns the staged migration, `pkg/db/queries/agent.sql`, SQLC
  generated output, runtime-provider write normalization, reclaim behavior and their tests.
- The snapshot applies only to `codex`, `kiro` and `antigravity`; `agy` is an input alias that
  must be stored as `antigravity`.
- Migration preflight/backfill must produce canonical stored `accounts.vendor` and
  `agent_runtime.provider`; claim then uses exact equality rather than a second read-time
  canonicalizer.
- Reclaim preserves the existing snapshot, never re-resolves it, and fails closed when a covered
  dispatched task has a NULL snapshot.
- Promotion/deploy requires an aggregate zero-active-queue gate for `queued`, `dispatched`,
  `running` and `waiting_local_directory`.

No ORQ-12 file was edited to record this addendum.
