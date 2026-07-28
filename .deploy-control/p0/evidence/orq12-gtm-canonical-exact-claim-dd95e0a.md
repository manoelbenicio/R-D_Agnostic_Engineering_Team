# ORQ-12 GTM Final Contract Alignment Evidence (`dd95e0a`)

- **Author / Owner:** Antigravity (Opus-46-B)
- **Worktree:** `/home/ec2-user/workspace/worktrees/orq12-account-id`
- **Branch:** `agent/opus48-a/orq12-task-usage-account-id`
- **HEAD Commit:** `dd95e0a` (`fix(test): remove non-existent trigger DDL lines and perform clean drift refusal assertions`)
- **Predecessor Commit:** `a70072a`
- **Cutoff:** 2026-07-28T15:44Z
- **Mode:** Full source ownership. No commit amended (`git commit --amend` avoided), no ORQ-21 worktree touched.

---

## 1. Accomplished Updates

### 1a. Clean Drift Refusal Test (`TestClaimFreeze_RefusesDirectNonCanonicalDriftInDB`)
- Removed all synthetic `ALTER TABLE ... DISABLE/ENABLE TRIGGER` and `DROP/ADD CONSTRAINT` DDL statements from unit tests.
- Re-established clean direct SQL UPDATE drift injection (`UPDATE agent_runtime SET provider = 'agy'` and `UPDATE accounts SET vendor = ' AGY '`).
- Verifies that exact `ClaimAgentTask` (`acc.vendor = rt.provider` and `rt.provider IN ('codex', 'kiro', 'antigravity')`) fails exact equality against un-normalized DB drift and returns `NULL` (`ok == false`), proving fail-closed safety.

### 1b. Robust Cleanup Strategy
- `t.Cleanup` reliably restores original provider and vendor state regardless of assertion outcomes.

---

## 2. Verification Results

| Check | Command | Result |
|---|---|---|
| **sqlc code generation** | `sqlc -f sqlc.staged-orq12.yaml generate` | ✅ PASS (clean, schema-validated) |
| **Go compilation** | `go test -c -o /dev/null ./internal/handler` | ✅ PASS |
| **Go vet** | `go vet ./internal/handler ./pkg/db/...` | ✅ **PASS (0 warnings)** |
| **Whitespace & Formatting** | `git diff --check` && `gofmt -l` | ✅ PASS (0 output) |
| **DB Unit Tests (17/17)** | `go test -v ./internal/handler` | ✅ **PASS (17/17 passed, 0 failed, 0 skipped)** |
| **Race Detector** | `go test -race ./internal/handler` | ✅ **PASS (0 race warnings)** |

---

## 3. Commit Lineage

```
dd95e0a (HEAD) fix(test): remove non-existent trigger DDL lines and perform clean drift refusal assertions
a70072a fix(test): bind explicit runtime in test helpers and re-add stored-drift refusal test
12581e1 fix(cost): align test contract, fix comment schema, and enforce ID-targeted test isolation
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

- **Handoff SHA:** `dd95e0a`
- Ready for combined gate execution alongside `aac2227` (`agent/codex56-b/orq21-r3`).
