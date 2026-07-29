# Proposal: Live Recovery Reconciliation & ORQ2 Daemon Control

## Context
On 2026-07-29 at ~15:04:41Z, an ORQ2 daemon regression occurred during un-isolated execution of the ORQ-23 harness. A systemd service restart re-exposed the pre-existing binary referenced by `ExecStart` (`e0510d7...`), rather than a binary swap. This exposed pre-token-only binary behavior and caused AGY tasks to fail on sibling symlinks (`cli.log`).

Canonical Kanban Board URL: `https://orq1.tail96e2c0.ts.net`

## Objectives
- Document Phase-1 bounded rollback executed under admission freeze (`LOCK TABLE agent_task_queue IN SHARE MODE` using UTC DB snapshot method) to proven token-only artifact SHA-256 `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`.
- Document daemon health (`127.0.0.1:19514/health`) and backend readiness (`127.0.0.1:18080/readyz`) endpoint operational verification.
- Track active runtime allowlists across runtimes: AGY (162, 163, 168, 169), Kiro (139, 140, 143, 149), Codex (152, 170).
- Document ORQ-64 credential-exposure incident (content-free, zero secrets logged) and harness isolation requirements (`mktemp` test root, stubbed `systemctl`, output sanitization).
- Track ORQ-65 runtime regression recovery (restored AGY token-only task-home allowlist, status DONE).
- Document ORQ-66 durable combined-daemon requirement (combining token-only task-home and reasoning admission).
- Document completed ORQ-58 production deployment (git revision `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live image `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`, rollback `8227241` image `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`, ORQ-70 canary OK).
- Reconcile all 28 non-Done Kanban cards against updated OpenSpec, terminal queue evidence, and GTL decisions.
