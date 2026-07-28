# GTL-25 - design de preco unico versionado por provider/model/tier

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T11:58Z
Modo: READ-ONLY / DESIGN. Nao editei codigo, nao toquei no DB, nao compilei, nao rodei migration,
build, deploy ou restart. Gravei somente este arquivo e o meu check-out.
Base: GTL-03 (`.deploy-control/p0/evidence/gtl-cost-implementation-audit.md`).

Nenhum valor monetario e proposto neste documento. Preco e decisao do owner; aqui se define **onde**
o preco mora, **como** e versionado e **como** um miss aparece.

## 0. CORRECAO DE UMA AFIRMACAO MINHA NO GTL-03

No GTL-03 eu escrevi que "hoje um miss e invisivel". **Isso esta errado e eu corrijo.** O miss ja e
observavel. `internal/metrics/business.go:259-268` (literal):
```go
	price, priced := PriceForModelAlias(modelAlias)
	if !priced {
		provider := NormalizeRuntimeProvider(rawProvider)
		alias := NormalizeModelAlias(modelAlias)
		m.recordUnpricedTokens(provider, alias, "input", inputTokens)
		m.recordUnpricedTokens(provider, alias, "output", outputTokens)
		m.recordUnpricedTokens(provider, alias, "cache_read", cacheReadTokens)
		m.recordUnpricedTokens(provider, alias, "cache_write", cacheWriteTokens)
		m.llmRequests.WithLabelValues(provider, "unknown", runtimeMode).Inc()
		return
	}
```
e `business.go:289-294` (literal):
```go
func (m *BusinessMetrics) recordUnpricedTokens(provider, modelAlias, tokenType string, tokens int64) {
	if tokens <= 0 {
		return
	}
	m.llmUnpricedTokens.WithLabelValues(provider, modelAlias, NormalizeTokenType(tokenType)).Add(float64(tokens))
}
```
O contador e `multica_llm_unpriced_tokens_total` (`business.go:124-129`). Portanto o requisito
"miss observavel" **ja esta atendido para o eixo model**. O que falta e o eixo **tier**: hoje um
`claude-opus-4-6-thinking` nao cai no miss, ele casa a regra base e e cobrado como Opus normal.
Ou seja o miss de tier e **silencioso por sucesso falso**, que e pior que o miss aberto. Esse e o
requisito real desta entrega.

## 1. ESTADO ATUAL, ANCORADO EM LINHA

Fonte de preco, `internal/metrics/pricing.go`:
- `:8-15` `type ModelPrice struct { Provider, Model string; InputPerM, CacheReadPerM, CacheWritePerM, OutputPerM float64 }`
- `:17-39` mapa `modelPrices`, 22 entradas, chave `provider:model`
- `:41-66` `modelAliasRules`, 21 regex
- `:68-77` `PriceForModelAlias(model string) (ModelPrice, bool)`
- `:79-84` `tokenCostUSD(tokens int64, pricePerM float64) float64`

Unico consumidor: `business.go:253` `RecordLLMUsage(source, runtimeMode, rawProvider, modelAlias string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64)`, que chama
`recordPricedTokens` em `:271-274` e o miss em `:263-266`.

Destino: contadores Prometheus `multica_llm_cost_usd_total` (`:118-123`),
`multica_llm_unpriced_tokens_total` (`:124-129`), `multica_llm_tokens_total`.

`ls migrations/ | grep -iE 'price|cost|rate'` nao retorna nada: **nao existe preco em banco**, nem
coluna `cost_usd` em nenhuma tabela.

Tres defeitos herdados, todos confirmados no GTL-03:
1. sem `effective_from` nem `version` — recomputar historico apos reajuste muda o numero sem trilha;
2. sem eixo de tier — `claude-opus-4[-.]6` casa `claude-opus-4-6-thinking`; `gemini-3[.]1-pro` casa
   `-high` e `-low`;
3. sem entrada para os modelos AGY realmente disponiveis — `gemini-3.6-flash` nao existe no mapa
   (existem `gemini-3-flash` e `gemini-2.5-flash`).

## 2. DECISAO: CODIGO VERSIONADO, NAO TABELA DB

Recomendo **codigo versionado** (`internal/metrics/pricing.go` estendido) como fonte unica, e
**rejeito** tabela no Postgres nesta fase. Tradeoffs, honestos nos dois sentidos:

| criterio | codigo versionado (recomendado) | tabela DB |
|---|---|---|
| trilha de auditoria | git: quem, quando, diff, review obrigatorio | requer tabela de auditoria propria + quem escreveu |
| imutabilidade do historico | natural: uma entrada antiga nunca e reescrita, so se acrescenta linha nova com `effective_from` | precisa de `UPDATE` proibido por trigger, senao alguem edita preco passado |
| onde o consumidor roda | `pkg/agent` e `internal/metrics` rodam **no binario do daemon**, que nao tem acesso ao Postgres do backend | exigiria plumbing novo daemon -> DB, ou duplicar preco nos dois lados |
| latencia / disponibilidade | zero: mapa em memoria, sem I/O, sem falha parcial | consulta por evento ou cache com invalidacao; DB fora do ar degrada metrica |
| mudar preco sem deploy | **nao da** — precisa rebuild e restart | da, e essa e a unica vantagem real |
| risco de duas verdades | baixo: uma fonte | alto: se o codigo mantiver fallback, os dois divergem no primeiro reajuste |
| teste | tabela de testes puro, sem fixture de banco | precisa seed e migration em teste |

O argumento decisivo nao e conveniencia, e topologia: o consumidor de preco vive no **daemon**
(`internal/metrics` e compilado no binario `multica-auth-credential-home-v1`), e o Postgres e do
**backend no ORQ1**. Colocar preco em DB obrigaria o daemon a falar com o banco do backend ou a
manter uma copia — e copia de tabela de preco e exatamente a segunda fonte de verdade que o dispatch
manda evitar.

Quando eu mudaria de opiniao, e registro para o owner decidir: se o requisito passar a ser
**preco editavel por operador sem deploy**, ou **preco por tenant/contrato**, a tabela vira
obrigatoria. Nesse caso a arquitetura correta e o banco ser a fonte e o codigo **nao** ter fallback
de valor, apenas falhar para o caminho de miss observavel. Nao decido isso: e produto.

## 3. MODELO DE DADOS EM CODIGO (patch map)

### 3.1 Tipo estendido, aditivo
```go
// ANTES  internal/metrics/pricing.go:8-15
type ModelPrice struct {
	Provider       string
	Model          string
	InputPerM      float64
	CacheReadPerM  float64
	CacheWritePerM float64
	OutputPerM     float64
}

// DEPOIS — campos novos, nenhum removido, nenhum renomeado.
type ModelPrice struct {
	Provider       string
	Model          string
	Tier           string    // "" = tier base do modelo; "high", "low", "thinking", "xhigh", ...
	InputPerM      float64
	CacheReadPerM  float64
	CacheWritePerM float64
	OutputPerM     float64
	EffectiveFrom  time.Time // UTC, inclusivo
	Version        string    // identificador imutavel da tabela, ex. "pricing-2026-07"
}
```
Manter os quatro campos de valor com o mesmo nome preserva `business.go:271-274` sem alteracao.

### 3.2 Historico imutavel: lista ordenada, nao mapa por chave
O mapa atual (`chave -> 1 preco`) nao consegue representar historico. Substituir por **slice
append-only por chave**:
```go
// chave canonica: provider:model:tier  (tier vazio = ":")
// invariante: para cada chave, entradas ordenadas por EffectiveFrom crescente.
// REGRA DE IMUTABILIDADE: uma entrada ja publicada NUNCA e editada nem removida.
// Reajuste = APPEND de nova entrada com EffectiveFrom posterior.
var modelPriceHistory = map[string][]ModelPrice{
	// "anthropic:claude-opus-4.6:":         { {..., EffectiveFrom: t0, Version: "pricing-2026-07"} },
	// "anthropic:claude-opus-4.6:thinking": { {..., EffectiveFrom: t0, Version: "pricing-2026-07"} },
	// valores monetarios NAO definidos aqui: entram por decisao do owner.
}
```
Por que slice e nao mapa aninhado por data: a resolucao e "ultima entrada com
`EffectiveFrom <= at`", uma busca binaria trivial em slice ordenado, e a ordenacao torna a
invariante verificavel por teste (item 7.4). Mapa por data esconderia buracos e sobreposicoes.

### 3.3 Resolucao: nova funcao, antiga delega
```go
// NOVA: resolucao explicita por tier e por instante.
// Devolve ok=false quando nao ha entrada para a chave OU quando nenhuma entrada
// tem EffectiveFrom <= at. Nunca extrapola para tras.
func PriceFor(provider, model, tier string, at time.Time) (ModelPrice, bool) {
	// 1. tenta provider:model:tier
	// 2. NAO cai para provider:model:"" silenciosamente — ver secao 4
}

// LEGADO: assinatura preservada, comportamento preservado.
func PriceForModelAlias(model string) (ModelPrice, bool) {
	// delega com tier="" e at=time.Now().UTC()
}
```
`PriceForModelAlias` continuar existindo e o que garante que `business.go:259` nao muda e que a
regressao de metrica e testavel (item 7.7).

