# T20 boot/constructor RED→target tests — exact missing seams

- agent: Opus48#D · lane: boot-tests · task: T20-BOOT-TESTS · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__T20-BOOT-TESTS__20260724T001718Z.json`
- as-of (UTC): 2026-07-24T00:19Z · **test-only, no product edits, new files only, no other lane's tests touched.**
- go=/home/ec2-user/goroot/go/bin/go · `GOCACHE=/tmp/kgc GOTMPDIR=/tmp/kgt`.

## Files added (test-only, mine)
- `internal/daemon/boot_recorder_wiring_test.go` (package `daemon`).
- `cmd/server/boot_recorder_wiring_test.go` (package `main`).

Genuine RED assertions are gated behind `T20_BOOT_TARGET=1` so the shared build stays GREEN for other
lanes (both tests SKIP by default); running with the flag asserts the target.

## Run results (gofmt clean on both)
| Test | Default (`go test`) | `T20_BOOT_TARGET=1` |
|---|---|---|
| `TestBootDaemonInstallsSharedAdmissionAndCLIRecorder` | SKIP (green) | **PASS** — target MET |
| `TestBootDaemonOwnsClosable0600ExportFileAndOTLPLifecycle` | SKIP (missing seams) | SKIP (missing seams) |
| `TestBootServerInstallsSpanRecordersAndOwns0600File` | SKIP (missing seam) | SKIP (missing seam) |
- `go test ./internal/daemon/ -run BootDaemon -count=1` → `ok` (both SKIP), exit 0.
- `T20_BOOT_TARGET=1 go test ./internal/daemon/ -run TestBootDaemonInstallsSharedAdmissionAndCLIRecorder` → **PASS**, exit 0.
- `go test ./cmd/server/ -run BootServer -count=1` → `ok` (SKIP), exit 0.
- `gofmt -l <both files>` → empty.

## State change since the wiring map (live tree advanced)
- **DAEMON admission+CLI shared recorder: NOW WIRED (target MET).** `newDaemon` now sets
  `cliObs: e2e.NewRecorder(daemonSpanSink)` at **daemon.go:288** (shared sink), alongside the admission
  observer. My flagged RED→target assertion therefore **passes** — it flipped RED→GREEN because a
  concurrent lane landed the CLI-hop recorder. (Supersedes the map's earlier `d.cliObs` no-op.)
- Transient build note: at 00:18 the daemon package briefly failed to build (`daemon.go:3785:19:
  undefined: cliProcID`, foreign in-flight CLI-hop edit); resolved by the owning lane by 00:19
  (`cliProcID := ""` present), after which the tests ran.

## EXACT missing seams still RED→target (verified now, non-test grep)
1. **SERVER — all server-side hops still no-op (confirmed STILL NONE):**
   - `middleware.SetIngressRecorder(...)` — 0 production callers → `ingressRecorder` nil (hop-1 ingress no-op).
   - `taskSvc.Obs = ...` — no assignment (cmd/server/main.go:333 leaves `TaskService.Obs` nil → hop-2/6 no-op).
   - `daemonHub.SetDeliveryRecorder(...)` — 0 production callers → `Hub.delivRecorder` nil (hop-7 no-op).
   - no closable 0600 JSONL export file owned by the server; **no testable server constructor** (main() inline)
     → the server target can only be a documented SKIP until an `installObservabilityRecorders(...)`-style
     seam is extracted.
2. **DAEMON — export-file + OTLP lifecycle seams missing:**
   - no `Daemon` field owning a **closable** `*os.File` for the 0600 JSONL export (opened locally inside
     `daemonAdmissionSink`, daemon.go:~4033, never stored/`Close`d) and **no rotation**.
   - loopback **OTLP** `otlpreceiver.LoopbackServer(port)` (otlpreceiver.go:160) is **never
     constructed/started/Shutdown** → hop-5 receiver lifecycle absent.

## Coordination / build-status (2026-07-24T00:2xZ)
- The coordinator removed the deleted `Config.RotationDatabaseURL` reference from this test file (it had
  blocked the daemon package build after a concurrent Config API change); I did NOT re-add it. Re-run with
  true exit codes: `gofmt -l`=clean, `go vet ./internal/daemon/`=0, `go test -run BootDaemon`=0 (both SKIP),
  `T20_BOOT_TARGET=1 … TestBootDaemonInstallsSharedAdmissionAndCLIRecorder`=0 (**PASS**, target met),
  `go test ./cmd/server/ -run BootServer`=0 (SKIP).
- Authoritative build+vet+focused rerun (5 sets) surfaced ONE unrelated exact failure ROUTED to F6 owner:
  `internal/service` `TestQueueHopEmitsExactlyOneLifecycleSpanPerTask` (`anchor_queue_cardinality_test.go:44`)
  — my F6 queue wiring double-emits a HopQueue span (enqueue+dequeue) → `e2e.Assemble` flags
  `duplicate_task_hop`; fix = one completed queue span per task at dequeue. Handled under the F6-fix lane.

## Non-claims / scope
- Test-only; no product source edited; no other lane's test files touched; no deploy/inference/secret.
- The daemon closable-0600-file and OTLP assertions are SKIP placeholders (their fields/lifecycle do not
  exist yet); the server assertion is a SKIP placeholder (no testable constructor). These compile and turn
  into real assertions when the seams land (remove the skip / flip under the flag).
- `-race` not run (gcc/cgo unavailable). Live-dirty tree — re-run on merged HEAD.
