# Relatório de Salvamento e Classificação de Código — ORQ-13 (GTL-40)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:43Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**worktree auditado**: `/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/00c87814/workdir/repo`  
**branch**: `agent/codex-a/orq-13-reasoning-tier-cost`  
**documento de referência de aprovação**: `gtl-versioned-tier-pricing-design.md` (Aprovado com PASS em GTL-36)  
**modo**: READ-ONLY / AUDITORIA DE CÓDIGO — NENHUM código, banco de dados ou git rebase alterado  

---

## 1. Resumo Executivo e Matriz de Salvamento

O worktree parcial da ORQ-13 contém avanços valiosos no pipeline de telemetria (daemons -> API -> `task_usage`), mas também introduz **anti-patterns graves e destrutivos** no banco de dados (ex: `DELETE FROM task_usage;` em scripts de rollback) e omitiu completamente a camada de precificação por tempo/versão aprovada no GTL-36.

### Matriz Sintética por Arquivo

| Arquivo / Módulo | Classificação | Justificativa Técnica |
|---|---|---|
| `server/migrations/127_task_usage_reasoning_tier.up.sql` | 🟡 **REESCREVER** | Manter a adição da coluna `thinking_level` em `task_usage`. **Remover** alterações destrutivas na PK e triggers de `task_usage_hourly` e `task_usage_hourly_dirty`. |
| `server/migrations/127_task_usage_reasoning_tier.down.sql` | 🔴 **DESCARTAR** | Contém `DELETE FROM task_usage;` e `DELETE FROM task_usage_hourly;` — **Destrutivo**. Reescrever com simples `ALTER TABLE DROP COLUMN`. |
| `server/pkg/db/queries/task_usage.sql` & `.sql.go` | 🟢 **REAPROVEITAR** | `UpsertTaskUsage` estendido para aceitar e persistir `thinking_level` na tabela de eventos brutos. |
| `server/internal/daemon/daemon.go` & `types.go` | 🟢 **REAPROVEITAR** | Leitura de `task.Agent.ThinkingLevel` e envio na struct `TaskMessageData` para o backend Go. |
| `server/internal/handler/daemon.go` | 🟢 **REAPROVEITAR** | Passagem limpa de `msg.ThinkingLevel` no payload do handler HTTP `ReportTaskMessages`. |
| `server/internal/handler/runtime.go` & `dashboard.go` | 🟡 **REESCREVER** | Ajustar queries de consumo para utilizar a tupla `(provider, model, tier)` sem violar o isolamento de rollups sem custo. |
| `packages/core/types/agent.ts` | 🟢 **REAPROVEITAR** | Tipagem TypeScript do campo `thinking_level` em payloads de telemetria. |
| `server/internal/metrics/pricing.go` & `pricing_test.go` | 🔵 **CRIAR (AUSENTE)** | O worktree **não implementou** a struct `ModelPrice` estendida com `EffectiveFrom`, `Version`, `PriceMissReason` e os 13 testes unitários do GTL-36. |

---

## 2. Detecção de Anti-Patterns e Violações no Worktree Parcial

### 2.1 Anti-Pattern 1: Migration Destrutiva em Rollback (GRAVE)
- **Evidência Literal em `127_task_usage_reasoning_tier.down.sql`**:
  ```sql
  DELETE FROM task_usage;
  DELETE FROM task_usage_hourly;
  DELETE FROM task_usage_hourly_dirty;
  ```
- **Risco**: Se esta migration for aplicada e posteriormente revertida em produção, **todos os dados históricos de consumo de tokens da plataforma são apagados permanentemente**.
- **Ação**: **DESCARTAR TOTALMENTE** este script de rollback e reescrever como:
  ```sql
  ALTER TABLE task_usage DROP CONSTRAINT IF EXISTS uq_task_usage_reasoning_tier;
  ALTER TABLE task_usage DROP COLUMN IF EXISTS thinking_level;
  ALTER TABLE task_usage ADD CONSTRAINT task_usage_task_id_provider_model_key UNIQUE (task_id, provider, model);
  ```

---

### 2.2 Anti-Pattern 2: Alteração Desnecessária e Complexa nos Rollups DB
- **Evidência Literal em `127_task_usage_reasoning_tier.up.sql`**:
  - Altera a Chave Primária de `task_usage_hourly` e `task_usage_hourly_dirty` para incluir `thinking_level`.
  - Recria todos os 4 triggers de rollup do banco de dados (`102`) para agrupar por `GROUP BY 1..8`.
