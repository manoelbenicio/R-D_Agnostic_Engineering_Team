# Contrato E2E de Reasoning (`thinking_level`) — Audit & Proposta GTL-06 (ORQ-20)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:34Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**modo**: READ-ONLY — NENHUM código, agentes, build ou estado de produção alterados  

---

## SEÇÃO CORREÇÃO GTL-06R (Retificação & Distinção Rígida de Estado)

Por determinação do General-TL, esta seção corrige os equívocos do relatório inicial:

1. **Retificação sobre Persistência em `task_usage`**:
   - **ERRADO (relatório anterior)**: Afirmava que `task_usage` já registrava `thinking_level`.
   - **CORRETO (comprovado por GTL-03 e audit do schema)**: A tabela `task_usage` **NÃO persiste `thinking_level`** atualmente. A coluna não existe em `task_usage` nem nas queries SQL do daemon/server. Essa persistência é uma **LACUNA / PROPOSTA A IMPLEMENTAR** necessária para a resolução da issue ORQ-13 ("Calcular preço por tier de reasoning").

2. **Ajuste para o Catálogo Live (11/19/11)**:
   - Removidos modelos hipotéticos e genéricos (`o3-mini`, `gpt-4o`, `claude-opus-4.8`).
   - A matriz e os defaults da frota foram alinhados aos modelos **REAIS** do catálogo descoberto (ex: os 11 modelos AGY medidos via `agy models`: `gemini-3.6-flash-high`, `gemini-3.6-flash-medium`, `gemini-3.6-flash-low`, `gemini-3.5-flash-high`, `gemini-3.5-flash-medium`, `gemini-3.5-flash-low`, `gemini-3.1-pro-high`, `gemini-3.1-pro-low`, `claude-sonnet-4-6`, `claude-opus-4-6-thinking`, `gpt-oss-120b-medium`).

3. **Separação Rígida entre Estado Implementado vs Estado Proposto**:

### A. Estado Atual Implementado no Código
- **Schema DB (`server/migrations/095_agent_thinking_level.up.sql`)**:
  - `ALTER TABLE agent ADD COLUMN thinking_level TEXT;` adiciona o campo à entidade `agent`.
- **API Backend (`server/internal/handler/agent.go`)**:
  - `CreateAgent` (L767-770) e `UpdateAgent` (L1159-1206) aplicam a validação tri-state:
    - Omitido: preserva o valor atual se válido para o runtime target.
    - String vazia `""`: executa `ClearAgentThinkingLevel` (remove override, usa default do CLI).
    - Valor literais: valida via `agent.IsKnownThinkingValue(provider, value)`. Rejeita valores desconhecidos com HTTP 400.
- **Validação de Enum (`server/pkg/agent/thinking.go:934`)**:
  - `providerThinkingEnums` contém o dicionário síncrono para `claude`, `codex`, `codebuddy`, `kiro`, `kimi`, `gemini`, `cline`.
- **Frontend UI (`packages/views/agents`)**:
  - `create-thinking-field.tsx` & `thinking-picker.tsx` consultam `thinking.supported_levels`.
  - Se `levels.length > 0`, exibe o picker. Se `levels.length === 0` (ex: AGY, onde o tier é embutido no modelo), o picker fica **oculto** e envia `""`.
- **Daemon Dispatch (`server/internal/daemon/daemon.go:3772-3801`)**:
  - Lê `task.Agent.ThinkingLevel`.
  - Executa `d.agentBrain.validateThinking(...)` como guard pré-execução.
  - Injeta em `agent.ExecOptions{ ThinkingLevel: thinkingLevel }`.
  - CLI execution drivers:
    - Claude: `--effort <level>` (`claude.go:709`)
    - Codex: `applyCodexReasoningEffort` (`codex.go:771, 1002, 1033`)
    - Kiro: `kiro-cli acp --effort <level>` (`kiro.go:63`)
    - Gemini: `writeGeminiThinkingOverride` (`gemini.go:45`)
    - Cline: `--thinking <level>` (`cline.go:63`)
    - Kimi: `KIMI_MODEL_THINKING_EFFORT` env (`kimi.go:70`)
    - OpenCode: `--variant <level>` (`opencode.go:69`)

