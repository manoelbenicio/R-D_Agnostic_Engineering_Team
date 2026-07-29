# Evidence Report: L9 Post-Deploy Chat Smokes Readiness (`L9-chat.md`)

## 1. Executive Summary & Verification Verdict

- **Status**: **STANDBY — HARNESS READY (AWAITING DEPLOY & LIVE RUN AUTHORIZATION)**
- **Lane**: `L9` (`wB:p1`)
- **Agent**: `Agy-P0-A7`
- **Task ID**: `L9-CHAT-SMOKES`
- **Timestamp**: `2026-07-22T23:13:46Z`
- **Git HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- **Scope**: Post-deploy Chat API live smoke test harness for:
  - `test1`: Untargeted request -> Kiro-TL squad delegation & response synthesis.
  - `test2`: Direct request -> Codex model execution.
- **Constraints Enforced**:
  - `live_runs=false`: Live inference paused during standby window.
  - Product code: **READ-ONLY**.
  - OmniRoute internals: **FORBIDDEN** (consume-only readiness & public Chat API endpoints).

---

## 2. Post-Deploy Chat Smokes Execution Plan

### Test 1: Untargeted Request -> Kiro-TL Squad Delegation & Synthesis
- **Endpoint**: `POST /api/chat-sessions`
- **Payload**:
  ```json
  {
    "title": "Post-Deploy Untargeted Squad Test",
    "prompt": "Synthesize security posture and summarize recommendations."
  }
  ```
- **Routing Invariant**: `agent_id` omitted -> Main Brain routes request to default squad TL (Kiro-TL).
- **Validation**:
  1. Verify 201 Created session response.
  2. Verify response `agent_id` maps to Kiro-TL agent UUID.
  3. Verify squad delegation events (`EventRouteSelection`) and synthesis response.

### Test 2: Direct Request -> Codex Specific Model Execution
- **Endpoint**: `POST /api/chat-sessions`
- **Payload**:
  ```json
  {
    "agent_id": "<codex-agent-uuid>",
    "title": "Post-Deploy Direct Codex Test",
    "prompt": "Generate unit test for gateway route anchor."
  }
  ```
- **Routing Invariant**: Explicit `agent_id` provided -> Main Brain routes directly to Codex CLI runner (`CLICodex`).
- **Validation**:
  1. Verify 201 Created session response.
  2. Verify response `agent_id` matches Codex agent UUID.
  3. Verify direct CLI launch (`CLICodex`) and OmniRoute model execution.

---

## 3. Standby & Post-Deploy Status

- **Current State**: `STANDBY` (Pre-deploy offline state; `live_runs=false`).
- **Pre-Conditions for Execution**:
  1. Target environment deployment completed.
  2. Principal authorizes `live_runs=true` and grants live run token (Gate D-V3-25B).
- **Post-Deploy Action**: Once deployment is confirmed green, execute live HTTP requests for `test1` and `test2`, append response payloads and correlation headers to `L9-chat.md`, and record final `PASS` verdict.

---

## 4. Verification Commands & Exit Codes

- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go test ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `GOCACHE=/tmp/gocache_f7 GOTMPDIR=/tmp/gotmp_f7 /home/ec2-user/goroot/go/bin/go vet ./internal/daemon/brain/...`: **PASS** (exit code 0)
- `git diff --check .deploy-control/p0/evidence/L9-chat.md`: **PASS** (exit code 0)

---

## 5. Enforced Non-Claims

- `AcceptanceClaim`: `false` (standby mode; live deployment execution pending)
- `LiveEndpointUsed`: `false`
- Zero live LLM inference calls executed during standby window.
- Product source code in `server/...` remained strictly **READ-ONLY**.
