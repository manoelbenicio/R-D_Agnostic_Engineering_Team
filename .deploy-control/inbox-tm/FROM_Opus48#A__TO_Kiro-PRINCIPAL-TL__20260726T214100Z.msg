FROM: Opus48#A (kiro) - ORQ2 - pane w6:p1
TO: KIRO-PRINCIPAL-TL - LOCAL - dataops-lab@100.117.245.15 - pane wB:p1
UTC: 2026-07-26T21:41:00Z
SUBJECT: PATCH PROPOSTO (NAO aplicado) - agentBrainBuiltInCLIFor + trustedAdapterEntries para kiro e antigravity

NENHUM ARQUIVO FOI EDITADO. Nada recompilado, nada substituido, daemon intocado. Isto e texto de patch.

=================================================================
1. TRECHO LITERAL - internal/daemon/config.go linhas 141-155
=================================================================
141 // agentBrainBuiltInCLIFor is the immutable gateway-mode executable mapping.
142 // It deliberately contains no workspace profile, provider-account, or
143 // operator-supplied command path input.
144 func agentBrainBuiltInCLIFor(kind brain.CLIKind) (agentBrainBuiltInCLI, error) {
145 	switch kind {
146 	case brain.CLIClaudeCode:
147 		return agentBrainBuiltInCLI{Provider: "claude", Command: "claude"}, nil
148 	case brain.CLICodex:
149 		return agentBrainBuiltInCLI{Provider: "codex", Command: "codex"}, nil
150 	case brain.CLIOpenAICompatible:
151 		return agentBrainBuiltInCLI{Provider: "cline", Command: "cline"}, nil
152 	default:
153 		return agentBrainBuiltInCLI{}, fmt.Errorf("agent brain CLI has no accepted built-in executable mapping")
154 	}
155 }

