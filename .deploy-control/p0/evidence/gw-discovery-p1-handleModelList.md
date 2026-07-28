# PACOTE 1 - p1-handleModelList - patch PROPOSTO, NAO APLICADO

- agente: Opus48#A (kiro) - ORQ2 - pane w6:p1
- data: 2026-07-26T22:47Z
- arquivo alvo: `multica-auth-work/server/internal/daemon/daemon.go`, func `handleModelList`
- estado: **nenhum arquivo de codigo editado, nada recompilado, nada substituido, daemon intocado**
- assinatura pendente: Codex56-TL

---

## 1. TRECHO LITERAL ATUAL - daemon.go:2039-2068

```go
2039 // handleModelList resolves the provider's supported models (via static
2040 // catalog or by shelling out to the agent CLI) and reports the result
2041 // back to the server. Discovery is bounded independently from the daemon
2042 // lifetime, and failures are reported explicitly so the UI can stop polling
2043 // and show a useful error instead of rendering a misleading empty catalog.
2044 func (d *Daemon) handleModelList(ctx context.Context, rt Runtime, requestID string) {
2045 	d.logger.Info("model list requested", "runtime_id", rt.ID, "request_id", requestID, "provider", rt.Provider)
2046 	discoveryCtx, cancelDiscovery := context.WithTimeout(ctx, 40*time.Second)
2047 	defer cancelDiscovery()
2048
2049 	entry, ok := d.cfg.Agents[rt.Provider]
2050 	if !ok {
2051 		d.reportModelListResult(ctx, rt, requestID, map[string]any{
2052 			"status": "failed",
2053 			"error":  fmt.Sprintf("no agent configured for provider %q", rt.Provider),
2054 		})
2055 		return
2056 	}
2057
2058 	models, err := agent.ListModels(discoveryCtx, rt.Provider, entry.Path)
2059 	if err != nil {
2060 		if discoveryCtx.Err() != nil {
2061 			err = fmt.Errorf("model discovery exceeded the 40 second daemon limit: %w", discoveryCtx.Err())
2062 		}
2063 		d.reportModelListResult(ctx, rt, requestID, map[string]any{
2064 			"status": "failed",
2065 			"error":  err.Error(),
2066 		})
2067 		return
2068 	}
```

O resto (2070-2119) monta `modelWire{ID, Label, Provider, Default, Thinking}` e reporta
`{"status":"completed","models":wire,"supported":agent.ModelSelectionSupported(rt.Provider)}`.

## 2. COMO DECIDIR: Agent Brain ou legado

Nao inventar campo novo. Os tres insumos ja existem:

1. `d.agentBrainGatewayRequired()` - daemon.go:970-973, literal:
```go
970 func (d *Daemon) agentBrainGatewayRequired() bool {
971 	return d != nil && d.cfg.AgentBrain.DevelopmentEnabled &&
972 		d.cfg.AgentBrain.Neutral.Gateway.Required
973 }
```
2. `d.agentBrain != nil && d.agentBrainInitErr == nil` - daemon.go:244-245, 299. Precedente do
   proprio codigo em daemon.go:3337.
3. `brain.LegacyProviderCLIKind(rt.Provider)` - brain/compatibility.go:116-133, o mapeamento
   congelado provider->CLIKind: claude, codex, kimi, antigravity/agy, nim, cline/opencode.

PATCH A - ADICIONAR apos daemon.go:973:
```go
+// runtimeUsesGatewayDiscovery decides the catalog source for a runtime. The
+// gateway owns the model registry whenever the Agent Brain slice is active and
+// the runtime's persisted provider has a frozen CLI mapping; every other
+// runtime keeps the legacy native-CLI discovery path unchanged. It never reads
+// a credential, an account pool, or an executable path.
+func (d *Daemon) runtimeUsesGatewayDiscovery(rt Runtime) bool {
+	if !d.agentBrainGatewayRequired() || d.agentBrain == nil || d.agentBrainInitErr != nil {
+		return false
+	}
+	if _, err := brain.LegacyProviderCLIKind(rt.Provider); err != nil {
+		return false
+	}
+	return true
+}
```

QUESTAO DE DESENHO PARA O CODEX56-TL ASSINAR, com consequencia oposta:
- variante PERMISSIVA (acima): qualquer provider com mapeamento congelado usa o gateway.
- variante ESTRITA: exigir tambem `kind == d.cfg.AgentBrain.CLIKind`, espelhando
  brain_integration.go:314 (`translation.Task.Request.CLIKind != r.config.CLIKind` -> rejeita).
RECOMENDO A PERMISSIVA: a estrita reproduz exatamente o colapso de UM runtime que
config.go:425-431 causa hoje, ou seja so um runtime popularia o dropdown.
FATO QUE AFETA AS DUAS: `kiro` NAO existe em LegacyProviderCLIKind (compatibility.go:117-132),
entao com qualquer variante o kiro cai no ramo LEGADO e continua sem catalogo. Incluir kiro exige
`case "kiro": return CLIKiro, nil` ali E a constante CLIKiro em identity.go - e o mesmo bloqueio
que reportei no pacote anterior.

## 3. PATCH B - novo metodo em brain_integration.go (o daemon NAO constroi cliente)

Justificativa: hoje o `gateway.Client` e criado dentro de `admitTask`, brain_integration.go:325,
junto de `r.dependencies.CredentialSource`. Se `handleModelList` construisse o cliente, daemon.go
passaria a tocar credencial - exatamente o que o principio proibe. Entao o segredo continua so em
`agentBrainRuntime`, e daemon.go recebe apenas documentos de modelo.

