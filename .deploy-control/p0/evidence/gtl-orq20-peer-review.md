# Peer Review READ-ONLY GTL-21 — Contrato E2E de Reasoning (ORQ-20)

## 1. Identificação do Reviewer e Escopo
- **Reviewer**: Antigravity w8:p2
- **Documento Revisado**: `.deploy-control/p0/evidence/gtl-orq20-reasoning-contract.md` (especialmente a seção `CORREÇÃO GTL-06R`)
- **Modo**: READ-ONLY. Zero alterações de código, banco de dados ou ambiente de produção.

## 2. Validação Factual contra HEAD e Catálogo Live

### A. Confirmação do Estado Implementado no HEAD
- **Schema DB (\`server/migrations/095_agent_thinking_level.up.sql\`)**: Confirmado `ALTER TABLE agent ADD COLUMN thinking_level TEXT`.
- **API Handlers (\`server/internal/handler/agent.go\` L767-770, L1159-1206)**: Confirmada validação tri-state e chamada a `agent.IsKnownThinkingValue`.
- **Enum Validation (\`server/pkg/agent/thinking.go\` L934)**: Confirmado o mapa `providerThinkingEnums`.
- **Frontend UI (\`packages/views/agents/create-thinking-field.tsx\` & \`thinking-picker.tsx\`)**: Confirmado ocultamento automático do picker quando `levels.length === 0`.
- **Daemon Dispatch (\`server/internal/daemon/daemon.go\` L3772-3801)**: Confirmada injeção de `ThinkingLevel` em `agent.ExecOptions` e flags CLI correspondentes (`claude.go:709`, `codex.go:771`, `kiro.go:63`, `cline.go:63`).

### B. Confirmação do Estado Proposto (Lacunas)
- **Lacuna de \`task_usage\` (ORQ-13)**: Retificado corretamente na seção `CORREÇÃO GTL-06R` (L14-16). A tabela `task_usage` atual **não persiste `thinking_level`**, sendo este o requisito pendente da ORQ-13.
- **Tratamento de AGY**: Validado que para `antigravity`, o tier de raciocínio é incorporado no próprio Model ID (`claude-opus-4-6-thinking`, `gemini-3.1-pro-high`), e `thinking_level` deve trafegar como `""` (L87).

### C. Validação de Custos e Catálogo Live (11/19/11)
- Todos os 10 agentes da frota (Seção 3, L98-109) foram mapeados exclusivamente contra os modelos **REAIS** medidos no ORQ1/ORQ2:
  * Agentes de alto raciocínio (Architect/Lead/Backend): `claude-opus-4-6-thinking`, `codex high`, `kiro high`.
  * Agentes de execução/QA/Docs: `gemini-3.6-flash-high/medium`, `gemini-3.5-flash-medium/low`.
- Zero modelos genéricos/hipotéticos. Zero defaults com custo não aprovado.

## 3. Veredito Final
- **STATUS: PASS (APROVADO SEM RESSALVAS)**
- **Linhas de Evidência**: `gtl-orq20-reasoning-contract.md` L10-54 (distinção implementado vs proposto), L83-91 (matriz live) e L98-109 (alocação dos 10 agentes).
