# GTL execution authorization — fix the ORQ-12 reclaim integration defect now

This instruction supersedes the stale statement that the executor is released
from ORQ-12 writes. Opus48-A is the authorized sole writer for the existing clean
ORQ-12 worktree and its locked files for this correction.

The combined gate found one real defect: the ORQ-12
`ReclaimStaleDispatchedTaskForRuntime` candidate selection excludes covered-provider
rows that are already `dispatched` with `credential_account_id IS NULL`. Those are
exactly the pre-migration in-flight rows. Filtering them out prevents ORQ-21's
fail-closed claim/cancel path from observing and cancelling them.

Execute, do not redesign:

1. In `/home/ec2-user/workspace/worktrees/orq12-account-id` at `dd95e0a`, change
   reclaim candidate selection so these legacy NULL-snapshot rows remain selectable
   and reach the existing fail-closed cancellation path. Do not invent a live
   assignment fallback and do not assign a new account during reclaim.
2. Add focused regression tests proving a covered-provider dispatched legacy NULL
   row is observed and cancelled/fails closed, while a row with a frozen account
   preserves that snapshot and reclaims normally.
3. Commit the correction as a new commit, without amend.
4. Re-run the combined disposable-PostgreSQL gate with the `orq21db` build tag,
   `ORQ21_TEST_DATABASE_URL`, nominal test discovery/counting, race, build, vet,
   gofmt and migration up/down/up. No zero-test success is acceptable.
5. If the combined gate is green, perform the fresh collision scan, bind the staged
   migration to definitive number 128, regenerate canonical SQLC output, rerun the
   final gates and create the final promotion commit.

Stop only for a reproducible technical failure, actual migration-number collision,
or overlapping writer. No production mutation, push, merge, deploy, secret read or
destructive cleanup. Report exact commits, changed files and test totals.