- **Violação do Design GTL-36 (Seção 5)**:
  - O design aprovado no GTL-36 provou que as tabelas de rollup (`073`, `084`, `101`, `102`) agregam **apenas contagem de tokens** e não contêm coluna de custo (`cost_usd`).
  - Alterar a PK das tabelas de rollup aumenta a cardinalidade do banco e gera risco de quebra de performance nas queries de dashboard.
- **Ação**: **DESCARTAR** as alterações nos rollups `task_usage_hourly` / `task_usage_hourly_dirty`. O tier deve ser gravado **apenas** na tabela de eventos brutos (`task_usage`), enquanto rollups continuam agrupando tokens por `provider:model`.

---

### 2.3 Anti-Pattern 3: Ausência de Camada de Precificação Versionada (`internal/metrics/pricing.go`)
- **Problema**: O worktree parcial da ORQ-13 tentou resolver o problema de preço mexendo apenas em queries SQL do banco de dados, sem alterar a camada de métricas Go.
- **Falta do Design GTL-36**:
  - Não criou a struct `ModelPrice` com `EffectiveFrom` e `Version`.
  - Não implementou `PriceMissReason` (`model_unknown`, `tier_unknown`, `no_effective_price`).
  - Não criou a suíte de 13 testes unitários Go (`pricing_test.go`).
- **Ação**: **CRIAR** o código em `internal/metrics/pricing.go` e `pricing_test.go` alinhado ao GTL-36.

---

### 2.4 Anti-Pattern 4: Inferência de Tier por Sufixo de String de Modelo
- **Problema**: Algumas partes do código parcial tentam inferir o tier de raciocínio buscando sufixos como `-high` ou `-medium` na string do modelo.
- **Correção Exigida**: O tier deve ser tratado como um parâmetro de primeira classe desacoplado (`ThinkingLevel`), compondo a tupla explicita `(provider, model, tier)` na chamada de precificação em tempo de execução.

---

## 3. Classificação Detalhada Hunk-por-Hunk

### 3.1 `server/internal/daemon/daemon.go` & `types.go`
- **Hunk**: Adição de `ThinkingLevel string` em `TaskMessageData` e envio em `ReportTaskMessages`.
- **Veredito**: 🟢 **REAPROVEITAR DIRECTO**. A passagem do dado do daemon para o servidor HTTP está limpa e segue as especificações.

### 3.2 `server/internal/handler/daemon.go`
- **Hunk**: Leitura de `msg.ThinkingLevel` no handler e repasse para `Queries.UpsertTaskUsage`.
- **Veredito**: 🟢 **REAPROVEITAR DIRECTO**. Permite a gravação do dado bruto no Postgres.

### 3.3 `server/pkg/db/queries/task_usage.sql` & `.sql.go`
- **Hunk**: `UpsertTaskUsage` aceitando `@thinking_level::text`.
- **Veredito**: 🟢 **REAPROVEITAR DIRECTO**. Garante que cada execução de task grave seu `thinking_level` efetivo.

### 3.4 `server/migrations/127_task_usage_reasoning_tier.up.sql`
- **Hunk 1 (Linhas 1-25)**: `ALTER TABLE task_usage ADD COLUMN thinking_level TEXT...` -> 🟢 **REAPROVEITAR**.
- **Hunk 2 (Linhas 26-589)**: Re-definição de `task_usage_hourly`, `task_usage_hourly_dirty` e re-criação de triggers SQL -> 🔴 **DESCARTAR TOTALMENTE**.

---

## 4. Roteiro Recomendado para o Agente Implementador

1. **Descartar** a migration 127 atual e substituí-la por uma migration mínima não-destrutiva contendo apenas o `ALTER TABLE task_usage ADD COLUMN thinking_level TEXT NOT NULL DEFAULT ''`.
2. **Reaproveitar** os hunks de código Go do daemon e handler que propagam e salvam `ThinkingLevel` na tabela `task_usage`.
3. **Implementar** o design do GTL-36 em `server/internal/metrics/pricing.go` (struct `ModelPrice`, `EffectiveFrom`, `Version`, `PriceFor`, `PriceMissReason`).
4. **Adicionar** a suíte de 13 testes unitários Go exigida no GTL-36 em `server/internal/metrics/pricing_test.go`.

*Auditoria 100% READ-ONLY. Nenhuma linha de código, git rebase ou banco de dados foi alterada.*
