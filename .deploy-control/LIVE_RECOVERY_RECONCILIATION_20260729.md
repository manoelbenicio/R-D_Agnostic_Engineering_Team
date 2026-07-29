# P0 Live Recovery & Kanban Reconciliation Ledger (2026-07-29)

**Author:** Gemini-3.6-Flash-A (Kanban Control Steward)  
**Date:** 2026-07-29  
**Issue:** ORQ-67 (`1d27239f-ac7b-4936-8cfe-9e64662125e6`)  
**OpenSpec Validation:** `openspec validate --all --strict` (5/5 passed)

---

## 1. Live Incident & Recovery Record (2026-07-29)

### 1.1 ORQ2 Daemon Regression (15:04:41Z)
At 2026-07-29T15:04:41Z, validation of the ORQ-23 harness invoked systemd service lifecycle commands on the live system, changing daemon PID from 720029 to 1285683 without binary change. This interrupted in-flight tasks (including ORQ-54 and ORQ-63) and exposed pre-token-only binary behavior (`e0510d7...`), causing AGY tasks to fail before execution on sibling symlinks (`cli.log`) in task-home credential directories.

### 1.2 Phase-1 Bounded Rollback (Artifact SHA-256 `88ca4f39`)
Under active queue `SHARE` lock (`pg_advisory_lock`), Phase-1 runtime recovery performed a bounded rollback to the proven token-only artifact SHA-256 `88ca4f39`.
- **Active AGY Allowlists Restored:** Slots 162, 163, 168, 169.
- **Operational Verification:** Task 07172690 reached AGY, cloned the repository, and executed for 5 minutes without `cli.log` or task-home directory traversal errors.
- **Service Status:** Active with `NRestarts=0`.

### 1.3 ORQ-64 Credential Exposure Incident (Content-Free)
During the un-isolated execution of the ORQ-23 harness, Codex credentials were printed in assertion failure output.
- **Content-Free Record:** Zero secret values or token contents are included in repository logs, diffs, or markdown artifacts.
- **Remediation Requirement:** The test harness must be patched to execute strictly inside an isolated `mktemp` test root, stub/refuse `systemctl`, and sanitize all stdout/stderr output using synthetic sentinel fixtures before any future run.
- **Status:** `blocked` on harness code containment patch and owner credential re-authentication.

### 1.4 ORQ-65 Recovery (Done)
- **Scope:** Restoration of AGY token-only task-home allowlist.
- **Status:** Closed as `done` by GTL (`534a72a9-0dca-40ae-a433-5c4d4100f834`). Live daemon hash confirmed as `88ca4f39` with operational allowlists (162, 163, 168, 169).

### 1.5 ORQ-66 Combined-Daemon Card (In Review)
- **Scope:** Single clean daemon binary combining AGY token-only task-home allowlist (`antigravity_home.go`) and reasoning admission validation (`6bf0090`, `d9d569d`).
- **Current Status:** `in_review` (R2 rejected by GTL `4b3f6f15-d744-4f82-ba07-f9bb931db2de`; awaiting clean two-file token-only port by a strong agent with deterministic build evidence).

---

## 2. Comprehensive Non-Done Kanban Card Reconciliation (28 Cards)

| Card | Identifier | Current Status | Assignee / Role | Evidence & Context | Proposed GTL Action |
|:---|:---|:---|:---|:---|:---|
| 23 | ORQ-23 | `blocked` | Unassigned | Harness printed credential (ORQ-64) & changed daemon PID. | Maintain `blocked` pending ORQ-64 harness patch & owner re-auth. |
| 26 | ORQ-26 | `in_review` | Opus48-A | Code topic accepted (`0d7a6df`/`8227241`). | Maintain `in_review`; integrate in ORQ-58 canonical rebuild. |
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
| 54 | ORQ-54 | `in_progress` | Opus48-A | Direct @agent chat escape hatch implementation. | Continue `in_progress`; await ORQ-57/58 deploy for canary. |
| 57 | ORQ-57 | `in_review` | Unassigned | Deploy safety preflight accepted (`127069b`). | Maintain `in_review`; merge into canonical integration tree. |
| 58 | ORQ-58 | `in_review` | Unassigned | Canonical rebuild combining ORQ-26 and upload fixes. | Maintain `in_review`; execute image build & canary. |
| 59 | ORQ-59 | `in_review` | Unassigned | GSD Wave 3 live rebaseline accepted (`f9fafe70`). | Maintain `in_review` pending final GTL close. |
| 60 | ORQ-60 | `in_review` | Unassigned | PostgreSQL least privilege R3 (`5e18bc0f`). | Maintain `in_review` pending GTL security verdict. |
| 61 | ORQ-61 | `in_review` | Unassigned | OpenSpec orphan rotation-router fix (`0cdf4a`). | Maintain `in_review` (unblocked after ORQ-62 accept). |
| 62 | ORQ-62 | `in_review` | Unassigned | Full-lineage OpenSpec reconciliation accepted (`7618599f`). | Maintain `in_review` pending final GTL close. |
| 63 | ORQ-63 | `in_review` | Unassigned | Codex slot 170 capacity registration completed. | Maintain `in_review`; rerun canary post-ORQ-66 daemon cutover. |
| 64 | ORQ-64 | `blocked` | Unassigned | Security incident: Codex credential printed by harness. | Maintain `blocked` pending harness patch & credential re-auth. |
| 66 | ORQ-66 | `in_review` | Kiro-Opus5 | Combined daemon (token-only + reasoning admission). | Maintain `in_review`; dispatch strong agent for two-file port. |
| 67 | ORQ-67 | `in_progress` | Gemini-3.6-Flash-A | OpenSpec & Kanban control steward (this card). | Submit results and move to `in_review`. |
| 68 | ORQ-68 | `in_progress` | Agy-P0-A7 | Scheduler reconciliation (failed task agent release). | Continue `in_progress` for DB/handler race fix & tests. |
| 69 | ORQ-69 | `todo` | Unassigned | Temporary runtime recovery (clear thinking_level). | Temporary config applied for Opus48-A; pending canary. |

---

## 3. OpenSpec Lineage & Validation

OpenSpec changeset `live-recovery-reconciliation` created under `openspec/changes/live-recovery-reconciliation/`:
- `proposal.md`
- `design.md`
- `specs/live-recovery/spec.md`
- `tasks.md`

All 5 OpenSpec changesets pass strict validation:
```
- Validating...
✓ change/agent-credential-isolation
✓ change/chat-orchestration-standard
✓ change/live-recovery-reconciliation
✓ change/native-runtimes-onboarding
✓ change/rotation-parity-polyglot
Totals: 5 passed, 0 failed (5 items)
```
