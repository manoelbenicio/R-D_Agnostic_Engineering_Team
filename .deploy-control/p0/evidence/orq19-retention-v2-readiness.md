# ORQ-19 — Matriz de Prontidão para Implementação de Retenção e Lifecycle V2 de Runtimes (READ-ONLY)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T13:39Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**governança**: Kanban Issue `ORQ-19`  
**base técnica**: Especificação V2 `gtl-runtime-retention-safe-lifecycle-v2.md` & Peer Review `gtl-runtime-retention-safe-lifecycle-v2-peer-review.md` (GTL-84 PASS)  
**modo**: READ-ONLY / MATRIZ DE PRONTIDÃO — NENHUMA alteração de código, banco, container, API ou quadro executada  

---

## 1. Síntese Executiva de Prontidão

A especificação V2 do Lifecycle Seguro de Runtimes (GTL-84 PASS) converte o modelo destrutivo de exclusão de runtimes (`DELETE FROM agent_runtime`) em um modelo de **Retenção Perpétua com Soft-Archive (`archived_at TIMESTAMPTZ`)**, garantindo que 100% do histórico de execuções e billing (`task_usage`) seja preservado.

Esta matriz formaliza todos os requisitos de schema, transação, backend, frontend, trava de ressurreição por heartbeat, backfill dos 8 agentes e plano de testes/rollback para execução imediata pós-sequenciamento Z01.

---

## 2. Matriz Detalhada de Prontidão para Implementação

### 2.1 Dependência de Migrations Placeholder (Sequenciamento Z01)
> **Invariante**: A migration não materializa um número fixo e aguarda a alocação canônica na lane Z01.

#### `NEXT_CANONICAL_MIGRATION_runtime_retention_v2.up.sql`
```sql
-- Adição de colunas de Soft-Archive
ALTER TABLE agent_runtime
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS archived_by UUID REFERENCES "user"(id) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS archive_reason TEXT DEFAULT NULL;

-- Remoção de Constraints Globais e Criação de Índices Únicos Parciais
ALTER TABLE agent_runtime DROP CONSTRAINT IF EXISTS agent_runtime_workspace_id_daemon_id_provider_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_runtime_unique_active_provider
    ON agent_runtime (workspace_id, daemon_id, provider)
    WHERE daemon_id IS NOT NULL AND archived_at IS NULL;

DROP INDEX IF EXISTS agent_runtime_workspace_daemon_profile_key;

CREATE UNIQUE INDEX IF NOT EXISTS agent_runtime_workspace_daemon_profile_key
    ON agent_runtime (workspace_id, daemon_id, profile_id)
    WHERE profile_id IS NOT NULL AND archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_runtime_archived_at
    ON agent_runtime (workspace_id, archived_at)
    WHERE archived_at IS NOT NULL;
```

#### `NEXT_CANONICAL_MIGRATION_runtime_retention_v2.down.sql`
```sql
-- Exclusão prévia de linhas arquivadas para permitir a restauração da UNIQUE constraint global sem violação de chave
DELETE FROM agent_runtime WHERE archived_at IS NOT NULL;

DROP INDEX IF EXISTS idx_agent_runtime_unique_active_provider;
DROP INDEX IF EXISTS idx_agent_runtime_archived_at;

ALTER TABLE agent_runtime 
    DROP COLUMN IF EXISTS archived_at,
    DROP COLUMN IF EXISTS archived_by,
    DROP COLUMN IF EXISTS archive_reason;

ALTER TABLE agent_runtime 
    ADD CONSTRAINT agent_runtime_workspace_id_daemon_id_provider_key 
    UNIQUE (workspace_id, daemon_id, provider);
```

---

### 2.2 Inventário Completo de Arquivos (`FILES_LOCKED`)

