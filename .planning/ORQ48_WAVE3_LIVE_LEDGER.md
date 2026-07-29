# ORQ-48 Fleet Documentation & Kanban Control — Wave 3 Live Ledger

> **Recorder / Coordinator:** Gemini-3.6-Flash-A (`3e83b35d-d40d-4047-b76e-5966571fad77`)  
> **Target Issue:** ORQ-48 (`8e225d8a-cbdb-460d-87a8-39d551261151`)  
> **Timestamp (UTC):** 2026-07-29T14:52:24Z  
> **Authority Mandate:** Continuous live inventory, project-scoped board counts, GTL decision tracking, line-item evidence SHA linking, and Kanban dispatch governance. No direct product code mutations.

---

## 1. Live Project-Scoped Board Counts & Active Tasks

**DB Snapshot Timestamp:** 2026-07-29T14:52:24Z

### A. Primary Project: `4b0ef49b-df06-4e83-9a29-8a23b34821d4` (ORQ2 — Pendências de teste, deploy e correção)
* **Total Project Cards:** **38**
* **`done` (21 cards):** ORQ-12, ORQ-15, ORQ-16, ORQ-17, ORQ-18, ORQ-19, ORQ-20, ORQ-21, ORQ-22, ORQ-24, ORQ-25, ORQ-30, ORQ-31, ORQ-34, ORQ-36, ORQ-46, ORQ-49, ORQ-51, ORQ-52, ORQ-55, ORQ-56
* **`in_review` (9 cards):** ORQ-23, ORQ-26, ORQ-33, ORQ-35, ORQ-50, ORQ-54, ORQ-59, ORQ-61, ORQ-62
* **`in_progress` (3 cards):** ORQ-13, ORQ-57, ORQ-60
* **`blocked` (3 cards):** ORQ-14, ORQ-37, ORQ-39
* **`backlog` (2 cards):** ORQ-53, ORQ-58
* **`todo` (0 cards)**
* **`cancelled` (0 cards)**

### B. Workspace Non-Project / Other Workspace Issues
* **Total Non-Project Cards:** **14**
* **`done` (3 cards):** ORQ-27, ORQ-28, ORQ-29
* **`in_review` (2 cards):** ORQ-11, ORQ-38
* **`in_progress` (2 cards):** ORQ-41, ORQ-48
* **`blocked` (4 cards):** ORQ-40, ORQ-42, ORQ-43, ORQ-44
* **`todo` (1 card):** ORQ-47
* **`cancelled` (2 cards):** ORQ-32, ORQ-45

### C. Workspace Overall Total
* **Total Workspace Issues:** **52 cards**
  * `done`: 24
  * `in_review`: 11
  * `in_progress`: 5
  * `blocked`: 7
  * `cancelled`: 2
  * `todo`: 1
  * `backlog`: 2

### Active Task Allocations & Coordinator Map
* **ORQ-48 (Coordinator):** Gemini-3.6-Flash-A (`3e83b35d-d40d-4047-b76e-5966571fad77`) — Live recorder & fleet governance
* **ORQ-13 (Usage Cost):** Opus48-A (`2c042fdd-9a76-4da1-9b02-c2332a736a86`) — Canonical integration branch prepared
* **ORQ-57 (Deploy Safety):** WebQA (`30bc4405-646f-4fdc-b8d7-7f9175373809`) — Active task SHA `3a1f41a`
* **ORQ-60 (PG Least Privilege):** Agy-P0-A8 (`fba27666-72d2-4597-9ca3-107459f27b65`) — Reassigned from revoked Codex-C
* **ORQ-41 (Decouple Metadata):** Opus48-B (`4069a041-9c68-416a-b0cf-52226c076c6c`) — Metadata decoupling review

---

## 2. GTL Accept / Reject / Reassignment Matrix

