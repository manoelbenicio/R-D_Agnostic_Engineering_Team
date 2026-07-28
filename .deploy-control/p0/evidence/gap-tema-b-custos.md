# Relatório de Lacunas de Governança — Tema B: Contabilização de Token e Custo

## 1. Contexto e Diagnóstico Factual
O backend Multica possui pipeline de contabilização de tokens respaldada pelas tabelas \`task_usage\`, \`task_usage_hourly\` e pela API \`ReportTaskUsage\` (\`daemon.go\` L2064). Contudo, a arquitetura atual de credenciais e o modelo de persistência de métricas possuem lacunas críticas que impedem o atendimento dos requisitos de custo do Owner.

## 2. Lacunas Identificadas

### Lacuna 1: Impossibilidade de Atribuição de Custo por Conta do Gateway
- **O que falta**: A tabela \`task_usage\` e a API de reporte registram consumo apenas por \`(task_id, provider, model)\`. Não existe campo para \`account_id\` ou \`omniroute_account_id\`.
- **Evidência**: \`server/pkg/db/queries/task_usage.sql\` L6: \`INSERT INTO task_usage (task_id, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, updated_at)\`.
- **Situação com Credencial no HOME Global**: Como as credenciais estão no HOME global do ORQ1, a execução consome a conta física do servidor de forma opaca. Todas as tasks registram o mesmo provider genérico, sem qualquer rastreamento de qual conta/slot financiou a inferência.
- **Impacto**: **BLOQUEIA** a atribuição e isolamento de custos por conta exigidos pelo Owner.

### Lacuna 2: Falta de Tarifação e Métricas por Tier de Reasoning
- **O que falta**: O schema do banco armazena apenas contagens puras de tokens e o modelo string. Não há suporte a precificação ponderada pelo tier de esforço/reasoning (\`-xhigh\`, \`-high\`, \`-medium\`, \`-low\`).
- **Evidência**: \`server/internal/handler/daemon.go\` L2101-2113: \`UpsertTaskUsage\` armazena apenas \`InputTokens\` e \`OutputTokens\` brutos sem converter para custo monetário ou associar ao multiplicador do tier.
- **Impacto**: **DEGRADA** o controle orçamentário e a visibilidade financeira dos tiers de esforço.

### Lacuna 3: Extração de Tokens Dependente de Inspecção Local de Arquivos da CLI
- **O que falta**: O daemon depende da leitura de arquivos de log locais gerados pela CLI (\`codex.go\` L1882) para capturar o consumo. CLIs como \`agy\`, \`kiro\` e \`opencode\` não possuem esse scanner padronizado, resultando em 0 tokens registrados. O recibo de uso deveria vir diretamente da resposta do Gateway OmniRoute.
- **Evidência**: \`server/pkg/agent/codex.go\` L1882 (\`codexInt64(usageMap, "input_tokens"...)\`).
- **Impacto**: **DEGRADA** a contabilização para 3 dos 6 runtimes ativos.
