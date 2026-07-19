# Integration Push-Scope Matrix

**Snapshot HEAD:** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
**Timestamp:** 2026-07-18T17:35:00Z

## Objective
Authoritative file-level matrix mapping the mixed worktree against independent evidence records. This matrix is strictly for the principal to create atomic commits, excluding unaccepted/unowned/hostile work. 

## Atomic Commit Groups & Grades

### 1. Agent Brain v3: G2A Core & Contracts
**Grade:** ACCEPTED (via EV-G2A, REVIEW-I1I5-ACCEPT)
- `multica-auth-work/server/internal/daemon/brain/contracts.go`
- `multica-auth-work/server/internal/daemon/brain/identity.go`
- `multica-auth-work/server/internal/daemon/brain/coordinator.go`
- `multica-auth-work/server/internal/daemon/brain/executor.go`
- `multica-auth-work/server/internal/daemon/brain/capacity.go` (and test)
- `multica-auth-work/server/internal/daemon/brain/compatibility.go`
- `multica-auth-work/server/internal/daemon/brain/config.go`
- `multica-auth-work/server/internal/daemon/brain_integration.go` (and test)
- `multica-auth-work/server/internal/daemon/daemon.go`
- `multica-auth-work/server/internal/daemon/config.go`
- `multica-auth-work/server/internal/daemon/health.go`
- `multica-auth-work/server/cmd/multica/cmd_daemon.go`

### 2. Agent Brain v3: G2B Gateway & I3/I4/I5 Producers
**Grade:** ACCEPTED (via REVIEW-GW-RUNTIME-ACCEPT, REVIEW-I3I5-ACCEPT)
- `multica-auth-work/server/internal/daemon/gateway/policy.go`
- `multica-auth-work/server/internal/daemon/gateway/telemetry.go`
- `multica-auth-work/server/internal/daemon/gateway/registry.go` (and test)
- `multica-auth-work/server/internal/daemon/gateway/projection.go` (and test)
- `multica-auth-work/server/internal/daemon/gateway/profiles.go`
*Note: Some synthetic tests in gateway/ are PRODUCED/PENDING pB review.*

### 3. Agent Brain v3: G2C Runtimeenv & Security Corrections
**Grade:** ACCEPTED (via EV-NATIVE-OFFLINE, REVIEW-G3-02)
- `multica-auth-work/server/internal/daemon/runtimeenv/codex.go` (and test)
- `multica-auth-work/server/internal/daemon/runtimeenv/model.go` (and test)
- `multica-auth-work/server/internal/daemon/runtimeenv/assert.go`
- `multica-auth-work/server/internal/daemon/runtimeenv/env.go`
- `multica-auth-work/server/internal/daemon/runtimeenv/home.go`
- `multica-auth-work/server/internal/daemon/runtimeenv/adapter.go`
- `multica-auth-work/server/pkg/agent/codex.go` (and test)
- `multica-auth-work/server/pkg/agent/claude.go` (and test)
- `multica-auth-work/server/pkg/agent/models.go` (and test)
- `multica-auth-work/server/pkg/agent/environment.go`

### 4. Agent Brain v3: G2D Ops & Aggregate
**Grade:** ACCEPTED (via EV-G2D, REVIEW-AGG-ACCEPT)
- `multica-auth-work/server/internal/daemon/observability/aggregate.go` (and test)
- `multica-auth-work/server/internal/daemon/observability/realtime_process.go`
- `multica-auth-work/server/internal/daemon/observability/realtime_test.go`
- `multica-auth-work/server/internal/daemon/observability/schema.go`
- `multica-auth-work/server/internal/daemon/deploy/topology.go`
- `multica-auth-work/server/internal/daemon/deploy/rollout.go`
- `multica-auth-work/server/internal/daemon/deploy/secret_reference.go`

### 5. Native Runtimes Onboarding: Auth Backend (Task 1.7)
**Grade:** ACCEPTED (via EV-AUTH-1.7)
- `multica-auth-work/server/internal/handler/auth.go`
- `multica-auth-work/server/internal/middleware/auth.go` (and test)
- `multica-auth-work/server/internal/handler/passwordtest/provision_test.go`

### 6. Credential Isolation: Task 4.2 & 4.1
**Grade:** ACCEPTED (via EV-CREDISO-4.2, EV-CREDISO-4.1)
- `multica-auth-work/server/internal/rotation/service.go`
- `multica-auth-work/server/internal/rotation/rotation_e2e_test.go`

### 7. Credential Isolation: Task 5.4 Slices (Core + Email)
**Grade:** ACCEPTED (via EV-CREDISO-5.4-EMAIL, QA5-4-CORE)
- `multica-auth-work/server/pkg/redact/redact.go`
- `multica-auth-work/server/pkg/redact/redact_test.go`
- `multica-auth-work/server/internal/service/email.go`
- `multica-auth-work/server/internal/service/email_test.go`

---

## Pending & Excluded Sets (Do Not Commit)

### A. Persist Prodex Runtime (1.1-1.3)
**Grade:** PENDING (TECH PASS / CONTRACT INCOMPLETE via EV-PP-1.1-1.3-REVIEW)
- `multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go`

### B. Chat Orchestration Routing (Tasks 1.2-1.3)
**Grade:** REJECTED / REOPENED (via EV-CHAT-1.2-1.3-REOPENED)
- `multica-auth-work/server/internal/handler/chat.go`
- `multica-auth-work/server/internal/handler/agent.go`
- `multica-auth-work/server/internal/handler/workspace.go`

### C. Native Runtimes Mobile/Web (Task 1.5-1.6)
**Grade:** REJECTED / REOPENED
- `multica-auth-work/apps/mobile/**`
- `multica-auth-work/apps/web/**`
- `multica-auth-work/packages/core/api/client.ts`

### D. Vendor Model Visibility UI (Packet B)
**Grade:** PENDING (Pending final review)
- `multica-auth-work/packages/views/agents/**`
- `multica-auth-work/packages/core/runtimes/**`

### E. Credential Isolation Task 4.3 (Discovery) & 4.4 (Alerting)
**Grade:** REJECTED (via EV-CREDISO-4.3-INTERIM, EV-CREDISO-4.4-REVIEW)
- `multica-auth-work/server/internal/daemon/credential_session_discovery_producer.go`
- `multica-auth-work/server/internal/daemon/credential_session_monitor.go`

### F. Junk / Generated
**Grade:** JUNK / UNOWNED (Flagged, do not delete or commit)
- `files.txt`
- `multica-auth-work/NUL`
- `multica-auth-work/server/NUL`
- `opencode.json`
- `opencode.json.backup.*`

### G. Evidence / Planning Documents
**Grade:** ACCEPTED (Documentary tracking)
- `.planning/agent-brain-v3/**`
- `.deploy-control/**`
- `openspec/changes/**` (some updated, some untouched)

---
## Uncertainty List
- **Overlaps:** `daemon.go` and `config.go` contain overlaps between G3 security corrections, G4 capacity ledger, and G1/G2 setups. These were developed by Codex 1 in an integrated slice and should be committed atomically with the G2A core.
- **Missing provenance:** Any new file not explicitly matching the above independent reviews is considered UNKNOWN/UNOWNED and is excluded from the atomic commit lists.
- **Credential paths:** `secret_reference.go`, `codex_home.go`, `home.go`, `rotation/service.go` touch credential semantics but have been verified as reference-only or properly redacted (no real secret reads/prints).
