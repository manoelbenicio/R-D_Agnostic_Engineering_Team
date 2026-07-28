[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:49:25Z
TASK: GTL-49 Design Corrigido do Monitor Automático de Integridade do Kanban V4 (READ-ONLY)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v4.md (Arquivo V4 separado)

RESUMO DO DESIGN V4:
1. Anomalia 4 (UNION/FILTER): Agrega parent_issue_id e issue_relation em fontes distintas com teste cobrindo ambos.
2. Indexação & EXPLAIN: Queries filtradas por workspace_id com análise EXPLAIN nos índices reais (idx_issue_workspace_status e idx_agent_task_queue_issue_id).
3. Acoplamento no Sweeper: runtime_sweeper.go adicionado a FILES_LOCKED e sweepKanbanIntegrity acoplado ao fim do tick.
4. Advisory Lock Seguro (Constante 4247): pg_try_advisory_lock(4247) gerenciado em conexão dedicada pgxpool.Acquire com unlock garantido no mesmo handler.
5. Emissão Passiva: Eventos kanban:integrity_anomaly publicados diretamente no bus desacoplado.
6. Suíte de Testes + StrictReadOnly: 7 testes cobrindo anomalias e concorrência + teste assertivo que garante zero alteração em updated_at ou colunas de issue.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO EM CÓDIGO OU BANCO VIVO.
