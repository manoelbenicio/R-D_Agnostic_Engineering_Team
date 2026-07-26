# F6 — queue/persist span wiring into task.go (DONE)

- agent: Opus48#D · lane: F6 · task: F6-queue-persist-anchors · pane `w8:p2`
- check-in receipt: `.deploy-control/p0/checkins/CHECKIN__Opus48-D__F6__F6-QUEUE-PERSIST-ANCHORS__20260722T113411Z.json`
- ownership (edited, exact): `internal/service/task.go`, `task_complete_race_test.go`, `task_notify_test.go`
- OpenSpec 6.2 · AB-REQ-39/40 (OBS-3 queue hop, OBS-7 persist hop) · consumes frozen L6/L5 `agent-brain.e2e.v1`
- preflight: HEAD `a6d5098`, git status 331, go1.26.1, node v22.23.1, root disk 100% → GOCACHE/GOTMPDIR under `/tmp` (tmpfs 7.6G).

## 1. Wiring (minimal, contract-correct, behavior-preserving)
- `TaskService.Obs *e2e.Recorder` field added (optional; nil disables emission). Emission is
  best-effort, metadata-only, and never affects task flow (errors from `Emit` are ignored; the
  recorder itself fail-closes and refuses any content).
- Three nil-safe helpers added in `task.go`: `emitQueueEnqueued`, `emitQueueDequeued`,
  `emitPersistSpan` (+ `timestamptzUnixMs`). They call the L6 helpers `EmitQueue`/`EmitPersist`
  (same package) with **real ids + only frozen closed counter/label keys**.
- Call sites (single choke points, no new anchors invented):
  - **enqueue** → `NotifyTaskEnqueued` (the choke point all 6 enqueue paths call): `emitQueueEnqueued`.
  - **claim/dequeue** → `captureTaskDispatched` (reached by `ClaimTask` and by `ClaimTaskForRuntime`
    which delegates to `ClaimTask`): `emitQueueDequeued`.
  - **persist** → `CompleteTask`, immediately after the `runInTx`/`CompleteAgentTask` commit and
    `captureTaskCompleted`, using the persisted row and `time.Since(persistStart)`.

## 2. Identifiers & counters (real, closed set, metadata-only)
- task_id = queue_msg_id = `util.UUIDToString(task.ID)` (the `AgentTaskQueue` row IS the queued task;
  the row id is both the queue message id and the task id).
- result_id = `"result-"+util.UUIDToString(task.ID)` — deterministic from the persisted row (schema has
  no separate result-id column; the result is stored on the task row).
- queue counters: `enqueue_unix_ms` (created_at), `dequeue_unix_ms` (now), `wait_ms` (now−created_at, ≥0).
- persist counters: `persist_latency_ms`, `byte_count`=`len(task.Result)`; label `terminal_status`=`task.Status`.
- No request/task/result content, DB payload, or free-form labels. `e2e.ScanSpans` asserted Clean in tests.

## 3. Persistence ordering proven
`emitPersistSpan` consumes the **persisted** row (it derives `byte_count` from `task.Result` after the
DB commit) and is called only after `CompleteAgentTask` commits — so a persist span can only carry
post-persistence state. `TestEmitPersistSpanOrderingConsumesPersistedRow` asserts `byte_count` equals
the persisted `Result` length; the call-site placement (after the tx + `captureTaskCompleted`) enforces
ordering at the source.

## 4. Validation (GOCACHE=/tmp/f6-gocache; go=/home/ec2-user/goroot/go/bin/go)
| # | Command | Result | Exit |
|---|---|---|---|
| 1 | `gofmt -l internal/service/{task,task_complete_race_test,task_notify_test}.go` | empty (clean) | 0 |
| 2 | `go vet ./internal/service/` | clean (task.go compiles with edits) | 0 |
| 3 | `go test ./internal/service/ -run 'EmitQueue|EmitPersist' -count=1 -v` | **5 tests PASS** (persist x2, queue x3); `ok 0.005s` | 0 |
| 4 | `git diff --check -- <3 files>` | clean | 0 |

`-race` NOT run (gcc/cgo unavailable — environment limit, as documented by L5/L6).

## 5. Handoffs / non-claims
- **Recorder injection (handoff):** `Obs` is nil by default (`NewTaskService` does not set it), so
  emission is a no-op in production until the observability **sink/exporter owner** injects a real
  `*e2e.Recorder` (e.g. via the `TaskService` DI / router wiring). This lane deliberately does not
  fabricate a sink/exporter (that is another lane's scope, Priority-2 promexport). Call sites are wired
  and ready.
- **Stale-reclaim path:** `ClaimTaskForRuntime`'s stale-dispatched reclaim (`return &stale`) is a
  re-dispatch, not a fresh dequeue, and is intentionally not separately spanned.
- No shared-anchor logic changed beyond the additive best-effort emit calls; no deploy/inference/
  secret/commit; no OpenSpec checkbox closed; live_runs remain false.
