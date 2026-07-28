# Peer Review de Integridade: Monitor Automático do Kanban (GTL-28)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:37Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor.md` (autor: Agy-P0-A8 wB:p2)  
**modo**: READ-ONLY — NENHUM código, banco, build ou estado alterados  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: BLOCK** 🔴

**Resumo da Avaliação**:
O princípio arquitetural de **Monitoramento Passivo com Zero Rerun Automático** é **EXCELENTE e CORRETO**. No entanto, o documento possui **erros críticos de compatibilidade com o Schema SQL do banco de dados**, que tornam 2 das 4 queries inoperantes (retornando zero resultados em todas as execuções) e geram falsos positivos/negativos nas outras 2.

---

## 2. Auditoria Detalhada das 4 Consultas SQL (Invariantes)

### 2.1 Query 1 (Anomalia 1: `assigned todo` sem Task Enfileirada)
- **SQL Proposta no Documento**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  LEFT JOIN agent_task_queue atq ON atq.issue_id = i.id
  WHERE i.status = 'todo'
    AND i.assignee_type IN ('agent', 'squad')
    AND i.assignee_id IS NOT NULL
    AND atq.id IS NULL;
  ```
- **Falha Encontrada (Falso Negativo)**:
  - `LEFT JOIN ... WHERE atq.id IS NULL` só detecta issues em `todo` que **NUNCA** tiveram nenhuma task na história.
  - Se uma issue em `todo` teve uma task anterior que falhou (ex: `ORQ-12`), o registro antigo existe em `agent_task_queue`. O `LEFT JOIN` encontra a task antiga e a condição `atq.id IS NULL` se torna **FALSA**, ignorando a anomalia mesmo que a issue esteja em `todo` sem nenhuma tarefa ativa enfileirada.
- **SQL Corrigida Recomendada**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  WHERE i.status = 'todo'
    AND i.assignee_type IN ('agent', 'squad')
    AND i.assignee_id IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM agent_task_queue atq
        WHERE atq.issue_id = i.id
          AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
    );
  ```

---

### 2.2 Query 2 (Anomalia 2: `in_progress` sem Tarefa Ativa)
- **SQL Proposta no Documento**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  WHERE i.status = 'in_progress'
    AND NOT EXISTS (
      SELECT 1 FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
        AND atq.status IN ('queued', 'dispatched', 'running')
    );
  ```
- **Falha Encontrada (Falso Positivo)**:
  - Omite o status `'waiting_local_directory'` de `agent_task_queue`.
  - O código-fonte em `server/pkg/db/queries/issue.sql:120` (`ResetIssueToTodoIfNoActiveTask`) e `server/internal/service/task.go` define tarefas ativas como `status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')`.
  - Uma task aguardando diretório local geraria um **falso positivo**, alertando erradamente que `in_progress` não tem tarefa ativa.
- **SQL Corrigida Recomendada**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  WHERE i.status = 'in_progress'
    AND NOT EXISTS (
      SELECT 1 FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
        AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
    );
  ```

---

### 2.3 Query 3 (Anomalia 3: `completed` sem Conclusão da Task) — **CRÍTICO**
- **SQL Proposta no Documento**:
  ```sql
  WHERE i.status = 'completed' ...
  ```
- **Falha Encontrada (Incompatibilidade com Schema SQL / Bug Silencioso)**:
  - Na tabela `issue` (`server/migrations/001_init.up.sql:56`), o valor do enum de conclusão é **`'done'`**, e NÃO `'completed'`.
  - A restrição CHECK no Postgres é: `CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'blocked', 'cancelled'))`.
  - O status `'completed'` só existe na tabela `agent_task_queue`, nunca em `issue`.
  - **Consequência**: A cláusula `WHERE i.status = 'completed'` **nunca retornará nenhuma linha**. A Query 3 é inoperante.
- **SQL Corrigida Recomendada**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status,
         latest_task.status AS latest_task_status, latest_task.completed_at
  FROM issue i
  LEFT JOIN LATERAL (
      SELECT status, completed_at
      FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
      ORDER BY created_at DESC
      LIMIT 1
  ) latest_task ON TRUE
  WHERE i.status = 'done'
    AND i.assignee_type IN ('agent', 'squad')
    AND (latest_task.status IS NULL OR latest_task.status != 'completed' OR latest_task.completed_at IS NULL);
  ```

