# ORQ-17 — Stage 3B initial owner credential provisioning runbook

- Status: **DESIGN / READ-ONLY**. No execution is authorized by this artifact.
- Prepared: `2026-07-27`.
- Current premise supplied by owner: users=`0`, password credentials=`0`.
- Safety skill: `.agents/skills/aws-secrets-manager/SKILL.md` read completely before design.
- Scope: create exactly one bootstrap user and one password credential. It does **not** create a
  workspace, member row, session, secret or owner-role assignment.

## Decision

Use a one-shot helper that calls the application’s generated `CreateUser` query and
`PostgresPasswordCredentialStore.ProvisionPassword` in one serializable transaction. Identity and
password enter only through NUL-delimited stdin inside an `asm-exec` child. The password is validated
and bcrypt-hashed by the existing server code before a parameterized hash write.

Never use `psql INSERT`, a plaintext SQL file, a password/hash argument, a shell-parent variable,
Docker `-e`, a temporary credential file, API login or browser session.

## Supported paths compared

| Path | Supported behavior | Stage 3B decision |
|---|---|---|
| `POST /auth/login` | Verifies an existing row through `PasswordAuthProvider`; cannot create the first user or credential (`auth_provider.go:115-131`). | Reject |
| Google login | Can call `findOrCreateUser`, but creates an OAuth/session flow and no password credential (`auth.go:610-725`). | Reject |
| Verification-code signup | Handler code can create a user and session, but the current router exposes only `/auth/login`, `/auth/google`, `/auth/logout`; it still would not create a password. | Reject |
| `PUT /api/me/password` | Uses the correct provisioner but requires an authenticated human and recent/current authentication (`auth.go:522-590`). With zero users it is unreachable. | Reject |
| `multica user password update --password-stdin` | Safely reads stdin, but merely wraps authenticated `/api/me/password` (`cmd_user.go:67-126`). | Reject for initial bootstrap; preferred later for rotation |
| Direct SQL | Could bypass validation, bcrypt cost, atomic duplicate gates and identity protections. | Prohibited |
| One-shot application helper | Uses `CreateUser`, `ValidatePassword`, bcrypt default cost, `ProvisionPassword`, and provider verification without minting a session. | **Selected** |

There is no platform-owner flag on `"user"`. Workspace ownership is a `member.role='owner'` created
by the authenticated `CreateWorkspace` flow. Stage 3B creates the bootstrap identity only. Creating
or attaching a workspace is a separate owner-authorized stage.

## Minimal human inputs

The human must provide only:

1. The approved real Secrets Manager secret ID or full ARN. It must be a JSON secret with keys
   `email` and `password` at `AWSCURRENT`.
2. Confirmation that the secret’s `email` field is the intended sole bootstrap identity and that no
   workspace membership is requested in Stage 3B.

Do **not** send the email, password, JSON value, version ID or hash. The password must be valid UTF-8,
12 or more characters, at most 72 bytes, and contain no NUL. The helper enforces the server policy.
Secret creation/rotation and granting resolver access are human-owned prerequisites, not part of
this runbook.

Derived references:

```text
{{resolve:secretsmanager:<SECRET_ID>:SecretString:email:AWSCURRENT}}
{{resolve:secretsmanager:<SECRET_ID>:SecretString:password:AWSCURRENT}}
```

## Gate 0 — immutable source and target

Run on ORQ1 only. Stop unless hostname/IP, Serve/Funnel, backend health and these ORQ1 source hashes
match:

```text
auth_provider.go                         3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857
pkg/db/generated/user.sql.go             a89cf9a2413df383930f6807812d932d0099075217405ad55949fe42cfa43a25
pkg/db/generated/db.go                   0c9302e79a855e76de414411e9c730a96b56c46782e65d19f7f4b919a7771799
125_user_password_credential.up.sql      bb6c01ac50e77e4a072c5fe7b30744c8f72a265297f3b4735de5847367143388
go.mod                                   e4122ce29e8e85b566f209ccbcefbe2450bdce2f97f0778a04e09c2f94b180d7
go.sum                                   5d3b288a17bae9685b103c183b5b2ad7fa440f25484de6fd8ece7aac7f91ece5
```

