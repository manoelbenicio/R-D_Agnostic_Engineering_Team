# ORQ-17 Stage3B — independent adversarial review (READ-ONLY)

- **Card:** ORQ-17
- **Reviewed package:**
  - `.deploy-control/p0/evidence/orq17-stage3b-owner-credential-provisioning-runbook.md`
  - `.deploy-control/p0/evidence/orq17-stage3b-bootstrap-helper.go.txt`
  - `.deploy-control/p0/evidence/orq17-stage3b-helper-go.mod.txt`
  - `.deploy-control/p0/checkins/CHECKOUT__Codex__ORQ-17-STAGE-3B-CREDENTIAL-PROVISIONING-DESIGN__20260727T161040Z.json`
- **Attribution note:** the dispatch attributes the package to Codex56#B; its checkout self-identifies the agent as `Codex` at line 4.
- **Safety authority:** `.agents/skills/aws-secrets-manager/SKILL.md`, read completely, plus its `references/asm-exec` implementation.
- **Mode:** independent, documentation/source-only review. No secret, provider, database, container, host service, queue or board mutation.
- **Verdict:** **BLOCK — credential creation is not authorized.**

## 1. Executive finding

The selected application path is materially better than direct SQL. The helper uses the generated
`CreateUser`, the server-authoritative `ValidatePassword`, `ProvisionPassword`, bcrypt default cost
and `PasswordAuthProvider.Login`. A serializable transaction and table locks close the ordinary
second-user/second-credential race, and the provider-level verification does not call the HTTP JWT
issuer.

Those positives do not make Stage3B safe to execute. Eight blocking defects remain:

1. resolved identity/password are placed in a child `env` process argv before they become stdin;
2. helper, helper module, asm-exec and full compiled source closure are not pinned, and no build was
   performed;
3. exact host/stack/database identity and the four-state queue-zero gate are incomplete;
4. `stop()` calls `os.Exit`, bypassing every deferred explicit rollback, pool close and byte clear;
5. the provision success path does not lock or prove `member=0`, and `SESSION=NONE` is a static
   inference rather than a measured state;
6. rollback checks only `member`, despite many other FKs that can cascade, restrict or be nulled;
7. raw Compose/helper stdout and stderr are not mechanically reduced to a fixed allowlist;
8. secret region/version stability and the exact one-time owner authorization are under-specified.

No user or credential may be created until a corrected runbook/helper receives a new independent
PASS.

## 2. Gate-by-gate verdict

| Requested attack | Verdict | Evidence and reasoning |
|---|---|---|
| Transactionality | **BLOCK** | The serializable transaction and `SHARE ROW EXCLUSIVE` locks are sound, but helper lines 27–29 call `os.Exit`. That skips `defer tx.Rollback`, `defer pool.Close` and both byte-clearing defers. PostgreSQL will eventually abort an open transaction when the OS closes the connection, but the runbook promises an explicit automatic rollback that the program does not execute. `memberRefs` can also call `stop()` from inside compensation, violating the compensation function’s boolean contract. |
| Duplicate prevention | **PASS, narrow** | Runbook requires external `0|0` twice; helper lines 51–57 recount globally inside the locked transaction; lines 109–114 fail on nonempty state. `user.email` is UNIQUE, `user_password_credential.user_id` is a PK, and rerun reaches `E_NOT_EMPTY`. This proves only user/credential uniqueness, not a sealed no-membership window. |
| bcrypt/application compatibility | **PASS, static** | `auth_provider.go` enforces 12 Unicode characters and 72 bytes, hashes with `bcrypt.GenerateFromPassword(..., bcrypt.DefaultCost)`, and verifies through `CompareHashAndPassword`. Helper lines 125–139 call those exact paths and enforce the current default cost. Migration 125 accepts the `$2...` bcrypt family and cascades credential deletion from user deletion. |
| asm-exec child-only stdin | **BLOCK** | Runbook lines 133–134 and 157–158 put dynamic references in arguments to `env`. The asm-exec implementation resolves command arguments into `args` at lines 372–373 and launches them at line 383. Therefore `OWNER_PASSWORD=<plaintext>` exists transiently in the child `env` argv and is readable from `/proc/<pid>/cmdline` by same-UID/root observers. The container receives NUL stdin, but the host-side path is not stdin-only and contradicts runbook lines 14–18. |
| Output redaction | **BLOCK** | Helper success/errors are fixed strings, which is good, but the command forwards raw `docker compose run` stdout/stderr through asm-exec. The prose “only accepted output” at line 177 does not capture, inspect or suppress Compose/runtime diagnostics. A fixed-output contract must be enforced, not merely expected. |
| Source/build hash | **BLOCK** | All six application pins match the current repository copy, but the runbook does not pin its own helper, helper module or asm-exec. Their observed SHA-256 values are recorded below. The checkout reports `gofmt` unavailable and explicitly says no build was attempted. The module template replaces the server module with a mutable absolute live path, and importing `internal/handler` compiles more source than the single pinned `auth_provider.go`. There is no reviewed dependency-closure manifest or expected binary hash. |
| Wrong-host / Serve / queue | **BLOCK** | Serve/Funnel `{}` and health/readiness 200 are valid gates. Host identity is incomplete: line 82 pins only Tailscale IP, while Stage2 identifies the hostname as `ip-172-31-18-217.sa-east-1.compute.internal`. The stop list names “active task queue” only at line 235; no command or two-reading gate exists. Canonical active states are `queued`, `dispatched`, `running`, `waiting_local_directory`. No admission freeze prevents a new task after a read. The helper accepts any nonempty inherited `DATABASE_URL` and does not bind host/database/user to the Stage2 stack. |
| Rollback safety | **BLOCK** | Helper lines 73–79 and 176–179 check only `member`, then delete the user. The schema has user FKs in personal access tokens, task tokens, chat sessions, pinned items, feedback, notification preferences, invitations, agents/runtimes, GitHub/Lark integration and others. Some cascade, some restrict, some set null. The claim at runbook lines 227–228 that a workspace/session causes `E_ROLLBACK_REFERENCED` is false. A stateless JWT issued by HTTP login is not discoverable in the database, so a separate rollback invocation cannot prove no login happened. |
| No membership/session | **BLOCK as gate; PASS as code-path intent** | The selected helper never calls workspace creation or HTTP login, and provider `Login` does not mint a JWT. However provision neither locks `member` nor verifies `member=0` in-transaction or post-commit. The output `SESSION=NONE` is not computed. A concurrent post-commit membership can appear before success because the queue/admission window is not closed. |
| Minimal owner inputs | **BLOCK pending precision** | Secret values and email correctly remain excluded. But accepting a bare secret ID permits ambient `AWS_REGION` to select a different regional secret; use a full ARN only. `AWSCURRENT` is resolved in separate dry/provision/rollback invocations and can rotate between them. The handoff confirmation describes scope but is not an unambiguous one-time authorization bound to ORQ1, the exact database tuple and the sealed window. |

