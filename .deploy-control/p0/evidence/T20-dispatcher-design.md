# T20-dispatcher-design — daemon wakeup/pickup drain to capacity (design only)

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-DISPATCHER-DESIGN` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/T20-dispatcher-design.md`
- MODE: **DESIGN ONLY** (read-only; no edits, no deploy, no inference). Preserve cancellation/backpressure; no busy-loop; no unbounded launches.

> ROOT CAUSE of "~3 tasks/poll": the per-runtime poller claims **one task per server round-trip**, and on
> the **first transient `task==nil`** (common mid-burst while rows are still committing / when the poller
> outruns enqueue) it **sleeps a full `PollInterval` (30s)**; meanwhile the per-runtime wakeup channel is
> **coalescing/lossy** (`signalTaskWakeup` is a non-blocking send with `default:`), so the wakeups fired
> for the other already-queued tasks **while the poller was busy are dropped** — the 30s nap is not
> interrupted. Net throughput collapses to a handful of tasks per 30s cycle instead of draining to the
> 20 free execution slots. A single-agent burst (T20 targets one AGENT ⇒ one runtime ⇒ one poller) makes
> claiming serial through one goroutine, amplifying it.

## 1. Current loop (verified: `daemon.go runRuntimePoller` 2557-2653 + helpers)

Per runtime `rid`, shared semaphore `sem` (`newTaskSlotSemaphore(MaxConcurrentTasks=20)`), per-runtime
`wakeup <-chan struct{}`:
```
loop:
  slot,acquired,woke = waitForTaskSlot(sem, wakeup, taskSlotWaitTimeout=2s)   // 1 slot
  if !acquired: at capacity → if woke continue else sleep capacityBackoff(≤5s); continue
  if !tryEnterClaim(): return slot; sleep PollInterval(30s|wakeup); continue    // auto-update barrier
  task = client.ClaimTask(rid)                                                 // ONE task / round-trip
  if err:  exitClaim; return slot; sleep PollInterval(30s|wakeup); continue
  if task==nil: exitClaim; return slot; sleep PollInterval(30s|wakeup); continue   // ← the 30s nap
  go handleTask(task, slot)  ; // loop immediately
```
Facts: `DefaultPollInterval=30s`, `DefaultMaxConcurrentTasks=20` (`config.go`); `signalTaskWakeup`
(`wakeup.go:321`) = `select { case ch<-…: default: }` (**drops** when not immediately received);
`sleepWithContextOrWakeup` wakes on one buffered signal or the 30s timer; `waitForTaskSlot` blocks ≤2s.

## 2. Why ~3/poll (three compounding factors)

1. **Serial single-claim:** `ClaimTask` returns one row; the loop only advances by one server round-trip
   at a time (latency-bound), so peak pickup rate ≪ 20 within a burst.
2. **30s nap on first transient empty:** during a 20-burst the rows commit over a short window; the
   poller claims those visible now (≈3), the next `ClaimTask` returns nil (not-yet-visible / outran
   enqueue) → the poller parks for up to `PollInterval=30s`.
3. **Lossy/coalescing wakeup:** the other 17 tasks' `signalTaskWakeup` calls fire while the poller is
   busy (spawning/claiming) or already has a pending signal → **dropped** (`default:`), so the 30s nap
   is not cut short. The queue drains ~one small batch per 30s.

## 3. Fix — bounded drain to available lease capacity on wakeup

Three coordinated changes; Fix 1+2 are daemon-local and sufficient; Fix 3 is an optional throughput boost.

### Fix 1 — level-triggered (lossless) wakeup (replace lossy edge signal)
Back the wakeup with a **monotonic generation counter** so a signal fired while the poller is busy is
never lost; the buffered channel is kept only to wake a sleeper early.
```go
type wakeSignal struct {
    gen atomic.Uint64
    ch  chan struct{} // buffered(1); best-effort early-wake only
}
func (w *wakeSignal) bump() {                 // called by NotifyTaskAvailable / server WS wakeup
    w.gen.Add(1)
    select { case w.ch <- struct{}{}: default: } // early-wake hint; loss here is harmless (gen is authoritative)
}
func (w *wakeSignal) generation() uint64 { return w.gen.Load() }
```
The poller treats `gen` as the authoritative "work may exist" level; `ch` only shortens sleeps.

### Fix 2 — bounded drain-to-capacity step (refactor the poller body)
```go
// drainRuntimeQueue claims and launches tasks until the queue is empty, the
// shared slot pool is exhausted (backpressure), or a hard budget is reached.
// It NEVER launches more than cap(sem) tasks and NEVER blocks on a full pool.
func (d *Daemon) drainRuntimeQueue(pollerCtx, parentCtx context.Context, rid string, sem chan int, taskWG *sync.WaitGroup) (claimed int) {
    budget := cap(sem) // == MaxConcurrentTasks (tier-20 gated) → hard upper bound; no unbounded launches
    for claimed < budget {
        if pollerCtx.Err() != nil { return claimed }
        slot, ok := tryAcquireSlot(sem)          // NON-BLOCKING receive; !ok ⇒ at capacity ⇒ stop (backpressure)
        if !ok { return claimed }
        if !d.tryEnterClaim() { sem <- slot; return claimed }   // auto-update barrier preserved
        task, err := d.client.ClaimTask(pollerCtx, rid)         // (or ClaimTasks batch — Fix 3)
        if err != nil {
            d.exitClaim(); sem <- slot
            if isRuntimeNotFoundError(err) { go d.handleRuntimeGone(rid) } // unchanged recovery
            return claimed // caller logs + interval-sleeps
        }
        if task == nil { d.exitClaim(); sem <- slot; return claimed }      // queue empty ⇒ stop drain
        taskWG.Add(1); d.activeTasks.Add(1)
        go func(t Task, slot int) {                                        // identical launch/teardown as today
            defer taskWG.Done(); defer d.exitClaim(); defer d.activeTasks.Add(-1); defer func(){ sem <- slot }()
            d.handleTask(parentCtx, t, slot)
        }(*task, slot)
        claimed++
    }
    return claimed // hit hard budget ⇒ stop
}
// tryAcquireSlot: select { case s := <-sem: return s,true; default: return 0,false }
```

