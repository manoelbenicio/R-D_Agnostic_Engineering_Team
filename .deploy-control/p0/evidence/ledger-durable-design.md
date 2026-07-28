# Ledger Durável — Design com TTL 24h

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:01Z  
**modo**: análise e design — NENHUM arquivo do repo editado  
**escritor exclusivo**: Codex56-TL w5:pC  

---

## 1. Problema raiz (evidência literal)

### daemon.go:301-302 — registry criado em memória, zerado a cada restart
```go
// daemon.go:301
d.commitLedgers = commitledger.NewLedgerRegistry()
d.commitAckHandler = commitledger.NewAckHandler(d.commitLedgers, logger)
```

### replay_gate.go:95-99 — `LedgerRegistry.ledgers` é map puro em RAM
```go
// replay_gate.go:95
type LedgerRegistry struct {
    mu      sync.RWMutex
    ledgers map[string]*Ledger  // SOMENTE memória; zerado no restart
}
```

### service/task.go:1666 — hook não configurado no backend → fail closed
```go
// service/task.go:1666
if err := commitledger.CheckOrAllow(s.ReplayGateHook, parentTaskID); err != nil {
    // ReplayGateHook == nil → "replay gate hook not configured; fail closed"
    // Resultado: TODOS os reruns automáticos bloqueados após restart
}
```

**Raiz**: `ReplayGateHook` em `TaskService` (processo do backend/handler) nunca é wired 
(`cmd/server/main.go` não atribui `ReplayGateHook`). O registry vive no processo do daemon; 
o hook é consultado no processo do backend. São processos separados — o registry não atravessa.

---

## 2. O que precisa ser persistido

A única estrutura relevante para o gate é `DurableSummary` (ledger.go:111-120):

```go
type DurableSummary struct {
    EverHadToolUse    bool   // qualquer tool_use registrado
    EverDefinite      bool   // qualquer tool_result recebido
    EverAmbiguous     bool   // crash/timeout antes de tool_result
    EverSaturated     bool   // overflow → fail closed
    HighestSeqSeen    int64
    TotalToolUseCount int64
}
```

`BlocksReplay()` — o único predicado do gate — só lê esses 6 campos. 
Entradas individuais (`Entry.Token`, `Entry.State`) são necessárias para auditoria 
mas **não** para a decisão de gate. Isso reduz o problema: persistir apenas o summary, 
não o ledger inteiro.

---

## 3. Schema mínimo (tabela no Postgres já existente no backend)

```sql
-- Nova tabela: task_ledger_summary
-- TTL: deleted_at setado por job quando created_at < NOW() - INTERVAL '24 hours'
CREATE TABLE task_ledger_summary (
    task_id          UUID        PRIMARY KEY,  -- correlação pseudônima (não exposta em UI)
    schema_version   INT         NOT NULL DEFAULT 1,
    ever_had_tool_use   BOOLEAN NOT NULL DEFAULT FALSE,
    ever_definite       BOOLEAN NOT NULL DEFAULT FALSE,
    ever_ambiguous      BOOLEAN NOT NULL DEFAULT FALSE,
    ever_saturated      BOOLEAN NOT NULL DEFAULT FALSE,
    highest_seq_seen    BIGINT  NOT NULL DEFAULT 0,
    total_tool_use_count BIGINT NOT NULL DEFAULT 0,
    recorded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- primeira gravação
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- última transição
    expires_at       TIMESTAMPTZ NOT NULL                 -- = recorded_at + 24h, indexado
);
CREATE INDEX ON task_ledger_summary (expires_at);  -- TTL sweep
```

**Invariante de imutabilidade**: colunas booleanas são escritas apenas com `OR` lógico 
(`ever_had_tool_use = ever_had_tool_use OR $new`). Nunca retrocedem. Schema version mismatch 
→ `ever_saturated = TRUE` imediatamente.

**TTL**: `expires_at = recorded_at + INTERVAL '24 hours'` (alinhado a `TerminalTTL` 
em `ledger.go:39`). Job ou `DELETE WHERE expires_at < NOW()` varrendo a cada hora. 
Tarefas concluídas há mais de 24h têm gate bloqueado por `ledger == nil` (correto — 
replay automático de tarefa velha nunca é safe).

---

## 4. Pontos de gravação

### 4.1 Pré-tool_use — daemon.go:4357
```go
// daemon.go:4357 — ANTES de invocar o tool
toolToken, _ = taskLedger.RecordToolUse(msg.CallID, int64(s))
// → ADICIONAR: persistir DurableSummary com ever_had_tool_use=true
//   via upsert idempotente (ON CONFLICT DO UPDATE SET ... WHERE NOT ever_had_tool_use)
```

**Por que pré**: se o daemon crashar entre `RecordToolUse` e o tool executar, 
o summary gravado com `ever_had_tool_use=true` garante que o gate bloqueie no restart. 
Conservador, mas correto — melhor falso-bloqueio que falso-permissão.

### 4.2 Pós-tool_result — daemon.go:4399
```go
// daemon.go:4399 — após tool_result recebido
_ = taskLedger.RecordToolResult(toolToken)
// → ADICIONAR: upsert com ever_definite=true
```

### 4.3 Defer de encerramento — daemon.go:4199 (MarkAllUnresolvedAmbiguous + Close)
```go
// daemon.go:4199
taskLedger.MarkAllUnresolvedAmbiguous()  // qualquer tool_use sem result → ambiguous
taskLedger.Close()
// → ADICIONAR: upsert final com ever_ambiguous se summary.EverAmbiguous, updated_at=NOW()
```

**Frequência**: 3 pontos por task, não por turn — sem hot-path de escrita por mensagem.

---

