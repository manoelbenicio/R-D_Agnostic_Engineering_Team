# PACOTE 2 - p2-gateway-client - validacoes do catalogo vindo do OmniRoute

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-26T22:50Z
Arquivo alvo: `multica-auth-work/server/internal/daemon/gateway/client.go`, `FetchModels` na linha 178.
Modo: PROPOSTA. Nenhum arquivo de codigo editado, nada recompilado, binario nao substituido,
daemon nao reiniciado, nada commitado. Aguarda assinatura do Codex56-TL e decisao do owner.

## 0. Medicao real do gateway (feita agora, read-only, ORQ1 -> 100.118.244.61:20128)

```
$ curl -sS -D - -o /tmp/gwmodels.json -w "http=%{http_code} bytes=%{size_download}\n" \
    -H "Accept: application/json" -H "Authorization: Bearer <secret-file, nunca impresso>" \
    http://100.118.244.61:20128/v1/models
http=200 bytes=110807

HTTP/1.1 200 OK
x-omniroute-route-class: CLIENT_API
x-request-id: e24917a6-c3b1-4edc-bc76-ef0e47d2e38f
content-type: application/json
x-model-catalog-version: model-metadata-v1:static
Transfer-Encoding: chunked
```

```
top-level keys: ['object', 'data']
n= 327
primeiro item keys: ['id', 'object', 'created', 'owned_by', 'permission', 'root', 'parent',
                    'context_length', 'max_input_tokens', 'max_output_tokens', 'capabilities']
id= auto/best-coding   owned_by= combo
capabilities= {"tool_calling": true, "reasoning": true, "thinking": true, "temperature": true}
```

```
$ curl -sS -o /tmp/gw403.json -w "http=%{http_code} bytes=%{size_download}\n" http://100.118.244.61:20128/v1/models
http=401 bytes=121
{"error":{"code":"AUTH_002","message":"Authentication required","correlation_id":"f528aa27-..."}}
```

Tres fatos que mudam o desenho:
1. NAO existe header `X-OmniRoute-Registry-Version`. O gateway manda `x-model-catalog-version`.
2. NAO existe `registry_version` no corpo. Top-level e so `object` + `data`.
3. Sem credencial o gateway responde **401**, nao 403. O corpo e um envelope `error` com
   `correlation_id`.

## 1. ASSINATURA ATUAL

`client.go:178` (literal):

```go
func (c *Client) FetchModels(ctx context.Context, correlation brain.Correlation) (ModelsDocument, error) {
	response, err := c.do(ctx, operationModels, http.MethodGet, "/v1/models", correlation, true)
	if err != nil {
		return ModelsDocument{}, err
	}
	defer response.Body.Close()
	body, err := readBounded(response.Body, c.maxResponseBody)
	if err != nil {
		return ModelsDocument{}, classifyBodyReadError(operationModels, err)
	}
	if devModelsCompatEnabled() {
		// DEV-only compatibility: OmniRoute serves an OpenAI-basic /v1/models
		// shape that lacks the enriched fields the registry requires. Decode it
		// natively and project into the enriched schema. The enriched-schema
		// path below is used unchanged when the flag is absent.
		var native OmniRouteNativeModels
		if err := json.Unmarshal(body, &native); err != nil {
			return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
		}
		return ProjectOmniRouteModels(native, strings.TrimSpace(response.Header.Get(HeaderRegistryVersion))), nil
	}
	var document ModelsDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
	}
	headerVersion := strings.TrimSpace(response.Header.Get(HeaderRegistryVersion))
	if headerVersion != "" {
		if document.RegistryVersion != "" && document.RegistryVersion != headerVersion {
			return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
		}
		document.RegistryVersion = headerVersion
	}
	return document, nil
}
```

Lacunas objetivas dessa versao:
- nao valida `Content-Type`;
- nao valida contagem de modelos nem catalogo vazio (isso so aparece em `registry.go:222`);
- le apenas um nome de header de versao, e esse header nao existe na producao medida;
- se o header estiver ausente, `document.RegistryVersion` fica "" e `buildSnapshot`
  (`registry.go:222`) rejeita tudo com `ErrorProtocol` - falha correta, mas sem diagnostico;
- nao devolve proveniencia (versao de catalogo, request id, instante) ao chamador;
- timeout e o generico `c.requestTimeout` (`client.go:20`, 30s), sem teto proprio de catalogo.