---

### 2.4 Query 4 (Anomalia 4: `blocked` com Dependência Pai Resolvida) — **CRÍTICO**
- **SQL Proposta no Documento**:
  ```sql
  WHERE i.status = 'blocked'
    AND p.status IN ('completed', 'cancelled');
  ```
- **Falha Encontrada (Incompatibilidade com Schema & Omissão de Tabela)**:
  1. `p.status IN ('completed', 'cancelled')` omite o status real de conclusão da issue pai, que é **`'done'`**. Não detectará pais concluídos.
  2. O schema de dependências do Multica possui a tabela dedicada **`issue_dependency`** (`server/migrations/001_init.up.sql:89`), com `type IN ('blocks', 'blocked_by', 'related')`. A query proposta considera apenas `parent_issue_id` e ignora a tabela de dependências primária `issue_dependency`.
- **SQL Corrigida Recomendada**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title AS issue_title,
         dep_issue.id AS dependency_id, dep_issue.title AS dependency_title, dep_issue.status AS dependency_status
  FROM issue i
  LEFT JOIN issue parent ON i.parent_issue_id = parent.id
  LEFT JOIN issue_dependency idep ON idep.issue_id = i.id AND idep.type = 'blocked_by'
  LEFT JOIN issue dep_issue ON idep.depends_on_issue_id = dep_issue.id
  WHERE i.status = 'blocked'
    AND (
      (parent.id IS NOT NULL AND parent.status IN ('done', 'cancelled'))
      OR
      (dep_issue.id IS NOT NULL AND dep_issue.status IN ('done', 'cancelled'))
    );
  ```

---

## 3. Avaliação da Arquitetura de Notificação & Zero Rerun Automático

- **Garantia de NUNCA Rerun Automático**: **APROVADO (PASS)**.
  - A decisão de não executar rerun automático ou mutação de banco no monitor é **fundamental**. Reruns automáticos desacoplados causam duplicidade de chamadas LLM e concorrência sobre task ledgers (`commitledger`).
- **Eventos UI & Reconciliação Existente**:
  - O backend já possui reconciliação síncrona via `ResetIssueToTodoIfNoActiveTask` (`server/pkg/db/queries/issue.sql:110`) invocada em `HandleFailedTasks` (`server/internal/service/task.go:1830`).
  - A emissão passiva do evento WebSocket `kanban:integrity_anomaly` complementa a reconciliação sem colidir com ela.

---

## 4. Requisitos para Aprovação (Unblock Steps)

Para que o documento `gtl-kanban-integrity-monitor.md` seja promovido a **PASS**, o autor (`Agy-P0-A8`) deve aplicar as seguintes correções no documento:

1. **Substituir `'completed'` por `'done'`** em todas as cláusulas `WHERE issue.status`.
2. **Atualizar a Query 1** para utilizar `NOT EXISTS (status IN ('queued', 'dispatched', 'running', 'waiting_local_directory'))`.
3. **Adicionar `'waiting_local_directory'`** na Query 2.
4. **Expandir a Query 4** para incluir a tabela `issue_dependency` além de `parent_issue_id`.
5. **Adicionar Suíte de Testes com Mocks do Schema Real**:
   - `TestAnomaly3_DoneWithoutCompletedTask`: Inserir issue com status `'done'` e task `'failed'`; verificar alerta.
   - `TestAnomaly4_BlockedByDependencyDone`: Inserir dependência `blocked_by` com status `'done'`; verificar alerta.

*Nenhuma ação de escrita executada nesta auditoria. Relatório 100% read-only.*
