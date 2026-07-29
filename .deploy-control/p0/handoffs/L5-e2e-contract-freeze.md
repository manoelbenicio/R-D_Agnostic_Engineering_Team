# L5 — E2E correlation contract: PUBLISHED + FROZEN (metadata-only)

- agent: **Opus48#C** · lane **L5** · task **P0-L5-E2E-CORRELATION-FREEZE** · pane `w8:p1`
- exclusive ownership: `server/internal/daemon/observability/e2e/**` (this handoff = namespaced output, not product)
- check-in receipt: `.deploy-control/p0/checkins/CHECKIN__Opus48-C__L5__P0-L5-E2E-CORRELATION-FREEZE__20260722T040401Z.json`
- OpenSpec 6.2 · AB-REQ-39/40 · G4-OBS (OBS-1/OBS-9/OBS-10)
- as-of (UTC): `2026-07-22T04:08Z` · HEAD `a6d50986…`
- **FROZEN contract version: `agent-brain.e2e.v1`** — L6/L7/L1 MUST consume this API; do NOT invent or fork it.
- constraints honored: metadata-only; no product edit outside `e2e/**` (in fact **zero** product edits — package already freeze-quality); no deploy/inference/secret/commit; no anchors/dashboards/central/gateway/runtimeenv/adapters/OpenSpec touched.

## 0. Freeze verdict

The `e2e/**` library is **complete, correct, and verified green** against all L5 acceptance criteria. **No product change was required** ("minimal verifiable changes only" → zero edits). The public API below is **frozen** at `agent-brain.e2e.v1`. L6 (ingress/queue/persist) and L7 (WS/UI delivery) build against it; L1 consumes it at the shared anchors during Wave-C serial integration.

## 1. Verification evidence (exact commands + exit codes)

Run from `multica-auth-work/server`, `GOROOT=/home/ec2-user/goroot/go`, `GOCACHE=/tmp/l5-gocache`, `GOPROXY=off`:

| Command | Result | Exit |
|---|---|---|
| `gofmt -l internal/daemon/observability/e2e/` | (empty — clean) | 0 |
| `go vet ./internal/daemon/observability/e2e/` | clean | 0 |
| `go test ./internal/daemon/observability/e2e/ -count=1` | `ok … 0.004s` | 0 |
| `go test … -count=1 -v` | **38 top-level tests PASS** (88 RUN/PASS lines incl. subtests) | 0 |
| `CGO_ENABLED=1 go test -race …` | **NOT_AVAILABLE** — `gcc` absent, cgo cannot build (environment limit, not a code defect) | (env) |

`git diff --check` on `e2e/**`: **PASS** (no whitespace errors; no product bytes changed by L5).

## 2. Frozen public API surface (what L6/L7/L1 call)

**Version gate** — `ContractVersion = "agent-brain.e2e.v1"`; `SupportedContractVersion(v) bool`. Consumers MUST reject unsupported versions.

**Hops** (`HopKind`) — `HopIngress`(1,W6) `HopQueue`(2,W7) `HopAdmission`(3,W1) `HopCLI`(4,W3) `HopRoute`(5,W2) `HopPersist`(6,W7) `HopDelivery`(7,W6) `HopTrace`(8,W5 assembler). `EmittingHops()` (1–7), `OrderedHops()` (1–8).

**Identifiers** (`IDField`) — `request_id, queue_msg_id, task_id, session_id, launch_id, proc_id, omni_request_id, result_id, delivery_id`. Struct `Correlation{...}` (+ `Get(IDField)`, `Validate()`, `ToCarrier() map[string]string`, `CorrelationFromCarrier(map)`); carrier headers `X-AB-*` + `HeaderContractVersion`.

**Span** — `NewSpan(hop, corr) *Span`; builders `WithLabel/WithCounter/WithOutcome/WithHTTPStatus/WithArgvShape/Finish`; `DurationMs()`; `Validate()`. Fields are metadata-only; **there is no free-form content field**; `SecretsPresent bool` must stay false (enforced).

**Emit path** — `Sink` interface (`Record(Span) error`); `NewRecorder(sink) *Recorder`; `Recorder.Emit(*Span) error` (validates + fail-closed refuse, deep-clones to prevent aliasing); `NewMemorySink()` (+ `Record/Spans/Len`) for tests/assembler/harness.

**Assembler (OBS-9)** — `Assemble([]Span) AssemblyReport` / `AssembleFromSink(*MemorySink)`. Returns `Traces []Trace` (Present/Missing/Continuous/Anchor), `Orphans []OrphanSpan`, `Anomalies []AssemblyAnomaly` (`duplicate_span`/`conflicting_join`), `AllContinuous bool` (true only when every task has all 7 emitting hops, zero orphans, zero anomalies).

**Leak scan (OBS-10)** — `ScanSpans([]Span) ScanReport` / `ScanFromSink`; `ScanEvents([]Event)`; `ScanLogLines([]string)` **rejects every non-empty free-form line** (fail-closed). `ScanReport{Clean, Scanned, Findings}` — reasons never echo offending values.

