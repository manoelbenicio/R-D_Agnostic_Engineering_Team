# ORQ-42 — V5 independent adversarial review (READ-ONLY)

- **Card:** ORQ-42 · UUID `64bfcae0-b867-4812-ad33-ae03ef7f25ae`
- **Reviewed:** `.deploy-control/p0/evidence/orq42-jwt-rotation-runbook-v5-design.md`
- **Reviewed SHA-256:** `2504cc3b2624ab465bfe743ca846f26713106deae148df05589a00e2de605936` — **MATCH**
- **Reviewer independence:** this reviewer is not V5 author Opus48#A, V4 author Antigravity, or the V4 reviewer.
- **Safety authority:** `.agents/skills/aws-secrets-manager/SKILL.md` v1, re-read completely for this review.
- **Mode:** source/documentation/web-reference only. No GetSecretValue, BatchGetSecretValue, SMA,
  asm-exec, Docker, AWS, environment-file, provider, token, daemon, database or board operation.
- **Verdict:** **BLOCK — V5 is not executable and authorizes no JWT rotation.**

## 1. Executive result

V5 correctly identifies its central Compose precedence problem: the base file declares
`services.backend.environment.JWT_SECRET: ${JWT_SECRET:?...}`, and a service `environment` value
wins over a service `env_file` value. Official Docker documentation also confirms that interpolation
for `${JWT_SECRET}` comes from the shell, an explicit CLI `--env-file`, or the project `.env`.

The correction is nevertheless incomplete. The direct forward command does not pass
`--env-file "$ENVF"` and does not pin a project directory, so it does not consume the durable
`dev.env` merely because that file is also named by `services.backend.env_file`. Only the existing
ORQ-30 helper may be considered, after exact hash equality and source/behavior verification.

Nine material blocks remain: unsafe target derivation, destructive env rewrite behavior, plaintext
in child argv, incomplete helper pin enforcement, stale-CID false PASS, incomplete old/new token
proof, asymmetric rollback artifacts, over-broad/open authorizations, and a factually wrong claim
that `mdt_` daemon tokens depend on `JWT_SECRET`.

## 2. Adversarial matrix

