# T5 — Kiro tool_use/tool_result Telemetry Analysis (v2 — CORRIGIDO)
**Autor:** Antigravity Opus48#C / w8:p1  
**Data original:** 2026-07-27T10:13Z | **Corrigido:** 2026-07-27T10:14Z  
**Correção solicitada por:** Codex56#B w7:p4  
**Status:** READ-ONLY — nenhum código editado

> **ATENÇÃO:** A versão anterior (v1) continha uma conclusão incorreta.
> A v1 afirmava que `tool_result` com output vazio poderia ser filtrado/ignorado
> pelo persistidor. **Isso é falso.** Correção abaixo.

---

## Linhas literais que determinam o comportamento

### 1. `tool_use` é emitido NO INÍCIO — antes de completed

**`hermes.go:L906-924` — `handleToolCallStart`:**
```go
// Hermes pre-populates rawInput on the initial tool_call — emit
// MessageToolUse immediately so the UI can show the tool invocation live.
if rawInput != nil {
    c.trackTool(msg.ToolCallID, &pendingToolCall{
        toolName: toolName, input: rawInput, emitted: true,
    })
    if c.onMessage != nil {
        c.onMessage(Message{
            Type:   MessageToolUse,   // ← emitido no tool_call, NÃO no completed
            Tool:   toolName,
            CallID: msg.ToolCallID,
            Input:  rawInput,
        })
    }
    return   // ← sai sem esperar tool_call_update[completed]
}
```

`MessageToolUse` é emitido quando o frame `tool_call` chega com `rawInput` —
**antes** de qualquer `tool_call_update`. Prova "ferramenta iniciada", não "concluída".

### 2. `tool_result` é emitido SEMPRE quando tool_call_update[completed] chega

**`hermes.go:L981-998` — `handleToolCallUpdate` (completion path):**
```go
// Completion: emit any deferred MessageToolUse first, then the result.
pending := c.takePendingTool(msg.ToolCallID)
c.emitDeferredToolUse(pending, msg.ToolCallID, title, msg.Kind, rawInput)

output := msg.RawOutput
if output == "" { output = msg.Output }
if output == "" { output = extractACPToolCallText(msg.Content) }
if c.onMessage != nil {
    c.onMessage(Message{
        Type:   MessageToolResult,   // ← emitido SEMPRE quando completed chega
        CallID: msg.ToolCallID,
        Output: output,              // pode ser "" — mas o emit ocorre
    })
}
```

`MessageToolResult` é emitido **incondicionalmente** quando `tool_call_update
[status=completed]` é recebido. Output vazio não suprime o emit.

### 3. daemon.go persiste tool_result incondicionalmente — sem filtro por output vazio

**`daemon.go:L4370-4411` — persistência de `MessageToolResult`:**
```go
case agent.MessageToolResult:
    // ... decrement inFlightTools ...
    s := seq.Add(1)
    output := msg.Output
    if len(output) > 8192 {
        output = output[:8192]   // apenas trunca se muito longo
    }
    // ... resolve toolName ...
    if msg.CallID != "" {
        toolToken := taskLedger.TokenizeCallID(msg.CallID)
        _ = taskLedger.RecordToolResult(toolToken)   // ← ledger registra
    }
    mu.Lock()
    batch = append(batch, TaskMessageData{
        Seq:    int(s),
        Type:   "tool_result",   // ← persiste em task_message SEMPRE
        Tool:   toolName,
        Output: output,          // "" é persistido normalmente
    })
    mu.Unlock()
```

Não há `if output == "" { continue }`. Output vazio é persistido como linha
`tool_result` normal. O `CommitLedger` registra o token de conclusão.

---

## Diagnóstico corrigido de 72 tool_use / 0 tool_result

Dado que:
- `tool_result` é emitido sempre que `tool_call_update[completed]` chega (hermes.go:L992)
- `tool_result` é sempre persistido, mesmo com output vazio (daemon.go:L4405-4411)

A única explicação para **0 `tool_result`** em 72 `tool_use` é:

> **O Kiro CLI não entregou `tool_call_update[status=completed]` pareados
> para as 72 ferramentas invocadas** — ou esses frames chegaram fora da
> janela `streamingCurrentTurn=true` (kiro.go:L123-128) e foram descartados.

Os 72 `tool_use` provam que o Kiro CLI **iniciou** 72 chamadas de ferramenta.
Não provam que alguma delas concluiu. O estado de cada ferramenta é **ambíguo**.

---

## Implicação para replay safety

| Sinal | Significado correto |
|---|---|
| `tool_use` sem `tool_result` pareado | Ferramenta **iniciada**, estado **ambíguo** — pode ter concluído, falhado ou sido interrompida no meio |
| `tool_result` presente | Ferramenta concluiu e o daemon registrou o resultado no ledger |
| 72/0 em tasks interrompidas | 72 ferramentas iniciadas sem confirmação de conclusão |

**Replay continua bloqueado pelo replay gate — e corretamente.**
O gate bloqueia pela presença de `tool_use` sem `tool_result` correspondente
no ledger: exatamente o sinal de "efeito potencialmente parcial ou duplicável".
Reexecutar cegamente pode duplicar escrita em disco ou chamadas externas cujo
estado real é desconhecido.

---

## Conclusão corrigida

> **A ausência de `tool_result` NÃO prova conclusão silenciosa.**
> Em ACP via hermesClient, `tool_use` significa "ferramenta iniciada/atividade
> observada". `tool_result` significa "conclusão confirmada e registrada no ledger".
> 72 `tool_use` / 0 `tool_result` indica que o Kiro CLI não entregou frames
> `tool_call_update[completed]` — o estado das 72 ferramentas é **ambíguo**.
> O replay gate bloquear é o comportamento correto e seguro.

---

## Erro da versão anterior (v1)

A v1 afirmava: *"MessageToolResult emitido com Output='' — o persistidor provavelmente
filtra ou agrupa results vazios"*. **Isso é falso.** `daemon.go:L4405-4411` persiste
toda linha `tool_result` independentemente do conteúdo do output. A v1 não verificou
o código do persistidor antes de concluir.

---

## Arquivos inspecionados (read-only)

- `pkg/agent/kiro.go` L118-139, L123-128
- `pkg/agent/hermes.go` L906-924, L970-999
- `internal/daemon/daemon.go` L4370-4411