Required read-only checks:

```bash
hostname
tailscale ip -4
tailscale serve status --json
tailscale funnel status --json
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18080/health
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18080/readyz
```

Accept only ORQ1 `100.118.244.61`, Serve=`{}`, Funnel=`{}`, health/readiness=`200`.

## Gate 1 — private backup and helper build

Create a task-specific directory outside `/tmp`, mode `0700`, for example:

`/home/ec2-user/.local/state/orq17-stage3b-owner-bootstrap-<UTC>`

Before any DB write:

1. Record only backend/postgres IDs, image IDs, project/service labels, safe ports and source hashes.
2. Record three aggregate counts: `"user"`, `user_password_credential`, and `member`. Require
   `0|0|0` twice, three seconds apart.
3. Save a data-only `pg_dump` of those three tables at mode `0600`. Contents must never be printed;
   the expected baseline is empty.
4. Copy the reviewed helper and module templates from:
   - `.deploy-control/p0/evidence/orq17-stage3b-bootstrap-helper.go.txt`
   - `.deploy-control/p0/evidence/orq17-stage3b-helper-go.mod.txt`
5. Build in the private directory, never in the repository:

```bash
cd /home/ec2-user/.local/state/orq17-stage3b-owner-bootstrap-<UTC>
install -m 0600 <reviewed-helper-template> main.go
install -m 0600 <reviewed-go-mod-template> go.mod
GOWORK=off CGO_ENABLED=0 go build -trimpath -o orq17-stage3b-helper .
chmod 0700 orq17-stage3b-helper
```

Require helper architecture to equal the backend image architecture. Record filenames, modes and
SHA-256 only. Do not print backup or helper source during the operational window.

Suggested aggregate backup commands, executed by the human operator without exposing row data:

```bash
docker exec multica-dev-transition-postgres-1 \
  pg_dump -U multica_transition -d multica_transition \
  --data-only --no-owner --no-privileges \
  --table='public.user' \
  --table='public.user_password_credential' \
  --table='public.member' \
  > /home/ec2-user/.local/state/orq17-stage3b-owner-bootstrap-<UTC>/pre-state.sql
chmod 0600 /home/ec2-user/.local/state/orq17-stage3b-owner-bootstrap-<UTC>/pre-state.sql
```

## Gate 2 — reference and duplicate preflight

The human privately substitutes `<SECRET_ID>` in the two literal references. Do not place resolved
values in the parent environment. A child-only dry gate may emit only fixed statuses/length:

```bash
asm-exec -- env \
  OWNER_EMAIL='{{resolve:secretsmanager:<SECRET_ID>:SecretString:email:AWSCURRENT}}' \
  OWNER_PASSWORD='{{resolve:secretsmanager:<SECRET_ID>:SecretString:password:AWSCURRENT}}' \
  sh -eu -c '
    set +x
    test -n "$OWNER_EMAIL"
    test -n "$OWNER_PASSWORD"
    n=$(printf %s "$OWNER_PASSWORD" | wc -c)
    test "$n" -le 72
    printf "REFERENCES=RESOLVED PASSWORD_BYTES_OK=1\n"
  '
```

Stop on any resolver error, empty field or missing exact owner authorization. Never use
`GetSecretValue`, `gh`, `aws ... get-secret-value`, SMA loopback, `env`, `printenv`, `set -x`,
`docker inspect .Config.Env`, or command substitution into the parent shell.

## One-time provision

Use the five live Compose files from the Stage 2 backend label. The helper binary is mounted
read-only. Secrets are passed as NUL-delimited stdin; `env -u` prevents Docker Compose and the
one-shot container configuration from inheriting them:

