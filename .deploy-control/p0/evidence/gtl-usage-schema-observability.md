# GTL-17 - instrumentacao minima e segura para observar o SCHEMA de usage (AGY / Kiro / Codex)

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T11:45Z
Modo: READ-ONLY / DESENHO. Nao editei codigo, nao compilei, nao rodei build, runtime, deploy ou
restart. Gravei somente este arquivo e o meu check-out. Base: GTL-03
(`.deploy-control/p0/evidence/gtl-cost-implementation-audit.md`).

## 0. OBJETIVO E LINHA VERMELHA

Objetivo: descobrir **quais chaves** o objeto `usage` realmente traz em AGY, Kiro e Codex, para que
o parser seja corrigido com base em medicao e nao na hipotese snake_case que o GTL-03 classificou
como **nao comprovada**.

Linha vermelha, nao negociavel: a instrumentacao observa **nome e tipo JSON da chave**, jamais
valor. Nenhum numero de token, nenhum trecho de prompt, nenhuma resposta de modelo, nenhum
identificador de conta, nenhuma credencial. O motivo nao e so privacidade: contagem de token e dado
de billing e prompt e conteudo de cliente; nada disso precisa sair para responder "qual e o formato".

## 1. FATO QUE JUSTIFICA A INSTRUMENTACAO

Medicao do GTL-03, banco real:
```
select provider, count(*), sum(input_tokens), sum(output_tokens) from task_usage group by provider;
claude|123|4801162|10396
codex |  5|  161421| 19442
```
`antigravity` e `kiro` nao tem NENHUMA linha. A causa proxima e o gate de emissao
(`hermes.go:426-432`), que so cria o `usageMap` se algum campo for maior que zero. A causa raiz do
zero — chave errada, notificacao ausente, ou ambas — e exatamente o que nao esta medido.

## 2. OS DOIS CAMINHOS HERMES (traco exato)

`kiro` e `antigravity` usam o mesmo `hermesClient`. Existem **dois** pontos onde usage entra, e um
terceiro que e apenas roteamento para o segundo.

### Caminho 1 - notificacao ACP `usage_update`
Dispatch, `hermes.go:797-798` (literal):
```go
	case "usage_update":
		c.handleUsageUpdate(updateData)
```
Handler, `hermes.go:1190-1201` (literal):
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
```
Semantica: snapshot **cumulativo**, aplicado por maximo em `hermes.go:1205-1213`.

### Caminho 2 - resultado do RPC `session/prompt`
Gatilho, `hermes.go:725-727` (literal):
```go
		// If this is a prompt response, extract usage and stop reason.
		if pr.method == "session/prompt" {
			c.extractPromptResult(raw["result"])
		}
