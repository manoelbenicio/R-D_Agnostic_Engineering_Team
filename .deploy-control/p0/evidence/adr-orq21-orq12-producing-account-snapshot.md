# ADR — ORQ-21/ORQ-12 producing-account attribution

Date: 2026-07-28 UTC  
Status: GTM decision — implementation and independent DB verification required

## Decision

Persist `credential_account_id` on the `agent_task_queue` row atomically when that row is claimed. `task_usage.account_id` must be copied only from that immutable snapshot; it must never be derived from the current mutable assignment and must never be accepted from the daemon payload.

In this product, an `agent_task_queue` row is an execution attempt. Automatic retry uses `CreateRetryTask`, which inserts a fresh queued row with a new task ID and a parent-task link. The retry row intentionally starts without a frozen account and freezes the approved account valid at its own claim. Therefore a separate attempt table is not required for accurate attribution.

## Required invariants

1. Claim status transition and account snapshot are one atomic SQL statement/transaction boundary.
2. Resolution is server-side and scoped to the task agent, workspace/tenant, runtime provider, approved state, usable account status, and supported worktype policy.
3. Ambiguous resolution fails closed; no `LIMIT 1` arbitrary choice.
4. One credential account/home cannot be actively assigned to multiple agents; application validation is backed by a durable database uniqueness constraint.
5. Reclaim of the same dispatched row preserves its existing snapshot.
6. Retry creates a new queue row and freezes independently when claimed.
7. Usage upsert preserves an already stored account (`existing-first` semantics) and can only fill a previously null legacy value from the queue snapshot.
8. Account deletion may null attribution through the declared FK policy but must never delete token-usage facts.
9. Reports are workspace-scoped and expose unattributable/null usage explicitly rather than hiding it.
10. Mixed-version deployment is fail-closed behind an explicit rollout gate: server/schema first, daemon enforcement second.

## Applicability and canonicalization ruling

- The snapshot contract applies only to providers that require approved credential assignment: `codex`, `kiro`, and canonical `antigravity` (`agy` is an input alias). Other providers retain a null snapshot even if incidental account metadata exists.
- Provider identity has one durable representation. Migration 128 must preflight and canonicalize existing provider/vendor data; write paths must persist canonical values; the atomic claim compares canonical stored values by exact equality. A permanent SQL `CASE` duplicating a Go alias table is not accepted as a second authority.
- A new alias must fail a parity/normalization gate until its write-path migration and tests are supplied. It must not silently make attribution null.
- Reclaim never resolves or changes an account. It preserves the original attempt snapshot and fails closed when a covered provider reaches `dispatched` with a null snapshot. Activation requires the active four-state queue to be zero so no ambiguous legacy row crosses the rollout boundary.

## Rejected alternatives

- Resolve the current assignment when usage first arrives: rejected because reassignment can rewrite history before the first report.
- Accept account ID from the daemon: rejected because the caller could misattribute consumption.
- Select an arbitrary approved account with `LIMIT 1`: rejected because ambiguity must be surfaced, not hidden.
- Add a new attempt table now: rejected as redundant because each retry already creates a distinct `agent_task_queue` row and `task_usage` is keyed by that task ID.

## Acceptance evidence

The combined ORQ-21/ORQ-12 ephemeral-PostgreSQL gate must prove claim atomicity, reclaim preservation, retry re-snapshotting, reassignment after claim, tenant/provider/worktype mismatches, revoked/unavailable/ambiguous accounts, exclusivity under concurrency, legacy null handling, account deletion behavior, workspace isolation, usage idempotency, zero false-green package execution, and rollback/up-down-up migration behavior.
