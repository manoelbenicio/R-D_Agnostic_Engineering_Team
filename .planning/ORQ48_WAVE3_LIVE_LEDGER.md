# ORQ-48 Fleet Documentation & Kanban Control — Wave 3 Live Ledger

> **Recorder / Coordinator:** Gemini-3.6-Flash-A (`3e83b35d-d40d-4047-b76e-5966571fad77`)  
> **Target Issue:** ORQ-48 (`8e225d8a-cbdb-460d-87a8-39d551261151`)  
> **Timestamp (UTC):** 2026-07-29T16:36:31Z  
> **Mandate Reference:** `[GTL-DISPATCH-RECORDER-20260729]` (`e874800c-19ef-4c19-86f3-17b7083a1b52`)  
> **Authority & Governance:** Continuous live inventory, project-scoped board counts, GTL decision tracking, line-item evidence SHA linking, and Kanban dispatch governance. Zero direct product code mutations.

---

## 1. Fresh Project-Scoped Board Counts & Active Tasks

**DB Snapshot Timestamp:** 2026-07-29T16:36:31Z

### A. Primary Project: `4b0ef49b-df06-4e83-9a29-8a23b34821d4` (ORQ2 — Pendências de teste, deploy e correção)
* **Total Project Cards:** **46**
* **`done` (29 cards):** ORQ-12, ORQ-15, ORQ-16, ORQ-17, ORQ-18, ORQ-19, ORQ-20, ORQ-21, ORQ-22, ORQ-24, ORQ-25, ORQ-26, ORQ-30, ORQ-31, ORQ-33, ORQ-34, ORQ-36, ORQ-46, ORQ-49, ORQ-51, ORQ-52, ORQ-55, ORQ-56, ORQ-57, ORQ-58, ORQ-59, ORQ-61, ORQ-65, ORQ-70
* **`in_review` (11 cards):** ORQ-13, ORQ-14, ORQ-35, ORQ-39, ORQ-50, ORQ-54, ORQ-60, ORQ-62, ORQ-67, ORQ-68, ORQ-69
* **`in_progress` (2 cards):** ORQ-23 (Opus48-B), ORQ-66 (Opus48-A)
* **`blocked` (2 cards):** ORQ-37, ORQ-64 (Human-only blocker)
* **`backlog` (1 card):** ORQ-53
* **`todo` (1 card):** ORQ-63

### B. Non-Project / Other Workspace Cards
* **Total Non-Project Cards:** **14**
* **`done` (5 cards):** ORQ-11, ORQ-27, ORQ-28, ORQ-29, ORQ-38
* **`in_review` (0 cards)**
* **`in_progress` (3 cards):** ORQ-41 (Opus48-B), ORQ-43 (Codex-A), ORQ-48 (Gemini-3.6-Flash-A)
* **`blocked` (3 cards):** ORQ-40, ORQ-42 (Human-only blocker), ORQ-44 (Human-only blocker)
* **`todo` (1 card):** ORQ-47
* **`cancelled` (2 cards):** ORQ-32, ORQ-45

### C. Workspace Overall Total
* **Total Workspace Issues:** **60 cards**
  * `done`: 34
  * `in_review`: 11
  * `in_progress`: 5
  * `blocked`: 5
  * `cancelled`: 2
  * `todo`: 2
  * `backlog`: 1

### Active Task Allocations & Coordinator Map
* **ORQ-48 (Coordinator):** Gemini-3.6-Flash-A (`3e83b35d-d40d-4047-b76e-5966571fad77`) — Continuous program recorder & fleet governance
* **ORQ-66 (Durable Daemon):** Opus48-A (`2c042fdd-9a76-4da1-9b02-c2332a736a86`) — Combining AGY token-only with ORQ2 daemon
* **ORQ-68 (Scheduler Reconciliation):** WebQA (`30bc4405-646f-4fdc-b8d7-7f9175373809`) — In review (`ed20ecebae710680114250dd7e12635d7cdb4af1`)
* **ORQ-23 (P0 Rollback Safety):** Opus48-B (`4069a041-9c68-416a-b0cf-52226c076c6c`) — Rollback & roll-forward safety proof
* **ORQ-43 (Daemon Token Lifecycle):** Codex-A (`4dc3b1f5-451f-4f68-b71e-91e856fed286`) — Token lifecycle & rotation

---

## 2. Explicit Human-Only Blockers Preserved

