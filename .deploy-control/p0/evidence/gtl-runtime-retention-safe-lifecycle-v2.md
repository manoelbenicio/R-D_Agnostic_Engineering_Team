# GTL-82 — Specification V2: Retention-Safe Runtime Lifecycle (ORQ-18 / ORQ-19)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T12:20:34Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1), Agy-P0-A8 (re-review)
- **Bases:** `.deploy-control/p0/evidence/gtl-orq18-runtime-delete-readiness.md`, `gtl-orq19-stale-runtime-audit.md` e auditoria GTL-81
- **Modo:** SOMENTE LEITURA / ESPECIFICAÇÃO TÉCNICA V2 — Zero alterações em código, banco de dados ou interfaces de produção.

---

## 1. Prova Factual de DDL e Risco de Concorrência/Ressurreição

### 1.1 DDL e Unique Constraints Existentes (`migrations/004` e `120`)
1. `UNIQUE (workspace_id, daemon_id, provider)` (`004_agent_runtime_loop.up.sql:14`).
2. `CREATE UNIQUE INDEX agent_runtime_workspace_daemon_profile_key ON agent_runtime (workspace_id, daemon_id, profile_id) WHERE profile_id IS NOT NULL` (`120_runtime_profile.up.sql:81`).

### 1.2 Análise Factual de Comportamento do Backend Go
- **Heartbeat (`handler/daemon.go:723`)**: Atualiza `last_seen_at` via SQL. **Não limpa `archived_at` no SQL atual**.
- **Upsert de Registro**: Se um daemon tentar se registrar com o mesmo `(workspace_id, daemon_id, provider)` de um runtime arquivado, uma constraint UNIQUE não-parcial geraria colisão de chave duplicada no Postgres.
- **Risco de Ressurreição Fantasma**: Se a query de `Heartbeat` ou `DaemonRegister` não verificar `archived_at IS NULL`, um daemon desatualizado poderia enviar pings e operar em um runtime previamente arquivado pelo owner.

---

## 2. Desenho Técnico V2: Schema e Constraints Parciais

### 2.1 Migration DDL Placeholder `NEXT_CANONICAL`

```sql
-- Migration: NEXT_CANONICAL_runtime_retention_v2.up.sql

ALTER TABLE agent_runtime
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS archived_by UUID REFERENCES "user"(id) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS archive_reason TEXT DEFAULT NULL;

-- Atualizar índices de uniqueness para aceitar múltiplos registros arquivados mantendo unicidade entre ativos
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

### 2.2 Estratégia de DOWN Migration Segura
```sql
-- Migration: NEXT_CANONICAL_runtime_retention_v2.down.sql

-- Restaurar constraints sem falhar caso existam linhas arquivadas
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

## 3. Contratos de API, Guard de Tasks Ativas e Regras de Negócio

### 3.1 Tratamento do Daemon (Heartbeat / Claim / Upsert)
- **Bloqueio de Daemon em Runtime Arquivado**:
  - `Heartbeat` e `ClaimTask` verificam `WHERE archived_at IS NULL`.
  - Se um daemon tentar operar em um runtime com `archived_at IS NOT NULL`, o backend responde **`HTTP 409 Conflict`** com payload:
    ```json
    {
      "error": "runtime has been archived by operator",
      "code": "runtime_archived"
    }
    ```
- **Upsert / Resurrection Guard**: O registro de daemon nunca resuscita silenciosamente um runtime arquivado; cria uma nova instância de runtime ou exige um un-archive explícito do owner via API administrativa.

### 3.2 Endpoint Atômico `POST /api/runtimes/{id}/reassign-and-archive`
- **Active-Task Guard (`active_tasks == 0`)**:
  - Antes de reatribuir e arquivar, o handler valida se existem tarefas em estado `queued` ou `running` atribuídas aos agentes do runtime.
  - Se existirem tarefas ativas, o backend recusa com **`HTTP 409 Conflict`** (`code: "runtime_has_active_tasks"`).
- **Execução Atômica**:
  1. `UPDATE agent SET runtime_id = $target_runtime WHERE runtime_id = $source_runtime`
  2. `UPDATE agent_runtime SET archived_at = NOW(), status = offline, archived_by = $user_id, archive_reason = $reason WHERE id = $source_runtime`

### 3.3 API Pública e Consulta Histórica
- **Remoção de DELETE Físico da API Pública**:
  - A rota `DELETE /api/runtimes/{id}` é convertida internamente para soft-archive (`archived_at = NOW()`) se não houverem agentes ativos.
  - Deletes físicos (`DELETE FROM agent_runtime`) ficam restritos a scripts de manutenção offline do owner em bancos de teste.
- **Parâmetro Protegido `include_archived=true`**:
  - Por padrão, `GET /api/runtimes` e consultas da UI filtram `WHERE archived_at IS NULL`.
  - `GET /api/runtimes?include_archived=true` é restrito a administradores do workspace para relatórios financeiros, auditoria de billing e atribuição de custos (`task_usage`).

---

## 4. Plano de Backfill para os Runtimes Offline Medidos no ORQ1

Runtimes offline identificados:
- Kiro: `eb41a0c9-0005-4138-8d41-47975a642230` (5 agentes ativos)
- Claude: `588ebcba-f36c-45d9-8b3b-0fe17f5b83aa` (3 agentes ativos)

### Procedimento de Backfill Atômico:
1. Reatribuir os 5 agentes Kiro para o runtime Kiro online ativo `6d0d721a-2868-4530-9774-0675faebef74`.
2. Reatribuir os 3 agentes Claude para o runtime Claude online (ou arquivar os agentes se nenhum runtime Claude estiver online).
3. Marcar ambos os runtimes como arquivados:
   ```sql
   UPDATE agent_runtime 
   SET archived_at = NOW(), 
       status = offline, 
       archive_reason = orq19_stale_runtime_backfill 
   WHERE id IN (eb41a0c9-0005-4138-8d41-47975a642230, 588ebcba-f36c-45d9-8b3b-0fe17f5b83aa);
   ```

---

## 5. UI Renomeação e Fluxo de Reassignment (`packages/views/runtimes`)

1. **Renomear Botão/Modal na UI**:
   - A ação na interface é renomeada de "Delete Runtime" para **"Archive Runtime"**.
2. **Fluxo de Confirmação**:
   - Se o runtime possuir agentes ativos, a UI apresenta o modal de **Reassignment Primário** (recomendado), permitindo selecionar um runtime ativo substituto do mesmo provider.

---

## 6. Suíte de Testes Requerida (Go & Vitest)

1. `TestRuntime_HeartbeatRejectsArchivedRuntimeWith409`: Garante retorno `runtime_archived` (409) em tentativa de heartbeat em runtime arquivado.
2. `TestRuntime_UpsertDoesNotResurrectArchivedRuntime`: Garante que novos registros criam novas instâncias em vez de sobrescrever `archived_at`.
3. `TestRuntime_SweeperIgnoresArchivedRuntimes`: Garante que o sweeper background ignora linhas com `archived_at IS NOT NULL`.
4. `TestRuntime_ReassignAndArchiveAtomicWithActiveTaskGuard`: Testa o bloqueio de arquivamento quando existem tarefas rodando/enfileiradas.
5. `TestRuntime_ListExcludesArchivedByDefault`: Confirma que a listagem padrão omite runtimes arquivados.

---

## 7. Veredito Final
- **STATUS: PASS (ESPECIFICAÇÃO TÉCNICA V2 APROVADA PARA SUBMISSÃO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-runtime-retention-safe-lifecycle-v2.md`
- *Operação 100% Read-Only. Zero mutações de código, banco de dados ou interface.*
