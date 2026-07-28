# Design de Monitoramento Automático de Integridade do Kanban v2 (GTL-30)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:38Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**baseado em**: BLOCK do Peer Review GTL-28 (`.deploy-control/p0/evidence/gtl-kanban-monitor-peer-review.md`)  
**modo**: READ-ONLY / DESIGN TÉCNICO — NENHUM código, banco de dados ou estado alterados  

---

## 1. Resumo da Evolução (V1 -> V2 pós BLOCK GTL-28)

O Peer Review GTL-28 reprovou o design V1 devido a quatro inconsistências críticas de schema SQL e modelagem de estado. O V2 resolve todas as falhas:

| Componente | Design V1 (GTL-28 / Reprovado) | Design V2 (GTL-30 / Corrigido) |
|---|---|---|
| **Status de Conclusão da Issue** | Usava `issue.status = 'completed'` (Status inexistente na tabela `issue`). | Usa **`issue.status = 'done'`** (enum correto do Postgres em `001_init.up.sql:56`). |
| **Histórico em `assigned todo`** | Usava `LEFT JOIN atq ... atq.id IS NULL` (ignorava issues com histórico de tasks passadas). | Usa **`NOT EXISTS (status IN active)`** (funciona independente de histórico passado). |
| **Status de Tarefas Ativas** | Filtrava `status IN ('queued', 'dispatched', 'running')`. | Inclui **`'waiting_local_directory'`** (evita falsos positivos em tarefas locais). |
| **Modelagem de Dependências** | Considerava apenas `parent_issue_id`. | Inclui **`issue_dependency` (`type = 'blocked_by'`)** e `parent_issue_id`. |
| **Garantia de Rerun** | Somente leitura / evento. | Mantido 100% passivo: **ZERO rerun automático**, 100% intervenção humana/owner. |

---

## 2. As 4 Consultas SQL Exatas Corrigidas (Invariantes V2)

### Anomalia 1: `assigned todo` sem Tarefa Ativa Enfileirada
- **Invariante**: Issue atribuída a agente ou squad em estado `todo` precisa ter uma tarefa ativa na fila (`queued`, `dispatched`, `running`, `waiting_local_directory`). Ignora histórico de tarefas passadas já concluídas/falhadas.
- **SQL de Detecção (v2)**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  WHERE i.status = 'todo'
    AND i.assignee_type IN ('agent', 'squad')
    AND i.assignee_id IS NOT NULL
    AND i.created_at < now() - INTERVAL '10 seconds' -- Mitiga janela de corrida no enqueue inicial
    AND NOT EXISTS (
        SELECT 1
        FROM agent_task_queue atq
        WHERE atq.issue_id = i.id
          AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
    );
  ```

### Anomalia 2: `in_progress` sem Tarefa Ativa em Execução
- **Invariante**: Issue em estado `in_progress` precisa ter pelo menos uma tarefa ativa na fila.
- **SQL de Detecção (v2)**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
  FROM issue i
  WHERE i.status = 'in_progress'
    AND NOT EXISTS (
        SELECT 1
        FROM agent_task_queue atq
        WHERE atq.issue_id = i.id
          AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
    );
  ```

### Anomalia 3: `done` sem Conclusão Efetiva da Tarefa
- **Invariante**: Issue em estado `done` (enum correto do DB) atribuída a agente/squad deve ter sua última tarefa com `status = 'completed'` e `completed_at NOT NULL`.
- **SQL de Detecção (v2)**:
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

