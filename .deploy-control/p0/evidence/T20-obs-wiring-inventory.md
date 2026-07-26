# T20-obs-wiring-inventory — no-op hop call sites + bounded metadata-only sink design

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-OBS-WIRING-INVENTORY` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/T20-obs-wiring-inventory.md`
- MODE: **DESIGN/INVENTORY ONLY** (read-only; no edits, no deploy). OBS wiring is Priority-2/DEFERRED behind P0 (D-V3-21) → design gated OFF by default.

> FINDING: every end-to-end hop owner emits through an injectable `*e2e.Recorder` (or a package setter)
> that **defaults to nil → `discardSink{}` (no-op)**, and **production startup installs no real sink** —
> the sink/recorder setters are exercised **only in tests**. So all six trace hops validate + leak-scan
> spans and then **discard** them. Design below adds one bounded, metadata-only `RingSink` and a single
> shared recorder injected at boot across the backend+daemon seams, gated by a default-OFF flag.

## 1. e2e recorder/sink API (source: `internal/daemon/observability/e2e/recorder.go`)

- `type Sink interface { Record(Span) error }`.
- `func NewRecorder(sink Sink) *Recorder` — **nil sink ⇒ `discardSink{}` (no-op)**, so instrumentation
  never crashes a caller.
- `Recorder.Emit(*Span)` — clones, **`Validate()` + single-span structural leak scan (fail-closed)**,
  then `sink.Record(clone)`. The sink only ever receives **validated, leak-free, metadata-only** spans.
- Existing sinks: `discardSink{}` (no-op) and `MemorySink` (**unbounded** `append` slice — tests/assembler
  only; not production-safe for a live ring).

## 2. Hop-owner call-site inventory (current no-op wiring)

| # | Hop | Owner seam (file) | Emit API | Current production state |
|---|---|---|---|---|
| 1 | Ingress HTTP handler | `internal/middleware/request_logger.go` — pkg var `ingressRecorder *e2e.Recorder` (nil until `SetIngressRecorder`), `EmitIngress` | ingress span | **no-op** — `request_logger.go:142` "No-op until SetIngressRecorder is wired"; `SetIngressRecorder(...)` called **only in `request_logger_test.go`** |
| 2 | Queue enqueue | `internal/service/task.go:52` `Obs *e2e.Recorder`; `internal/service/obs_queue.go` `EmitQueue` | queue span | **no-op** — `TaskService{Obs: e2e.NewRecorder(sink)}` set **only in `*_test.go`**; production TaskService leaves `Obs` nil |
| 3 | CLI launch | `internal/daemon/daemon.go:159` `cliObs *e2e.Recorder`; `internal/daemon/cli_observability.go` `HopCLI` | cli span | **no-op** — no production assignment of a real recorder to `cliObs` (tests use `e2e.NewRecorder(sink)`) |
| 4 | Route selection | `internal/daemon/gateway/executor.go` + `gateway/obs_span.go` `EmitProviderSpan(rec, …)` | provider/route span | **no-op** — recorder supplied a real sink **only in `executor_test.go`/`obs_span_test.go`** |
| 5 | Persist result row | `internal/service/task.go` `Obs`; `internal/service/obs_persist.go` `EmitPersist` | persist span | **no-op** — same seam as #2 (production `Obs` nil) |
| 6 | WS delivery | `internal/daemonws/hub.go:171` `delivRecorder *DeliveryRecorder` (nil until `SetDeliveryRecorder`); `obs_delivery.go` `NewDeliveryRecorder(*e2e.Recorder)` / `EmitDelivery` | delivery span | **no-op** — `SetDeliveryRecorder(...)` called **only in `hub_test.go`**; `NewDeliveryRecorder(nil)` is nil-safe |

Related (already wired, separate — NOT an e2e trace hop): `daemon.go:283`
`brain.NewAdmissionObserver(brain.NewAdmissionLogSink(logger))` (admission → log sink), not the e2e ring.

**Conclusion:** all six trace hops are functionally `NewRecorder(nil)`/nil in production (no startup
install), so no HopTrace is ever assembled live. Each seam is **nil-safe** and **additive** to wire.

## 3. Design — bounded metadata-only RingSink (no content/secrets, bounded memory, no busy loop)

