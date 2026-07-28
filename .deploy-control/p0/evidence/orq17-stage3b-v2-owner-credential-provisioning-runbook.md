# ORQ-17 Stage3B V2 — sealed initial-owner credential runbook

- Status: **PROPOSAL — BLOCKED pending independent PASS and a new one-time owner authorization**.
- Scope: create exactly one `user` for `dataops.cloud.mbf@gmail.com` and one
  `user_password_credential`; create no workspace,
  `member`, session or token.
- This document was prepared without resolving a secret, contacting AWS/provider, reading a DSN,
  querying/writing a DB, running Docker/Compose, changing Serve/Funnel, or mutating runtime/board.
- Authority: `.agents/skills/aws-secrets-manager/SKILL.md`, loaded completely. Never use
  `GetSecretValue`, `BatchGetSecretValue`, direct secret CLI/SDK retrieval, or plaintext SQL.

## 1. Exactly two human inputs

1. A full Secrets Manager ARN of the form
   `arn:aws:secretsmanager:sa-east-1:<12-digit-account>:secret:<name>-<suffix>`. Bare names and
   non-`sa-east-1` ARNs are rejected. It must hold JSON string keys `email` and `password` at one
   stable `AWSCURRENT` version. Do not send either value, JSON, identity, hash or version ID to an
   agent/chat.
2. One exact, expiring authorization:

```text
AUTH_ID=<unique>; EXPIRES_UTC=<time>; authorize exactly one ORQ-17 Stage3B V2 provision on
ip-172-31-18-217.sa-east-1.compute.internal / 100.118.244.61, Compose project
multica-dev-transition service backend, PostgreSQL postgres:5432/multica_transition as
multica_transition, using the privately recorded single AWSCURRENT version fingerprint;
create user+password credential only, no workspace/member/login; rollback only inside the same
sealed window under V2 gates.
```

The authorization receipt is mode `0600`, consumed atomically once, and bound to the SHA-256 of the
full ARN plus the private version-ID fingerprint. Missing, expired, reused or differently scoped
authorization is `STOP`. Secret creation, rotation, IAM changes and permission grants are separate
owner actions and outside Stage3B. Their human-owned, metadata-only prerequisite receipt must state
`SECRET_EXISTS=1 JSON_KEYS_ATTESTED_BY_OWNER=1 RESOLVER_PERMISSION_PREVALIDATED=1 ROTATION_FROZEN=1`;
it points to the owner change record but contains no identity, value, ARN or version ID.

Before `provision`, that receipt must also state
`CUSTODY_PROVEN=1 LOGIN=dataops.cloud.mbf@gmail.com`: the real password remains recoverable by the
owner from the approved reference and was not copied to a terminal, ticket or agent. The runner
verifies the private `0600` receipt and its pre-authorized SHA-256 before mutation.

## 2. Immutable package

The executable package is:

- `orq17-stage3b-v2-bootstrap-helper.go.txt`
- `orq17-stage3b-v2-bootstrap-helper_test.go.txt`
- `orq17-stage3b-v2-helper-go.mod.txt`
- `orq17-stage3b-v2-helper-go.sum.txt`
- `orq17-stage3b-v2-sealed-runner.sh.txt`
- `orq17-stage3b-v2-source-closure.sha256`
- `orq17-stage3b-v2-build-evidence.md`

All expected hashes and the candidate binary hash are in the build evidence. Before an operational
window, an independent reviewer must:

1. make a private read-only source snapshot;
2. verify all 1,696 closure entries and the helper/module/runner/asm-exec hashes;
3. build twice with Go `1.26.1` executable
   `548e61b2d08ae52043be2f1924ed3c1d2b2c41967e360f3e317667f6fa912fc2`,
   `GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0`, private `HOME/GOCACHE/GOMODCACHE`,
   `-trimpath -buildvcs=false -ldflags=-buildid=`;
4. require both binaries equal
   `1710e01001440c82e12a4326d18d49e8905d72ea6b97dabe5d482a24934af87b`;
5. re-run the 1,696-entry closure check after build and require zero failures.

No mutable absolute source replacement is accepted. The approved binary is installed mode `0700`
in the task-specific rollback directory; no build occurs during secret resolution.

## 3. Exact ORQ1/stack gate, before secret resolution

Create `/home/ec2-user/.local/state/orq17-stage3b-v2-<UTC>` at `0700`, with files `0600`. Prove:

```text
hostname -f                 ip-172-31-18-217.sa-east-1.compute.internal
tailscale IPv4              100.118.244.61
tailscale serve status      {}
tailscale funnel status     {}
127.0.0.1:18080/health      200
127.0.0.1:18080/readyz      200
Compose version             5.3.1
project/service             multica-dev-transition/backend
DB tuple inside helper      postgres:5432/multica_transition, user multica_transition
```

