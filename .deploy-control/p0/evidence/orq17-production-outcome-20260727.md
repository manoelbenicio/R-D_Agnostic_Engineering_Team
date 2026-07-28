# ORQ-17 production outcome — 2026-07-27

## Verdict

`ORQ17_PRODUCTION_OUTCOME_PASS`

Owner adoption, contained login, seven canonical Tailscale Serve routes, HTTPS/UI health, and authenticated zero-task Kanban validation completed. No secret value was printed, logged, returned to agent context, placed in argv, or written to plaintext evidence. No `GetSecretValue`/`BatchGetSecretValue` tool or direct Secrets Manager Agent access was used.

## Immutable inputs

- Final CloudFormation template SHA-256: `ba967fc4b56f357fc766fd7ecce44a6ae16f4522b1cb79fc2fceed43be53ce09`.
- ADOPT helper source SHA-256: `04cfc15f39c33c57da5d58dd91e6d96c6ab1d312d785432a9323f86f09f5519f`.
- Atomically promoted ADOPT binary SHA-256: `f6f1870b16cedc8c38f52ecfc2c54a9c88f9c35810a14d27247fd59f8a700adb`.
- Sealed runner SHA-256: `cf820f9f828d4a0e63ee08a178df2304fe7ec11f611914ad521dd2561128d2d9`.
- Pinned `asm-exec` SHA-256: `d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556`.
- ORQ1 private state: `/home/ec2-user/.local/state/orq17-stage3b-adopt-20260727T171900Z`, mode `0700`.
- Transactional affected-row backup: `adopt-backup.json`, mode `0600`; contents never printed or read into agent context.

## Milestones

1. `2026-07-27T17:18:47Z`: content-free predicates proved `1 user | 0 credential | 1 valid owner/admin member | 1 workspace | 0 active queue`, target-email conflict `0`, valid membership relation, exact user-FK catalog count `16`.
2. Owner explicitly ruled existing nonzero FK references expected and required sole-user UUID preservation. ADOPT changed only the sole user email and inserted one password credential; it did not delete or modify member/workspace/Kanban/FK rows.
3. Helper used a serializable transaction, `user`/`user_password_credential`/`member` locks, an independent queue-admission freeze, product `PostgresPasswordCredentialStore.ProvisionPassword`, affected-row backup, postcondition checks, and compensating rollback limited to credential deletion plus prior-email restoration.
4. Initial `asm-exec` attempt stopped before child launch on resolver failure. Metadata-only STS proved ORQ1 identity `arn:aws:sts::809809509961:assumed-role/cw-agent-orquestradores/i-0d9d441dd364039f9`; one retry with explicit `sa-east-1` succeeded. No helper or DB mutation occurred on the failed attempt.
5. Fixed execution token: `ORQ17_STAGE3B_ADOPT=PASS LOGIN=200 ME=200`.
6. `2026-07-27T17:34:54Z`: post-state `1 user | 1 credential | 1 member | 1 workspace | 0 active queue`; owner match `1|1|1`; backend health/readiness `200`; frontend `200`; anonymous local `/api/me=401`.
7. `2026-07-27T17:34:54Z`: exactly seven Serve handlers applied:
   - `/auth/login` → `http://127.0.0.1:18080/auth/login`
   - `/auth/google` → `http://127.0.0.1:18080/auth/google`
   - `/auth/logout` → `http://127.0.0.1:18080/auth/logout`
   - `/ws` → `http://127.0.0.1:18080/ws`
   - `/api` → `http://127.0.0.1:18080/api`
   - `/uploads` → `http://127.0.0.1:18080/uploads`
   - `/` → `http://127.0.0.1:13100/`
8. `2026-07-27T17:35:32Z`: Tailscale HTTPS root `200`; anonymous HTTPS `/api/me=401`.
9. `2026-07-27T17:37:34Z`: authenticated `POST /api/projects=201`, project `abe3c461-c921-4a51-b91a-b08529429145`; authenticated unassigned `POST /api/issues=201`, issue `bc633226-ab08-42fe-8aba-93eba2cdd616`; active queue `0→0`; `active-task` empty; task runs `0`.
10. `2026-07-27T17:38:08Z`: authenticated GET-back project/issue `200`, issue assignee fields null, seven-route mapping exact, no helper container, backend/frontend/HTTPS root `200`, anonymous API `401`, active queue `0`.

## Rollback

- ADOPT rollback remains available in the private package and is intentionally limited to deleting the newly inserted credential and restoring the backed-up prior email while preserving UUID/member/workspace/all FK rows.
- Serve rollback: `tailscale serve reset`.
- No rollback was required because all forward and final acceptance gates passed.

## Scope exclusions

No comment, mention, agent assignment, paid task, full test suite, browser matrix, race suite, CI run, direct board DB write, auth bypass, or unrelated backlog action occurred.
