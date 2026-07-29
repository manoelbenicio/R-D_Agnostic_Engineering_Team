# T20 production startup-wiring map (READ-ONLY) — e2e recorder / spans / OTLP

- agent: Opus48#D · lane: wiring-map · task: T20-WIRING-MAP · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__T20-WIRING-MAP__20260724T000643Z.json`
- as-of (UTC): 2026-07-24T00:07Z · HEAD `a6d5098` · **READ-ONLY: no source edits, no deploy.** All paths `multica-auth-work/server/`.

## 0. Two processes (constructors)
- **SERVER** — `cmd/server/main.go`: `realtime.NewHub()`:167, `daemonws.NewHub()`:169,
  `service.NewTaskService(...)`:333; graceful shutdown `signal.Notify`:406, `srv.Shutdown`:417,
  `metricsServer.Shutdown`:447. (alt wiring: `internal/handler/handler.go:196`.)
- **DAEMON** — `cmd/multica/cmd_daemon.go`: `daemon.NewWithAgentBrainDependencies(...)`:439 /
  `daemon.New(...)`:444 → `newDaemon` (`internal/daemon/daemon.go`), which builds `agentBrainOBS`:283.

## 1. "One 0600 JSONL recorder per process" — current state
- Durable sink primitive: `e2e.NewJSONLSink(io.Writer)` (`observability/e2e/export_sink.go`; fail-closed;
  caller owns the 0600 file open; **no built-in rotation**), fan-out `e2e.NewMultiSink`, wrap in `e2e.NewRecorder`.
- **DAEMON**: the only real 0600 recorder path today — `daemonAdmissionSink(logger)` (`daemon.go:4023`)
  opens the export file `os.OpenFile(path, O_CREATE|O_APPEND|O_WRONLY, 0o600)` at **daemon.go:4033**,
  gated on env **`AGENT_BRAIN_E2E_EXPORT_FILE`** (empty ⇒ log-only; open-fail ⇒ `NewJSONLSink(nil)` fail-closed
  per record, daemon.go:4035). Wrapped once into `agentBrainOBS` at **daemon.go:283**. It is built per-call
  and threaded ONLY to admission — **not** to the CLI hop.
- **SERVER**: **no e2e recorder is constructed at all** (grep of `cmd/server/main.go` + `router.go` for
  `e2e.`/`SetIngressRecorder`/`SetDeliveryRecorder`/`.Obs`/`otlpreceiver` = NONE). ⇒ all server-side hops no-op.

## 2. Per-hop recorder source → wired vs NO-OP (file:line) + insertion point
| Hop | Emit call site (wired) | Recorder source | State | Insertion point |
|---|---|---|---|---|
| 1 ingress (server) | `middleware/request_logger.go:309` `EmitIngress(rec,...)` (via `emitIngressSpan`:297) | pkg var `ingressRecorder` decl `request_logger.go:217`; setter `SetIngressRecorder`:222 | **NO-OP** — setter has **no production caller**; `ingressRecorder==nil` (self-doc no-op :142/:215) | `cmd/server/main.go` startup: `middleware.SetIngressRecorder(rec)` |
| 2 queue (server) | `service/task.go:191,219` `EmitQueue(s.Obs,...)` | field `TaskService.Obs` (F6) | **NO-OP** — **no `.Obs =` assignment anywhere**; nil-guards task.go:187/203 | after `NewTaskService`:333 → `taskSvc.Obs = rec` |
| 6 persist (server) | `service/task.go:236` `EmitPersist(s.Obs,...)` | field `TaskService.Obs` | **NO-OP** (same; guard :228) | same as hop 2 (shared `taskSvc.Obs`) |
| 7 delivery (server) | `daemonws/hub.go:264` `rec.EmitDelivery(...)` (via `emitDelivery`:260) | field `Hub.delivRecorder` decl `hub.go:171`; setter `SetDeliveryRecorder`:235 | **NO-OP** — setter **no production caller**; `delivRecorder==nil` (guard `deliveryRecorder()`:244) | after `daemonws.NewHub()`:169 → `daemonHub.SetDeliveryRecorder(daemonws.NewDeliveryRecorder(rec))` |
| 3 admission (daemon) | `brain/admission_observability.go` (`AdmissionObserver.recorder = e2e.NewRecorder(sink)`:34) | `daemonAdmissionSink(logger)` → `agentBrainOBS` `daemon.go:283` | **WIRED** — log always; **0600 JSONL export when `AGENT_BRAIN_E2E_EXPORT_FILE` set** (file open daemon.go:4033) | (already wired; hoist to shared recorder — see §4) |
| 4 CLI (daemon) | `daemon.go:3778` `EmitCLI(d.cliObs,...)` (boundary :3764) via `cli_observability.go` | field `Daemon.cliObs` decl `daemon.go:159` | **NO-OP** — `d.cliObs` is **nil until injected** (self-doc :3767) | in `newDaemon`, inject the shared daemon recorder into `d.cliObs` |
| 5 route/OTLP (daemon) | OTLP receiver `otlpreceiver.LoopbackServer(port)` returns `*http.Server` (`otlpreceiver.go:148`) | `otlpreceiver.Receiver` + a `Sink` (SanitizedRecord) | **NO-OP** — **no production caller** of `NewReceiver`/`LoopbackServer`/`ListenAndServe`; receiver never started | start `LoopbackServer(127.0.0.1:<fixed daemon-injected port>)` in the daemon; bridge `SanitizedRecord`→route span sink |