```go
+// fetchGatewayModels returns the OmniRoute registry projection for the model
+// dropdown. It is the only model-discovery path that touches the gateway, and
+// it returns no credential, no account pool, and no account identity.
+func (r *agentBrainRuntime) fetchGatewayModels(ctx context.Context, correlation brain.Correlation) ([]gateway.ModelDocument, error) {
+	if r == nil || r.dependencies.CredentialSource == nil {
+		return nil, &agentBrainAdmissionError{class: "credential_source_unavailable"}
+	}
+	client, err := gateway.NewClient(gateway.ClientOptions{
+		Gateway:    r.config.Neutral.Gateway,
+		Endpoints:  gateway.EndpointSet{Liveness: "/api/health/ping", Readiness: "/v1/models"},
+		Credential: r.dependencies.CredentialSource, HTTPClient: r.dependencies.HTTPClient,
+	})
+	if err != nil {
+		return nil, &agentBrainAdmissionError{class: "gateway_client_invalid"}
+	}
+	document, err := client.FetchModels(ctx, correlation)
+	if err != nil {
+		return nil, err
+	}
+	return document.Models, nil
+}
```
Assinatura verificada: `func (c *Client) FetchModels(ctx context.Context, correlation brain.Correlation) (ModelsDocument, error)`
- gateway/client.go:178. `ModelsDocument{Object, RegistryVersion, Models []ModelDocument}` -
gateway/health_models.go:18-22. Construcao do cliente copiada literalmente de
brain_integration.go:325-329, sem alterar a forma.

## 4. PATCH C - handleModelList: SUBSTITUIR a linha 2058

ANTES (2058):
```go
	models, err := agent.ListModels(discoveryCtx, rt.Provider, entry.Path)
```
DEPOIS:
```go
+	var models []agent.Model
+	var err error
+	if d.runtimeUsesGatewayDiscovery(rt) {
+		// The gateway owns the registry: credential selection and account
+		// choice never reach the daemon. Bound this leg tighter than the
+		// legacy 40s CLI budget - it is one authenticated GET, not a process
+		// spawn.
+		gatewayCtx, cancelGateway := context.WithTimeout(discoveryCtx, 10*time.Second)
+		defer cancelGateway()
+		var documents []gateway.ModelDocument
+		documents, err = d.agentBrain.fetchGatewayModels(gatewayCtx, brain.AdmissionCorrelation(requestID, rt.ID))
+		if err == nil {
+			models = gatewayModelsToAgentModels(documents, rt.Provider, rt.Model)
+		}
+	} else {
+		models, err = agent.ListModels(discoveryCtx, rt.Provider, entry.Path)
+	}
	if err != nil {
```
`entry` (2049) continua obrigatorio: no ramo gateway ele nao e usado como caminho de executavel,
mas a checagem 2049-2056 permanece o guarda de "provider nao configurado". Correlacao:
`brain.AdmissionCorrelation(taskID, sessionSeed)` - brain/admission_observability.go:83; aqui nao
existe task, entao o par natural e (requestID, rt.ID).

## 5. PATCH D - projecao, e AQUI esta o guarda de isolamento

```go
+// gatewayModelsToAgentModels projects the OmniRoute registry into the UI
+// catalog. It DELIBERATELY drops AccountPool, Rotation, Affinity and Fallback:
+// those are gateway-owned account-selection facts and must never reach the
+// daemon wire, the server, or the browser. Only route ID and capability
+// survive.
+func gatewayModelsToAgentModels(documents []gateway.ModelDocument, provider, currentModel string) []agent.Model {
+	models := make([]agent.Model, 0, len(documents))
+	for _, doc := range documents {
+		if doc.ID == "" {
+			continue
+		}
+		if doc.Available != nil && !*doc.Available {
+			continue
+		}
+		models = append(models, agent.Model{
+			ID:       doc.ID,
+			Label:    doc.ID,
+			Provider: provider,
+			Default:  doc.ID == currentModel,
+		})
+	}
+	return models
+}
```
`ModelDocument` tem AccountPool, Rotation, Affinity, Fallback (health_models.go:32-35). Nenhum
deles e copiado. `Thinking` fica nil: o gateway nao publica niveis de thinking, e inventar um
nivel seria pior que nao ter. `Label = ID` porque o registro nao traz nome de exibicao; se o
OmniRoute passar a publicar um, e um campo, nao uma reescrita.

## 6. O QUE ESTE PATCH NAO RESOLVE, declarado

- `kiro` continua sem catalogo: falta o mapeamento em LegacyProviderCLIKind e a constante CLIKind.
- Sob a variante estrita, so o CLIKind configurado popula.
- RISCO ADJACENTE, fora do meu pacote: `agent.ModelKnownIncompatibleWithProvider`
  (pkg/agent/models.go:273-283) compara o modelo salvo com o catalogo estatico
  (`acceptedModelIDsForProvider`). IDs de rota do OmniRoute, como
  `claude_code_kimi_2.7_Code`, nao estao nesse catalogo, entao o servidor pode classificar o
  modelo escolhido como incompativel e apagar o valor. Quem tocar esse arquivo precisa saber.
  `agent.ModelSelectionSupported` (models.go:263-265) retorna `true` sempre, entao a linha 2118
  NAO precisa mudar - verificado.
- Nao ha teste proposto aqui: cabe ao pacote de testes, e eu nao rodei `go build` nem `go vet`
  nesta rodada, por ordem.

## 7. NADA MUTADO

Nenhum arquivo de codigo editado (`git status --porcelain multica-auth-work/server` vazio), sem
build, sem vet, sem binario, sem restart de daemon ou container, sem commit, push, reset, clean
ou checkout --. Nenhum valor de segredo lido ou transcrito.