## 3. Verified positive facts

### 3.1 Application and duplicate path

- `CreateUser` is a parameterized generated query and returns the created UUID.
- The user table has a unique email constraint.
- The credential table has one row per user and a bcrypt-family check.
- `ProvisionPassword` validates before hashing and writes only the hash.
- `PasswordAuthProvider.Login` lowercases/trims email and compares bcrypt.
- Provider login is stateless. JWT issuance exists only in the HTTP handler after provider login; the
  helper does not invoke that handler.
- The transaction’s global `0|0` check plus conflicting table locks prevents an ordinary concurrent
  second user/credential insertion.
- Secrets are not printed by the helper’s explicit error paths.

### 3.2 Hash audit

The six application hashes printed by the runbook match the current reviewed repository copy:

```text
MATCH 3c8f75b5ac2e9a4e2ca83b228285c3ff21d2ced484aab3b29f71cb7b67d70857 auth_provider.go
MATCH a89cf9a2413df383930f6807812d932d0099075217405ad55949fe42cfa43a25 user.sql.go
MATCH 0c9302e79a855e76de414411e9c730a96b56c46782e65d19f7f4b919a7771799 db.go
MATCH bb6c01ac50e77e4a072c5fe7b30744c8f72a265297f3b4735de5847367143388 125_user_password_credential.up.sql
MATCH e4122ce29e8e85b566f209ccbcefbe2450bdce2f97f0778a04e09c2f94b180d7 server/go.mod
MATCH 5d3b288a17bae9685b103c183b5b2ad7fa440f25484de6fd8ece7aac7f91ece5 server/go.sum
```

The following execution-critical files are not pinned by the runbook:

```text
24777e12428078f45631f252ccbc08873626f7390b6dbb5bd41bd7b4f290f04d orq17-stage3b-bootstrap-helper.go.txt
b3411a64ef6828177e9ad55ff425c92515419046431ab81f3918d238629d2127 orq17-stage3b-helper-go.mod.txt
d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556 .agents/skills/aws-secrets-manager/references/asm-exec
```

These observed hashes are review evidence, not authorization to execute those files.

## 4. Exact corrections required before re-review

### C1 — remove plaintext from argv

Do not pass references as `env NAME={{resolve...}}` command arguments. Put only unresolved references
in asm-exec’s own incoming environment, which its documented `child_env` path resolves for the
child:

```bash
OWNER_EMAIL='{{resolve:secretsmanager:<FULL_ARN>:SecretString:email:AWSCURRENT}}' \
OWNER_PASSWORD='{{resolve:secretsmanager:<FULL_ARN>:SecretString:password:AWSCURRENT}}' \
asm-exec -- sh -eu -c '
  set +x
  printf "%s\0%s\0" "$OWNER_EMAIL" "$OWNER_PASSWORD" |
    env -u OWNER_EMAIL -u OWNER_PASSWORD docker compose ... run ...
'
```

The asm-exec process sees unresolved references; the shell child receives resolved values in its
environment; Docker/Compose and the container receive neither secret environment variable, only
stdin. Document the residual same-UID/root `/proc/<pid>/environ` risk and require a sealed host
window with no other process under the operator UID. A stronger future asm-exec `stdin-template`
mode would remove the child-environment exposure as well.

