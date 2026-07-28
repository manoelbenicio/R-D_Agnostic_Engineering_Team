# Peer Review Adversarial: Design Kanban Integrity Monitor V5 (GTL-58)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:55Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v5.md` (autor: Agy-P0-A8 wB:p2, V5)  
**documento de referência**: `.deploy-control/p0/evidence/gtl-kanban-v4-peer-review.md` (BLOCK do GTL-54) e `001_init.up.sql`  
**modo**: READ-ONLY / AUDITORIA ADVERSARIAL — NENHUM arquivo de código, banco live ou quadro alterado  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: PASS** ✅

**Resumo da Avaliação**:
A versão V5 do design do Kanban Integrity Monitor (`gtl-kanban-integrity-monitor-v5.md`) **resolveu 100% dos bloqueantes técnicos apontados no GTL-54 e está perfeitamente alinhada ao schema real do banco de dados PostgreSQL (`001_init.up.sql`)**. O documento utiliza a tabela correta `issue_dependency`, a chave estrangeira `depends_on_issue_id`, os enums válidos (`'blocked_by'`, `'blocks'`), o status real da issue `'done'`, o nome do índice real `idx_issue_status` e mantém a concorrência via `pg_try_advisory_lock(4247)` em conexão dedicada com emissão passiva de eventos via `events.Bus`.

---

## 2. Auditoria Confrontada Item por Item

| # | Item Auditado | Validação contra o Schema Real | Evidência & Linha Literal em V5 | Status |
|---|---|---|---|---|
| **1** | **Tabela `issue_dependency` & Colunas** | Substituiu a tabela fantasma `issue_relation` do V4 pela tabela real `issue_dependency`, com `depends_on_issue_id` e `d.type`. | Section 2 (Anomalia 4): `FROM issue_dependency d JOIN issue dep ON d.depends_on_issue_id = dep.id` | ✅ PASS |
| **2** | **Status `'done'` em `issue` vs `'completed'` em `task`** | Corrigiu a Anomalia 3 e 4 para filtrar `i.status = 'done'` na tabela `issue` (conforme `CHECK (status IN (... 'done' ...))`), reservando `'completed'` para `agent_task_queue`. | Section 2 (Anomalia 3 & 4): `WHERE i.status = 'done'` e `p.status IN ('done', 'cancelled')` | ✅ PASS |
| **3** | **Nome do Índice Real `idx_issue_status`** | Corrigiu o nome do índice para `idx_issue_status ON issue(workspace_id, status)` de acordo com `001_init.up.sql:170`. | Section 1, Tabela L18: `idx_issue_status` (`CREATE INDEX idx_issue_status ON issue(workspace_id, status)`) | ✅ PASS |
| **4** | **Agregação Parent + Dependências** | Utiliza `UNION ALL` entre hierarquia pai (`parent_issue_id`) e dependências relacionais (`issue_dependency`) sem bugs de `COALESCE` ou duplicatas. | Section 2 (Anomalia 4): `UNION ALL SELECT ... 'issue_dependency' AS dependency_source` | ✅ PASS |
| **5** | **Filtro Scoped por Workspace** | Todas as 4 consultas aplicam `WHERE i.workspace_id = $1` garantindo busca delimitada por tenant. | Section 2, Queries 1-4: `WHERE i.workspace_id = $1` | ✅ PASS |
| **6** | **Advisory Lock 4247 em Conexão Dedicada** | Adquire e libera a trava 4247 na mesma conexão adquirida via `pool.Acquire(ctx)` e `defer conn.Release()`. | Section 3, L48: `pg_try_advisory_lock(4247)` via `pool.Acquire(ctx)` | ✅ PASS |
| **7** | **Acoplamento no `runtime_sweeper.go`** | Posiciona `sweepKanbanIntegrity` ao final do tick ticker em `runRuntimeSweeper` (`server/cmd/server/runtime_sweeper.go:88`). | Section 3, L44-46: Acoplado em `FILES_LOCKED` ao final de `runRuntimeSweeper`. | ✅ PASS |
| **8** | **Emissão Passiva no `events.Bus`** | Publica eventos `kanban:integrity_anomaly` desacoplados, sem dependência de tabelas de listeners no banco. | Section 3, L50: `bus.Publish("kanban:integrity_anomaly", map[string]any{...})` | ✅ PASS |
| **9** | **Suíte com 7 Testes + `StrictReadOnly`** | Define 7 testes unitários e de integração Go incluindo validação de imutabilidade estrita (`StrictReadOnly`). | Section 4: 7 testes Go definidos. | ✅ PASS |
| **10** | **Busca por Nomes Inventados** | Nenhum nome de tabela, coluna, enum ou índice foi inventado. 100% dos símbolos conferidos e validados no repositório. | Todas as seções auditadas contra `001_init.up.sql`. | ✅ PASS |

---

## 3. Resumo dos Símbolos Validados no Schema Real

- `issue.status`: `'todo'`, `'in_progress'`, `'done'`, `'blocked'`, `'cancelled'` (`001_init.up.sql:58`).
- `agent_task_queue.status`: `'queued'`, `'dispatched'`, `'running'`, `'completed'` (`001_init.up.sql`).
- `issue_dependency`: `id`, `issue_id`, `depends_on_issue_id`, `type IN ('blocks', 'blocked_by', 'related')` (`001_init.up.sql:89-94`).
- `idx_issue_status`: `ON issue(workspace_id, status)` (`001_init.up.sql:170`).
- `runtime_sweeper.go`: `server/cmd/server/runtime_sweeper.go`.

---

## 4. Conclusão e Prontidão para Implementação

O design V5 (`gtl-kanban-integrity-monitor-v5.md`) está **TOTALMENTE APROVADO (PASS)**. O General-Tech-Lead pode autorizar a implementação técnica do monitor no arquivo `server/cmd/server/runtime_sweeper.go` conforme o plano aprovado.

*Auditoria 100% READ-ONLY. Nenhuma linha de código, banco live ou quadro foi modificada.*
