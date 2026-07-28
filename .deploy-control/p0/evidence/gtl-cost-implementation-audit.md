# GTL-03 - auditoria READ-ONLY do cost-accounting-design.md contra HEAD

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T11:34Z
Modo: READ-ONLY. Nao editei codigo, nao compilei, nao rodei build/deploy/restart, nao commitei.
Gravei somente este arquivo de evidencia e o check-out. Auditoria concluida sob o comunicado de
encerramento: nenhuma frente nova proposta, nada reexecutado.

Documento auditado: `.deploy-control/p0/evidence/cost-accounting-design.md` (Opus48#C, w8:p1).
Arvore de referencia: HEAD em `multica-auth-work/`.

## 0. PLACAR DA AUDITORIA

| # | Afirmacao do design | Veredito contra HEAD |
|---|---|---|
| 1 | `task_usage` nao tem `account_id` | **CONFIRMADO** |
| 2 | `UpsertTaskUsageParams` nao tem `AccountID` | **CONFIRMADO** |
| 3 | `thinking_level` existe em `agent` e nao em `task_usage` | **CONFIRMADO** |
| 4 | Extrair tier por sufixo do model ID (Opcao A) | **REJEITADO** - inferencia fragil sem regra explicita |
| 5 | AGY registra zero por snake_case no `usage_update` | **NAO COMPROVADO** - hipotese condicional do proprio autor |
| 6 | Kiro nao emite `usage_update`, gap de vendor | **PLAUSIVEL, NAO COMPROVADO** - sem captura de protocolo |
| 7 | "Preco versionado por tier" | **AUSENTE DO DESIGN** - o design nao propoe nenhuma fonte de preco |
| 8 | "Nenhum restart de daemon necessario" | **INCORRETO** para o fix de `hermes.go` |
| 9 | Impacto nos rollups horario/diario | **OMITIDO** pelo design |

## 1. MEDICAO QUE ANCORA TUDO (banco real, read-only)

```sql
select provider, count(*), sum(input_tokens), sum(output_tokens)
from task_usage group by provider order by 2 desc;
```
```
claude|123|4801162|10396
codex |  5|  161421| 19442
```
**`antigravity` e `kiro` nao tem NENHUMA linha em `task_usage`.** Nao e "registram zero": nao
registram linha alguma. Isso e consequencia direta de `hermes.go:421-432`, que so monta o
`usageMap` quando algum campo e maior que zero — sem tokens, nenhum registro e emitido e o handler
nunca recebe entrada para fazer upsert. Distinguir "zero tokens" de "ausencia de registro" importa:
qualquer dashboard de custo por conta vai mostrar AGY e Kiro como inexistentes, nao como gratuitos.

## 2. `account_id` - CONFIRMADO, mas a premissa da origem esta furada

Schema real, `migrations/032_task_usage.up.sql:1-14` (literal):
```sql
CREATE TABLE task_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (task_id, provider, model)
);
```
Params gerados, `pkg/db/generated/task_usage.sql.go:408-416` (literal):
```go
type UpsertTaskUsageParams struct {
	TaskID           pgtype.UUID `json:"task_id"`
	Provider         string      `json:"provider"`
	Model            string      `json:"model"`
	InputTokens      int64       `json:"input_tokens"`
	OutputTokens     int64       `json:"output_tokens"`
	CacheReadTokens  int64       `json:"cache_read_tokens"`
	CacheWriteTokens int64       `json:"cache_write_tokens"`
}
```
O design esta correto nos dois pontos. **Mas a origem que ele propoe nao produz dado:**
a query sugerida faz `JOIN assignments`, e `assignments` tem ZERO linhas (FATO 5 do TL, e coerente
com a auditoria da ponte que registrei em `sol-bridge-registry-to-multica-correction.md`). Alem
disso o pacote Go `internal/daemon/rotation`, que era o unico escritor dessas tabelas, **nao existe
mais no HEAD** — `ls -d internal/daemon/rotation` falha e nenhum `.go` importa `rotation`.

Consequencia pratica: aplicar a migration 127 e o wiring do design resultaria em
`account_id` **sempre NULL**, com uma coluna nova, dois indices novos e uma query nova que nunca
retorna linha. Custo de schema sem ganho de dado.

Correcao minima que eu recomendo, sem inverter a ordem: a coluna e correta e deve entrar, mas o
**preenchimento** depende de existir um escritor de `assignments`. Enquanto nao existir, `account_id`
e um campo estrutural reservado, nao uma fonte de billing. Isso tem de estar escrito na migration,
nao presumido.

## 3. `thinking_level` - CONFIRMADO; Opcao B correta, Opcao A deve ser barrada

Unica migration do campo, `migrations/095_agent_thinking_level.up.sql:8` (literal):
```sql
ALTER TABLE agent ADD COLUMN thinking_level TEXT;
```
Existe em `agent`, nao em `task_usage`. O design esta certo.

**Rejeito a Opcao A do design** (derivar o tier por `model LIKE '%-high'` etc.). O proprio design
admite a fragilidade ("frágil para modelos sem sufixo padrão"), e o dispatch proibe inferencia por
sufixo sem regra explicita. Tres motivos concretos:
1. O catalogo medido do OmniRoute tem IDs como `auto/best-coding` e `owned_by: combo`, sem sufixo
   algum de tier — cairiam em `'standard'` silenciosamente.
2. `-thinking` e `-high` nao sao a mesma dimensao: `claude-opus-4-6-thinking` e modo de raciocinio,
   `gemini-3.6-flash-high` e nivel. Colapsar os dois em uma coluna `thinking_tier` perde semantica.
3. Um `CASE` em SQL duplicado entre rollup e frontend cria duas verdades divergentes na primeira
   vez que um sufixo novo aparecer.

Aceito a Opcao B com uma correcao: a coluna deve ser `NOT NULL DEFAULT ''` (como o design propoe) e
o valor **so** pode vir do daemon, que e quem conhece `thinkingLevel` no escopo do `runTask`. Se o
payload nao trouxer o campo, o valor fica `''` e o consumidor trata como "nao declarado" — nunca
adivinha a partir do nome do modelo.

## 4. PRECO VERSIONADO POR TIER - a maior lacuna, e o design nao a cobre

O design **nao propoe nenhuma fonte de preco**. Nao ha tabela, coluna, versao nem data de vigencia
em nenhuma das suas secoes. O que existe hoje no HEAD e uma tabela em codigo:

`internal/metrics/pricing.go:8-15` (literal):
```go
type ModelPrice struct {
	Provider       string
	Model          string
	InputPerM      float64
	CacheReadPerM  float64
	CacheWritePerM float64
	OutputPerM     float64
}
```
`internal/metrics/pricing.go:17-39`: mapa `modelPrices` com 22 entradas hardcoded
(ex.: `"anthropic:claude-opus-4.6": {InputPerM: 5.00, CacheReadPerM: 0.50, CacheWritePerM: 6.25, OutputPerM: 25.00}`).
`internal/metrics/pricing.go:41-66`: `modelAliasRules`, 21 regex de alias.
`internal/metrics/pricing.go:68-77`: `PriceForModelAlias(model string) (ModelPrice, bool)`.
`internal/metrics/pricing.go:79-84`: `tokenCostUSD(tokens int64, pricePerM float64) float64`.

Unico consumidor, `internal/metrics/business.go:259-274` (literal, recortado):
```go
	price, priced := PriceForModelAlias(modelAlias)
	...
	m.recordPricedTokens(price.Provider, price.Model, "input", runtimeMode, source, inputTokens, tokenCostUSD(inputTokens, price.InputPerM))
	m.recordPricedTokens(price.Provider, price.Model, "output", runtimeMode, source, outputTokens, tokenCostUSD(outputTokens, price.OutputPerM))
	m.recordPricedTokens(price.Provider, price.Model, "cache_read", runtimeMode, source, cacheReadTokens, tokenCostUSD(cacheReadTokens, price.CacheReadPerM))
	m.recordPricedTokens(price.Provider, price.Model, "cache_write", runtimeMode, source, cacheWriteTokens, tokenCostUSD(cacheWriteTokens, price.CacheWritePerM))
```
O destino e o contador Prometheus `multica_llm_cost_usd_total` (`business.go:118-123`). Nao existe
`cost_usd` em nenhuma migration: `ls migrations/ | grep -iE 'price|cost|rate'` nao retorna nada.

Quatro conclusoes de auditoria:
1. **Origem unica de preco JA EXISTE** e e `internal/metrics/pricing.go`. Qualquer proposta de custo
   tem de partir dela em vez de inventar um segundo lugar; senao o produto passa a ter duas verdades
   de preco (Prometheus e billing) que divergem na primeira atualizacao de tabela.
2. **Nao ha versionamento.** O mapa nao tem `effective_from`, `version` nem moeda. Recomputar custo
   historico apos um reajuste de preco produz numero diferente para o mesmo periodo, sem trilha.
   Para billing isso e inaceitavel; para metrica de Prometheus (contador incremental) passa.
3. **Nao ha dimensao de tier.** As regras de alias colapsam tier: `gemini-3[.]1-pro` casa tanto
   `gemini-3.1-pro-high` quanto `-low`, e `claude-opus-4[-.]6` casa `claude-opus-4-6-thinking`.
   Ou seja, hoje **um Opus thinking e cobrado igual a um Opus normal**. Se o objetivo do GTL-03 e
   preco por tier, a lacuna esta aqui, e nao em `task_usage`.
4. **Nao ha preco para AGY nem para Kiro.** As 22 entradas cobrem openai, anthropic, deepseek,
   minimax e google por nome de modelo do vendor. Os modelos medidos no slot AGY
   (`gemini-3.6-flash-*`, `claude-opus-4-6-thinking`, `claude-sonnet-4-6`) so casariam por acidente
   de regex, e `gemini-3.6-flash` **nao tem entrada** (existe `gemini-3-flash`, `gemini-2.5-flash`).
   Sem entrada, `PriceForModelAlias` devolve `ok=false` e o custo simplesmente nao e contado.

Arquitetura minima correta que eu recomendo, aditiva e em uma unica fonte:
- manter `internal/metrics/pricing.go` como **origem unica**, e estende-la com as duas dimensoes que
  faltam: `Tier string` na chave e `EffectiveFrom time.Time` por entrada;
- chave de preco explicita `provider:model:tier` (ex. `google:gemini-3.6-flash:high`), com o tier
  vindo do campo `thinking_level` gravado em `task_usage` (Opcao B), **nunca** do nome do modelo;
- `PriceForModelAlias` ganha um irmao `PriceFor(provider, model, tier string, at time.Time)`, e o
  antigo passa a delegar com `tier=""` e `at=now`, preservando `business.go:259` sem alteracao de
  comportamento;
- ausencia de preco continua sendo `ok=false` e deve ser **observada** (contador de
  `price_lookup_miss` por provider/model/tier), nao silenciada. Hoje um miss e invisivel.

Nao proponho tabela SQL de preco: seria uma segunda fonte. Se o owner quiser preco editavel em
runtime, essa e uma decisao de produto, nao um detalhe de implementacao — e nao e minha para tomar.

## 5. AGY snake_case - o design NAO COMPROVA a causa

Codigo real, `pkg/agent/hermes.go:1190-1206` (literal):
```go
func (c *hermesClient) handleUsageUpdate(data json.RawMessage) {
	var msg struct {
		Usage struct {
			InputTokens      int64 `json:"inputTokens"`
			OutputTokens     int64 `json:"outputTokens"`
			TotalTokens      int64 `json:"totalTokens"`
			CachedReadTokens int64 `json:"cachedReadTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	c.usageMu.Lock()
	// Usage updates from ACP are cumulative snapshots, so take the latest.
	if msg.Usage.InputTokens > c.usage.InputTokens {
		c.usage.InputTokens = msg.Usage.InputTokens
```
O design escreve, literalmente: "**Se** o campo `usage` da resposta ACP usa snake_case
(`input_tokens`) em vez de camelCase (`inputTokens`), o struct em L1192-1197 não deserializa".
Isso e hipotese condicional, nao causa medida. Nao ha no documento nenhuma captura de payload ACP
do `agy`, nem log, nem teste que mostre a chave real.

Observacao tecnica que reduz a plausibilidade: `json.Unmarshal` do Go e **case-insensitive** no
casamento de nome de campo, mas **nao** ignora separadores. `inputTokens` casa `InputTokens` e
`inputtokens`; **nao** casa `input_tokens`. Portanto a hipotese e mecanicamente possivel — porem
segue nao verificada, e um patch escrito sobre ela pode nao corrigir nada.

Alem disso, o proprio design admite em 3.1 que existe uma segunda via de acumulo,
`hermes.go:380-385`, somando `promptDone.usage`. Se essa via funciona, o campo do
`usage_update` nao e o unico suspeito. **Nao aceito o fix sem captura previa do payload.**

O que eu recomendo como passo minimo e barato, antes de qualquer patch: instrumentar temporariamente
o `handleUsageUpdate` para registrar apenas as **chaves** do objeto `usage` (nunca valores de
conteudo), rodar uma task AGY, e ler as chaves reais. Sem isso, o patch multi-formato do design e
uma aposta. Nao executei essa instrumentacao: exigiria escrita em codigo e build, ambos proibidos
agora.

Se, apos a captura, o multi-formato se confirmar necessario, o helper `get(keys ...string)` proposto
no design e uma solucao razoavel — com duas ressalvas: (a) usar `json.Number` em vez de `float64`
para nao perder precisao em contagens grandes; (b) manter a semantica de snapshot cumulativo
(`if novo > atual`), que o esboco do design preserva corretamente.

## 6. Kiro - gap plausivel, nao comprovado

O design conclui "se o Kiro CLI 2.x nao emite `usage_update`, nao ha extracao possivel". A conclusao
e coerente com a arquitetura (`hermesClient` depende da notificacao ACP), mas tambem nao ha captura
de protocolo no documento. Duas alternativas que o design nao considerou e que eu registro sem
recomendar nenhuma como pronta:
1. o `result` de `session/prompt` pode trazer usage — o proprio design levanta e abandona;
2. o Kiro grava sessao em `data.sqlite3` sob `xdg-data/kiro-cli/`, o que **poderia** conter contagem
   — mas ler o store de credencial do vendor para fins de billing e decisao de politica, nao
   detalhe tecnico. Nao recomendo sem ruling.
Enquanto nao houver medicao, "gap de vendor" e a hipotese mais provavel e deve ser documentada como
hipotese, nao como fato.

## 7. Dois erros do design que passariam batido

### 7.1 "Nenhum restart de daemon necessario" - INCORRETO
O resumo final do design afirma: "Nenhum restart de daemon necessário para as mudanças de schema +
handler." Verdadeiro para migration e handler, que vivem no backend. **Falso para o item de
`hermes.go`**: `pkg/agent/hermes.go` e compilado **no binario do daemon**, hoje
`multica-auth-credential-home-v1` rodando como unit systemd no ORQ2 (MainPID 3240496). Qualquer fix
de deserializacao de usage exige recompilar o daemon e reiniciar a unit. A tabela de resumo lista o
fix de `hermes.go` na mesma leva que as mudancas de backend, o que subestima o blast radius.

### 7.2 Rollups nao foram considerados
Existem `migrations/073_task_usage_daily_rollup`, `084_task_usage_dashboard_rollup`,
`101_task_usage_hourly_schema` e `102_task_usage_hourly_pipeline`, e as tabelas
`task_usage_hourly`, `task_usage_hourly_dirty` e `task_usage_hourly_rollup_state` existem no banco.
O design adiciona duas colunas a `task_usage` e discute rollup por `account_id` ("índice composto
para rollup por conta+provider+model") **sem tocar em nenhum dos quatro artefatos de rollup**. Um
indice novo nao faz o agregador emitir a nova dimensao. Se `account_id` e `thinking_level` devem
aparecer em custo agregado, o pipeline horario e o diario precisam de mudanca explicita, e ela nao
esta no documento.
Ponto correto do design que confirmo: a UNIQUE `(task_id, provider, model)` pode permanecer, e o
comentario em `pkg/db/generated/task_usage.sql.go:418-420` mostra que o upsert bumpa `updated_at`
justamente para o worker de rollup marcar o bucket como sujo — logo adicionar colunas sem mexer no
upsert **nao** invalida a deteccao de sujeira.

## 8. ORDEM MINIMA CORRETA (recomendacao, nao decisao)

1. **Medir antes de corrigir**: capturar as chaves reais do `usage` ACP de AGY e de Kiro. Sem isso,
   os itens de `hermes.go` e de Kiro sao apostas. Custo: uma task por provider.
2. **Preco**: estender `internal/metrics/pricing.go` com `tier` e `EffectiveFrom`, adicionar
   entradas para os modelos AGY realmente disponiveis, e instrumentar o miss de lookup. Sem isso,
   AGY e Kiro continuam com custo zero mesmo depois de contarem tokens.
3. **`thinking_level` em `task_usage`** (Opcao B, `NOT NULL DEFAULT ''`), preenchido pelo daemon.
   Nunca derivado por sufixo.
4. **`account_id` em `task_usage`**, nullable, com a limitacao registrada na propria migration:
   permanece NULL enquanto nao existir escritor de `assignments`.
5. **Rollups**: decidir explicitamente se as duas dimensoes novas sobem para horario/diario, e
   alterar os quatro artefatos se sim.
6. **Testes** minimos, na ordem em que falhariam hoje:
   - `PriceFor` resolve `google:gemini-3.6-flash:high` e `anthropic:claude-opus-4-6:thinking` com
     precos distintos do tier base;
   - lookup miss incrementa o contador de miss em vez de custar zero em silencio;
   - `PriceForModelAlias` legado continua devolvendo o mesmo valor de hoje (regressao de
     `business.go:259`);
   - upsert com `thinking_level=''` e com valor preenchido nao viola a UNIQUE existente;
   - `handleUsageUpdate` acumula corretamente snapshot cumulativo para o formato de chave que a
     captura do passo 1 comprovar;
   - `task_usage` recebe linha para `antigravity` quando a usage for maior que zero — hoje o banco
     tem zero linhas para esse provider, entao esse teste e o unico que prova a correcao ponta a ponta.

## 9. NAO-AFIRMACOES
- READ-ONLY: nao editei codigo, nao compilei, nao rodei build, deploy, restart, commit ou push.
  Gravei apenas este arquivo e o meu check-out.
- Nao apliquei nem testei nenhum patch; nao ha codigo nesta entrega.
- Nao capturei payload ACP de AGY nem de Kiro: por isso classifico as causas 5 e 6 como nao
  comprovadas, e nao afirmo causa alternativa.
- Nao consultei `accounts`/`approved_accounts`/`assignments` neste ciclo: a informacao de zero
  linhas e o FATO 5 do TL, cruzada com a ausencia do pacote `internal/daemon/rotation` que eu medi
  antes e registrei em `sol-bridge-registry-to-multica-correction.md`.
- Nao inspecionei os "patches parciais ORQ-12/13" como diffs: nao localizei artefato de patch
  separado no repo alem do proprio `cost-accounting-design.md`; auditei o design e o HEAD.
- Nao propus tabela SQL de preco nem preco editavel em runtime: seria segunda fonte de verdade e e
  decisao de produto do owner.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
- Encerrando conforme o comunicado: nenhuma frente nova, nenhum pedido de tarefa.
