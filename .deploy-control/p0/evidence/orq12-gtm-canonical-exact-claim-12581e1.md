# ORQ-12 GTM Final Contract Alignment Evidence (`12581e1`)

- **Author / Owner:** Antigravity (Opus-46-B)
- **Worktree:** `/home/ec2-user/workspace/worktrees/orq12-account-id`
- **Branch:** `agent/opus48-a/orq12-task-usage-account-id`
- **HEAD Commit:** `12581e1` (`fix(cost): align test contract, fix comment schema, and enforce ID-targeted test isolation`)
- **Predecessor Commit:** `c146635`
- **Cutoff:** 2026-07-28T15:37Z
- **Mode:** Full source ownership. No commit amended (`git commit --amend` avoided), no ORQ-21 worktree touched.

---

## 1. Accomplished Updates

### 1a. Migration 128 Schema Comment Correction (`NEXT_CANONICAL_task_usage_account_id.up.sql`)
- Updated `COMMENT ON COLUMN task_usage.account_id` to accurately state:
  ```sql
  COMMENT ON COLUMN task_usage.account_id IS
      'Provider account that produced this usage, snapshotted at task claim time on agent_task_queue.credential_account_id and copied to task_usage upon report. Never live assignment lookup.';
  ```

### 1b. Deterministic ID-Targeted Test Isolation (`handler_test.go` & `task_usage_account_test.go`)
- Added `ORDER BY created_at ASC, id ASC LIMIT 1` to `handlerTestRuntimeID` to eliminate non-deterministic tie resolution when multiple runtimes exist.
- Introduced `freezeAttemptWithRuntime` helper to explicitly bind `agent` and `agent_task_queue` fixtures to specific target `runtimeID`s without modifying workspace-wide runtimes.
- Cleaned up syntax artifacts and removed diagnostic `t.Log` calls.

### 1c. Exact Unit Test Contract Compliance (`task_usage_account_test.go`)
- **`TestWritePath_NormalizeProviderAndClaim`**: Verifies write-path provider normalization (`"agy"` / trim / case -> `"antigravity"`), proving canonical stored persistence and successful exact claim (`acc.vendor = rt.provider`).
- **`TestClaimFreeze_CoveredProvidersOnly`**: Tests covered stored canonical values (`"codex"`, `"kiro"`, `"antigravity"`).

---

## 2. Verification Results

| Check | Command | Result |
|---|---|---|
| **sqlc code generation** | `sqlc -f sqlc.staged-orq12.yaml generate` | ✅ PASS (clean, schema-validated) |
| **Go compilation** | `go test -c -o /dev/null ./internal/handler` | ✅ PASS |
| **Go vet** | `go vet ./internal/handler ./pkg/db/...` | ✅ PASS (0 warnings) |
| **Whitespace & Formatting** | `git diff --check` && `gofmt -l` | ✅ PASS (0 output) |
| **DB Unit Tests (16/16)** | `go test -v ./internal/handler` | ✅ **PASS (16/16 passed, 0 failed, 0 skipped)** |
| **Race Detector** | `go test -race ./internal/handler` | ✅ **PASS (0 race warnings)** |

---

## 3. Commit Lineage

```
12581e1 (HEAD) fix(cost): align test contract, fix comment schema, and enforce ID-targeted test isolation
c146635 test(cost): align unit test contract with write-path provider normalization and exact claim
d93ad9f fix(cost): exact equality claim matching and write/migration provider canonicalization per GTM ruling
de38d3d fix(cost): enforce covered providers and fail-closed reclaim per GTM ADR
d33903f fix(cost): canonicalise the provider on both sides, as ORQ-21 R3 does
d9f0ae2 fix(test): drop the duplicated strPtr helper
41ae01b fix(cost): apply the authoritative claim predicates and drop the arbitrary LIMIT
1822dd9 fix(cost): freeze the producing account at claim, not at first usage report
e7c6a5c feat(cost): snapshot the producing account on task_usage (ORQ-12, staged)
```

---

## 4. Handoff SHA

- **Handoff SHA:** `12581e1`
- Ready for combined gate execution alongside `aac2227` (`agent/codex56-b/orq21-r3`).