Remove the separate secret-resolving dry run or combine it with the one provisioning resolution.
The helper already validates emptiness, length, UTF-8 and policy. Resolving `AWSCURRENT` twice creates
a version TOCTOU without improving safety.

### C2 — pin and verify the complete build

1. Add expected SHA-256 pins for helper source, module, asm-exec and a complete compiled-source/module
   manifest.
2. Build from a read-only, hash-verified source snapshot, not the mutable absolute live checkout.
3. Make the helper module complete, include its reviewed `go.sum`, and build with
   `GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 -trimpath`.
4. Pin and record Go version, Go executable hash, `GOOS` and `GOARCH`.
5. Independently build/review first; establish the expected binary SHA-256 and require equality in
   the operational window.
6. Prove that build inputs did not change during build. A newly computed hash with no expected value
   is not a provenance gate.

### C3 — close host, stack, database and queue gates

Require exact equality before any secret resolution:

```text
hostname = ip-172-31-18-217.sa-east-1.compute.internal
tailscale IPv4 = 100.118.244.61
Compose project label = multica-dev-transition
Compose service label = backend
Serve = {}
Funnel = {}
health = 200
readiness = 200
```

Derive the live five-file Compose list and image/container IDs from safe labels and compare it with
the Stage2 evidence. Inside the helper, parse the inherited pgx configuration and fail before
`BeginTx` unless host, database and username equal the approved Stage2 tuple; never print the DSN.

Add a queue admission freeze plus two read-only counts three seconds apart:

```sql
SELECT count(*)
FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```

Both results must be zero, and admission must remain frozen until provision/rollback and verification
finish. A prose stop condition is not a gate.

### C4 — make failure rollback explicit

Refactor helper functions to return typed fixed-code errors. `main` must defer byte clearing and pool
closure, and each transaction owner must explicitly call and verify `Rollback` before returning an
error. Call `os.Exit` only after all cleanup has completed. `memberRefs` must return `(int64, error)`;
it must never terminate from inside compensation.

### C5 — prove zero membership and accurately describe sessions

Lock `member` with the user/credential tables. Require `member=0` before creation, before commit and
after commit. The success line may assert only facts actually checked. Replace `SESSION=NONE` with a
precise static statement such as `HTTP_LOGIN=NOT_CALLED`, backed by the pinned provider/handler source.
Do not claim detection of a stateless JWT.

### C6 — make rollback non-cascading and symmetric

Before deletion, enumerate the current PostgreSQL FK catalog for references to `user(id)` and compare
it with a reviewed allowlist. Under locks, require zero rows for the target user in every referencing
table except `user_password_credential`. This must include at least member, personal access token,
task token, chat session, pinned item, feedback, notification preference, invitation, agent/runtime,
GitHub and Lark tables. Any unknown/new FK is `BLOCK`.

Delete exactly one credential row explicitly, then exactly one user row in the same serializable
transaction; do not rely on `ON DELETE CASCADE`. Apply the same host/source/build/queue/secret/output
gates to rollback. Because issued JWTs are stateless and unobservable, automatic rollback is allowed
only in the uninterrupted sealed pre-login window. If any login may have occurred, retain the user
and escalate; do not delete.

### C7 — enforce output allowlisting

Capture stdout/stderr to task-private `0600` files. Never print raw files. On success, compare stdout
byte-for-byte with the single approved line and require empty or separately allowlisted stderr.
Print only a new fixed reviewer-approved status. On failure print only a fixed error code and retain
no diagnostic containing arguments/environment. Compose progress output must not flow into agent or
chat context.

### C8 — make two owner inputs sufficient and unambiguous

Retain exactly two human inputs, but tighten them to:

1. full Secrets Manager ARN only, binding account and region, with a no-rotation freeze from
   authorization until the sealed window closes;
2. an exact authorization statement: `authorize one ORQ-17 Stage3B provision on the pinned ORQ1
   host/stack/database; bootstrap user+password only; no workspace/member; rollback only under the
   corrected sealed-window gates`.

The owner must never provide email, password, JSON, hash or version content. Secret creation,
rotation and resolver permission remain prerequisites outside Stage3B.

## 5. Required re-review evidence

A new independent reviewer must receive:

- corrected runbook/helper/module and their expected hashes;
- full dependency/source manifest and independently reproduced binary hash;
- static proof that resolved values never enter argv;
- exact host/stack/database and four-state queue gates;
- explicit cleanup/rollback control flow without lower-level `os.Exit`;
- full user-FK rollback guard and zero-member postcondition;
- mechanically fixed output allowlist;
- exact two-input owner authorization contract.

The reviewer must issue `PASS` or `BLOCK` before any credential resolution or creation.

## 6. Non-actions

This review did not call GetSecretValue or BatchGetSecretValue, did not access SMA, did not run
asm-exec, did not resolve a dynamic reference, and did not read identity/password/hash/DSN values.
It did not query or mutate a database, run Docker/Compose, contact a provider, alter Serve/Funnel,
create a user/credential/session/member, mutate a queue/board, or modify application/helper code.
Only this review evidence and its checkout are created.
