# ORQ-37 — user-scoped umask and private TMPDIR

This bounded tool writes only beneath a canonical, caller-owned `--root`. It creates a systemd user
drop-in with `UMask=0077` and a private `TMPDIR`, plus an opt-in shell fragment. It never edits shell
startup files, invokes `systemctl`, reads credentials, or performs a host cutover.

## Safety contract

- Dry-run is the default. `--apply` and `--rollback` are mutually exclusive and all options require values.
- Units must use the strict `*.service` grammar and repeats are deduplicated.
- Root must be an existing, non-symlink directory owned by the caller. TMPDIR must be an absolute,
  canonical descendant. Existing descendant components must be owned directories and cannot be symlinks.
- Targets must be caller-owned regular files. Existing content must exactly match the content this invocation
  renders; apply never overwrites and rollback never deletes an unmanaged destination.
- Writes use unpredictable same-directory `0600` temporary files, `systemd-analyze verify`, atomic rename,
  and trap cleanup. The harness uses a private fake root and proves the stubbed `systemctl` is never called.

## Verification

Run from this directory:

```sh
bash -n install-umask-hardening.sh test-umask-hardening.sh
shellcheck install-umask-hardening.sh test-umask-hardening.sh
git diff --check
./test-umask-hardening.sh
```

The harness reports its exact PASS count and zero skips. It covers hostile roots and components, prefix-sibling
containment, target types and symlinks, unmanaged-content preservation, option arity/conflicts, strict unit
grammar/deduplication, same-directory temporary cleanup, rendered systemd verification, idempotency, rollback,
and zero service invocation.

## Apply and rollback boundaries

After an authorized dry-run, `--apply` writes only the managed files. Reload/restart is deliberately absent and
requires a separately authorized zero-queue window, rollback plan, and post-apply runtime/credential-mode canary.
`--rollback` removes only byte-exact managed files and preserves the private TMPDIR because it may contain state.
External IAM/SMA resolution remains a separate blocker. Do not combine this cutover with ORQ-58.