| Requested attack | Verdict | Evidence |
|---|---|---|
| Compose `environment` > `env_file` | **PASS premise / BLOCK command** | The premise matches the base Compose declaration and official Docker precedence. But V5 section 5.4 runs `docker compose -p "$PROJ" "$@" up ...` without `--env-file "$ENVF"`. A service `env_file` does not supply Compose-file interpolation. The required `${JWT_SECRET:?}` therefore depends on unrelated shell/project `.env` state or fails. |
| Live-label derivation | **BLOCK** | P2 selects any sole container labeled service=`backend`; P3 trusts its project/working-dir/config labels but never compares them with the independently approved ORQ1 values. A wrong sole backend becomes self-authorizing. P4 accepts empty, duplicate, relative, whitespace-bearing or unapproved config paths. P5 prints mode/owner but does not test them. P8 prints helper hashes but does not compare with the two full expected hashes. |
| Atomic durable `dev.env` fidelity | **BLOCK** | `grep -v "^JWT_SECRET=" ... || true` masks read errors: an I/O/permission failure can produce a file containing only the new JWT and then replace the original. It recognizes only one textual spelling, while Compose dotenv permits spaces, `:`, quotes, inline comments and multiline single quotes. It can leave semantic duplicates, changes terminal-newline behavior, discards metadata/ACL/xattrs/ownership, lacks an EXIT cleanup trap, and performs no file/directory `fsync`. `rename(2)` gives atomic visibility, not crash durability or content fidelity. |
| Child-only resolution and `/proc` argv | **BLOCK** | V5 embeds the dynamic reference inside the `sh -c` argument. `asm-exec` resolves command arguments, so the full resolved script contains `S="<plaintext>"` in the child `/proc/<pid>/cmdline`. V5 explicitly admits this exposure. “Best effort” is not an approval; no Q-A..Q-F or A1..A4 precisely accepts it. |
| Nonempty/placeholder/hex64 guards | **PASS, static** | G1 rejects empty, G2 finds literal `{{resolve:` anywhere, G3 permits only hex, and G4 requires exactly 64 bytes using `printf %s`. These checks correctly reject newline, quoting and arbitrary-byte hazards before replacement. |
| Auth-bypass effect gate | **PASS, static** | Anonymous loopback `/api/me=401` is an appropriate effect gate for bypass-off; old=401 plus new=200 later prevents treating universal 401 as success. It still must be executed only in an authorized window. |
| Backend-only recreate | **PASS flags / BLOCK path** | `up -d --force-recreate --no-deps backend` has the correct isolation and `restart` would not update container environment. The direct command lacks the interpolation env source. The preferred helper path is not proven by P8 because only hashes are printed and helper source is unavailable on this review host. |
| Old/new token probes | **BLOCK** | V5 never proves the old token returns 200 before mutation; an already expired/bad old token would later return 401 without proving rotation invalidation. Token config files are not mechanically checked as distinct regular owner files with mode 0600/parent 0700. Most critically, V5 deletes both files immediately after forward probes, but rollback R4 later needs both to prove old=200/new=401. |
| Symmetric rollback | **BLOCK** | R1 uses fixed `"$ENVF.restore.tmp"`; `cp` can clobber/follow an existing destination and the equality check only prints hashes. Restore lacks mktemp/no-clobber, `cmp -s`, fsync and metadata checks. R4 depends on token files already deleted. R5 repeats the stale-CID inspection defect. Forward and rollback are not mechanically symmetric. |
| ORQ2 same-window daemon re-pair | **BLOCK — premise false for `mdt_`** | Current `daemon_auth.go` handles `mdt_` first by `HashToken`, daemon cache and `GetDaemonTokenByHash`. It does not call `JWTSecret`. Only the later legacy JWT fallback uses `JWTSecret()`. Rotating JWT does not invalidate an actual `mdt_` credential. V5 section 4/8 incorrectly says all `mdt_` tokens stop validating and mandates an ORQ2 mutation without proving the active auth path. |
| Q-A..Q-F | **BLOCK overall** | All are still open. Q-D has a sound no-value attestation contract. Q-B must require a full ARN, not a region-ambiguous bare ID. Q-E lacks a reviewed post-rotation token-acquisition procedure and pre-forward old-token proof. Q-F is based on the incorrect unconditional re-pair premise. |
| Four authorizations | **BLOCK pending rewrite** | A2/A3 are directionally separate and correct. A1 improperly authorizes AWS “mutation/read” although Q-C needs metadata-only read; any secret mutation is separate and out of scope. A4 must be conditional on proving legacy JWT daemon auth, not required for `mdt_`/PAT paths. None cures the technical blocks above. |

## 3. Exact findings and corrections

### B1 — the direct Compose command does not consume durable `dev.env`

Official Docker docs establish:

- interpolated `environment` has higher precedence than service `env_file`;
- interpolation source order is shell, CLI `--env-file`, then project `.env`;
- the project directory defaults from `--project-directory`, first `-f`, then PWD.

Sources:

- <https://docs.docker.com/compose/environment-variables/envvars-precedence/>
- <https://docs.docker.com/compose/environment-variables/variable-interpolation/>

The V5 premise is correct, but section 5.4 omits the exact bridge it requires. Correction:

1. Q-A must select the existing ORQ-30 helpers; remove the direct command from the executable path.
2. Require exact equality against the full preserved hashes:

```text
multica-backend-recreate     fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af
multica-backend-env-rollback 0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4
```

3. Read/review helper source or authoritative source hash evidence proving it passes the durable file
   as Compose interpolation input and uses only `--force-recreate --no-deps backend`.
4. If helpers are superseded, a new V6 must explicitly use `--env-file "$ENVF"`, pin
   `--project-directory`, and receive separate review. Do not infer interpolation from service
   `env_file`.