| Módulo | Caminho do Arquivo | Função no Lifecycle V2 |
|---|---|---|
| **Migrations** | `server/migrations/NEXT_CANONICAL_MIGRATION_runtime_retention_v2.up.sql` | DDL de colunas de Soft-Archive e Índices Únicos Parciais |
| **Migrations** | `server/migrations/NEXT_CANONICAL_MIGRATION_runtime_retention_v2.down.sql` | DDL de rollback seguro com expurgo limpo de arquivados |
| **Backend SQL** | `server/pkg/db/queries/runtime.sql` | Queries de soft-archive, heartbeat guard e listagem filtrada |
| **Backend SQL** | `server/pkg/db/queries/agent.sql` | Query de reatribuição em lote de agentes (`UpdateAgentRuntimeInBatch`) |
| **Code Generated** | `server/pkg/db/generated/runtime.sql.go`, `models.go` | Interfaces e structs Go geradas pelo `sqlc` |
| **Backend Handlers** | `server/internal/handler/runtime.go` | Endpoint `POST /runtimes/{id}/reassign-and-archive` e alias de `DELETE` |
| **Backend Handlers** | `server/internal/handler/daemon.go` | Trava de `HTTP 409` no `Heartbeat`, `DaemonRegister` e `ClaimTask` |
| **Backend Services** | `server/internal/service/runtime.go` | Serviço transacional de reatribuição e validação de tarefas ativas |
| **Sweeper** | `server/cmd/server/runtime_sweeper.go` | Ignorar runtimes com `archived_at IS NOT NULL` na varredura de stale |
| **Core Client & API** | `packages/core/api/client.ts`, `schema.ts` | Adição da chamada `reassignAndArchiveRuntime` no cliente TypeScript |
| **Core Types** | `packages/core/types/runtime.ts` | Adição de `archived_at`, `archived_by`, `archive_reason` no tipo `AgentRuntime` |
| **Frontend UI** | `packages/views/runtimes/components/runtime-card.tsx` | Badge de arquivado e desativação de controles operacionais |
| **Frontend UI** | `packages/views/runtimes/components/delete-runtime-dialog.tsx` | Renomeado para Modal de Arquivamento com opção de Reassignment |
| **Frontend UI** | `packages/views/runtimes/components/reassign-archive-modal.tsx` | Componente de seleção de runtime de destino antes de arquivar |
| **Testes Backend** | `server/internal/handler/runtime_retention_test.go` | Suíte de testes Go deSoft-Archive, Heartbeat Guard e Reassignment |
| **Testes Frontend** | `packages/views/runtimes/__tests__/retention.test.tsx` | Testes React/Vitest do modal de arquivamento e filtro `include_archived` |

---

### 2.3 Semântica de Soft-Archive & Deprecação de DELETE Físico

- **Alias de `DELETE` HTTP**: A rota `DELETE /api/runtimes/{id}` é mantida por retrocompatibilidade, mas executa internamente:
  `UPDATE agent_runtime SET archived_at = NOW(), archived_by = $2, archive_reason = 'User soft-delete' WHERE id = $1 AND archived_at IS NULL`.
- **Filtro de Listagem Padrão**: As chamadas da UI (`GET /api/runtimes`) aplicam tacitamente `WHERE archived_at IS NULL`.
- **Filtro Administrativo Especial**: `GET /api/runtimes?include_archived=true` é restrito a administradores para relatórios de custos e auditoria.

---

### 2.4 Transação Atômica de Reatribuição e Arquivamento (`POST /reassign-and-archive`)

Endpoint atômico: `POST /api/runtimes/{id}/reassign-and-archive`

```go
// Protocolo da Transação em server/internal/service/runtime.go
func (s *RuntimeService) ReassignAndArchive(ctx context.Context, sourceID, targetID, actorID uuid.UUID, reason string) error {
	return s.db.RunInTx(ctx, func(tx *db.Queries) error {
		// 1. Lock no runtime de origem
		source, err := tx.GetRuntimeForUpdate(ctx, sourceID)
		if err != nil || source.ArchivedAt.Valid {
			return ErrRuntimeNotFoundOrArchived
		}

		// 2. Trava de tarefas ativas (queued, dispatched, running, waiting_local_directory)
		activeTasks, err := tx.CountActiveTasksForRuntime(ctx, sourceID)
		if err != nil {
			return err
		}
		if activeTasks > 0 {
			return ErrRuntimeHasActiveTasks // Mapeado para HTTP 409 Conflict
		}

		// 3. Reatribuição de agentes associados para o targetID
		if err := tx.ReassignAgentsToRuntime(ctx, db.ReassignAgentsParams{
			OldRuntimeID: sourceID,
			NewRuntimeID: targetID,
		}); err != nil {
			return err
		}

		// 4. Soft-Archive do runtime de origem
		return tx.SoftArchiveRuntime(ctx, db.SoftArchiveParams{
			ID:            sourceID,
			ArchivedBy:    actorID,
			ArchiveReason: reason,
		})
	})
}
```