* **ORQ-42 (`blocked`):** Controlled rotation of `JWT_SECRET`. Requires owner-bounded rotation window, secret resolution contract, and live cutover gate.
* **ORQ-44 (`blocked`):** Lifecycle and rotation of OmniRoute gateway inference key. Requires owner-approved rotation window and secret-safe execution path.
* **ORQ-64 (`blocked`):** Security Incident (Codex Credential Printed). Requires human revocation of compromised session/credential at provider, re-authentication of isolated slot 170 (ORQ-63), and log purge verification.

---

## 3. GTL Accept / Reject / Reassignment Matrix

| Issue | Title | Status | Assignee | GTL Verdict | Branch / Evidence SHA | Lineage & State Classification | Key GTL Findings / Mandatory Requirements |
|---|---|---|---|---|---|---|---|
| **ORQ-13** | P0 Usage Cost | `in_review` | `7efc68e4` | **STRUCTURALLY ACCEPTED** | Branch: `integration/orq13-canonical-cost-stack-20260729`<br>Commit: `f8bb3406867c97e26601e2cac09f26160f89ab9c` | **Topic-Branch (Ready for Deploy)** | Structurally accepted; contains tier/effectiveAt pricing delta. Pending ORQ-14 telemetry artifact and additive integration onto 112e8dada. |
| **ORQ-14** | P0 Usage Telemetry | `in_review` | `7efc68e4` | **CHANGES REQUIRED** | No commit pushed | **Topic-Branch (In Review)** | Pending UTC-dated content-free AGY/Kiro protocol artifact commit with version/payload linkage before closure. |
| **ORQ-23** | P0 Rollback | `in_progress` | Opus48-B (`4069a041`) | **REJECTED / REASSIGNED** | Branch: `agent/agy-p0-a7/5af50410`<br>Commit: `29bd430` | **Topic-Branch (In Progress)** | In progress under Opus48-B for atomic roll-forward/rollback safety proof and TOCTOU guard fixes. |
| **ORQ-35** | P0 PostgreSQL Hardening | `in_review` | Agy-P0-A8 (`fba27666`) | **CHANGES REQUIRED** | Branch: `agent/agy-p0-a8/e0f71f2f`<br>Commit: `cde2e0898b18065658340d14bc3a71c3e354dcd2` | **Topic-Branch (In Review)** | Structure correct. Requires PG17 no-skip peer-map evidence and non-argv/non-plaintext auth runbook updates. |
| **ORQ-54** | P0 Chat Escape Hatch | `in_review` | `7efc68e4` | **REJECTED BY GTL** | Branch: `agent/opus48-a/orq54-escape-hatch`<br>Commit: `9a844883770445...` | **Topic-Branch (In Review)** | Session concurrency flaw (`ClaimAgentTask` serializes by agent+session). Pending strict `chat_session_id` serialization refactor. |
| **ORQ-57** | P0 Deploy Safety | `done` | `7efc68e4` | **ACCEPTED DONE** | Branch: `agent/codex-b/3bac1623`<br>Commit: `3a1f41a` | **Canonical-Main (Integrated)** | Mandatory env-file preflight, fail-closed recreate, and lock guards integrated and verified. |
| **ORQ-58** | P0 Canonical Rebuild | `done` | `7efc68e4` | **ACCEPTED DONE** | Production Deployed<br>Revision: `112e8dada455b4e7a3400e63728e00e6e3a0aa27` | **Canonical Production Deployed** | Rebuild deployed. Live image `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`. Rollback image `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`. |
| **ORQ-59** | P0 GSD Wave 3 | `done` | `7efc68e4` | **ACCEPTED DONE** | Branch: `.planning/`<br>Commit: `192993a5d2d0fbd193a83be2aca5fc74115558e1` | **Canonical-Main (Integrated)** | GSD Wave 3 rebaseline accepted after consuming canonical ORQ-62 OpenSpec SHA. |
| **ORQ-60** | P0 PostgreSQL Least Privilege | `in_review` | `unassigned` | **REJECTED BY GTL (R2)** | Branch: `agent/agy-p0-a8/e0f71f2f`<br>Commit: `9737ed396a2265d5ea68f56e18ec2b6df16363f0` | **Topic-Branch (In Review)** | Unsafe `LOGIN SUPERUSER PASSWORD NULL` state. Requires peer map coordination with ORQ-35 and isolated DML assertion tests. |
| **ORQ-61** | P0 OpenSpec Integrity | `done` | `7efc68e4` | **ACCEPTED DONE** | Branch: `agent/agy-p0-a7/58d6df14`<br>Commit: `192993a5d2d0fbd193a83be2aca5fc74115558e1` | **Canonical-Main (Integrated)** | OpenSpec integrity repair accepted. Bounded repair for rotation-router archive references verified. |
| **ORQ-62** | P0 OpenSpec Full Lineage | `in_review` | `7efc68e4` | **IN REVIEW** | Branch: `agent/agy-p0-a7/eb2c0b57`<br>Commit: `7618599f29d43e964a485ab12a9932a9fd037e1f` | **Topic-Branch (In Review)** | Restored runbooks (242/130 lines), content-parity matrix, and strict 5/5 OpenSpec validation submitted. |
| **ORQ-63** | P0 Capacity Slot 170 | `todo` | `8d9da3ab` | **TODO / REGISTERED** | None | **Kanban Todo** | Register isolated Codex slot 170 for capacity preparation. |
| **ORQ-65** | P0 Runtime Regression | `done` | `7efc68e4` | **ACCEPTED DONE** | Log path restored | **Canonical-Main (Integrated)** | Restored AGY token-only log path. |
| **ORQ-66** | P0 Durable Daemon | `in_progress` | Opus48-A (`2c042fdd`) | **IN PROGRESS** | Branch: `agent/opus48-a/orq66-durable-daemon` | **Topic-Branch (In Progress)** | Combining AGY Token-Only with ORQ2 daemon. In progress under Opus48-A. |
| **ORQ-67** | P0 OpenSpec Steward | `in_review` | `7efc68e4` | **IN REVIEW** | Branch: `agent/agy-p0-a8/orq67-steward`<br>Commit: `ea501879df4d0b12700734e018b0457ce102b1f5` | **Topic-Branch (In Review)** | Re-audit submitted; awaiting final GTL verification. |
| **ORQ-68** | P0 Scheduler Reconcile | `in_review` | WebQA (`30bc4405`) | **CHANGES SUBMITTED** | Branch: `agent/codex-b/92870148`<br>Commit: `ed20ecebae710680114250dd7e12635d7cdb4af1` | **Topic-Branch (In Review)** | Patch restricts auto-cancel to explicit `cancelled` status and maintains active tasks in `done`/`in_review`. Awaiting GTL review. |
| **ORQ-69** | P0 Temp Recovery | `in_review` | `7efc68e4` | **IN REVIEW** | Authenticated API clear | **Live Config State** | Cleared `thinking_level` to NULL for Opus48-A, Codex-B, Codex-C; restore-high obligation pending durable daemon deploy. |
| **ORQ-70** | P0 Post-Deploy Canary | `done` | `7efc68e4` | **ACCEPTED DONE** | Task: `ee480900-f29c-4745-88cf-947f99466ca8` | **Canonical Production Verified** | Live canary returned `ORQ58_CANARY_OK` from Opus48-A (`claude-opus-5`). Terminal proof for revision `112e8dada...`. |