### 3.4 O tier NUNCA vem do nome do modelo
Regra explicita, e e o ponto que o GTL-03 barrou: o `tier` chega como **parametro**, originado da
coluna `thinking_level` de `task_usage` (proposta do GTL-03, Opcao B), preenchida pelo daemon a
partir de `task.Agent.ThinkingLevel`. Nao ha `CASE WHEN model LIKE '%-high'`, nao ha regex de
sufixo para tier. Se `thinking_level` vier vazio, o tier e `""` e resolve o preco base — nunca
adivinha.

Consequencia que precisa ser aceita conscientemente: as 21 `modelAliasRules` atuais **continuam**
casando por regex no nome do modelo para achar o **modelo**. Isso e diferente de inferir tier. O
alias resolve identidade de modelo (`gemini-3.1-pro-high` -> modelo `gemini-3.1-pro`); o tier vem
do dado. Registro a implicacao: as regras precisam ser auditadas para **nao** engolir o sufixo de
tier como parte do nome do modelo, senao dois tiers colapsam na mesma chave antes de o tier ser
consultado.

## 4. MISS OBSERVAVEL, COM TRES CATEGORIAS DISTINTAS

Hoje existe uma categoria (`unpriced_tokens_total`). Proponho **tres**, porque tratam-se de defeitos
diferentes com acoes diferentes:

| categoria | quando | acao esperada |
|---|---|---|
| `model_unknown` | nao existe nenhuma entrada para `provider:model` | cadastrar modelo (ex.: `gemini-3.6-flash` hoje) |
| `tier_unknown` | existe `provider:model:""` mas nao `provider:model:<tier>` | cadastrar tier, ou declarar que o tier nao muda preco |
| `no_effective_price` | existe a chave, mas nenhuma entrada com `EffectiveFrom <= at` | corrigir `EffectiveFrom` ou reprocessar |

Implementacao proposta: um label **fechado** `miss_reason` no contador existente, com esses tres
valores mais nada. Cardinalidade controlada: 3 valores, nao um conjunto aberto.

**Decisao de politica que o design precisa fixar, e eu recomendo explicitamente:** em
`tier_unknown`, **nao** cair para o preco base. Cair para a base e o comportamento de hoje e e a
razao pela qual Opus thinking e cobrado como Opus normal — um erro que se apresenta como sucesso.
Recomendo contar como miss e nao cobrar, deixando o token em `unpriced` com
`miss_reason="tier_unknown"`. Assim o buraco aparece em vez de virar numero errado.
Alternativa aceitavel, se o owner preferir cobertura a exatidao: cair para a base **e** incrementar
o contador de miss no mesmo evento, para que o fallback nunca seja silencioso. Nao escolho por ele;
recomendo a primeira.

## 5. ROLLUPS 073 / 084 / 101 / 102

Chaves reais, medidas:
- `073_task_usage_daily_rollup.up.sql:11-23` — `task_usage_daily`, PK
  `(bucket_date, workspace_id, runtime_id, provider, model)`; `:38` `task_usage_rollup_state`;
  `:76` `rollup_task_usage_daily_window(...)`.
- `084_task_usage_dashboard_rollup.up.sql:37-52` — `task_usage_dashboard_daily`, PK
  `(bucket_date, workspace_id, agent_id, project_id, model)`; `:68` state; `:86-95` `dirty`.
  O comentario `:14-15` justifica a cardinalidade `runtime × agent × project × model`.
- `101_task_usage_hourly_schema.up.sql:37-54` — `task_usage_hourly`, PK
  `(bucket_hour, workspace_id, runtime_id, agent_id, project_id, provider, model)`;
  `:85` state; `:114-125` `dirty` com a **mesma** chave.
- `102_task_usage_hourly_pipeline.up.sql` — `:31` bucket, `:50`/`:142`/`:176`/`:233` triggers de
  dirty, `:274` `rollup_task_usage_hourly_window(...)` com `GROUP BY 1,2,3,4,5,6,7` em `:348`,
  `:418` prune.

**Fato central: nenhuma das quatro tabelas de rollup tem coluna de custo.** Elas agregam apenas
`input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_write_tokens`
(`101:45-48`, e equivalente em 073). Custo vive **so** em Prometheus.

