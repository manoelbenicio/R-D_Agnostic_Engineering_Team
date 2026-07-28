# Auditoria READ-ONLY GTL-10 / ORQ-15 — Grafo de Dependências e Critério de Integração

## 1. Auditoria do Critério de Aceite da ORQ-15
- **Título**: Relatório de integração por conta / slots (AGY).
- **Diagnóstico da Task Histórica**: A task legada da ORQ-15 falhou pre-start (\`0/0\` mensagens de ferramenta, erro de symlink no \`cli.log\` em \`slot-146\`).
- **Resolução do Pre-Start**: O Gate 2 e o allowlist live (\`MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146,150\`) corrigiram essa classe de falha, comprovada pelo discovery de 11 modelos OK e pelo smoke ORQ-27 completado.
- **O que Falta Exatamente**: Para fechar o aceite da ORQ-15, não é necessário "reexecutar a task antiga morta". Exige-se:
  1. Abertura do mapa de contas pseudônimas no Postgres (\`accounts\`, \`approved_accounts\`, \`assignments\`) via **ORQ-21**.
  2. Persistência do \`account_id\` pseudônimo na \`task_usage\` via **ORQ-12**.
  3. Relatório consolidado demonstrando atribuição financeira individualizada para os 4 slots ativos (\`141, 145, 146, 150\`).

## 2. DAG Exato de Dependências (ORQ-12, 13, 14, 15, 20, 21, 23)

\`\`\`
          [ ORQ-21 ] (Ponte Postgres: accounts / approved / assignments)
              │
              ▼
          [ ORQ-12 ] (Atribuição account_id na task_usage)
              │
      ┌───────┴───────────────┐
      ▼                       ▼
[ ORQ-13 ] (Tiers & Cost)   [ ORQ-14 ] (Suíte de Testes de Integração)
      │                       │
      └───────────┬───────────┘
                  ▼
          [ ORQ-15 ] (Relatório de Integração por Conta/Slot AGY)
                  │
                  ▼
          [ ORQ-20 ] (Dashboard de Observabilidade & Métricas)
                  │
                  ▼
          [ ORQ-23 ] (Agregador da Fase 3 & Gate de Rollback 1-Comando)
\`\`\`

## 3. Matriz de Atendimento (Evidência Existente vs Necessidade de Código)

| Componente da ORQ-15 | Atendido por Evidência Atual? | Evidência / Requisito Factual |
|---|---|---|
| **Elegibilidade dos 4 Slots AGY** | **SIM (Fechado por Evidência)** | Allowlist live \`141,145,146,150\` + Discovery 11/11 modelos (\`sol-tema-7.md\`). |
| **Correção de Symlink no Boot** | **SIM (Fechado por Evidência)** | Validação do Gate 2 e execução limpa da smoke ORQ-27 (\`RCA-HANDOVER-20260727.md\`). |
| **Estrutura de Atribuição no DB** | **NÃO (Exige Código / ORQ-21)** | Tabelas \`accounts\`, \`approved_accounts\` e \`assignments\` com 0 linhas no Postgres live. |
| **Rastreabilidade em task_usage** | **NÃO (Exige Código / ORQ-12)** | \`server/migrations/032_task_usage.up.sql:1-11\` carece da coluna \`account_id\`. |
| **Relatório Consolidado de Custo** | **NÃO (Exige Código / ORQ-13 & 15)** | Cálculo por \`thinking_level\` depende da mescla das migrations da ORQ-13. |

## 4. Veredito Final
- A **ORQ-15 é 100% dependente da cadeia financeira** (\`ORQ-21 → ORQ-12 → ORQ-13/14 → ORQ-15\`).
- A task legada deve ser mantida preservada no estado \`failed\` como histórico. O aceite da ORQ-15 será emitido assim que a cadeia ORQ-21+12+13 gerar o relatório por conta com os 4 slots aprovados.
