# Evidence: Single Clean Kanban Live-Run Observation Plan (`RES-live.md`)

## 1. Executive Summary & Verification Verdict

- **Status**: **STANDBY — HARNESS & WATCHER READY (AWAITING POST-DEPLOY SINGLE LIVE RUN)**
- **Lane**: `live-watcher` (`wB:p1`)
- **Agent**: `Agy-P0-A7`
- **Task ID**: `RES-LIVE-WATCH`
- **Timestamp**: `2026-07-23T02:13:55Z`
- **Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- **Scope**: Observation contract and pre-flight harness for watching the single clean Kanban live run post-deploy.
- **Constraints Enforced**:
  - `No hammering`: Observe exactly ONE clean live run.
  - `No inference during standby`: `live_runs=false` strictly maintained (`LiveEndpointUsed: false`).
  - `No OmniRoute-internal inspection`: Strictly consume-only readiness & external telemetry spans.
  - Product code: **READ-ONLY**.

---

## 2. Single Kanban Live-Run Pipeline Architecture

```
[Phase 1: Enqueue] ──► [Phase 2: Claim] ──► [Phase 3: Admission] ──► [Phase 4: Launch] ──► [Phase 5: OmniRoute] ──► [Phase 6: Persisted Result]
 Client enqueues        Worker claims        GatewayAdmission        agentBrainLaunch       Executor dispatches     Terminal status & result
 task / session        task_claim           `Admitted` & `Ready`   child env              model call              saved to DB / ResultSink
```

### Phase-by-Phase Single-Run Invariants
1. **Phase 1 (Enqueue)**: Client creates chat session (`POST /api/chat-sessions`). Task enters `TaskStatusPending`.
2. **Phase 2 (Claim)**: Daemon worker claims task (`TaskStatusClaimed`), assigning worker lease and correlation IDs.
3. **Phase 3 (Admission)**: `GatewayAdmissionController` evaluates readiness snapshot (`GatewayReadinessReady`).
4. **Phase 4 (Launch)**: `agentBrainRuntime` provisions controlled execution root and launches target CLI runner (`CLICodex` or `Kiro-TL`).
5. **Phase 5 (OmniRoute)**: Model request routed via OmniRoute (`http://100.118.244.61:20128`), emitting sanitized `ProviderSpanRecord` (`EmitProviderSpan`).
6. **Phase 6 (Persisted Result)**: Task status updated to `TaskStatusCompleted` (or `TaskStatusFailed`), emitting `EmitPersist` with closed metadata only.

---

## 3. Post-Deploy Observation Protocol & Activation Plan

- **Standby State**: `live_runs=false` enforced. Zero live LLM API calls executed before post-deploy authorization.
- **Single-Rerun Execution Rule**: Once post-deploy signal is received and Principal grants live-run token authorization (Gate D-V3-25B):
  1. Observe the single live task run in real time.
  2. Capture `task_id`, `session_id`, `request_id`, phase timestamps, and HTTP status codes.
  3. Validate zero secret, prompt, or response body leakage in emitted spans and persistence records.
  4. Record final `PASS` verdict in `RES-live.md`.

---

## 4. Verification Commands & Exit Codes

- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go test ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `git diff --check .deploy-control/p0/evidence/RES-live.md`: **PASS** (exit code 0)

---

## 5. Enforced Non-Claims

- `AcceptanceClaim`: `false` (standby mode; single live run pending post-deploy signal)
- `LiveEndpointUsed`: `false`
- Zero polling loops, hammering, or unauthorized live API calls executed.
- Product source code in `server/...` remained strictly **READ-ONLY**.
