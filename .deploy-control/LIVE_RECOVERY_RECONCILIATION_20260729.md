# P0 Live Recovery & Kanban Reconciliation Ledger (2026-07-29)

**Author:** Agy-P0-A8 (Kanban Control Steward)  
**Date:** 2026-07-29  
**Issue:** ORQ-67 (`1d27239f-ac7b-4936-8cfe-9e64662125e6`)  
**Canonical Kanban Board URL:** `https://orq1.tail96e2c0.ts.net`  
**OpenSpec Validation:** `openspec validate --all --strict` (5/5 passed)

---

## 1. Live Incident & Recovery Record (2026-07-29)

### 1.1 ORQ2 Daemon Regression (15:04:41Z Root Cause)
At 2026-07-29T15:04:41Z, validation of the ORQ-23 harness executed un-isolated systemd service commands on the live system, restarting the daemon process and changing its PID from 720029 to 1285683.
- **Root Cause:** The systemd service restart re-exposed the pre-existing binary already referenced by `ExecStart` (`e0510d7...`), rather than a binary swap.
- **Impact:** In-flight tasks (including ORQ-54 and ORQ-63) were interrupted, and pre-token-only binary behavior was exposed, causing AGY tasks to fail before execution due to directory traversal on sibling symlinks (`cli.log`) in task-home credential paths.

### 1.2 Phase-1 Bounded Rollback & Operational Verification
- **Admission Freeze Mechanism:** Executed under queue freeze via `LOCK TABLE agent_task_queue IN SHARE MODE` using the UTC DB snapshot method.
- **Target Artifact SHA-256:** Bounded rollback restored the proven token-only daemon artifact `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`.
- **Health & Readiness Endpoints:** Verified operational health via daemon endpoint `127.0.0.1:19514/health` and backend readiness via `127.0.0.1:18080/readyz` (both returning 200 OK).
- **Active Runtime Allowlists Restored:**
  - **AGY Slots:** 162, 163, 168, 169
  - **Kiro Slots:** 139, 140, 143, 149
  - **Codex Slots:** 152, 170
- **Operational Verification:** Task 07172690 reached AGY, cloned the repository, and ran for 5 minutes cleanly with `NRestarts=0` and zero `cli.log` traversal errors.

### 1.3 ORQ-64 Credential Exposure Incident (Content-Free Control)
During the un-isolated ORQ-23 harness run, Codex credentials were printed in error output.
- **Content-Free Compliance:** Zero secret values, credentials, or token string contents are recorded in repository logs, git commits, or markdown artifacts.
- **Harness Isolation Standard:** The ORQ-23 harness must be patched to run strictly within an isolated `mktemp` test root, stub/refuse `systemctl`, and sanitize all stdout/stderr output prior to any re-execution.
- **Status:** `blocked` pending harness patch and owner credential re-authentication.

### 1.4 ORQ-65 Recovery (Status: DONE)
- **Scope:** Restoration of AGY token-only task-home allowlist.
- **Status:** Closed as `done` by GTL (`534a72a9-0dca-40ae-a433-5c4d4100f834`). Live daemon hash confirmed as `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000` with active allowlists.

### 1.5 ORQ-66 Combined-Daemon Card (Status: IN REVIEW)
- **Scope:** Single clean daemon binary combining AGY token-only task-home allowlist (`antigravity_home.go`) and reasoning admission validation (`6bf0090`, `d9d569d`).
- **Current Status:** `in_review` (R2 rejected by GTL `4b3f6f15-d744-4f82-ba07-f9bb931db2de`; awaiting clean two-file port by a strong agent).

### 1.6 ORQ-58 Production Deployment (Status: DONE)
- **Scope:** Canonical rebuild combining ORQ-26 and upload fixes.
- **Status:** Production deployment completed.
  - **Git Revision:** `112e8dada455b4e7a3400e63728e00e6e3a0aa27`
  - **Live Docker Image:** `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`
  - **Rollback Git Revision:** `8227241`
  - **Rollback Docker Image:** `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`
  - **Canary Result:** ORQ-70 canary OK.

---

## 2. Comprehensive Non-Done Kanban Card Reconciliation (28 Cards Audited)

