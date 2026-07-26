# L7 — WS/UI Delivery + Kanban UI Handoff

agent: Agy-P0-A7
lane: L7
task: P0-L7-WS-UI
pane: wB:p1
timestamp: 2026-07-22T04:17:00Z
status: BLOCKED (Go toolchain absent; node_modules absent)

## Deliverables

### 1. `obs_delivery.go` (NEW — created)

**Path**: `server/internal/daemonws/obs_delivery.go`

Metadata-only HopDelivery span helper. Follows L5 `e2e.contract.go` exactly:
- Uses `e2e.NewSpan(e2e.HopDelivery, corr)` with the L5 Recorder API
- Required IDs per L5 contract: `SessionID` + `DeliveryID`
- Counters: only the four allowed by L5 for HopDelivery: `delivery_latency_ms`, `drop_count`, `reconnect_count`, `backpressure_count`
- Zero content/payload fields — metadata-only by construction
- `DeliveryResult` struct carries only classification-level fields
- `EmitDelivery()` constructs, finishes, and emits the span in one call
- Nil recorder is safe (no-op)
- Fail-closed: validation errors refuse the span, caller does not retry

### 2. `obs_delivery_test.go` (NEW — created)

**Path**: `server/internal/daemonws/obs_delivery_test.go`

Five test cases using `e2e.MemorySink` (deterministic, no network):
- `TestEmitDelivery_ValidSpan` — happy path with delivered outcome, verifies all correlation IDs, counters, contract_version, secrets_present=false, timestamps
- `TestEmitDelivery_Dropped` — dropped outcome with reason_code, all four counters verified
- `TestEmitDelivery_MissingSessionID_Rejected` — fail-closed: empty session_id rejected, zero spans recorded
- `TestEmitDelivery_MissingDeliveryID_Rejected` — fail-closed: empty delivery_id rejected, zero spans recorded
- `TestEmitDelivery_NilRecorder_NoOp` — nil recorder returns nil error

### 3. UI Contract Verification (static analysis, no edits)

All three UI components verified clean:
- `board-view.tsx`: No direct API calls; receives data from stores; no fake/mock/stub/fallback patterns
- `issue-detail.tsx`: Uses real `useQuery` with `api.getIssue()`; proper `isLoading` → `<Skeleton>` path; no synthetic fallback data; proper loading/error/terminal state handling
- `execution-log-section.tsx`: Uses real `api.listTasksByIssue()`; proper status filtering (queued/dispatched/waiting_local_directory/running → active, completed/failed/cancelled → past); real cancel via `api.cancelTask()`; real retry via `api.rerunIssue()`; WS invalidation via `issueKeys.tasks` prefix-match — no polling; no fake success

Zero grep hits for: `fake|mock|stub|fallback.*success|synthetic.*result|placeholder.*data|hardcoded.*status` across all three files.

## Validation Commands & Results

| Command | Exit Code | Result |
|---|---|---|
| `git diff --check -- server/internal/daemonws/obs_delivery.go` | 0 | PASS |
| `git diff --check -- server/internal/daemonws/obs_delivery_test.go` | 0 | PASS |
| `git diff --check -- packages/views/issues/components/{board-view,issue-detail,execution-log-section}*` | 0 | PASS (no changes) |
| `rg 'fake\|mock\|stub\|fallback.*success' board-view.tsx` | 0 | 0 matches |
| `rg 'fake\|mock\|stub\|fallback.*success' issue-detail.tsx` | 0 | 0 matches |
| `rg 'fake\|mock\|stub\|fallback.*success' execution-log-section.tsx` | 0 | 0 matches |

## BLOCKERS

### B1: Go toolchain absent — cannot compile or test Go files

- **Fact**: `$HOME/.local/toolchains/go1.26.1/bin/go` does not exist
- **Impact**: `obs_delivery.go` and `obs_delivery_test.go` are syntactically correct by static analysis but CANNOT be compiled or tested
- **Owner**: Infrastructure / L8 verifier (if toolchain appears)
- **Next action**: When Go toolchain is available, run `go test ./server/internal/daemonws/... -run TestEmitDelivery -v` and `go vet ./server/internal/daemonws/...`

### B2: node_modules absent — cannot run UI tests

- **Fact**: `multica-auth-work/packages/views/node_modules` does not exist; `multica-auth-work/node_modules` does not exist
- **Impact**: `execution-log-section.test.tsx` and `issue-detail.test.tsx` cannot be executed
- **Owner**: Infrastructure (dependency install prohibited by L7 constraints)
- **Next action**: When deps available, run `pnpm vitest run packages/views/issues/components/execution-log-section.test.tsx`

### B3: hub.go call site — handoff to L1

- **Fact**: `hub.go` is MUST-NOT-TOUCH for L7
- **Impact**: The `DeliveryRecorder.EmitDelivery()` call must be inserted in hub.go at the point where a WS frame is confirmed delivered or dropped
- **Owner**: L1 (Lead Integrator)
- **Next action**: L1 adds `NewDeliveryRecorder(rec)` construction during Hub init and calls `EmitDelivery()` at the delivery/drop site in `hub.go:notifyTaskAvailable` and `hub.go:notifyRuntimeProfilesChanged`

## Files Changed

| File | Action | Lines |
|---|---|---|
| `server/internal/daemonws/obs_delivery.go` | CREATED | 67 |
| `server/internal/daemonws/obs_delivery_test.go` | CREATED | 131 |
| `packages/views/issues/components/board-view.tsx` | REVIEWED (no edit) | 0 |
| `packages/views/issues/components/issue-detail.tsx` | REVIEWED (no edit) | 0 |
| `packages/views/issues/components/execution-log-section.tsx` | REVIEWED (no edit) | 0 |

## Non-claims

- Go code NOT compiled (no Go toolchain)
- Go tests NOT executed (no Go toolchain)
- UI tests NOT executed (no node_modules)
- hub.go NOT edited (MUST-NOT-TOUCH; handoff to L1)
- No deploy, inference, secret read, commit, push, or dependency install performed