### B. Estado Proposto / Lacunas a Implementar
- **Persistência em `task_usage`**: Adicionar coluna `thinking_level TEXT` na tabela `task_usage` e atualizar queries SQL do daemon e server (Requisito ORQ-13).
- **Tratamento de AGY em `providerThinkingEnums`**: Registrar `antigravity` em `providerThinkingEnums` permitindo apenas `""`, garantindo rejeição imediata se um cliente enviar um `thinking_level` explícito para AGY.
- **Inclusão do Mapa Completo do Catálogo Live 11/19/11**: Atualizar os anotadores em `thinking.go` para cobrir os nomes exatos de modelos retornados pelo OmniRoute no ORQ1/ORQ2.

---

## 1. Mapeamento do Fluxo Contratual E2E (`thinking_level`)

### 1.1 Discovery Payload (`daemon` -> `server` -> `UI`)
- **AGY (Antigravity)**: Descoberto via `agy models`. Retorna os 11 modelos medidos no ORQ2 com o tier de raciocínio incorporado no próprio ID (`gemini-3.6-flash-high`, `gemini-3.6-flash-medium`, `gemini-3.6-flash-low`, `gemini-3.5-flash-high`, `gemini-3.5-flash-medium`, `gemini-3.5-flash-low`, `gemini-3.1-pro-high`, `gemini-3.1-pro-low`, `claude-sonnet-4-6`, `claude-opus-4-6-thinking`, `gpt-oss-120b-medium`). `Model.Thinking` fica `nil`.
- **Codex**: Descoberto via `discoverCodexModels`. Projeta `SupportedLevels`: `none`, `minimal`, `low`, `medium`, `high`, `xhigh`.
- **Kiro**: Descoberto via `discoverACPModels` (`kiro-cli acp`). `annotateKiroThinking` associa os níveis `low`, `medium`, `high`, `xhigh`, `max`.

### 1.2 Formulário / UI Frontend (`views/agents`)
- `create-thinking-field.tsx` renderiza `ThinkingPicker` se `levels.length > 0`.
- Quando `levels.length === 0` (ex: AGY), o campo permanece **oculto** e o valor trafega como `""` ("Follow CLI config").

### 1.3 Persistência no Banco de Dados (`server/internal/handler` & `db`)
- Tabela `agent` possui a coluna `thinking_level`.
- HTTP `POST /api/agents` e `PATCH /api/agents/{id}` recebem `thinking_level`.
- `IsKnownThinkingValue` valida o enum no backend.

### 1.4 Dispatch e Execução (`daemon` -> CLI)
- Daemon lê `task.Agent.ThinkingLevel` e passa para `agent.ExecOptions`.
- Para AGY, o modelo é enviado verbatim (`agy --model <id>`), onde o tier já está embutido. Para Codex e Kiro, o valor de `ThinkingLevel` é injetado via RPC/CLI flag (`--effort`).

### 1.5 Custo e Contabilização (LACUNA ORQ-13)
- **Status atual**: `task_usage` **não** grava `thinking_level`.
- **Proposta**: Adicionar `thinking_level` na tabela `task_usage` para permitir precificação por tier de esforço.

---

## 2. Matriz de Níveis Válidos / Modelo / Provider (Catálogo Live)

| Provider | Modelo Exemplo (Catálogo Live) | Formato de Seleção | Níveis Suportados (`supported_levels`) | Sentinel / Default | Flag CLI / Mecanismo de Injeção |
|---|---|---|---|---|---|
| **antigravity (AGY)** | `gemini-3.6-flash-high`<br>`gemini-3.6-flash-medium`<br>`gemini-3.6-flash-low`<br>`gemini-3.1-pro-high`<br>`claude-opus-4-6-thinking` | **Modelo com Tier Embutido** | *(Nenhum picker UI — `Thinking=nil`)* | `""` | `agy --model <id_com_tier>` |
| **codex** | Modelos descobertos do runtime Codex (19 modelos live) | Campo `thinking_level` | `none`, `minimal`, `low`, `medium`, `high`, `xhigh` | `""` ("Follow CLI config") | `applyCodexReasoningEffort` (RPC/config) |
| **kiro** | Modelos descobertos do runtime Kiro (11 modelos live) | Campo `thinking_level` | `low`, `medium`, `high`, `xhigh`, `max` | `""` ("Follow CLI config") | `kiro-cli acp --effort <level>` |
| **claude** | `claude-sonnet-4-6` | Campo `thinking_level` | `low`, `medium`, `high`, `xhigh`, `max` | `""` ("Follow CLI config") | `claude --effort <level>` |

