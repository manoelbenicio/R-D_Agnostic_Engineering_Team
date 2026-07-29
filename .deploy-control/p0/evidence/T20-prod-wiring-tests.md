# T20 — Production-wiring OBS integration tests (real seams)

- agent `Opus48#B` · lane `T20-WAVE` · pane `w6:p2` · task `T20-PROD-WIRING-TESTS`/`…-IMPL`
- check-in: `.deploy-control/p0/checkins/Opus48-B__T20-PROD-WIRING-IMPL__20260723T225734Z.json`
- posture: added tests + one test-package anchor; no production logic edited.

## Approach
`SyntheticTraceSpans` is retained as the CONTRACT regression (see `T20-obs-tests.md`). This task adds
**production integration tests over the REAL per-seam emitters** (not the synthetic fixture), each
producing a real, spec-valid hop span with real IDs through the real `e2e.Recorder`:

| Hop | Real emitter (package) |
|---|---|
| ingress (1) | `middleware.EmitIngress(rec, IngressObservation{...})` |
| queue (2) | `service.EmitQueue(rec, QueueObservation{...})` |
| admission (3) | `brain.NewAdmissionObserver(sink).Emit(AdmissionObservation{...})` |
| cli (4) | `daemon.EmitCLI(rec, CLIObservation{...})` |
| route (5) | `gateway.EmitProviderSpan(rec, ProviderSpanRecord{Telemetry:…})` |
| persist (6) | `service.EmitPersist(rec, PersistObservation{...})` |
| delivery (7) | `daemonws.NewDeliveryRecorder(rec).EmitDelivery(DeliveryResult{...})` |

Home: new **leaf** package `internal/daemon/observability/e2ewiring/` (imported by nothing → zero import
cycles) with `doc.go` (package anchor, no logic) + `wiring_integration_test.go` (`package e2ewiring_test`
importing all six seam packages + e2e).

Cross-hop join realism: the admission emitter derives its own opaque `launch_id`, so the test emits
admission first, **reads the recorded admission span's `launch_id` back from the sink**, and uses it for
the CLI hop — a real join, not a guessed value.

## Tests (both PASS)
1. `TestProductionSeamsEmitContinuousLifecycle` — drives all 7 real emitters into one shared
   `MemorySink`; asserts: 7 real hop spans (exactly one each; one persist + one delivery), all 9 IDs
   present, `Assemble().AllContinuous==true`, 1 trace, 0 orphans, 0 anomalies, 0 gaps (`Present==7`),
   `ScanFromSink` clean (`Scanned==7`, 0 findings). Each `Emit*` returning nil proves the real span
   passed the recorder's fail-closed validation + single-span leak scan.
2. `TestRouteHopEmitsFromActualTelemetryFixture` — route hop tested SEPARATELY with an actual
   `gateway.Telemetry` fixture (real model/route/pseudonyms/usage/selection/quota/circuit); asserts a
   single real `HopRoute` span carrying `request_id`+`omni_request_id`, leak-clean.

## Results (exit codes; `/tmp` caches)
| Command | Result | exit |
|---|---|---|
| `gofmt -l …/e2ewiring/*.go` | clean | 0 |
| `go vet ./internal/daemon/observability/e2ewiring/` | clean | 0 |
| `go test ./internal/daemon/observability/e2ewiring/ -count=1` | 2/2 PASS (ok 0.005s) | 0 |
| `git diff --check …/e2ewiring/` | clean | 0 |

**Verdict: PASS.** The real ingress→queue→admission→CLI→route→persist→delivery emitters produce real,
spec-valid hop spans with real IDs that assemble into one continuous, leak-clean trace; route hop proven
separately from actual telemetry.

## Non-claims / notes
- Files created: one test (`wiring_integration_test.go`) + one minimal package anchor (`doc.go`, no
  logic). No production seam source edited. New leaf package → zero import cycles; verified free/no lock.
- These exercise the real emitter FUNCTIONS end-to-end (the production instrumentation API) with real IDs
  — NOT a live daemon/HTTP/DB/WS run (no server/socket/DB; values are synthetic identifiers,
  `secrets_present==false`). Call-site wiring inside the shared anchors (hub.go/task.go/router chain) is
  L1's Wave-C integration and is out of a test's scope.
- No inference/secret/deploy/commit/push; no OpenSpec checkbox closed.
