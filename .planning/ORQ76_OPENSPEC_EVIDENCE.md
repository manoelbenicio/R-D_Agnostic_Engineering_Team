# OpenSpec Evidence Reconciliation — ORQ-76

> **Recorder / Coordinator:** Gemini-3.6-Flash-A (`3e83b35d-d40d-4047-b76e-5966571fad77`)  
> **Target Issue:** ORQ-76 (`ed73632e-4e10-4839-98e2-aa315b02dd73`)  
> **Timestamp (UTC):** 2026-07-29T18:39:46Z  
> **Mandate Reference:** `[ORQ-76-OPENSPEC-EVIDENCE-STEWARD-20260729]`  
> **Scope:** Documentation-only stewardship. Reconciliation of updated OpenSpec with verified production and Kanban evidence from 2026-07-29. Zero secrets exposed, zero product code edits, zero invented completions.

---

## 1. Summary of Reconciled Technical Evidence & Incident Governance

### A. ORQ-66 Rollback & Preserved Artifact Metadata
- **Candidate Daemon Binary SHA:** Candidate commit SHA `9c8b5401...` deployed for production canary testing (ORQ-74) and initial remediation attempt (ORQ-75).
- **Incident Trigger:** Task `0d29613c-bbb4-4f95-a1de-2d379bfc318c` (ORQ-74) and task `dd9bf284-8964-4bed-afa4-d711da695b6d` (ORQ-75) claimed approved AGY account `adb99fe1-9fd2-4a81-9ce6-7755fc1dc70e`. The candidate daemon initialized the physical token-only task-home successfully, but failed prior to CLI invocation with `agent brain admission failed closed: launch_plan_unavailable`.
- **Root Cause Analysis:** Unconditional call `d.agentBrain.buildLaunch(ctx, agentBrainPlan, ...)` at `daemon.go` (~line 3728) when `agentBrainPlan` is `nil` in disabled/native mode. `buildLaunch` correctly fails closed when given a `nil` plan.
- **Rollback Procedure & Execution:** Executed under a PostgreSQL `SHARE` admission freeze with active queue count = 0. The daemon binary was atomically restored from SHA `9c8b5401...` to the previously validated SHA `88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`.
- **Preserved Failed Artifact:** The failed daemon binary was preserved content-free for post-mortem analysis as `multica-auth-credential-home-v1.failed-orq66-20260729T182744Z`.
- **Post-Rollback Production Status:** Daemon process active/running (PID 1609241, NRestarts=0), health endpoint HTTP 200 responsive, native runtimes listed, active queue count = 0.
- **Rollback Acceptance:** Independently reviewed and accepted by Kiro-Opus5 (`[Kiro-Opus5 rollback acceptance]`).

### B. ORQ-75 Remediation State
- **Card Status:** `in_progress` (assigned to Codex-B / `30bc4405-646f-4fdc-b8d7-78262fe1aff9`).
- **Remediation Contract & Requirements:**
  1. Implement nil Agent Brain plan bypass on top of ORQ-66 in `daemon.go`.
  2. Add focused regression test proving native runtime launch proceeds cleanly when `agentBrainPlan` is `nil` / disabled.
  3. Correct cutover mode assertion and evidence mismatch prior to requesting review.
- **Lineage Qualification:** ORQ-75 is currently in progress. Final commit SHA will be recorded in OpenSpec and live ledgers **only after** verified implementation evidence exists. Completion is strictly NOT invented.

