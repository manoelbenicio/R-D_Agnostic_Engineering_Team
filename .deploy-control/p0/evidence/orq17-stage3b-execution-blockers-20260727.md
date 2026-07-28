# ORQ-17 Stage3B — execution gate outcome and blockers

- **UTC:** 2026-07-27T17:10:55Z–17:11Z
- **Execution lead:** Kiro
- **State:** `BLOCKED_FAIL_CLOSED`; `BOARD_SYNC_PENDING`; no board/API post attempted.
- **Secret safety:** loaded `aws-secrets-manager`; zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA direct access, zero secret value in agent output/argv/file.

## Milestones completed

1. Gate 0 tool/permission preflight delivered and accepted local-only:
   `.deploy-control/p0/evidence/orq17-stage3b-gate0-tool-permission-preflight-20260727.md`.
2. CloudFormation template prepared for dedicated rotating KMS key plus
   `prod/multica/bootstrap-owner`, with a 32-character server-side generated password and exact JSON
   keys `email,password`: `.deploy-control/p0/evidence/orq17-bootstrap-owner-secret-stack.yaml`, SHA-256
   `7c6d2b547a071cd8d7ad06437a80055a3879ee2764caa4fc45cccaf68a60c582`.
3. `cfn-lint 1.46.0`, region `sa-east-1`: exit 0, 0 findings.
4. Final V2 package pins measured:
   - helper `fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82`;
   - runner `c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42`;
   - closure `719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1`;
   - candidate binary `1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b`.
5. Opus48#A F1–F5 PASS evidence is local at
   `.deploy-control/p0/evidence/orq17-stage3b-f1-f5-final-verification.md`, SHA-256
   `9113f2a7760fa2702344cdb97fd70ea42e50c1dca7e66a53ce2eefc839e829c3`; Owner steering declares two
   independent PASSes and technical gate cleared for those exact pins.
6. Pinned `asm-exec` staged but never executed on ORQ1 at
   `/home/ec2-user/.local/state/orq17-stage3b-v2-tools/asm-exec`, `ec2-user:ec2-user 0700`, SHA-256
   `d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556`.
7. ORQ1 remains healthy: frontend `13100=200`, backend health/readiness `200`, Serve `{}`, Funnel `{}`,
   active task queue `0`.

## Hard blockers found before mutation

### B1 — owner AWS profile is absent

The required `owner-p0` profile was checked twice with `sts:GetCallerIdentity`; both attempts failed
before any AWS request with: `The config profile (owner-p0) could not be found`. Per Owner direction,
there was no fallback to the EC2 instance role.

An earlier explicitly authorized instance-role `CreateStack` attempt failed at IAM authorization:
`cloudformation:CreateStack` denied on
`arn:aws:cloudformation:sa-east-1:809809509961:stack/prod-multica-bootstrap-owner/*`; no stack/resource
was created. It was not retried after the `owner-p0 only` ruling.

**Exact unblock:** configure/authenticate profile `owner-p0` on ORQ2 and prove
`sts:GetCallerIdentity` equals account `809809509961` and `arn:aws:iam::809809509961:user/dataops.cloud.mbf`.

### B2 — live database no longer satisfies the approved virgin-database contract

Read-only aggregate checks produced:

```text
users=1 credentials=0 members=1 active_queue=0
owner_email_matches=0 owner_credentials=0 owner_role_memberships=0
```

No existing identity, UUID, hash or DSN was printed. The one existing user/member is not the authorized
owner login. The approved helper requires `0|0|0` and must fail with `E_NOT_EMPTY`; executing it would
not provision the owner. Manual DELETE/direct SQL is prohibited and was not attempted.

**Exact unblock:** General-TL/Owner must choose and independently review one safe path while preserving
the existing non-owner user/member: either (a) an application-supported second-user bootstrap package
with exact duplicate/membership/rollback guards, or (b) a separately owner-authorized, application-safe
lifecycle decision for the existing user/member. No direct deletion or relaxation of V2 gates.

## Non-actions

- No KMS key, secret, stack, secret version, database row, login/JWT, Tailscale Serve mount, project,
  issue, assignment, paid agent task, board note or board transition was created.
- No secret value API was invoked and no credential value was observed.
- Seven Serve mounts remain intentionally unapplied because owner bootstrap/login gates did not pass.

## 2026-07-27T17:17:01Z — Stack-success handoff and mandatory Stage3B stop

- Owner reported P0 stack success and provided metadata-only secret ARN `arn:aws:secretsmanager:sa-east-1:809809509961:secret:prod/multica/bootstrap-owner-3BIwjo`.
- The local final CloudFormation template independently matches reported SHA-256 `ba967fc4b56f357fc766fd7ecce44a6ae16f4522b1cb79fc2fceed43be53ce09`.
- Immutable pins reconfirmed before any resolution: helper `fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82`, runner `c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42`, closure `719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1`, and ORQ1 `asm-exec` `d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556` mode `0700`.
- Corrected read-only helper-schema aggregate returned `counts=1|0|1|0` (`users|user_password_credential|members|active_queue`) and owner match `0|0|0`.
- The approved runbook requires `0|0|0` before secret resolution. It also requires pre-authorized private mode-`0600` custody and one-shot authorization receipts bound to the full-ARN hash and private `AWSCURRENT` fingerprint; filename/mode-only inventory found none.
- Therefore the exact approved package was not invoked. For the observed state, its reducer classification is `E_PROVISION_FAILURE_STATE_REQUIRES_OWNER_REVIEW`; forcing invocation or fabricating receipts would violate the approved package.
- Secret values were never requested, resolved, printed, logged, or written. No `GetSecretValue`/`BatchGetSecretValue`, direct Secrets Manager Agent access, DB mutation, HTTP login, Serve change, Kanban API call, comment, assignment, or paid execution occurred.

## 2026-07-27T17:18:47Z — Content-free ADOPT predicate result

A new owner directive authorized read-only evaluation of an adopt-existing path, conditional on all predicates passing. No row values, IDs, emails, workspace names, tokens or secrets were read or emitted.

- Global aggregate: `1 user | 0 password credential | 1 member | 1 workspace | 0 active queue`.
- Accepted-role predicate: `1 total | 1 owner-or-admin | 0 invalid`; live schema has one role check accepting both `owner` and `admin`.
- Membership integrity: `1 total | 1 resolving to an existing user and workspace | 0 orphan`.
- Target-email conflict count: `0`.
- Live single-column user FK catalog count: exact `16`, matching the approved helper catalog.
- Per-target FK counts in immutable helper order: `10|20|5|191|0|0|0|1|0|2|0|45|9|0|0|0`.
- Required predicate “all user FK counts zero except member” **failed**. Nonzero references exist for `agent.archived_by`, `agent.owner_id`, `agent_runtime.owner_id`, `chat_session.creator_id`, `member.user_id`, `personal_access_token.user_id`, `skill.created_by`, and `task_token.user_id`.
- Because preparation was explicitly conditional on all predicates passing, no ADOPT helper/runner was created and no existing package was changed. No secret resolution or mutation occurred.