## 5. Wiring: como o backend consulta (sem atravessar processo)

O backend (`TaskService`) não acessa o registry do daemon (processos separados).
Com a tabela acima, `ReplayGateHook` é reimplementado no backend como:

```go
// Novo: DatabaseReplayGateHook (no pacote service ou handler)
type DatabaseReplayGateHook struct {
    DB *db.Queries   // já existente — mesmo Queries usado pelo TaskService
}

func (h *DatabaseReplayGateHook) Check(correlationID string) error {
    summary, err := h.DB.GetTaskLedgerSummary(ctx, taskID)
    if err != nil || summary == nil {
        return fmt.Errorf("%w: ledger unavailable; fail closed", commitledger.ErrReplayBlocked)
    }
    if summary.SchemaVersion != commitledger.SchemaVersion {
        return fmt.Errorf("%w: schema mismatch; fail closed", commitledger.ErrReplayBlocked)
    }
    // Reusar a lógica existente de DurableSummary.BlocksReplay()
    ds := commitledger.DurableSummary{
        EverHadToolUse: summary.EverHadToolUse,
        EverDefinite:   summary.EverDefinite,
        EverAmbiguous:  summary.EverAmbiguous,
        EverSaturated:  summary.EverSaturated,
    }
    if ds.BlocksReplay() {
        return fmt.Errorf("%w: tool activity recorded", commitledger.ErrReplayBlocked)
    }
    return nil
}
```

`TaskService.ReplayGateHook` recebe `DatabaseReplayGateHook` em `cmd/server/main.go`.
O daemon continua usando o registry em memória para performance durante a task ativa — 
a tabela é o fallback durável consultado somente no `MaybeRetryFailedTask`.

---

## 6. Invariante de telemetria — não autoriza replay

O campo `TotalToolUseCount` e `HighestSeqSeen` são gravados para auditoria/observabilidade.
**Não participam de `BlocksReplay()`** (ledger.go:113 — função só lê os 4 booleans).
A tabela não expõe esses campos na query de gate; a query de gate retorna apenas os 4 booleans.
Telemetria é gravada em tabela separada (`task_usage`) e não pode modificar `ever_*` 
(colunas diferentes, escrita separada).

Comentário de código obrigatório no `DatabaseReplayGateHook.Check`:
```go
// INVARIANTE: apenas os 4 campos ever_* decidem o gate.
// TotalToolUseCount e HighestSeqSeen são auditoria — nunca lidos aqui.
// Telemetria (task_usage) não tem permissão de escrita nesta tabela.
```

---

## 7. Impacto em MaybeRetryFailedTask — service/task.go:1666

Caminho atual:
```
MaybeRetryFailedTask → CheckOrAllow(s.ReplayGateHook, taskID)
  → hook == nil → ErrReplayBlocked (falha em todos os reruns automáticos)
```

Caminho após patch:
```
MaybeRetryFailedTask → CheckOrAllow(s.ReplayGateHook, taskID)
  → DatabaseReplayGateHook.Check(taskID)
    → GetTaskLedgerSummary(taskID)
      → task sem tool_use: summary.ever_had_tool_use=false → ReplayAllowed
      → task com tool_use: ever_had_tool_use=true → ErrReplayBlocked (correto)
      → task expirada (>24h) ou não encontrada → ErrReplayBlocked (correto)
```

**ORQ-12/13/17/18/21/23**: com `ever_had_tool_use=true` (57/83/31/45/57/87 tool_use reais), 
o gate continuaria bloqueando reruns automáticos — **comportamento correto**. 
O rerun manual via `multica issue rerun` bypassa o gate por design 
(`RerunIssue` chama `EnqueueTaskForIssue` diretamente, não `MaybeRetryFailedTask`).

---

## 8. Mudanças necessárias (resumo para o escritor exclusivo)

| Arquivo | Mudança | Linhas referência |
|---|---|---|
| `db/queries/*.sql` | Adicionar `task_ledger_summary` CRUD (upsert, get, TTL sweep) | nova migration |
| `daemon/daemon.go` | Nos 3 pontos de gravação (4357, 4399, 4199): chamar client HTTP `/api/daemon/tasks/{id}/ledger-summary` (ou canal interno) para persistir summary | daemon.go:4357, 4399, 4199 |
| `handler/` | Endpoint `PUT /api/daemon/tasks/:id/ledger-summary` recebe DurableSummary do daemon | novo handler |
| `service/task.go` | `DatabaseReplayGateHook` implementado; `NewTaskService` recebe `db.Queries`; `main.go` wires o hook | task.go:42, main.go |
| `commitledger/replay_gate.go` | Nenhuma mudança — lógica de gate reutilizada via `DurableSummary.BlocksReplay()` | L152-205 inalterado |

**Alternativa mais simples** (sem novo endpoint HTTP daemon→backend): o daemon grava 
diretamente no Postgres do backend via string de conexão já disponível em `daemon.go` 
(o daemon já conhece o DB — verifica em `d.cfg`). Evita roundtrip HTTP mas acopla o daemon ao schema do backend.

---

## 9. O que este design NÃO resolve (escopo explícito)

- ORQ-26 (bug `uploadFile` / schema.ts:56) — escopo diferente.
- Reruns manuais das ORQ-12/13/17/18/21/23 — permanecem OWNER/AMBÍGUO; este design não autoriza.
- Race entre `MaybeRetryFailedTask` e gravação do daemon — mitigado pelo pré-tool_use 
  (gravar antes do tool executar), não eliminado. Janela residual: tarefa falha antes 
  do primeiro tool_use (ORQ-15/16) — nesse caso `ever_had_tool_use=false` → gate permite, 
  o que é correto.