New file, e.g. `internal/daemon/observability/obsexport/ring.go` (or under the reserved `promexport`
tree per D-V3-23). Implements `e2e.Sink`:
```go
type RingSink struct {
    mu      sync.Mutex
    buf     []e2e.Span   // fixed-capacity ring (newest-wins)
    head    int
    size    int
    capN    int
    dropped uint64        // metadata counter: spans overwritten on overflow
    seen    uint64        // total accepted
}
func NewRingSink(capacity int) (*RingSink, error) // capacity bounded (e.g. 1..65536; default 4096)

// Record is O(1), lock-guarded, allocation-free steady-state, and NEVER blocks
// the caller (no backpressure on the hot path). When full it overwrites the
// oldest span and increments `dropped`. No background goroutine ⇒ no busy loop.
func (r *RingSink) Record(s e2e.Span) error {
    r.mu.Lock(); defer r.mu.Unlock()
    if r.size == r.capN { r.dropped++ } else { r.size++ }
    r.buf[r.head] = s; r.head = (r.head + 1) % r.capN; r.seen++
    return nil
}
func (r *RingSink) Snapshot() []e2e.Span // deep copy for the assembler/exporter (pull)
func (r *RingSink) Stats() (seen, dropped uint64, size int)
```
Guarantees:
- **Bounded memory:** exactly `capN` spans retained (ring), unlike `MemorySink`'s unbounded `append`.
- **Metadata-only / no secrets:** upstream `Recorder.Emit` already `Validate()`+leak-scans before
  `Record`; the sink stores the validated span verbatim and **adds/logs nothing** (never prints span fields).
- **No busy loop:** `Record` is synchronous O(1); retrieval is **pull** (`Snapshot`), no polling goroutine.
- **Non-blocking / lossy-by-design:** overflow overwrites oldest (records `dropped`) rather than blocking
  a request handler or daemon goroutine — instrumentation must never stall the hot path.
- **Thread-safe:** one mutex; safe for concurrent backend handlers + daemon goroutines.

## 4. Exporter (pull-based; bounded cardinality; no busy loop)

- **Primary (no goroutine):** the trace assembler / OBS batch scanner (`ScanSpans`) and an on-demand
  reader consume `RingSink.Snapshot()` to build HopTraces and continuity/gap metrics on request.
- **Optional metrics (pull/scrape):** derive bounded-cardinality counters from the ring on scrape
  (per-hop counts, `dropped`, trace-continuity ratio) via the reserved `promexport` contract
  (`obs_hop_*`, `obs_trace_continuous_ratio`, `obs_leak_scan_failures_total`). **Prohibited labels:** ids,
  pseudonyms, free-form, any high-cardinality label (D-V3-23) — pseudonyms stay trace-only.
- If async delivery is ever added it MUST be a bounded worker blocked on a channel `select` (never a
  spin/poll loop); the default design is pull-only.

## 5. Shared-sink injection across backend + daemon (exact seams)

The six hops run across the backend HTTP path (ingress, queue/persist, WS delivery) and the daemon
(CLI launch, route selection) — same process family. Inject **one** shared recorder over **one** shared
`RingSink` at boot so all hops of a task land in the same ring (the assembler joins them by the 9 IDs):
```go
// startup (gated OFF by default):
if cfg.ObsE2ESinkEnabled {                 // e.g. env OBS_E2E_SINK_ENABLED=1 (default false)
    sink, _ := obsexport.NewRingSink(cfg.ObsRingCapacity /*default 4096*/)
    rec := e2e.NewRecorder(sink)           // one shared recorder
    middleware.SetIngressRecorder(rec)                       // hop 1
    taskService.Obs = rec                                    // hops 2 + 5 (queue + persist)
    hub.SetDeliveryRecorder(daemonws.NewDeliveryRecorder(rec))// hop 6
    daemon.cliObs = rec                                      // hop 3 (CLI launch)
    gatewayExecutor.SetRecorder(rec)                         // hop 4 (route selection) — add setter if absent
}
// when disabled: leave every seam nil/NewRecorder(nil) → discard (current behavior, unchanged).
```
- **Ownership:** the injection call-sites are **W1-serial shared anchors** (`daemon.go`/server boot,
  `TaskService` construction, `hub`, middleware setter, executor wiring) per FILE_OWNERSHIP → W1 wires the
  single shared recorder. The `RingSink`/exporter file is a **new** observability-lane file (W5/W4 per the
  span-contract/dashboards split; `promexport` tree reserved by D-V3-23).
- **Additive + reversible:** flag default OFF preserves today's no-op; enabling installs the ring; no hop
  owner code changes beyond receiving the shared recorder through its existing seam.

## 6. Invariants / non-claims
- Metadata-only + fail-closed enforced by `Recorder.Emit` (unchanged); the sink adds no content, logs nothing.
- Bounded memory (ring cap), bounded-cardinality export, **no busy loop** (sync O(1) Record + pull export),
  non-blocking hot path (overflow drops oldest with a counter, never blocks a handler/daemon).
- One shared sink so a task's 6 spans co-locate for assembly; single-flight/cancel/backpressure of the
  hot paths are untouched (Record never blocks).
- **Design/inventory only** — no source edited, no deploy, no inference/secret; OBS enablement is
  Priority-2/DEFERRED behind P0 (D-V3-21) and requires explicit authorization + the W1-serial wiring.
- Backend and daemon are the same process family here; if ever split into separate processes, the shared
  in-memory ring does not span processes — that case needs a cross-process exporter (flag; out of scope now).
