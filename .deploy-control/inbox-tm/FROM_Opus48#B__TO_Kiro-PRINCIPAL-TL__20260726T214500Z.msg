FROM: Opus48#B (ORQ2, pane w6:p2)
TO: KIRO-PRINCIPAL-TL (100.117.245.15, pane wB:p1)
UTC: 2026-07-26T21:45:00Z
SUBJECT: Como o daemon lista modelos de um CLI - codex vs kiro vs agy. CORRECAO DE PREMISSA.

## CORRECAO DE PREMISSA, LEIA PRIMEIRO
agentBrainBuiltInCLIFor / trustedAdapterEntries (config.go:144-155) NAO participam da lista de
modelos. Aquele mapa de 3 CLIKinds e o caminho de ADMISSAO de execucao do agent-brain.
A lista de modelos e outro caminho, em pkg/agent/models.go, e ele TEM caso proprio para kiro
(linha 194) e para antigravity (linha 151). Nao falta mapeamento. Nao e preciso alterar
config.go para popular o dropdown de kiro e agy.

## O MECANISMO
Entrada HTTP: cmd/server/router.go:937  r.Post("/models", h.InitiateListModels)
             internal/handler/runtime_models.go:288  func (h *Handler) InitiateListModels
Executor no daemon: internal/daemon/daemon.go:2058
    models, err := agent.ListModels(discoveryCtx, rt.Provider, entry.Path)
    if err != nil {
        if discoveryCtx.Err() != nil {
            err = fmt.Errorf("model discovery exceeded the 40 second daemon limit: %w", discoveryCtx.Err())
        }
        d.reportModelListResult(ctx, rt, requestID, map[string]any{
            "status": "failed",
            "error":  err.Error(),
        })
        return
    }
Dispatch por provider: pkg/agent/models.go:120
    func ListModels(ctx context.Context, providerType, executablePath string) ([]Model, error) {

## CODEX - por que SEMPRE popula (nao executa CLI nenhum)
pkg/agent/models.go:129-134, literal:
    case "codex":
        models := codexStaticModels()
        if requireDiscoveryProcessContainment() == nil {
            annotateCodexThinking(ctx, models, discoveryExecutablePath)
        }
        return models, nil
codexStaticModels() esta em pkg/agent/models.go:510 e e um catalogo ESTATICO compilado no binario.
COMANDO EXECUTADO NO CLI: NENHUM. Nao existe `codex --list-models` nem `codex app-server` nesse
caminho. O unico subprocesso opcional e annotateCodexThinking, e ele so enriquece thinking levels;
se falhar, o catalogo estatico continua retornando. Por isso o dropdown do codex nunca fica vazio.
Mesmo padrao para claude (models.go:123-128, claudeStaticModels) e gemini (models.go:147-150).

## KIRO - equivalente
pkg/agent/models.go:194-205, literal:
    case "kiro":
        if err := requireDiscoveryProcessContainment(); err != nil {
            return nil, err
        }
        return cachedDiscovery(ctx, discoveryCacheKey(providerType, discoveryExecutablePath), func() ([]Model, error) {
            models, err := discoverKiroModels(ctx, discoveryExecutablePath)
            if err != nil {
                return nil, err
            }
            annotateKiroThinking(models)
            return models, nil
        })
pkg/agent/models.go:1043-1049, literal:
    func discoverKiroModels(ctx context.Context, executablePath string) ([]Model, error) {
        return discoverACPModels(ctx, executablePath, acpDiscoveryProvider{
            defaultBin:   "kiro-cli",
            clientName:   "multica-model-discovery",
            tmpdirPrefix: "multica-kiro-discovery-",
        })
    }
pkg/agent/models.go:1134-1137, literal (argv default, porque kiro nao define acpArgs):
    cmdArgs := p.acpArgs
    if len(cmdArgs) == 0 {
        cmdArgs = []string{"acp"}
    }
COMANDO EXECUTADO NO CLI: `kiro-cli acp`
Protocolo: ACP por stdio (JSON-RPC), handshake `initialize` + `session/new`, e o catalogo e lido
do proprio payload de session/new em models.availableModels / models.currentModelId
(models.go:1119, 1307-1318). Timeout 15s (models.go:1131). NAO ha fallback estatico.

## ANTIGRAVITY (agy) - equivalente
pkg/agent/models.go:151-160 chama discoverAntigravityModels.
pkg/agent/models.go:1384-1411, literal do trecho decisivo:
    if executablePath == "" {
        executablePath = "agy"
    }
    if _, err := exec.LookPath(executablePath); err != nil {
        return nil, nil
    }
    runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    cmd := exec.CommandContext(runCtx, executablePath, "models")
    hideAgentWindow(cmd)
    out, err := cmd.Output()
    ...
    models := parseAntigravityModels(string(out))
    if len(models) == 0 {
        return nil, fmt.Errorf("antigravity model discovery returned no models")
    }
COMANDO EXECUTADO NO CLI: `agy models` (models.go:1398). Uma linha por nome de modelo; o proprio
display string E o valor que `--model` espera (models.go:1414-1416). Timeout 30s. Sem fallback
estatico: 0 modelos vira ERRO.

## CAUSA REAL DO DROPDOWN VAZIO - EVIDENCIA AO VIVO NO ORQ1
Executei os dois comandos exatos, read-only, no ORQ1:
  $ /home/ec2-user/.local/bin/agy models
  Error: Please sign in to view available models. Launch the CLI without arguments to sign in.

  $ printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{}}}' | /home/ec2-user/.local/bin/kiro-cli acp
  error: You are not logged in, please log in with kiro-cli login

Os dois CLIs NAO ESTAO AUTENTICADOS na conta do daemon (HOME=/home/ec2-user no ORQ1).
`agy models` retorna erro -> discoverAntigravityModels retorna erro -> ListModels retorna erro ->
daemon.go:2058-2067 reporta status=failed -> dropdown vazio.
`kiro-cli acp` recusa o handshake -> discoverACPModels falha -> mesmo caminho.
O codex nao depende de login porque nunca chama o CLI.

Containment NAO e o problema: pkg/agent/proc_other.go:21 `func requireDiscoveryProcessContainment()
error { return nil }` com build tag `//go:build ... || linux || ...`. Em Linux o gate passa.

## CONCLUSAO
O dropdown de kiro e agy nao popula por FALTA DE LOGIN nos CLIs, nao por falta de mapeamento no
codigo. Nenhuma alteracao de codigo-fonte e necessaria para esse sintoma especifico.
Nao editei, nao recompilei, nao substitui binario, nao reiniciei nada.

## BLOQUEIO
Autenticar kiro-cli e agy no ORQ1 e mudanca de credencial/auth = STOP-AND-WAIT (AUTHORITY-AMENDMENT-001
§0.1). NAO executei e nao vou executar sem decisao escrita do owner.
DONO DO BLOQUEIO: human owner via KIRO-PRINCIPAL-TL.
PROXIMA ACAO NECESSARIA: ruling escrito autorizando `kiro-cli login` e o sign-in do `agy` sob o HOME
do daemon no ORQ1. Depois disso o dropdown popula sem tocar em Go.
OBSERVACAO SEPARADA: se o objetivo final for os 6 CLIs COM --agent-brain-gateway-required ligado,
ai sim config.go:144-155 precisa mudar. Sao dois problemas distintos; este aqui nao exige codigo.
