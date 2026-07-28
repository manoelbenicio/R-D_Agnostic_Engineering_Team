# Peer Review de Arquitetura: Preço Versionado por Tier (`gtl-versioned-tier-pricing-design.md`) — GTL-36

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:42Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-versioned-tier-pricing-design.md` (autor: Opus48#B w6:p2)  
**modo**: READ-ONLY / PEER REVIEW — NENHUM código, banco de dados ou estado alterados  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: PASS** ✅

**Resumo da Avaliação**:
O documento de design `gtl-versioned-tier-pricing-design.md` é de **excelente qualidade técnica**, perfeitamente ancorado nas linhas reais do código-fonte Go (`internal/metrics/pricing.go` e `business.go`) e no schema das migrations do PostgreSQL (`073`, `084`, `101`, `102`). O design resolve de forma elegante o falso sucesso silencioso no cálculo de custos por tier de reasoning, implementa versionamento temporal imutável (`effective_from`), rejeita fallbacks silenciosos e não exige migrations destrutivas no banco de dados.

---

## 2. Matriz de Validação de Critérios Requisitados

| Critério de Auditoria | Avaliação no Design | Evidência no Código / Documento | Status |
|---|---|---|---|
| **1. Validação Código-vs-DB** | Ancoragem exata nas linhas do código Go e migrations. Confirma que preços vivem em memória/Prometheus (`pricing.go:17-39`) e que o banco não tem coluna de custo. | `pricing.go:8-39`, `business.go:259-268`, Migrations `073`, `084`, `101`, `102`. | ✅ PASS |
| **2. Modelo Append-Only / `effective_from`** | Versionamento temporal via `ModelPrice{ Version, EffectiveFrom, Tier }`. Histórico ordenado por `EffectiveFrom` sem mutação destructiva (`UPDATE`). | Seções 2 e 3 do design. | ✅ PASS |
| **3. Resolução da Tupla `provider:model:tier`** | Chave primária estendida para considerar o tier de esforço (`thinking_level`). | Seção 2.1 e 2.2 do design. | ✅ PASS |
| **4. Nenhum Fallback Silencioso** | Se `provider:model:tier` não for encontrado, **não** cai para `tier=""`. Falha fechado retornando `priced=false`. | Seção 4 ("Por que nao fazer fallback silencioso"). | ✅ PASS |
| **5. Observabilidade do Miss (`miss_reason`)** | Registra tokens não precificados em `multica_llm_unpriced_tokens_total` com rótulo fechado: `model_unknown`, `tier_unknown`, `no_effective_price`. | Seções 0 e 4 do design. | ✅ PASS |
| **6. Normalização de Aliases** | Regexes de alias resolvem a base do modelo (`provider:model`), enquanto o sufixo de tier (ou modelo AGY com tier) é extraído para a tupla. | Seção 3.4 do design. | ✅ PASS |
| **7. Rollups sem Custo Materializado** | Os rollups DB agregam apenas tokens. Manter o custo na camada de leitura/Prometheus evita quebrar PKs de rollups e evita inconsistência em reajustes. | Seção 5 do design. | ✅ PASS |
| **8. Flag de Política e Rollback Não-Destrutivo** | Flag de ativação gradual. Rollback por append de nova versão com `EffectiveFrom=NOW()` ou desligamento do flag. Zero perda de dados. | Seção 7 do design. | ✅ PASS |
| **9. Ausência de Valores Inventados** | Nenhum preço monetário foi inventado no documento. Preço permanece responsabilidade e cadastro exclusivo do Owner. | Seção 10 ("Não-Afirmações"). | ✅ PASS |

---

## 3. Análise Detalhada dos Pontos Fortes do Design

### 3.1 Correção do Falso Sucesso no Miss de Tier
O autor identificou corretamente que a precificação atual não gera um erro visível quando um tier é omitido: ela erroneamente faz o match com o modelo base (ex: `claude-opus-4-6-thinking` é precificado como `claude-opus-4.6` básico). O design corrige isso tornando a falha explícita (`tier_unknown`) via contador Prometheus `multica_llm_unpriced_tokens_total{miss_reason="tier_unknown"}`.

### 3.2 Imutabilidade Temporal (`EffectiveFrom`)
Ao incluir `EffectiveFrom time.Time` e `Version string` na struct `ModelPrice`, reajustes futuros de preços de LLMs não corrompem nem recalculam retroativamente tarefas executadas no passado. A busca por corte temporal garante determinismo histórico.

### 3.3 Preservação do Schema de Rollups no Postgres
Ao provar que as tabelas de rollup (`task_usage_daily`, `task_usage_hourly`) não possuem coluna de custo `cost_usd`, o design evita migrations destrutivas de alteração de Chave Primária nas tabelas de rollup (`073`, `084`, `101`, `102`), eliminando o risco de instabilidade na pipeline de agregação em tempo real.

---

## 4. Plano de Testes Automatizados Aprovado (13 Cenários)

O documento especifica uma suíte completa de testes unitários Go:

1. **`TestPriceForResolvesTierDistinctFromBase`**: Garante que `provider:model:tier` retorna valor diferente de `provider:model:""`.
2. **`TestPriceForEffectiveFromPicksLatestNotFuture`**: Valida a resolução da versão temporal mais recente em uma data específica.
3. **`TestPriceForUnknownTierDoesNotFallBackToBase`**: Confirma que tier desconhecido retorna `priced=false` e não aceita o preço base.
4. **`TestModelPriceHistoryInvariantsSortedAndNonOverlapping`**: Garante que o histórico append-only é mantido estritamente ordenado por `EffectiveFrom`.
5. **`TestModelPriceHistoryVersionNonEmpty`**: Garante que toda entrada de preço possui identificador de versão de auditoria.
6. **`TestModelPriceHistoryEffectiveFromIsUTC`**: Valida que todas as comparações temporais usam fuso horário UTC.
7. **`TestPriceForModelAliasLegacyUnchanged`**: Teste de regressão para os 22 modelos existentes.
8. **`TestAliasRulesDoNotSwallowTierSuffix`**: Valida a extração correta de modelos AGY com sufixo de tier (`gemini-3.1-pro-high`, `claude-opus-4-6-thinking`).
9. **`TestMissReasonModelUnknown`**: Valida o incremento do contador de miss por modelo desconhecido.
10. **`TestMissReasonTierUnknown`**: Valida o incremento do contador de miss por tier desconhecido.
11. **`TestMissReasonLabelSetIsClosed`**: Garante que o enum de `miss_reason` é estritamente fechado.
12. **`TestUnpricedTokensStillCountedOnMiss`**: Confirma que os tokens continuam sendo contabilizados no total de tokens sem preço.
13. **`TestRollupSchemaHasNoCostColumn`**: Asserção de schema garantindo que nenhuma tabela de rollup DB contém coluna de custo materializado.

---

## 5. Conclusão e Próximos Passos

O documento `gtl-versioned-tier-pricing-design.md` está **TOTALMENTE APROVADO (PASS)**. O General-Tech-Lead pode autorizar a implementação técnica seguindo a ordem de aplicação recomendada na Seção 9 do documento original.

*Auditoria 100% READ-ONLY. Nenhuma linha de código ou banco de dados foi modificada.*