**Descriptor (OBS-1)** — `Descriptor() ContractDescriptor` (serializable: version, hops, identifiers, joins, carriers, `secrets_invariant`).

**Synthetic (OBS-9 acceptance)** — `SyntheticTraceSpans(taskID) []Span`; `EmitSyntheticTask(rec, taskID) error`.

## 3. FROZEN join contract (per-hop required IDs = `RequiredIDs(hop)`)

| Hop | Required IDs | Assembler join to task |
|---|---|---|
| ingress | `request_id, task_id` | direct (`task_id`); also anchors `request_id → task` |
| queue | `queue_msg_id, task_id` | direct (`task_id`) |
| admission | `task_id, session_id, launch_id` | direct (`task_id`); anchors `launch_id → task`, `session_id → task`; sets trace Anchor |
| cli | `launch_id, proc_id` | via `launch_id → task` (from admission) |
| route | `request_id, omni_request_id` | via `request_id → task` (from ingress) |
| persist | `task_id, result_id` | direct (`task_id`) |
| delivery | `session_id, delivery_id` | via `session_id → task` (from admission) |

Conflicting anchors (same key → two tasks) and duplicate task-hop spans are recorded as `Anomalies` and force `AllContinuous=false` (fail-closed). Materialization is deterministic (`sort.Strings(tasks)`).

## 4. Metadata-only guardrails L6/L7 MUST obey (enforced by `Validate`/leak scan)

- Labels: only the **closed** `allowedLabelKeys` set; values are charset-checked per kind (`code`/`path`/`route_model`/`pseudonym`). Pseudonyms must be `principal_`/`acct_`/`conn_`-prefixed hex digests — never raw identity/email/connection string.
- Counters: **closed per-hop** `allowedCounterKeys` (e.g. ingress→`latency_ms`; queue→`wait_ms/queue_depth/enqueue_unix_ms/dequeue_unix_ms`; persist→`persist_latency_ms/byte_count/token_count`; delivery→`delivery_latency_ms/drop_count/reconnect_count/backpressure_count`). A counter valid for one hop is rejected on every other hop. Non-negative only.
- Argv: only `allowedArgvShapeTokens` (`subcommand`, `flag`, `flag=<redacted>`, `arg=<redacted>`, `path=<redacted>`, `value=<redacted>`) — shape, never values.
- `safeID`/`safeCode` charset excludes `/ + = @` and whitespace → URL/base64/email/JWT/connection-string shapes are structurally rejected; `detectSecretMarkers` adds marker/JWT-structure defense-in-depth.
- **No free-form message/body/error field exists on `Span` or `Event`.** Emit content → `Recorder.Emit` returns an error and the span is never recorded.

## 5. Usage example (metadata-only; no sensitive values) — for L6/L7

```go
rec := e2e.NewRecorder(sink) // sink owned by the caller; nil → discard
sp := e2e.NewSpan(e2e.HopIngress, e2e.Correlation{RequestID: reqID, TaskID: taskID}).
    WithLabel("method", "POST").
    WithLabel("route_template", "/v1/tasks").
    WithCounter("latency_ms", ms).
    WithHTTPStatus(202).
    WithOutcome("accepted", "ok").
    Finish()
if err := rec.Emit(sp); err != nil {
    // fail-closed: span refused; classify err WITHOUT printing values
}
```
Propagation across a hop boundary: `carrier := corr.ToCarrier()` (headers/metadata) → downstream `e2e.CorrelationFromCarrier(carrier)` (validates version + charset before trust).

## 6. Boundaries / handoff notes
- L6/L7 own only their helper files; **call sites in shared anchors** (`internal/metrics/http.go`, `internal/daemonws/hub.go`, `internal/service/task.go`) are **W1-serial** (Wave C) — L6/L7 emit helpers and hand the anchor insertions to L1, per FILE_OWNERSHIP.
- L5 introduces **no dependency** beyond the standard library (doc.go), keeping the contract stable for every caller.
- Nothing here authorizes deploy/inference/live-run; OBS spans are metadata-only and secret-free by construction.

## 7. Agent status
- STATUS: DONE (contract published + frozen; verified green)
- DELIVERED: freeze of `agent-brain.e2e.v1` public API (§2), join contract (§3), guardrails (§4), usage (§5); verification evidence (§1); zero product edits (package already correct).
- FILES CHANGED: none (product). FILE CREATED: this handoff only.
- VALIDATION: gofmt PASS, vet PASS(0), test PASS(0, 38 tests), race NOT_AVAILABLE (no gcc/cgo), `git diff --check` PASS.
- NON-CLAIMS: no `-race` run (cgo unavailable); no live run/inference; no caller/anchor edited; L6/L7 finalization not done here.
- HANDOFF: L6/L7 build against §2/§3/§4/§5; L1 integrates anchors serially in Wave C.
