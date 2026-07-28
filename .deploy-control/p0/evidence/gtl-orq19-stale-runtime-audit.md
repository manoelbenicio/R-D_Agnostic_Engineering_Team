# ORQ-19 — Stale Runtime 409 Audit
**Autor:** Antigravity Opus48#C / w8:p1 — READ-ONLY  
**Data:** 2026-07-27T11:27Z  
**Dispatch:** GTL-05  

---

## 1. Runtimes antigos identificados (medido no Postgres do ORQ1)

| runtime_id | provider | status | última atualização |
|---|---|---|---|
| `588ebcba-f36c-45d9-8b3b-0fe17f5b83aa` | claude | **offline** | 2026-07-26 23:48:53 UTC |
| `eb41a0c9-0005-4138-8d41-47975a642230` | kiro | **offline** | 2026-07-26 23:48:53 UTC |

Runtimes online (sem problema): codex `0f7133db`, kiro `6d0d721a`, antigravity `405b751d`.

---

## 2. Cadeia FK que impede DELETE — literal do schema e do código

### Vínculo primário: `agent.runtime_id ON DELETE RESTRICT`

```sql
-- migrations/004_agent_runtime_loop.up.sql:72-73
ADD CONSTRAINT agent_runtime_id_fkey
    FOREIGN KEY (runtime_id) REFERENCES agent_runtime(id) ON DELETE RESTRICT;
```

`RESTRICT` significa que `DELETE FROM agent_runtime WHERE id = $1` falha
imediatamente se qualquer linha em `agent` ainda aponta para esse runtime.

### Agentes ativos vinculados aos runtimes offline (medido)

| runtime | provider | agentes ativos |
|---|---|---|
| `588ebcba` | claude (offline) | **3 agentes** (`8101fcf3`, `b2f5448`, `2954cddf`) |
| `eb41a0c9` | kiro (offline) | **5 agentes** (`24b10605`, `bd1ebd0a`, `d09b1fc1`, `1e0f4914`, `c37759f7`) |

### Segundo vínculo: `agent_task_queue.runtime_id ON DELETE CASCADE`

```sql
-- migrations/004_agent_runtime_loop.up.sql:85-86
ADD CONSTRAINT agent_task_queue_runtime_id_fkey
    FOREIGN KEY (runtime_id) REFERENCES agent_runtime(id) ON DELETE CASCADE;
```

Este em cascade — as tasks desaparecem se o runtime for deletado. Mas como o
`agent` usa RESTRICT, nunca se chega ao DELETE do runtime enquanto houver agentes.

### 123 linhas de `task_usage` associadas (via task → agent → runtime offline)

`task_usage` não tem FK direta em `agent_runtime` — referencia `agent_task_queue`
que tem `ON DELETE CASCADE`. Logo `task_usage` seria apagado em cascata via tasks
ao deletar o runtime. Dado histórico preservável via reassignment antes do delete.

### Vínculo terciário: squads com líderes arquivados

`runtime.go:L586-594`: se houver squads ativos cujo líder está arquivado no mesmo
runtime, outro 409 bloqueia:
```go
// runtime.go:L592-594
if activeSquadCount > 0 {
    writeError(w, http.StatusConflict,
        "cannot delete runtime: it has active squads led by archived agents...")
}
```

---

## 3. Fluxo de API completo (UI → backend → DB)

```
DELETE /api/runtimes/:id
  └─ handler/runtime.go:L545 DeleteAgentRuntime
        ├─ ListActiveAgentsByRuntime → len > 0?
        │     YES → 409 {code:"runtime_has_active_agents", active_agents:[...]}
        │           ↑ ESTE É O 409 QUE O OWNER ESTÁ VENDO
        └─ [se len=0] CountActiveSquadsWithArchivedLeadersByRuntime
              ├─ > 0 → 409 "active squads with archived leaders"
              └─ = 0 → DeleteArchivedAgentsByRuntime → DeleteAgentRuntime → 200 OK

POST /api/runtimes/:id/archive-agents-and-delete
  └─ handler/runtime.go:L714 ArchiveAgentsAndDeleteRuntime
        ├─ LockAgentRuntime (FOR UPDATE — bloqueia novos inserts)
        ├─ ListActiveAgentsByRuntimeForUpdate
        ├─ activeAgentSetMatches(expected) → NO → 409 {code:"runtime_delete_plan_changed"}
        └─ [se match] TRANSAÇÃO:
              1. ArchiveAgentsByIDs
              2. CancelAgentTasksByRuntimeOrAgent
              3. PauseAutopilotsByAgentAssignees
              4. DeleteArchivedAgentsByRuntime
              5. DeleteAgentRuntime
              → COMMIT → 200 OK
```

---

## 4. UX existente (o que já existe no frontend)

O `DeleteRuntimeDialog` em `packages/views/runtimes/components/delete-runtime-dialog.tsx`
já implementa o fluxo em **3 modos**:

**Modo 1 — Light** (sem agentes ativos): diálogo simples "Tem certeza?" → `DELETE`.

**Modo 2 — Cascade** (agentes ativos): abre com lista de agentes que serão
arquivados, checkbox de confirmação, botão "Arquivar e deletar" → `POST .../archive-agents-and-delete`.

**Modo 3 — Plan changed**: se o backend retorna `runtime_delete_plan_changed`,
o diálogo recarrega a lista e força novo `checkbox` antes de resubmeter.

**O problema atual:** o diálogo no bundle rollback (`cf8017e3`, imagem de 2026-07-24
— confirmado em ORQ-26 como rollback intencional) pode estar num estado que:
- Não exibe o modo cascade corretamente para os runtimes offline; ou  
- A lista de agentes está vazia no cache (runtimes offline = agentes "sem heartbeat")
  levando o diálogo a tentar `DELETE` direto e receber 409.

