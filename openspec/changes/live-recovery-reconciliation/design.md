# Design: Live Recovery Reconciliation & Daemon Control

## Architecture & Recovery Posture
1. **Admission Freeze & Phase-1 Rollback**:
   - Queue freeze executed via `LOCK TABLE agent_task_queue IN SHARE MODE` using UTC DB snapshot method.
   - Rolled back daemon to proven token-only artifact SHA-256 `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`.
   - Verified daemon health via `127.0.0.1:19514/health` and backend readiness via `127.0.0.1:18080/readyz` (200 OK).
   - Restored active runtime allowlists: AGY (162, 163, 168, 169), Kiro (139, 140, 143, 149), Codex (152, 170).
   - Operational status verified via task 07172690 (5-minute AGY run without `cli.log` traversal errors, `NRestarts=0`).

2. **ORQ-64 Harness Isolation & Content-Free Governance**:
   - ORQ-23 harness printed Codex credentials during un-isolated systemd service execution.
   - Requirement: Harness execution must require an explicit private test root via `mktemp`, stub/refuse `systemctl`, and sanitize stdout/stderr with zero secret values logged.

3. **ORQ-66 Combined Daemon Specification**:
   - Single clean daemon binary combining AGY token-only task-home allowlist (`antigravity_home.go`) and reasoning admission validation (`6bf0090`, `d9d569d`).
   - One-restart cutover plan under queue `SHARE` lock (`LOCK TABLE agent_task_queue IN SHARE MODE`).

4. **ORQ-58 Production Deployment Audit**:
   - Completed deployment recorded: git revision `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live image `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`, rollback `8227241` image `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`, ORQ-70 canary OK.
