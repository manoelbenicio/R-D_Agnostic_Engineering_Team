# ORQ-37 — user-scoped umask 0077 + private TMPDIR (Wave B structural)

Status: **implemented and tested locally. No host cutover applied.** The card stays `in_progress`;
origin rotation of the MCP headers remains external input and is not covered here.

## What this is, and what it deliberately is not

`/etc/bashrc` sets `umask 002` on ORQ2 and ORQ1, so every artifact created by the execution user is born
world-readable. Containment then has to be re-applied by hand, which is why the population of
world-readable files in `/tmp` grew from 209 to 237 in a single day on ORQ2.

This patch hardens **only what the execution user owns**:

- a systemd **user** drop-in (`UMask=0077`, `TMPDIR`/`TMP` pointing at a private directory) for the units
  named on the command line;
- an **opt-in** shell fragment at `~/.config/orq37-umask-hardening.sh` that an interactive session can
  source.

It never edits `/etc/bashrc`, never writes a system-scope unit, never restarts a running unit, and never
reads, moves or inspects a credential file. The shell fragment is **not** wired into `.bashrc`: enabling
it for live sessions is part of cutover, which is a separate authorization.

## Files

```text
scripts/ops/umask-hardening/install-umask-hardening.sh   dry-run by default; --apply; --rollback
scripts/ops/umask-hardening/test-umask-hardening.sh      25 gates against a sandbox root
scripts/ops/umask-hardening/README.md                    this runbook
```

## Local verification already performed

```text
shellcheck install-umask-hardening.sh test-umask-hardening.sh   -> clean (exit 0, zero findings)
bash -n on both files                                            -> syntax ok
./test-umask-hardening.sh                                        -> ALL GATES PASSED (25/25)
```

The gates cover, against a throwaway sandbox and never the live home:

1. dry run is the default and writes nothing, while still printing the exact paths and the drop-in body;
2. `--apply` creates the drop-in `0600` inside a `0700` directory, with `UMask=0077` and `TMPDIR` pointing
   at the private directory, and the fragment `0600`;
3. `--apply` is idempotent — a second run leaves a byte-identical drop-in;
4. a behavioural probe: with `umask 0077` a freshly created file really is `0600`, so the setting is
   verified by effect and not only by text;
5. `--rollback` removes exactly the two artifacts it created, **preserves** the private TMPDIR (it may
   hold task state; deleting data is not this tool's job) and says so;
6. a second rollback on a clean tree is safe and reports `ABSENT`;
7. fail-closed refusals for `--root /`, `--root /etc`, a nonexistent root and an unknown flag;
8. a static assertion that the installer never targets `/etc/bashrc`.

## Current live state, for the cutover decision (measured, read-only)

```text
unit    multica-daemon-orq2-credential.service   UMask=0022   (no umask drop-in present)
shell   /etc/bashrc:75 umask 002                 umask -> 0002
```

The unit already carries `MULTICA_CREDENTIAL_SLOTS_ROOT` and the three slot allowlists; this patch adds
nothing but `UMask` and the two TMPDIR variables, in a separate drop-in file, so it cannot disturb the
existing configuration.

## Cutover (NOT authorized yet)

1. `install-umask-hardening.sh --root "$HOME" --unit multica-daemon-orq2-credential.service` — dry run,
   read the plan;
2. `--apply` the same command;
3. `systemctl --user daemon-reload`;
4. **restart of the daemon is disruptive**: it must land in a window with the active task queue at zero
   across `queued`, `dispatched`, `running`, `waiting_local_directory`, with admission frozen;
5. post-checks: `systemctl --user show <unit> -p UMask` reports `0077`; a file created by the unit is
   `0600`; the daemon health port answers before and after; slot directories keep `0700`.

Gate to abort: any pre-check failing, or the unit not returning to `active (running)`.

## Rollback

1. `install-umask-hardening.sh --rollback --root "$HOME" --unit <unit>` removes the drop-in and the
   fragment and leaves data untouched;
2. `systemctl --user daemon-reload`;
3. restart in the same zero-queue conditions;
4. `UMask` returns to `0022` — note that this restores the *insecure* default, so rollback should be
   treated as an incident response, not as a routine step.

The private TMPDIR is never deleted by rollback. If it must go, that is a separate, explicit decision
with its own authorization, because it can contain task state.

## Explicit non-goals

- No change to `/etc/bashrc` or any system-scope unit.
- No restart, reload or any mutation of a running unit by this tooling.
- No handling, reading or rotation of MCP header values (`/tmp/mh`, `/tmp/mcp_h`) — custody and rotation
  live in the Plan C contract, and origin rotation stays external input.
- No change on ORQ1: this patch targets the ORQ2 execution user only.
