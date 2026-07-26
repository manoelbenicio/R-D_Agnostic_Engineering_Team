# T20-obs-collector — cross-process span-log collector (implemented)

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-OBS-COLLECTOR` · pane: `w7:p3`
- lock: `internal/daemon/observability/e2e/collector.go`, `collector_test.go` + this evidence file
- STATUS: **IMPLEMENTED + focused tests GREEN.** Collector reads BOTH process span logs, reconstructs
  `e2e.Span`, feeds `e2e.Assemble`; **dropped=0** proven for a durable, complete cross-process trace.

## 1. Design + acceptance

Backend and daemon are SEPARATE processes (no shared pointer sink). Each process:
- keeps an operational **in-memory ring** (`e2e.BoundedSink`, eviction-lossy) for live snapshots, AND
- writes a **durable per-span line** to its own export file via the durable sink (drop-free source of truth).

The collector merges the durable files across processes into the W5 assembler:
1. read each file line-by-line (bounded 4 MiB/line scanner — no busy loop, bounded memory);
2. reconstruct + fully re-validate each span line (fail-closed);
3. `e2e.Assemble(spans)` → per-task traces + orphans/anomalies.

**Acceptance:** `report.Dropped()==0` (the durable log records every validated span; nothing lost to ring
eviction) and `report.Assembly.AllContinuous` for a complete run.

## 2. IMPORTANT — durable-sink API changed under this lane (adapted)

The dispatch referenced `e2e.StructuredLogSink` (slog key/values, `e2e=true`, flat 9 IDs). During this
lane a concurrent edit **replaced it** with `e2e.JSONLSink` + `e2e.ParseSpanLine` (verified on disk in
`export_sink.go`):
- `JSONLSink` (`NewJSONLSink(io.Writer)`) writes each validated span as **one full-schema JSON object
  per line** (JSONL) — contract version, hop, nested `correlation` (all 9 IDs), **original
  StartedAt/EndedAt**, outcome/reason, http status, labels, counters, argv shape, secrets_present — and
  is fail-closed (nil/short-write → error). No `e2e=true` marker; the file is a dedicated span stream.
- `ParseSpanLine([]byte) (Span, error)` decodes the closed schema (`DisallowUnknownFields`), checks the
  contract version, and re-runs `Span.Validate()` (schema + structural leak scan).

The collector was built on the **actual current API** (`ParseSpanLine` per line), not the stale
`StructuredLogSink`. Because JSONL carries the original timestamps, no synthetic `StartedAt` is needed
(a slog-only export would have required one, since `Span.Validate` rejects a zero start time).

## 3. Implementation (`collector.go`)

- `CollectSpans(paths ...string) ([]Span, CollectStats, error)` — opens each JSONL export file, scans
  lines; blank / non-`{` lines are ignored (not drops); each `{`-prefixed line → `ParseSpanLine`;
  success → span, failure → `Dropped++` with a **bounded, value-free** reason (`classifyDrop`:
  `secrets_present` / `unsupported_contract` / `missing_required_id` / `bad_hop` / `unsafe_identifier` /
  `malformed_json` / `invalid_span`). No identifier/secret values are ever recorded.
- `AssembleFromLogs(paths ...string) (CollectorReport, error)` — collect then `Assemble`; pass every
  process's export path (order-independent; assembly joins on IDs).
- `CollectorReport.Dropped()` — the acceptance gate (0 for a durable, clean merge).
- `CollectStats` — metadata-only counts (`LinesTotal`, `SpanLines`, `Reconstructed`, `Dropped`,
  `DropReasons`); no contents.
- Bounds/safety: 4 MiB line cap; streaming scan (bounded memory); no goroutine/busy loop; fail-closed via
  `ParseSpanLine.Validate`; never logs span fields. Point it at the **dedicated** JSONL export files.

## 4. Validation (executed; /tmp GOCACHE/GOTMPDIR/GOMODCACHE, go1.26.1)

| Command | Result |
|---|---|
| `gofmt -l collector.go collector_test.go` | empty (exit 0) |
| `go vet ./internal/daemon/observability/e2e/...` | exit 0 |
| `go test -run 'Collect\|AssembleFromLogs' -v ./internal/daemon/observability/e2e/...` | **PASS** (exit 0): all 3 tests |

Tests (`collector_test.go`):
- `TestAssembleFromLogs_CrossProcessMerge_DroppedZeroAndContinuous` — a full 7-hop trace split across
  `backend.jsonl` (ingress/queue/persist/delivery) and `daemon.jsonl` (admission/cli/route), emitted via
  `JSONLSink`; asserts **Dropped()==0**, SpanLines/Reconstructed==7/7, `AllContinuous`, one trace with 7
  present hops. **PASS** (proves cross-process merge + dropped=0).
- `TestAssembleFromLogs_SecretsPresentLineFailsClosed` — a line declaring `secrets_present:true` →
  **Dropped==1** (reason `secrets_present`), not assembled. **PASS** (fail-closed).
- `TestCollectSpans_IgnoresBlankAndNonJSONLines` — blank/plain lines ignored (SpanLines/Dropped==0). **PASS**.

## 5. Foreign failure (separately evidenced; NOT this lane)

`go test ./internal/daemon/observability/e2e/...` (full package) currently reports **2 failures in
`export_sink_test.go`** (the concurrent JSONLSink lane's own tests):
`TestJSONLConcurrentAtomicLines` and `TestJSONLExportedSpansScanClean`, both failing with
`invalid span: span label rejected: key is not in approved metadata set`. Cause: those tests build spans
with a label key that the current `contract.go allowedLabelKeys` allowlist rejects — a mid-edit mismatch
between a concurrent lane's `export_sink_test.go` and `contract.go`. **Out of this lane's ownership;** not
caused by and unrelated to `collector.go`/`collector_test.go` (which pass in isolation, gofmt/vet clean).
Route to the JSONLSink/contract owner. My focused run scoped to `Collect|AssembleFromLogs` is green.

## 6. Non-claims / limitations
- Implemented `collector.go` + `collector_test.go` only; did not edit `export_sink.go`/`contract.go`/other
  hop files (concurrent-lane owned). `git diff --check`: new untracked files, clean.
- Collector assumes the **dedicated** JSONL export file per process (JSONLSink's writer target); a
  co-mingled app-log line that is `{`-JSON-but-not-a-span would be a drop (by design/fail-closed).
- Wiring `JSONLSink` into the two live processes' startup (which writer/file each uses) is the W1-serial /
  observability-lane injection step (see `T20-obs-wiring-inventory.md`), gated behind P0 (D-V3-21); this
  lane delivers the collector/reader, not the process wiring or a live run.
- No deploy, no inference, no secret; end-to-end live capture still gated (tier-20 / `live_runs=false`).
