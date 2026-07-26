# P0 realtime UI terminal-delivery aggregation — implemented (GREEN)

- agent: Opus48#D · lane: realtime-delivery · task: P0-REALTIME-DELIVERY · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-REALTIME-DELIVERY__20260724T012117Z.json`
- ownership honored: edited ONLY `internal/realtime/**`. Did NOT touch cmd/server, middleware, service, daemonws, e2e contract, or other shared source.

## Changed files
- `internal/realtime/obs_delivery.go` (NEW): aggregate metadata-only `HopDelivery` emission in `realtime.Hub`.
- `internal/realtime/obs_delivery_test.go` (NEW): 10 focused tests.
- `internal/realtime/hub.go` (MINIMAL, 2 edits): added `deliveryObs *deliveryObserver` field to `Hub` + `deliveryObs: newDeliveryObserver()` in `NewHub`.

## Design / requirements coverage
- **Injectable recorder:** `Hub.SetDeliveryRecorder(*e2e.Recorder)` (nil disables emission).
- **Classify terminal only:** `EventTask{Completed,Failed,Cancelled}`; any other event → no span (`isTerminalDeliveryEvent`).
- **Canonical session:** `session_id = e2e.CanonicalSessionID(chat_session_id, task_id)`.
- **Genuine per-delivery ULID:** `newDeliveryULID` = `oklog/ulid/v2` `ulid.Make().String()`, minted at the delivery op (injectable in tests).
- **Exactly ONE aggregate span per task/event:** one span per broadcast regardless of tab count (all clients counted in one fan-out); per-`(session_id,event)` dedup (`deliveryObserver.seen`) makes Redis loopback / repeat calls a no-op; per-client `markSeen(eventID)` dedups the frame at the client layer.
- **Outcome:** `delivered` iff ≥1 client enqueued, else `dropped`; slow clients → `backpressure_count`/`drop_count` counters + `backpressure_state` label.
- **Content-free:** only closed delivery-hop keys (`delivery_latency_ms`, `drop_count`, `backpressure_count`, `backpressure_state`) + correlation IDs; the Hub never parses frame payloads — the caller (cmd/server listener, later) passes content-free `TerminalDeliveryMeta`. `e2e.ScanSpans` clean asserted in tests.
- **Bounded parsing/state:** no payload parse; dedup set FIFO-bounded at `deliveryDedupCapacity=4096`.
- **Retry-safe / no false success:** `(session,event)` marked seen ONLY after a successful `recorder.Emit`; a sink failure returns `false` and is retriable.
- **Concurrency safe:** check→emit→mark serialized under `deliveryObserver.mu` → exactly one span under concurrent same-task calls.
- **Seam:** `Hub.BroadcastTerminalDelivery(scopeType, scopeID, message, meta, eventID)` — fans out + aggregates counts + records the single span; non-terminal events are fanned out with no span.

## Results (go=/home/ec2-user/goroot/go/bin/go, GOCACHE=/tmp/kgc)
| Command | Result | Exit |
|---|---|---|
| `gofmt -l internal/realtime/{obs_delivery.go,obs_delivery_test.go,hub.go}` | empty (clean) | 0 |
| `go vet ./internal/realtime/` | clean | 0 |
| `go test ./internal/realtime/ -count=1` | `ok 0.424s` (whole package green) | 0 |

10 delivery tests PASS: ManyClientsEmitExactlyOneDeliveredSpan, ZeroClientsIsDropped,
SlowConsumerBackpressure, DedupRedisLoopbackOneSpan, ConcurrentSameTaskOneSpan,
RecorderFirstFailRetriesNoFalseSuccess, BoundedDedupState, CanonicalSessionID,
NoNonTerminalEmission, NilRecorderIsNoop.

## Review fixes (blocking) — applied 2026-07-24T01:3xZ
1. **Dedup key was session-scoped (collapse bug):** two distinct tasks in the SAME chat session both
   ending `completed` shared `sessionID|event` → the second span was wrongly deduped. Fixed:
   `dedupKey = rawTaskID|event` (+ reject empty `meta.TaskID`); the emitted span still JOINS on
   `CanonicalSessionID`. New test `TestTerminalDeliveryDistinctTasksSameSessionEmitTwoSpans` (same
   chat/session + 2 task ids → 2 spans, sharing one canonical session_id; same task replay → 1) and
   `TestTerminalDeliveryRequiresNonEmptyTaskID`.
2. **Broadcaster bypass:** naively calling the Hub-only method alongside a `DualWriteBroadcaster` would
   double-deliver local frames / bypass the Redis relay. Added optional `realtime.TerminalBroadcaster`
   interface `BroadcastTerminalToWorkspace(workspaceID, message, meta)`; `*Hub` implements it (local leg
   only, no relay event) with a compile-time `var _ TerminalBroadcaster = (*Hub)(nil)`. The central
   DualWrite implementation (LATER, redis_relay.go — not this lane) will implement the same interface,
   calling `local.BroadcastTerminalDelivery(...)` with ONE shared event ULID + `relay.PublishWithID`;
   the cmd/server listener type-asserts the interface for terminal events and falls back to a normal
   broadcast otherwise. No frame parsing. Test `TestHubImplementsTerminalBroadcaster`.
3. **No-subscriber drop_count:** was `dropped`/count 0; now the single aggregate frame that reached no
   client counts as `drop_count >= 1` (`delivered<=0 && backpressure==0 -> 1`). Test updated.
- Re-run: gofmt clean; `go vet ./internal/realtime/`=0; `go test ./internal/realtime/ -count=1`=0
  (`ok 0.435s`); 13 delivery/broadcaster tests PASS; `git diff --check` clean.

## Non-claims
- `-race` NOT run (gcc/cgo unavailable in env) — concurrency correctness covered by the serialized
  check→emit→mark design + the concurrent-same-task test.
- The cmd/server listener call site for `BroadcastTerminalDelivery` is a LATER wiring step (out of this
  lane's ownership); the typed seam is provided here.
- No deploy/inference/secret/commit; no OpenSpec checkbox.
