# Auditoria READ-ONLY GTL-13 / ORQ-14 — Contabilidade de Tokens por CLI

## 1. Auditoria dos 3 CLIs Obrigatórios

### A. AGY (\`antigravity\`)
- **Estado Atual**: \`antigravity.go\` L162-165 retorna \`Usage: map[string]TokenUsage{}\` (mapa vazio por falta de captura no stream).
- **Formato da Wire**: Formatos Gemini / Antigravity expõem campos em **camelCase** (\`inputTokens\`, \`outputTokens\`, \`cachedContentTokenCount\`) e em **snake_case** (\`input_tokens\`, \`output_tokens\`, \`cache_read_tokens\`).
- **Decisão**: Implementar parser dual (camelCase + snake_case) no stream e no log runner de \`antigravity.go\`.

### B. CODEX (\`codex\`)
- **Estado Atual**: \`codex.go\` L1861-1887 (\`extractUsageFromMap\`) e L1923-2075 (\`scanCodexSessionUsage\`).
- **Mecanismo**: 
  1. Captura eventos JSON-RPC \`turn/completed\` em tempo de execução.
  2. Scanner secundário de fallback em arquivos JSONL de sessão (\`~/.codex/sessions/*.jsonl\`) analisando eventos \`token_count\` (\`TotalTokenUsage\` / \`LastTokenUsage\`).
- **Decisão**: Estrutura sólida e confiável. Manter o scanner como fallback secundário.

### C. KIRO (\`kiro-cli\`)
- **Estado Atual**: \`kiro.go\` L337-340 acumula \`pr.usage\` de eventos ACP (\`session/prompt\`). Se o daemon ACP do Kiro omitir a struct \`usage\`, o retorno fica zerado.
- **Fonte Alternativa Confiável (Sem Inventar Tokens)**:
  * **Gateway Receipts (OmniRoute)**: Como toda a inferência trafega pelo gateway OmniRoute (\`http://100.118.244.61:20128/v1/chat/completions\`), o gateway devolve a contagem exata no payload (\`usage.prompt_tokens\`, \`usage.completion_tokens\`) e nos cabeçalhos de resposta (\`x-omniroute-tokens-input\`, \`x-omniroute-tokens-output\`).
  * **Decisão**: Usar o **OmniRoute Gateway Receipt** como fonte autoritativa primária/fallback para Kiro e para qualquer CLI sem telemetria nativa no stream.

## 2. Decisão Arquitetural Implementável
1. **Unificação via Gateway Receipt Interceptor**: Criar um middleware no daemon/gateway que intercepta a contagem de tokens retornada pelo OmniRoute e a vincula diretamente ao \`task_id\`.
2. **Normalizador Dual-Format**: Função utilitária \`ParseTokenUsageMap(data map[string]any)\` suportando simultaneamente camelCase (\`inputTokens\`) e snake_case (\`input_tokens\`).

## 3. Desenho da Suíte de Testes (Sem Mudar Produção)
- **Unitários**: Testes com \`httptest.Server\` simulando respostas do OmniRoute com cabeçalhos e payloads contendo tokens.
- **Testes de Parsers**: Suíte cobrindo camelCase, snake_case, e arquivos JSONL legados do Codex.