| Issue | Title | Status | Assignee | GTL Verdict | Branch / Evidence SHA | Lineage & State Classification | Key GTL Findings / Mandatory Requirements |
|---|---|---|---|---|---|---|---|
| **ORQ-13** | P0 Usage Cost | `in_progress` | Opus48-A (`2c042fdd`) | **Reconciled / Handoff** | Branch: `canonical-integration`<br>Commit: `eb3f0dd` | **Topic-Branch (Ready for Canary)** | Integration branch ready (`eb3f0dd` + frozen account/thinking snapshot stack). Blocked on external operator deployment permissions (Docker/AWS/GitHub). |
| **ORQ-23** | P0 Rollback | `in_review` | Agy-P0-A8 (`5b336637`) | **REJECTED FOR DONE** | Branch: `agent/agy-p0-a7/5af50410`<br>Commit: `29bd430` | **Topic-Branch (In Review)** | Non-atomic rollback state management, lack of roll-forward capability under failure, incomplete TOCTOU guard on session teardown. |
| **ORQ-54** | P0 Chat Escape Hatch | `in_review` | `unassigned` | **REJECTED BY GTL** | Branch: `agent/opus48-a/orq54-escape-hatch`<br>Commit: `9a844883770445...` | **Topic-Branch (In Review)** | Session concurrency flaw: `ClaimAgentTask` serializes by `(agent_id, chat_session_id)` allowing multi-agent interleaving and owner session pointer overwrites. Must serialize strictly by `chat_session_id`. |
| **ORQ-57** | P0 Deploy Safety | `in_progress` | WebQA (`30bc4405`) | **UNDER FINAL LOCK PROOF** | Branch: `agent/codex-b/3bac1623`<br>Commit: `3a1f41a` | **Topic-Branch (In Progress)** | 8/8 tests pass. Blocker: `LOCK TABLE agent_task_queue IN SHARE MODE` conflicts with queue writers during restart. Must prove non-deadlocking admission freeze + fsync on `last-known-good.yml` before declared dry-run on ORQ1. |
| **ORQ-59** | P0 GSD Wave 3 | `in_review` | Agy-P0-A7 (`780104f1`) | **REJECTED BY GTL** | Branch: `agent/agy-p0-a7/58d6df14`<br>Commit: `192993a5d2d0fbd193a83be2aca5fc74115558e1` | **Topic-Branch (In Review)** | Propagated nonexistent OpenSpec SHA (`69880b98e1ae...`), labeled ORQ-62 canonical prematurely, cited stale 52-card count without project scoping. Must consume final canonical SHA from ORQ-62. |
| **ORQ-60** | P0 PostgreSQL Least Privilege | `in_progress` | Agy-P0-A8 (`fba27666`) | **REJECTED BY GTL (R2)** | Branch: `agent/agy-p0-a8/e0f71f2f`<br>Commit: `9737ed396a2265d5ea68f56e18ec2b6df16363f0` | **Topic-Branch (In Progress)** | Unsafe `LOGIN SUPERUSER PASSWORD NULL` state reachable via local HBA trust. Codex-C execution failed due to revoked refresh token (`provider_auth_or_access`). Reassigned through Kanban to Agy-P0-A8. |
| **ORQ-61** | P0 OpenSpec Integrity | `in_review` | `unassigned` | **REJECTED BY GTL** | Branch: `agent/agy-p0-a7/58d6df14`<br>Commit: `192993a5d2d0fbd193a83be2aca5fc74115558e1` | **Topic-Branch (In Review)** | Propagated nonexistent OpenSpec SHA (`69880b98e1ae...`) and labeled ORQ-62 canonical prematurely. Waiting on ORQ-62 correction. |
| **ORQ-62** | P0 OpenSpec Full-Lineage Reconciliation | `in_review` | Agy-P0-A7 (`780104f1`) | **REJECTED (Init) / Adjustments Applied** | Branch: `agent/agy-p0-a7/eb2c0b57`<br>Init SHA: `69880b9bc7bfd4cd...`<br>Updated SHA: `7618599f29d43e964a485ab12a9932a9fd037e1f` | **Topic-Branch (In Review)** | Initial rejection: wrong 40-char SHA cited, loss of 225 runbook lines in `prod-rollout-runbook.md` and 122 in `rollback-runbook.md`. Agent Agy-P0-A7 pushed updated SHA `7618599f...` with restored runbooks and content-parity matrix. |

---

## 3. Code Lineage & Deployment State Distinctions

* **Canonical-Main (`origin/main` @ `b657129`):** Base product code repository. Tag `dev-freeze-20260719-main`.
* **Topic-Branches (In-Review / In-Progress):**
  * `agent/codex-b/3bac1623` (ORQ-57 @ `3a1f41a`): Deploy safety & mandatory env-file preflight
  * `agent/agy-p0-a7/eb2c0b57` (ORQ-62 @ `7618599f29d43e964a485ab12a9932a9fd037e1f`): OpenSpec full-lineage reconciliation
  * `agent/agy-p0-a7/58d6df14` (ORQ-59/61 @ `192993a5d2d0fbd193a83be2aca5fc74115558e1`): GSD rebaseline & OpenSpec integrity
  * `agent/opus48-a/orq54-escape-hatch` (ORQ-54 @ `9a844883770445...`): Chat escape hatch
  * `agent/agy-p0-a8/e0f71f2f` (ORQ-60 @ `9737ed396a2265d5ea68f56e18ec2b6df16363f0`): PostgreSQL least privilege
  * `agent/agy-p0-a7/5af50410` (ORQ-23 @ `29bd430`): Independent rollback safety review
* **Production-Overlay:** Live production overlay running on `8227241`. Direct product deployments remain blocked until ORQ-57 deploy safety verification and canonical rebuild sequence (ORQ-58).

---

## 4. Control Governance Rules Enforced

1. **Kanban-Only Dispatch:** All task assignments, state changes, and reviews are managed exclusively through `multica issue` CLI commands. Herdr and direct pane prompts are strictly forbidden.
2. **GTL Acceptance Gate:** No card is transitioned to `done` without explicit GTL review acceptance.
3. **No Duplicate Residual Cards:** OpenSpec change sets (`build-omniroute-agent-brain`, `native-runtimes-onboarding`, `chat-orchestration-standard`, `credential-account-home-restoration`) map 1-to-1 to existing cards (ORQ-13, 14, 18, 23, 44). Zero duplicate cards created.
4. **No Code Mutation by Coordinator:** ORQ-48 acts strictly as documentation recorder and fleet coordinator. Product codebase (`multica-auth-work/`) remains untouched.