Read safe Docker labels only, never `Config.Env`. The backend label must give these five paths,
in this exact order, and no sixth:

1. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`
2. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml`
3. `/home/ec2-user/.config/multica-transition/images.yml`
4. `/home/ec2-user/.config/multica-transition/backend-env.override.yml`
5. `/home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml`

Record only container/image IDs, image digest, labels, ports, file names/modes and checksums. Copy
the five Compose files and existing `dev.env` into the private rollback set without opening or
printing them; mode `0600`; verify checksums. Save Serve/Funnel JSON. No container is recreated.

Make a mode-`0600`, data-only `pg_dump` of `public.user`,
`public.user_password_credential` and `public.member`, redirecting directly to the private file.
The preceding aggregate baseline must be `users=0 credentials=0 members=0`; never print dump
contents. Preserve it for manual recovery only—V2 rollback does not blindly restore SQL.

## 4. Metadata-only secret/version gate

The human operator may call only metadata operations `DescribeSecret` and
`ListSecretVersionIds`; no value-returning operation. Validate mechanically and write private
metadata only:

- returned ARN equals the full input ARN byte-for-byte;
- ARN region is exactly `sa-east-1`, with explicit `AWS_REGION=sa-east-1`;
- exactly one version ID has `AWSCURRENT`;
- no rotation occurs from authorization through provision/rollback and cleanup;
- the single version ID before resolution equals the version ID after the final DB gate.

Store the version ID only in a `0600` private file. Public evidence records
`FULL_ARN_MATCH=1 REGION_MATCH=1 CURRENT_COUNT=1 VERSION_STABLE=1`, never ARN, name, version ID or
secret fields. If `AWSCURRENT` changes, rollback is not attempted with a different password:
freeze admission, retain the row, and escalate.

Dynamic references use the full ARN and `AWSCURRENT`:

```text
OWNER_EMAIL={{resolve:secretsmanager:<FULL_ARN>:SecretString:email:AWSCURRENT}}
OWNER_PASSWORD={{resolve:secretsmanager:<FULL_ARN>:SecretString:password:AWSCURRENT}}
```

## 5. Sealed transport and one-time execution

There is no separate resolve/dry run. On the sealed ORQ1 host, no other process may run under the
operator UID; shell tracing, process inspection, `env`, `printenv`, core dumps and diagnostic
collectors are disabled. This mitigates the residual same-UID/root visibility of the asm-exec child
environment through `/proc/<pid>/environ`.

Place only the **unresolved** references in asm-exec's incoming environment:

```bash
OWNER_EMAIL='{{resolve:secretsmanager:<FULL_ARN>:SecretString:email:AWSCURRENT}}' \
OWNER_PASSWORD='{{resolve:secretsmanager:<FULL_ARN>:SecretString:password:AWSCURRENT}}' \
ORQ17_HELPER_BIN='<private-approved-binary>' \
ORQ17_PRIVATE_STATE='/home/ec2-user/.local/state/orq17-stage3b-v2-<UTC>/run' \
/path/to/pinned/asm-exec -- \
  /bin/sh '<private-reviewed-runner>' provision
```

The command arguments contain no identity/password and only static paths/mode. asm-exec resolves
the two variables into the child shell environment. The runner immediately performs child-only
`printf '%s\0%s\0'`, pipes NUL-delimited bytes to helper stdin, and executes Compose behind
`env -u OWNER_EMAIL -u OWNER_PASSWORD`. Neither Docker argv, Compose argv, container config nor
parent environment receives plaintext. The helper validates input length/policy and best-effort
clears byte buffers; Go string copies are blanked on return, but perfect runtime-memory erasure is
not claimed.

Before any credential expansion, the runner proves `printf` is a shell builtin with
`command -V printf` and executes `ulimit -c 0`. External `printf` is a fixed failure; core dumps are
disabled for the entire child tree.

The runner uses the existing `dev.env`, exact five files, project `multica-dev-transition`, backend
only, `--no-deps --pull never`; it runs an ephemeral helper and does not recreate a service.

## 6. Queue freeze, duplicate prevention and commit

Before reading stdin, the helper:

1. parses `DATABASE_URL` without printing it and requires the exact DB tuple;
2. opens a guard transaction and takes `SHARE` only on `agent_task_queue`;
3. reads the aggregate count for precisely
   `queued,dispatched,running,waiting_local_directory`, requires zero, waits three seconds and
   requires zero again;
4. requires global `member=0`.

The guard stays open across mutation commit, post-commit verification and compensation, freezing
queue admission. A serializable mutation transaction locks `"user"`,
`user_password_credential` and `member` together in `SHARE ROW EXCLUSIVE`, requires
`0|0|0`, calls generated `CreateUser`, the existing `ValidatePassword`,
`ProvisionPassword`/bcrypt and provider-level `Login`, then requires `1|1|0` before commit.
The still-open guard requires `1|1|0` after commit. It calls no HTTP route and claims only
`HTTP_LOGIN=NOT_CALLED`; stateless JWT absence is not measurable and is not claimed.

