# Cost Accounting Design — ORQ-12, 13, 14
**Autor:** Antigravity Opus48#C / w8:p1  
**Data:** 2026-07-27T11:00Z  
**Para:** Codex56-TL (escritor único) aplicar  
**Status:** PROPOSTA — nenhum arquivo editado, nenhum deploy

---

## 1. `account_id` em `task_usage`

### Estado atual — schema (migration 032)

```sql
-- migrations/032_task_usage.up.sql
CREATE TABLE task_usage (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id           UUID NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL DEFAULT '',
    model             TEXT NOT NULL,
    input_tokens      BIGINT NOT NULL DEFAULT 0,
    output_tokens     BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (task_id, provider, model)
);
```

Sem `account_id`. A tabela `accounts` existe desde migration 123 com
`account_id UUID PK`, `vendor TEXT`, `home_dir TEXT`, etc.

### Ponto de inserção — `handler/daemon.go:2101`

```go
// handler/daemon.go:L2101-2109
if err := h.Queries.UpsertTaskUsage(r.Context(), db.UpsertTaskUsageParams{
    TaskID:           parseUUID(taskID),
    Provider:         provider,
    Model:            u.Model,
    InputTokens:      u.InputTokens,
    OutputTokens:     u.OutputTokens,
    CacheReadTokens:  u.CacheReadTokens,
    CacheWriteTokens: u.CacheWriteTokens,
    // ← sem AccountID aqui
}); err != nil { ... }
```

`account_id` não está em `UpsertTaskUsageParams` nem no INSERT gerado pelo sqlc.

### Como `account_id` chega ao handler

`assignments` (migration 123) mapeia `agent_id → account_id`. O handler já
carrega `task.RuntimeID` (L2091) para resolver provider. A mesma query pode
resolver o `account_id` via:

```sql
SELECT a.account_id
FROM assignments a
JOIN agent_task_queue atq ON atq.agent_id = a.agent_id
WHERE atq.id = $1  -- task_id
```

### Migration proposta — `127_task_usage_account_id.up.sql`

```sql
-- migrations/127_task_usage_account_id.up.sql
ALTER TABLE task_usage
    ADD COLUMN account_id UUID REFERENCES accounts(account_id) ON DELETE SET NULL;

-- índice para billing por conta
CREATE INDEX idx_task_usage_account_id ON task_usage(account_id)
    WHERE account_id IS NOT NULL;

-- índice composto para rollup por conta+provider+model
CREATE INDEX idx_task_usage_account_provider_model
    ON task_usage(account_id, provider, model)
    WHERE account_id IS NOT NULL;
```

`NULLABLE` + `ON DELETE SET NULL`: linhas históricas e linhas de accounts
deletadas não quebram queries. Sem `NOT NULL` até backfill completar.

**Down:**
```sql
-- migrations/127_task_usage_account_id.down.sql
DROP INDEX IF EXISTS idx_task_usage_account_provider_model;
DROP INDEX IF EXISTS idx_task_usage_account_id;
ALTER TABLE task_usage DROP COLUMN IF EXISTS account_id;
```

### Mudança em `handler/daemon.go:2101`

Antes (L2085-2109) o loop já lê `task.RuntimeID`. Adicionar antes do loop:

```go
// ANTES do loop for _, u := range req.Usage
var taskAccountID *uuid.UUID
if row, err := h.Queries.GetTaskAccountID(r.Context(), parseUUID(taskID)); err == nil {
    taskAccountID = &row
}

// DENTRO do loop, no UpsertTaskUsageParams:
db.UpsertTaskUsageParams{
    ...
    AccountID: pgtype.UUID{Bytes: *taskAccountID, Valid: taskAccountID != nil},
}
```

Nova query sqlc necessária (`pkg/db/queries/task_usage.sql`):
```sql
-- name: GetTaskAccountID :one
SELECT a.account_id
FROM assignments a
JOIN agent_task_queue atq ON atq.agent_id = a.agent_id
WHERE atq.id = $1;
```

UNIQUE constraint atual `(task_id, provider, model)` permanece — não precisa
incluir `account_id` porque um task pertence a um único account no momento da
execução; a constraint continua correta.

---

## 2. Preço por tier de reasoning — `handler/daemon.go:2101`

### Estado atual

`UpsertTaskUsage` não grava `thinking_level`. O campo existe em `agent` (coluna
`thinking_level TEXT`) mas não em `task_usage`. O handler recebe `u.Model` com
o ID de rota (ex: `agy/gemini-3.5-flash-high`) — o tier já está codificado no
sufixo do modelo para os IDs do OmniRoute.

