# Design: Live Recovery Reconciliation

## Architecture & Recovery Posture
1. **Phase-1 Daemon Rollback**:
   - Active queue `SHARE` lock acquired.
   - Rolled back to artifact SHA-256 `88ca4f39` (token-only task-home allowlist: slots 162, 163, 168, 169).
   - Operational status verified via task 07172690 (5-minute AGY run without `cli.log` failure).
2. **ORQ-64 Harness Isolation & Content-Free Governance**:
   - ORQ-23 harness printed Codex credentials during un-isolated systemd service execution.
   - Requirement: Harness execution must require an explicit private test root via `mktemp`, stub/refuse systemctl, and zero secrets logged.
3. **ORQ-66 Combined Daemon Specification**:
   - Single clean daemon binary combining AGY token-only task-home allowlist (`antigravity_home.go`) and reasoning admission validation (`6bf0090`, `d9d569d`).
   - One-restart cutover plan under queue `SHARE` lock.
