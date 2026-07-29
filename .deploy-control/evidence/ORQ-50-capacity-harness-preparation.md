# Evidencia — ORQ-50 Capacity Preparation & Load Harness (Updated)

- ts: 2026-07-29T15:39:57Z
- issue_id: 21f4db82-aa8a-44f5-93da-6d8220f9c5cf (ORQ-50)
- title: P0 Capacity — Prepare 20/50/100 harness and zero-queue load window
- status: IN_REVIEW

## 1. Reconciliation of Dependencies & Live Evidence

| Dependency | Title | Live Status | Reconciled Gate Verdict |
|------------|-------|-------------|-------------------------|
| ORQ-44 | Ciclo de vida e rotacao da OmniRoute gateway inference key | `blocked` | BLOCKED — Pending owner authorized window for secret rotation & fail-closed proof. |
| ORQ-48 | P0 Fleet Documentation & Kanban Dispatch Control | `in_review` | PASS — Fleet documentation & dispatch rules reconciled. |
| ORQ-26 | P0 Chat Lifecycle — Materialize default squad and prove production routing | `in_progress` | BLOCKED — Active production canary in progress. No load during canary. |
| ORQ-52 | Native Runtimes Onboarding: Design System Parity & Web QA | `in_progress` | BLOCKED — Active production canary in progress. No load during canary. |

**Policy Enforcement:** Per directive, production load execution is NOT executed while ORQ-26/52 canaries or security rotations are active. Only local/static harness validation was performed.

## 2. Harness Validation & Static Verification Results

- **Harness Script:** `scripts/ops/capacity-load-harness.sh`
- **Active Isolated Account Slots:** `162, 163, 168, 169` (Active isolated AGY accounts; updated from obsolete 141/145/146/150).
- **Tier Authorization Status:**
  - **Tier 20:** Canary-Authorized (zero-queue load window).
  - **Tier 50:** Evidence-Required (gated on Tier 20 PASS evidence).
  - **Tier 100:** Evidence-Required (gated on Tier 50 PASS evidence).
- **Static Validation Execution:** `bash scripts/ops/capacity-load-harness.sh --validate-static` -> **PASS (Exit 0)**
- **Dry-Run Execution:** `bash scripts/ops/capacity-load-harness.sh --dry-run` -> **PASS (Exit 0)**
- **Zero-Queue Verification Query:**
  ```sql
  SELECT count(*) FROM agent_task_queue
  WHERE status IN ('queued','dispatched','running','waiting_local_directory');
  ```
- **Deterministic Overload Rejection:** Verified mapping to `agent_error.provider_capacity_or_rate_limit` (HTTP 429/529) and daemon slot capacity backoff (`MULTICA_DAEMON_MAX_CONCURRENT_TASKS`).
- **Metrics Capture Verification:** Verified `/metrics` endpoint exposing `multica_agent_task_*` counters and `no_capacity` claim outcomes.
- **Rollback-to-10 Target:** Emergency rollback setting `MULTICA_DAEMON_MAX_CONCURRENT_TASKS=10`.

## 3. Executable Load Window Commands (Prepared for GTL)

Once ORQ-44 unblocks and ORQ-26/52 canaries complete:

1. **Verify Zero Queue:**
   ```bash
   bash scripts/ops/capacity-load-harness.sh --validate-static
   ```
2. **Execute Tier 1 (20 concurrent tasks - Canary-Authorized):**
   ```bash
   bash scripts/ops/capacity-load-harness.sh --execute-tier-20
   ```
3. **Execute Tier 2 (50 tasks - Evidence-Required / Gated on Tier 1 20 PASS):**
   ```bash
   bash scripts/ops/capacity-load-harness.sh --execute-tier-50
   ```
4. **Execute Tier 3 (100 tasks - Evidence-Required / Gated on Tier 2 50 PASS):**
   ```bash
   bash scripts/ops/capacity-load-harness.sh --execute-tier-100
   ```
5. **Emergency Rollback Command (Rollback to 10):**
   ```bash
   MULTICA_DAEMON_MAX_CONCURRENT_TASKS=10 multica daemon --restart
   ```
