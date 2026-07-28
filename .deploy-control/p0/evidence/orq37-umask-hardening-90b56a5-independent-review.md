# ORQ-37 I3 — independent adversarial review of `90b56a5`

- Commit: `90b56a55b87826e70706914737ecd3d0d44602d7`
- Parent: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
- Worktree: `/home/ec2-user/workspace/worktrees/gtl-orq37-umask-hardening`
- Reviewer: Codex56#B (`w7:p4`)
- Mode: sandbox/read-only review. No live home, SSH, root, AWS, secret, service,
  reload, restart, cutover, board, source edit, or commit mutation.
- **Verdict: BLOCK.**

The design direction is correct, the declared 3-file lock is exact, and all 25
positive gates pass. The installer nevertheless accepts multiple hostile path
forms that escape its user-root contract, follows a root symlink, uses predictable
temporary paths that can overwrite through symlinks, and deletes an unmanaged
drop-in during rollback.

## Scope and positive findings

| item | result | evidence |
|---|---|---|
| Commit/file scope | PASS | Exactly three files under `scripts/ops/umask-hardening/`; zero `.deploy-control`; `git show --check` passes; owner worktree remained clean after review. |
| Syntax/static analysis | PASS | ShellCheck 0.11.0: zero findings; `bash -n`: PASS. |
| Existing sandbox suite | PASS | 25/25 gates, zero fail. |
| Dry-run default | PASS | No write in default mode; planned paths and drop-in are shown. |
| Intended modes | PASS | Normal sandbox apply creates drop-in/fragment `0600`, drop-in directory/TMPDIR `0700`, and `UMask=0077`. |
| Behavioral umask | PASS | A sandbox file created under `umask 0077` is `0600`. |
| Normal idempotency | PASS | A second normal apply produces the same drop-in hash. |
| TMPDIR rollback policy | PASS | Preserving the private TMPDIR is correct. It can contain task state; deletion must remain a separately authorized data-lifecycle action. |
| Opt-in shell fragment | PASS | Keeping it disconnected from `.bashrc` is correct in this non-cutover commit. Automatically sourcing it would itself be a live-session cutover. |
| Restart severity | PASS with correction | README calls restart disruptive and requires admission freeze plus zero active queue across all four states. The eventual cutover gate still needs exact commands/identity/health checks. |

Reviewed file hashes:

- installer: `3b86358d00952b9b02254de0fd6bd6fdec3c63cdc2b3973163c051fb8f278404`
- test: `bdf2da56ff063bb83a80be5179c73942210f951fe9777ffcfe54d17c945e2612`
- README: `73c3f26473a15b2d5283c975b21783ba74aa547927e2194d74f9a8daf0fa6c99`

## Blocking findings

All adversarial effects below were reproduced inside one private disposable
sandbox. No live path was used.

### B1 — `--unit` traversal escapes the declared root

`unit` is interpolated directly into:

```text
$root/.config/systemd/user/$unit.d/10-orq37-umask-hardening.conf
```

There is no validation against `/`, `..`, whitespace, control characters, or an
invalid unit suffix. Supplying `../../../../escaped/evil.service` created the
drop-in in a sibling outside the supplied root:

```text
UNIT_TRAVERSAL=ESCAPED_ROOT
```

A unit name containing a space was also accepted. Required fix: allow only a
strict systemd user-service basename, for example
`^[A-Za-z0-9][A-Za-z0-9_.@:-]*\.service$`, reject slash, `..`, whitespace and
control characters, and deduplicate repeated units.

### B2 — root and TMPDIR containment are not enforced

The root check uses `-d`, which follows symlinks. A symlink root was accepted and
the real target was modified:

```text
ROOT_SYMLINK=ACCEPTED_AND_WRITTEN
```

A relative root was accepted. An arbitrary `--tmpdir` outside the root was also
accepted and created:

```text
RELATIVE_ROOT=ACCEPTED
TMPDIR_ESCAPE=ACCEPTED_AND_CREATED
```

The explicit `/` and `/etc` string checks can therefore be bypassed by aliases,
symlinks or traversal. Required fix:

1. require an absolute root;
2. canonicalize with an existence-requiring operation;
3. refuse when the supplied root is a symlink or differs from its canonical path;
4. prove owner equals effective UID and reject system roots after canonicalization;
5. require canonical TMPDIR to be a strict descendant of the canonical root;
6. validate every existing path component as owned and non-symlink before writing.

