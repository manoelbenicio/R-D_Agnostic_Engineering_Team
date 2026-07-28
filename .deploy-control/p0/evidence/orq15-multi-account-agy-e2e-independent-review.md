# ORQ-15 — independent E2E review of AGY multi-account affinity

- Card: `ORQ-15` (`4ddb3400…`) — multi-account AGY activation with correct affinity
- Reviewer: Codex56#B (`w7:p4`)
- Inputs:
  - ORQ-21 final code stack: `feccee4be2d9443e37f8b050de05473d1885ad38`
  - ORQ-12 final code stack: `dd95e0a7ef78d8844ef55e0a1765d8849b7d18bf`
  - ORQ-21 combined-gate evidence: `42db56128b9339f1cf42342708de9a33e881685b`
- Mode: READ-ONLY. No source edit, merge, migration, database, provider, task,
  secret, production, container, runtime, or board mutation.
- **Verdict: BLOCK for ORQ-15 E2E acceptance.**

The two stacks establish strong isolated database and filesystem contracts, but
they do not yet prove that the account frozen for attribution is the same account
whose HOME actually executes an AGY task. Actual Antigravity executions also
produce no token-usage entries today, so the required per-account financial report
cannot be generated from a real AGY task.

## What passes

| contract | result | evidence |
|---|---|---|
| Tenant/provider/approval/status/worktype predicates | **PASS at DB/unit level** | ORQ-12 `agent.sql:317-337` binds both tenants to the agent workspace, exact canonical runtime provider to account vendor, `allowed=true`, status `available/leased`, and scope `GENERAL`, with no arbitrary `LIMIT`. ORQ-21 `resolver.go:83-151` applies the corresponding metadata-only fail-closed checks. |
| Write-time canonicalization | **PASS** | ORQ-21 seed maps trimmed `agy` to stored `antigravity` and rejects unsupported vendors; ORQ-12 staged migration backfills both `accounts.vendor` and `agent_runtime.provider`, then the claim uses exact equality. The combined gate exercised real `AGY` drift and obtained exact `antigravity`. |
| Account exclusivity | **PASS after staged migration** | ORQ-21 has advisory locking and application conflict checks; ORQ-12 adds preflight plus unique `assignments(account_id)`. The uniqueness is not present before the staged migration. |
| Immutable snapshot and usage copy | **PASS in the database layer** | `ClaimAgentTask` freezes `agent_task_queue.credential_account_id` inside the dispatch update. `UpsertTaskUsage` copies only that snapshot and uses existing-first `COALESCE`, never a daemon account field or live assignment. |
| Retry semantics | **PASS in DB tests** | A retry is a new queue row, starts unfrozen, and may freeze a new approved account. The prior row and its usage retain their original account. |
| Reclaim semantics | **PASS in DB tests** | Reclaim updates only `dispatched_at`; covered-provider rows with a NULL snapshot fail closed. A frozen snapshot is not rewritten. |
| Mixed-version daemon guard | **PASS as dormant mechanism** | `MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED` makes a covered provider fail closed even when an old server omits assignment metadata. Default is off, so activation still needs an explicit deploy gate. |
| Private HOME validation | **PASS at unit level** | ORQ-21 validates controlled slot-root containment, exact `slot/home` shape, owner, private modes, no symlink, and AGY credential-tree structure without reading credential values. |
| Existing combined DB gate | **PASS within its scope** | Recorded results: handler matrix 39/0/0; focused race 7/0/0; registry DB 5/0/0; full handler adds zero failures and zero skips relative to baseline; staged up/down/up and canonical backfill pass. |

## Blocking findings

### B1 — frozen account is not bound to the HOME sent to the daemon

ORQ-12 freezes account A inside `ClaimAgentTask`. After that statement returns,
ORQ-21 calls `CredentialAssignments.Resolve(agentID, provider)` again in
`internal/handler/daemon.go:1295-1325`. That resolver reads the **current**
`assignments` row and returns both `Assignment.AccountID` and `HomeDir`, but the
handler only sends `HomeDir`; it never compares `assignment.AccountID` with
`task.CredentialAccountID`.

