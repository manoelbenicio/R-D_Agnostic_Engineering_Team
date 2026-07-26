# FIX — SQL trace: why StartAgentTask returns no rows under concurrent rerun

- agent `Opus48#B` · lane `SQL-TRACE` · pane `w6:p2` · task `FIX-SQL-TRACE`
- check-in receipt: `.deploy-control/p0/checkins/Opus48-B__FIX-SQL-TRACE__20260723T000838Z.json`
- posture: **READ-ONLY** (no edits). Source at HEAD `a6d50986…`.

## Exact SQL

**StartAgentTask** (`pkg/db/queries/agent.sql:326`, sqlc `:one`):
```sql
UPDATE agent_task_queue
SET status = 'running', started_at = now(), wait_reason = NULL
WHERE id = $1 AND status IN ('dispatched', 'waiting_local_directory')
RETURNING *;
```
`:one` → returns `pgx.ErrNoRows` when the WHERE matches 0 rows (i.e. the row exists but its status is
NOT in `{'dispatched','waiting_local_directory'}` at execution time).

**CancelAgentTasksByIssueAndAgent** (`pkg/db/queries/agent.sql:214`, sqlc `:many`):
```sql
UPDATE agent_task_queue
SET status = 'cancelled', completed_at = now()
WHERE issue_id = $1 AND agent_id = $2
  AND status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
RETURNING *;
```

**RerunIssue** (`internal/service/task.go:1727`) ordering: (1) resolve target agent →
(2) `CancelAgentTasksByIssueAndAgent(issueID, agentID)` (flips prior active tasks → `'cancelled'`) →
(3) `enqueueRerunTask` (fresh task). Cancel-prior and the daemon's claim/start are **not** in one
transaction, so they can interleave.

## Root cause (exact)

`StartAgentTask` is an **optimistic-concurrency guard**: a task may flip to `running` **only** from
`dispatched`/`waiting_local_directory`. The daemon claim→run flow is two steps — the row is `dispatched`
(claimed), then the daemon calls `StartAgentTask` to flip to `running` (sometimes after a
`waiting_local_directory` wait).

Concurrent-rerun interleave that yields **no rows**:
```
T0  prior task P is 'dispatched' (or 'waiting_local_directory'); daemon about to StartAgentTask(P)
T1  RerunIssue(same issue, same agent) runs CancelAgentTasksByIssueAndAgent
        -> P.status := 'cancelled'  (P was in the cancel set)
T2  daemon StartAgentTask(P): WHERE id=P AND status IN ('dispatched','waiting_local_directory')
        -> P.status is now 'cancelled' -> 0 rows -> pgx.ErrNoRows
```
So **StartAgentTask returns no rows precisely because a concurrent RerunIssue superseded/cancelled the
prior task** (status moved out of the dispatched/waiting set). Equivalent no-row causes: P already
`running` (double-start) or already terminal. This is the guard working as intended — not a wrong-status
schema bug. Relaxing the WHERE clause would (wrongly) let a cancelled/superseded task start.

## Caller gap (why it surfaces as a hard error)

`internal/service/task.go:1229`:
```go
task, err := s.Queries.StartAgentTask(ctx, taskID)
if err != nil {
    return nil, fmt.Errorf("start task: %w", err)   // <- wraps pgx.ErrNoRows as a generic failure
}
```
This path does **not** special-case `pgx.ErrNoRows`, unlike sibling task.go methods that already do
(`:942`, `:988`, `:1061`, `:1140`, `:1318-1321`, `:1514`). So a benign supersede/cancel is reported as a
generic "start task: no rows in result set" error instead of "task was cancelled/superseded → abort
launch".

## Minimal invariant recommendation (no edits made; recommend only)

1. **Preserve the SQL guard as the invariant** — keep `StartAgentTask`'s
   `WHERE status IN ('dispatched','waiting_local_directory')`. Invariant: *a task transitions to
   `running` only from dispatched/waiting; any concurrent cancel/supersede wins.* Do NOT widen it.
2. **Fix at the caller (`task.go:1229`), minimally:** treat `errors.Is(err, pgx.ErrNoRows)` as a
   **benign superseded/cancelled outcome** — abort the launch cleanly (no hard error, no retry, route to
   the existing cancellation/cleanup path), mirroring the `ErrNoRows` handling already present at
   `:942/:988/:1061/:1318/:1514`. Optionally re-read the row to log `cancelled` vs `running` vs missing
   for diagnostics. This makes the concurrent-rerun race a no-op supersede instead of a spurious failure,
   without changing schema or the guard.

## Non-claims
Static read-only SQL/code trace; no edit; no DB executed; no runtime/live reproduction. Recommendation is
advisory (no code changed). Reported to Kiro via this evidence file.
