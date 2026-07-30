## Context

The accepted implementation originated at `45cefdf1afe6e9e7a2cad086f4835f353415a07f` and was ported without its
unrelated lineage as `7dd6f7df2cf1285c43ab2a0934d511189d1babb6`, whose sole parent is
`edd7b932f7c44c3396fd87853c2795d746fc5134`. The port is preserved at
`checkpoint/20260730/orq37-source-port`. It is accepted source only: it has not been fast-forwarded to main,
applied, deployed, reloaded, restarted, or accepted by a production canary.

## Goals / Non-Goals

**Goals:**

- Define the exact fail-closed filesystem and invocation contract.
- Record immutable source provenance and validation evidence.
- Make unapplied runtime and owner/external gates explicit.

**Non-Goals:**

- No code change, merge to main, apply, deploy, reload, restart, credential access, or production canary.
- No resolution of secret ownership, AWS permissions, SMA configuration, queue/admission coordination, or ORQ-58.

## Decisions

### Filesystem boundary

The root is canonical and caller-owned. A custom TMPDIR is a canonical descendant of that root. Existing target
files are caller-owned regular, non-symlink files with exactly one link. Writes use an unpredictable `0600`
temporary file in the destination directory followed by atomic rename. This avoids traversal, cross-filesystem
replacement, predictable temporary names, and hardlink retention.

### Invocation identity and rollback provenance

Units use the exact `*.service` grammar. Dry-run is the default, and the installer never invokes `systemctl`,
reload, or restart. Apply and rollback reuse the exact root/TMPDIR/unit tuple because TMPDIR is part of the
rendered managed bytes. A changed or omitted custom TMPDIR fails closed and preserves files. A multiply-linked
target fails with `E_TARGET_MULTIPLY_LINKED` before mutation.

### Operational separation

Source acceptance does not authorize runtime application. ORQ-58 remains separate. Runtime work begins only
after every external gate in `tasks.md` is explicitly satisfied by its owner.

## Risks / Trade-offs

- [Tuple mismatch prevents rollback] → Preserve and replay the exact root/TMPDIR/unit tuple.
- [Hardlinks retain managed bytes under another name] → Require link count one before apply and rollback.
- [Source acceptance is mistaken for production acceptance] → Record integrated-but-not-applied truth and keep
  all runtime tasks unchecked.
- [Cutover races active work] → Require separate queue-zero reads and PostgreSQL admission lock before mutation.

## Migration Plan

There is no authorized migration in this change. A future bounded cutover requires resolved secret metadata,
approved secret-resolution infrastructure, exact invocation tuple, tested rollback, two queue-zero reads with
admission locked, daemon/process/health/restart/queue checks, and a bounded credential-mode/MCP canary.

## Open Questions

- What are the exact secret identity, ARN, Region, version stage, and accountable owner?
- Will the owner approve least-privilege metadata-only IAM or an owner-reviewed loopback SMA on port 2773?
- What exact window authorizes queue-zero reads, PostgreSQL admission lock, rollback, and canary execution?