The race is:

1. claim atomically freezes A;
2. assignment changes A → B before response construction, or before stale-row reclaim;
3. ORQ-21 resolves current B and sends HOME B;
4. daemon executes with B;
5. usage copies immutable snapshot A.

This is silent misattribution: the database tests prove that usage stays A, but
do not prove that A supplied the credential used by execution. Search across
both final stacks found no frozen-ID-to-resolved-ID comparison. The ADR rejects
current-assignment authority for attribution, so allowing it to select the
execution HOME violates the same invariant at the other half of the boundary.

Required correction: in the integrated handler, a covered-provider task must have
a valid frozen ID and the metadata resolver must be bound to that ID. At minimum,
resolve then require `assignment.AccountID == task.CredentialAccountID`; on
mismatch, revoke, missing assignment, or metadata drift, cancel/fail closed and
never emit either HOME. A stronger API is `ResolveFrozen(agentID, accountID,
provider)` so the immutable task snapshot is an explicit input. Add initial-claim
and reclaim race tests that rotate A → B between the SQL claim and response build.

### B2 — real AGY tasks cannot populate the required account report

At `feccee4:server/pkg/agent/antigravity.go:156-165`, the result explicitly sets:

```text
Usage: map[string]TokenUsage{}
```

because the Antigravity CLI does not expose per-turn token usage. The daemon
therefore emits no `TaskUsageEntry`, and `ReportTaskUsage` inserts no row for an
actual AGY task. ORQ-12's `/api/dashboard/usage/by-account` endpoint aggregates
token rows correctly, but for AGY there is no row to aggregate.

The historic ORQ-15 criterion requires an individualized financial report for
the active AGY accounts. Synthetic direct SQL/upsert tests prove storage, not
AGY E2E. Do not invent zero-token rows or guess a model: obtain an authoritative
provider usage source, or obtain an owner product ruling that changes ORQ-15's
acceptance to a separate execution-attribution receipt plus externally sourced
cost. Until then, the financial E2E is impossible.

### B3 — the integration target does not exist yet

The final commits share exactly three files:

- `server/internal/daemon/daemon.go`
- `server/internal/daemon/types.go`
- `server/internal/handler/daemon.go`

A three-way `merge-tree` has no textual conflict markers, but the files combine
the assignment/HOME wire, provider normalization, claim response, and ORQ-13
thinking-level usage wire. This is a semantic integration surface and must be
reviewed as one exact commit.

The ORQ-12 migration remains
`migrations/staging/NEXT_CANONICAL_task_usage_account_id.*`; it is not applied by
the canonical migrator. The durable uniqueness constraint and snapshot columns
therefore do not exist merely because the branches pass isolated review. Registrar
materialization to the already governed number and canonical SQLC regeneration
must precede integration acceptance.

The preserved `orq12-combined-gate` branch is not a final integration target:
neither `feccee4` nor `dd95e0a` is its ancestor. It contains older intermediate
commits and cannot substitute for a fresh exact-stack integration.

### B4 — no four-account AGY E2E or settled eligible set

Neither final stack contains a test that creates the ORQ-15 eligible account set,
claims tasks across every account HOME, executes through the daemon, and reconciles
the resulting report. Existing tests use one or two synthetic accounts and test
contracts separately.

The eligibility documents also conflict:

- the original ORQ-15 acceptance names four slots: `141,145,146,150`;
- the later blocker audit directs preparation against `141,145,146` **without
  slot 150 OAuth**.

The final gate cannot predict whether the cardinality is three or four. Owner/GTL
must freeze the exact pseudonymous eligible set and either authorize slot 150 or
formally amend the four-account criterion. No login or credential inspection is
implied by this ruling.

### B5 — report is token aggregation, not yet the complete cost report

