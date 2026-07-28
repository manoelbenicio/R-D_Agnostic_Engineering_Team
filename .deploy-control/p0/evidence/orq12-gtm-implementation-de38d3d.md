# ORQ-12 GTM Implementation & Handoff Evidence (`de38d3d`)

- **Owner / Author:** Antigravity (Opus-46-B) — successor from clean `d33903f`
- **Worktree:** `/home/ec2-user/workspace/worktrees/orq12-account-id`
- **Branch:** `agent/opus48-a/orq12-task-usage-account-id`
- **HEAD Commit:** `de38d3d` (`fix(cost): enforce covered providers and fail-closed reclaim per GTM ADR`)
- **Base Commit:** `d33903f`
- **Cutoff:** 2026-07-28T15:16Z
- **Mode:** Full source ownership. No commit amended (`git commit --amend` avoided), no ORQ-21 worktree touched.

---

## 1. Summary of Changes

### 1a. Covered Providers Filter (`agent.sql`)
Enforced that `ClaimAgentTask` only freezes producing accounts for covered providers:
- `codex`
- `kiro`
- `antigravity` (accepting `agy` as input alias)

Uncovered providers (e.g. `openai`, `anthropic`, `ollama`) retain a `NULL` snapshot even if an assigned, approved account exists.

### 1b. Reclaim Fail-Closed (`agent.sql`)
Updated `ReclaimStaleDispatchedTaskForRuntime`:
- Preserves the frozen `credential_account_id` snapshot without re-resolving or modifying it.
- **Fails closed**: If a covered provider task reaches `dispatched` state with `credential_account_id IS NULL`, reclaim refuses to re-deliver (subquery excludes the row).

### 1c. Migration 128 Preflight Backfill (`NEXT_CANONICAL_task_usage_account_id.up.sql`)
Added preflight canonicalization backfill to update legacy `agy` provider/vendor strings to `antigravity` prior to constraint enforcement:
```sql
UPDATE accounts SET vendor = 'antigravity' WHERE lower(btrim(vendor)) = 'agy';
UPDATE agent_runtime SET provider = 'antigravity' WHERE lower(btrim(provider)) = 'agy';
```

### 1d. DB-Backed Unit Tests (`task_usage_account_test.go`)
Added 100% passing tests:
- `TestClaimFreeze_CoveredProvidersOnly`: Verifies covered providers freeze accounts and uncovered providers return `NULL`.
- `TestReclaim_FailsClosedForCoveredProviderWithNullSnapshot`: Verifies `ReclaimStaleDispatchedTaskForRuntime` fails closed when a covered provider task has `credential_account_id == NULL`.
- Updated test helper `handlerTestRuntimeProvider` to ensure test fixtures evaluate against covered provider `codex`.

---

## 2. Test Execution & Verification Results

| Check | Command | Result |
|---|---|---|
| **sqlc code generation** | `sqlc -f sqlc.staged-orq12.yaml generate` | ✅ PASS (Schema-validated, clean output) |
| **Go compilation** | `go test -c -o /dev/null ./internal/handler` | ✅ PASS |
| **Go vet** | `go vet ./internal/handler ./pkg/db/...` | ✅ PASS (0 warnings) |
| **Whitespace & Formatting** | `git diff --check` && `gofmt -l` | ✅ PASS (0 output) |
| **DB Unit Tests (17/17)** | `go test -v ./internal/handler` | ✅ **PASS (17/17 passed, 0 failed, 0 skipped)** |
| **Race Detector** | `go test -race ./internal/handler` | ✅ **PASS (0 race warnings, 2.7s execution time)** |

---

## 3. Commit Lineage

```
de38d3d (HEAD) fix(cost): enforce covered providers and fail-closed reclaim per GTM ADR
d33903f fix(cost): canonicalise the provider on both sides, as ORQ-21 R3 does
d9f0ae2 fix(test): drop the duplicated strPtr helper
41ae01b fix(cost): apply the authoritative claim predicates and drop the arbitrary LIMIT
1822dd9 fix(cost): freeze the producing account at claim, not at first usage report
e7c6a5c feat(cost): snapshot the producing account on task_usage (ORQ-12, staged)
```

---

## 4. Handoff SHA & Next Step

- **Handoff Commit SHA:** `de38d3d`
- **Ready for Combined Tree Integration:** Combined tree integration with `aac2227` (`agent/codex56-b/orq21-r3`) can proceed on an authorized branch.