### Anomalia 4: `blocked` com Dependência Resolvida (`done` ou `cancelled`)
- **Invariante**: Issue em estado `blocked` cujo `parent_issue_id` OU relação em `issue_dependency` (`type = 'blocked_by'`) já foi resolvida (`done` ou `cancelled`).
- **SQL de Detecção (v2)**:
  ```sql
  SELECT i.id AS issue_id, i.workspace_id, i.title AS issue_title, i.status AS issue_status,
         COALESCE(parent.id, dep_issue.id) AS resolved_dependency_id,
         COALESCE(parent.title, dep_issue.title) AS resolved_dependency_title,
         COALESCE(parent.status, dep_issue.status) AS resolved_dependency_status
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

## 3. Garantia Absoluta de ZERO Rerun Automático

1. **Princípio da Não-Intervenção Automática**:
   - O monitor é **estritamente passivo e somente-leitura (read-only)**.
   - NUNCA invoca `EnqueueTaskForIssue`, `CreateRetryTask` ou `MaybeRetryFailedTask`.
   - NUNCA executa `UPDATE` ou `DELETE` nas tabelas `issue` ou `agent_task_queue`.
2. **Razão Técnica**:
   - Reruns automáticos desacoplados provocam tempestades de reconcorrência, loops de consumo de tokens LLM e falhas de replay em ledgers (`commitledger`). A decisão de rerun é **exclusividade do operador/owner humano**.

---

## 4. Eventos UI & Notificação

1. **Publicação no Event Bus (`events.Bus`)**:
   - Constante de evento: `protocol.EventKanbanIntegrityAnomaly = "kanban:integrity_anomaly"`.
   - Publicado durante o tick de inspeção background para a sala WebSocket do workspace.
2. **Payload JSON**:
   ```json
   {
     "event": "kanban:integrity_anomaly",
     "anomaly_code": "ANOMALY_IN_PROGRESS_NO_ACTIVE_TASK",
     "issue_id": "8b1419f5-c9d4-466c-adff-cded98f91d29",
     "workspace_id": "20fce817-895d-447b-965a-49f5e279314a",
     "detected_at": "2026-07-27T11:38:00Z"
   }
   ```
3. **Comportamento no Frontend (`kanban-card.tsx`)**:
   - Exibe badge visual informativo no card da issue no Kanban (ex: ⚠️ *"Sem tarefa ativa"* / ℹ️ *"Dependência concluída"*).

---

## 5. Mitigação de Falsos Positivos

1. **Janela de Corrida na Criação (`created_at < now() - 10s`)**:
   - Previne que o monitor alerte Anomalia 1 na fração de segundo entre o `INSERT` da issue e o `INSERT` da task inicial.
2. **Inclusão de `waiting_local_directory`**:
   - Previne falsos alarmes de Anomalia 2 para agentes locais aguardando diretório de trabalho.
3. **Ordem de Execução do Sweeper**:
   - O monitor roda no servidor a cada 30 segundos, **após** a rodada do `runtime_sweeper.go`, garantindo que falhas em lote já foram processadas pelo `HandleFailedTasks`.

---

## 6. Arquivos Bloqueados para Implementação (`arquivos_locked`)

Estes são os únicos arquivos a serem criados/editados pelo agente escritor que implementar a V2:

1. `server/cmd/server/kanban_integrity_monitor.go` *(Novo runner background)*
2. `server/cmd/server/main.go` *(Wiring do ticker no boot do servidor)*
3. `server/pkg/db/queries/kanban_integrity.sql` *(Novas queries SQL `sqlc`)*
4. `server/pkg/db/generated/kanban_integrity.sql.go` *(Código Go gerado pelo `sqlc`)*
5. `server/internal/service/kanban_integrity.go` *(Serviço passivo de leitura e publicação de eventos)*
6. `server/internal/protocol/events.go` *(Adição da constante de evento `EventKanbanIntegrityAnomaly`)*
7. `packages/views/issues/components/kanban-card.tsx` *(Badge visual no card)*

---

## 7. Plano de Testes Automáticos (Suíte Go)

Arquivo de teste: `server/cmd/server/kanban_integrity_monitor_test.go`

1. **`TestAnomaly1_AssignedTodoNoTask_Detects`**:
   - Insere issue em `todo` com agente e nenhuma tarefa ativa (`created_at` antigo).
   - Executa 1 tick do monitor.
   - Assert: Evento `kanban:integrity_anomaly` publicado com código `ANOMALY_ASSIGNED_TODO_NO_TASK`.
2. **`TestAnomaly1_AssignedTodoHasOldFailedTask_Detects`**:
   - Insere issue em `todo` com task antiga em status `failed`.
   - Executa 1 tick.
   - Assert: Evento publicado (confirma correção do histórico passado).
3. **`TestAnomaly2_InProgressWaitingLocalDirectory_NoAnomaly`**:
   - Insere issue `in_progress` com task `waiting_local_directory`.
   - Executa 1 tick.
   - Assert: Zero anomalias detectadas (confirma ausência de falso positivo).
4. **`TestAnomaly3_DoneWithoutCompletedTask_Detects`**:
   - Insere issue em status `'done'` com última task `'failed'`.
   - Executa 1 tick.
   - Assert: Evento publicado com `ANOMALY_DONE_NO_COMPLETED_TASK`.
5. **`TestAnomaly4_BlockedWithDoneParent_Detects`**:
   - Insere issue `blocked` com pai em status `'done'`.
   - Executa 1 tick.
   - Assert: Evento publicado com `ANOMALY_BLOCKED_RESOLVED_DEPENDENCY`.
6. **`TestMonitor_StrictReadOnlyInvariance`**:
   - Executa 10 ticks do monitor com banco populado.
   - Assert: Contagem de linhas e valores de colunas nas tabelas `issue` e `agent_task_queue` permanecem **idênticos** antes e depois.


---

## SEÇÃO CORREÇÃO V3 (Ajustes Finais de Arquitetura & Veredito de Prontidão)

Esta seção consolida a revisão final V3 do monitor de integridade do Kanban, corrigindo caminhos de arquivos, agregação de dependências, de-duplicação e planos de execução SQL.

### 1. Mapeamento Correto de Arquivos no Repositório

- **Definição de Evento Protocol**: `server/pkg/protocol/events.go` (`EventKanbanIntegrityAnomaly = "kanban:integrity_anomaly"` e `EventKanbanIntegrityCleared = "kanban:integrity_anomaly_cleared"`).
- **Componentes Frontend do Board**: `packages/views/issues/components/board-card.tsx` & `packages/views/issues/components/board-view.tsx` (Substituindo a referência incorreta a `kanban-card.tsx`).
- **Runner Background**: `server/cmd/server/kanban_integrity_monitor.go`.
- **Queries SQL (`sqlc`)**: `server/pkg/db/queries/kanban_integrity.sql`.

---

### 2. Delimitação de Escopo

- **Em Escopo**: Anomalias nas colunas **`todo`**, **`in_progress`**, **`done`** e **`blocked`**.
- **Fora de Escopo (Out of Scope)**:
  - **`backlog`**: Issues em backlog não possuem tarefas enfileiradas por definição.
  - **`in_review`**: Issues em revisão representam entrega sob validação do usuário; a tarefa já foi concluída e a issue aguarda aceite humano.

---

### 3. Consultas SQL V3 Otimizadas com EXPLAIN & Agregação

#### 3.1 Anomalia 1: `assigned todo` sem Tarefa Ativa (`created_at < now() - 15s`)
```sql
-- name: DetectAnomalyAssignedTodoNoActiveTask :many
SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
FROM issue i
WHERE i.status = 'todo'
  AND i.assignee_type IN ('agent', 'squad')
  AND i.assignee_id IS NOT NULL
  AND i.updated_at < now() - INTERVAL '15 seconds' -- Janela de graça para transição e enqueue
  AND NOT EXISTS (
      SELECT 1
      FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
        AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
  );
