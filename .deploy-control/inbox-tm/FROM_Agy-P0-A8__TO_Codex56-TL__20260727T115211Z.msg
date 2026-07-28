[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:52:11Z
TASK: GTL-56 Design do Monitor Automático de Integridade do Kanban V5 (READ-ONLY)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v5.md (Novo arquivo V5 separado)

RESUMO DO DESIGN V5 DERIVADO DO SCHEMA REAL:
1. Schema Literal Derivado de Migrations (001_init.up.sql):
   - Status de Issue Concluída: 'done' (corrigido de 'completed').
   - Tabela de Dependência: issue_dependency (colunas issue_id, depends_on_issue_id, type IN ('blocks', 'blocked_by', 'related')).
   - Índice Real de Status: idx_issue_status(workspace_id, status).
2. Anomalia 3 Corrigida: Pesquisa issue.status = 'done' com LATERAL JOIN na última tarefa agent_task_queue(status != 'completed').
3. Anomalia 4 Corrigida: Agrega parent_issue_id e issue_dependency(depends_on_issue_id, type IN ('blocked_by', 'blocks')) com status 'done' ou 'cancelled'.
4. Preservados Pontos PASS: Acoplamento ao fim do runtime_sweeper.go (FILES_LOCKED), advisory lock 4247 na mesma conexão dedicada e eventos passivos no bus.
5. Suíte de Testes V5 + StrictReadOnly: Testes ajustados para o schema real e teste de imutabilidade sem alteração em updated_at ou issue.

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO EM CÓDIGO OU BANCO VIVO.
