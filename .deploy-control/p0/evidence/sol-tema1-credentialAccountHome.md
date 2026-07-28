# TEMA 1 - por que credentialAccountHome fica vazio - PREPARADO, NAO APLICADO

- agente: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-26T23:07Z
- escopo: agy, codex, kiro obrigatorios. opencode FORA de escopo por ordem do owner.
- estado: nenhum arquivo de codigo editado, sem build, sem vet, sem restart.

## 1. RESPOSTA: nao existe condicao, feature flag, campo de banco nem lookup que falha

Nao ha nada para "consertar" numa condicao. `credentialAccountHome` e um **literal vazio
hardcoded, nunca reatribuido**. Prova por enumeracao completa das ocorrencias em daemon.go:

```
internal/daemon/daemon.go:3448	credentialAccountHome := ""
internal/daemon/daemon.go:3475			CredentialAccountHome: credentialAccountHome,
internal/daemon/daemon.go:3492			CredentialAccountHome: credentialAccountHome,
```
`grep -rn "credentialAccountHome" internal/daemon/ | grep -v _test` retorna exatamente estas 3
linhas: uma declaracao e dois consumidores. Nao existe `credentialAccountHome =` em lugar algum.
Nao ha `if`, nao ha env var, nao ha leitura de banco, nao ha resolver que retorne erro.

Trecho literal, daemon.go:3444-3449:
```go
3444 	// Resolve any local_directory assignment again here so runTask can plumb
3445 	// LocalWorkDir into execenv. handleTask already validated + locked the
3446 	// path; this call is a pure JSON parse over the same task payload.
3447 	localAssignment, _ := findLocalDirectoryAssignment(task.ProjectResources, d.cfg.DaemonID)
3448 	credentialAccountHome := ""
3449 	startedTask := false
```

E consumido literalmente em Reuse (3467-3478) e Prepare (3482-3495):
```go
3475 			CredentialAccountHome: credentialAccountHome,
3476 			CredentiallessGateway: true,
...
3492 			CredentialAccountHome: credentialAccountHome,
3493 			CredentiallessGateway: true,
```

O proprio codigo documenta o resolver que nunca foi escrito, execenv.go:68-72:
```go
68 	// The daemon resolves this from the agent→account assignment. Each vendor
69 	// maps it onto its native isolation lever (Codex: CODEX_HOME source;
70 	// Kiro: XDG_DATA_HOME / KIRO_API_KEY; Antigravity: HOME; Cline:
71 	// CLINE_DATA_DIR; OpenCode/GLM: XDG_DATA_HOME + XDG_CONFIG_HOME).
72 	// Empty is invalid for credential-bearing providers that require isolation.
```
"The daemon resolves this from the agent→account assignment" e a unica especificacao do valor, e
esse resolver **nao existe**. O parametro esta plumbado ponta a ponta e alimentado com "".

## 2. SEGUNDO LITERAL, que muda o patch: CredentiallessGateway: true

`CredentiallessGateway: true` tambem e hardcoded, 3476 e 3493. Ele NAO e neutro para o codex:

```go
281 	if params.Provider == "codex" {
282 		codexHome := filepath.Join(envRoot, "codex-home")
283 		var err error
284 		if params.CredentiallessGateway {
285 			err = prepareCredentiallessCodexHome(codexHome)
286 		} else {
287 			err = prepareCodexHomeWithOpts(codexHome, CodexHomeOptions{CodexVersion: params.CodexVersion, AccountHome: params.CredentialAccountHome}, logger)
288 		}
```
Com `true`, o codex desvia para `prepareCredentiallessCodexHome` e **ignora AccountHome mesmo se
preenchido**. Ja kiro e antigravity testam so o vazio:
```go
301 	if params.Provider == "kiro" && params.CredentialAccountHome != "" {
303 		if err := prepareKiroHome(kiroDataHome, KiroHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
312 	if params.Provider == "antigravity" && params.CredentialAccountHome != "" {
314 		if err := prepareAntigravityHome(agyHome, AntigravityHomeOptions{AccountHome: params.CredentialAccountHome}, logger); err != nil {
```
CONCLUSAO POR PROVIDER, com os 3 obrigatorios:
- kiro: basta preencher CredentialAccountHome. O gate da 301 passa a ser verdadeiro.
- agy/antigravity: idem, gate da 312.
- codex: precisa preencher CredentialAccountHome **E** passar CredentiallessGateway=false, senao o
  desvio da 284 anula tudo.
As mesmas quatro condicoes existem duplicadas no caminho Reuse (503, 519, 529, 539), entao o patch
tem de cobrir Prepare E Reuse.

## 3. DE ONDE O VALOR DEVE VIR

- `~/.agent-cred-homes/slots/slot-NNN/home` **nao aparece em nenhum lugar do codigo Go**:
  `grep -rn "agent-cred-homes\|cred-homes\|CredentialSlot" --include=*.go .` retorna ZERO. Logo o
  daemon nao pode inferir o caminho; ele tem de ser entregue.
- O payload do agente, `AgentData` em types.go:129-144, NAO tem campo de conta nem de slot. Tem
  ID, Name, Instructions, Skills, CustomEnv, CustomArgs, McpConfig, Model, ThinkingLevel,
  RuntimeConfig.
- CAMINHO RECOMENDADO, sem migracao de banco: usar `AgentData.RuntimeConfig` (types.go:143), que
  ja e o canal per-provider estabelecido e ja tem precedente de decodificacao no proprio runTask,
  daemon.go:3462-3466 (`decodeOpenclawRuntimeConfig`). Substituir 3448 por um
  `decodeCredentialAccountHome(task.Agent.RuntimeConfig, provider, d.logger)` que le uma chave
  tipo `credential_account_home`, valida que e diretorio absoluto existente e devolve "" em caso
  de duvida (fail-soft igual ao openclaw).
- Alternativa (b): campo novo em AgentData + endpoint claim + coluna. Mais correto a longo prazo,
  mas exige mudanca de servidor e banco. Para HOJE, recomendo (a).

## 4. EXISTE CAMINHO SEM RECOMPILAR?

Para preencher CredentialAccountHome, NAO. E literal compilado; nenhuma variavel de ambiente,
config ou registro de banco o alcanca. Qualquer solucao que dependa desse parametro exige rebuild.
`go build` e `go vet` passam exit 0 hoje. A mudanca e pequena e local: 1 decoder novo, 2 literais
(3448 e o par 3476/3493) e cobertura do caminho Reuse. Estimativa 30 a 60 min incluindo build e
vet, mais o relancamento do daemon, que e STOP-AND-WAIT.

O unico efeito imediato sem rebuild e o que o TL ja fez a mao (credencial no HOME global), que e
justamente o estado que o Codex56-TL vetou como permanente, porque com CredentialAccountHome vazio
o comportamento e explicitamente "shared/global" (comentarios das linhas 300, 311, 321, 340).

## 5. NADA MUTADO

Nenhum arquivo de codigo editado, sem go build, sem go vet, sem binario, sem restart de daemon ou
container, sem commit, push, reset, clean ou checkout --. Nenhum segredo lido ou transcrito.