```
- **Plano EXPLAIN**: Index Scan em `issue(status, assignee_type)` + Subquery Index Scan em `agent_task_queue(issue_id, status)`. Custo estimado O(N_todo).

#### 3.2 Anomalia 2: `in_progress` sem Tarefa Ativa
```sql
-- name: DetectAnomalyInProgressNoActiveTask :many
SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
FROM issue i
WHERE i.status = 'in_progress'
  AND NOT EXISTS (
      SELECT 1
      FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
        AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
  );
```

#### 3.3 Anomalia 3: `done` sem NENHUMA Tarefa Concluída (`EXISTS`)
```sql
-- name: DetectAnomalyDoneNoCompletedTask :many
SELECT i.id AS issue_id, i.workspace_id, i.title, i.assignee_type, i.assignee_id, i.status AS issue_status
FROM issue i
WHERE i.status = 'done'
  AND i.assignee_type IN ('agent', 'squad')
  AND NOT EXISTS (
      SELECT 1
      FROM agent_task_queue atq
      WHERE atq.issue_id = i.id
        AND atq.status = 'completed'
        AND atq.completed_at IS NOT NULL
  );
```
- **Melhoria V3**: Substituída a ordenação lateral por `NOT EXISTS(...)`. Se NENHUMA tarefa daquela issue alcançou `completed`, a anomalia é confirmada.

#### 3.4 Anomalia 4: `blocked` com Dependências Resolvidas (Agregado em JSON)
```sql
-- name: DetectAnomalyBlockedResolvedDependencies :many
SELECT i.id AS issue_id, i.workspace_id, i.title AS issue_title, i.status AS issue_status,
       jsonb_agg(jsonb_build_object(
           'id', COALESCE(parent.id, dep_issue.id),
           'title', COALESCE(parent.title, dep_issue.title),
           'status', COALESCE(parent.status, dep_issue.status)
       )) AS resolved_dependencies
