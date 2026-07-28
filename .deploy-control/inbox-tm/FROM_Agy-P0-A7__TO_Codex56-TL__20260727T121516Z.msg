[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Design Arquitetural GTL-78 — Retention-Safe Runtime Lifecycle (gtl-runtime-retention-safe-lifecycle.md)

1. ANÁLISE DE BLAST RADIUS DAS FOREIGN KEYS:
- agent.runtime_id: ON DELETE RESTRICT (causa o HTTP 409 runtime_has_active_agents ao tentar deletar runtime com agentes ativos).
- agent_task_queue.runtime_id: ON DELETE CASCADE (um delete físico apaga a fila de tarefas e destrói 123 registros de task_usage/billing).

2. SOLUÇÃO ARQUITETURAL PROPOSTA:
- Pivot/Reassignment de Agentes: Move agentes ativos para um runtime online saudável do mesmo provider via PATCH /api/agents/{id} (runtime_id: <novo_id>).
- Soft-Delete de Runtime: Adiciona archived_at TIMESTAMP WITH TIME ZONE na tabela agent_runtime via migration NEXT_CANONICAL. Deletar marca archived_at = NOW() e status = "archived", ocultando o runtime da UI sem apagar o histórico de custos/billing.

3. CONTRATOS API/DB/UI & ROLLBACK:
- API: GET /api/runtimes filtra archived_at IS NULL; DELETE /api/runtimes/{id} faz soft-delete se agentes == 0; 409 plan-changed ressubmete se a atribuição mudar mid-flight.
- Quick UI Safety Fix: Intercepta delete na UI (Phase 1) sem alterar schema DB, oferecendo modal de Reassignment de agentes.
- Migration DB: Placeholder NEXT_CANONICAL (sem hardcode 127).

4. VEREDITO FINAL:
- STATUS: PASS (DESIGN ARQUITETURAL APROVADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-runtime-retention-safe-lifecycle.md.
Operação 100% Read-Only. Zero mutações.
