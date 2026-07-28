FROM: Opus48#B (ORQ2, pane w6:p2)
TO: KIRO-PRINCIPAL-TL (100.117.245.15, pane wB:p1)
UTC: 2026-07-26T22:52:00Z
SUBJECT: PACOTE 2 p2-gateway-client - proposta pronta + 3 fatos medidos que mudam o desenho

ARTEFATO: .deploy-control/p0/evidence/gw-discovery-p2-gateway-client.md
MODO: proposta. Nao editei .go, nao compilei, nao substitui binario, nao reiniciei daemon,
nao commitei. Aguarda assinatura do Codex56-TL e decisao do owner.

## 3 FATOS MEDIDOS AGORA CONTRA O GATEWAY (read-only, ORQ1)
1. NAO existe header X-OmniRoute-Registry-Version. O gateway manda
   `x-model-catalog-version: model-metadata-v1:static`. Logo client.go:203-209, que le somente
   HeaderRegistryVersion (health_models.go:11), e CODIGO MORTO na producao atual, e
   document.RegistryVersion fica vazio -> buildSnapshot (registry.go:222) rejeita tudo.
2. Sem credencial o gateway responde 401 AUTH_002, NAO 403. Corpo:
   {"error":{"code":"AUTH_002","message":"Authentication required","correlation_id":"..."}}
3. GET /v1/models = 200, 110807 bytes, 327 modelos, Transfer-Encoding chunked.
   Item real: id, object, created, owned_by, permission, root, parent, context_length,
   max_input_tokens, max_output_tokens, capabilities{tool_calling,reasoning,thinking,temperature}.

## ASSINATURA ATUAL (client.go:178)
func (c *Client) FetchModels(ctx context.Context, correlation brain.Correlation) (ModelsDocument, error)

## ASSINATURA PROPOSTA (aditiva, zero call site alterado)
type ModelsCatalog struct {
    Document ModelsDocument; CatalogVersion string; RequestID string; FetchedAt time.Time; ModelCount int
}
func (c *Client) FetchModelsCatalog(ctx context.Context, correlation brain.Correlation) (ModelsCatalog, error)
func (c *Client) FetchModels(...) (ModelsDocument, error)  // delega, preserva ModelsFetcher
Aditiva porque ModelsFetcher (health_models.go:51) e registry.go:118 chamam FetchModels(ctx) com
aridade diferente. Trocar o retorno quebraria os dois.

## VALIDACOES PROPOSTAS
V1 timeout: teto proprio catalogFetchTimeout=20s, min com c.requestTimeout (30s, client.go:20),
   dentro do orcamento de 40s do daemon (daemon.go:2060).
V2 corpo: MANTER readBounded/2 MiB (client.go:21,184). Medido 110KB = 5,3%; pior caso ~350KB com
   MaxRegistryModels=1024. Resposta e chunked, sem Content-Length: o bound e a unica protecao.
V3 Content-Type estrito application/json via mime.ParseMediaType. Hoje nao existe checagem e um
   HTML de portal cativo com 200 chega ao json.Unmarshal.
V4 versao: aceitar X-OmniRoute-Registry-Version com precedencia e cair para X-Model-Catalog-Version;
   validar token (<=128, sem CR/LF/NUL); divergencia header vs corpo -> ErrorProtocol; AUSENCIA DOS
   DOIS -> FALHA. Nunca sintetizar versao: sem proveniencia nao ha ancora de retencao 24h nem de custo.
V5 catalogo vazio com 200 -> ErrorProtocol, Retryable=false, Detail="empty_catalog". Sem fallback
   para catalogo estatico nem para probe de CLI, que e justamente o que este pacote remove.
V6 contagem > MaxRegistryModels (1024, registry.go:29) -> rejeita no cliente.
V7 401/403: FetchModels nunca ve o corpo, c.do (client.go:272-274) ja classifica.
   401 -> ErrorAuthentication CircuitAccount, retry so se AuthOutcomeAccessExpired.
   403 -> ErrorAuthorization CircuitAccount Retryable=false (classification.go:107-108).
   PROPOSTA: RegistryRefreshFailureBackoff=2s (registry.go:24) e agressivo demais para authz
   terminal - martela o gateway a cada 2s. Criar catalogAuthzBackoff=60s so para ErrorAuthorization.
   Em 401/403 NUNCA servir snapshot expirado e NUNCA cair para CLI. Corpo nunca logado.
V8 zero identidade de conta: AccountPool validado como rotulo OPACO e rejeicao de campo fora do
   allow-list de ModelDocumentMeta, ou account_pool com forma de e-mail/token. A fronteira de
   validacao e o que garante que o daemon nao aprenda identidade de conta.

## BLOQUEADOR QUE VALIDACAO NENHUMA RESOLVE
buildSnapshot (registry.go:235-240) exige protocol, streaming, tools, reasoning, structured_output,
context_limit>0, account_pool, rotation, affinity, available. O gateway NAO entrega 8 desses:
protocol, streaming, structured_output, account_pool, rotation, affinity, available,
registry_version. context_limit vem com outro nome (context_length).
OPCAO A (aditiva, RECOMENDO): OmniRoute passa a servir o schema enriquecido em /v1/models ou em
endpoint proprio de registry. O PACOTE 2 fica exatamente como proposto, sem projecao.
OPCAO B: projecao explicita nao-dev no PACOTE 2 com defaults assinados pelo owner para os 8 campos.
Recomendo A. Em B, default inventado no cliente e fail-open disfarcado: o daemon afirmaria
capacidade que o gateway nunca declarou e a readiness estrita perde sentido.
NAO escolhi. E decisao do owner com voto do Codex56-TL.

## BLOQUEIO
Escrever o codigo depende de (1) assinatura do Codex56-TL no split e (2) ruling do owner entre A e B.
Dono: human owner via KIRO-PRINCIPAL-TL. Proxima acao: assinatura + escolha A ou B.

## NAO-AFIRMACOES
Nao editei .go, nao rodei build/test, nao substitui binario, nao reiniciei daemon, nao commitei.
Nao li valor de segredo: o Authorization foi montado no shell remoto a partir do secret-file e o
valor nunca entrou no meu contexto. Nao observei 403 empiricamente - o endpoint devolve 401; a
semantica de 403 vem de classification.go:107-108. Nao toquei nos pacotes dos outros agentes.
