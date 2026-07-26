# T20-final-obs-integration — server e2e observability wired to green

- agent: `Codex56#A` · task: `P0-FINAL-OBS-INTEGRATION` · pane: `w7:p3`
- ownership (edited only): `cmd/server/{main.go,listeners.go,e2e_recorder.go,boot_recorder_wiring_test.go,listeners_terminal_test.go}`,
  `internal/realtime/redis_relay.go` + `redis_relay_terminal_test.go`, `internal/daemon/observability/e2ewiring/production_anchor_test.go`.
- read-only (not edited): middleware, service, daemon, realtime hub/obs_delivery, daemonws.

## Deliverables

1. **hub delivery recorder at startup + testable wiring helper** (`e2e_recorder.go`, `main.go`):
   `installServerObservability(hub, taskServices...)` / `installServerObservabilityWith(rec, hub, taskServices...)`
   install the recorder on ingress (`middleware.SetIngressRecorder`), BOTH `TaskService.Obs` (HTTP +
   sweeper), and the **realtime `hub.SetDeliveryRecorder`** (previously missing). main() now calls the
   helper; `serverObs.closeFn()` owns the 0600 JSONL file on every exit path; removed now-unused
   `middleware` import.
2. **DualWrite terminal broadcast** (`redis_relay.go`): `DualWriteBroadcaster.BroadcastTerminalToWorkspace`
   mints ONE ULID, `injectEventID` for the local frame, `local.BroadcastTerminalDelivery(ScopeWorkspace,
   ws, frame, meta, id)` (local leg + single aggregate HopDelivery span), then `relay.PublishWithID(...,
   same id)` — no local duplicate, no relay bypass. `var _ TerminalBroadcaster = (*DualWriteBroadcaster)(nil)`.
3. **terminal-only listener dispatch** (`listeners.go`): in `registerListeners` SubscribeAll, ONLY
   `task:completed/failed/cancelled` with non-empty `TaskID`+`WorkspaceID` type-assert
   `realtime.TerminalBroadcaster` and call `BroadcastTerminalToWorkspace(ws, data, meta{TaskID,
   ChatSessionID, Event})` (content-free meta); all other events + terminal-without-capability use the
   normal workspace broadcast unchanged. Added `isTerminalTaskEvent`.
4. **boot test rewritten** (`boot_recorder_wiring_test.go`): removed `t.Skip`; real ungated assertions —
   TaskService.Obs installed; driving the real realtime Hub terminal seam records a HopDelivery span;
   driving the real `RequestLogger` handler records a HopIngress span; the file wrapper owns a 0600
   export file with an idempotent closer. (Adapted to the current holder-based
   `SetIngressTaskID(ctx, taskID)` signature after a concurrent middleware edit.)
5. **terminal seam tests**: `internal/realtime/redis_relay_terminal_test.go` (DB-free, RUNS) proves via
   single Hub + DualWrite fake relay: exactly one span + one relay publish on the canonical session, all
   3 terminal kinds, non-terminal fans out with no span, same-chat-two-tasks → 2 spans.
   `cmd/server/listeners_terminal_test.go` drives the real `registerListeners` call site (DualWrite +
   fake relay, single Hub, no daemonws) with the same assertions + terminal-without-task_id falls back to
   normal broadcast.
6. **production-anchor Assemble** (`e2ewiring/production_anchor_test.go`, DB-free, RUNS): drives the seven
   REAL exported emitters (middleware.EmitIngress, service.EmitQueue/EmitPersist, brain admission observer,
   daemon.EmitCLI, gateway.EmitProviderSpan) plus the **realtime terminal delivery seam** (not daemonws),
   with production-derived join keys (admission's own launch_id read back; delivery joined via
   `e2e.CanonicalSessionID`) — no synthetic/backfill IDs — and asserts exactly one span per 7 hops, all 9
   IDs, `AllContinuous`, 1 trace, 0 orphans/anomalies.
7. **skips/stale removed**: boot `t.Skip` gone. Server e2e delivery hop is `realtime.Hub` (browser WS);
   daemonws is a separate transport and remains read-only (its comments not edited per the read-only
   constraint — noted here rather than modified).