### B2 — labels must be observations checked against an independent allowlist

Replace “derive everything, zero hardcode” with derive-and-compare. Require exact:

```text
host       ip-172-31-18-217.sa-east-1.compute.internal
project    multica-dev-transition
service    backend
env file   /home/ec2-user/.config/multica-transition/dev.env
config 1   /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
config 2   /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
config 3   /home/ec2-user/.config/multica-transition/images.yml
config 4   /home/ec2-user/.config/multica-transition/backend-env.override.yml
```

If the current approved stack now has five files, V5 must name that fifth absolute path and cite the
owner-approved evidence; “five live files” cannot be inferred from the selected container alone.
Reject relative, duplicate, empty, symlinked or unapproved paths. Test owner/mode numerically; do not
just print `stat`. Require parent ownership/mode and regular-file type. Re-select exactly one new
backend container after recreate and require `NEW_CID != OLD_CID`.

### B3 — replace grep rewriting with a reviewed byte-preserving editor

Do not use `grep ... || true`. Use a small pinned helper that:

1. opens the regular file without following symlinks and takes an exclusive lock;
2. reads bytes once and fails on read error/NUL;
3. parses the Compose dotenv grammar sufficiently to find active `JWT_SECRET` assignments including
   whitespace, `=`, `:`, quotes, inline comments and multiline quoted values;
4. requires exactly one active assignment; duplicate or zero is STOP;
5. replaces only the value token, preserving every other byte, comments, line endings and terminal
   newline;
6. preserves uid/gid/mode and required metadata;
7. writes a same-directory `O_EXCL` temp, fsyncs it, renames atomically, then fsyncs the directory;
8. has EXIT/INT/TERM cleanup and emits only fixed status.

A simpler acceptable policy is to require one exact canonical unquoted `JWT_SECRET=<64hex>` line and
fail if any alternate/duplicate spelling exists, then prove byte-for-byte that all non-target bytes
are unchanged. Never mask source read errors.

### B4 — remove plaintext from argv

Pass an unresolved reference in asm-exec’s incoming environment, not in the script argument:

```bash
ROTATING_JWT='{{resolve:secretsmanager:<FULL-ARN>:SecretString:<JSON-KEY>:<STAGE>}}' \
ENVF_PATH="$ENVF" \
asm-exec -- sh -eu -c '
  S=$ROTATING_JWT
  # G1-G4 and pinned editor invocation
'
```

The asm-exec process receives only the literal reference; its documented `child_env` resolution puts
plaintext in the child environment, not argv. Document residual same-UID/root `/proc/<pid>/environ`
risk and require explicit owner acceptance or a future asm-exec stdin-template mode. A1..A4 must not
silently imply acceptance.

### B5 — repair G5 and post-recreate identity

`CID` is the pre-recreate ID. After recreate:

1. derive exactly one `NEW_CID` scoped by both approved project and service labels;
2. require inspect success and `NEW_CID != OLD_CID`;
3. for the placeholder pipeline, capture both statuses (for example Bash `PIPESTATUS`) and require
   inspect status 0 plus grep status 1;
4. emit only fixed booleans/status codes, never `.Config.Env`.

Current behavior can accept a nonexistent old container: `docker inspect` fails, grep sees nothing,
returns 1, and V5 prints the expected `rc=1`.

### B6 — make token evidence complete and rollback-available

Before mutation, require:

```text
anonymous /api/me = 401
old-token /api/me = 200
```

After forward require old=401 and newly issued token=200. Q-E must specify a reviewed owner-only
login/acquisition procedure performed after recreate, with no token entering agent context. Require
both config paths to be distinct, regular, non-symlink, owner-controlled, mode 0600 in a 0700
directory. Keep both until the entire window is accepted and rollback is impossible; do not delete
them immediately after G4. Rollback must prove old=200/new=401, then owner-authorized cleanup.

### B7 — make backup/rollback genuinely symmetric