---

### 2.5 Trava de Ressurreição por Heartbeat do Daemon

Para impedir que pings de daemons antigos des-arquivem silenciosamente um runtime:
- **`handler/daemon.go`**: As rotas `DaemonRegister`, `Heartbeat` e `ClaimTask` filtram `WHERE archived_at IS NULL`.
- **Resposta em Caso de Violação**: Se um daemon tentar heartbeat sobre um runtime arquivado, o backend retorna **`HTTP 409 Conflict`**:
  ```json
  {
    "error": "runtime has been archived by operator",
    "code": "runtime_archived"
  }
  ```
  Isso força o daemon a encerrar a sessão local sem gerar loops de reconexão ou ressurreição fantasma no Postgres.

---

### 2.6 Preservação de Histórico de Execuções e Billing

- **Integridade Referencial Mantida**: As tabelas `task_usage`, `agent_task_queue` e `agent_execution_log` possuem Foreign Keys apontando para `agent_runtime.id`.
- Como o runtime é retido permanentemente via soft-archive (sem `DELETE`), todas as métricas históricas de consumo de tokens, custos por modelo e duração de tarefas continuam totalmente preservadas e consultáveis nos relatórios financeiros.

---

### 2.7 Plano de Backfill para os 8 Agentes Medidos no ORQ1

Mapeamento factual dos runtimes offline medidos na auditoria ORQ1:
1. **Runtime Kiro Offline (`eb41a0c9-0005-4138-8d41-47975a642230`)**: Possui 5 agentes ativos vinculados.
2. **Runtime Claude Offline (`3fa85f64-5717-4562-b3fc-2c963f66afa6`)**: Possui 3 agentes ativos vinculados.

**Protocolo de Backfill**:
- Executar `POST /api/runtimes/{source_id}/reassign-and-archive` reatribuindo os 5 agentes do Kiro e os 3 agentes do Claude para um runtime ativo válido no workspace (`127.0.0.1:18080`) antes de arquivar as instâncias obsoletas.
- Nenhum registro fantasma ou runtime fictício é criado durante o procedimento.

---

### 2.8 Suíte de Testes & Estratégia de Rollback

#### Suíte de Testes (7 Casos Obrigatórios):
1. `TestSoftArchiveRuntime_Success`: Valida gravação de `archived_at`, `archived_by` e `archive_reason`.
2. `TestHeartbeatOnArchivedRuntime_Returns409`: Valida que pings de daemon em runtime arquivado recebem `HTTP 409` (`runtime_archived`).
3. `TestReassignAndArchive_ActiveTasksExist_Returns409`: Valida que tarefas ativas em andamento bloqueiam o arquivamento.
4. `TestReassignAndArchive_TransactionalSuccess`: Valida a reatribuição dos agentes e o arquivamento em transação única (`RunInTx`).
5. `TestPartialUniqueIndex_AllowsMultipleArchived`: Valida que múltiplos runtimes arquivados com o mesmo `(workspace_id, daemon_id, provider)` não violam o índice único parcial.
6. `TestTaskUsageHistory_PreservedAfterArchive`: Valida que os registros financeiros em `task_usage` permanecem intactos.
7. `TestFrontendDeleteDialog_ArchiveAlias`: Valida que a UI chama a rota de arquivamento com confirmação de destino.

#### Estratégia de Rollback:
- **Wave 1 (UI Rollback)**: Reverter componentes do frontend para os diálogos anteriores (sem alteração de banco).
- **Wave 2 (Backend/Schema Rollback)**: Executar a migration `NEXT_CANONICAL_MIGRATION_runtime_retention_v2.down.sql`, que limpa previamente as linhas arquivadas antes de restaurar a UNIQUE constraint global sem erros de violação de chave.

---

## 3. Check-out Citing ORQ-19

- **Governança**: `ORQ-19`
- **Artefato Gerado**: `.deploy-control/p0/evidence/orq19-retention-v2-readiness.md`
- **Veredito**: **PASS / PRONTO PARA IMPLEMENTAÇÃO** ✅
- **Status de Mutação**: READ-ONLY. Zero alterações executadas.
