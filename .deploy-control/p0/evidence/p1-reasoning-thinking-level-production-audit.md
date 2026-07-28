# P1 — Auditoria de reasoning / `thinking_level`: produção real × código (READ-ONLY)

- Executor único: Codex56#A (`w7:p3`) · UTC 2026-07-27T17:34Z · GATE 0 e DELTA GATE 0 aceitos
- Fontes de verdade comparadas separadamente, conforme a decisão do TL:
  **(A)** deploy em execução no ORQ1, imagem `multica-backend:agy-status-20260727T102815Z`
  (criada `2026-07-27T10:30:51Z`, sem labels) + schema aplicado no Postgres de produção;
  **(B)** branch `integration/dev-transition-candidate-20260719`, HEAD `0cb8aeb`;
  **(C)** worktrees não mergeadas com trabalho de reasoning.
- Autorizações usadas: DB de produção somente leitura via `ssh ORQ1 + docker exec psql -c "SELECT …"`,
  colunas nomeadas e não secretas, sem export, sem DML/DDL. Nenhum arquivo de ORQ-13 tocado.
- Runtime de UI/API: **NÃO_VERIFICADO** — a API responde 401 desde o restart de 15:56Z
  (`APP_ENV=production`, `MULTICA_LOCAL_AUTH_BYPASS=false`); reverifico após login do Kiro.

## 1. Estado medido no banco de produção (A)

```text
migrations aplicadas   163 linhas, max = 126_runtime_profile_protocol_family_native_runtimes
agent.thinking_level   EXISTE (text)            -> migration 095 aplicada
task_usage.thinking_level  AUSENTE (0 colunas)  -> migration 127 não aplicada
```

Colunas com semântica de reasoning em todo o schema: **apenas** `agent.thinking_level`. As demais
correspondências (`failure_reason`, `wait_reason`, `drop_reason`, …) não têm relação.

Inventário de agentes e runtimes:

```text
agent.thinking_level:  NULL -> 20 de 20 agentes   (nenhum valor configurado em produção)
agent_runtime.provider: antigravity 1 | claude 1 | codex 1 | kiro 2
agent.model (distintos): gpt-5.6-sol 4, claude-opus-5 2, claude-opus-4-6-thinking 2,
                         gemini-3.6-flash-high 2, Opus-4.8 2, Opus 4.8 1, gpt-5.5 1,
                         5.6-Sol 1, '' 1, NULL 4
runtime_profile: 0 linhas
```

Dois fatos que importam: **nenhum** agente em produção tem `thinking_level` definido, e **não existe
runtime registrado** para gemini, kimi, opencode, codebuddy ou cline — só antigravity, claude, codex e
kiro.

## 2. Suporte efetivo por CLI, no código da branch (B)

| CLI | mecanismo nativo | evidência | enum de validação |
|---|---|---|---|
| **Codex** | `applyCodexReasoningEffort` em turn/resume/start params | `pkg/agent/codex.go:771,1002,1033,1065` | `none, minimal, low, medium, high, xhigh` |
| **Kiro** | flag `--effort <nível>` | `pkg/agent/kiro.go:62-63` | `low, medium, high, xhigh, max` |
| **Claude** | flag `--effort`, com o arg bloqueado para custom args | `pkg/agent/claude.go:676-704` | `low, medium, high, xhigh, max` |
| **Gemini** | arquivo de settings temporário (`writeGeminiThinkingOverride`) | `pkg/agent/gemini.go:44-50` | `minimal, low, medium, high` |
| **Kimi** | env `KIMI_MODEL_THINKING_EFFORT` | `pkg/agent/kimi.go:66-70,304-308` | `low, medium, high, xhigh, max` |
| **OpenCode** | flag `--variant <nome>`, nome validado dinamicamente | `pkg/agent/opencode.go:22,68-69`; `thinking.go:938` | catálogo dinâmico (`opencode.json`) |
| **Antigravity** | **nenhum** | `pkg/agent/antigravity.go` não referencia `ThinkingLevel` | **ausente do enum** → `IsKnownThinkingValue("antigravity", x) = false` |

