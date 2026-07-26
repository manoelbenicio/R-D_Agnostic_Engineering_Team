# T20 — OBS end-to-end lifecycle integration/regression test

- agent `Opus48#B` · lane `T20-WAVE` · pane `w6:p2` · task `T20-OBS-TESTS`
- check-in: `.deploy-control/p0/checkins/Opus48-B__T20-OBS-TESTS__20260723T224600Z.json`
- posture: added deterministic tests; product source READ-ONLY (one new test file created).

## Target (real emit path, `internal/daemon/observability/e2e`)
`SyntheticTraceSpans(taskID)` (synthetic.go) is the canonical one-lifecycle fixture (7 emitting hops,
all 9 IDs cross-joined). The production emit path is `NewRecorder(sink).Emit` (recorder.go, fail-closed
validate + single-span leak scan) → `MemorySink`; the acceptance path is `Assemble` (assemble.go) +
`ScanSpans`/`ScanFromSink` (leakscan.go). Contract: 7 `EmittingHops` (ingress/queue/admission/cli/route/
persist/delivery) + 9 IDs (request/queue_msg/task/session/launch/proc/omni_request/result/delivery).

## Added — `internal/daemon/observability/e2e/lifecycle_integration_test.go` (NEW; deterministic)

1. `TestOneLifecycleEmitsAllHopsIDsContinuousAndLeakClean` — emits one lifecycle through the real
   `Recorder`→`MemorySink` (`EmitSyntheticTask`, fail-closed), then asserts **in one test**:
   - **7 source hops**, exactly once each (`len(spans)==7`, each `EmittingHops()` hop count==1);
   - **exactly one persist + one delivery** (`byHop[HopPersist]==1`, `byHop[HopDelivery]==1`);
   - **all 9 IDs** present across the lifecycle (distinct count==9);
   - **assembler**: `Assemble(spans).AllContinuous==true`, exactly 1 `Trace`, `Orphans==0`,
     `Anomalies==0`, trace `Continuous==true`, `Missing==0` (**zero gaps**), `Present==7`, correct `TaskID`;
   - **leak scan clean**: `ScanFromSink(sink).Clean==true`, `Scanned==7`, `Findings==0`.
2. `TestLifecycleRegressionMissingHopBreaksContinuity` — dropping the persist hop → `AllContinuous==false`,
   trace discontinuous, `Missing` includes `persist` (proves the gap/continuity assertion has teeth).
3. `TestLifecycleRegressionSecretLeakNotClean` — a `secrets_present=true` span → `ScanSpans.Clean==false`
   with ≥1 finding (proves the leak-clean assertion has teeth).

## Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/lifecycle_integration_test.go` | clean | 0 |
| `go vet ./internal/daemon/observability/e2e/` | clean | 0 |
| `go test ./internal/daemon/observability/e2e/ -run 'OneLifecycle|LifecycleRegression' -count=1` | 3/3 PASS (ok 0.002s) | 0 |
| `git diff --check …` | clean | 0 |

**Verdict: PASS.** One lifecycle emits all 7 source hops + all 9 IDs; assembler `AllContinuous=true` with
zero gaps/orphans/anomalies; leak scan clean; exactly one persist and one delivery. Negative guards
confirm the assertions detect a missing hop and a secrets_present leak.

## Non-claims / notes
- One product file created (test-only); no e2e source edited. e2e package was unlocked and the new test
  file had zero overlap with any active record (verified). Complements existing per-piece tests
  (`TestAssembleContinuousSyntheticTasks`, `TestScanCleanSyntheticTrace`, …) with a single consolidated proof.
- "One real lifecycle" = the canonical `SyntheticTraceSpans` emitted through the real Recorder/Sink/
  Assembler/Scanner (the production instrumentation API each lane uses). NOT a live daemon/HTTP run — no
  server/DB/network is involved; values are synthetic; `secrets_present==false` by construction.
- No inference/secret/deploy/commit/push; no OpenSpec checkbox closed.