### Fix 2b — poller main loop rewired (no busy-loop, lossless)
```go
for {
    if pollerCtx.Err() != nil { return }
    startGen := d.wake.generation()
    _ = d.drainRuntimeQueue(pollerCtx, parentCtx, rid, sem, taskWG)  // bounded, backpressured
    if d.wake.generation() != startGen {
        continue                              // new work arrived DURING the drain → re-drain now (no nap)
    }
    // Empty queue OR at capacity: sleep the interval, wakeable early by ch or a freed slot.
    if err := sleepWithContextOrWakeup(pollerCtx, d.cfg.PollInterval, d.wake.ch); err != nil { return }
}
```
- **No busy-loop:** after a drain that finds nil *and* no generation advance, the poller sleeps the full
  `PollInterval` (interruptible). The `generation()!=startGen` re-drain only fires when real new work was
  enqueued during the drain — bounded, self-terminating (eventually gen stops advancing → nap).
- **Drains to capacity:** one wakeup drains until nil or all free slots are used, up to the 20-slot cap.
- **Capacity/backpressure:** `tryAcquireSlot` is non-blocking → at capacity the drain stops immediately;
  the poller naps until a slot frees or a wakeup. Optional: have the slot-release `defer sem<-slot` also
  `d.wake.ch`-notify so a freed slot promptly reclaims the next queued task (bounded, still capped).

### Fix 3 — OPTIONAL throughput: batch claim `ClaimTasks(ctx, rid, limit)`
Add a server API + client method + generated SQL that atomically claims up to `limit = freeSlots(sem)`
queued tasks in ONE round-trip (e.g. `UPDATE agent_task_queue SET status='dispatched', … WHERE id IN
(SELECT id FROM agent_task_queue WHERE runtime-eligible AND status='queued' ORDER BY priority, created_at
LIMIT $2 FOR UPDATE SKIP LOCKED) RETURNING …`). The daemon calls it once per drain with the free-slot
count, removing serial per-task round-trip latency so a 20-burst drains in a single call. **Scope note:**
this spans server handler + `daemon/client.go` + `pkg/db` generated SQL (owned by the query/handler
owner) — flag/handoff; Fix 1+2 already eliminate the 30s-nap collapse without it.

## 4. Invariants preserved (self-check)
- **Cancellation:** `handleTask` still runs under `parentCtx` with the existing `cancelPollInterval`(5s)
  cancel poll; `pollerCtx` cancellation still exits both the drain and the loop; slots still released via
  `defer sem <- slot`. Unchanged.
- **Backpressure:** non-blocking `tryAcquireSlot` stops the drain at capacity — dispatched rows never
  exceed free execution slots (preserves the current "don't push into dispatched then race the dispatch
  timeout" property).
- **No unbounded launches:** hard `budget = cap(sem) = MaxConcurrentTasks` (tier-20 gated via
  `effectiveTaskAdmissionLimit`); a single drain launches ≤ free slots; the shared `sem` bounds all
  runtime pollers globally.
- **No busy-loop:** empty+no-gen-advance ⇒ full `PollInterval` sleep; re-drain only on a real generation
  advance.
- **Auto-update barrier:** `tryEnterClaim`/`exitClaim` retained around each claim.
- **Tier gate + poll fallback:** MaxConcurrentTasks stays tier-gated; the WS-fallback polling path still
  functions (interval nap is the fallback; the generation counter makes it lossless).

## 5. Focused tests (author with the owning edit)
- `daemon_test.go`: with 20 queued tasks + 20 free slots + a stub `ClaimTask` returning tasks then nil,
  one wakeup **drains all 20** (not ~3); with 20 tasks + 5 free slots, drains exactly 5 then stops
  (backpressure), and drains the rest as slots free.
- Lossless wakeup: bump `gen` during an in-flight drain ⇒ poller re-drains without waiting the interval
  (no lost wakeup); no bump ⇒ poller sleeps the interval (no busy-loop).
- Cancellation: `pollerCtx` cancel mid-drain stops promptly; slots released; `taskWG` drains.
- Never launches > cap(sem) even if `ClaimTask` keeps returning tasks (hard budget).
Run focused: `go test ./internal/daemon -count=1 -run 'Poller|Drain|Wakeup|Slot'` + `go vet`.

## 6. Non-claims / limitations
- Design only — no source edited, no tests run, no deploy, no inference/secret.
- Fix 3 (batch claim) spans server/client/generated-SQL owned outside the daemon-loop lane → handoff, not assumed.
- Exact `~3` figure is environment/latency-dependent; the mechanism (serial claim + 30s nap on transient nil + lossy wakeup) is the verified cause and is what the design removes.
- End-to-end 20-drain confirmation needs the gated tier-20 live run (`live_runs=false`, `6.3`; owner Principal `w5:p9` + operator).
