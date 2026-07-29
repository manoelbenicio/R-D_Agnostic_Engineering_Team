# Integration Push-Scope Hygiene Audit

**Snapshot HEAD:** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
**Timestamp:** 2026-07-18T17:38:00Z

## Objective
Read-only hygiene verification extending the integration push-scope matrix. Checks for whitespace violations, hazardous paths, generated/backup files, and incorrectly staged files.

## Commands Executed
- `git rev-parse HEAD`
- `git diff --check`
- `git diff --cached --check`
- `git status --porcelain`

## Hygiene Findings

### 1. Whitespace Violations (`git diff --check`)
The following unaccepted file contains whitespace errors (trailing whitespace and new blank line at EOF):
- `multica-auth-work/server/internal/handler/chat_test.go` (lines 474, 477, 512)
*Note: This file belongs to the rejected Chat Orchestration tasks and is already excluded from the push matrix.*

### 2. Hazardous Paths (Case/Path Hazards)
The following files are Windows path hazards and should NOT be staged or committed:
- `multica-auth-work/NUL`
- `multica-auth-work/server/NUL`
- `nul`

### 3. Generated / Backup / Vendor Files
The following generated or backup files are present in the worktree and should NOT be staged:
- `files.txt`
- `opencode.json`
- `opencode.json.backup.20260718-110331`

### 4. Staged Index Violations
The following files are currently **staged** (`git add` was run) but belong to pending or rejected lanes (Native Onboarding / Vendor UI / Packet B). They **MUST be unstaged** before atomic commits:
- `multica-auth-work/packages/core/api/client.ts` (Modified)
- `multica-auth-work/packages/core/runtimes/models.test.tsx` (Added)
- `multica-auth-work/packages/core/runtimes/models.ts` (Modified)
- `multica-auth-work/packages/core/types/agent.test.ts` (Added)
- `multica-auth-work/packages/core/types/agent.ts` (Modified)
- `multica-auth-work/packages/views/agents/components/inspector/model-picker.test.tsx` (Added)
- `multica-auth-work/packages/views/agents/components/inspector/model-picker.tsx` (Modified)
- `multica-auth-work/packages/views/agents/components/model-dropdown.test.tsx` (Added)
- `multica-auth-work/packages/views/agents/components/model-dropdown.tsx` (Modified)
- `multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx` (Added)
- `multica-auth-work/packages/views/agents/components/runtime-picker.tsx` (Modified)

## Conclusion
The principal must:
1. `git restore --staged` the incorrectly staged `packages/` files.
2. Ensure hazardous files (`NUL`, `nul`) and backups (`files.txt`, `opencode.json*`) are excluded.
3. Exclude `chat_test.go` from commits, neutralizing its whitespace violations.
4. Proceed with atomic commits using the accepted files listed in `integration-push-scope-matrix.md`.
