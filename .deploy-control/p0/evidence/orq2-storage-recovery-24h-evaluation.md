# ORQ2 storage recovery evaluation — objects older than 24 hours

- **Measured:** 2026-07-28T20:36Z
- **Mode:** read-only metadata audit
- **Host scope:** ORQ2 `/home/ec2-user`, `/tmp`, credential homes, task workspaces and Git worktrees
- **Content boundary:** no credential, token, cookie, database or secret value was opened or printed
- **Deletion performed:** none

## Executive result

The root filesystem is 60 GiB, 95% used, with 3.4 GiB available. `/tmp` has
6.9 GiB available and is not the pressure point.

At least **11.6 GiB** can be recovered conservatively after the currently running
agent tasks reach a terminal state, without deleting credential material:

| Class | Files older than 24 h | Recovery classification |
|---|---:|---|
| Regenerable caches inside credential slots (`go-build`, Go module cache, UV, npm cache and task GOCACHE) | 8.4 GiB | Low risk after active-task/open-file check |
| General `/home/ec2-user/.cache` content | 1.8 GiB | Low risk after active-task/open-file check |
| Completed ORQ-41/ORQ-42 private task caches | 1.5 GiB | Low risk; preserve evidence files and remove only named cache directories |
| **Conservative recovery** | **11.6 GiB** | Expected root availability: approximately **15 GiB**, approximately 76% used |

The 11.6 GiB figure is deduplicated across these three classes. It does not count
credential databases, tokens, SharePoint data, quarantine evidence, current worktrees
or task workspaces.

## Credential-home findings

`.agent-cred-homes` occupies 16 GiB:

- `slots/`: 15 GiB.
- `codex-logins/`: 1.4 GiB.
- Files older than 24 hours under the full credential root: 12.1 GB decimal.

The age figure alone is **not** a deletion predicate. Current daemon allowlists are:

- Kiro: slots 139, 140, 143 and 149.
- Antigravity: slots 141, 145 and 146.
- Codex: slot 152.

These allowlisted slots must not be removed. Their old cache subdirectories may be
cleaned only after no task/process is using them. The cleanup must remove whole named
cache trees or use the native cache tool; selectively removing individual files from
Go/npm/UV caches is prohibited because it can leave inconsistent cache metadata.

Non-allowlisted slots occupy 5.8 GiB in total, of which 2.8 GiB is older than 24
hours. This is an **alternative wholesale candidate**, not additive to the 8.4 GiB
cache figure. A slot may be retired only after checking every daemon/service allowlist,
the credential registry, active task references and rollback metadata.

Slot 140 remains protected. Its directory skeleton was restored, but the required
Kiro `data.sqlite3` authentication artifact is absent. No database or credential may
be fabricated, copied or linked from another slot.

## Additional conditional recovery

These amounts are not included in the conservative 11.6 GiB:

| Class | Old/total footprint | Required gate before removal |
|---|---:|---|
| Codex login snapshots | 1.4 GiB total; roughly 1.3 GiB potentially stale after preserving current/referenced logins | Prove no live process/session/reference and retain the current login plus rollback manifest |
| `multica_workspaces` task directories | 3.0 GiB of files older than 24 h | Task terminal, result/evidence promoted, no uncommitted-only deliverable, no active work directory |
| Git worktrees | 4.3 GiB total; 3.0 GiB of files older than 24 h | Branch integrated or explicitly abandoned, worktree clean, evidence retained; use `git worktree remove`, never raw recursive deletion |
| Remaining private task data including `p1-reasoning` | about 0.2 GiB old beyond the conservative completed-task set | Owner task terminal and reproducibility/evidence confirmed |

The first three conditional classes could add approximately 7.3 GiB, but only after
their individual reference and integrity gates. The theoretical combined recovery is
therefore around **18.9 GiB**, excluding any wholesale non-allowlisted-slot retirement
to avoid double counting.

## Explicit preservation set

The following are excluded from cleanup:

- `/home/ec2-user/sharepoint` and the uploaded SharePoint assessment data.
- Credential/token/database files and every current allowlisted slot.
- `.private-tmp/quarantine-20260727T123153Z`.
- `.private-tmp/agy-slot150-fix-20260727T175847Z`.
- Canonical `.deploy-control` evidence and check-outs.
- Dirty or unreviewed worktrees and branches.
- Any active Multica task work directory.
- Product Docker volumes, product databases and migration evidence.

## Recommended temporary cleanup sequence

1. Wait for the current two agent executions to become terminal and freeze new
   dispatch for the short cleanup window.
2. Recheck open files/processes and current daemon allowlists.
3. Clean only the named regenerable cache classes, one class at a time, measuring
   `df` before and after.
4. Remove completed ORQ-41/ORQ-42 private cache directories while retaining evidence.
5. Stop after the conservative target is reached; do not touch login snapshots,
   workspaces, worktrees or whole slots in the first pass.
6. Record exact paths, bytes recovered and rollback/rebuild behavior.

This first pass is expected to restore enough headroom for builds, race tests and
ephemeral database gates while the permanent lifecycle policy is completed.
