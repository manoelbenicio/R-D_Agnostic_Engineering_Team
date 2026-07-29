# FIX — reproducer/test DESIGN: StartTask no-rows on concurrent rerun (superseded-skip)

- agent: **Opus48#C** · lane **reproducer-test** · task **FIX-REPRO-TEST-DESIGN** · pane `w8:p1`
- as-of (UTC): `2026-07-23T00:10Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- check-in: `.deploy-control/p0/checkins/Opus48-C__FIX-REPRO-TEST-DESIGN__20260723T000948Z.json`
- **DESIGN-ONLY per Kiro steering: no source edits (no test written) until Kiro assigns a writer.** Only this evidence file is written.

## 0. Report (immediate)

Reproducer identified and precisely grounded. **Root cause:** `TaskService.StartTask` hard-wraps `pgx.ErrNoRows` as a fatal error, so when a concurrent `RerunIssue` (with `max_concurrent_tasks=1`) has already flipped task A to `cancelled`, the daemon's later `StartTask(A)` returns `start task: no rows` → surfaces as `execution_error` and can trip the gateway, instead of a graceful superseded-skip. Exact test file + assertions below; the existing `mockDBTX` harness reproduces it with zero new infrastructure.

## 1. Root cause (exact symbols)

- `internal/service/task.go:1228` `StartTask`:
  ```go
  task, err := s.Queries.StartAgentTask(ctx, taskID)
  if err != nil {
      return nil, fmt.Errorf("start task: %w", err)   // <-- hard error even for pgx.ErrNoRows
  }
  ```
- `pkg/db/queries/agent.sql:326` / `generated/agent.sql.go:2556` `StartAgentTask` (`:one`):
  ```sql
  UPDATE agent_task_queue
  SET status = 'running', started_at = now(), wait_reason = NULL
  WHERE id = $1 AND status IN ('dispatched', 'waiting_local_directory')
  RETURNING ...
  ```
  A `:one` query whose UPDATE matches **0 rows** returns `pgx.ErrNoRows`. If A is no longer `dispatched`/`waiting_local_directory` (e.g. `cancelled` by a concurrent rerun), StartTask fails hard.
- **Contrast — the correct pattern already exists** in `CompleteTask` (`task.go:1318`): on `pgx.ErrNoRows` it calls `GetAgentTask` and, if the task is already finalized, returns it as an **idempotent success** ("complete task: already finalized"). `StartTask` lacks this idempotent/skip handling. `CancelTask` follows the same idempotent pattern.

## 2. Concurrent mechanism (why "no rows" happens): rerun + `max_concurrent_tasks=1`

- `RerunIssue` (`task.go:1727`) → `CancelAgentTasksByIssueAndAgent` (`agent.sql:215/226` → `SET status='cancelled', completed_at=now()`) cancels the target agent's active/queued tasks on the issue (task **A → `cancelled`**), then `enqueueRerunTask` enqueues a fresh task B.
- With `agent.MaxConcurrentTasks == 1` (`task.go:1051` capacity gate), cancelling A is what frees the single slot for B — so the rerun path deterministically cancels A.
- The daemon had already claimed/dispatched A and calls `StartTask(A)` in the window **after** the concurrent rerun cancelled it. `StartAgentTask`'s `WHERE status IN ('dispatched','waiting_local_directory')` no longer matches (A is `cancelled`) → `pgx.ErrNoRows` → hard `start task: no rows`.
- Note: there is **no `superseded` status** in the schema; the real post-rerun status is **`cancelled`** (via `CancelAgentTasksByIssueAndAgent`). "Superseded-skip" = treat this cancelled-by-rerun A as a graceful skip, not an execution error.

## 3. Exact test file + assertions (for the assigned writer)

- **New file (disjoint from F6's `task.go`/`task_complete_race_test.go`/`task_notify_test.go`):**
  `multica-auth-work/server/internal/service/start_task_supersede_repro_test.go`, `package service`.
- **Reuse the existing harness** already in `task_complete_race_test.go` (same package — no redefinition): `mockDBTX` (routes any SQL containing `"SET status ="` → `mockRow{err: pgx.ErrNoRows}`; else `GetAgentTask` → stored task), `mockRow`, `testUUID`. `StartAgentTask` SQL contains `SET status = 'running'` → the mock returns `ErrNoRows`, exactly reproducing the 0-rows UPDATE.
- **Test:** `TestStartTask_CancelledByConcurrentRerun_GracefulSupersededSkip`
  ```go
  func TestStartTask_CancelledByConcurrentRerun_GracefulSupersededSkip(t *testing.T) {
      taskID := testUUID(1)
      agentID := testUUID(2)
      // A was cancelled by a concurrent RerunIssue (CancelAgentTasksByIssueAndAgent).
      mock := &mockDBTX{task: db.AgentTaskQueue{ID: taskID, AgentID: agentID, Status: "cancelled"}}
      svc := &TaskService{Queries: db.New(mock), Bus: events.New()}

      got, err := svc.StartTask(context.Background(), taskID)

      // (1) MUST NOT hard-error / must not surface pgx.ErrNoRows (no execution_error, no gateway trip):
      if err != nil {
          t.Fatalf("StartTask on a rerun-cancelled task must be a graceful skip, got hard error: %v", err)
      }
      if errors.Is(err, pgx.ErrNoRows) {
          t.Fatal("StartTask leaked pgx.ErrNoRows (would surface as execution_error)")
      }
      // (2) graceful superseded-skip returns the existing finalized task unchanged:
      if got == nil || got.Status != "cancelled" || got.ID != taskID {
          t.Fatalf("expected the existing cancelled task returned as a skip, got %+v", got)
      }
  }
  ```
- **Assertions summary:** (a) `err == nil`; (b) `!errors.Is(err, pgx.ErrNoRows)`; (c) returned task non-nil with the existing terminal status (`cancelled`) and matching ID (proves the daemon receives a skip signal, not a failure).
- **Optional table extension:** same assertions for the other already-finalized statuses reachable in the race (`completed`, `failed`) so StartTask is idempotent like CompleteTask/CancelTask.

## 4. Expected outcomes (reproducer semantics — no fake PASS)

- **Against current code (unfixed):** the test **FAILS** — `StartTask` returns `start task: no rows` (hard error). This is the reproducer confirming the defect.
- **Against the fix (§5):** the test **PASSES** — StartTask returns the existing cancelled task with `nil` error (graceful skip).
- Because the design is not yet written and would fail on current code, it is delivered as a design (not committed as a red test) pending the writer + fix, per the "no source edits until Kiro assigns writer" instruction.

## 5. Fix sketch (for the FIX lane / assigned writer — NOT this design lane)

Mirror the `CompleteTask` idempotency block in `StartTask` (`task.go:1228`):
```go
task, err := s.Queries.StartAgentTask(ctx, taskID)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        if existing, lookupErr := s.Queries.GetAgentTask(ctx, taskID); lookupErr == nil &&
            isTerminalOrNonStartable(existing.Status) { // cancelled/completed/failed/superseded-by-rerun
            slog.Info("start task: already superseded/finalized — graceful skip",
                "task_id", util.UUIDToString(taskID), "current_status", existing.Status)
            return &existing, nil // graceful superseded-skip; daemon must treat nil-err as skip, not run
        }
    }
    return nil, fmt.Errorf("start task: %w", err)
}
```
The daemon's StartTask caller must interpret the returned already-terminal task (or a dedicated sentinel like `ErrTaskSupersededSkip`) as a skip — NOT dispatch execution and NOT classify `execution_error`. (Writer to confirm the daemon-side contract; keeping the graceful signal typed avoids ambiguity with a genuine start.)

## 6. Scope / non-claims
- DESIGN-ONLY: **no source or test file written**; no edit to `task.go` or F6-owned files; only this evidence file created.
- No test executed (nothing to run yet); the current-vs-fixed outcomes in §4 are derived from the confirmed `:one`→`ErrNoRows` semantics + the `StartTask` hard-wrap, not from a run.
- No install; no OmniRoute internals; no live run/inference/secret.
- Handoff: assigned writer creates `start_task_supersede_repro_test.go` (§3) and the FIX lane applies §5; both are outside this design lane.

## 7. Status
- STATUS: DONE (design delivered; awaiting Kiro's writer assignment).
- DELIVERED: root cause (§1), concurrent mechanism (§2), exact test file + assertions reusing the existing harness (§3), reproducer semantics (§4), fix sketch (§5).