---

## 5. UX segura proposta (para os 2 runtimes offline com agentes ativos)

### Por que o owner precisa de opção de reassignment (não só archive)

Os 8 agentes (3 claude + 5 kiro) nos runtimes offline têm **histórico de tasks**
(123 linhas de `task_usage`). Arquivar é destrutivo para agentes com trabalho em
andamento. A UX segura apresenta **duas opções**:

---

### Opção A — Reassign agents to active runtime (recomendada)

**Quando usar:** o owner quer manter os agentes operacionais, apenas mover para
o runtime online correto do mesmo provider.

**Fluxo UX proposto:**

```
┌─────────────────────────────────────────────────────────────┐
│ ⚠️  "kiro (offline)" has 5 active agents                     │
│                                                              │
│  These agents are bound to an offline runtime and cannot    │
│  run tasks. Choose what to do:                               │
│                                                              │
│  ● Reassign to active runtime        ← recomendado          │
│    Move all 5 agents to:                                     │
│    ┌─────────────────────────────────────────┐               │
│    │ ▸ kiro (online) — kiro-cli 2.13.0       │               │
│    │   6d0d721a · last seen 10:37            │               │
│    └─────────────────────────────────────────┘               │
│    Agents keep their history. No tasks cancelled.            │
│                                                              │
│  ○ Archive all agents & delete runtime   ← irreversível      │
│    5 agents archived · queued tasks cancelled                │
│    [ ] I confirm archiving: [agent1], [agent2], ...          │
│                                                              │
│  [ Cancel ]                    [ Reassign & Delete ]         │
└─────────────────────────────────────────────────────────────┘
```

**API para reassignment:** `PATCH /api/agents/:id` com `{ runtime_id: "<novo_runtime>" }`
já existe (`handler/agent.go:L1094-L1211`) e inclui guard de concorrência (409
se `runtime_id` mudou entre fetch e submit).

**Sequência:**
1. Frontend chama `PATCH /api/agents/:id { runtime_id: "6d0d721a..." }` para cada
   um dos 5 (ou 3) agentes — pode ser paralelo.
2. Após todos com 200, chama `DELETE /api/runtimes/eb41a0c9...`
3. Com agentes removidos, `ListActiveAgentsByRuntime = 0` → DELETE passa.

**Nenhuma migration necessária.** API de reassignment já existe.

---

### Opção B — Archive agents & delete runtime (cascade existente)

Já implementada via `POST .../archive-agents-and-delete`. O que falta é o
diálogo exibi-la claramente para runtimes offline (onde o cache de agentes pode
mostrar os agentes como "sem runtime online", enganando a detecção de modo).

**Fix no frontend:** garantir que `cachedActiveAgents` inclua agentes cujo
`runtime_id` corresponde ao runtime sendo deletado, **independentemente** do
status online/offline do runtime. A condição atual é:
```ts
// delete-runtime-dialog.tsx
agents.filter((a) => a.runtime_id === runtime.id && !a.archived_at)
```
Esta condição está correta — o problema pode ser que o cache não tem os agentes
dos runtimes offline se a query de `agentListOptions` filtra por runtime online.
Verificar `agentListOptions` no bundle correto pós-ORQ-26.

---

### Opção C — Disable runtime without delete (baixo impacto imediato)

Se o owner não quer tomar decisão agora: marcar o runtime como `offline`
(já está) e garantir que o daemon nunca mais tente registrá-lo. Nenhuma
ação necessária — o status já é `offline`. Os agentes continuam inacessíveis
para novas tasks mas seu histórico é preservado indefinidamente.

---

## 6. Testes relevantes (existentes)

```
packages/views/runtimes/components/delete-runtime-dialog.test.tsx
packages/core/runtimes/mutations.ts (useDeleteRuntime, useArchiveAgentsAndDeleteRuntime)
packages/core/permissions/rules.test.ts (canEditRuntime)
```

Os testes existentes cobrem os modos light/cascade/plan-changed. Nenhum teste
novo é **inevitável** para o fluxo de reassignment (já coberto pelos testes de
`PATCH /api/agents/:id`).

---

## 7. Resumo executivo — o que impede delete e o que fazer

| Bloqueio | Causa literal | Resolução |
|---|---|---|
| 409 `runtime_has_active_agents` | `agent.runtime_id ON DELETE RESTRICT` (migration 004) — 8 agentes ativos nos 2 runtimes offline | Reassign (opção A) ou archive-cascade (opção B) |
| Histórico de 123 task_usage | Sem FK direta em runtime; sobrevive via agent_task_queue.runtime_id ON DELETE CASCADE | Reassign preserva; archive-cascade apaga as tasks via cascade |
| Possível 2º bloqueio de squads | `CountActiveSquadsWithArchivedLeadersByRuntime` | Verificar antes de deletar; provavelmente zero para estes runtimes |

**Recomendação:** Opção A (reassign para runtime online do mesmo provider) para
os 5 agentes kiro → `6d0d721a` e os 3 agentes claude → nenhum runtime claude
online disponível (verificar se existe ou criar). Se não houver runtime claude
online, os 3 agentes claude devem ir para archive (Opção B limitada).

---

## Arquivos inspecionados (read-only)

- `internal/handler/runtime.go` L537-870 (DeleteAgentRuntime, ArchiveAgentsAndDeleteRuntime)
- `migrations/004_agent_runtime_loop.up.sql` L72-86 (FKs RESTRICT/CASCADE)
- `packages/views/runtimes/components/delete-runtime-dialog.tsx` L1-120
- PostgreSQL: `agent_runtime`, `agent`, `agent_task_queue`, `task_usage` (queries read-only)
