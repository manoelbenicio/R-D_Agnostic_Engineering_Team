# Project 360 status — 2026-07-28

Cutoff: 2026-07-28 13:45 UTC. Workspace: `20fce817-895d-447b-965a-49f5e279314a`.

Production is healthy: frontend, health and readiness return 200; anonymous `/api/me` returns 401; seven Tailnet mounts are active; Funnel is off; active queue is zero. ORQ2 root is 86% used with 8.7 GB free, and the daily lifecycle timer last completed successfully.

Board totals: 29 open issues (`todo=17`, `in_review=7`, `blocked=5`), including 13 urgent. Several board columns are stale; the Actual column below is authoritative for this report.

## Full issue matrix

| Issue | Pri | Board -> actual | Current owner / executor | Pending work | ETA / dependency | Owner decision or action |
|---|---|---|---|---|---|---|
| ORQ-11 | Low | in_review -> acceptance | Archived E2E agent / reviewer needed | Accept root-count evidence or close as obsolete. | 15 min after review | Decide whether this legacy E2E card still has business value. |
| ORQ-12 | Urgent | todo -> blocked predecessor | Opus48-A / LANE-DB | Implement account identity in `task_usage` after ORQ-21 stabilizes. | 2–4 h after ORQ-21 | No immediate decision; preserve serial DB order. |
| ORQ-13 | High | todo -> blocked predecessors; candidate testing | Opus48-B / registrar / independent reviewer | Combined four-commit DB gate and lint; final integration waits ORQ-21 then ORQ-12. Reservation 127 renewed to 2026-07-29 13:21:29Z. | Candidate gate 60–90 min; final date depends on ORQ-21/12. | Merge authorization only after predecessors and independent PASS. |
| ORQ-14 | Urgent | blocked -> externally blocked | Codex-C | Providers AGY/Kiro do not expose truthful per-turn token totals. | No honest ETA | Decide whether estimates are acceptable; current acceptance requires real totals. |
| ORQ-15 | Urgent | todo -> dependency blocked | Gemini-A | Multi-account affinity DAG after registry/assignments authority. | 2–4 h after ORQ-21 | Confirm approved account inventory when ORQ-21 requests it. |
| ORQ-16 | High | in_review -> done candidate | Gemini-B / AGY closeout | A7/A8 real retries completed; allowlist 141/145/146; accept closeout. | 15 min | Approve closure. |
| ORQ-17 | High | todo -> production operational / card reconciliation | Opus48-B | Product is live through Tailnet with seven mounts; reconcile card wording and remaining authenticated-login acceptance. | 15–30 min for board reconciliation | Confirm whether current Tailnet-only access satisfies this card. |
| ORQ-18 | High | todo -> blocked by test foundation | Codex-B | Runtime-delete UI patch needs clean-base tests and browser QA. | 1–2 h after ORQ-26 | No decision until CI foundation works. |
| ORQ-19 | Medium | blocked -> destructive decision | Codex-C | Obsolete runtimes remain linked to archived agents; deletion requires cascade. | 30–60 min after approval | Explicitly approve or reject cascade deletion. |
| ORQ-20 | High | blocked -> code PASS, deployment/values pending | Opus48-A / reasoning lane | Reasoning admission fix `f5660e9` has independent technical PASS; evidence wording correction, clean integration and deploy remain. Antigravity effort wiring is separate. | 10–15 min integration gates + deploy window + 10 min smoke | Select actual thinking levels/cost policy; authorize deploy when clean integration is ready. |
| ORQ-21 | Urgent | todo -> critical DB predecessor | Codex-B | Stabilize approved-account/assignment registry; current worktree is dirty and incomplete. | 2–4 h after ownership resumes | Confirm approved accounts/assignments authority if the implementation cannot derive it. |
| ORQ-23 | High | todo -> dependency blocked | Codex-A | Complete Phase-3 usage accounting and rollback 4.5 after usage chain. | 1–2 h after ORQ-21→12→13 | No immediate decision. |
| ORQ-26 | Urgent | in_review -> local DB gate active | Opus48-D / local ephemeral runner | Branch `ci/orq26-db-gate` is at `20cab478`. GitHub reported a billing lock, but Actions is not a product dependency. The same 11-test gate is now executing with an ephemeral PostgreSQL on ORQ1 and a private tunnel. | 30–60 min | No owner action required for the local gate; GitHub billing is now a separate infrastructure issue. |
| ORQ-30 | Urgent | in_review -> acceptance | Owner/reviewer | Durable environment/JWT remediation evidenced; complete acceptance review. | 15–30 min | Approve closeout or state remaining acceptance gap. |
| ORQ-31 | Urgent | in_review -> evidence gap | Unassigned | Reconcile Security Wave A containment evidence and acceptance. | 30–60 min | Assign reviewer/acceptance owner. |
| ORQ-33 | Urgent | todo -> preflight active | **Opus-46#A** | Rev-token rotation/readiness design, blast radius and rollback; no secret mutation yet. | Preflight 30–60 min; execution 1–2 h after authorization | Later authorize bounded secret rotation and rollback window. |
| ORQ-34 | Urgent | todo -> preflight active | **Opus-46-B** | `OPENAI_API_KEY` consumers, secret-safe rotation and rollback; no value read/mutation. | Preflight 30–60 min; execution 1–2 h after authorization | Later provide/confirm secret ARN/key/stage and authorize rotation window. |
| ORQ-35 | Urgent | todo -> unassigned | Unassigned | PostgreSQL/DATABASE_URL hardening with backup, rollback and credential rotation. | 2–4 h after assignment/authorization | Assign executor; authorize credential rotation and rollback. |
| ORQ-36 | Urgent | todo -> unassigned | Unassigned | MULTICA_TOKEN lifecycle and rollback design/execution. | 2–4 h after assignment/authorization | Assign executor; authorize token lifecycle mutation. |
| ORQ-37 | Urgent | todo -> blocked coordination | Navy_Seals squad | Wave-B leader task did not populate ORQ-33..36 workers; ORQ-33/34 are now manually staffed. ORQ-35/36 remain. | 15–30 min to re-plan after staffing | Approve proposed owners for ORQ-35/36 or let TL assign. |
| ORQ-38 | High | in_review -> acceptance | Unassigned / reviewed commits | Identifier collision fix pair is reviewed and ready for acceptance/integration. | 15–30 min plus CI availability | Authorize integration after CI path is available. |
| ORQ-39 | High | blocked -> implementation active | Codex56-Z | Implement six missing browser specs; workflow itself passed static review. | 1–2 h, then CI requires ORQ-26/billing | No immediate decision unless browser scope changes. |
| ORQ-40 | Low | blocked -> maintenance approval | Unassigned | Align Codex CLI versions with maintenance and rollback gate. | 30–60 min after authorization | Authorize maintenance window/package alignment. |
| ORQ-41 | High | todo -> code PASS, runtime dormant | Unassigned / W4 reviewed | Commit `e0b0155` is safe dormant; activation still needs LANE-DB, W2/W3 and deployment wiring. | Dormant integration 15–30 min; activation 4–8 h after dependencies | Authorize dormant merge separately from activation. |
| ORQ-42 | Urgent | todo -> in_review/BLOCK small fix | Opus48-A | `fc77e89` closed most tool blockers; re-review found exact-JSON verdict bug and one loose regression assertion. New commit/re-review required. Real rotation remains gated. | Code fix/re-review 30–60 min; rotation 1–2 h after owner window | Later authorize real metadata preflight, secret-safe window and rollback; no action for code fix. |
| ORQ-43 | High | todo -> integration BLOCK | Codex56-B / LANE-DB queued | Current stack exposes routes prematurely, lacks durable idempotency/cross-daemon proof and touched generated code outside SQLC lane. Redesign sequence S0–S4 required. | 90–150 min after DB predecessors and number reservation | No immediate action; later authorize token issuance/rotation behavior. |
| ORQ-44 | High | todo -> externally blocked design | Unassigned | Provider questions Q1–Q6: overlap, scope, introspection cost, AWSPREVIOUS, post-swap proof, exact secret identity/permissions. | No honest ETA; execution 1–2 h after answers | Answer provider/rotation questions and authorize window. |
| ORQ-45 | Low | todo -> scope missing | Unassigned | Zero-task validation card has no execution scope. | 15 min after decision | Close as obsolete or define exact validation objective. |
| ORQ-46 | Low | in_review -> acceptance | Navy_Seals squad | Priority triage completed; accept result. | 15 min | Approve closure. |

