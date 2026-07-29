# Proposal: Live Recovery Reconciliation & ORQ2 Daemon Control

## Context
On 2026-07-29 at ~15:04:41Z, an ORQ2 daemon regression occurred due to an accidental daemon restart / invalid binary deployment. The daemon reverted to pre-token-only binary behavior (`e0510d7...`), causing AGY tasks to fail before execution on sibling symlinks (`cli.log`).

## Objectives
- Document Phase-1 bounded rollback to proven token-only artifact SHA-256 `88ca4f39` with active allowlist slots (162, 163, 168, 169).
- Document ORQ-64 credential-exposure incident (content-free, zero secrets) and harness isolation requirements.
- Track ORQ-65 runtime regression recovery (Restored AGY token-only task-home allowlist, status DONE).
- Document ORQ-66 durable combined-daemon requirement (combining token-only task-home and reasoning admission).
- Reconcile all 28 non-Done Kanban cards against updated OpenSpec, terminal queue evidence, and GTL decisions.