=================================================================
2. TRECHO LITERAL - internal/daemon/runtimeenv/env.go linhas 302-350
=================================================================
302 func trustedAdapterEntries(profile AdapterEnvironment) (map[string]environmentEntry, string, string, error) {
303 	if !profile.StableSecret.IsSet() {
304 		return nil, "", "", fmt.Errorf("stable OmniRoute secret is required")
305 	}
306 	root, err := normalizeGatewayRoot(profile.GatewayRoot)
307 	if err != nil {
308 		return nil, "", "", err
309 	}
310 	if err := validatePhysicalControlledDirectory(profile.TaskHome, "task home"); err != nil {
311 		return nil, "", "", err
312 	}
313 	entries := map[string]environmentEntry{
314 		"HOME": {key: "HOME", value: profile.TaskHome, origin: originTrustedLocal},
315 	}
316 	switch profile.CLI {
317 	case brain.CLIClaudeCode:
318 		entries["ANTHROPIC_BASE_URL"] = environmentEntry{key: "ANTHROPIC_BASE_URL", value: root, origin: originTrustedGateway}
319 		entries["ANTHROPIC_AUTH_TOKEN"] = environmentEntry{key: "ANTHROPIC_AUTH_TOKEN", value: profile.StableSecret.value, origin: originTrustedSecret}
320 		telemetry, err := trustedTelemetryEnv(profile)
321 		if err != nil {
322 			return nil, "", "", err
323 		}
324 		for k, v := range telemetry {
325 			entries[k] = environmentEntry{key: k, value: v, origin: originTrustedLocal}
326 		}
327 		return entries, root, "ANTHROPIC_AUTH_TOKEN", nil
328 	case brain.CLICodex:
329 		if err := validatePhysicalControlledDirectory(profile.CodexHome, "Codex home"); err != nil {
330 			return nil, "", "", err
331 		}
332 		entries["CODEX_HOME"] = environmentEntry{key: "CODEX_HOME", value: profile.CodexHome, origin: originTrustedLocal}
333 		entries[CodexOmniRouteAPIKeyEnv] = environmentEntry{key: CodexOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
334 		return entries, root, CodexOmniRouteAPIKeyEnv, nil
335 	case brain.CLIOpenAICompatible:
336 		// Accepted OmniRoute OpenAI-compatible (Cline) route. The controlled
337 		// per-task Cline data dir carries providers.json; the stable OmniRoute
338 		// secret is injected trusted-last as CLINE_OMNIROUTE_API_KEY. Both keys
339 		// are rejected by the deny-list for inherited/local/custom origins and
340 		// are legal only as trusted entries merged after validation.
341 		if err := validatePhysicalControlledDirectory(profile.ClineDataDir, "Cline data dir"); err != nil {
342 			return nil, "", "", err
343 		}
344 		entries["CLINE_DATA_DIR"] = environmentEntry{key: "CLINE_DATA_DIR", value: profile.ClineDataDir, origin: originTrustedLocal}
345 		entries[ClineOmniRouteAPIKeyEnv] = environmentEntry{key: ClineOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
346 		return entries, root, ClineOmniRouteAPIKeyEnv, nil
347 	default:
348 		return nil, "", "", ErrAdapterFailClosed
349 	}
350 }

=================================================================
3. BLOQUEIO 1 - kiro NAO TEM CLIKind. O patch precisa de um TERCEIRO arquivo.
=================================================================
internal/daemon/brain/identity.go:13-28 (literal): as constantes existentes sao
	CLIClaudeCode "claude-code", CLICodex "codex", CLIKimi "kimi",
	CLIOpenAICompatible "openai-compatible", CLIAntigravity "antigravity", CLINIM "nim"
Nao existe CLIKiro. `case brain.CLIKiro` nao compila hoje.
antigravity JA tem constante (CLIAntigravity), kiro NAO.

PATCH A - internal/daemon/brain/identity.go
  no bloco const, APOS a linha 16 (`CLIKimi CLIKind = "kimi"`), ADICIONAR 1 linha:
+	CLIKiro             CLIKind = "kiro"
  no mapa supportedCLIKinds, APOS a linha 25 (`CLIKimi: {},`), ADICIONAR 1 linha:
+	CLIKiro:             {},

=================================================================
4. PATCH B - internal/daemon/config.go, ADICIONAR 4 linhas ENTRE a linha 151 e a 152
=================================================================
+	case brain.CLIKiro:
+		return agentBrainBuiltInCLI{Provider: "kiro", Command: "kiro-cli"}, nil
+	case brain.CLIAntigravity:
+		return agentBrainBuiltInCLI{Provider: "antigravity", Command: "agy"}, nil
Os strings de Provider NAO sao livres: precisam casar exatamente com as chaves do mapa de
deteccao, config.go:401 `agents["kiro"]` (comando kiro-cli) e config.go:414
`agents["antigravity"]` (comando agy). Command casa com o que resolve no PATH do daemon,
verificado: /home/ec2-user/.local/bin/kiro-cli e /home/ec2-user/.local/bin/agy.
Este patch B, com o A, e mecanico e seguro. So ele NAO resolve nada sozinho: sem o item 5
o adapter continua caindo em ErrAdapterFailClosed.

=================================================================
5. BLOQUEIO 2, O GRAVE - o patch de env.go NAO E ESCREVIVEL COM O QUE O REPO SABE
=================================================================
Os tres cases existentes entregam DUAS coisas ao filho: o endpoint (root) e o segredo.
Para kiro e agy o repo nao tem NEM UM dos dois vetores:
5.1 ENDPOINT: as unicas variaveis de base URL que existem no codigo sao
    ANTHROPIC_BASE_URL, OPENAI_BASE_URL, NIM_BASE_URL e MODEL_BASE_URL
    (grep em internal/daemon/runtimeenv/ e pkg/agent/). NAO existe KIRO_BASE_URL nem
    qualquer endpoint de antigravity/gemini. Sem isso, `root` nao chega ao processo filho.
5.2 SEGREDO kiro: existe KIRO_API_KEY, e ele esta no deny-list
    (runtimeenv/policy.go:41 "KIRO_API_KEY": DenyProviderCredential, e prefixo "KIRO_" em
    policy.go:73). Isso NAO impede: e o mesmo padrao do cline, cujo CLINE_ tambem esta no
    prefixo negado e funciona porque entra como origin originTrustedSecret. Precedente OK.
    Mas execenv/execenv.go:70 diz "Kiro: XDG_DATA_HOME / KIRO_API_KEY" e kiro_home.go:16
    diz que Kiro e fork do Amazon Q e IGNORA KIRO_HOME, usando XDG_DATA_HOME. Logo o case
    precisaria de um campo novo em AdapterEnvironment (env.go:108-124 hoje so tem CLI,
    GatewayRoot, TaskHome, CodexHome, ClineDataDir, StableSecret, 3 de telemetria).
5.3 SEGREDO antigravity: NAO EXISTE variavel de API key. A credencial e um diretorio de
    token OAuth, execenv/antigravity_home.go:10
    `const antigravityCredentialRelDir = ".gemini/antigravity-cli"`, isolado por HOME.
    StableSecret nao tem vetor de entrega nenhum para o agy.
CONSEQUENCIA: se eu escrevesse `case brain.CLIAntigravity:` retornando entries+root, a
funcao passaria a devolver sucesso sem realmente rotear o agy para o OmniRoute. Seria
fail-OPEN vestido de fail-closed, pior que o ErrAdapterFailClosed atual. NAO escrevo isso.

PATCH C - PROPOSTO MAS INCOMPLETO POR FALTA DE FATO, precisa de decisao/insumo antes:
  internal/daemon/runtimeenv/env.go, ADICIONAR entre a linha 346 e a 347:
+	case brain.CLIKiro:
+		if err := validatePhysicalControlledDirectory(profile.KiroDataHome, "Kiro data home"); err != nil {
+			return nil, "", "", err
+		}
+		entries["XDG_DATA_HOME"] = environmentEntry{key: "XDG_DATA_HOME", value: profile.KiroDataHome, origin: originTrustedLocal}
+		entries[KiroOmniRouteAPIKeyEnv] = environmentEntry{key: KiroOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
+		entries[<KIRO_BASE_URL_ENV_DESCONHECIDO>] = environmentEntry{key: <idem>, value: root, origin: originTrustedGateway}
+		return entries, root, KiroOmniRouteAPIKeyEnv, nil
  e ainda: novo campo `KiroDataHome string` em AdapterEnvironment (apos env.go:113) e nova
  constante `KiroOmniRouteAPIKeyEnv = "KIRO_API_KEY"` no estilo de cline.go:38.
  O placeholder <KIRO_BASE_URL_ENV_DESCONHECIDO> e exatamente o fato que falta: nao esta no
  repo. Sem ele o patch C nao deve ser aplicado.
  Para antigravity NAO proponho case algum: falta vetor de segredo E de endpoint.

=================================================================
6. RESSALVA QUE MUDA A PRIORIDADE - o dropdown pode nao ser este codigo
=================================================================
O daemon vivo (PID 3244391) roda SEM --agent-brain-gateway-required, entao
trustedAdapterEntries/ErrAdapterFailClosed NAO estao no caminho de execucao agora. O
dropdown de modelos vem de outra funcao: pkg/agent/models.go:1043
`func discoverKiroModels(ctx, executablePath)` -> `discoverACPModels` com
defaultBin "kiro-cli", tmpdirPrefix "multica-kiro-discovery-". Diferente do cline
(models.go:1035-1038, que faz fallback para clineStaticModels() quando a descoberta
falha ou vem vazia) e do copilot (comentario em 1057-1059: "Falls back to
copilotStaticModels() ... so the UI dropdown always has something to show"), o kiro
NAO tem fallback estatico: retorna direto o resultado da descoberta ACP. Se `kiro-cli acp`
falhar por auth, o dropdown fica VAZIO. E em pkg/agent/antigravity.go nao existe nenhuma
func de listagem de modelos (grep "func.*Models" = vazio).
LEITURA: o sintoma que o owner ve para kiro/antigravity e compativel com descoberta de
modelos sem fallback + auth ausente, nao com o gate do adapter. Patch A+B+C nao promete
popular o dropdown. Isso e ressalva, nao rediagnostico: nao investiguei alem do que esta
citado aqui. Se quiser, meco `kiro-cli acp` e `agy models` read-only e volto com o erro
literal, sem tocar em codigo.

=================================================================
7. NADA MUTADO
=================================================================
Nao editei arquivo, nao rodei go build nem go vet, nao substitui binario, nao reiniciei
daemon, nenhum commit, push, reset, clean, checkout --, nenhuma instalacao. Somente leitura
de fonte no ORQ2. Nenhum valor de segredo lido ou transcrito.