### B3 — predictable `.tmp` permits overwrite and installs a symlink

Apply writes to fixed names:

- `10-orq37-umask-hardening.conf.tmp`
- `orq37-umask-hardening.sh.tmp`

Shell redirection follows an existing symlink before `chmod` or `mv`. The sandbox
test pre-created the drop-in temporary name as a symlink to a sentinel. Apply
overwrote the sentinel and then moved the symlink into the final drop-in:

```text
PREDICTABLE_TMP_SYMLINK=TARGET_OVERWRITTEN_AND_LINK_INSTALLED
```

Required fix: create an unpredictable `0600` temporary regular file in the
already validated destination directory, refuse symlinks/non-regular files, use
a cleanup trap, fsync as appropriate, and atomically rename only after validating
the temporary inode. Never redirect into a predictable caller-controlled path.

### B4 — apply/rollback do not preserve unmanaged state

Apply overwrites the reserved destination without proving it was created by this
tool. Rollback is worse for the drop-in: it deletes any regular file at the
reserved path, without a managed marker or exact-content check. A sandbox
unmanaged drop-in was deleted:

```text
ROLLBACK_UNMANAGED=DELETED
```

This contradicts “remove only the artifacts this script created.” Required fix:

- first apply may create an absent target;
- an existing target must be a non-symlink regular file, owned by the caller,
  and exactly match a previously managed representation; otherwise STOP;
- rollback must verify the same ownership/type and managed content/hash before
  removal;
- apply must never overwrite an unmanaged fragment or drop-in.

### B5 — argument parsing and systemd value syntax are not fail-closed

A missing `--root` value exits with code `1`, not the documented usage code `2`.
Conflicting `--apply`/`--rollback` flags are accepted with “last one wins.”
Whitespace/control characters are not rejected in root, TMPDIR, or unit values.
The generated unquoted `Environment=TMPDIR=$tmpdir` is not safe for paths with
spaces and can be split or interpreted incorrectly by systemd.

Required fix: validate arity before every `shift 2`, reject conflicting modes,
reject control characters, and either reject whitespace in managed paths or
render them with correct systemd escaping. Add a `systemd-analyze verify` gate
against a synthetic unit containing the generated drop-in.

## Required test additions

The next commit must make each case below fail closed and prove zero write outside
the sandbox root:

1. symlink root;
2. relative root;
3. root containing `..`;
4. external/equal/symlink TMPDIR;
5. unit containing slash, `..`, whitespace, newline, or wrong suffix;
6. missing value for `--root`, `--tmpdir`, and `--unit`;
7. simultaneous `--apply --rollback`;
8. pre-existing `.tmp` symlink;
9. pre-existing final symlink/directory/FIFO;
10. unmanaged final drop-in and fragment on apply and rollback;
11. path containing spaces, either safely escaped and verified or explicitly rejected;
12. repeated units and rollback of a partially managed set.

Retain the existing 25 positive gates, the effect-based `0600` probe, byte-level
idempotency, and the deliberate TMPDIR preservation test.

## Cutover documentation corrections

The no-cutover boundary is correct. Before any later operational authorization,
the runbook must replace generic post-check prose with executable, fixed checks:

1. prove exact ORQ2 hostname/user and exact user-unit name;
2. freeze admission and capture two aggregate zero-queue readings;
3. record old `MainPID`, `UMask`, health endpoint/port and drop-in list;
4. dry-run and compare the exact managed paths;
5. apply, `daemon-reload`, restart only the named user unit;
6. prove new `MainPID`, `active/running`, `UMask=0077`, exact TMPDIR/TMP values,
   exact health readiness, and a unit-context temporary-file mode;
7. prove credential slot directories remain private without reading their contents;
8. on failure, rollback under the same zero-queue gate and record that `0022` is
   an insecure incident fallback.

The README sentence that the new drop-in “cannot disturb” existing configuration
is too strong. It is non-overlapping with the slot allowlists, but changing UMask
and temporary paths can affect behavior; the cutover tests exist precisely to
measure that effect.

## Conclusion

**BLOCK.** The intended user-scoped hardening, safe rollback data policy, dormant
shell fragment, and restart warning are sound. The installer is not yet safe to
apply because unvalidated paths and unit names can escape the root, predictable
temporary names allow symlink overwrite, and rollback can delete unmanaged
configuration. No cutover should be authorized until B1–B5 and the adversarial
matrix pass.