- Create backup and restore temp with no-clobber/random same-directory names.
- Reject symlinks and unexpected owner/mode.
- Use `cmp -s`, not printed hashes, as the mechanical equality gate.
- fsync backup/temp/directory as applicable.
- Restore through the same pinned byte/file helper.
- Re-run with the hash-pinned ORQ-30 rollback helper.
- Re-derive the rollback container ID and apply corrected G5.
- Retain old/new token configs through R4.
- Any failed rollback gate is STOP; no second mutation.

### B8 — correct the daemon dependency model

Current source order is:

1. `mdt_` -> SHA-256 hash -> daemon cache/DB;
2. `mcn_` -> cloud verifier;
3. `mul_` -> PAT hash/cache/DB;
4. only other/legacy tokens -> JWT HMAC with `JWTSecret()`.

Therefore Q-F must first obtain safe evidence of the active daemon auth-path label, never the token.
If the path is `daemon_token` (`mdt_`), JWT rotation requires heartbeat/online verification but no
re-pair. If the path is `jwt`, then and only then require the separately reviewed same-window re-pair
card/procedure/authorization. PAT/cloud paths likewise need path-specific verification, not JWT
assumptions. Remove the false statement that all `mdt_` tokens are invalidated.

### B9 — rewrite Q-A..Q-F and A1..A4 precisely

| Item | Required correction |
|---|---|
| Q-A | Owner selects hash-pinned ORQ-30 helpers; supersession requires a new design/review. |
| Q-B | Full secret ARN, JSON key and version stage only; bind account/region and freeze stage movement for the window. |
| Q-C | Metadata-only existence confirmation by authorized identity; no value API and no AWS mutation. Key existence remains owner attestation. |
| Q-D | Keep 64-hex attestation and G1-G4; PASS as designed. |
| Q-E | Name the owner-only post-rotation token acquisition procedure and pre-forward old-token=200 gate. |
| Q-F | Replace unconditional re-pair with safe auth-path evidence and conditional action. |
| A1 | Metadata-only AWS read authorization. Secret create/update/rotation is out of scope and separately authorized. |
| A2 | Backup + one durable edit + hash-pinned backend-only helper + forward gates. |
| A3 | Corrected symmetric rollback preauthorization, including retained probes. |
| A4 | Conditional ORQ2 re-pair only if legacy JWT auth is proved; otherwise authorize verification only, no mutation. |

All six Q answers and all four written authorizations must exist before any secret resolution or host
mutation.

## 4. What V5 gets right

- V5 is honest that it is a proposal and leaves Q-A..Q-F open.
- The durable source must be changed, not a throwaway temporary interpolation file.
- G1-G4 are sound for a 64-hex value.
- `force-recreate`, not restart, is required because container environment is immutable and
  `JWTSecret()` is process-cached.
- Backend-only `--no-deps backend` is the correct isolation intent.
- Anonymous 401 is a useful bypass-effect gate.
- Exact old/new 401/200 status checks are superior to `curl -f`.
- Curl config files avoid token argv exposure.
- Forward and rollback authorizations are separate.
- ORQ2 action must never be implied by ORQ-42 authority; it must be conditional and separately
  authorized if actually needed.

## 5. Final gate

**BLOCK.** Do not resolve the dynamic reference, edit `dev.env`, recreate backend, inspect container
environment, issue/probe a real token, or touch ORQ2. Produce V6 (or a corrected V5 supersession)
closing B1–B9, answer Q-A..Q-F, obtain A1..A4 with corrected scopes, and request another independent
review.

## 6. Non-actions

This review re-read the secret-safety skill and statically read repository/evidence files plus public
Docker documentation. It did not call GetSecretValue/BatchGetSecretValue, access SMA, execute
asm-exec, run Docker/Compose, call AWS, read or mutate `dev.env`, inspect `.Config.Env`, obtain/probe a
token, mutate a daemon/service/database/provider/board, or expose any secret value.