### C. Combined Backend Candidate `63ead4df` (ORQ-13 / ORQ-41 / ORQ-54)
- **Topic Integration Branch:** `origin/integration/orq13-orq54-production-20260729`
- **Candidate Tip Commit (40-hex):** `63ead4d31481b7e41acdfd1ef24a1ff9c43d3b74` (`63ead4df`)
- **Integrated Component Scope:**
  - **ORQ-13 (Usage Cost):** Tier-aware pricing resolution (`6a968ad` / `bdf3a55`), effective-time windows, and immutable account snapshot.
  - **ORQ-41 (Decouple Metadata from Execution):** Enqueues tasks strictly on explicit assignee transitions carrying execution intent (`fieldOnlyUpdateCarriesExecutionIntent` + `HasAnyTaskForIssueAndAgent`), plus `documentation_only` comment mode on `POST/PUT /api/comments`.
  - **ORQ-54 (Chat Escape Hatch):** Direct `@agent` turn routing, cross-agent session serialization keyed by `chat_session_id`, scoped `(chat_session, agent)` resume pointers, and offline runtime fallback.
- **PG17 Focused Gate Verification:** Executed against a disposable PostgreSQL 17 test cluster (port 55432, schema through migration 129). All 57 subtests passed (status-only updates with stale assignee = 0 tasks, explicit transition = 1 task, `documentation_only` comment mode = 0 tasks, chat session serialization across agents = 1 in-flight turn per session, issue work parallelism preserved, owner session pointer uncorrupted).
- **GTL / Kiro Verdict:** **Kiro ACCEPT DEPLOY** (`[GTL+KIRO INTEGRATION ACCEPT 2026-07-29]`), held in `in_review` pending joint production cutover deploy.

---

## 2. Qualification of Source-Accepted vs. Production-Live Claims

| Scope / Component | Source-Accepted Commit / Branch | Live Production Status | Governance & Qualification Details |
|---|---|---|---|
| **Canonical Production Base** | `15626386da2725af8e8d4ac611754cffe359fe31` | **PRODUCTION LIVE** | Deployed via ORQ-68 wrapper (`multica-backend:orq68-reconcile-1562638-20260729`, image `sha256:ee1c4b71...`). Verified via live canary task `7ab2977d...` (`ORQ68_CANARY_OK`). |
| **Combined Candidate (ORQ-13/41/54)** | `63ead4d31481b7e41acdfd1ef24a1ff9c43d3b74` (`63ead4df`) | **SOURCE-ACCEPTED (NOT LIVE)** | Integrated on `origin/integration/orq13-orq54-production-20260729`. Kiro ACCEPT DEPLOY. Pending joint cutover. |
| **ORQ-66 Daemon Candidate** | `9c8b5401...` | **ROLLED BACK TO `88ca4f39`** | Rolled back to verified SHA `88ca4f39...` after `launch_plan_unavailable` failure. Failed binary preserved as `multica-auth-credential-home-v1.failed-orq66-20260729T182744Z`. |
| **ORQ-75 Remediation** | `in_progress` | **NOT LIVE (IN PROGRESS)** | Active remediation. Final commit pending verified implementation evidence. |
| **ORQ-35 (PG Hardening)** | `f8657cd6...` / `cde2e089...` | **SOURCE-ACCEPTED (NOT LIVE)** | Source/traceability accepted by GTL/Kiro (`[GTL+KIRO ACCEPT 2026-07-29]`). Not ancestral/live in production revision `15626386...`. |
| **ORQ-48 (Fleet Documentation)** | `f05166fb9ad882b2cf8338ad4ec1e1e7d96d4991` | **DOCUMENTATION ACCEPTED** | Docs-only accepted by GTL/Kiro (`[GTL+KIRO ACCEPT 2026-07-29]`). Cline rebuild/canary tasks 2.4/3.2 remain pending ORQ-66 durable daemon deployment. |

---

## 3. Strict OpenSpec & Hygiene Verification

- **OpenSpec Validation:** `openspec validate --all --strict --no-interactive`  
  **Result:** `4 passed, 0 failed` (100% strict compliance across all active change proposals: `agent-credential-isolation`, `chat-orchestration-standard`, `native-runtimes-onboarding`, `rotation-parity-polyglot`).
- **Whitespace & Formatting:** `git diff --check`  
  **Result:** `0 issues` (clean).
- **Product Code Protection:** Zero edits to product source files (`multica-auth-work/` remains 100% untouched). Zero secret exposures.