Enum estático completo em `pkg/agent/thinking.go:875-935`: `claude, codex, codebuddy, kiro, kimi,
gemini, cline` (+ `opencode` por validação dinâmica). Antigravity está registrado como provider de CLI
(`pkg/agent/agent.go:169,220,241` — `agy -p`), mas **não** no registro de reasoning.

## 3. API, persistência e UI (B)

- **Criação**: `internal/handler/agent.go:768-769` valida `thinking_level` contra o enum do provider e
  responde 400 com mensagem nomeando o runtime; persiste em `:823` como `pgtype.Text`.
- **Atualização**: tri-state documentado (`:897-903`, `:1146`) — ausente = não muda, `null` = limpa,
  valor = grava; resolve o runtime uma vez para validar (`:1075-1077`).
- **Leitura**: exposto como `thinking_level` no JSON do agente (`:62-65`, `:140`).
- **UI**: `packages/views/agents/components/inspector/thinking-picker.tsx`,
  `thinking-prop-row.tsx`, `create-thinking-field.tsx`, com testes. O gate de exibição é correto:
  `thinking-prop-row.tsx:50-51` esconde a linha quando `supported_levels` está vazio **e** nada está
  persistido — logo Antigravity não exibe seletor, desde que o backend não anuncie níveis.

## 4. Dispatch — aqui está o defeito que explica os 20 NULL

`internal/daemon/daemon.go:3772-3799` lê `task.Agent.ThinkingLevel`, chama
`d.agentBrain.validateThinking(plan, thinkingLevel)` e, em erro, aborta com
`agentBrainAdmissionError{class: "thinking_not_approved"}`.

`internal/daemon/brain_integration.go:520-537`:

```go
if plan == nil { return nil }                      // caminho nativo: backend do provider valida
if strings.TrimSpace(thinking) != "" {
    return runtimeenv.ErrThinkingNotApproved       // <-- rejeição INCONDICIONAL sob gateway
}
policy, err := runtimeenv.NewGatewayModelPolicy([]runtimeenv.ApprovedGatewayModel{{
    Model: …, Protocol: …, CLIs: …,                // ThinkingLevels NUNCA é populado
}})
return policy.ValidateSelection(cli, model, "")    // <-- passa "" em vez de `thinking`
```

Consequência, com `AGENT_BRAIN_GATEWAY_REQUIRED=true` (que o README define como o caminho único do
produto): **qualquer** agente com `thinking_level` não vazio falha a admissão da task. O usuário grava
um valor que a API aceita como válido e a execução morre depois, com `thinking_not_approved`. A única
configuração que despacha é `thinking_level = NULL` — exatamente o que o banco mostra em 20 de 20
agentes. A infraestrutura de política existe (`runtimeenv/model.go:20,50-54,64-82` sabe validar níveis
por modelo), mas ninguém a alimenta.

## 5. Matriz FUNCIONAL / PARCIAL / NÃO DEPLOYED

| capacidade | deploy ORQ1 (A) | branch `0cb8aeb` (B) | veredito |
|---|---|---|---|
| coluna `agent.thinking_level` | presente (mig. 095) | presente | **FUNCIONAL** |
| API create/update com tri-state e validação por provider | binário construído da branch | `agent.go:768,823,897-903,1075,1146` | **FUNCIONAL no código / NÃO_VERIFICADO em runtime (401)** |
| seletor de UI + gate de visibilidade | idem | `thinking-picker/prop-row/create-field` | **FUNCIONAL no código / NÃO_VERIFICADO em runtime** |
| dispatch **nativo** por CLI: codex, kiro, claude, gemini, kimi, opencode | nenhum runtime desses três últimos existe em prod | flags/env/arquivo implementados | **PARCIAL** — funcional em código; em produção só codex, claude e kiro têm runtime |
| dispatch **via gateway** com `thinking_level` ≠ vazio | mesmo código | `brain_integration.go:527-528` | **NÃO FUNCIONAL** — rejeição incondicional; é o caminho obrigatório do produto |
| Antigravity com reasoning | runtime existe (1) | sem enum e sem uso em `antigravity.go` | **NÃO SUPORTADO** (por desenho ou por omissão — precisa ruling) |
| gemini / kimi / opencode / codebuddy / cline | **0 runtimes** | suporte implementado | **NÃO DEPLOYED** |
| `task_usage.thinking_level` (telemetria por task) | coluna ausente | ausente na branch | **NÃO DEPLOYED** — existe só em (C) |
| aprovação de nível por modelo no gateway (`ThinkingLevels`) | — | tipo existe, nunca populado | **NÃO DEPLOYED** |