| Card | Identifier | Current Status | Assignee / Role | Evidence & Context | Proposed GTL Action |
|:---|:---|:---|:---|:---|:---|
| 23 | ORQ-23 | `blocked` | Unassigned | Harness printed credential (ORQ-64) & changed daemon PID. | Maintain `blocked` pending ORQ-64 harness patch & owner re-auth. |
| 26 | ORQ-26 | `in_review` | Opus48-A | Code topic accepted (`0d7a6df`/`8227241`). | Integrated into completed ORQ-58 deployment (`112e8dada455b4e7a3400e63728e00e6e3a0aa27`). |
| 32 | ORQ-32 | `cancelled` | Unassigned | Handshake token rotation (superseded by ORQ-36/ORQ-43). | Maintain `cancelled`. |
| 35 | ORQ-35 | `in_progress` | Gemini-3.6-Flash-B | SCRAM posture & cutover readiness. Unblocked by ORQ-65. | Continue execution towards production cutover. |
| 37 | ORQ-37 | `blocked` | Kiro-Opus5 | MCP credential lifecycle & Cedar evidence. | Maintain `blocked` pending umask & DB security cutovers. |
| 38 | ORQ-38 | `in_review` | Opus48-A | Identifier collision reporting fix accepted. | Maintain `in_review` pending final GTL close. |
| 39 | ORQ-39 | `blocked` | Unassigned | Browser QA pipeline execution. | Maintain `blocked` pending pipeline environment setup. |
| 40 | ORQ-40 | `blocked` | Unassigned | Codex CLI version alignment across environments. | Maintain `blocked` pending CLI binary release alignment. |
| 41 | ORQ-41 | `in_progress` | Opus48-B | Decouple metadata from paid task execution (`e0b0155`). | Continue `in_progress` under dormant-scaffolding review. |
| 42 | ORQ-42 | `blocked` | Opus48-B | Controlled JWT_SECRET rotation post-ORQ-30. | Maintain `blocked` pending GTL maintenance window. |
| 43 | ORQ-43 | `blocked` | Unassigned | Daemon token `mdt_` lifecycle and rotation. | Maintain `blocked` pending token rotation governance. |
| 44 | ORQ-44 | `blocked` | Unassigned | OmniRoute gateway inference key rotation. | Maintain `blocked` pending gateway governance. |
| 45 | ORQ-45 | `cancelled` | Unassigned | ORQ-17 zero-task validation card. | Maintain `cancelled`. |
| 47 | ORQ-47 | `todo` | Unassigned | Restore durable daily agent cache lifecycle. | Maintain `todo` for future sprint dispatch. |
| 48 | ORQ-48 | `in_review` | Gemini-3.6-Flash-A | Fleet documentation & Kanban control steward. | Maintain `in_review` for continuous stewardship. |
| 54 | ORQ-54 | `in_progress` | Opus48-A | Direct @agent chat escape hatch implementation. | Continue `in_progress`; await ORQ-70 canary verification. |
| 57 | ORQ-57 | `in_review` | Unassigned | Deploy safety preflight accepted (`127069b`). | Maintain `in_review`; merged into ORQ-58 deploy (`112e8dada455b4e7a3400e63728e00e6e3a0aa27`). |
| 58 | ORQ-58 | `done` | Unassigned | Completed PROD deploy (rev `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live `sha256:e14f5c35d064...`). | Close as `done` (ORQ-70 canary OK). |
| 59 | ORQ-59 | `in_review` | Unassigned | GSD Wave 3 live rebaseline accepted (`f9fafe70`). | Maintain `in_review` pending final GTL close. |
| 60 | ORQ-60 | `in_review` | Unassigned | PostgreSQL least privilege R3 (`5e18bc0f`). | Maintain `in_review` pending GTL security verdict. |
| 61 | ORQ-61 | `in_review` | Unassigned | OpenSpec orphan rotation-router fix (`0cdf4a`). | Maintain `in_review` (unblocked after ORQ-62 accept). |
| 62 | ORQ-62 | `in_review` | Unassigned | Full-lineage OpenSpec reconciliation accepted (`7618599f`). | Maintain `in_review` pending final GTL close. |
| 63 | ORQ-63 | `in_review` | Unassigned | Codex slot 170 capacity registration completed. | Maintain `in_review`; rerun canary post-ORQ-66 daemon cutover. |
| 64 | ORQ-64 | `blocked` | Unassigned | Security incident: Codex credential printed by harness. | Maintain `blocked` pending harness patch & credential re-auth. |
| 66 | ORQ-66 | `in_review` | Kiro-Opus5 | Combined daemon (token-only + reasoning admission). | Maintain `in_review`; dispatch strong agent for two-file port. |
| 67 | ORQ-67 | `in_progress` | Agy-P0-A8 | OpenSpec & Kanban control steward (this card). | Submit results and move to `in_review`. |
| 68 | ORQ-68 | `in_progress` | Agy-P0-A7 | Scheduler reconciliation (failed task agent release). | Continue `in_progress` for DB/handler race fix & tests. |
| 69 | ORQ-69 | `todo` | Unassigned | Temporary runtime recovery (clear thinking_level). | Temporary config applied for Opus48-A; pending canary. |

---

## 3. OpenSpec Lineage & Validation

OpenSpec changeset `live-recovery-reconciliation` created under `openspec/changes/live-recovery-reconciliation/`:
- `proposal.md`
- `design.md`
- `specs/live-recovery/spec.md`
- `tasks.md`

All 5 OpenSpec changesets pass strict validation (`openspec validate --all --strict`):
- `change/agent-credential-isolation`
- `change/chat-orchestration-standard`
- `change/live-recovery-reconciliation`
- `change/native-runtimes-onboarding`
- `change/rotation-parity-polyglot`
