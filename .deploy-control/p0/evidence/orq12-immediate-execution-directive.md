# ORQ-12 — Immediate execution directive

Evidence already established: `task_usage` needs the immutable producing-account
snapshot implemented and tested by the ORQ-12 stack. The solution is already
implemented in the clean worktree and passed the combined staged-schema gates.

Execute the remaining delivery work now. Do not restart design or request another
general preflight.

1. Own the existing clean worktree
   `/home/ec2-user/workspace/worktrees/orq12-account-id`, branch
   `agent/opus48-a/orq12-task-usage-account-id`, current expected HEAD `dd95e0a`.
2. Perform a fresh repository-wide collision scan. If migration number `128` is
   still free, bind the staged
   `NEXT_CANONICAL_task_usage_account_id.{up,down}.sql` files to the definitive
   `128_task_usage_account_id.{up,down}.sql` names. If `128` is occupied by an
   unrelated migration, stop and report the exact collision; do not choose a new
   number independently.
3. Remove the temporary staged-only SQLC configuration if it is no longer needed,
   run SQLC v1.31.1 from the canonical configuration, and accept generated changes
   only from the generator. Never hand-edit generated files.
4. Run the final disposable-PostgreSQL gates: migration up/down/up, ORQ-12 matrix,
   ORQ-21 registry/handler compatibility, race tests, build, vet, gofmt and
   anti-false-green nominal test counting. Do not touch production data.
5. If green, create one new code commit without amend or history rewrite and report
   its SHA, exact changed-file list, test totals, skips/failures and teardown proof.
6. Leave the card ready for an independent review dispatched separately through
   Kanban. Do not activate another agent yourself and do not use Herdr.

Stop only for a real migration-number collision, overlapping file ownership, or a
reproducible technical failure. Tool cost is not a blocker. No push, merge, deploy,
production mutation, secret read or destructive cleanup is authorized by this task.

Acceptance: migration 128 has definitive filenames; canonical generated code is
reproducible; the disposable DB up/down/up and combined ORQ-12/ORQ-21 tests are
green with nominal execution evidence; worktree is clean on a new commit; no
production system was mutated.