## 6. Gaps deploy × branch × worktrees não mergeadas

- **(A) vs (B)**: coerentes. A imagem foi construída hoje 10:30Z e o schema aplicado tem max 126,
  igual ao máximo da branch. Nenhum gap de reasoning entre deploy e branch.
- **(C) worktree `gtl-i03-orq13-phase1`, branch `agent/opus48-b/orq-13-thinking-level`**: contém a
  migration `127_task_usage_thinking_level.{up,down}.sql` (untracked) e alterações em
  `internal/daemon/{daemon,types}.go`, `internal/handler/daemon.go`, `pkg/db/queries/task_usage.sql` e
  três arquivos `generated`. É **território ORQ-13**: apenas reporto, não toco.
- As worktrees `p0-*` e `agent-brain-p0-*` apareceram no meu filtro inicial por causa de
  `failure_reason`; conferi `p0-w1-central` e **nenhum** arquivo com `thinking` no diff. Não há trabalho
  de reasoning oculto nelas.

## 7. Correção mínima (recomendação, sem execução)

1. **Decidir e implementar a política de thinking no gateway** — `brain_integration.go:527-537`. Duas
   saídas coerentes, e a atual não é nenhuma delas:
   - **7a** popular `ApprovedGatewayModel.ThinkingLevels` a partir da política de rota/catálogo e
     passar `thinking` (não `""`) em `ValidateSelection`, removendo a rejeição incondicional; ou
   - **7b** se o gateway realmente não suporta reasoning ainda, **falhar na API** (create/update)
     enquanto `AGENT_BRAIN_GATEWAY_REQUIRED=true`, para o usuário não gravar um valor que mata a task
     depois. Hoje o produto aceita a configuração e falha só no dispatch.
   Escopo: 1 arquivo (`brain_integration.go`) para 7a, 2 arquivos para 7b. Não toca migrations,
   `queries` nem `generated` — sem colisão com ORQ-13.
2. **Antigravity**: ruling do owner. Se `agy` suporta effort, adicionar ao `providerThinkingEnums` e
   implementar em `antigravity.go`; se não, documentar como não suportado no mapa de criação de agentes
   (`builtin_skills/multica-creating-agents`), que hoje é silencioso.
3. **Runtimes ausentes**: gemini, kimi e opencode têm suporte pronto e zero runtime registrado. É
   decisão de operação, não de código — registrar runtime/profile se o objetivo é usar reasoning nesses
   CLIs.
4. **Telemetria por task**: manter em ORQ-13. Não criar migration paralela.

## 8. Não-alegações

- Não escrevi nada em banco: apenas `SELECT` em colunas nomeadas e não secretas, sem export.
- Não chamei a API autenticada (401), portanto **todo** o comportamento de UI/API em runtime está
  marcado NÃO_VERIFICADO; a avaliação é de código.
- Não executei task, não despachei agente, não alterei configuração de runtime, não toquei board.
- Não li segredo algum; não inspecionei valores de `custom_env`, `runtime_config` ou `mcp_config`.
- Não abri nem modifiquei arquivo de ORQ-13; a leitura do worktree foi `git diff --name-only` e
  `git status --porcelain`.
- Não verifiquei por execução que o binário em produção contém o código de reasoning: a inferência vem
  da data de build (10:30Z de hoje) e do schema aplicado (max 126), não de introspecção do binário.