### Opção A — Extrair tier do model ID (sem nova coluna)

O sufixo `-high`, `-medium`, `-low`, `-thinking` no `model` field já identifica
o tier. O rollup e o frontend podem derivar o tier com:

```sql
CASE
    WHEN model LIKE '%-xhigh'    THEN 'xhigh'
    WHEN model LIKE '%-thinking' THEN 'thinking'
    WHEN model LIKE '%-high'     THEN 'high'
    WHEN model LIKE '%-medium'   THEN 'medium'
    WHEN model LIKE '%-low'      THEN 'low'
    ELSE 'standard'
END AS thinking_tier
```

**Vantagem:** zero schema change. **Desvantagem:** frágil para modelos sem
sufixo padrão (ex: `aug/claude-opus-4.6`).

### Opção B — Nova coluna `thinking_level TEXT` em `task_usage` (recomendada)

```sql
-- migrations/127_task_usage_account_id.up.sql (mesma migration, adicionar)
ALTER TABLE task_usage
    ADD COLUMN thinking_level TEXT NOT NULL DEFAULT '';
```

Handler passa o `thinkingLevel` do `task.Agent.ThinkingLevel` (daemon.go:L3774)
que já chega via `runTask`. O handler `ReportUsage` (`handler/daemon.go:2087`)
recebe a request do daemon via POST — precisa que o daemon inclua
`thinking_level` no payload. O daemon já tem `thinkingLevel` em escopo nessa
função (L3774, L3799).

**Mudança no payload de `ReportUsage` (daemon→handler):**
```go
// Struct de request em handler/daemon.go (buscar TaskUsageRequest)
type TaskUsageEntry struct {
    Provider         string `json:"provider"`
    Model            string `json:"model"`
    InputTokens      int64  `json:"input_tokens"`
    OutputTokens     int64  `json:"output_tokens"`
    CacheReadTokens  int64  `json:"cache_read_tokens"`
    CacheWriteTokens int64  `json:"cache_write_tokens"`
    ThinkingLevel    string `json:"thinking_level"` // ← ADICIONAR
}
```

E em `handler/daemon.go:2101`:
```go
db.UpsertTaskUsageParams{
    ...
    ThinkingLevel: u.ThinkingLevel,
}
```

---

## 3. Extração de tokens para AGY e Kiro — hoje registram zero

### Causa raiz confirmada no código

**codex.go:L1862-1886 — `extractUsageFromMap`:**
```go
// Tenta "usage", "token_usage", "tokens" — formato Codex/OpenAI
var usageMap map[string]any
for _, key := range []string{"usage", "token_usage", "tokens"} {
    if v, ok := data[key].(map[string]any); ok {
        usageMap = v
        break
    }
}
if usageMap == nil {
    return  // ← retorna sem contar nada
}
```

Este scanner é exclusivo do `codexClient`. AGY e Kiro usam `hermesClient`.

**hermes.go:L1190-1214 — `handleUsageUpdate` (AGY/Kiro):**
```go
func (c *hermesClient) handleUsageUpdate(data json.RawMessage) {
    var msg struct {
        Usage struct {
            InputTokens      int64 `json:"inputTokens"`      // camelCase ACP
            OutputTokens     int64 `json:"outputTokens"`
            TotalTokens      int64 `json:"totalTokens"`
            CachedReadTokens int64 `json:"cachedReadTokens"`
        } `json:"usage"`
    }
    if err := json.Unmarshal(data, &msg); err != nil {
        return
    }
    c.usageMu.Lock()
    // Usage updates from ACP are cumulative snapshots, so take the latest.
    if msg.Usage.InputTokens > c.usage.InputTokens {
        c.usage.InputTokens = msg.Usage.InputTokens
    }
    ...
```

O handler existe e está correto para ACP `usage_update` com payload camelCase.

**hermes.go:L421-432 — Condição de emissão da usage:**
```go
c.usageMu.Lock()
u := c.usage
c.usageMu.Unlock()

var usageMap map[string]TokenUsage
if u.InputTokens > 0 || u.OutputTokens > 0 || u.CacheReadTokens > 0 {
    // ← só envia se algum campo > 0
    model := opts.Model
    if model == "" { model = "unknown" }
    usageMap = map[string]TokenUsage{model: u}
}
```

**Também:** hermes.go:L380-385 acumula de `promptDone.usage`:
```go
c.usage.InputTokens += pr.usage.InputTokens
c.usage.OutputTokens += pr.usage.OutputTokens
c.usage.CacheReadTokens += pr.usage.CacheReadTokens
```

