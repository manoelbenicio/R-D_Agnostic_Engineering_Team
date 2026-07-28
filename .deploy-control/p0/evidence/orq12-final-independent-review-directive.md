# ORQ-12 — final independent review directive

Review only the final ORQ-12 delta on branch
`agent/opus48-a/orq12-task-usage-account-id`, HEAD
`ea1eee725fd49bd6bded94e2f1fe2c2257fd5f18`, including correction commit
`785a8ace60307ea92ad02ae9d4e5024b715c270a` and migration-promotion commit
`ea1eee725fd49bd6bded94e2f1fe2c2257fd5f18`.

This is an independent final review, not implementation and not a redesign.

Verify:

1. Migration `128_task_usage_account_id.{up,down}.sql` is collision-free,
   reversible and canonical; no staged placeholder or temporary SQLC config remains.
2. Legacy covered-provider dispatched rows with a NULL frozen account stay visible
   to reclaim and reach ORQ-21's fail-closed cancellation path; reclaim never
   resolves or invents an account.
3. A row with an existing frozen account preserves that snapshot on reclaim.
4. ORQ-12 must integrate together with the compatible ORQ-21 stack; explicitly
   reject any sequence that would land ORQ-12 alone and weaken fail-closed behavior.
5. Reproduce the focused disposable-PostgreSQL matrix with nominal test discovery:
   migration up/down/up, ORQ-12 tests, ORQ-21 handler and `orq21db` registry tests,
   race, build, vet, gofmt and zero skipped/zero-test false green for required tests.
6. Confirm the worktree is clean and generated SQLC output is reproducible without
   hand edits.

Return one binary verdict: PASS for the exact integration stack, or BLOCK with the
specific reproducible defect and smallest safe correction. Do not edit the author's
files, do not amend, push, merge, deploy or mutate production. Do not activate
another agent and do not use Herdr.
