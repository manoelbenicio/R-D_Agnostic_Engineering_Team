# ORQ-23 Phase 3 Accounting & Rollback Design — Adversarial Review Evidence

- **Reviewer:** Antigravity (Opus-46-B)
- **Target Document:** `orq23-phase3-accounting-rollback-design.md` (`wN:p1`)
- **Base Commit / Stacks:** ORQ-21 (`feccee4`), ORQ-12 (`dd95e0a`)
- **UTC:** 2026-07-28T16:26Z
- **Verdict:** ⛔ **BLOCK** (Requires design correction prior to execution)

---

## Executive Summary

An independent, read-only adversarial review of `orq23-phase3-accounting-rollback-design.md` was conducted against the live codebase (`multica-auth-work`), authoritative SQL query schemas (`pkg/db/queries/task_usage.sql`), staged/applied migrations (000084, 000101, 000102, 000128), and the ORQ-12 producing account snapshot contract.

While the document correctly identifies the live daemon service (`multica-daemon-orq2-credential.service`) and the exact known-green binary SHA-256 hash (`88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8`), it contains **critical contract violations and invented schema fields** that break the ORQ-12 architecture and would cause SQL/Go compilation failures.

---

## Detailed Audit & Defect Breakdown

### 1. 🚨 Contract Violation: Wire Payload vs Server-Side Snapshot (ORQ-12)
- **Design Claim (Delta 1.1)**: Claims the daemon wire payload (`TaskUsageEntry`) must include `account_id` (UUID from claim).
- **Actual Implementation (`pkg/db/queries/task_usage.sql:9-14`)**:
  ```sql
  -- account_id is COPIED from the task, never accepted from the caller and never
  -- re-resolved here. agent_task_queue.credential_account_id is frozen at
  -- claim/dispatch (see ClaimAgentTask), so this row records the account that
  -- ACTUALLY produced the tokens even if the agent has rotated since.
  ```
- **Defect**: In ORQ-12 (`dd95e0a`), `UpsertTaskUsage` populates `account_id` server-side via subquery `(SELECT q.credential_account_id FROM agent_task_queue q WHERE q.id = $1)`. Forcing the daemon to transmit `account_id` over the wire bypasses server-side immutability, introduces wire security risks (a compromised daemon could forge spend attribution), and breaks the ORQ-12 contract.

---

### 2. ❌ Invented Schema Columns & Mismatched Names
- **Design Claim (Delta 1.1 & 1.2)**: Claims `task_usage` table and `UpsertTaskUsage` query take `reasoning_tier` and `cost_cents`.
- **Actual PostgreSQL Schema**:
  - Column in DB is **`thinking_level`** (nullable `VARCHAR`), NOT `reasoning_tier`.
  - Column **`cost_cents` does NOT exist** on the `task_usage` table. `task_usage` records raw token receipts (`input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`). Cost calculation is performed dynamically or aggregated in `task_usage_hourly` / pricing tier tables (ORQ-13/ORQ-14).
  - Parameter names in Go are `input_tokens` / `output_tokens`, NOT `tokens_input` / `tokens_output`.

---

### 3. ⚠️ Invalid E2E Gate Assertions (E2E-2)
- **Design Claim (Gate E2E-2)**: Asserts `task_usage` table contains `cost_cents` column as a pass criterion.
- **Defect**: Gate E2E-2 will unconditionally fail on any valid Multica database schema because `cost_cents` is intentionally decoupled from raw task usage to allow retroactive pricing updates without mutating immutable token counts.

---

## Mandatory Corrections Required Before Unblocking

1. **Fix Wire Payload & Query Parameters**:
   - Remove `account_id` from daemon `TaskUsageEntry` wire payload; preserve server-side subquery lookup from `agent_task_queue.credential_account_id`.
   - Rename `reasoning_tier` -> `thinking_level`.
   - Remove `cost_cents` from raw `task_usage` table expectations; keep `cost_cents` in downstream financial aggregation views / rollups (ORQ-13 / ORQ-14).
2. **Align E2E-2 Pass Criteria**:
   - Update E2E-2 gate assertion to verify `task_usage` columns: `account_id`, `thinking_level`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`.

---

## Verdict Summary

- **Verdict:** ⛔ **BLOCK**
- **Action Required:** Update `orq23-phase3-accounting-rollback-design.md` to fix the ORQ-12 wire contract violation and remove invented database column names (`reasoning_tier`, `cost_cents` on `task_usage`).
