# T20-anchor-tests — production anchor/seam integration tests (test files only)

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-ANCHOR-TESTS` · pane: `w7:p3`
- lock: `internal/middleware/anchor_ingress_recorder_test.go` (created) + 3 reserved paths (see §4) + this evidence
- MODE: **test files only; no shared source edits.** No deploy/inference/secret.

> DELIVERED: a genuinely-new **real-call-site** anchor test that drives the production `RequestLogger`
> HTTP middleware end-to-end and FAILS if the ingress hop is wired to a nil/discard recorder. Verified
> that the per-hop **fail-if-discard** seam contract for the other four hops is **already covered** by
> owner tests + the cross-package `e2ewiring` test, so duplicating them is barred by the no-duplicate-QA
> rule; the remaining true gap is **production startup wiring** (source, P0-gated), which no test-file-only
> change can prove. Real-call-site tests for queue/persist/delivery/cli require DB/WS/daemon harnesses in
> concurrently-owned packages — specified as bounded handoffs (§4).

## 1. Delivered anchor test (NEW, real call site)

`internal/middleware/anchor_ingress_recorder_test.go` (`package middleware`):
- `TestRequestLoggerAnchorEmitsIngressSpanWhenWired` — installs a `MemorySink` recorder via
  `SetIngressRecorder`, serves a request through the **real `RequestLogger(inner)` handler** (inner
  stamps `SetIngressTaskID`; chi request_id seeded via `chimw.RequestIDKey`), and asserts the sink
  received exactly **one HopIngress span** with the real `request_id`/`task_id`, `secrets_present=false`.
  **Fails if** the handler stops invoking the ingress emitter or the seam is nil/discard (sink stays empty).
  Distinct from the existing `TestEmitIngressSpanRecordsMetadataOnlyWithBothIDs`, which calls the
  `emitIngressSpan` helper directly rather than exercising the middleware handler.
- `TestRequestLoggerAnchorFailsClosedWithoutTaskID` — recorder wired but no `task_id` stamped → **0
  spans** (required-ID fail-closed), proving the anchor is genuinely gated, not blindly emitting.

Validation (executed, /tmp caches, go1.26.1):
`gofmt -l` empty (exit 0); `go vet ./internal/middleware/` exit 0;
`go test -run RequestLoggerAnchor -v ./internal/middleware/` → **PASS 2/2** (exit 0).

## 2. Existing per-hop fail-if-discard coverage (verified — duplicating is barred)

Each hop's seam already has a test that injects a real recorder → asserts a span, AND a nil/discard
test asserting no emission (exactly "FAIL if a hop owner uses a nil/discard recorder"):

| Hop | Real emit call site | Existing seam test(s) | nil/discard covered |
|---|---|---|---|
| ingress | `request_logger.go emitIngressSpan` (handler L143) | `request_logger_test.go` `TestEmitIngressSpanRecordsMetadataOnlyWithBothIDs`; **+ my new handler-level test §1** | `obs_ingress_test.go` `TestEmitIngressNilRecorderIsNoop` |
| queue | `task.go emitQueueEnqueued`/`emitQueueDequeued` (via `TaskService.Obs`) | `task_notify_test.go` `TestEmitQueueEnqueued…`/`…Dequeued…` | `TestEmitQueueNilObsIsNoop` |
| persist | `task.go emitPersistSpan` (via `TaskService.Obs`) | `task_complete_race_test.go` `TestEmitPersistSpan…` | `(&TaskService{}).emitPersistSpan(...)` nil-Obs case |
| delivery | `hub.go emitDelivery` (via `SetDeliveryRecorder`) | `hub_test.go` `TestHub_EmitDelivery_OutcomesAreValidMetadataSpans` | `TestHub_NoDeliveryRecorder_NoEmitNoPanic` |
| cli | `daemon.go` L3778 `EmitCLI(d.cliObs, …)` | `cli_observability_test.go` `EmitCLI*` | `EmitCLI(nil,…)` noop case |

Cross-process assembly of all 7 emit seams into one continuous trace is covered by
`observability/e2ewiring/wiring_integration_test.go` `TestProductionSeamsEmitContinuousLifecycle`.

Adding my own copies of the four table above would be **duplicate QA** (globally prohibited), so I did
not create the reserved service/daemonws/daemon files as duplicates.

## 3. The genuine remaining gap (NOT test-file-addressable)

Per `T20-obs-wiring-inventory.md`, **production startup installs no real sink** — `SetIngressRecorder`,
`TaskService.Obs`, `hub.SetDeliveryRecorder`, `daemon.cliObs`, gateway executor recorder are all
set only in tests; production leaves them nil→discard. That is the actual "hop owner uses a nil/discard
recorder" condition. Proving it requires either (a) the **source startup wiring** (W1-serial anchors,
P0-gated per D-V3-21 — a source edit, out of "test files only"), or (b) a startup-path integration test
that asserts the wired sink after boot. Neither is a test-files-only change to a hop package. Owner:
W1 + Principal (`w5:p9`). This is the real acceptance blocker for live e2e traces.

## 4. Reserved real-call-site tests (bounded handoffs — need owner harnesses)

Higher-fidelity real-call-site tests for the remaining hops need harnesses that (i) largely duplicate
the §2 seam coverage and (ii) touch concurrently-owned packages, so they are best authored by the hop
owner with the appropriate harness:
- **service queue/persist** (`anchor_obs_recorder_test.go`): drive the real enqueue + terminal-persist
  paths (needs a DB/`Queries` harness) with `TaskService{Obs: MemorySink recorder}` → assert HopQueue at
  enqueue/dequeue and HopPersist after persistence; 0 with nil Obs.
- **daemonws delivery** (`anchor_delivery_recorder_test.go`): register a client on `NewHub()` with
  `SetDeliveryRecorder(real)`, drive the broadcast frame path (needs a WS/in-memory client) → assert
  HopDelivery; 0 without a recorder.
- **daemon cli/persist** (`anchor_cli_recorder_test.go`): run a task through the daemon launch path with
  `cliObs` set (needs a daemon/task harness) → assert HopCLI at L3778.
These were reserved in my lock but not authored to avoid duplicate QA and cross-lane harness conflict.

## 5. Non-claims / limitations
- Delivered one new test file (`anchor_ingress_recorder_test.go`); no source edits; `git diff --check`
  clean (new untracked file).
- Did not duplicate existing per-hop seam tests (§2) or build DB/WS/daemon harnesses in concurrently-
  owned packages (§4 handoff).
- The production-startup-wiring gap (§3) is the real "nil/discard" condition and is source/P0-gated, not
  test-file-only; routed to W1 + Principal.
- No deploy/inference/secret; no live run.

---

## 6. Per-hop CARDINALITY tests at real call sites (added per steering)

Goal: prove exactly ONE span per task per hop at the real call sites; fail on missing emit (ingress)
and on double-emit (queue).

### (A) ingress — exactly one span per accepted request
`internal/middleware/anchor_ingress_recorder_test.go`
`TestRequestLoggerAnchorEmitsExactlyOneIngressSpanPerAcceptedRequest`: drives the real `RequestLogger`
handler once (task_id stamped), asserts **exactly one HopIngress span** and `e2e.Assemble` reports **no
duplicate_task_hop** for ingress. **PASS** (`go test -run RequestLoggerAnchor ./internal/middleware/`
exit 0). Production-gap note (audit M1): no production accepted-control endpoint calls
`SetIngressTaskID`, so real-prod ingress cardinality is currently **ZERO**; the true end-to-end (A) test
driving the task-creating HTTP handler needs the handler+DB harness (handler lane) — bounded handoff.

### (B) queue — exactly one completed lifecycle span per task (RED-BY-DESIGN)
`internal/service/anchor_queue_cardinality_test.go`
`TestQueueHopEmitsExactlyOneLifecycleSpanPerTask`: drives the REAL call sites `emitQueueEnqueued`
(task.go:186) + `emitQueueDequeued` (task.go:199) for one task, asserts exactly one HopQueue span and
`Assemble` has no `duplicate_task_hop`.
- **Result: FAIL (exit 1) — RED-BY-DESIGN**, message: "queue hop produced **2** HopQueue spans for one
  task; want EXACTLY 1". This is the executable specification of audit finding **M5**: the queue hop
  double-emits (enqueued + dequeued, same task_id) → `e2e.Assemble` `AnomalyDuplicateSpan`
  (duplicate_task_hop) → `AllContinuous=false`.
- **Fix (source, out of this lane): emit ONE completed queue lifecycle span per task** (e.g. a single
  span at dequeue carrying `wait_ms`, dropping the separate enqueue span), OR have the enqueue/dequeue
  pair collapse to one HopQueue per task. Owner: service/L1 + Principal.
- **Attribution note for F2/L8:** this package-`service` test is red **on purpose** as a P0 defect guard;
  it turns green when the queue emits a single lifecycle span. It is not an accidental/foreign break.

### Commands (executed, /tmp caches, go1.26.1)
```
gofmt -l anchor_queue_cardinality_test.go anchor_ingress_recorder_test.go   -> empty (exit 0)
go test -run RequestLoggerAnchor ./internal/middleware/                      -> ok (exit 0)
go test -run QueueHopEmitsExactlyOne ./internal/service/                     -> FAIL (exit 1, RED-by-design: 2 spans)
```