## 2. ASSINATURA PROPOSTA (aditiva, nao quebra `ModelsFetcher`)

`ModelsFetcher` (`health_models.go:51`) e `registry.go:118` chamam `FetchModels(ctx)` com aridade
diferente. Por isso a proposta e ADITIVA: nasce um metodo novo com proveniencia e o antigo passa a
delegar. Zero call site alterado.

```go
// ModelsCatalog e o resultado VALIDADO de um GET /v1/models. Carrega apenas
// identificador de rota, capacidade declarada, proveniencia e correlacao.
// NUNCA carrega credencial, identidade de conta, cookie ou token.
type ModelsCatalog struct {
	Document       ModelsDocument
	CatalogVersion string    // proveniencia opaca: X-OmniRoute-Registry-Version ou X-Model-Catalog-Version
	RequestID      string    // correlacao apenas (x-request-id / X-OmniRoute-Request-Id)
	FetchedAt      time.Time
	ModelCount     int
}

// FetchModelsCatalog busca, valida e devolve o catalogo com proveniencia.
// Fail-closed: qualquer violacao devolve GatewayError e NENHUM documento parcial.
func (c *Client) FetchModelsCatalog(ctx context.Context, correlation brain.Correlation) (ModelsCatalog, error)

// FetchModels mantem a assinatura atual e delega, para nao tocar
// ModelsFetcher (health_models.go:51) nem registry.go:118.
func (c *Client) FetchModels(ctx context.Context, correlation brain.Correlation) (ModelsDocument, error) {
	catalog, err := c.FetchModelsCatalog(ctx, correlation)
	if err != nil {
		return ModelsDocument{}, err
	}
	return catalog.Document, nil
}
```

## 3. VALIDACOES PROPOSTAS

Novas constantes, junto de `client.go:20-24`:

```go
const (
	catalogFetchTimeout   = 20 * time.Second
	maxCatalogVersionLen  = 128
	HeaderModelCatalogVersion = "X-Model-Catalog-Version"
)
```

### V1. Timeout proprio do catalogo
ANTES: `client.go:221` usa `c.requestTimeout` (default 30s, `client.go:20`).
DEPOIS: `FetchModelsCatalog` aplica `context.WithTimeout(ctx, min(catalogFetchTimeout, c.requestTimeout))`
antes de chamar `c.do`.
JUSTIFICATIVA: o consumidor final tem orcamento de 40s (`daemon.go:2060`,
"model discovery exceeded the 40 second daemon limit"). Um teto de 20s no catalogo deixa margem
para retry/telemetria dentro do orcamento e impede que um `ctx` de chamador mais longo estenda a
busca. Medicao: 110.807 bytes em 200 OK, bem abaixo de 20s.

### V2. Limite de corpo - manter, nao aumentar
ANTES: `client.go:184` `readBounded(response.Body, c.maxResponseBody)`; default 2 MiB
(`client.go:21`), faixa aceita 1 KiB..16 MiB (`client.go:100-102`).
DEPOIS: inalterado. Adicionar apenas o mapeamento explicito de `errResponseTooLarge` para
`&GatewayError{Operation: operationModels, Class: ErrorProtocol, Retryable: false}`.
JUSTIFICATIVA: medido 110.807 bytes = 5,3% do orcamento. Pior caso com o teto de
`MaxRegistryModels = 1024` (`registry.go:29`) fica em torno de 350 KB. Nao ha motivo para elevar o
limite, e a resposta e `Transfer-Encoding: chunked` (sem `Content-Length`), logo o bound de leitura
e a UNICA protecao real.

### V3. Content-Type estrito (novo)
ANTES: nenhuma checagem; um HTML de portal cativo com 200 chega ate `json.Unmarshal`.
DEPOIS:
```go
mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
if err != nil || mediaType != "application/json" {
	return ModelsCatalog{}, &GatewayError{
		Operation: operationModels, Class: ErrorProtocol,
		StatusCode: response.StatusCode, RequestID: requestID, Detail: "content_type",
	}
}
```
JUSTIFICATIVA: falha precisa e barata antes de parsear 2 MiB. Medido `content-type: application/json`,
logo a checagem nao regride o caminho valido.

