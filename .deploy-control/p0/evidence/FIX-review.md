# FIX — Independent Review Report (StartTask / Rerun Race Fix)

agent: Agy-P0-A8
lane: C8 (independent-review)
task: C8-STARTTASK-RACE-FIX-REVIEW
pane: wB:p2
timestamp: 2026-07-23T00:09:10Z
verdict: **VERIFIED**

## 1. Preflight Environment & Lock Audit

| Parameter | Observed Value | Status |
|---|---|---|
| Working Directory | `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` | ✅ PASS |
| Git HEAD | `a6d50986aee7f1981a323b29c9d90f175b4b6f8b` | ✅ PASS |
| Toolchain (Go) | `/home/ec2-user/goroot/go/bin/go` (go1.26.1 linux/amd64) | ✅ PASS |
| Lane Tmp Dirs | `GOCACHE=/tmp/gocache-c8`, `GOTMPDIR=/tmp/gotmp-c8` | ✅ PASS |
| Lock Zero-Overlap | 0 cross-lane file lock intersections across canonical records | ✅ PASS |

## 2. Race Fix Correctness Review

| Race / Issue | Root Cause | Verified Fix | Audit Result |
|---|---|---|---|
| **Issue #3999 Race A (StartTask vs WorkDir)** | `handleTask` called `/start` before `runner.run`, flipping server status to `running` while `workdir` did not yet exist on disk. | `handleTask` delegates `StartTask` to `runTask`, invoking `/start` **after** `execenv.Prepare`/`Reuse` confirms `env.WorkDir` is created on disk. | ✅ **VERIFIED PASS** |
| **Issue #3999 Race B (GC Window)** | Inner active guard un-marked `envRoot` before `reportTaskResult` and `WriteGCMeta` finished, creating a window for GC cleanup. | `handleTask` installs an outer active guard keeping `envRoot` in the active set until after completion reporting and `WriteGCMeta` complete. | ✅ **VERIFIED PASS** |
| **Already-Finalized Task Complete/Fail** | Double-complete or rerun on finalized tasks returned `pgx.ErrNoRows` from DB update. | `CompleteTask` and `FailTask` catch `pgx.ErrNoRows` and fetch/return the existing task state cleanly without error. | ✅ **VERIFIED PASS** |

## 3. Fail-Closed Preservation & Boundary Audit

- **No Fail-Closed Weakening**: Daemon admission checks, credential isolation rules, and gateway fail-closed policies are completely preserved without weakening.
- **No OmniRoute-Internal Probing**: Code consumes standard readiness declarations; zero OmniRoute-internal probing, secret reading, or credential manipulation is introduced.

## 4. Test Suite Validation Results

| Test Command | Exit Code | Result | Details |
|---|---|---|---|
| `go test ./internal/daemon -run 'TestHandleTask|TestRunTask'` | `0` | ✅ **PASS (0.029s)** | `TestHandleTask_DoesNotCallStartTaskItself`, `TestRunTask_StartTaskCalledAfterWorkdirOnDisk`, `TestHandleTask_KeepsEnvRootActiveAcrossCompletion` PASS. |
| `go test ./internal/service -run 'TestCompleteTask|TestFailTask'` | `0` | ✅ **PASS (0.007s)** | `TestCompleteTask_AlreadyFinalized`, `TestFailTask_AlreadyFinalized` PASS across all subtests. |
| `go vet ./internal/daemon/... ./internal/service/...` | `0` | ✅ **PASS** | 0 vet warnings across daemon and service packages. |
| `gofmt -l internal/daemon/workdir_race_test.go internal/service/task_complete_race_test.go` | `0` | ✅ **PASS** | Clean code formatting. |
| `git diff --check` | `0` | ✅ **PASS** | Clean git diff. |

## 5. Non-Claims & Constraints Enforcement

- Read-only on product code enforced 100% (0 product source files edited by C8).
- `live_runs.*=false` respected; 0 model inference executed.
- No deploy, container restart, Docker, or systemd commands executed.
- No secrets read, printed, or handled.