Consequencia forte e boa para esta entrega: **preco versionado por tier nao exige mudar nenhum dos
quatro rollups**, desde que a decisao seja "custo continua sendo derivado na leitura, nunca
persistido no rollup". Recomendo exatamente isso. Motivo: se o custo fosse materializado no rollup,
um reajuste de preco tornaria o agregado historico **inconsistente com a tabela de preco vigente**,
e reprocessar exigiria refazer buckets antigos — precisamente o que `EffectiveFrom` existe para
evitar.

Se, ainda assim, o owner quiser custo agregado por tier no banco, o custo minimo e alto e eu
registro para que ninguem descubra depois:
1. `thinking_level` entra na PK das quatro tabelas (`073:23`, `084:52`, `101:54`, `101:125`) —
   mudanca de chave primaria, nao adicao de coluna;
2. `rollup_task_usage_hourly_window` (`102:274`) e o `GROUP BY 1..7` (`102:348`) passam a `1..8`;
3. `rollup_task_usage_daily_window` (`073:76`) idem;
4. as quatro tabelas `dirty` e os quatro triggers (`102:50,142,176,233`) precisam propagar a nova
   dimensao, senao buckets nao sao marcados como sujos corretamente;
5. cardinalidade cresce por um fator igual ao numero de tiers por modelo — o comentario `084:14-15`
   mostra que cardinalidade de rollup ja foi uma preocupacao de projeto ali.
Nao recomendo. Fica registrado como caminho, nao como plano.

Ponto que confirmo do desenho existente: `pkg/db/generated/task_usage.sql.go:418-420` documenta que
o upsert bumpa `updated_at` para o worker marcar o bucket como sujo; adicionar coluna a `task_usage`
**sem** mexer no upsert nao quebra essa deteccao.

## 6. MIGRATIONS

**Nenhuma migration e necessaria para o preco.** Preco fica em codigo (secao 2), e os rollups nao
recebem custo (secao 5).

A unica migration que este design pressupoe e a que o GTL-03 ja propos e que **nao e minha**:
`thinking_level TEXT NOT NULL DEFAULT ''` em `task_usage`, porque e a origem do parametro `tier`.
Sem ela, `PriceFor` sempre recebe `tier=""` e o design fica correto porem inerte. Registro a
dependencia; nao redesenho a migration de outro pacote.

## 7. ROLLBACK NAO DESTRUTIVO

Tres niveis, do mais barato ao mais caro, todos sem perda de dado:

1. **Reverter comportamento sem reverter codigo.** `PriceFor` fica atras de um seletor de politica
   com default conservador: enquanto o flag estiver desligado, `PriceForModelAlias` mantem
   exatamente a resolucao atual (regex + mapa base) e nenhum tier e consultado. Ligar o novo
   caminho e uma mudanca de configuracao, nao de codigo. Reverter = desligar.
2. **Reverter uma entrada de preco errada.** Nunca editando nem removendo: **append** de nova
   entrada com `EffectiveFrom` no instante da correcao e `Version` nova. O historico anterior
   continua valido para o periodo em que valeu. Isso e o que torna o rollback nao destrutivo por
   construcao — o oposto de um `UPDATE` em tabela de preco.
3. **Reverter o codigo.** Como nada foi persistido no banco (sem migration de preco, sem custo em
   rollup), reverter o binario e suficiente e nao deixa dado orfao. Aqui vale a ressalva registrada
   pelo Codex56#B e que respeito: reverter binario do daemon **nao** pode significar reaplicar a
   branch antiga, que reintroduziria o AGY task-incapaz. O rollback e "voltar ao commit anterior a
   este patch de pricing", nao "voltar a um backup pre token-only".

O que **nao** e reversivel e por isso nao proponho: materializar `cost_usd` em rollup. Uma vez
gravado custo agregado com preco antigo, desfazer exige reprocessar buckets, o que e destrutivo de
fato.

## 8. TESTES

Sobre preco e resolucao:
1. `TestPriceForResolvesTierDistinctFromBase` — a chave `provider:model:thinking` devolve entrada
   diferente da `provider:model:""`. E o teste que falha hoje e que da sentido a entrega.
2. `TestPriceForEffectiveFromPicksLatestNotFuture` — com duas entradas para a mesma chave, `at`
   entre elas resolve a anterior; `at` antes da primeira devolve `ok=false` com
   `no_effective_price`, nunca extrapola para tras.
3. `TestPriceForUnknownTierDoesNotFallBackToBase` — trava a decisao da secao 4. Se alguem trocar a
   politica para fallback silencioso, este teste quebra.
