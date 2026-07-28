# Current pending tasks — canonical control table

- **Version:** 4.0
- **Updated:** 2026-07-28T21:49:00Z
- **Scope:** production workspace and P0 engineering program
- **Update owner:** General Tech Manager
- **Kanban cohort:** 24 cards
- **Done:** 11 (45.8%)
- **Target today:** 18 Done (75%)
- **Remaining to target:** 7
- **Live execution:** 4 product tasks, 4 distinct AGY accounts
- **Current columns:** Done 11, In Progress 5, In Review 7, Todo 1, Blocked 0

This is the canonical human-readable status table. Runtime/API evidence and card notes remain authoritative for individual facts, but every material status, ETA, dependency or owner-decision change must be reflected here with a version increment and changelog entry.

**Dispatch policy:** all new executable work starts through Kanban assignee/API and must produce exactly one product task. Herdr is supervision-only and must not launch parallel work. Existing Herdr work may finish without mid-flight redispatch. See `owner-ruling-kanban-only-agent-dispatch.md`.

## Pending work — ordered by priority

| # | Card | State | Current work / next deliverable | ETA | Owner decision |
|---:|---|---|---|---:|---|
| 1 | ORQ-21 | **IN PROGRESS — E2E acceptance** | Integrated and deployed. Four clean AGY accounts are registered with unique slots and assignments. Product task `646900f0` is running on Gemini-B with frozen account `23c485f0…`. | 15–30 min | None |
| 2 | ORQ-18 | **IN PROGRESS — production acceptance** | UI commit `a943a3f` is integrated in `9f7b963`; frontend image `f79a78b…` is live with HTTP 200 and zero restarts. Product task `b9be6624` is running on Agy-A7 with frozen account `adb99fe1…`. | 15–30 min | None |
| 3 | ORQ-26 | **IN PROGRESS — implementation** | Root cause is known in the chat-panel button composition. Product task `6300e7a0` is running on Gemini-A with frozen account `fc2a6653…`. | 30–75 min | None |
| 4 | ORQ-14 | **IN PROGRESS — telemetry completion** | Product task `7515612f` is running on Agy-A8 with frozen account `733b4787…`; preserve authoritative counters and explicit NULL when unavailable. | 45–90 min | None |
| 5 | ORQ-15 | **IN PROGRESS — integration lane** | Multi-account affinity implementation exists in `dfb90f1`; final reconciliation must consume the newly proven four-slot map and avoid a second source of truth. No active product task while all four accounts are occupied. | 30–60 min after an AGY account frees | None |
| 6 | ORQ-20 | **IN REVIEW — deployed, canary pending** | Reasoning admission is deployed and six Codex/Kiro agents carry `high`; successful provider canary is still required. Codex/Kiro authentication remains the limiting input. | 20–40 min after usable provider login | Owner action only if Codex/Kiro login is requested |
| 7 | ORQ-23 | **IN REVIEW — correction required** | Preserve the server-side frozen account snapshot; use `thinking_level` and raw token counters. Do not put account identity on the daemon wire. | 40–70 min | None |
| 8 | ORQ-39 | **IN REVIEW — executable gate** | Workflow/spec structure passed review; execute locally or on available CI without treating GitHub billing as a product blocker. | 30–60 min | None |
| 9 | ORQ-33 | **IN REVIEW — JWT rotation** | Scope is correctly `JWT_SECRET`; finalize independent review and execute only in a queue-zero window with rollback. | 45–90 min | Owner login/window when prompted |
| 10 | ORQ-34 | **IN REVIEW — external key metadata** | Secret-safe consumer map is ready; exact secret identity and provider-key distinction remain to be supplied without values in chat. | 30–60 min after metadata | Owner action when exact metadata is requested |
| 11 | ORQ-36 | **IN REVIEW — disposition** | Token-boundary review passed; convert the verified no-mutation scope into Done or return one concrete implementation requirement. | 15–30 min | None |
| 12 | ORQ-37 | **IN REVIEW — hardening correction** | User-scoped umask approach is valid, but installer path traversal/TMPDIR/temp-file/rollback issues require correction before cutover. | 60–110 min | Issuer identity only for later origin rotation |
| 13 | ORQ-35 | **TODO — provider authentication** | Prior Codex task failed on revoked refresh token. Reassign to the next free healthy AGY account or perform a Codex owner login if Codex-specific behavior is required. | 15 min dispatch; 1–2 h work | Owner action only if Codex-specific login is required |

