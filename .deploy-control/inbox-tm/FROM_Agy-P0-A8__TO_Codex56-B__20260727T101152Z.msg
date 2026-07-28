[FROM Agy-P0-A8][wB:p2] TO Codex56#B (w7:p4) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T10:11:52Z
TASK: Plano de Rollout T8 (Backend + Daemon + AGY/Status Patches) — READ-ONLY

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/t8-rollout-plan.md

RESUMO DO PLANO (4 ETAPAS):
1. FASE 1 (Build Dev / ORQ2): Validação `go vet`/`go test`. Rebuild de binário/imagem é PONTOS STOP-AND-WAIT (Decisão Owner D1).
2. FASE 2 (Cutover ORQ1): Transferência de binário e restart do daemon. Parar daemon PID 3244391 e subida de novo binário é PONTO STOP-AND-WAIT (Decisão Owner D2).
3. FASE 3 (Healthcheck): Validação de Runtimes Online (/api/daemon/runtimes), Dropdown de Modelos (/api/models) e Execução com Slot Isolado (slot/home para AGY, slot/xdg-data para Kiro).
4. FASE 4 (Rollback Strategy): Rollback do backend via backups /tmp/daemon.env.bak e /tmp/daemon.cmd.bak. Frontend rollback 100% preservado e intocado.

NENHUM BUILD, DEPLOY OU RESTART FOI EXECUTADO (READ-ONLY).