### V4. Versao de registry - aceitar os dois headers, nunca sintetizar
ANTES: `client.go:203-209` le so `HeaderRegistryVersion` (`health_models.go:11` =
`X-OmniRoute-Registry-Version`). Esse header NAO existe na producao medida, entao esse bloco e
codigo morto hoje e `RegistryVersion` fica vazio.
DEPOIS:
```go
version := strings.TrimSpace(response.Header.Get(HeaderRegistryVersion))
if version == "" {
	version = strings.TrimSpace(response.Header.Get(HeaderModelCatalogVersion))
}
if !validCatalogVersion(version) { // nao vazio, <= maxCatalogVersionLen, sem \r \n \x00, ASCII imprimivel
	return ModelsCatalog{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol, Detail: "catalog_version"}
}
if body := strings.TrimSpace(document.RegistryVersion); body != "" && body != version {
	return ModelsCatalog{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol, Detail: "catalog_version_mismatch"}
}
document.RegistryVersion = version
```
JUSTIFICATIVA: `X-OmniRoute-Registry-Version` continua tendo precedencia para nao quebrar contrato
futuro; `X-Model-Catalog-Version` (medido: `model-metadata-v1:static`) e o valor real de hoje.
Ausencia dos dois e FALHA, nunca valor sintetizado: um catalogo sem proveniencia nao pode alimentar
cache nem readiness, senao a retencao de 24h e a atribuicao de custo perdem ancora.

### V5. Catalogo vazio com 200 (novo, no limite do cliente)
ANTES: so `registry.go:222` rejeita `len(document.Models) == 0`, e o faz junto de outras 5
condicoes, produzindo um `ErrorProtocol` indistinguivel.
DEPOIS:
```go
if len(document.Models) == 0 {
	return ModelsCatalog{}, &GatewayError{
		Operation: operationModels, Class: ErrorProtocol, Retryable: false,
		StatusCode: response.StatusCode, RequestID: requestID, Detail: "empty_catalog",
	}
}
```
JUSTIFICATIVA: 200 com catalogo vazio e regressao do gateway, nao condicao transitoria.
`Retryable: false` para nao mascarar o defeito em backoff. `Detail` distinto separa
"gateway respondeu vazio" de "gateway respondeu malformado" na telemetria.
NAO deve haver fallback para catalogo estatico nem para probe de CLI: isso reintroduz exatamente a
dependencia de credencial local que este pacote existe para eliminar.

### V6. Teto de contagem no cliente
DEPOIS:
```go
if len(document.Models) > MaxRegistryModels {
	return ModelsCatalog{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol, Detail: "catalog_too_many_models"}
}
```
JUSTIFICATIVA: `MaxRegistryModels = 1024` (`registry.go:29`); medido 327. Rejeitar no cliente evita
construir mapa gigante em `buildSnapshot` e mantem o custo de parse limitado.

### V7. 401 e 403
MEDIDO: sem credencial o gateway devolve **401** com `{"error":{"code":"AUTH_002", ...}}`.
Nao observei 403 neste endpoint.
`c.do` (`client.go:272-274`) literal:
```go
if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
	_ = response.Body.Close()
	return nil, classifyStatus(operation, response)
}
```
Ou seja `FetchModels` NUNCA ve o corpo de um 401/403; o status ja foi classificado em
`classification.go:104-108` literal:
```go
case signal.StatusCode == http.StatusUnauthorized:
	retryable := signal.AuthOutcome == AuthOutcomeAccessExpired
	return FailureDecision{Class: ErrorAuthentication, Scope: CircuitAccount, Retryable: retryable, Quota: QuotaUnknown}
case signal.StatusCode == http.StatusForbidden:
	return FailureDecision{Class: ErrorAuthorization, Scope: CircuitAccount, Retryable: false, Quota: QuotaUnknown}
```
PROPOSTA - nao duplicar classificacao no cliente, e sim fixar a semantica de consumo:
1. `401 ErrorAuthentication` -> retry so quando `AuthOutcomeAccessExpired`; o catalogo em cache NAO
   e renovado e NAO e servido apos o TTL. Sem credencial valida nao existe catalogo.
2. `403 ErrorAuthorization` -> terminal. `RegistryRefreshFailureBackoff = 2 * time.Second`
   (`registry.go:24`) e agressivo demais para falha de autorizacao: martela o gateway a cada 2s.
   Propor `catalogAuthzBackoff = 60 * time.Second` aplicado quando
   `Class == ErrorAuthorization`, mantendo os 2s para as classes transitorias.
