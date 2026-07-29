# Tasks: Live Recovery Reconciliation & Control Steward

## Tasks
- [x] 1.1 Document 2026-07-29 ORQ2 daemon regression (systemd ExecStart binary re-exposure) and Phase-1 rollback to artifact `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000` under `LOCK TABLE agent_task_queue IN SHARE MODE`
- [x] 1.2 Verify daemon health `127.0.0.1:19514/health` and backend readiness `127.0.0.1:18080/readyz` endpoints
- [x] 1.3 Record active allowlist slots: AGY (162, 163, 168, 169), Kiro (139, 140, 143, 149), Codex (152, 170)
- [x] 1.4 Document ORQ-64 credential exposure incident content-free without secret values and specify `mktemp` harness isolation requirements
- [x] 1.5 Verify ORQ-65 recovery (token-only allowlist restored, status DONE)
- [x] 1.6 Track ORQ-66 combined-daemon requirements and cutover plan
- [x] 1.7 Record completed ORQ-58 production deployment (rev `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live image `sha256:e14f5c35d064...`, rollback `8227241` image `sha256:922b13862036...`, ORQ-70 canary OK)
- [x] 1.8 Reconcile all 28 non-Done Kanban cards against OpenSpec and GTL evidence