---

## 3. Proposta de Defaults Explícitos para os 10 Agentes Reais da Frota

Modelos selecionados exclusivamente a partir do catálogo live de 11 modelos AGY medidos e das instâncias Codex/Kiro online:

| # | Agente / Função da Frota | Provider | Modelo Proposto (Catálogo Live 11/19/11) | `thinking_level` Proposto | Justificativa de Alocação de Custo / Raciocínio |
|---|---|---|---|---|---|
| 1 | **Orchestrator / General-TL** (w5:pC / wB:p1) | `antigravity` | `claude-opus-4-6-thinking` | `""` *(embutido)* | Raciocínio máximo para orquestração global e decisões de arquitetura |
| 2 | **Core Backend Architect** (w7:p4) | `codex` | Modelo Codex com reasoning | `high` | Análise profunda de código Go, concorrência e transações DB |
| 3 | **Frontend & UI Lead** | `kiro` | Modelo Kiro com reasoning | `high` | Resolução de bugs complexos de componentes Next.js e estado React |
| 4 | **DB & Data Engineer** | `antigravity` | `gemini-3.1-pro-high` | `""` *(embutido)* | Projeção de schemas SQL, migrations e validação de consistência |
| 5 | **Security & IAM Auditor** | `antigravity` | `gemini-3.1-pro-low` | `""` *(embutido)* | Auditoria estrita de código e permissões de segurança com custo controlado |
| 6 | **Feature Developer A** | `antigravity` | `claude-sonnet-4-6` | `""` *(embutido)* | Implementação robusta de novas rotas e manipuladores |
| 7 | **Feature Developer B** | `antigravity` | `gemini-3.6-flash-high` | `""` *(embutido)* | Execução rápida de código com alto raciocínio Gemini 3.6 |
| 8 | **QA & Test Specialist** | `antigravity` | `gemini-3.6-flash-medium` | `""` *(embutido)* | Execução rápida de suítes de teste e verificação de regressão |
| 9 | **Log & Telemetry Auditor** | `antigravity` | `gemini-3.5-flash-medium` | `""` *(embutido)* | Varredura e parsing de logs de execução com consumo reduzido |
| 10 | **Docs & Maintenance** | `antigravity` | `gemini-3.5-flash-low` | `""` *(embutido)* | Atualização de documentação, evidências e checagens simples |

---

## 4. Validação Server-Side & Suíte de Testes Recomendada

### 4.1 Validação Server-Side
1. **Gate `IsKnownThinkingValue`**:
   - Manter rejeição HTTP 400 no backend (`CreateAgent` / `UpdateAgent`) para qualquer token de `thinking_level` fora do enum do provider.
   - Para `antigravity`, aceitar apenas `""` (qualquer valor não-vazio retorna 400 com mensagem: `"antigravity uses model-embedded reasoning tier; explicit thinking_level is not allowed"`).
2. **Guard `validateThinking` no Daemon**:
   - `daemon.go:3786` valida a tupla `(provider, model, thinking_level)` antes do dispatch. Se incompatível, emite warning e faz fallback para o default da CLI.

### 4.2 Suíte de Testes
1. **Unit Tests (`server/pkg/agent/thinking_test.go`)**:
   - Testar `IsKnownThinkingValue` com tokens válidos e inválidos para cada provider.
   - Testar `ValidateThinkingLevel` resolvendo modelos default.
2. **Integration Tests (`server/internal/handler/agent_test.go`)**:
   - Testar `POST /api/agents` com `thinking_level` válido vs inválido.
   - Testar alteração de runtime mantendo `thinking_level` antigo inválido -> verificar 400 Bad Request.
3. **Frontend Tests (`packages/views/agents/components/create-thinking-field.test.tsx`)**:
   - Validar que para modelos AGY (`levels.length === 0`), o selector de thinking level não é renderizado.

