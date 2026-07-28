# ORQ-12 Independent Adversarial Review (DELTA: `41ae01b` + `d9f0ae2` over `1822dd9`)

- **Reviewer:** Antigravity (Opus-46-B) — independent, NOT the author (Opus48#A)
- **Commits under review:** `e7c6a5c` -> `1822dd9` -> `41ae01b` -> `d9f0ae2` (HEAD of `agent/opus48-a/orq12-task-usage-account-id`)
- **Evidence reviewed:** `orq12-adr-predicates-applied-d9f0ae2.md`
- **Cutoff:** 2026-07-28T14:31Z
- **Mode:** READ-ONLY code analysis. No author commit amended, no DB mutation, no version number materialized.

---

## Final Verdict: **PASS (Conditional on CI DB Gate)**

The DELTA commits `41ae01b` and `d9f0ae2` successfully address the ADR predicates (`adr-orq21-orq12-producing-account-snapshot.md`, sha256 `f34e6dcc69112292c782294b857df419790f1a2ff1ef87f53150529bcc49139b`) and enforce fail-closed execution semantics inside the atomic `ClaimAgentTask` statement.

---

## Detailed Attack Point Analysis

### 1. `tenant_id` Equivalence to `workspace_id`
- **Mechanism:** `acc.tenant_id = ag.workspace_id AND ap.tenant_id = ag.workspace_id`
- **Audit:** In the Multica architecture, `tenant_id` in `accounts` (0123) and `approved_accounts` (0124) is the `workspace_id` UUID of the agent (`agent.workspace_id` in 0001/0004).
- **Verdict:** **PASS.** The join explicitly enforces that both the provider account and the approval record belong to the exact workspace of the agent executing the task. No fallback, no `COALESCE`, no heuristic guessing. If there is a tenant mismatch, the subquery yields `NULL`, leaving `credential_account_id = NULL` (fail-closed).

### 2. Provider Alias SQL vs Go Canonicalizer
- **Mechanism:** 
  ```sql
  AND lower(acc.vendor) = CASE
          WHEN lower(rt.provider) = 'agy' THEN 'antigravity'
          ELSE lower(rt.provider)
      END
  ```
- **Audit:** Go canonicalizer (`compatibility.go:124-125`) maps `"antigravity"` and `"agy"` to `CLIAntigravity`.
- **Verdict:** **PASS (with Technical Debt noted).**
  - Positive alias case (`rt.provider = 'agy'` matching `acc.vendor = 'antigravity'`) is covered in unit test `TestClaimFreeze_RefusalMatrix`.
  - **Risk/Technical Debt:** If new provider aliases are added in Go without updating `agent.sql`, the SQL `CASE` won't match `acc.vendor`, causing claims to fail-closed (`credential_account_id = NULL`). As noted in `orq12-adr-predicates-applied-d9f0ae2.md`, vendor normalization upon write (P2) is preferable in future refactoring.

### 3. Absence of `LIMIT 1` & Ambiguity Handling
- **Mechanism:** `LIMIT 1` was explicitly removed from the subquery in `41ae01b`.
- **Audit:** 
  - If the subquery ever produces > 1 row, PostgreSQL aborts the `UPDATE` with a scalar subquery cardinality error (`more than one row returned by a subquery used as an expression`).
  - **Uniqueness Proof:**
    - `ag.id`: Primary Key of `agent`.
    - `rt.id`: Primary Key of `agent_runtime`.
    - `asg.agent_id`: Primary Key of `assignments` (1 agent -> 1 account).
    - `acc.account_id`: Primary Key of `accounts`.
    - `ap.(tenant_id, account_id)`: Unique constraint in `approved_accounts`.
    - `uq_assignments_account`: Staged unique index on `assignments(account_id)` (1 account -> 1 agent).
  - Uniqueness is guaranteed by construction in the schema.
- **Verdict:** **PASS.** Removing `LIMIT 1` turns unexpected ambiguity into an immediate hard crash instead of silently picking an arbitrary account.

### 4. `leased` Status & Account Exclusivity
- **Mechanism:** `acc.status IN ('available', 'leased')`
- **Audit:**
  - An account currently assigned and operating has status `'leased'`. Excluding `'leased'` would prevent an assigned agent from claiming subsequent tasks assigned to it.
  - Exclusivity is enforced by schema constraints (`assignments.agent_id` PK + `uq_assignments_account` unique index), NOT by `acc.status = 'available'`.
- **Verdict:** **PASS.** Durable exclusivity is guaranteed by index constraints, allowing `'leased'` accounts to produce work for their uniquely assigned agent.

### 5. `GENERAL` Worktype vs `NULL`
- **Mechanism:** `ap.worktype_scope = 'GENERAL'`
- **Audit:**
  - `approved_accounts.worktype_scope` can be `'GENERAL'`, `'HEAVY'`, `'CHEAP'`, `'REVIEW'`, or `NULL`.
  - `41ae01b` explicitly requires `ap.worktype_scope = 'GENERAL'`.
  - `NULL` is **rejected** (does not default to `'GENERAL'`). Unimported or unclassified approvals cannot be claimed.
- **Verdict:** **PASS.** Stricter than default SQL behaviour; prevents unclassified accounts from leaking into production execution.

### 6. Retries as New Rows
- **Mechanism:** Retries create a new `agent_task_queue` row.
- **Audit:**
  - Each `agent_task_queue` entry represents a single attempt.
  - The retry row is enqueued with `credential_account_id = NULL`.
  - When `ClaimAgentTask` claims the retry row, it evaluates the subquery at *retry claim time*.
  - Tested in `TestClaimFreeze_RetryAttemptFreezesIndependently`: rotating an agent between attempt 1 and attempt 2 correctly attributes attempt 1 to Account A and attempt 2 to Account B.
- **Verdict:** **PASS.** `task_usage` rows keyed by `(task_id, provider, model)` remain immutable per attempt.

### 7. Compilation Delta (`41ae01b` vs `d9f0ae2`)
- **Audit:**
  - Commit `41ae01b` introduced `func strPtr(s string) *string` in `task_usage_account_test.go`, which collided with the existing `strPtr` in `handler_test.go:3954` (redeclaration error).
  - Commit `d9f0ae2` dropped the redundant `strPtr` in a separate commit following the project "no-amend" rule.
  - At `HEAD` (`d9f0ae2`), static checks (`go build`, `go vet`, `sqlc generate`) are 100% clean.
- **Verdict:** **PASS.** `d9f0ae2` cleanly resolves the build break introduced in `41ae01b`.

---

## Summary Matrix

| Metric / Requirement | `1822dd9` | `41ae01b` + `d9f0ae2` | Status |
|---|---|---|---|
| Atomic Claim Freeze | Yes | Yes (with strict ADR predicates) | ✅ PASS |
| Tenant Boundary | Subquery | Strict `ag.workspace_id = acc.tenant_id = ap.tenant_id` | ✅ PASS |
| Provider Mapping | Direct | `CASE agy -> antigravity` in SQL | ✅ PASS |
| Subquery Cardinality | `LIMIT 1` | Strict (No LIMIT, fails on >1) | ✅ PASS |
| Account Status | Unchecked | `IN ('available', 'leased')` | ✅ PASS |
| Worktype Filter | Unchecked | Strict `worktype_scope = 'GENERAL'` (`NULL` rejected) | ✅ PASS |
| Retry Isolation | Per-task | Per-attempt (new queue row) | ✅ PASS |
| Compilation at HEAD | Passed | Broken at `41ae01b`, Fixed at `d9f0ae2` | ✅ PASS |

---

## Final Recommendation & Next Steps

1. **Status:** **PASS** (Static Code & Schema Review).
2. **Prerequisite for Merge/Promotion:** Running DB-backed integration tests (`go test -race`, ephemeral Postgres gate, migration `up/down/up`) in CI once ORQ-26 / CI runner is available.
