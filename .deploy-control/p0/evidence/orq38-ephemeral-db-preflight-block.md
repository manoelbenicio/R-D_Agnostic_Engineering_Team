# ORQ-38 — ephemeral PostgreSQL execution preflight

**Previous BLOCK retracted.** The initial check ran in the wrong host context
and is not an ORQ-1 capability finding.

**Corrected ORQ-1 preflight: PASS.** Using
`ssh -o BatchMode=yes ec2-user@100.118.244.61`, the target reported:

- hostname: `ip-172-31-18-217.sa-east-1.compute.internal`
- Tailscale IPv4: `100.118.244.61`
- Docker: `/usr/bin/docker`
- Docker client/server: `25.0.14` / `25.0.16`

No runtime was installed. No product container, volume, database, tunnel,
migration, test, credential, remote, board, or worktree was changed. The
approved image pin remains
`pgvector/pgvector:pg17@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0`.

The authorized ephemeral gate may proceed on ORQ-1 with a distinct container
and loopback-only ephemeral port, disposable credentials, and teardown receipts.

## Execution result (2026-07-27)

The corrected gate was exercised on ORQ-1 using a uniquely named container,
the approved pgvector digest, a loopback-only mapped port, an SSH local
forward, and a disposable database/user. Migrations completed through version
126. `go test -json ./internal/handler -run
^TestGetIssueWorkspaceFailClosed$ -count=1` produced nominal `Action:"pass"`
for S01, S02, S03, S04, S05, S06, S07 and S08, plus package pass; no
`Skipping tests` text and no `Action:"skip"` event occurred.

Teardown completed: the exact ephemeral container was removed, the SSH
forward was closed, and the private Go cache was removed. No product container
or volume was referenced.

## Final differential gate

The same disposable database/container and environment were used for both
runs, with the schema reset and migrations reapplied between runs:

| run | pass | fail | skip | exit |
|---|---:|---:|---:|---:|
| baseline `0cb8aeb` | 1334 | 4 | 37 | 1 (existing failures) |
| six-file candidate | 1343 | 4 | 37 | 1 (same failures) |

The four failing test names and all 37 skipped test names were identical. The
candidate added only the parent plus S01–S08 as passes; it introduced zero new
failures or skips. The dedicated S01–S08 run also produced eight nominal pass
events and zero skip events.

After teardown, the exact six-file lock was revalidated and committed locally:
`c536b609dc882a211ebc8362036cb29044811719` (`fix(cli): enforce issue get
workspace contract`). The worktree is clean. No remote, push, PR, board, or
ORQ-26 file was touched.
