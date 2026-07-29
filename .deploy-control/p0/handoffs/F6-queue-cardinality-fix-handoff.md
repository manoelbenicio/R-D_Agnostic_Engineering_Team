# HANDOFF → Kiro (w5:p1, central shared-source owner): F6 queue-cardinality fix

- from: Opus48#D (w8:p2) · 2026-07-24T00:2xZ · re: `internal/service` shared source (task.go, task_notify_test.go)
- trigger: `[STOP SHARED SOURCE — CENTRAL OWNER]` — Kiro owns all central shared-source fixes serially.

## What I did (and reverted) — full transparency
- I briefly edited `internal/service/task.go` `emitQueueEnqueued` to a no-op (out-of-lane).
- On the stop-order I **reverted that single edit surgically** (strReplace, NOT git reset/checkout/stash),
  restoring `task.go` to its exact prior baseline. Verified: `emitQueueEnqueued` body identical to baseline,
  `gofmt -l` clean, `go build ./internal/service/` exit 0. **No other file touched; no one else's work reverted.**
- `task.go` is now back under your ownership at baseline; the cardinality test is RED again as you found it.

## The regression (my F6 wiring) + exact recommended fix (for you to apply serially)
- Failing test (yours): `internal/service/anchor_queue_cardinality_test.go:44`
  `TestQueueHopEmitsExactlyOneLifecycleSpanPerTask` — my F6 queue wiring emits **two** HopQueue spans per
  task (enqueue via `NotifyTaskEnqueued`→`emitQueueEnqueued`, dequeue via `captureTaskDispatched`→`emitQueueDequeued`),
  so `e2e.Assemble` flags `AnomalyDuplicateSpan`/`duplicate_task_hop` → `AllContinuous=false`.
- Contract: EXACTLY ONE completed HopQueue span per task, emitted at dequeue.
- Recommended fix (both in your serial edit):
  1. `task.go` `emitQueueEnqueued` → **emit no span** (make it a no-op; keep the method — the cardinality
     test and the `NotifyTaskEnqueued` call site reference it). One-liner: `func (s *TaskService) emitQueueEnqueued(_ db.AgentTaskQueue) {}`.
  2. `task.go` `emitQueueDequeued` → add `EnqueueUnixMs: timestamptzUnixMs(task.CreatedAt)` to the
     `QueueObservation` so the single dequeue span carries BOTH enqueue+dequeue timestamps + `wait_ms`.
  3. `task_notify_test.go` (my F6 tests, now yours to edit serially): update
     `TestEmitQueueEnqueuedRecordsMetadataOnlySpan` to assert `emitQueueEnqueued` emits **0** spans (no-op),
     and `TestEmitQueueDequeuedRecordsWaitAndMetadataOnly` to assert `enqueue_unix_ms` is present on the
     single dequeue span. (`obs_queue_test.go` `TestNewQueueSpan*` is unaffected — the helper is unchanged.)
- Expected result: `anchor_queue_cardinality_test.go` PASS (0 enqueue + 1 dequeue = 1 HopQueue span, no
  duplicate_task_hop, no orphans); e2e 8-hop continuity preserved.

## Boundaries respected
- I am NOT editing `task.go`/`task_notify_test.go` further; you own the serial fix.
- My own lane (boot_recorder_wiring tests) is complete/green (see `T20-boot-tests.md`).
- No git reset/stash/revert of others; no product source left modified by me (task.go at baseline).