`ListTaskUsageByAccount` returns the four token counters, task count, and row
count by pseudonymous account. It does not calculate monetary cost. ORQ-13 adds
`thinking_level`, but no integrated per-account pricing/version join is proven
here. If ORQ-15 retains “financial/cost report” as its exit criterion, the final
gate must reconcile per-account+tier cost as well as token totals, with the global
sum preserved.

## Final gate plan

### G0 — freeze authority and inputs

1. Record exact integrated parent SHAs and Registrar materialization for the
   ORQ-12 migration; no placeholder remains.
2. Freeze the eligible pseudonymous AGY account set and cardinality. Resolve the
   `141/145/146` versus `141/145/146/150` conflict in writing.
3. Record metadata only: account UUID, agent UUID, canonical provider, and opaque
   slot label. Never record email, token, credential file content, or provider
   identity.
4. For any operational stage, require two aggregate zero-active-queue readings
   over `queued`, `dispatched`, `running`, and `waiting_local_directory`.

### G1 — exact integration and fail-closed binding

1. Integrate `feccee4` and `dd95e0a` on a dedicated branch; resolve the three
   shared files with both assignment and thinking-level behavior.
2. Implement frozen account ↔ resolved HOME equality/bound resolver.
3. Regenerate SQLC from the canonical materialized migration and require a second
   generation with zero diff.
4. Build/vet and run focused race tests. No mixed-version gate activation yet.

### G2 — disposable PostgreSQL contract

On one private ephemeral database and the exact integrated SHA:

1. Run canonical migrations and 128 up → down → up with schema and uniqueness
   assertions.
2. Seed N distinct accounts and N distinct agents under one workspace using only
   metadata; prove one-to-one uniqueness under concurrent imports.
3. Exercise tenant, provider, approval, `GENERAL`, status, revoke, duplicate, and
   non-canonical drift failures with zero skips.
4. Claim A, rotate live assignment to B before response build, and require no
   HOME B response for snapshot A.
5. Reclaim frozen A after reassignment and require either HOME A from the frozen
   resolver or explicit fail-closed cancellation — never HOME B.
6. Retry as a fresh row and prove it freezes B independently.
7. Report usage twice and prove existing-first immutability, workspace isolation,
   NULL visibility, and global-token conservation.

### G3 — content-free daemon/filesystem integration

Use private synthetic account homes with no real credentials:

1. N unique `0700` slot/home trees with only structurally valid dummy fixtures;
2. prove each pseudonymous account selects exactly its bound HOME;
3. repeat tasks for the same agent and prove stable affinity;
4. prove distinct agents cover all N accounts;
5. wrong root, symlink, wrong owner/mode, missing layout, frozen/live mismatch,
   and pre-existing task root all fail closed;
6. enable `MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED` only inside this isolated gate.

### G4 — separately authorized provider E2E

This is a later operational window, not authorized by this review:

1. Resolve B2 with an authoritative AGY usage/cost source or an explicit revised
   product criterion.
2. Deploy schema/server first, then daemon gate-on, with queue zero maintained.
3. Execute one bounded new task per approved pseudonymous account; do not replay
   the historical failed ORQ-15 task.
4. Capture only task ID, frozen account UUID, opaque slot label, terminal result,
   usage-row presence, and report totals.
5. Require all N accounts represented, zero cross-account HOME/snapshot mismatch,
   zero unattributed post-cutover rows, repeat-task affinity, and per-account sums
   exactly equal the global total.

### G5 — rollback

Application rollback disables admission/enforcement before reverting binaries but
keeps the expanded schema and immutable attribution data. Do not run the destructive
down migration in production. Preserve frozen rows and pseudonymous receipts; a
new retry is a new attempt after rollback.

## Conclusion

**BLOCK.** Predicates, snapshot storage, exclusivity, retry/reclaim database
semantics, and private HOME validation are individually strong and have credible
ephemeral-DB evidence. ORQ-15 still lacks the critical binding between the frozen
account and execution HOME, an authoritative AGY usage source, a final integrated
SHA/materialized migration, and a settled eligible account set. Passing isolated
tests cannot replace that E2E proof.