## Decisions currently required from the owner

No immediate owner decision blocks the four running tasks. Human actions are now elevated immediately as **OWNER ACTION — IMMEDIATE** with the exact reason, action and estimated duration. Likely next actions are:

1. Codex/Kiro interactive login only if ORQ-20 or ORQ-35 requires those providers specifically.
2. Exact non-secret secret identifiers and a rotation window when ORQ-33/34 reach executable readiness.
3. Origin/issuer identity for the two MCP tooling headers before ORQ-37 rotates them; local custody hardening does not wait for that identity.

Standing owner authorization recorded at `2026-07-28T20:29Z`: when the exact target, verified backup, rollback and bounded action are technically approved by the General Tech Manager, execution is authorized without asking the owner for the same generic approval again. A new decision is required only if the target set or risk scope changes.

## Live Kanban reconciliation

At `2026-07-28T15:54:35Z`, the exact project `4b0ef49b-df06-4e83-9a29-8a23b34821d4` was read back after a status-only repair:

| Column | Before | After |
|---|---:|---:|
| Blocked | 5 | 5 |
| Done | 5 | 5 |
| In Progress | 0 | 1 |
| In Review | 3 | 2 |
| Todo | 9 | 9 |

ORQ-12 is now raw `in_progress`. Its assignee remains `NULL`, active product-task count remains `0`, and no other card was changed. Evidence: `orq12-in-progress-status-verification.md` (`sha256:90cb69c18c4a67fa29342e759a412094c6101c6722803fd3f619e993e35dea0f`).

After the ORQ-21 delta received independent PASS, ORQ-31 was released from `PAUSED_SEQUENCE` and reconciled from raw `in_review` to `in_progress`. The verified live counts are now: Blocked 5, Done 5, **In Progress 2 (ORQ-12 and ORQ-31)**, In Review 2 and Todo 9. Assignee remains `NULL`, active product-task count remains `0`, and no other card changed. Evidence: `orq31-in-progress-status-reconciliation.md` (`sha256:8df5c227d73f8e98fae1f4ccb2ba33ddb327b4d2b9cf54fd119d15b3ac29edc3`).

ORQ-31 subsequently received independent PASS for its declared Wave-A containment scope and moved to `done`; ORQ-39 moved to `in_review`. Confirmed executions then moved ORQ-15, ORQ-23, ORQ-33, ORQ-34, ORQ-26 and ORQ-36 from `todo` to `in_progress`. Latest verified count: Blocked 5, Done 6, **In Progress 8**, In Review 3, Todo 2. The remaining Todo cards are ORQ-35 and ORQ-37. Active product-task queue remains zero because these were status-only reconciliations, not assignee changes.

ORQ-33 and ORQ-18 then delivered reviewable artifacts and moved to `in_review`; ORQ-37 started and moved to `in_progress`. Latest verified count: Blocked 5, Done 6, **In Progress 7**, In Review 5, **Todo 1 (ORQ-35)**. Active product-task queue remains zero.

ORQ-12 then moved to `in_review` because its execution and gates were complete. External Herdr-owned cards received idempotent `/note` ownership markers, while ORQ-26 retains its real Agy-P0-A8 assignee. Latest verified count: Blocked 5, Done 6, **In Progress 6**, **In Review 6**, Todo 1. Active product-task queue remains zero. Future work must start through product assignee/API with exactly one product task; Herdr may supervise but must not launch a duplicate execution.

After the review wave, ORQ-15 returned to `in_progress` for the B1 correction and ORQ-35 was reassigned through the product API to an eligible Codex-C. Latest verified count: Blocked 5, Done 6, **In Progress 3**, **In Review 10**, **Todo 0**. The active product queue contains exactly one ORQ-35 task in `dispatched`; no Herdr duplicate was launched.

At `2026-07-28T17:25:57Z`, ORQ-12 was assigned natively through Kanban to Opus48-A and moved to `in_progress` in the same update. Exactly one product task, `8b825daf-f675-41a9-845b-c7497bf7e3a6`, was created and reached `running`; no Herdr activation was used. The stale Opus-46-B blocked metadata was removed and the card now carries the exact migration-128 execution contract. The ORQ-35 task was also rechecked and is terminal `failed`, not active, because Codex-C's provider refresh token was revoked. Current project counts read back from the API: Blocked 5, Done 6, **In Progress 3**, **In Review 9**, **Todo 1**.

