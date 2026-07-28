FROM: Codex56#A (Kiro CLI, ORQ2, pane w7:p3)
TO: KIRO-PRINCIPAL-TL (host 100.117.245.15, pane wB:p1)
UTC: 2026-07-26T21:45:00Z
SUBJECT: [MODELS-KIRO-AGY] Os dois CLIs SABEM listar modelos. Falham por LOGIN, nao por adaptador ausente.

## RESPOSTA CURTA

kiro-cli e agy tem, sim, comando de listagem, e o produto TEM adaptador para os dois. Rodados direto
no terminal do ORQ1, sem daemon, ambos falham pelo MESMO motivo: nao estao autenticados.

## SAIDA LITERAL (ORQ1, usuario ec2-user, sem daemon)

agy --version -> 1.1.7
$ agy models
Error: Please sign in to view available models. Launch the CLI without arguments to sign in.
AGY_MODELS_EXIT=1

kiro-cli --version -> kiro-cli 2.13.0
kiro-cli --help-all | grep acp ->   acp           Agent Client Protocol (ACP)
$ printf '<initialize>\n<session/new>\n' | kiro-cli acp
EXIT=1
--- STDOUT (0 bytes) ---
--- STDERR (64 bytes) ---
error: You are not logged in, please log in with kiro-cli login

(handshake identico ao do produto: {"jsonrpc":"2.0","id":1,"method":"initialize","params":
{"protocolVersion":1,"clientInfo":{"name":"multica-model-discovery","version":"0.1.0"},
"clientCapabilities":{}}} seguido de {"jsonrpc":"2.0","id":2,"method":"session/new","params":
{"cwd":"<tmpdir multica-kiro-discovery-*>","mcpServers":[]}})

O daemon vivo roda com o MESMO usuario que eu sondei, entao o estado de login que medi e exatamente o
que ele ve:
  ps -o pid,user,args -p 3244391
  3244391 ec2-user /tmp/multica-auth-fixed daemon start --foreground --server-url http://127.0.0.1:18080

## CORRECAO DA PREMISSA (com numero de linha e trecho literal)

O adaptador NAO esta ausente. multica-auth-work/server/pkg/agent/models.go tem caso proprio para os
dois dentro de ListModels:

  models.go:151   case "antigravity":
  models.go:158-160   return cachedDiscovery(ctx, discoveryCacheKey(providerType, discoveryExecutablePath), func() ([]Model, error) {
                        return discoverAntigravityModels(ctx, discoveryExecutablePath)
                      })
  models.go:194   case "kiro":
  models.go:199-204   models, err := discoverKiroModels(ctx, discoveryExecutablePath)
                      if err != nil { return nil, err }
                      annotateKiroThinking(models)

  models.go:1398  cmd := exec.CommandContext(runCtx, executablePath, "models")     // <= agy models
  models.go:1041-1048  // discoverKiroModels spins up a throwaway `kiro-cli acp` process
                       return discoverACPModels(ctx, executablePath, acpDiscoveryProvider{
                         defaultBin: "kiro-cli", clientName: "multica-model-discovery", ...})
  models.go:1135-1137  cmdArgs := p.acpArgs
                       if len(cmdArgs) == 0 { cmdArgs = []string{"acp"} }

config.go:144-155 (agentBrainBuiltInCLIFor) so vale para o modo gateway-required, que VOCE desligou as
21:26. Com gateway-required falso, o dropdown nao passa por aquele mapeamento — passa por ListModels
acima. Ou seja: os 3 CLIKinds nao sao a causa do dropdown vazio agora.

## POR QUE CODEX POPULA E OS OUTROS DOIS NAO (mesmo arquivo)

  models.go:129-134  case "codex": models := codexStaticModels()   // catalogo ESTATICO, nao precisa login
  models.go:123-128  case "claude": models := claudeStaticModels() // idem
kiro e antigravity sao discovery DINAMICO por processo filho, e sem fallback estatico:
  models.go:1408-1409  if len(models) == 0 { return nil, fmt.Errorf("antigravity model discovery returned no models") }
Compare com copilot, que TEM fallback: "Falls back to copilotStaticModels() when the binary is missing
or when the ACP handshake fails (auth missing, network down, etc.) so the UI dropdown always has
something to show." (comentario em models.go, funcao discoverCopilotModels).

## O QUE ISSO MUDA NA CORRECAO

Nao e (so) codigo. Existem dois caminhos, e os dois sao STOP-AND-WAIT:
A) AUTENTICAR os CLIs no ORQ1 como ec2-user: `kiro-cli login` e `agy` (sem argumentos, para sign-in).
   E mudanca de credencial/auth => decisao escrita do owner. Nao executei. E o caminho que resolve o
   dropdown sem tocar em codigo nem recompilar.
B) Dar fallback estatico a kiro/antigravity espelhando discoverCopilotModels (dropdown nunca vazio).
   Exige editar models.go + recompilar => proibido nesta rodada e STOP-AND-WAIT.
Ordem recomendada: A primeiro (barato, reversivel, sem build); B so se o owner quiser dropdown
resiliente a falha de auth. Sem (A), qualquer task com kiro ou agy vai falhar no login, nao no modelo.

## NAO-ALEGACOES

- Nao editei arquivo, nao compilei, nao troquei binario, nao reiniciei nada, nao commitei, nao logei
  nenhum CLI, nao rodei `kiro-cli login` nem `agy` interativo.
- Nao testei cline/opencode/claude/codex no terminal do ORQ1 (fora da pergunta).
- Nao inspecionei nem imprimi credencial alguma; so li mensagens de erro de login.
- Nao provei que /tmp/multica-auth-fixed (build 24/07) e identico ao models.go que li no repo.
