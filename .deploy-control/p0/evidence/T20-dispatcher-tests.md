# T20 — Bounded queue drain (dispatcher) tests

- agent `Opus48#B` · lane `T20-WAVE` · pane `w6:p2` · task `T20-DISPATCHER-TESTS`
- check-in: `.deploy-control/p0/checkins/Opus48-B__T20-DISPATCHER-TESTS__20260723T221958Z.json`
- posture: added deterministic tests; product source READ-ONLY (only a new test file created).

## Target (bounded drain primitives, `internal/daemon/daemon.go`)

The bounded queue drain is `pollLoop` (daemon.go:2441) which drains claimable tasks on each wakeup up to
`MaxConcurrentTasks` using two isolable, deterministically-testable primitives:
- `newTaskSlotSemaphore(n) chan int` (daemon.go:2701) — buffered channel pre-populated with `n` distinct
  slot indices `[0,n)`; receive = acquire, send back = release. This is the hard capacity bound.
- `waitForTaskSlot(ctx, sem, wakeup, wait) (slot, acquired, woke, err)` (daemon.go:2669) — the bounded
  acquire: non-blocking try (`sem`/`ctx.Done`/default), then (if `wait>0`) a blocking
  `select` over `sem` (acquire) / `wakeup` (re-drain) / `ctx.Done` (cancel) / `timer` (at-capacity).
  `pollLoop` uses `acquired=false && !woke` as the "poll: at capacity" backpressure signal (daemon.go:2584).

Note: full `pollLoop` is not unit-isolable without heavy fixtures (server/DB/WS); the drain **contract**
(drain-to-capacity / never-exceed / cancel / wakeup-backpressure) is fully exercised at these primitives.

## Added tests — `internal/daemon/bounded_drain_test.go` (NEW; deterministic, no sleeps on the happy paths)

| Test | Proves |
|---|---|
| `TestNewTaskSlotSemaphoreCapacityAndTokens` | `cap==n`, `len==n`, `n` distinct slots `[0,n)`, empty after draining all (hard bound) |
| `TestWaitForTaskSlotDrainsUpToCapacityNeverExceeds` | acquires exactly `n` distinct slots; once empty, bounded wait → not-acquired (at capacity, limit never exceeded); `wait<=0` fast-path also not-acquired |
| `TestWaitForTaskSlotHonorsCancellation` | pre-cancelled ctx → ctx error via non-blocking select; cancel during the blocking wait → ctx error, not acquired |
| `TestWaitForTaskSlotHonorsWakeupBackpressure` | pre-signalled `wakeup` on an empty sem → `woke=true`, not acquired (loop re-drains) |
| `TestWaitForTaskSlotReleaseResumesDrain` | after acquiring the only slot (at capacity), releasing it lets the next wait re-acquire the same slot; `len` stays ≤ limit |

(Complements the pre-existing `TestNewTaskSlotSemaphoreReturnsStableSlotIndexes`, which also passed.)

## Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l internal/daemon/bounded_drain_test.go` | clean | 0 |
| `go vet ./internal/daemon/` | clean | 0 |
| `go test ./internal/daemon/ -run 'TaskSlotSemaphore|WaitForTaskSlot' -count=1` | 6/6 PASS (ok 0.031s) | 0 |
| `git diff --check …` | clean | 0 |

**Verdict: PASS.** Bounded drain acquires up to capacity, never exceeds the limit, and honors
cancellation and wakeup/backpressure deterministically.

## Non-claims / notes
- One product file created (test-only); no daemon source edited. daemon.go was unlocked and the new test
  file's path had zero overlap with any active record (verified) — lock extended beyond the initial
  evidence-only entry because the exact file path was determined during read-only investigation.
- `pollLoop` end-to-end (claim→dispatch across a live server/DB) is NOT driven here; the drain contract is
  proven at the `newTaskSlotSemaphore`/`waitForTaskSlot` primitive level. A loop-level test would require
  extracting the drain body or a full daemon fixture (handoff to L1 if desired).
- No inference/secret/deploy/commit/push; no OpenSpec checkbox closed.