3. Em 401 e 403, jamais cair para descoberta via CLI e jamais servir snapshot expirado. Dropdown
   vazio com erro visivel e o comportamento correto; dropdown populado por credencial local e o
   defeito que este pacote remove.
4. Corpo de 401/403 nunca e logado. `GatewayError.Detail` ja e token sanitizado por contrato
   (`errors.go:40-43`: "never contains bodies, URLs, credentials, or free-form upstream text").

### V8. Nenhuma identidade de conta, nenhum segredo entrando
DEPOIS: validar `AccountPool` como rotulo OPACO (nao vazio, <= 128, sem `\r \n \x00`) - hoje isso
existe em `registry.go:238` e deve ser espelhado no cliente - e rejeitar o documento se algum row
trouxer campo fora do allow-list de `ModelDocumentMeta` (`health_models.go:40-42`), ou se
`account_pool` tiver forma de e-mail ou de token portador.
JUSTIFICATIVA: o principio do desenho e que o daemon recebe ID de rota, correlacao e recibo de
custo, nunca segredo nem identidade de conta. Um parse permissivo permitiria ao gateway vazar
identidade de conta para dentro do daemon sem ninguem perceber. A validacao e a fronteira.

## 4. BLOQUEADOR QUE NENHUMA VALIDACAO RESOLVE - precisa de decisao

`ModelDocument` (`health_models.go:24-38`) exige, e `buildSnapshot` (`registry.go:235-240`) rejeita
se faltar: `protocol`, `streaming`, `tools`, `reasoning`, `structured_output`, `context_limit > 0`,
`account_pool` nao vazio, `rotation`, `affinity`, `available`.

O `/v1/models` medido entrega: `id`, `object`, `created`, `owned_by`, `permission`, `root`,
`parent`, `context_length`, `max_input_tokens`, `max_output_tokens`,
`capabilities{tool_calling, reasoning, thinking, temperature}`.

FALTAM: `protocol`, `streaming`, `structured_output`, `account_pool`, `rotation`, `affinity`,
`available`, `registry_version`. `context_limit` existe com outro nome (`context_length`).

Consequencia: com o schema atual do gateway, o caminho enriquecido NAO passa, por mais validacao
que se escreva. Ha duas saidas e as duas exigem decisao, nao codigo meu:
- **A (aditiva, preferida):** OmniRoute passa a servir o schema enriquecido em `/v1/models` ou em
  um endpoint proprio de registry. O PACOTE 2 fica exatamente como proposto acima e nao precisa de
  projecao nenhuma.
- **B:** definir no PACOTE 2 uma projecao explicita e nao-dev, com defaults assinados pelo owner
  para os 8 campos ausentes.
Recomendo A. Em B, defaults inventados no cliente sao fail-open disfarcado: o daemon passaria a
afirmar capacidade que o gateway nunca declarou, e a readiness estrita perde sentido.
NAO escolhi entre A e B: e decisao do owner, com voto do Codex56-TL.

## 5. Testes propostos (a escrever depois da assinatura)
- `TestFetchModelsCatalogRejectsNonJSONContentType`
- `TestFetchModelsCatalogAcceptsModelCatalogVersionHeaderFallback`
- `TestFetchModelsCatalogFailsClosedWhenBothVersionHeadersAbsent`
- `TestFetchModelsCatalogRejectsEmptyCatalogWithDetailToken`
- `TestFetchModelsCatalogRejectsCountAboveMaxRegistryModels`
- `TestFetchModelsCatalogHonoursTwentySecondCap`
- `TestFetchModelsCatalog401DoesNotServeStaleSnapshot`
- `TestFetchModelsCatalog403UsesTerminalBackoff`
- `TestFetchModelsCatalogRejectsAccountIdentityShapedPool`
- `TestFetchModelsDelegatesToCatalogAndPreservesSignature`

## 6. Nao-afirmacoes
- Nao editei `client.go` nem qualquer `.go`. Nao rodei `go build` nem `go test`. Nao substitui
  binario, nao reiniciei daemon, nao commitei, nao dei push.
- Nao li valor de segredo: o `Authorization` foi montado dentro do shell remoto a partir do
  secret-file e o valor nunca entrou no meu contexto nem em log.
- Nao validei o comportamento de 403 empiricamente: o endpoint medido devolve 401. A semantica de
  403 acima vem de `classification.go:107-108`, nao de observacao.
- Nao toquei nos pacotes dos outros agentes: escrevi apenas este arquivo de evidencia.
