# Current pending tasks — canonical control table

- **Version:** 3.2
- **Updated:** 2026-07-28T20:32:00Z
- **Scope:** production workspace and P0 engineering program
- **Update owner:** General Tech Manager
- **Initial cohort:** 29 open cards
- **Closed today:** ORQ-16, ORQ-31, ORQ-46
- **Cancelled today:** ORQ-45
- **Original cards still pending:** 25
- **New operational defect:** ORQ2 lifecycle restoration
- **Total tracked pending work:** 26

This is the canonical human-readable status table. Runtime/API evidence and card notes remain authoritative for individual facts, but every material status, ETA, dependency or owner-decision change must be reflected here with a version increment and changelog entry.

**Dispatch policy:** all new executable work starts through Kanban assignee/API and must produce exactly one product task. Herdr is supervision-only and must not launch parallel work. Existing Herdr work may finish without mid-flight redispatch. See `owner-ruling-kanban-only-agent-dispatch.md`.

## Pending work — ordered by priority

| # | Card | State | Current work / next deliverable | ETA | Owner decision |
|---:|---|---|---|---:|---|
| 1 | ORQ-21 | **FINAL PASS / awaiting integration sequence** | Code-only stack `ee87f7b + ec8f945 + aac2227 + feccee4` independently passed. Evidence-only commit `42db561`; worktree clean. No technical defect remains in this lane. | Complete | None |
| 2 | ORQ-12 | **IN PROGRESS / final independent review running** | Fix commits `785a8ac` and `ea1eee7` are complete: reclaim exposes legacy NULL snapshots, migration 128 is canonical, and the combined ORQ-12/21 gate is green. Two Kiro reviews failed before start because slot 140 lacked runtime artifacts; its safe directory skeleton was restored without reading credentials. Independent review task `174565eb` is now running on Gemini-3.6-Flash-A through Kanban. | 20–45 min | None |
| 3 | ORQ-13 | **READY / queued immediately after ORQ-12** | The attempted task `81ff5f41` failed before start because Kiro slot 140 lacks its authenticated `data.sqlite3`; no product work ran. The card is unassigned and queued for Gemini-3.6-Flash-A immediately after its ORQ-12 independent verdict. Exact integration contract and standing authorization are already on the card. | 45–90 min after ORQ-12 review | None; owner authorization already recorded |
| 4 | ORQ-26 | **DIAGNOSIS COMPLETE / implementation dispatch pending** | Root cause is confirmed in `chat-window.tsx`: icons sit outside the `Button` supplied through `TooltipTrigger.render`, producing empty controls. The next implementation must be assigned natively through Kanban; no Herdr follow-up is allowed. | 45–90 min after native dispatch | None |
| 5 | OPS-LIFECYCLE | **Preflight complete / cutover package pending** | User-scope/system-scope topology, install/enable/verify/rollback and first-run acceptance were mapped. Local package plus review takes 30–55 min; real cutover is separate and final acceptance requires the first scheduled run after about 24 h. | 30–55 min code; 24 h acceptance | Cutover authorization |
| 6 | ORQ-42 | **Independent PASS / integration pending** | `fc77e89` passed 24/24 race tests and strict JSON/STOP_VARIANT review. Integrate tools; real JWT rotation remains separate. | 20–30 min code | Rotation window later |
| 7 | ORQ-39 | **IN REVIEW / runtime readiness PASS** | Read-only dependency audit of `dc1ed12` found zero blockers and confirmed supply-chain pins and anti-false-green structure. Await independent/integration disposition; real browser execution remains separately measurable. | 20–40 min | Integration authorization |
| 8 | ORQ-18 | **INDEPENDENT PASS / integration pending** | Commit `a943a3f` passed independent review: exact four-file scope, auth gating, pending/error/409/cascade/invalidation behavior, 23/23 Views, 4/4 Core, TypeScript and ESLint. Browser/live-delete acceptance remains separate. | 15–30 min integration | Integration authorization |
| 9 | ORQ-17 | **Production operational / closeout pending** | Frontend/backend/Tailnet access is healthy. Reconcile card wording and authenticated acceptance. | 15–30 min | Confirm Tailnet-only acceptance |
| 10 | ORQ-20 | **Code PASS / integration and deploy pending** | Integrate reasoning admission fix, deploy, smoke with a controlled Codex level; Antigravity effort wiring remains separate. | 30–60 min plus deploy | Final fleet-level policy may follow |
| 11 | ORQ-15 | **PARTIAL IMPLEMENTATION / native reassignment needed** | Exact integrated stack and `ResolveFrozen` path were implemented far enough for full build and targeted vet to pass. Provider monthly-limit interrupted before tests/commit. Continue only through a new Kanban assignment; no Herdr retry. | 45–90 min after native reassignment | None |
| 12 | ORQ-23 | **REVIEW BLOCK / design correction** | The design incorrectly put `account_id` on the daemon wire, invented `reasoning_tier`/`cost_cents`, and used wrong token-field names. Required correction: preserve server-side snapshot, use `thinking_level` and raw token counters, then re-review. | 20–40 min correction + 20–30 min review | None |
| 13 | ORQ-41 | **Code/evidence PASS; dormant** | Integrate dormant code separately from activation. Activation still needs DB/W2/W3 and queue-zero gate. | 20–30 min merge; 4–8 h activation | Dormant merge authorization |
| 14 | ORQ-38 | **Reviewed / integration pending** | Verify reviewed commit pair on final integration tree and run focused fail-closed gate. | 20–40 min | Final integration authorization |
| 15 | ORQ-43 | **Blocked by DB sequence** | Durable daemon-token idempotency/migration work waits for migration 128 and a fresh registrar scan. | 90–150 min after DB chain | Token behavior authorization later |
| 16 | ORQ-33 | **IN REVIEW / independent review incomplete** | Scope correction to `JWT_SECRET` and the V3 contract are ready, but the independent-review session stopped at a tool-approval boundary without a final verdict. Reassign the review natively through Kanban. | 20–40 min after native review assignment | Rotation window only after PASS |
| 17 | ORQ-34 | **READINESS COMPLETE / external metadata** | Secret-safe consumer map, `asm-exec` contract, rollback and authorization matrix are complete. Missing ARN/region/JSON-key/stage identity, provider-key distinction, overlap/reload confirmation and rotation window remain external inputs. | 15–30 min metadata validation; 1–2 h rotation; 24 h acceptance | Metadata and rotation window |
| 18 | ORQ-35 | **TASK FAILED / agent authentication repair needed** | The single native task `9248f3e2` started but failed because Codex-C's refresh token was revoked. The card must be reassigned to a healthy authenticated agent or Codex-C must log in again; no duplicate task is active. | 2–4 h after healthy reassignment | Rotation authorization later |
| 19 | ORQ-36 | **INDEPENDENT PASS / disposition pending** | Independent review confirmed `mat_`, `mdt_`, `mul_` and `mcn_` boundaries, fail-closed task-token isolation and lifecycle/rollback semantics without secret exposure. | 15–30 min close/integration disposition | Mutation authorization later |
| 20 | ORQ-44 | **External questions open** | Resolve overlap, scope, introspection, AWSPREVIOUS, post-swap proof and exact secret identity. | No honest ETA until answers | Provider/owner answers and window |
| 21 | ORQ-14 | **IN PROGRESS / authoritative telemetry investigation** | Kanban task `f43e3bc1` is running on Gemini-3.6-Flash-B. It must locate machine-readable per-attempt counters or prove the exact missing provider contract. Estimates from text length, time, cost or averages are forbidden; unavailable data remains explicit NULL. | 45–90 min for factual verdict; implementation ETA follows evidence | None |
| 22 | ORQ-19 | **READY / queued immediately after ORQ-14** | Kiro task `def4e23a` failed before start on the incomplete slot 140 and Opus-46 task `c2a35b50` failed before start on individual provider quota. No preflight ran. The card is unassigned and queued for Gemini-3.6-Flash-B after ORQ-14. Owner authorization for the bounded cascade is already recorded. | 30–60 min preflight + 15–30 min execution after ORQ-14 | None unless target set changes |
| 23 | ORQ-40 | **Maintenance pending** | Align Codex CLI version with exact pin, verification and rollback. | 30–60 min | Maintenance-window authorization |
| 24 | ORQ-30 | **Acceptance pending** | Review durable backend/JWT containment evidence and close or return one concrete correction. | 15–30 min | Acceptance |
| 25 | ORQ-11 | **Legacy acceptance pending** | Close the legacy E2E root-count card or declare it obsolete. | 15 min | Business-value decision |
| 26 | ORQ-37 | **REVIEW BLOCK / installer hardening** | Commit `90b56a5` has the right user-scoped direction and 25 positive gates, but adversarial review reproduced path traversal, root/TMPDIR escape, predictable-temp symlink overwrite and unmanaged rollback deletion. Correct B1–B5 and rerun the adversarial matrix through a native Kanban task. | 45–75 min correction + 20–35 min review | Identify issuer before origin rotation |

## Decisions currently required from the owner

1. Confirm Tailnet-only production access for ORQ-17.
2. Authorize the reviewed ORQ-26 integration sequence when handed off.
3. Authorize dormant-only ORQ-41 merge separately from activation.
4. Approve maintenance window for ORQ-40.
5. Return acceptance on ORQ-30 and decide whether ORQ-11 still has value.
6. Provide rotation metadata/windows only when ORQ-34/35/36/42/44 readiness is complete; never send secret values in chat.

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

## Version history

| Version | Date | Change |
|---|---|---|
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