```bash
asm-exec -- env \
  OWNER_EMAIL='{{resolve:secretsmanager:<SECRET_ID>:SecretString:email:AWSCURRENT}}' \
  OWNER_PASSWORD='{{resolve:secretsmanager:<SECRET_ID>:SecretString:password:AWSCURRENT}}' \
  sh -eu -c '
    set +x
    printf "%s\0%s\0" "$OWNER_EMAIL" "$OWNER_PASSWORD" |
      env -u OWNER_EMAIL -u OWNER_PASSWORD docker compose \
        --env-file /home/ec2-user/.config/multica-transition/dev.env \
        -p multica-dev-transition \
        -f /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml \
        -f /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml \
        -f /home/ec2-user/.config/multica-transition/images.yml \
        -f /home/ec2-user/.config/multica-transition/backend-env.override.yml \
        -f /home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml \
        run --rm --no-deps -T \
        -v /home/ec2-user/.local/state/orq17-stage3b-owner-bootstrap-<UTC>/orq17-stage3b-helper:/run/orq17-stage3b-helper:ro \
        --entrypoint /run/orq17-stage3b-helper \
        backend provision
  '
```

The only accepted output is:

```text
PROVISION=PASS USERS=1 CREDENTIALS=1 VERIFY=PASS SESSION=NONE
```

Internally, the helper:

- takes a serializable transaction and table locks;
- rechecks users=0 and credentials=0;
- normalizes and validates identity without printing it;
- calls generated `CreateUser`;
- calls the existing `ValidatePassword` and bcrypt-default-cost `ProvisionPassword`;
- verifies the password through `PasswordAuthProvider.Login` without HTTP/JWT/session;
- requires counts `1|1` before commit and again afterward;
- compensates back to `0|0` if post-commit verification fails.

A rerun fails closed with `E_NOT_EMPTY`; it cannot create a second user.

## Verification without identity/session exposure

After success, record only:

- aggregate users=`1`, password credentials=`1`, joined credential rows=`1`;
- bcrypt prefix/cost validity as booleans inside the helper, never the hash;
- backend `/health=200`, `/readyz=200`;
- anonymous `/api/me=401|403`;
- Serve/Funnel still `{}`;
- no persistent one-shot container;
- no session/token row or JWT was created by this runbook.

Do not verify with `/auth/login`: it would create and return a session token. The in-process
`PasswordAuthProvider.Login` check is sufficient and does not mint a JWT.

## Rollback

Rollback is automatic before commit. After a successful commit, automatic rollback is permitted
only in the same sealed window, before any browser/CLI login or workspace/member creation. Use the
same child-only transport and replace the final word `provision` with `rollback`.

Accepted output:

```text
ROLLBACK=PASS USERS=0 CREDENTIALS=0
```

Rollback requires exactly `1|1`, re-verifies the same secret-backed identity/password, requires zero
`member` references, and performs one parameterized user-ID delete; the password credential is
cascade-deleted. It then requires `0|0` before commit. No email, password, UUID or hash is printed.

If a member/workspace/session has been created, the rollback helper stops with
`E_ROLLBACK_REFERENCED`; do not cascade or improvise. Escalate for a new owner-approved data
retention decision.

## Stop conditions

Stop without writes on: wrong host; non-empty Serve/Funnel; health failure; source-hash drift;
users/credentials not exactly `0|0`; unresolved/empty reference; password-policy failure; helper
build/hash failure; unexpected Compose path/project/service; active task queue; output containing
identity/credential material; or missing explicit one-time authorization.

After any non-PASS helper exit, re-read aggregate counts. `0|0` means clean stop. `1|1` without the
exact PASS line means run the approved rollback immediately; any other shape is BLOCK/manual review.

## Minimal handoff requested

To make this runbook executable in a separately authorized turn, reply only with:

1. the approved Secrets Manager **secret ID or ARN** containing JSON keys `email` and `password`;
2. `confirmed: bootstrap user+password only; no workspace membership`.

Do not provide either secret value or the email itself.
