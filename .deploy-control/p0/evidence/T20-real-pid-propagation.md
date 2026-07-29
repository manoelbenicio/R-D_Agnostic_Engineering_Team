# T20 — Real PID Propagation Audit & Test Evidence (`pid_propagation_test.go`)

- agent: **Antigravity** · lane: **pid-propagation-audit** · task: **T20-REAL-PID-PROPAGATION**
- as-of (UTC): `2026-07-24T00:16Z`
- lock (sole mutable): `.deploy-control/p0/evidence/T20-real-pid-propagation.md`
- test file (created, test-only): [pid_propagation_test.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2e/pid_propagation_test.go)
- posture: **TEST FILES ONLY; ZERO product source modified.**

---

## 1. Executive Summary & Audit Matrix

We performed a comprehensive audit of `agent.Session`, process backends (`claude.go`, `antigravity.go`, `codex.go`), and `Daemon.executeAndDrain` for the minimal backward-compatible `ProcessID` seam.

| Component / Subsystem | Audit Subject | Current State / API Availability | Minimal Backward-Compatible Seam Recommendation |
|---|---|---|---|
| **`agent.Session`** | `ProcessID` struct field | **ABSENT** (`Session` has `Messages`, `Result`) | Add `ProcessID int` (or `ProcID string`) to `agent.Session`. Zero-value (`0`) preserves non-process/mock backends without breaking changes. |
| **Process Backends** (`claudeBackend`) | `cmd.Process.Pid` capture | **Internal only** (`cmd.Process.Pid` exists on spawned `exec.Cmd`) | Assign `cmd.Process.Pid` to `Session.ProcessID` upon `cmd.Start()` before returning `Session`. |
| **`Daemon.executeAndDrain`** | `executeAndDrain` return tuple | Returns `(agent.Result, int32, error)` (tool count `int32`) | Extract `sess.ProcessID` during execution to populate `CLIObservation.ProcID = strconv.Itoa(pid)` when emitting `HopCLI`. |
| **`HopCLI` Span Validation** | Decimal PID (`ProcID`) + `AdmissionLaunchID` | **Enforced Fail-Closed** | `HopCLI` requires `LaunchID` + `ProcID`. Real decimal PID (e.g. `"12345"`) passes; zero (`"0"`), empty (`""`), or invalid PID is rejected fail-closed. |

---

## 2. Test Verification & Code Proofs

All tests were implemented in [pid_propagation_test.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2e/pid_propagation_test.go) (`package e2e`):

### A. Audit of `agent.Session` & `agent.Result` API (`TestAuditProcessIDSeam_SessionAndBackends`)
- Audits `reflect.TypeOf(agent.Session{})` and `reflect.TypeOf(agent.Result{})`.
- Confirms current field structure and logs exact API seam availability findings.

### B. Valid Decimal PID & Exact `AdmissionLaunchID` (`TestHopCLISpan_ValidDecimalPIDAndLaunchIDEmitsValidSpan`)
- Obtains real OS PID (`strconv.Itoa(os.Getpid())`) and exact `AdmissionLaunchID` (`"launch-adm-20260724-100"`).
- Proves `NewSpan(HopCLI, Correlation{LaunchID: launchID, ProcID: pid})`:
  - Passes `Span.Validate()` with 0 errors.
  - Successfully records into `JSONLSink` with 0 drops (`report.Dropped() == 0`).
  - Passes structural leak scan (`ScanSpans(spans).Clean == true`).

### C. Zero or Empty PID Fail-Closed Rejection (`TestHopCLISpan_ZeroOrEmptyPIDFailsClosed`)
Tested 5 distinct PID validation cases:
1. **Empty PID (`""`)**: Rejected by `Span.Validate()` (`"missing required correlation proc_id"`).
2. **Zero PID (`"0"`)**: Rejected fail-closed as uninitialized/invalid process ID.
3. **Negative PID (`"-100"`)**: Rejected by `safeID` (`"not a safe identifier"`).
4. **Non-numeric PID (`"proc_invalid_xyz"`)**: Rejected by PID decimal validator (`"not a positive decimal process ID"`).
5. **Valid Decimal PID (`"12345"`)**: Accepted cleanly.

### D. Joining `HopCLI` with `HopAdmission` (`TestHopCLISpan_JoinsWithAdmissionLaunchID`)
- Emits `HopAdmission` with `LaunchID = "launch-adm-join-777"`.
- Emits `HopCLI` with matching `LaunchID` and real decimal PID `strconv.Itoa(os.Getpid())`.
- `Assemble(spans)` joins `HopCLI` cleanly into the task trace with `len(Orphans) == 0`.

---

## 3. Go Test Command & Verification Output

```bash
cd multica-auth-work/server
GOCACHE=/tmp/gocache GOTMPDIR=/tmp/gotmp GOMODCACHE=/tmp/gomodcache \
/home/ec2-user/goroot/go/bin/go test -v ./internal/daemon/observability/e2e/ -run 'TestAuditProcessIDSeam|TestHopCLISpan' -count=1
```

### Output
```text
=== RUN   TestAuditProcessIDSeam_SessionAndBackends
    pid_propagation_test.go:29: AUDIT FINDING: agent.Session.ProcessID field present: false
    pid_propagation_test.go:36: AUDIT FINDING: agent.Result.SessionID present: true, ProcessID present: false
--- PASS: TestAuditProcessIDSeam_SessionAndBackends (0.00s)
=== RUN   TestHopCLISpan_ValidDecimalPIDAndLaunchIDEmitsValidSpan
--- PASS: TestHopCLISpan_ValidDecimalPIDAndLaunchIDEmitsValidSpan (0.00s)
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed/Empty_PID
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed/Zero_PID_'0'
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed/Negative_PID_'-100'
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed/Non-numeric_PID_'proc_invalid_xyz'
=== RUN   TestHopCLISpan_ZeroOrEmptyPIDFailsClosed/Valid_Decimal_PID_'12345'
--- PASS: TestHopCLISpan_ZeroOrEmptyPIDFailsClosed (0.00s)
=== RUN   TestHopCLISpan_JoinsWithAdmissionLaunchID
--- PASS: TestHopCLISpan_JoinsWithAdmissionLaunchID (0.00s)
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon/observability/e2e	0.003s
```

---

## 4. Status & Conclusion

- **STATUS:** **AUDITED & VERIFIED**
- Audited `agent.Session`, process backends, and `executeAndDrain`.
- Proven that `HopCLI` with real decimal PID and exact `AdmissionLaunchID` validates, records, and joins cleanly, while zero/empty/invalid PIDs fail closed.
- Delivered test suite [pid_propagation_test.go](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/multica-auth-work/server/internal/daemon/observability/e2e/pid_propagation_test.go) and evidence document [.deploy-control/p0/evidence/T20-real-pid-propagation.md](file:///home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/.deploy-control/p0/evidence/T20-real-pid-propagation.md). Zero product source edited.
