# T20 — Live Anchor Integration Test Evidence (`live_anchor_integration_test.go`)

- agent: **Antigravity** · lane: **live-anchor-test** · task: **T20-LIVE-ANCHOR-TEST**
- as-of (UTC): `2026-07-24T00:08Z`
- lock (sole mutable): `.deploy-control/p0/evidence/T20-live-anchor-test.md`
- test file (created, test-only): [live_anchor_integration_test.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2ewiring/live_anchor_integration_test.go)
- posture: **TEST FILES ONLY; ZERO product source modified.**

---

## 1. Executive Summary & Test Design

To prove trace assembly against real production seam emitters (rather than hand-constructed or pre-aligned synthetic spans), we built a failing-then-target integration test in `internal/daemon/observability/e2ewiring/live_anchor_integration_test.go`.

The test suite consists of two complementary integration tests:
1. `TestLiveAnchorProductionDiscrepanciesExposeFailingJoins`: Drives the actual production call sites (`middleware.EmitIngress`, `service.EmitQueue`, `brain.NewAdmissionObserver`, `daemon.EmitCLI`, `gateway.EmitProviderSpan`, `service.EmitPersist`, `daemonws.NewDeliveryRecorder`) with current unaligned ID derivations. It **exposes and proves the 3 exact failing joins**.
2. `TestTargetCanonicalIDPropagationProducesContinuousTrace`: Drives the same production call sites with canonical ID propagation enabled across all 7 seams, proving that `e2e.Assemble` succeeds with `AllContinuous=true`, 0 orphans, 0 anomalies, and 1 complete 8-hop / 9-ID trace.

---

## 2. Exposed Production Discrepancies & Failing Join Findings

When production call sites are driven with default unaligned ID derivations, `e2e.Assemble(spans)` yields `AllContinuous = false` and 2 orphaned spans.

| Discrepancy # | Production Seam Discrepancy | Seam Call Sites | Discrepancy Details | Assembly Failure Impact |
|---|---|---|---|---|
| **1** | **Raw UUID vs Task-Hash Mismatch** | `middleware.EmitIngress` / `service.EmitQueue` / `service.EmitPersist` vs `brain.AdmissionObserver` / `daemon.EmitCLI` | Ingress, Queue, and Persist use raw HTTP/Task UUID (e.g. `11111111-2222-3333-4444-555555555555`), while Admission and CLI use a derived task-hash (e.g. `thash-a1b2c3d4e5f6`). | Splits the single task lifecycle into **2 fragmented, incomplete traces**. Neither trace contains all 7 hops (`Continuous = false`). |
| **2** | **Independent Request IDs Mismatch** | `middleware.EmitIngress` vs `gateway.EmitProviderSpan` | Ingress stamps HTTP `request_id` (`req-http-ingress-9999`), while Gateway Route independently generates its own `request_id` (`req-gateway-route-7777`). | `HopRoute` span cannot resolve to ingress `request_id`. Becomes an **Orphan Span** (`Reason: "unresolved request_id"`). |
| **3** | **Random WS Session Mismatch** | `brain.AdmissionObserver` vs `daemonws.NewDeliveryRecorder` | Admission stamps session ID (`sess-chat-user-123`), while `daemonws.Hub` mints a random per-connection WS session ID (`wsdel-session-random-99`). | `HopDelivery` span cannot resolve to admission `session_id`. Becomes an **Orphan Span** (`Reason: "unresolved session_id"`). |

---

## 3. Target Alignment Contract & Verification Results

When canonical ID propagation is applied across all 7 production seam call sites:

```text
ingress   (req-canonical-http-200, task-canonical-uuid-100)
  │
queue     (qmsg-canonical-400, task-canonical-uuid-100)
  │
admission (task-canonical-uuid-100, sess-canonical-user-300, launchID)
  ├── cli   (launchID, proc-canonical-500)
  └── delivery (sess-canonical-user-300, delivery-canonical-800)
route     (req-canonical-http-200, omni-canonical-600)
  │
persist   (task-canonical-uuid-100, result-canonical-700)
```

### Verified Target Results
- `report.Assembly.AllContinuous`: **TRUE**
- `len(report.Assembly.Orphans)`: **0**
- `len(report.Assembly.Anomalies)`: **0**
- `len(report.Assembly.Traces)`: **1 Complete Trace**
- Hops: All 7 source emitting hops present (`ingress`, `queue`, `admission`, `cli`, `route`, `persist`, `delivery`) + 1 assembled `HopTrace` = **8 Hops**.
- Correlation IDs: Full **9 correlation IDs** present.
- Leak Scan: `ScanSpans(spans).Clean` = **TRUE**.

---

## 4. Test Verification Command & Output

```bash
cd multica-auth-work/server
GOCACHE=/tmp/gocache GOTMPDIR=/tmp/gotmp GOMODCACHE=/tmp/gomodcache \
/home/ec2-user/goroot/go/bin/go test -v ./internal/daemon/observability/e2ewiring/ -run 'TestLiveAnchor' -count=1
```

### Test Results
```text
=== RUN   TestLiveAnchorProductionDiscrepanciesExposeFailingJoins
--- PASS: TestLiveAnchorProductionDiscrepanciesExposeFailingJoins (0.00s)
=== RUN   TestTargetCanonicalIDPropagationProducesContinuousTrace
--- PASS: TestTargetCanonicalIDPropagationProducesContinuousTrace (0.00s)
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon/observability/e2ewiring	0.008s
```

---

## 5. Non-Claims & Scope

- **Test Files Only**: Only [live_anchor_integration_test.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2ewiring/live_anchor_integration_test.go) created under `internal/daemon/observability/e2ewiring/`. Zero product `.go` files modified.
- **Identified Production Remediation**: Closing the 3 exposed discrepancies in production requires updating shared startup/handler wiring to propagate canonical `task_id`, `X-AB-Request-Id`, and `session_id` across the HTTP middleware, gateway router, and daemonws hub callers.