4. `TestModelPriceHistoryInvariantsSortedAndNonOverlapping` — para toda chave, `EffectiveFrom`
   estritamente crescente e sem duplicata. Guarda a invariante do append-only.
5. `TestModelPriceHistoryVersionNonEmpty` — toda entrada tem `Version` preenchido, senao a trilha
   de auditoria tem buraco.
6. `TestModelPriceHistoryEffectiveFromIsUTC` — evita a classe de bug em que fuso local muda a
   fronteira do periodo cobrado.
7. `TestPriceForModelAliasLegacyUnchanged` — **regressao**: para os 22 modelos hoje cadastrados,
   `PriceForModelAlias` devolve exatamente os mesmos quatro valores de antes do patch. Prova que
   `business.go:259` nao muda de comportamento.
8. `TestAliasRulesDoNotSwallowTierSuffix` — para entradas como `gemini-3.1-pro-high` e
   `claude-opus-4-6-thinking`, a regra de alias resolve o **modelo** base e nao produz duas chaves
   distintas de modelo; o tier fica fora do nome. Fecha a implicacao levantada em 3.4.

Sobre miss observavel:
9. `TestMissReasonModelUnknown` — modelo ausente (usar `gemini-3.6-flash`, que hoje realmente nao
   existe no mapa) incrementa o miss com `miss_reason="model_unknown"`.
10. `TestMissReasonTierUnknown` — modelo presente e tier ausente incrementa com
    `miss_reason="tier_unknown"` e **nao** incrementa custo.
11. `TestMissReasonLabelSetIsClosed` — o conjunto de valores de `miss_reason` e exatamente
    {`model_unknown`,`tier_unknown`,`no_effective_price`}. Guarda de cardinalidade.
12. `TestUnpricedTokensStillCountedOnMiss` — o token continua contado em
    `multica_llm_unpriced_tokens_total`; um miss nunca faz o token desaparecer.

Sobre rollups (testes de **nao regressao**, porque a recomendacao e nao mexer):
13. `TestRollupSchemaHasNoCostColumn` — assercao de schema sobre as quatro tabelas: nenhuma coluna
    de custo. Documenta a decisao em teste, para que materializar custo exija quebrar um teste
    deliberadamente.

## 9. ORDEM DE APLICACAO RECOMENDADA

1. `thinking_level` em `task_usage` (dependencia do GTL-03, nao minha) e o wiring do daemon.
2. Tipo `ModelPrice` estendido + `modelPriceHistory` + `PriceFor`, com `PriceForModelAlias`
   delegando e o flag de politica desligado. Neste ponto o comportamento observavel e **identico**
   ao de hoje e o teste 7 prova isso.
3. Cadastro de valores pelo owner, incluindo os modelos AGY ausentes. **So aqui entra dinheiro**, e
   nao e decisao de agente.
4. `miss_reason` com os tres valores fechados.
5. Ligar o flag. A partir daqui Opus thinking deixa de ser cobrado como Opus normal.
6. Nada nos rollups.

## 10. NAO-AFIRMACOES
- READ-ONLY: nao editei codigo, nao toquei no DB, nao criei migration, nao compilei, nao rodei
  build, teste, migration, deploy, restart ou rerun. Gravei apenas este arquivo e o meu check-out.
- **Nao defini nenhum valor monetario.** Todas as entradas de exemplo estao comentadas e vazias de
  valor. Preco e decisao do owner.
- Corrigi uma afirmacao minha do GTL-03: o miss por modelo **ja** e observavel via
  `multica_llm_unpriced_tokens_total` (`business.go:263-266`, `:289-294`). O que nao existe e o
  eixo tier, e o problema real e o falso sucesso, nao a invisibilidade.
- Nao verifiquei se `NormalizeModelAlias` e `NormalizeRuntimeProvider` (usados no caminho de miss)
  preservam sufixo de tier; isso importa para o teste 8 e precisa ser confirmado por quem escrever.
- Nao medi a cardinalidade atual das series de `multica_llm_cost_usd_total` em producao; a analise
  de cardinalidade aqui e sobre o conjunto de labels declarado, nao sobre contagem observada.
- Nao decidi entre "nao cobrar em tier_unknown" e "cair para base contando o miss": recomendei a
  primeira e registrei a segunda como aceitavel. A escolha e do owner.
- Nao decidi codigo vs DB de forma irrevogavel: recomendei codigo e declarei o gatilho que
  inverteria a recomendacao (preco editavel sem deploy, ou preco por tenant).
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
- Os arquivos de codigo modificados no repo nao sao meus: pertencem ao escritor unico.