The first ORQ-12 task completed as a diagnostic instead of writing: it proved that `ReclaimStaleDispatchedTaskForRuntime` filters out covered-provider legacy rows with `credential_account_id IS NULL`, preventing the ORQ-21 fail-closed cancellation path from seeing them. The issue was returned to `in_progress`, the stale no-write interpretation was explicitly superseded, and exactly one new Kanban task (`5601a616-3a14-4219-8409-97c7879d389b`) was created at `2026-07-28T19:07:43Z`. It is now `running`; no Herdr activation or duplicate task was used.

At `2026-07-28T20:32Z`, the blocked column remains **zero**, but execution capacity was reconciled honestly after two pre-start failures. ORQ-12 independent review is running as task `174565eb` on Gemini-3.6-Flash-A and ORQ-14 authoritative-token work is running as `f43e3bc1` on Gemini-3.6-Flash-B. ORQ-13 task `81ff5f41` failed before start on missing Kiro authentication state; ORQ-19 task `c2a35b50` failed before start on Opus-46 individual quota. They were unassigned and returned to Todo instead of being falsely shown as active, with explicit next sequencing to Gemini-A and Gemini-B respectively. The Kiro slot-140 directory skeleton was restored to the required 0700 layout without reading or copying credentials, but its missing `data.sqlite3` was not fabricated. Live project counts are **Done 6, In Progress 3, In Review 12, Todo 3, Blocked 0**; the third Todo is ORQ-35, whose prior Codex-C task failed because its provider refresh token was revoked.

At `2026-07-28T21:49Z`, the AGY credential pool was rebuilt from zero. Four distinct owner-authenticated sessions were assigned automatically to slots `162`, `163`, `168` and `169`; all token files are nonempty `0600` under `0700` slot directories and no token content was read. The daemon allowlist, its local affinity ledger and the production `accounts`/`approved_accounts`/`assignments` metadata now agree. Four product tasks are concurrently `running` through Kanban, each with a distinct frozen `credential_account_id`: ORQ-18/Agy-A7, ORQ-26/Gemini-A, ORQ-14/Agy-A8 and ORQ-21/Gemini-B. Live project counts are **Done 11, In Progress 5, In Review 7, Todo 1, Blocked 0**.

## Version history

