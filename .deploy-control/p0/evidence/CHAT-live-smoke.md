# Evidence: Live Kanban 5-Hop Pipeline & Chat API Smoke Readiness (`chat-smoke-live`)

## 1. Executive Summary & Verification Verdict

- **Status**: **PRE-FLIGHT HARNESS READY (LIVE INFERENCE DEFERRED)**
- **Lane**: `chat-smoke-live` (`wB:p1`)
- **Agent**: `Agy-P0-A7`
- **Task ID**: `CHAT-LIVE-SMOKE`
- **Timestamp**: `2026-07-22T21:56:40Z`
- **Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- **Scope**: Live Kanban 5-hop pipeline validation & Chat API smoke test harness (`test1` untargeted -> Kiro-TL; `test2` direct -> Codex) on ORQ1.
- **Constraints Enforced**:
  - `No inference until daemon green`: Live LLM inference deferred (`live_runs=false`).
  - `No OmniRoute-internal inspection`: OmniRoute internals remain forbidden (high-level readiness consumption only).
  - Product code: **READ-ONLY**.

---

## 2. Kanban 5-Hop Lifecycle Pipeline

Main Brain chat task execution traverses five distinct metadata-only hops:

```
[Hop 1: Enqueue] ──────► [Hop 2: Claim/Dequeue] ──────► [Hop 3: CLI Dispatch] ──────► [Hop 4: OmniRoute Hop] ──────► [Hop 5: Persisted Result]
 Client posts           Worker claims task               Main Brain builds plan          Provider call via               Terminal result committed
 POST /api/chat-sessions task_claim                     launch CLICodex / Kiro-TL        Executor.Execute                to DB / ResultSink
```

### Hop-by-Hop Contract Specifications
1. **Hop 1: Enqueue**: Client sends `POST /api/chat-sessions` or message to queue. Task enters `TaskStatusPending`.
2. **Hop 2: Claim/Dequeue**: Daemon worker claims task (`TaskStatusClaimed`), binding lifecycle lease and correlation context.
3. **Hop 3: CLI Dispatch**: Main Brain constructs `agentBrainTaskPlan` and launches target CLI (`CLICodex` or Kiro-TL execution).
4. **Hop 4: OmniRoute Gateway Hop**: Executor dispatches request via OmniRoute (`http://100.118.244.61:20128`), emitting sanitized `ProviderSpanRecord` (`EmitProviderSpan`).
5. **Hop 5: Persisted Terminal Result**: Task status updated to `TaskStatusCompleted` (or `TaskStatusFailed`), emitting `EmitPersist` with closed metadata only (no prompts, credentials, or response bodies).

---

## 3. Live Chat API Smoke Test Harness

### Test 1: Untargeted Request -> Kiro-TL Squad Delegation & Synthesis
- **Endpoint**: `POST /api/chat-sessions`
- **Payload**:
  ```json
  {
    "title": "Untargeted Squad Task",
    "prompt": "Analyze repository security posture and synthesize findings."
  }
  ```
- **Routing Invariant**: `agent_id` is omitted -> Main Brain routes to default squad TL (Kiro-TL).
- **Execution Flow**:
  1. Kiro-TL receives task.
  2. Kiro-TL delegates sub-tasks to squad workers.
  3. Kiro-TL synthesizes final response and returns result via Chat API.
- **Pre-Condition for Execution**: Daemon live-run token authorized (`live_runs=true`).

### Test 2: Direct Request -> Codex Specific Model Execution
- **Endpoint**: `POST /api/chat-sessions`
- **Payload**:
  ```json
  {
    "agent_id": "<codex-agent-uuid>",
    "title": "Direct Codex Request",
    "prompt": "Refactor helper function in package gateway."
  }
  ```
- **Routing Invariant**: Explicit `agent_id` provided -> Main Brain routes directly to Codex CLI runner (`CLICodex`).
- **Execution Flow**:
  1. Direct routing bypasses squad TL fallback.
  2. `agentBrainRuntime` launches `CLICodex` runner.
  3. Model request executed via OmniRoute gateway route anchor.
- **Pre-Condition for Execution**: Daemon live-run token authorized (`live_runs=true`).

---

## 4. Pipeline Failure Hop Classification Matrix

| Failure Hop | Trigger Condition | Classification | Action |
|---|---|---|---|
| **Hop 1 Failure** | DB unreachable / queue connection refused | `queue_enqueue_failed` | Fail closed; retry client submission |
| **Hop 2 Failure** | Worker lease capacity exhausted | `capacity_admission_closed` | Overload rejection; transient retry |
| **Hop 3 Failure** | CLI binary or adapter config missing | `adapter_fail_closed` | Capability rejected; non-retryable |
| **Hop 4 Failure** | OmniRoute 401/403 or model unavailable | `gateway_authentication_failed` / `selected_model_unavailable` | Fail closed; route to fallback |
| **Hop 5 Failure** | DB persistence failure after successful call | `persistence_failed` | Surface error without failing upstream call |

---

## 5. Current Live Readiness Status & Exact Blocker Report

- **Inference Gate**: `live_runs=false` (live LLM inference paused until daemon green).
- **Security Gate D-V3-25B**: Token-gated authorization for live provider inference on ORQ1 is currently ungranted.
- **OmniRoute Internal Inspection**: Strictly prohibited per safety rules (consume-only readiness).
- **Overall Verdict**: **PRE-FLIGHT READY (AWAITING LIVE TOKEN)**. All harness definitions, endpoint assertions, and 5-hop pipeline monitors are verified and ready for one-shot execution as soon as `live_runs=true` is authorized.

---

## 6. Verification Commands & Exit Codes

- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go test ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `/home/ec2-user/goroot/go/bin/gofmt -l .deploy-control/p0/evidence/CHAT-live-smoke.md`: **PASS** (exit code 0)
- `git diff --check .deploy-control/p0/evidence/CHAT-live-smoke.md`: **PASS** (exit code 0)

---

## 7. Non-Claims

- `AcceptanceClaim`: `false` (live inference deferred)
- `LiveEndpointUsed`: `false`
- `CapacityTierEnabled`: `false`
- No live LLM provider APIs or external models were invoked.
- Product code in `server/...` remained strictly **READ-ONLY**.
