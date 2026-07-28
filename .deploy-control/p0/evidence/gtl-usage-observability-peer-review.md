# Peer Review READ-ONLY GTL-26 — Observabilidade de Schema de Usage (GTL-17)

## 1. Identificação do Reviewer e Escopo
- **Reviewer**: Antigravity w8:p2
- **Documento Revisado**: `.deploy-control/p0/evidence/gtl-usage-schema-observability.md`
- **Modo**: READ-ONLY. Zero alterações de código, banco de dados ou ambiente de produção.

## 2. Validação Factual contra o Código-Fonte do Server/Daemon

### A. Validação dos Dois Caminhos Hermes
- **Caminho 1 (\`usage_update\`)**: Confirmado em \`hermes.go\` L797-798 e L1190-1201. Trata-se de snapshot cumulativo aplicado por máximo (\`hermes.go\` L1205-1213).
- **Caminho 2 (\`session/prompt\` / \`turn_end\`)**: Confirmado em \`hermes.go\` L725-727, L732-743 e L800-801. Trata-se de acumulador por soma (\`+=\`, \`hermes.go\` L380-385).

### B. Garantia de Redaction por Construção
- Validada a função \`usageSchemaKeys\` / \`jsonKind\` (\`gtl-usage-schema-observability.md\` L174-208):
  * Inspeção restrita ao primeiro byte do tipo JSON (\`s[0]\`).
  * NENHUM valor numérico, trecho de prompt, e-mail ou credencial é formatado ou concatenado na string de log.
  * Mitigação de nomes de chaves com hash/hex (\`[0-9a-f]{8,}\`) substituídos por \`<redacted-key-name>\`.

### C. Controle de Cardinalidade e Feature Flag
- Uso exclusivo de logs estruturados (\`slog\`), rejeitando Prometheus labels para evitar explosão de cardinalidade em vetores dinâmicos.
- Feature flag \`MULTICA_USAGE_SCHEMA_PROBE\` (default OFF, escopo de \`var\` de pacote).
- Limite estrito de no máximo 4 linhas de log por task execution (L255-265).

### D. Correção de Premissa Stale (AGY \`cli.log\` Symlink)
- **RETIFICAÇÃO NECESSÁRIA**: A observação no final da Seção 7 (L297-302) citando a falha de symlink em \`cli.log\` está **OBSOLETA (STALE)**.
- **Evidência Factual**: O Gate 2 e a tarefa de smoke **ORQ-27 já foram concluídos com SUCESSO**. O allowlist live (\`MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146,150\`) e a descoberta de 11 modelos AGY no ORQ2 comprovaram que a preparação dos 4 slots AGY conclui sem erros de symlink.

### E. Avaliação de Rebuild e Restart do Daemon
- Confirmado que \`pkg/agent/hermes.go\` e \`codex.go\` estão compilados no binário do daemon no ORQ2 (\`multica-daemon-orq2-credential.service\`).
- A inclusão da sonda exige rebuild e restart do daemon (com a fila de tasks vazia). A alteração do valor da flag (\`MULTICA_USAGE_SCHEMA_PROBE=1\`) é feita via unit systemd.

## 3. Desenho da Suíte de Testes Unitários
- Suíte de 10 testes especificada em \`gtl-usage-schema-observability.md\` Seção 8 (L303-327):
  * \`TestUsageSchemaKeysNeverLeaksValues\`: Teste negativo provando que valores não vazam.
  * \`TestUsageSchemaKeysRedactsHexLikeKeyNames\`: Sanitização de nomes de chave sensíveis.
  * \`TestUsageSchemaKeysDepthLimitedToTwo\`: Restrição de profundidade JSON em 2 níveis.
  * \`TestUsageSchemaProbeEmitsOncePerPath\`: Controle de concorrência e limite por caminho.

## 4. Veredito Final
- **STATUS: PASS (APROVADO COM RETIFICAÇÃO DE PREMISSA STALE)**
- **Linhas de Evidência Citadas**: \`gtl-usage-schema-observability.md\` L36-105 (caminhos Hermes), L174-227 (redaction), L228-266 (flag/cardinalidade) e L328-350 (impacto de restart).