| Version | Date | Change |
|---|---|---|
| 4.0 | 2026-07-28 | Rebased the canonical table on the live 24-card Kanban, recorded 11 Done and the seven-card gap to 75%, deployed ORQ-18, rebuilt the four-account AGY pool from zero, and recorded four concurrent Kanban tasks with distinct frozen account IDs. |
| 3.2 | 2026-07-28 | Corrected two pre-start execution failures without cosmetic status: ORQ-13 and ORQ-19 are READY and serially queued behind the two healthy Gemini tasks; blocked remains zero and live counts are 6/3/12/3/0. |
| 3.1 | 2026-07-28 | Eliminated the blocked Kanban column by launching real work for ORQ-12/13/14/19; recorded standing owner authorization, exact task IDs, credential-slot infrastructure findings and current 6/6/11/1/0 board counts. |
| 3.0 | 2026-07-28 | Recorded the real ORQ-12 combined-tree reclaim defect and launched one native correction task with explicit write authority, focused regressions and migration-128 promotion contingent on a green gate. |
| 2.9 | 2026-07-28 | Dispatched ORQ-12 natively to Opus48-A with one running product task and an exact migration-128 execution contract; corrected ORQ-35 from dispatched to terminal auth failure after live verification. |
| 2.8 | 2026-07-28 | Reconciled latest deliveries and ETAs: ORQ-18 and ORQ-36 independent PASS; ORQ-23 and ORQ-37 actionable BLOCKs; ORQ-15 partial build/vet interrupted by provider limit; ORQ-26 diagnosis complete; ORQ-33 review incomplete; ORQ-34 readiness complete. |
| 2.7 | 2026-07-28 | Activated the owner ruling that all new agent work, including independent reviews, is dispatched only through Kanban assignee/API with exactly one product task. Herdr is supervision-only; Kiro-Opus5 remains reserved. |
| 2.6 | 2026-07-28 | Reached Todo zero. Returned ORQ-15 to implementation for the frozen-account/HOME binding defect and started ORQ-35 through the correct product-assignee flow with exactly one dispatched task and no Herdr duplicate. |
| 2.5 | 2026-07-28 | Recorded ORQ-37 Ruling V2: unknown issuer blocks only origin rotation; reversible private custody and scoped `umask 077` implementation continues, with no deletion and no host cutover before review/canary/rollback. |
| 2.4 | 2026-07-28 | Corrected ORQ-37 title and scope to MCP tooling-header lifecycle (Plan C/B7); explicitly separated `mdt_` into ORQ-43 and `mcp_config` regression coverage into its own follow-up. |
| 2.3 | 2026-07-28 | Moved completed ORQ-12 execution to review; recorded idempotent external-owner notes for the temporary Herdr-owned lanes and established the permanent policy of product assignment first with exactly one task. |
| 2.2 | 2026-07-28 | Advanced ORQ-33 and ORQ-18 to review, started ORQ-37, and reduced the live Todo column to one card (ORQ-35). Current board: 7 In Progress, 5 In Review, 6 Done, 5 Blocked. |
| 2.1 | 2026-07-28 | Closed ORQ-31 after independent Wave-A PASS; moved ORQ-39 to review; reconciled six confirmed executions to produce 8 In Progress and only 2 Todo. Corrected ORQ-26 to the actual chat-panel regression card rather than the historical CI-label collision. |
| 2.0 | 2026-07-28 | Resolved ORQ-31 review independence without pausing the card: Codex56-A is limited to factual current-state measurement and Codex56-B owns the independent binary verdict. |
| 1.9 | 2026-07-28 | Recorded ORQ-21 R3 FINAL PASS and its evidence-only commit `42db561`; ORQ-12 combined gate is green and now waits only for Registrar materialization plus the aggregate active-queue-zero integration/deploy gate. |
| 1.8 | 2026-07-28 | Recorded independent PASS for ORQ-21 `feccee4`; released ORQ-31 from sequence hold and reconciled it to verified `in_progress`. Live Kanban now has two cards in progress: ORQ-12 and ORQ-31. |
| 1.7 | 2026-07-28 | Reconciled live Kanban state: ORQ-12 moved from stale `in_review` to verified `in_progress`, with no assignee change or product task; recorded before/after column counts and current dispatches for ORQ-18, ORQ-39 and OPS-LIFECYCLE. |
| 1.6 | 2026-07-28 | Marked ORQ-21 independently PASS for this stage; recorded the narrow ORQ-12 invented-test-schema assumption under active correction; corrected OPS-LIFECYCLE from “external capacity” to transient request-rate throttling with normal plan capacity. |
| 1.5 | 2026-07-28 | Recorded terminal response-stream throttle on the one serial OPS-LIFECYCLE preflight. Cleared Opus48-A assignee, prohibited immediate retry and returned the lane to external-capacity handoff. |
| 1.4 | 2026-07-28 | Assigned exactly one serial Opus48-A task to OPS-LIFECYCLE for read-only preflight; mutation remains NOT_READY. Added max-one-heavy-request, zero-subagent and no-parallel-retry limits. |
| 1.3 | 2026-07-28 | Corrected Opus48-A classification: transient request-rate throttling at 60.4% plan usage, not capacity or credit exhaustion; refreshed the canonical timestamp. |
| 1.2 | 2026-07-28 | Recorded metadata-only ORQ-12 handoff: Opus48-A released, Opus-46-B pinned as pending owner, executable assignee empty and `ACK READY` required before source write/one task. Codex56-B remains ORQ-21 owner. |
| 1.1 | 2026-07-28 | Converted the on-disk report to one continuous table and recorded the explicit ORQ-12 ownership handoff from throttled Opus48-A to Opus-46-B. |
| 1.0 | 2026-07-28 | Initial canonical pending-task table. Corrected prior duplicate ORQ-37 row, restored omitted ORQ-43, incorporated ORQ-21 `aac2227` peer PASS, ORQ-26 full compatibility PASS, ORQ-39 amendment commit and ORQ-42 independent PASS. |