`member` is present in both `beginMutation` lock lists, on the same transaction/connection as the
mutation. The guard never locks `member`; this prevents the cross-connection self-deadlock identified
in the final review.

All lower functions return fixed typed errors. Explicit rollback occurs and its result is checked
before an error returns.
Pool closure, guard rollback, buffer clear and string blanking complete before the sole outer
`os.Exit(realMain(...))`.

## 7. Exhaustive FK-safe rollback

Provision and rollback query `pg_catalog` and demand exact equality with 16 current FK constraints
in 15 tables. They cover `member`, `agent` (two columns), `agent_runtime`, `skill`,
`personal_access_token`, `chat_session`, `pinned_item`, `workspace_invitation` (two columns),
`feedback`, `notification_preference`, `github_installation`, `task_token`, `lark_installation`
and `user_password_credential`, including each `CASCADE`, `NO ACTION`, `RESTRICT` or `SET NULL`
action. Unknown, missing, multi-column or changed-action FKs are `E_FK_*` and BLOCK.

`daemon_pairing_session` was created by migration 005 and dropped by migration 029. It is not in
the expected current catalog; if it exists live, exact-catalog comparison blocks deletion.

Rollback is allowed only in the same sealed, no-login window and with the same stable secret
version. It locks every referencing table in deterministic scope, verifies the secret-backed
identity via the application provider, and requires zero target-user rows in every referring table
except the single credential. It deletes exactly one credential explicitly, then exactly one user;
it never relies on cascade, set-null or constraint failure. It requires `0|0|0` before commit and
again while the queue/member guard remains held. A possible HTTP login/JWT ends automatic rollback
authority because that fact is not DB-measurable.

`rollback-breakglass` covers lost/rotated passwords. It requires a separate written, unexpired
owner receipt bound to ORQ1, the fixed login, current counts and exact V2 hashes; the runner checks
its private `0600` file and pre-approved SHA-256. The helper skips only provider `Login`. It retains
host/DB binding, two queue-zero reads, admission/member freeze, `1|1|0`, exact FK catalog, every
reference lock/count, explicit one-credential then one-user delete, `0|0|0`, checked transaction
cleanup and fixed output. The sole user must equal `dataops.cloud.mbf@gmail.com`; no secret stdin is
accepted.

## 8. Fixed-output reducer and acceptance

Compose stdout/stderr go directly to private `0600` files. Raw files are never printed or returned
to agent/chat. The runner requires stdout byte-for-byte equal to one line and stderr empty, then
deletes both via trap. Any Compose/helper failure yields only `E_COMPOSE_OR_HELPER`; contract drift
yields only `E_STDOUT_CONTRACT` or `E_STDERR_CONTRACT`.

Only these reduced public statuses are allowed:

```text
ORQ17_STAGE3B_V2=PASS
```

After success, separately verify aggregate `1|1|0`, backend health/readiness `200`, anonymous
`/api/me=401|403`, frontend `13100=200`, unchanged persistent container IDs, Serve/Funnel `{}`,
no helper container, queue active count zero and secret metadata stability. No browser, CLI or
agent-driven login is allowed; only the contained gate below may call login.

The sole exception is the mandatory contained owner-login gate inside the sealed runner, before
success and before credentials are unset: helper-generated JSON flows by stdin to
`POST http://127.0.0.1:18080/auth/login`; a private `0600` cookie jar then authenticates
`GET /api/me`, which must return `200`. Bodies, headers and cookie are never printed and the jar is
removed by trap. On failure, aggregate counts are captured before guarded rollback. The rollback
window closes only after both HTTP gates pass.

The successful `guard.Commit` is the mechanical unfreeze; every error path verifies
`guard.Rollback`. Closing the one-shot DB session is a second fail-safe. Post-run verification
requires no helper DB session/lock and a fresh aggregate queue count of zero. Thus the freeze is
both reversible and measurably released.

Stop on any mismatch. Do not retry provision. Every failure first runs the read-only `state` mode
and classifies `0|0|0`, `1|1|0` or other before retry/rollback. Distinct fixed tokens identify
provision committed + guard failure (compensated or state unknown) and rollback committed + guard
failure. Safe compensation reacquires a fresh guard before exhaustive FK deletion. **Manual SQL
DELETE is always forbidden.** Any state outside exact guarded rollback/breakglass eligibility is
retained and escalated.

## 9. Review gate

This V2 package requests a reviewer independent of the V1/V2 author. Every C1–C8 gate is
conjunctive. Until a durable independent `PASS` names the exact artifact hashes and candidate
binary hash, **credential provisioning remains BLOCKED**.