## Verification (executed; go1.26.1, /tmp GOCACHE/GOTMPDIR/GOMODCACHE)

| Check | Result |
|---|---|
| `gofmt -l` (all owned/new files) | empty |
| `go vet ./cmd/server ./internal/realtime ./internal/daemon/observability/e2ewiring` | exit 0 |
| `go build ./cmd/server` / `./cmd/multica` | exit 0 / exit 0 |
| `go test ./internal/realtime` (incl. 4 terminal tests, ran not skipped) | ok |
| `go test ./internal/daemon/observability/e2ewiring` (anchor PASS) | ok |
| `go test ./internal/middleware` | ok |
| `go test ./internal/daemon/observability/e2e` | ok |
| `go test ./internal/service` | ok |
| `go test ./internal/daemon` | ok (49.4s) |
| `go test ./cmd/server` | ok (compiles; package-main tests execute only under the pre-existing DB-gated TestMain) |
| `git diff --check` (owned dirs) | CLEAN |

## Skip audit / non-claims
- The realtime + e2ewiring tests (the DB-free runnable proofs) **run, not skip** (verbose confirmed: RUN→PASS).
- `cmd/server` package tests are gated by the pre-existing `TestMain` (integration_test.go) that
  `os.Exit(0)` when `DATABASE_URL` is unreachable — I did NOT add any `t.Skip`/env gate; the boot +
  listener tests contain real ungated assertions and run under a DB-available env. `integration_test.go`
  is outside my ownership and its `TestMain` refactor would risk the unguarded integration tests, so the
  DB-free runnable equivalents were placed in `internal/realtime` + `e2ewiring` instead (both RUN here).
- No deploy, no inference, no live run, no secret; no commit/push.

## Final review/fix round (operator Tier-20 blocker)

Operator fixed `service/task.go` (persist ResultID now the real task-row UUID, not synthetic
`result-<task_id>`, which `t20NoGeneratedIDs` rejects) + updated `task_complete_race_test.go`. My
follow-ups (owned file only):

- **`production_anchor_test.go` corrected to production-derived IDs + reframed STRUCTURAL:**
  - `request_id = e2e.CanonicalRequestID(task)` (= `req-<sha256hex>`, suffix is a hash, NOT the task id → passes the gate);
  - `session_id = e2e.CanonicalSessionID(chat, task)`;
  - `queue_msg_id = result_id = raw task UUID` (matches production queue + the operator's persist fix);
  - `proc_id = strconv.Itoa(os.Getpid())` (real decimal pid, not a `proc-…` fixture);
  - added an inline assertion that no assembled id is of the rejected `<prefix><task_id>` synthetic form
    — proving the t20 generated-ID gate ACCEPTS these production-derived ids.
  - Renamed `TestProductionSeamStructuralAssemblyProductionDerivedIDs`; doc now states it is a STRUCTURAL
    seam+assembly test, **NOT** live acceptance / real OTLP / inference evidence — the route hop uses a
    `gateway.Telemetry` FIXTURE (`actualTelemetry` with a literal omni id) via the legacy provider-span
    emitter; the prior "never synthetic/real" wording was removed as an overclaim.

- **Validation (executed):** `gofmt -l` empty; `go vet ./…/e2ewiring` exit 0;
  `go test -run ProductionSeamStructural ./…/e2ewiring` **PASS**; **full `go test ./internal/service`
  ok** (operator's persist fix + updated race test); `go test ./internal/daemon/observability/e2e`
  (incl. t20 hardened gates) **ok**; full `e2ewiring` + `realtime` **ok**; `go build ./cmd/server`
  + `./cmd/multica` exit 0; `git diff --check` on OWNED dirs (cmd/server, realtime, e2ewiring) **CLEAN**.

- **Foreign finding (not mine):** `git diff --check` flags
  `internal/daemon/runtimeenv/env_test.go:499: new blank line at EOF` — a concurrent-lane file not in any
  Codex56#A lock and not edited here; routed to its owner, not fixed by this lane.
