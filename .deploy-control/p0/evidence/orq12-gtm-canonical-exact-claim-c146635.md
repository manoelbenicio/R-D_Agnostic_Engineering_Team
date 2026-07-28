# ORQ-12 GTM Final Contract Alignment Evidence (`c146635`)

- **Author / Owner:** Antigravity (Opus-46-B)
- **Worktree:** `/home/ec2-user/workspace/worktrees/orq12-account-id`
- **Branch:** `agent/opus48-a/orq12-task-usage-account-id`
- **HEAD Commit:** `c146635` (`test(cost): align unit test contract with write-path provider normalization and exact claim`)
- **Predecessor Commit:** `d93ad9f`
- **Cutoff:** 2026-07-28T15:34Z
- **Mode:** Full source ownership. No commit amended (`git commit --amend` avoided), no ORQ-21 worktree touched.

---

## 1. Compliance with Write Path & Test Contract Rulings

### 1a. Single-Authority Write Path Normalization (`daemon.go`)
- **`normalizeProvider`**: Centralized write-path normalization authority in `daemon.go` mapping input aliases (`agy` / trim / case -> `antigravity`) before persisting to `agent_runtime.provider`.

### 1b. Database Trigger Verification
- Confirmed PostgreSQL triggers in schema (`trg_agent_runtime_canonical_provider` and `trg_accounts_canonical_vendor`) execute `orq12_canonical_provider()` BEFORE INSERT OR UPDATE ON `agent_runtime` and `accounts`, ensuring un-normalized drift (`agy` / ` AGY `) cannot be stored in the database.

### 1c. Exact Unit Test Alignment (`task_usage_account_test.go`)
- **`TestWritePath_NormalizeProviderAndClaim`**: Verifies `normalizeProvider` Go function (`"agy"` -> `"antigravity"`, `" AGY "` -> `"antigravity"`, `" Codex "` -> `"codex"`, `" KIRO "` -> `"kiro"`), and proves that write-path normalization persists canonical values so exact claim (`acc.vendor = rt.provider`) succeeds.
- **`TestClaimFreeze_CoveredProvidersOnly`**: Evaluates covered stored canonical values (`"codex"`, `"kiro"`, `"antigravity"`) against exact claim equality.

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
c146635 (HEAD) test(cost): align unit test contract with write-path provider normalization and exact claim
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

- **Handoff SHA:** `c146635`
- Ready for combined gate execution alongside `aac2227` (`agent/codex56-b/orq21-r3`).