---

## 4. Code Lineage & Deployment State Distinctions

* **Canonical Production Tip:** Revision `112e8dada455b4e7a3400e63728e00e6e3a0aa27` (Deployed via ORQ-58, verified via ORQ-70 canary).
* **Live Production Container Image:** `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`
* **Durable Rollback Overlay Image:** `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6` (Overlay `8227241`)
* **Base Repository Branch:** `origin/main` (`b657129`).
* **Active Topic-Branches:**
  * `agent/codex-b/92870148` (ORQ-68 @ `ed20ecebae710680114250dd7e12635d7cdb4af1`)
  * `agent/agy-p0-a7/eb2c0b57` (ORQ-62 @ `7618599f29d43e964a485ab12a9932a9fd037e1f`)
  * `integration/orq13-canonical-cost-stack-20260729` (ORQ-13 @ `f8bb3406867c97e26601e2cac09f26160f89ab9c`)
  * `agent/agy-p0-a8/e0f71f2f` (ORQ-60 @ `9737ed396a2265d5ea68f56e18ec2b6df16363f0`)
  * `agent/opus48-a/orq54-escape-hatch` (ORQ-54 @ `9a844883770445...`)

---

## 5. Control Governance Rules Enforced

1. **Kanban-Only Governance:** All task assignments, status tracking, and reviews are managed exclusively via `multica issue` CLI commands. Herdr and direct pane prompts remain strictly prohibited.
2. **GTL Acceptance Gate:** No card is marked `done` without explicit GTL review acceptance.
3. **OpenSpec Residual Mapping:** OpenSpec change sets map 1-to-1 to existing cards (ORQ-13, 14, 18, 23, 44). Zero duplicate cards created.
4. **No Coordinator Code Mutation:** ORQ-48 acts strictly as documentation recorder and fleet coordinator. Product codebase (`multica-auth-work/`) remains 100% untouched.