### Por que AGY e Kiro registram zero

O `usage_update` ACP **nunca chega** ou chega com campos todos zero. Duas causas:

**Causa A — Kiro CLI 2.x:** `session/prompt` retorna `stopReason=end_turn` mas
não emite `usage_update` notification antes de encerrar. A usage total fica em
zero porque `handleUsageUpdate` nunca é chamado com valores > 0.

**Causa B — AGY (hermesClient para antigravity):** O `promptDone` em L380-385
acumula a usage do response final. Se o campo `usage` da resposta ACP usa snake_case
(`input_tokens`) em vez de camelCase (`inputTokens`), o struct em L1192-1197
não deserializa e fica em zero.

### Fix proposto para AGY — `hermes.go:L1191-1197`

Adicionar snake_case como fallback no struct de deserialização:

```go
// ANTES (hermes.go:L1191-1197):
var msg struct {
    Usage struct {
        InputTokens      int64 `json:"inputTokens"`
        OutputTokens     int64 `json:"outputTokens"`
        TotalTokens      int64 `json:"totalTokens"`
        CachedReadTokens int64 `json:"cachedReadTokens"`
    } `json:"usage"`
}

// DEPOIS — struct separado com unmarshal manual ou dois campos:
type acp usage struct{}
// alternativa pragmática: tentar inputTokens; se zero, tentar input_tokens
```

Ou, mais robusto, extrair com `json.Number` e tentar ambos os formatos:

```go
func (c *hermesClient) handleUsageUpdate(data json.RawMessage) {
    var outer map[string]json.RawMessage
    if err := json.Unmarshal(data, &outer); err != nil { return }
    raw, ok := outer["usage"]
    if !ok { return }
    var flat map[string]any
    if err := json.Unmarshal(raw, &flat); err != nil { return }

    get := func(keys ...string) int64 {
        for _, k := range keys {
            if v, ok := flat[k]; ok {
                switch n := v.(type) {
                case float64: return int64(n)
                case int64:   return n
                }
            }
        }
        return 0
    }
    c.usageMu.Lock()
    in  := get("inputTokens",  "input_tokens",  "input")
    out := get("outputTokens", "output_tokens", "output")
    cr  := get("cachedReadTokens", "cache_read_tokens", "cached_read_tokens")
    if in  > c.usage.InputTokens      { c.usage.InputTokens      = in  }
    if out > c.usage.OutputTokens     { c.usage.OutputTokens     = out }
    if cr  > c.usage.CacheReadTokens  { c.usage.CacheReadTokens  = cr  }
    c.usageMu.Unlock()
}
```

### Fix proposto para Kiro — extrair usage do response final de `session/prompt`

**kiro.go:L331-342** — após `session/prompt` retornar com sucesso, o result
JSON pode conter usage. Adicionar extração similar à de `extractACPCurrentModelID`:

```go
// kiro.go — após L301 c.request(runCtx, "session/prompt", ...)
// Se err == nil, extrair usage do result (hoje descartado):
if err == nil {
    // result já foi consumido pelo promptDone via hermesClient
    // A usage do Kiro vem exclusivamente via usage_update notifications.
    // Se o Kiro CLI não emite usage_update, não há ponto de extração
    // sem modificar o protocolo — documentar como gap do Kiro CLI.
}
```

**Conclusão para Kiro:** se o Kiro CLI 2.x não emite `usage_update`, não há
extração possível sem mudança no CLI ou parsing de stderr. Gap a documentar para
o vendor.

---

## Resumo de mudanças para o escritor único aplicar

| Item | Arquivo | Tipo |
|---|---|---|
| Schema `account_id` | `migrations/127_task_usage_account_id.up.sql` | Nova migration |
| Schema `thinking_level` | mesma migration | Nova coluna |
| Query `GetTaskAccountID` | `pkg/db/queries/task_usage.sql` | Nova query sqlc |
| `UpsertTaskUsageParams` | `pkg/db/generated/` | Regenerar sqlc |
| Wiring account_id | `internal/handler/daemon.go:2101` | 1 query + campo |
| Wiring thinking_level | `internal/handler/daemon.go:2101` | Campo no params |
| Usage multi-format | `pkg/agent/hermes.go:L1190-1214` | Fix deserialização |
| Kiro usage gap | documentar | Vendor limitation |

Nenhum restart de daemon necessário para as mudanças de schema + handler.
go test / go vet / go build devem permanecer verdes (sem mudança de lógica de
negócio, apenas expansão de struct e query).
