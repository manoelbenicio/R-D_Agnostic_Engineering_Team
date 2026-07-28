# ORQ-12 GTM Final Contract Alignment Evidence (`d93ad9f`)

- **Author / Owner:** Antigravity (Opus-46-B)
- **Worktree:** `/home/ec2-user/workspace/worktrees/orq12-account-id`
- **Branch:** `agent/opus48-a/orq12-task-usage-account-id`
- **HEAD Commit:** `d93ad9f` (`fix(cost): exact equality claim matching and write/migration provider canonicalization per GTM ruling`)
- **Predecessor Commit:** `de38d3d`
- **Cutoff:** 2026-07-28T15:22Z
- **Mode:** Full source ownership. No commit amended (`git commit --amend` avoided), no ORQ-21 worktree touched.

---

## 1. Compliance with Final GTM Rulings

### 1a. Removal of Read-Time CASE / lower / btrim Authority (`agent.sql`)
- **`ClaimAgentTask`**: Replaced read-time `CASE` provider canonicalization with **exact stored equality comparison**:
  ```sql
  AND acc.vendor = rt.provider
  AND rt.provider IN ('codex', 'kiro', 'antigravity')
  ```
- **`ReclaimStaleDispatchedTaskForRuntime`**: Replaced read-time `CASE` expression with exact covered list check:
  ```sql
  AND NOT (
      rt.provider IN ('codex', 'kiro', 'antigravity')
      AND atq.credential_account_id IS NULL
  )
  ```
- Reclaim **never resolves or changes** an account snapshot. If a covered provider task reaches `dispatched` state with `credential_account_id IS NULL`, reclaim fails closed.

### 1b. Write Path & Migration 128 Canonicalization
- **`daemon.go` (`normalizeProvider`)**: Normalized provider write paths so `agy` input alias maps to `antigravity` before writing `agent_runtime.provider` to PostgreSQL.
- **Migration 128 (`NEXT_CANONICAL_task_usage_account_id.up.sql`)**:
  - Preflight backfill updates legacy `agy` provider/vendor rows in `accounts` and `agent_runtime` to `'antigravity'`.
  - Normalizes case/padding for existing `'codex'`, `'kiro'`, `'antigravity'` records.
  - Added a preflight **DO block** that verifies all vendor/provider rows for covered providers are stored in canonical form (`'codex'`, `'kiro'`, `'antigravity'`) and fails closed (aborts migration) if unmapped alias drift remains.

---

## 2. Verification Results

| Check | Command | Result |
|---|---|---|
| **sqlc code generation** | `sqlc -f sqlc.staged-orq12.yaml generate` | ✅ PASS (clean, schema-validated) |
| **Go compilation** | `go test -c -o /dev/null ./internal/handler` | ✅ PASS |
| **Go vet** | `go vet ./internal/handler ./pkg/db/...` | ✅ PASS (0 warnings) |
| **Whitespace & Formatting** | `git diff --check` && `gofmt -l` | ✅ PASS (0 output) |
| **DB Unit Tests (17/17)** | `go test -v ./internal/handler` | ✅ **PASS (17/17 passed, 0 failed, 0 skipped)** |
| **Race Detector** | `go test -race ./internal/handler` | ✅ **PASS (0 race warnings)** |

---

## 3. Commit Lineage

```
d93ad9f (HEAD) fix(cost): exact equality claim matching and write/migration provider canonicalization per GTM ruling
de38d3d fix(cost): enforce covered providers and fail-closed reclaim per GTM ADR
d33903f fix(cost): canonicalise the provider on both sides, as ORQ-21 R3 does
d9f0ae2 fix(test): drop the duplicated strPtr helper
41ae01b fix(cost): apply the authoritative claim predicates and drop the arbitrary LIMIT
1822dd9 fix(cost): freeze the producing account at claim, not at first usage report
e7c6a5c feat(cost): snapshot the producing account on task_usage (ORQ-12, staged)
```

---

## 4. Handoff SHA

- **Handoff SHA:** `d93ad9f`
- Ready for combined gate execution alongside `aac2227` (`agent/codex56-b/orq21-r3`).
