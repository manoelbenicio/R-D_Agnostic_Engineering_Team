# Peer Review Adversarial: Design Kanban Integrity Monitor V4 (GTL-54)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:51Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v4.md` (autor: Agy-P0-A8 wB:p2, V4)  
**documentos de referência**: `.deploy-control/p0/evidence/gtl-kanban-monitor-v3-peer-review.md` e `001_init.up.sql`  
**modo**: READ-ONLY / AUDITORIA ADVERSARIAL — NENHUM código, banco live ou quadros alterados  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: BLOCK** ❌

**Resumo da Avaliação**:
Embora o design V4 tenha trazido um avanço importante na gestão de concorrência (usando `pool.Acquire(ctx)` em conexão dedicada para o `pg_try_advisory_lock(4247)`) e no desacoplamento da emissão passiva de eventos via `events.Bus`, ele **introduziu regressões graves e tabelas/colunas inexistentes no schema do banco de dados**:
1. **Tabela e Colunas Inexistentes na Anomalia 4**: A consulta referencia a tabela `issue_relation` (que não existe; o nome real é `issue_dependency`), a coluna `related_issue_id` (real: `depends_on_issue_id`), a coluna `relation_type` (real: `type`) e o valor de enum `'depends_on'` (inexistente no CHECK constraint).
2. **Valor de Status Incorreto nas Anomalias 3 e 4**: As consultas filtram `i.status = 'completed'`, mas a tabela `issue` utiliza o status `'done'` (conforme `CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'blocked', 'cancelled'))` em `001_init.up.sql:57-58`). `'completed'` é status exclusivo da tabela `agent_task_queue`. Isso faria as Anomalias 3 e 4 retornarem 0 linhas sempre.
3. **Nome de Índice Inexistente na Seção 4**: Refere-se a `idx_issue_workspace_status`, enquanto o índice real criado na migration `001_init.up.sql:170` é `idx_issue_status ON issue(workspace_id, status)`.

---

## 2. Auditoria Confrontada contra os Achados do GTL-43 / V3

| # | Item Requisitado | Situação no Design V4 | Evidência & Linha Literal | Veredito |
|---|---|---|---|---|
| **1** | **Parent + Dependências Distintos & Teste** | Resolveu o bug do `COALESCE` do V3 usando `UNION ALL` entre pai e dependências. No entanto, usou tabela e colunas inexistentes na query SQL da dependência relacional. | Section 3 (Anomalia 4): `FROM issue_relation r ... JOIN issue dep ON r.related_issue_id = dep.id` | ❌ **BLOCK** (Tabela inexistente) |
| **2** | **Queries Scoped por Workspace & EXPLAIN** | Adicionou filtro por `workspace_id = $1`. Porém citou nome de índice inexistente no Postgres (`idx_issue_workspace_status`). | Section 4: Refere-se a `idx_issue_workspace_status` em vez de `idx_issue_status` (`001_init.up.sql:170`). | ❌ **BLOCK** (Nome de índice incorreto) |
| **3** | **Integração no `runtime_sweeper.go`** | Correta. Acoplou a chamada `sweepKanbanIntegrity` ao final do loop ticker (linha 88). | Section 5.2: `sweepKanbanIntegrity(ctx, pool, bus)` executado após `sweepExpiredQueuedTasks`. | ✅ **PASS** |
| **4** | **Decisão Single-Instance / Concorrência** | Correta. Esclareceu o funcionamento em memória e garantiu execução exclusiva via advisory lock. | Section 1 & Section 2: Conexão dedicada para garantir release seguro na mesma conexão. | ✅ **PASS** |
| **5** | **Advisory Lock 4247 na Mesma Conexão** | Correta. Utiliza `conn, err := pool.Acquire(ctx)` e `defer conn.Release()`, resolvendo a perda do lock entre conexões do pool. | Section 2.2: `conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", 4247)` | ✅ **PASS** |
| **6** | **Eventos Passivos sem Activity/Notification Listeners** | Correta. Emite diretamente no `events.Bus` (`"kanban:integrity_anomaly"`) sem tabelas de listener no banco. | Section 6: `bus.Publish("kanban:integrity_anomaly", map[string]any{...})` | ✅ **PASS** |
| **7** | **7 Testes + Assertiva `StrictReadOnly`** | Especificou suíte com 7 testes unitários/integração incluindo validação de imutabilidade da tabela `issue`. | Section 7: 7 testes Go definidos. | ✅ **PASS** |

---

## 3. Detalhamento dos Erros Bloqueantes Encontrados no V4

### 3.1 Bloqueante 1: Tabela e Colunas Inexistentes na Anomalia 4 (SQL Inválido)
- **Código Proposto no V4 (Seção 3)**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, 'relation_dependency' AS dependency_source, dep.id AS dependency_id
  FROM issue i
  JOIN issue_relation r ON r.issue_id = i.id
  JOIN issue dep ON r.related_issue_id = dep.id
  WHERE i.workspace_id = $1
    AND i.status = 'blocked'
    AND r.relation_type IN ('blocked_by', 'depends_on')
    AND dep.status IN ('completed', 'cancelled');
  ```
- **Fato no Schema Postgres (`001_init.up.sql:89-94`)**:
  ```sql
  CREATE TABLE issue_dependency (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      issue_id UUID NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
      depends_on_issue_id UUID NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
      type TEXT NOT NULL CHECK (type IN ('blocks', 'blocked_by', 'related'))
  );
  ```
- **Incompatibilidades**:
  - `issue_relation` -> DEVE SER `issue_dependency`.
  - `r.related_issue_id` -> DEVE SER `r.depends_on_issue_id`.
  - `r.relation_type` -> DEVE SER `r.type`.
  - `'depends_on'` -> INEXISTENTE no CHECK constraint (os valores válidos são `'blocks'`, `'blocked_by'`, `'related'`).

---

### 3.2 Bloqueante 2: Status `'completed'` Inexistente na Tabela `issue` (Anomalias 3 e 4)
- **Código Proposto no V4 (Seção 3)**:
  ```sql
  WHERE i.status = 'completed'  -- Anomalia 3
  AND p.status IN ('completed', 'cancelled') -- Anomalia 4
  ```
- **Fato no Schema Postgres (`001_init.up.sql:57-58`)**:
  ```sql
  CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'blocked', 'cancelled'))
  ```
- **Consequência**: O status de término da issue é `'done'`. Procurar por `status = 'completed'` fará a query retornar 0 linhas sempre, tornando o monitor totalmente cego para issues concluídas sem task.

---

## 4. Roteiro Correto para a Versão V5 (Ajuste Mínimo Requerido)

1. Corrigir a Anomalia 3 para filtrar `WHERE i.status = 'done'`.
2. Corrigir a Anomalia 4 para usar a tabela real `issue_dependency`:
   ```sql
   SELECT i.id AS issue_id, i.workspace_id, i.title, 'dependency' AS dependency_source, dep.id AS dependency_id
   FROM issue i
   JOIN issue_dependency d ON d.issue_id = i.id
   JOIN issue dep ON d.depends_on_issue_id = dep.id
   WHERE i.workspace_id = $1
     AND i.status = 'blocked'
     AND d.type = 'blocked_by'
     AND dep.status IN ('done', 'cancelled')
   ```
3. Atualizar o nome do índice na Seção 4 para `idx_issue_status` (conforme `001_init.up.sql:170`).

*Auditoria 100% READ-ONLY. Nenhuma alteração foi realizada no código live ou banco de dados.*
