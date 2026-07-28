# P0 Kanban/Leader live execution — 2026-07-27

## Verdict

`LEADER_ORCHESTRATION_ACTIVATED_WITH_CONTRACT_GAP`

Authenticated owner operations used only the private session created by the prior `asm-exec` login. No credential, cookie value, token or secret was printed or copied into evidence.

## Code contract

- `project.lead_type` is constrained to `member|agent`. `CreateProject`/`UpdateProject` persist metadata and publish project events; they do not enqueue a Leader task.
- Executable Leader orchestration is issue/squad based: `assignee_type=squad` resolves `squad.leader_id` and enqueues `is_leader_task=true`.
- `/note` as the first token is stored as a human-only worklog and short-circuits all comment task triggers.
- Issue `start_date`/`due_date` are the live ETA fields; no separate worklog API/table exists.

## Live snapshot and configuration

- `2026-07-27T17:48:41Z`: owner `/api/me=200`; 2 projects; 28 nonterminal issues; 10 active idle agents; 3 online credential runtimes; active queue 0; waiting locks 0; no duplicate pending rows.
- ORQ2 project already had Leader Codex-A. The existing validation project was updated to the same valid Leader; queue remained `0→0`, empirically confirming project Leader is metadata only.
- `Navy_Seals` Leader is Codex-A, with eight additional ready agent members and one human member.
- ORQ-33..37 were attached to ORQ2 with `start_date=2026-07-27`, ETAs 2026-07-28 through 2026-07-30, and one `/note` each. Every note trigger-preview returned `agents=[]`.
- ORQ-37 alone was assigned to `Navy_Seals`. Exactly one initial `is_leader_task=true` task was admitted for Codex-A.

## Leader behavior observed

- Initial Leader task completed after asking for resource/rotation-gate clarification and made no assignments.
- An owner clarification was posted after trigger-preview identified exactly Codex-A; one follow-up Leader task was admitted.
- Follow-up stated it delegated ORQ-33..36, but the board assignees remained null. Instead, it created four normal worker tasks on ORQ-37 through a single multi-mention delegation comment.
- Worker outcomes at `2026-07-27T17:59:00Z`: Codex-D and Codex-C failed with revoked provider refresh-token access; Opus-46-B failed because credential slot `slot-150` was absent; Opus48-B remained running. Duplicate pending rows remained 0 and waiting locks remained 0.
- This is the exact contract gap: project Leader metadata does not orchestrate; squad Leader does run, but its delegation path used same-issue mentions rather than updating the real target-card assignees.

## Authorized AGY creation

Fresh authenticated exact-name GET returned zero matches before either POST. Runtime `405b751d-e831-4da3-8fd5-bb3744c49334` was public and online.

- Agy-P0-A7: `780104f1-1ff4-4292-8207-44b9ac4f5fca`
- Agy-P0-A8: `5b336637-bfb3-43ce-9bcf-17565baa2993`
- Both: runtime exact match; `visibility=workspace`; `status=idle`; `archived_at=null`; model `gemini-3.6-flash-high`; `max_concurrent_tasks=6`; `thinking_level` omitted on POST and empty on GET-back.

## Ownership stop and restoration

A late steering message identified existing ownership/commits on ORQ-38 and ORQ-26 after assignments had already been sent. Both admitted tasks failed before `started_at` because required credential directory `/home/ec2-user/.agent-cred-homes/slots/slot-150` did not exist. No retry was attempted.

All mutable effects on those cards were then reverted:

- AGY assignees removed.
- Original project/date state restored (ORQ-26 retained its original ORQ2 project; ORQ-38 returned to no project).
- Two Leader-assignment `/note` comments removed.
- Two automatic credential-isolation failure comments removed.
- Final ORQ-26 and ORQ-38 state: no assignee, no start/due dates, zero comments.
- A7/A8 final state: idle, not archived, zero active tasks.

Task failure rows remain in terminal history for auditability. No smoke/synthetic task was created. No bundle with stale SHA `4d922b25` was posted.

## Final safety state — 2026-07-27T17:59:00Z

- Duplicate pending rows: 0.
- Waiting locks: 0.
- A7/A8 active tasks: 0.
- A7/A8 remain available for a future Leader only after explicit issue/file ownership exclusivity is recorded and credential slot configuration is valid.