## 3. NO-OP proof (grep, non-test)
- `SetIngressRecorder` — declared `request_logger.go:222`; **zero callers** ⇒ `ingressRecorder` nil.
- `SetDeliveryRecorder` — declared `hub.go:235`; **zero callers** ⇒ `Hub.delivRecorder` nil.
- `TaskService.Obs` — **no `.Obs =` assignment** in `cmd/`/`internal/` (only nil-guards task.go:187/203/228).
- `d.cliObs` — declared `daemon.go:159`; **no assignment** (self-doc "nil until the central observability wiring injects a recorder" :3767).
- OTLP — `LoopbackServer`/`NewReceiver` have **zero production callers** ⇒ listener never started.
- `cmd/server/main.go`/`router.go` — **no** `e2e.`/recorder/OTLP wiring at all.

## 4. Close / shutdown / error / rotation lifecycle
- **DAEMON 0600 file (daemon.go:4033):** opened `O_APPEND|0600` but the returned `*os.File` is **never
  Closed** and there is **no shutdown hook** → handle leak on daemon stop. **Gap.** Insertion: build ONE
  `*os.File`+`JSONLSink`+`e2e.NewRecorder` in `newDaemon`, store on `Daemon`, `Close()` it in the daemon
  shutdown path; inject the same recorder into `agentBrainOBS` (admission) AND `d.cliObs` (CLI) so the
  process has exactly one recorder/file.
- **SERVER:** build ONE 0600 recorder in `cmd/server/main.go` startup; `Close()` it in the shutdown block
  (alongside `srv.Shutdown`:417 / `metricsServer.Shutdown`:447 after `signal.Notify`:406).
- **OTLP:** `LoopbackServer` returns `*http.Server` → must `ListenAndServe` in a goroutine and
  `Shutdown(ctx)` on process stop; currently neither. **Gap.**
- **Error:** `JSONLSink` is fail-closed per record (nil/unwritable writer → error every `Emit`; short-write
  detected — export_sink.go:37-56). Emit-site helpers deliberately drop that error (best-effort, never
  crash the request/task path). Acceptable; but a failed file open silently degrades to log-only + a
  per-record error (daemon.go:4035) — operators should alert on it.
- **Rotation:** **none** — `JSONLSink` is append-only over the caller's writer; the 0600 file grows
  unbounded. **Gap.** Options: size/time rotation wrapper around the writer, or external logrotate
  coordinated with the `e2e.Collector` (`collector.go`, which reassembles from the dedicated JSONL files).

## 5. Recommended single-recorder wiring (per process) — for the implementer (NOT applied here)
- **SERVER (`cmd/server/main.go`):** open 0600 JSONL file (env-gated) → `rec := e2e.NewRecorder(e2e.NewMultiSink(bounded, e2e.NewJSONLSink(f)))`; then `middleware.SetIngressRecorder(rec)`, `taskSvc.Obs = rec` (after :333), `daemonHub.SetDeliveryRecorder(daemonws.NewDeliveryRecorder(rec))` (after :169), start `otlpreceiver.LoopbackServer` with a route-span bridge sink; `defer f.Close()` + OTLP `Shutdown` in the :406/:417 block.
- **DAEMON (`newDaemon`):** hoist the daemon.go:4033 open into one process recorder; inject into `agentBrainOBS` (replacing the per-call `daemonAdmissionSink` at :283) AND `d.cliObs`; add `Close()` on daemon shutdown; start the loopback OTLP receiver (the daemon injects its fixed 127.0.0.1 endpoint into the child CLI env).

## 6. Non-claims / scope
- READ-ONLY map: no source edited, no recorder wired, no OTLP started, no deploy, no inference, no secret read.
- File:line are as-of HEAD `a6d5098` on a live-dirty tree; re-confirm on merged HEAD before implementing.
- Admission hop is the only currently-active e2e export path (and only when `AGENT_BRAIN_E2E_EXPORT_FILE` is set); all other hops are wired-but-no-op pending recorder injection per §2.
- OmniRoute internals never probed; OTLP is loopback-only/content-off by design (`otlpreceiver`).