```
Extrator, `hermes.go:732-743` (literal):
```go
func (c *hermesClient) extractPromptResult(data json.RawMessage) {
	var resp struct {
		StopReason string `json:"stopReason"`
		Usage      *struct {
			InputTokens      int64 `json:"inputTokens"`
			OutputTokens     int64 `json:"outputTokens"`
			TotalTokens      int64 `json:"totalTokens"`
			ThoughtTokens    int64 `json:"thoughtTokens"`
			CachedReadTokens int64 `json:"cachedReadTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
```
Acumulo, `hermes.go:380-385` (literal), via `onPromptDone`:
```go
				// Merge usage from the PromptResponse.
				c.usageMu.Lock()
				c.usage.InputTokens += pr.usage.InputTokens
				c.usage.OutputTokens += pr.usage.OutputTokens
				c.usage.CacheReadTokens += pr.usage.CacheReadTokens
				c.usageMu.Unlock()
```
Semantica: **soma** (`+=`), diferente do Caminho 1.

### Caminho 2b - notificacao `turn_end` reaproveita o mesmo extrator
`hermes.go:800-801` (literal):
```go
	case "turn_end":
		c.extractPromptResult(updateData)
```
Consequencia relevante para o desenho: os dois caminhos tem **schemas declarados diferentes** —
o Caminho 2 aceita `thoughtTokens` e trata `usage` como ponteiro opcional; o Caminho 1 nao tem
`thoughtTokens` e usa struct por valor. Instrumentar apenas um dos dois deixaria metade do problema
invisivel. E `extractPromptResult` e alcancado por duas origens distintas (RPC result e `turn_end`),
o que precisa aparecer no rotulo, senao nao se sabe de onde a chave veio.

### Gate de emissao (por que o zero e silencioso)
`hermes.go:421-432` (literal):
```go
		// Build usage map.
		c.usageMu.Lock()
		u := c.usage
		c.usageMu.Unlock()

		var usageMap map[string]TokenUsage
		if u.InputTokens > 0 || u.OutputTokens > 0 || u.CacheReadTokens > 0 {
			model := effectiveModel
			if model == "" {
				model = "unknown"
			}
			usageMap = map[string]TokenUsage{model: u}
		}
```

### Caminho Codex (contraste, ja funciona)
`codex.go:1863-1874` (literal, recortado):
```go
func (c *codexClient) extractUsageFromMap(data map[string]any) {
	// Try common field names for usage data.
	var usageMap map[string]any
	for _, key := range []string{"usage", "token_usage", "tokens"} {
		if v, ok := data[key].(map[string]any); ok {
			usageMap = v
			break
		}
	}
	if usageMap == nil {
		return
	}
```
e a leitura de campo por multiplos aliases, `codex.go:1882-1887` (literal):
```go
	inputTokens := codexInt64(usageMap, "input_tokens", "input", "prompt_tokens")
	cacheReadTokens := codexInt64(usageMap, "cached_input_tokens", "cache_read_tokens", "cache_read_input_tokens")
	c.usage.InputTokens += codexUncachedInputTokens(inputTokens, cacheReadTokens)
	c.usage.OutputTokens += codexInt64(usageMap, "output_tokens", "output", "completion_tokens")
	c.usage.CacheReadTokens += cacheReadTokens
	c.usage.CacheWriteTokens += codexInt64(usageMap, "cache_write_tokens", "cache_creation_input_tokens")
```
O Codex ja e tolerante a formato **por design**; o Hermes nao. Codex entra na instrumentacao como
**grupo de controle**: e o unico dos tres com linhas em `task_usage`, logo serve para provar que a
sonda funciona antes de confiar nela para AGY e Kiro.

## 3. PONTOS EXATOS DE INSTRUMENTACAO (4 sondas, uma linha de log cada)

| # | arquivo:linha | ponto | rotulo `path` |
|---|---|---|---|
| S1 | `hermes.go:1199` (imediatamente antes do `json.Unmarshal`) | notificacao `usage_update` | `acp.usage_update` |
| S2 | `hermes.go:742` (imediatamente antes do `json.Unmarshal`) | result de `session/prompt` e `turn_end` | `acp.prompt_result` |
| S3 | `hermes.go:426` (antes do gate) | resultado agregado: quais buckets ficaram em zero | `hermes.emit_gate` |
| S4 | `codex.go:1872` (quando `usageMap == nil`) | Codex nao achou objeto usage | `codex.usage_absent` |

S1 e S2 sao os que respondem a pergunta do formato. S3 responde "chegou usage mas o gate barrou".
S4 e o controle. Nenhuma sonda altera fluxo: sao observacao pura, sem `return` novo, sem mudanca de
condicao.

## 4. O QUE A SONDA EMITE (e o que ela NUNCA emite)

Funcao unica, colocada uma vez no pacote `agent`, chamada pelas quatro sondas:
```go
// usageSchemaKeys devolve APENAS nome e tipo JSON das chaves de primeiro e
// segundo nivel do objeto usage. Nunca valores. Usada exclusivamente pela
// sonda de schema, atras de feature flag.
func usageSchemaKeys(raw json.RawMessage) []string {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return []string{"<unparseable>"}
	}
	out := make([]string, 0, 16)
	for k, v := range top {
		out = append(out, k+":"+jsonKind(v))
		if jsonKind(v) == "object" {
			var inner map[string]json.RawMessage
			if json.Unmarshal(v, &inner) == nil {
				for ik, iv := range inner {
					out = append(out, k+"."+ik+":"+jsonKind(iv))
				}
			}
		}
	}
	sort.Strings(out) // determinismo: mapa Go tem ordem aleatoria
	return out
}

// jsonKind classifica pelo primeiro byte, sem materializar o valor.
func jsonKind(v json.RawMessage) string {
	s := bytes.TrimSpace(v)
	if len(s) == 0 { return "empty" }
	switch s[0] {
	case '{': return "object"
	case '[': return "array"
	case '"': return "string"
	case 't', 'f': return "bool"
	case 'n': return "null"
	default:  return "number"
	}
}
```
Saida de exemplo, hipotetica: `["usage:object","usage.inputTokens:number","usage.totalTokens:number"]`.

**Garantias de redaction, por construcao e nao por filtro:**
1. O valor **nunca** e convertido para string nem interpolado. `jsonKind` olha um unico byte
   (`s[0]`) e descarta o resto. Nao existe caminho no codigo em que o conteudo chegue ao log.
2. Profundidade fixa em 2 niveis. Sem recursao arbitraria, logo um `usage` que contenha um bloco
   de conteudo aninhado nao vaza estrutura profunda.
3. So o subobjeto `usage` e passado para a sonda — nunca o envelope inteiro da notificacao, que
   carrega `content`, `text` e argumentos de tool call.
4. Nomes de chave sao metadados de protocolo do vendor, nao dado de cliente. Ainda assim: se algum
   vendor colocar identificador em **nome** de chave (ex.: `usage.acct_<id>`), a sonda vazaria esse
   nome. Mitigacao proposta: truncar cada nome em 40 caracteres e rejeitar nomes que casem
   `[0-9a-f]{8,}` substituindo por `<redacted-key-name>`. Custo baixo, fecha o unico furo real.
5. `<unparseable>` como valor sentinela quando o JSON nao decodifica — nunca ecoar o payload cru.

**O que a sonda nao pode fazer, explicitamente:** nao logar `data` cru, nao logar `raw["result"]`,
nao logar `u` de `hermes.go:423`, nao logar `msg.Usage.*`, nao logar `effectiveModel` junto de
contagem, nao logar `sessionID` (correlacionavel a conta).

## 5. FEATURE FLAG

Proposta: **uma** variavel de ambiente booleana, default OFF, lida uma vez.
```go
var usageSchemaProbeEnabled = os.Getenv("MULTICA_USAGE_SCHEMA_PROBE") == "1"
```
Justificativa das escolhas:
- **Env var e nao config de banco**: o pacote `pkg/agent` roda dentro do binario do daemon e nao tem
  acesso ao Postgres; qualquer flag de banco exigiria plumbing novo. Precedente no proprio pacote:
  `os.Getenv("CLAUDE_FAKE_MODE")` e `os.Getenv("CODEX_HOME")` sao as duas unicas env vars ja lidas
  em `pkg/agent`, entao o padrao existe.
- **Default OFF e leitura unica em `var`**: custo zero quando desligada (uma comparacao de bool por
  chamada, nenhuma alocacao, nenhuma leitura de env em hot path).
- **Sem nivel de verbosidade**: um unico bit reduz a chance de alguem ligar "tudo" por engano.
- **Nome com prefixo `MULTICA_`**: coerente com `MULTICA_*` usado no resto do daemon.

Onde ligar: a unit systemd do daemon no ORQ2
(`multica-daemon-orq2-credential.service`, MainPID medido 3240496). Ligar exige editar o
`Environment=` da unit e reiniciar — ou seja **nao e mudanca de runtime a quente**. Isso e
deliberado: uma sonda de protocolo nao deve ser ligavel sem gate.

## 6. CARDINALIDADE

A sonda emite **log**, nao metrica com label livre. Isso e escolha de cardinalidade, nao de
conveniencia: nome de chave de vendor e um conjunto aberto, e virar label de Prometheus criaria
series novas a cada formato desconhecido — exatamente o anti-padrao de cardinalidade.

Limites propostos:
- **Uma linha por processo de CLI por caminho**, com `sync.Once` por `(path)`, nao por chamada.
  `usage_update` e cumulativo e pode chegar dezenas de vezes por task; logar todas nao acrescenta
  informacao, porque o schema nao muda dentro da sessao.
- Se o conjunto de chaves **mudar** dentro do mesmo processo, emitir uma segunda linha e parar
  (limite duro de 2 por caminho por processo). Cobre o caso raro de o vendor mudar o formato entre
  o `usage_update` e o `turn_end`.
- Teto absoluto de 4 linhas por task (S1, S2, S3, S4), independentemente de reentrancia.
- Campos do log, todos de baixa cardinalidade: `path` (4 valores possiveis), `provider`
  (`antigravity`, `kiro`, `codex`), `keys` (lista ordenada e truncada). Sem `task_id`, sem
  `session_id`, sem `model`.

Se depois se quiser metrica, o unico formato aceitavel e um contador com label **fechado**
`usage_schema_variant` assumindo valores enumerados (`camel`, `snake`, `mixed`, `absent`,
`unparseable`) — derivado da lista de chaves, nunca a lista em si. Registro como opcao, nao
recomendo agora.

## 7. COMO PROVAR QUAL FORMATO CHEGA, ANTES DE QUALQUER PATCH DE PARSER

Sequencia minima, e a ordem importa:
1. **Validar a sonda no grupo de controle.** Ligar a flag e rodar **uma** task Codex. `codex` e o
   unico dos tres com linhas em `task_usage` (5 linhas medidas), logo tem usage comprovadamente
   diferente de zero. Esperado: S4 **nao** dispara e o custo continua sendo contado. Se S4 disparar,
   a sonda esta no ponto errado e nada mais do experimento e confiavel.
2. **Uma task AGY.** Ler S1 e S2. Tres resultados possiveis e o que cada um significa:
   - S1 e S2 nao emitem nada -> o AGY **nao envia** objeto `usage` em nenhum dos dois caminhos.
     Patch de parser multi-formato seria inutil; o problema e de protocolo.
   - S1/S2 emitem com `usage.input_tokens:number` -> confirma a hipotese snake_case do design
     original, e so entao o patch multi-formato se justifica.
   - S1/S2 emitem com `usage.inputTokens:number` -> o formato ja e o esperado, e a falha esta em
     outro lugar (valor zero na origem, ou o `usage` como ponteiro nulo em `hermes.go:738`).
     Nesse caso S3 discrimina: se S3 mostra buckets em zero, os valores chegaram zerados.
3. **Uma task Kiro.** Mesma leitura. A hipotese do GTL-03 e que o Kiro nao emite `usage_update`;
   se S1 nao dispara e S2 dispara sem `usage`, isso passa de hipotese a fato medido, e a conclusao
   "gap de vendor" fica sustentada por evidencia.
4. **Desligar a flag** e so entao escrever o parser, com o formato provado no passo 2 e 3.

Criterio de decisao explicito: **nenhum patch de parser antes do passo 4**. O GTL-03 barrou o patch
justamente porque ele se baseava em um "se". Escrever parser tolerante a formatos que ninguem
observou adiciona caminhos mortos e mascara o defeito real.

Observacao importante sobre o AGY: o T3 mediu que AGY hoje falha **antes** de chegar a usage, com
`unsupported credential path type .../antigravity-cli/cli.log` em `antigravity_home.go`. Enquanto
esse defeito de preparo existir em um slot, a task AGY nao chega a emitir `usage` e a sonda nao vai
observar nada — o resultado seria um falso "AGY nao envia usage". **Pre-condicao do passo 2: rodar
em um slot cuja preparacao conclua**, ou o experimento e invalido. Nao verifiquei quais slots
concluem hoje; isso precisa ser confirmado por quem executar.

## 8. TESTES (unitarios, sem CLI real, sem rede)

Todos sobre `usageSchemaKeys`/`jsonKind` e sobre as sondas com payload sintetico. Nenhum exige
runtime de vendor.

1. `TestUsageSchemaKeysCamelCase` — entrada `{"usage":{"inputTokens":1,"outputTokens":2}}` produz
   exatamente `["usage:object","usage.inputTokens:number","usage.outputTokens:number"]`.
2. `TestUsageSchemaKeysSnakeCase` — entrada com `input_tokens` produz a chave snake, provando que a
   sonda distingue os dois formatos (que e todo o proposito).
3. `TestUsageSchemaKeysNeverLeaksValues` — teste **negativo** e o mais importante: para uma entrada
   com valor sentinela improvavel (ex.: `{"usage":{"inputTokens":987654321}}` e um campo string
   `{"note":"SECRET-CANARY"}`), a saida concatenada nao contem `987654321` nem `SECRET-CANARY`.
4. `TestUsageSchemaKeysDeterministicOrder` — duas chamadas com o mesmo payload devolvem a mesma
   lista (guarda contra a ordem aleatoria de mapa em Go).
5. `TestUsageSchemaKeysDepthLimitedToTwo` — payload com 3 niveis nao emite chave de terceiro nivel.
6. `TestUsageSchemaKeysRedactsHexLikeKeyNames` — chave `acct_deadbeef12345678` sai como
   `<redacted-key-name>`.
7. `TestUsageSchemaKeysUnparseable` — payload invalido devolve `["<unparseable>"]` e nao ecoa bytes.
8. `TestUsageSchemaProbeDisabledByDefault` — com a env var ausente, nenhuma linha e emitida
   (logger de teste sem registros).
9. `TestUsageSchemaProbeEmitsOncePerPath` — 50 chamadas com o mesmo conjunto de chaves emitem 1
   linha; uma 51a chamada com conjunto diferente emite a 2a e para.
10. `TestHandleUsageUpdateBehaviorUnchangedWithProbeOn` — regressao: com a flag ligada, o valor
    acumulado em `c.usage` e identico ao da flag desligada. Prova que a sonda nao altera semantica.

## 9. IMPACTO DE REBUILD E RESTART

Este e o ponto que o design de custo original errou (registrado no GTL-03), e aqui ele e explicito.

- `pkg/agent/hermes.go` e `pkg/agent/codex.go` sao compilados **no binario do daemon**, hoje
  `/home/ec2-user/.local/lib/multica/bin/multica-auth-credential-home-v1`, executando como unit
  systemd de usuario no ORQ2, `multica-daemon-orq2-credential.service`, MainPID medido 3240496,
  `NRestarts=0` desde 2026-07-27T01:18:53Z.
- Logo a sonda **exige rebuild do daemon e restart da unit**. Nao existe caminho a quente.
- O restart derruba os 3 runtimes obrigatorios por alguns segundos. O T3 mediu o custo real desse
  tipo de janela: 6 tasks falharam com `failure_reason=runtime_offline` e 1 com `runtime_recovery`
  na troca anterior de daemon. Portanto **o restart deve ocorrer com a fila vazia**. No momento da
  medicao do T3 havia zero tasks `queued` e zero `running`, o que e a janela correta.
- O backend Docker no ORQ1 **nao** e afetado: nenhuma mudanca em `internal/handler` nem em
  migration.
- Reversao: desligar a flag reverte o comportamento sem rebuild, mas remover o codigo da sonda
  exige novo rebuild. Recomendo tratar a sonda como temporaria e removive-la no mesmo ciclo em que
  o parser correto entrar — codigo de diagnostico que fica vira divida.
- Ressalva do encerramento anterior, que respeito: o AGY esta **sem rede de seguranca**, nao existe
  rollback de um comando e os backups anteriores ao patch token-only reintroduziriam o AGY
  task-incapaz. Portanto qualquer rebuild do daemon precisa de gate do General-Tech-Lead e de plano
  de volta que **nao** seja "reaplicar a branch antiga".

## 10. RESUMO DA MUDANCA PROPOSTA (para o escritor, nao aplicada)

| item | arquivo | tipo | linhas aprox. |
|---|---|---|---|
| `usageSchemaKeys` + `jsonKind` + `sync.Once` por path | novo `pkg/agent/usage_schema_probe.go` | novo arquivo | ~70 |
| flag `MULTICA_USAGE_SCHEMA_PROBE` | mesmo arquivo | `var` de pacote | 1 |
| sonda S1 | `pkg/agent/hermes.go:1199` | 1 chamada guardada por flag | 3 |
| sonda S2 | `pkg/agent/hermes.go:742` | 1 chamada guardada por flag | 3 |
| sonda S3 | `pkg/agent/hermes.go:426` | 1 chamada guardada por flag | 3 |
| sonda S4 | `pkg/agent/codex.go:1872` | 1 chamada guardada por flag | 3 |
| testes | novo `pkg/agent/usage_schema_probe_test.go` | 10 testes | ~180 |

Logger: `hermesClient` **nao tem** campo de logger (struct em `hermes.go:455-479`), mas
`agent.Config` tem `Logger *slog.Logger` (`agent.go:134,143`) e `hermesClient` carrega
`cfg Config` (`hermes.go:455`). Portanto a sonda deve usar `c.cfg.Logger` com guarda de nil, sem
adicionar campo novo nem logger global. Para `codexClient`, confirmar o mesmo padrao antes de
escrever — nao verifiquei essa struct.

## 11. NAO-AFIRMACOES
- READ-ONLY: nao editei codigo, nao criei o arquivo da sonda, nao compilei, nao rodei build, teste,
  deploy, restart ou rerun. Gravei apenas este arquivo e o meu check-out.
- Nao executei a sonda: nao sei qual formato AGY e Kiro enviam. Este documento desenha o
  experimento, nao entrega o resultado.
- Nao verifiquei a struct `codexClient` quanto a disponibilidade de logger; assumi por simetria e
  marquei como a confirmar.
- Nao confirmei quais slots AGY concluem a preparacao hoje, o que e pre-condicao do passo 2 do
  experimento por causa do defeito `unsupported credential path type` medido no T3.
- Nao propus metrica de Prometheus com label de nome de chave: seria explosao de cardinalidade.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
- Os arquivos de codigo modificados no repo nao sao meus: pertencem ao escritor unico.
