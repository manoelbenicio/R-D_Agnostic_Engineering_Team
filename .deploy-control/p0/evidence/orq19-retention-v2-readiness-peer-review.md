# ORQ-19 — Peer Review de Governança: Auditoria de Prontidão da Retenção e Lifecycle V2 de Runtimes

- **Autor do Peer Review:** Antigravity (wB:p1 / w8:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/orq19-retention-v2-readiness.md`
- **Data UTC:** 2026-07-27T13:40:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Kanban Issue `ORQ-19`
- **Modo:** PEER REVIEW ADVERSARIAL READ-ONLY — Zero alteração de código, banco de dados, containers, APIs ou quadros executada.

---

## 1. Veredito Final
- **VEREDITO: PASS (MATRIZ DE PRONTIDÃO ORQ-19 APROVADA PARA EXECUÇÃO)**
- A matriz de prontidão `.deploy-control/p0/evidence/orq19-retention-v2-readiness.md` cumpre rigorosamente todos os requisitos da especificação V2 (`gtl-runtime-retention-safe-lifecycle-v2.md`), preserva 100% dos dados históricos de billing (`task_usage`) e estabelece um isolamento limpo de arquivos.

---

## 2. Auditoria Adversarial dos Requisitos da Matriz

### 2.1 Validação do Inventário de 16 Arquivos Bloqueados (`FILES_LOCKED`)
- **Verificação de Não-Sobreposição**: Todos os 16 arquivos estão estritamente contidos nos domínios de `migrations`, `db/queries`, `handler`, `service`, `core/api`, `views/runtimes` e testes específicos de retenção.
- **Isolamento Confirmado**: NENHUM arquivo colide com `file.go` / `file_test.go` (bloqueados pelo escritor único da `ORQ-26`) nem com o roteador central `router.go`.

### 2.2 Índices Únicos Parciais e Compatibilidade no Postgres
- **Análise DDL**:
  - `idx_agent_runtime_unique_active_provider` em `(workspace_id, daemon_id, provider) WHERE daemon_id IS NOT NULL AND archived_at IS NULL`
  - `agent_runtime_workspace_daemon_profile_key` em `(workspace_id, daemon_id, profile_id) WHERE profile_id IS NOT NULL AND archived_at IS NULL`
- **Garantia Técnica**: Permite a existência de múltiplos registros arquivados com os mesmos parâmetros sem violar as constraints de Uniqueness do Postgres quando um daemon é re-registrado.

### 2.3 Desafio da Semântica HTTP 409 no Heartbeat do Daemon
- **Desafio**: O retorno de `HTTP 409 Conflict` (com payload `{"code": "runtime_archived"}`) para pings de daemon em runtimes arquivados é adequado?
- **Verificação**: **SIM, É CORRETO**.
  - Retornar `401 Unauthorized` faria o daemon interpretar o erro como expiração de credencial de auth (`mdt_` / `pat`) e tentar reautenticação em loop.
  - Retornar `404 Not Found` omitiria o fato de que a entidade existiu e foi intencionalmente arquivada pelo operador.
  - `409 Conflict` sinaliza com precisão um conflito de estado de ciclo de vida (runtime desativado pelo owner), instruindo o daemon a interromper o heartbeat sem reconexões zumbis.

### 2.4 Transação Atômica de `reassign-and-archive` e Active-Task Guard
- **Protocolo de Segurança**:
  - `CountActiveTasksForRuntime` verifica tarefas em estados `queued`, `dispatched`, `running` e `waiting_local_directory`.
  - Se `activeTasks > 0`, recusa atomicamente com `HTTP 409 Conflict` (`code: "runtime_has_active_tasks"`).
  - A reatribuição de agentes em lote e o soft-archive executam sob `RunInTx` único, evitando estados intermediários inconsistentes.

### 2.5 Preservação de Histórico (`task_usage`) e Backfill dos 8 Agentes
- **Preservação de Dados (0% Perda)**: Ao abolir o `DELETE` físico em favor do `archived_at TIMESTAMPTZ`, todas as 123 linhas de `task_usage` medidas no ORQ1 continuam vinculadas às suas chaves primárias.
- **Backfill Medido**: Os 5 agentes do runtime Kiro offline `eb41a0c9` e os 3 agentes do runtime Claude offline `3fa85f64` são reatribuídos para runtimes ativos válidos antes da marcação de arquivamento.

### 2.6 Migrations, Colisão no `sqlc` e DOWN Migration Segura
- **`sqlc` Check**: Queries explicitamente nomeadas em `runtime.sql` e `agent.sql` geram stubs únicos em `generated/runtime.sql.go`, sem risco de colisão.
- **DOWN Migration Safety**: O script de rollback executa `DELETE FROM agent_runtime WHERE archived_at IS NOT NULL;` antes de recriar as UNIQUE constraints globais, garantindo que o rollback não falhe por violação de chave duplicada no Postgres.

---

## 3. Veredito Final
- **STATUS: PASS (AUDITORIA DE PRONTIDÃO ORQ-19 APROVADA)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq19-retention-v2-readiness-peer-review.md`
- *Operação 100% Read-Only. Nenhuma alteração no código, banco de dados ou quadros.*