## Decisions currently required from the owner

| Urgency | Decision/action | Unlocks |
|---|---|---|
| P2 | Investigate the GitHub-reported billing lock if continued use of hosted Actions is desired. | Hosted CI only; it no longer blocks ORQ-26 because the equivalent local ephemeral-DB gate is active. |
| P0 | Confirm whether Tailnet-only live access satisfies ORQ-17. | ORQ-17 closeout. |
| P1 | Approve ORQ-16 and ORQ-46 closeout candidates. | Removes stale review cards. |
| P1 | Select Codex/Claude/Cline thinking-level policy, or authorize a conservative default proposal. | ORQ-20 and user-visible reasoning. |
| P1 | Approve or reject cascade deletion of obsolete linked runtimes. | ORQ-19. |
| P1 | Assign/authorize ORQ-35 and ORQ-36 security lanes. | Completes Wave-B staffing. |
| P1 | Authorize dormant-only merge of ORQ-41 separately from runtime activation. | Preserves safe code progress without claiming protection active. |
| P1 | Decide maintenance authorization for Codex CLI alignment. | ORQ-40. |
| P1 | Provide provider/owner answers and rotation authorization for ORQ-44; later provide exact metadata for ORQ-34/42. | Secret rotations. |
| P2 | Decide whether ORQ-11 and ORQ-45 still have business value. | Removes obsolete low-priority cards. |

## Capacity

- Nine Herdr executors plus the TL are currently monitored at 60-second intervals.
- The two newly enabled agents are active in the product and have been assigned read-only preflight work: `Opus-46#A` on ORQ-33 and `Opus-46-B` on ORQ-34.
- Active product task queue was zero at the cutoff; Herdr work is local/operational and does not necessarily appear as paid product tasks.

## Critical path

1. Stabilize ORQ-21 → integrate ORQ-13 migration 127 → implement ORQ-12 migration 128.
2. Complete the ORQ-26 local ephemeral-DB gate → unblock ORQ-18/38/39 without depending on GitHub-hosted Actions.
3. Finish ORQ-42 tool re-review before any JWT rotation.
4. Execute Wave-B only under secret-safe, rollback-approved windows.
