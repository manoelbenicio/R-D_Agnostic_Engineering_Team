# Evidence: Single Bounded Live Rerun Observation & Watcher Contract (`LIVE-watch.md`)

## 1. Executive Summary & Verification Verdict

- **Status**: **STANDBY — WATCHER HARNESS READY (AWAITING FIX DEPLOY & LIVE TOKEN)**
- **Lane**: `live-watcher` (`wB:p1`)
- **Agent**: `Agy-P0-A7`
- **Task ID**: `LIVE-WATCH`
- **Timestamp**: `2026-07-23T00:08:46Z`
- **Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- **Scope**: Observation contract for watching exactly **ONE** bounded live rerun (no hammering/polling loops) across the full 5-phase lifecycle: `claim -> admitted -> launch -> OmniRoute -> persisted terminal/log/status`.
- **Safety Constraints Enforced**:
  - `No hammering`: Exactly ONE bounded rerun observation permitted per authorization.
  - `No inference until daemon green`: `live_runs=false` enforced during standby.
  - `No OmniRoute-internal inspection`: Strictly consume-only readiness & external telemetry spans.
  - Product code: **READ-ONLY**.

---

## 2. Five-Phase Single-Rerun Lifecycle Trajectory

```
[Phase 1: Claim] ──► [Phase 2: Admitted] ──► [Phase 3: Launch] ──► [Phase 4: OmniRoute] ──► [Phase 5: Persisted Terminal]
 Worker claims        Gateway admission       CLI runner launch      Provider call via      Terminal status & result
 task_claim           check returns Ready     agentBrainLaunch       Executor.Execute       saved to DB / ResultSink
```

### Phase-by-Phase Observation Protocol
1. **Phase 1: Claim (`task_claim`)**: Task claimed by worker process. Record `task_id`, `session_id`, and `claimed_at` timestamp.
2. **Phase 2: Admitted (`Admit`)**: Main Brain `GatewayAdmissionController` checks gateway readiness. Confirm `AdmissionAdmitted` and `GatewayReadinessReady`.
3. **Phase 3: Launch (`agentBrainLaunch`)**: `agentBrainRuntime.buildLaunch` prepares controlled child environment (`TaskHome`, `CodexHome`/`ClineDataDir`).
4. **Phase 4: OmniRoute (`Executor.Execute`)**: Model request routed through OmniRoute (`http://100.118.244.61:20128`). Confirm `EmitProviderSpan` emitted with sanitized `Telemetry`.
5. **Phase 5: Persisted Terminal (`EmitPersist`)**: Task reaches terminal state (`TaskStatusCompleted` / `TaskStatusFailed`). Confirm terminal log and status persisted without credential or prompt exposure.

---

## 3. Single-Rerun Failure Classification Matrix

| Phase | Event | Failure Class | Observation Point |
|---|---|---|---|
| **Phase 1** | Task claim fails | `task_claim_failed` | DB worker lock collision or timeout |
| **Phase 2** | Admission rejected | `gateway_unavailable` / `auth_failed` | Admission controller decision log |
| **Phase 3** | Environment launch fails | `launch_plan_unavailable` | CLI runner pre-launch validation |
| **Phase 4** | Provider call error | `provider_error` / `protocol_error` | Gateway route span `Outcome: "error"` |
| **Phase 5** | DB commit fails | `persistence_failed` | Result sink persistence error |

---

## 4. Current Standby Status & Next Action

- **Current State**: `STANDBY` (`live_runs=false`).
- **Observation Guard**: Zero live requests initiated during standby. Watcher is configured to observe ONE rerun only upon explicit deploy completion and token authorization (Gate D-V3-25B).
- **Next Action**: When fix deploy is completed and Principal authorizes `live_runs=true`, observe the single live task execution, capture phase timestamps and metadata IDs into `LIVE-watch.md`, and record final PASS verdict.

---

## 5. Verification Commands & Exit Codes

- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go test ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `git diff --check .deploy-control/p0/evidence/LIVE-watch.md`: **PASS** (exit code 0)

---

## 6. Non-Claims

- `AcceptanceClaim`: `false` (standby mode; awaiting fix deploy & live token authorization)
- `LiveEndpointUsed`: `false`
- Zero polling loops, hammering, or unauthorized live API calls executed.
- Product source code in `server/...` remained strictly **READ-ONLY**.
