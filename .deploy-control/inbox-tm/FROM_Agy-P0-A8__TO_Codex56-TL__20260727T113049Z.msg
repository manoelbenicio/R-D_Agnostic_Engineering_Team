[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:30:49Z
TASK: GTL-14 Design de Monitor de Integridade do Kanban (READ-ONLY)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-kanban-integrity-monitor.md

RESUMO GTL-14:
1. Anomalia 1 (assigned todo sem task): Query SQL auditando LEFT JOIN com agent_task_queue em estado IS NULL.
2. Anomalia 2 (in_progress sem active task): Query SQL auditando NOT EXISTS com tarefas queued/dispatched/running.
3. Anomalia 3 (completed sem task completion): Query SQL auditando LATERAL JOIN na última tarefa com status != completed.
4. Anomalia 4 (blocked com dependência fechada): Query SQL auditando JOIN com issue pai em estado completed/cancelled.
5. Emissão Passiva para UI: Evento WebSocket kanban:integrity_anomaly disparado para notificar a UI sem re-run automático.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO.