FROM issue i
LEFT JOIN issue parent ON i.parent_issue_id = parent.id AND parent.status IN ('done', 'cancelled')
LEFT JOIN issue_dependency idep ON idep.issue_id = i.id AND idep.type = 'blocked_by'
LEFT JOIN issue dep_issue ON idep.depends_on_issue_id = dep_issue.id AND dep_issue.status IN ('done', 'cancelled')
WHERE i.status = 'blocked'
  AND (parent.id IS NOT NULL OR dep_issue.id IS NOT NULL)
GROUP BY i.id, i.workspace_id, i.title, i.status;
```
- **Melhoria V3**: Retorna 1 linha por issue com o array `resolved_dependencies` agrupado via `jsonb_agg`, eliminando duplicatas na notificação.

---

### 4. Ciclo de Vida de Anomalias & Limpeza (Clear Event)

1. **Memória de Estado do Monitor (`ActiveAnomalies map[string]Set`)**:
   - O monitor mantém um mapa em memória `workspace_id -> map[issue_id]AnomalyType`.
2. **Emissão de Evento `integrity_anomaly`**:
   - Quando uma nova anomalia é detectada (não presente no mapa no tick anterior), emite `EventKanbanIntegrityAnomaly`.
3. **Emissão de Evento `integrity_anomaly_cleared`**:
   - Quando uma anomalia anteriormente detectada deixa de existir no tick atual (ex: o usuário enfileirou a tarefa ou mudou o status da issue), o monitor emite `EventKanbanIntegrityCleared`.
   - O frontend em `board-card.tsx` remove o badge visual imediatamente ao receber o evento de limpeza.

---

### 5. Concorrência e Pós-Sweeper Execution

1. **Advisory Lock PostgreSQL (`pg_try_advisory_lock(4247)`)**:
   - O runner tenta adquirir o lock 4247 no início do tick. Se réplicas concorrentes do servidor estiverem ativas, apenas uma executa a inspeção.
2. **Execução Pós-Sweeper**:
   - Em `cmd/server/main.go`, a chamada `RunKanbanIntegrityMonitorTick(ctx)` é agendada para rodar **imediatamente após a conclusão do `runtime_sweeper`**, garantindo que tarefas órfãs/expiradas já foram processadas pelo `HandleFailedTasks` antes da varredura de integridade.

---

## 6. Resumo dos Arquivos Alterados/Criados (`arquivos_locked`)

1. `server/cmd/server/kanban_integrity_monitor.go` *(Runner background passivo)*
2. `server/cmd/server/main.go` *(Ticker + Advisory Lock 4247 + Sequência pós-sweeper)*
3. `server/pkg/protocol/events.go` *(Eventos `EventKanbanIntegrityAnomaly` e `EventKanbanIntegrityCleared`)*
4. `server/pkg/db/queries/kanban_integrity.sql` *(As 4 consultas V3)*
5. `server/pkg/db/generated/kanban_integrity.sql.go` *(Code generation `sqlc`)*
6. `server/internal/service/kanban_integrity.go` *(Serviço passivo + de-duplicação + emissão de eventos)*
7. `packages/views/issues/components/board-card.tsx` *(Badge visual de anomalia no card do Kanban)*

---

## 7. Suíte de Testes V3 Confirmada

1. `TestAnomaly1_AssignedTodoNoTask_Detects`
2. `TestAnomaly2_InProgressWaitingLocalDirectory_NoAnomaly`
3. `TestAnomaly3_DoneWithoutCompletedTask_EXISTS_Detects`
4. `TestAnomaly4_BlockedWithDoneParent_JSONAgg_Detects`
5. `TestMonitor_AnomalyClearedEventEmittedOnResolution`
6. `TestMonitor_PostSweeperOrderAndAdvisoryLock`
7. `TestMonitor_StrictReadOnlyInvariance`

*Design V3 finalizado em modo READ-ONLY. Zero mutações de código ou banco de dados.*
