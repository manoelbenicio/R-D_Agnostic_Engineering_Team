# ORQ-18 — Local commit evidence and independent peer-review request

- Date: 2026-07-28
- Author lane: Codex56-Z (`w8:p4`)
- Status: **REVIEW REQUESTED — no author verdict**
- Code worktree: `/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/91e70c79/workdir/repo`
- Branch: `agent/codex-b/orq-18-runtime-delete-ui`
- Commit: `a943a3fbfca8a8d1709e01152740ad5d424c8e4e`
- Parent: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
- Subject: `feat: expose runtime delete action safely`
- Delivery constraints observed: local commit only; no amend, push, merge, board mutation, or live runtime deletion.

## Exact committed scope (`FILES_LOCKED` only)

1. `multica-auth-work/packages/views/runtimes/components/runtime-list.tsx`
2. `multica-auth-work/packages/views/runtimes/components/runtime-row-menu.test.tsx`
3. `multica-auth-work/packages/views/runtimes/components/delete-runtime-dialog.test.tsx`
4. `multica-auth-work/packages/core/runtimes/mutations.test.tsx` (new)

Commit summary: 4 files changed, 361 insertions, 107 deletions. The commit has one parent and contains no `.deploy-control` or board path.

## Pre-commit ownership/integrity evidence

- Branch/HEAD before commit: `agent/codex-b/orq-18-runtime-delete-ui` at `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`.
- Upstream: none.
- Stage before explicit add: empty.
- Dirty set: exactly the four paths above.
- Scan across every registered worktree found locked-path changes in exactly one worktree: this ORQ-18 worktree.
- `git diff --check`: PASS.
- SHA-256 of received/preserved file contents:
  - `runtime-list.tsx`: `387fefe1791fa4294cf6b71154fdc3cab893429f24921223258f69aee77dea5f`
  - `runtime-row-menu.test.tsx`: `5ff47bdd7b1265c1d5e2a2fcbc7d38522099575d945635a12abd187442ea002a`
  - `delete-runtime-dialog.test.tsx`: `63c40b4135c29db45cc4a90515e91c832c0ca47ab7e3a5eca86cc4b9c12498f1`
  - `mutations.test.tsx`: `422fffae51eb69426610d9d57003b4983cc9d5216c3238cca78a8cb727bbe025`

## Validation evidence

Sanctioned package-local Vitest harnesses, post-commit:

- `packages/views`: `runtime-row-menu.test.tsx` 8/8 and `delete-runtime-dialog.test.tsx` 15/15; **23/23 PASS**, exit 0, duration 4.61s.
- `packages/core`: `runtimes/mutations.test.tsx` **4/4 PASS**, exit 0, duration 2.49s.
- Package-local TypeScript checks before commit, on the identical content: Views PASS; Core PASS.
- Targeted ESLint before commit, on the identical content: the three Views files PASS; Core mutations test PASS.
- `git diff HEAD^ HEAD --check`: PASS.
- Post-commit code worktree: clean; stage empty; upstream none.

Behavioral assertions retained include: visible authorized delete affordance; hidden unauthorized affordance; confirmation handoff and success toast; light/cascade cancel; successful delete callback; generic non-409 error keeps the dialog open; pending disables confirm/cancel; strict-delete `runtime_has_active_agents` pivots to cascade; `runtime_delete_plan_changed` refreshes the plan and requires reconfirmation; runtime/agent/task-query invalidation on both success and error settlements.

## Independent peer-review request

Please review commit `a943a3fbfca8a8d1709e01152740ad5d424c8e4e` independently against parent `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` and return PASS/BLOCK without modifying the author branch.

Requested review points:

1. Confirm exact four-path ownership/scope and absence of unrelated changes.
2. Confirm direct delete affordance remains authorization-gated and still routes through the single confirmation dialog.
3. Confirm error, pending, 409 pivot, cascade re-prompt, success, and cache-refetch/invalidation coverage are meaningful rather than false-green.
4. Reproduce the 23 Views tests and 4 Core tests from the package-local Vitest configs.
5. Check TypeScript/ESLint implications and identify any missing browser-level or integration evidence.
6. Verify no live runtime deletion is required for acceptance.

This document is an author-produced factual packet and review request, not an independent acceptance verdict.
