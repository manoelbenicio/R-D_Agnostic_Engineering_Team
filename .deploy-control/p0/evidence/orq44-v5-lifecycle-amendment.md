# ORQ-44 - EMENDA V5 ao ciclo de vida da chave de inferencia do OmniRoute (PROPOSTA, READ-ONLY)

- **Status: PROPOSTA.** Nao aprovada, **nao executavel**. **Nao me auto-aprovo** - peco revisao
  independente por agente que nao seja eu (Opus48#A) nem o autor do corpo/E1-E8 (Opus48#B).
- autor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T16:30Z
- emenda: `.deploy-control/p0/evidence/orq44-omniroute-inference-key-lifecycle.md` (corpo + E1-E8)
- corrige: os tres bloqueadores do meu proprio parecer
  `.deploy-control/p0/evidence/orq44-e1-e8-independent-rereview.md` (BLOCK)
- skill carregada **integralmente** antes de escrever, nesta rodada:
  `.agents/skills/aws-secrets-manager/SKILL.md` v1 - overview, aviso *"best-effort defense, not a
  security boundary"*, as **3 regras MUST**, sintaxe
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` com defaults
  `SecretString`/`AWSCURRENT`, `Using asm-exec`, `How It Works` (scan de **argumentos**, ordem
  SMA -> MCP SigV4, regiao por ARN ou `AWS_REGION`, `re.sub` single-pass, **sem** fallback para CLI
  local), SigV4, prerequisitos, **Common Patterns** incluindo *Configuration file templating*, hook
  `PreToolUse` do `aws-core` e troubleshooting.
- modo: **READ-ONLY**. Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA em `localhost:2773`,
  **zero `asm-exec` executado**, **zero chamada a provider ou AWS**, **zero `stat` no arquivo de
  credencial**, zero leitura de valor, zero troca de arquivo, zero restart, zero mutacao de quadro,
  zero task admitida.

---

## V5.1 - substitui E1.1: derivacao **canonica** da porta de health

**Motivo.** A E1.1 propunha `D1` (porta declarada pelo operador) e `D2` (porta derivada da unit).
Verifiquei: **as duas sao inexequiveis**. Nao existe flag `--health-port` nem variavel de ambiente, e
a porta **nunca** aparece na unit porque nao e parametro. O `ExecStart` da unit do ORQ2 carrega
`--daemon-id` e `--device-name`, e o bloco `Environment=` traz `PATH` e os `MULTICA_*_PATH` - nenhuma
porta.

**A derivacao real e deterministica e depende de um unico insumo: o nome do perfil.**
`cmd/multica/cmd_daemon.go:162-172`, literal:
```go
func healthPortForProfile(profile string) int {
	if profile == "" {
		return daemon.DefaultHealthPort
	}
	// Simple hash: sum of bytes mod 1000, offset from base+1.
	var h int
	for _, b := range []byte(profile) {
		h += int(b)
	}
	return daemon.DefaultHealthPort + 1 + (h % 1000)
}
```
e ela e usada de forma consistente em **todos** os pontos que precisam da porta:
`cmd_daemon.go:186`, `:383` (`HealthPort:` do override de config), `:517`, `:554`, `:636`. O perfil
entra no processo filho por `--profile` (`cmd_daemon.go:360`), e `DefaultHealthPort = 19514`
(`internal/daemon/config.go:58`), com o bind em `127.0.0.1:%d` (`internal/daemon/health.go:63`).

### D0 (canonico) - calcular a partir do perfil
```
perfil vazio (default)  ->  19514
perfil nomeado <P>      ->  19514 + 1 + ( soma_dos_bytes_de(<P>) mod 1000 )
```
Consequencia direta para o alvo conhecido: a unit do ORQ2 **nao** passa `--profile`, portanto o perfil
e vazio e a porta canonica e **19514**. O `19514` volta a ser correto - mas agora **derivado**, nao
presumido, e a diferenca importa em host multi-perfil.

### D3 (confirmacao obrigatoria, nao alternativa)
```bash
ss -ltnp 2>/dev/null | grep -F '127.0.0.1:<PORTA_D0>'
```
`D0` sozinho **nao** prova que o processo que escuta e o alvo; `D3` amarra porta -> PID.

## V5.2 - substitui E1.2: identidade fechada em ciclo, pelo proprio corpo do `/health`

A E1.2 propunha `I1` (`--daemon-id` no `ExecStart`) e `I2` (MainPID vs PID do socket). **Endosso as
duas** - e acrescento que o `/health` publica os campos de identidade necessarios para fechar o ciclo
sem depender apenas da unit. `internal/daemon/health.go:20-37` declara no payload:
`status`, **`pid`**, `os`, `uptime`, **`daemon_id`**, **`device_name`**, `server_url`, `cli_version`,
`active_task_count`, `agents`, `workspaces`, `agent_brain`.

### Sequencia de identidade (todos os passos exigidos, em ordem)
1. **D0** calcula a porta a partir do perfil declarado da unit alvo;
2. **D3** confirma que ha socket em `127.0.0.1:<D0>` e captura o **PID do socket**;
3. `systemctl --user show -p MainPID <unit>` e o MainPID **deve** casar com o PID do passo 2;
4. `GET http://127.0.0.1:<D0>/health` e o corpo **deve** trazer `pid` igual ao MainPID **e**
   `daemon_id` igual ao `--daemon-id` do `ExecStart` (para o ORQ2, `orq2-credential-runtime-v1`);
5. `os` deve ser `linux` (ver V5.6).

Qualquer divergencia em 1-5 = **STOP**. Um `200` sozinho **nao** e prova: o proprio codigo avisa, em
`internal/daemon/health.go:102-105`, que a porta e ligada **antes** do preflight *"for
liveness/diagnostics, so callers must not treat a reachable endpoint as ready"*.

## V5.3 - substitui E5: o `/health` pos-swap e **snapshot velho**; o canario pre-swap passa a ser **gate duro**

**Motivo.** A E5 substituiu, corretamente, a "task de smoke" por um probe - mas escolheu o
`agent_brain` do `/health`, que **nao e um probe**.

**Evidencia.** O probe de readiness do gateway existe em **um** unico ponto fora de teste:
`internal/daemon/brain_integration.go:356` chama `gateway.NewReadinessChecker(...)`, e esse ponto esta
**dentro** de `admitTask` (`brain_integration.go:269`). O que o `/health` publica vem de
`health.go:124 d.agentBrain.snapshot()` e e servido em `health.go:136-138` como
`state`/`readiness`/`cli_kind`/`route_model`/`router_owner`/`protocol` - todos preenchidos **durante a
ultima admissao**. Logo, apos a troca da chave e **sem** nova admissao, esses campos descrevem a chave
**antiga**.

### Reclassificacao honesta dos sinais

| sinal | prova o que | **nao** prova |
|---|---|---|
| **L1 - canario `/v1/models` PRE-swap, com a chave NOVA** | que a chave nova e aceita pelo OmniRoute | nada sobre a troca em si |
| `V4a` porta D0 + identidade V5.2 + `/health` `200` | que o processo alvo esta vivo | **nada** sobre credencial |
| `V4b` `agent_brain.secret_reference_configured` | que **existe referencia** de segredo configurada (booleano, nao valor) | que a referencia aponta para a chave **nova** |
| `V4c` `state`/`readiness`/`cli_kind`/`route_model`/`router_owner`/`protocol`/`last_outcome` | **ausencia de regressao** do estado anterior | **NADA sobre a chave nova - e snapshot da ultima admissao** |
| `V5` observacao de trafego real subsequente | so produz sinal **se** houver trafego | nada, se a fila estiver vazia |

`agent_brain.secret_reference_configured` e `agent_brain.last_outcome` sao campos reais do payload
(`health.go:41-56`) e sao **content-free** - booleano e rotulo de resultado, nunca valor.

### Consequencias normativas (as tres, obrigatorias)
1. **L1 deixa de ser recomendado e passa a ser PRE-CONDICAO DURA do swap.** Sem canario verde com a
   chave nova, o swap instala credencial **nao verificada** e a proxima requisicao real falha com
   `401`. Se `L1` nao puder ser executado, **nao se troca a chave**.
2. **Todo `V4b`/`V4c` pos-swap deve vir rotulado no runbook como "ausencia de regressao (snapshot da
   ultima admissao)"**, com a frase explicita: *"nao reflete a chave nova"*. Sem esse rotulo, o
   executor le um verde falso.
3. **Nao existe validacao positiva pos-swap sem custo.** Nao inventar uma. Isso e a **Q5** (secao
   V5.8), que **so o owner** pode responder.

### O que continua proibido
Nenhuma task e admitida, atribuida ou comentada para validar. Atribuir agente enfileira trabalho
**pago** e comentario em issue atribuida pode enfileirar. A E5 acertou a motivacao; a V5 mantem a
proibicao e apenas corrige o substituto.

## V5.4 - substitui E3: gate de **existencia do `AWSPREVIOUS`**, somente metadados, **antes** do swap

**Motivo.** A E3.1 (rollback por estagio, sem `.prev` em disco) esta marcada como **PREFERIDA** e, tres
secoes depois, a propria emenda registra como **NAO RESOLVIDA** a pergunta 4 - se a promocao preserva
`AWSPREVIOUS`. E inversao de precedencia: se o estagio nao existir, o executor descobre com a chave
nova instalada e falhando.

### E3.0 (novo, obrigatorio, antes do L2)
Verificar **por metadado** se ha versao com estagio `AWSPREVIOUS`:
```bash
aws secretsmanager describe-secret          --secret-id <SECRET-ID> --region <REGIAO>
aws secretsmanager list-secret-version-ids  --secret-id <SECRET-ID> --region <REGIAO>
# inspecionar VersionIdsToStages procurando o estagio AWSPREVIOUS
```
Justificativa de conformidade com a skill: `describe-secret` e `list-secret-version-ids` **nao** sao
`get-secret-value` nem `batch-get-secret-value`; retornam apenas metadado de versao e estagio, nao
`SecretString`. A **regra 1** proibe as APIs de valor, e nenhuma das duas o e. Ainda assim:

- **eu nao executei nenhuma das duas** - apenas as proponho como gate;
- a saida deve ser reduzida a **um booleano** no runbook (`AWSPREVIOUS presente: sim/nao`), sem colar
  ARNs de versao nem qualquer payload;
- `describe-secret` **nao** prova que uma `json-key` existe; isso exige atestacao do owner sem revelar
  valor.

### Arvore de decisao
```
AWSPREVIOUS presente     -> caminho E3.1 (rollback por estagio, sem .prev em disco)
AWSPREVIOUS ausente      -> caminho E3.2, e o arquivo .prev passa a ser OBRIGATORIO
nao foi possivel verificar (AccessDenied, sem identidade) -> STOP, nao trocar
```
Enquanto **Q4** estiver aberta, **remover** a marcacao "PREFERIDA" da E3.1: a preferencia passa a ser
**condicional ao resultado do E3.0**, decidido na janela.

Mantenho e endosso, da E3.2: o `trap ... EXIT INT TERM` cobrindo `.prev`, `.new` e `.rollback`, o
`test ! -e` final, e a **recusa fundamentada** de propor `shred` - em sistema com CoW e SSD com
wear-leveling ele daria falsa garantia; a protecao real e o modo restrito no diretorio restrito.

## V5.5 - E2 preservada: **UNIX-only**, com o limite do que verifiquei

A direcao da E2 esta correta e a mantenho sem alteracao. O que **verifiquei diretamente**:

- `internal/daemon/credential_file_source_windows.go` tem `//go:build windows` e falha fechado
  **incondicionalmente** nos dois helpers, retornando `&credentialFileError{reason:
  "platform_unsupported"}` em `openCredentialFile` **e** em `checkCredentialOwner`; o comentario
  registra a razao: sem equivalente TOCTOU-safe de `O_NOFOLLOW` sem Win32, e o daemon
  gateway-required nao e implantado em Windows;
- existe o par `internal/daemon/credential_file_source_unix.go` com `//go:build !windows`.

O que **nao** verifiquei: o corpo do `_unix.go` linha a linha. **Norma da V5**: o runbook so se aplica
a alvo com `os` = `linux` no corpo do `/health` (V5.2 passo 5); em qualquer outro `GOOS`, **STOP** -
nao por politica, mas porque o codigo ja falha fechado e a troca de arquivo seria inutil.

## V5.6 - E7 preservada: logs **content-free**, sem excecao

Mantida integralmente. Verificado nesta rodada em
`internal/daemon/credential_file_source.go:111-119`: `credentialFileError` guarda **apenas** um token
de razao, e o proprio comentario declara ser *"a deterministic, content-free error that never leaks the
secret value, file content, or detailed OS error messages"*.

Norma reafirmada: toda validacao usa **metadado** ou **codigo de saida**. Proibido `cat`, `head`,
`tail`, `xxd`, `od`, `strings`, `diff`, `grep` sem `-q`, ou qualquer redirecionamento do conteudo.
Nenhum valor, nem prefixo, nem comprimento de valor valido entra em evidencia. Os campos do `/health`
usados pela V5 (`secret_reference_configured`, `last_outcome`, `daemon_id`, `pid`, `os`) sao todos
content-free.

## V5.7 - sequencia consolidada (ordem e obrigatoria)

```
E3.0  gate de metadado: AWSPREVIOUS presente? -> escolhe E3.1 ou E3.2; indeterminado = STOP
D0    calcular a porta pelo perfil (19514 se perfil vazio)
D3    confirmar socket em 127.0.0.1:<D0> e capturar PID
V5.2  identidade: MainPID == PID do socket; /health.pid == MainPID; /health.daemon_id == --daemon-id;
      /health.os == linux                                          [qualquer falha = STOP]
L1    CANARIO PRE-SWAP com a chave NOVA em /v1/models              [GATE DURO - sem verde, nao trocar]
      -- fronteira de autorizacao: nada acima muta nada --
L2    troca atomica do arquivo de credencial (rename no mesmo diretorio, modo restrito, trap)
V4a   /health 200 na porta D0 + identidade V5.2 revalidada
V4b   secret_reference_configured == true                          [config, nao valor]
V4c   comparar snapshot agent_brain antes/depois                   [AUSENCIA DE REGRESSAO,
                                                                    NAO reflete a chave nova]
V5    se e quando houver trafego real, observar resultado          [sinal so se houver trafego]
R     rollback: E3.1 por estagio, ou E3.2 pelo .prev; repetir V4a/V4b/V4c
```
Notas de custo e de escopo: `L1` usa `/v1/models`, que **ja** e o endpoint de `Readiness` configurado
pelo proprio cliente (`internal/daemon/brain_integration.go:327`
`Endpoints: gateway.EndpointSet{Liveness: "/api/health/ping", Readiness: "/v1/models"}`), portanto o
canario nao inventa trafego de natureza nova - **mas** se listagem consome cota permanece a **Q3**.
Teto de **uma** requisicao, timeout curto, sem loop, sem retry.

## V5.8 - as cinco perguntas **permanecem bloqueadoras** (nenhuma respondida por agente)

| # | pergunta | estado | quem responde |
|---|---|---|---|
| **Q1** | o OmniRoute aceita **duas** chaves de inferencia ativas ao mesmo tempo? Por quanto tempo, e como se revoga a antiga? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q2** | a chave e por **conta** ou **global**? Rotacionar afeta as 4 contas Antigravity, 4 ClinePass e 2 OpenAI Codex, ou so o par do cliente? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q3** | existe endpoint de introspeccao que valide a chave **sem consumir cota**, e a listagem do `L1` conta em alguma cota? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q4** | a promocao no Secrets Manager **preserva `AWSPREVIOUS`** ou sobrescreve? | **NAO RESOLVIDA** - e o que o `E3.0` decide na janela | owner / operador AWS |
| **Q5** | aceita-se que, apos o swap, **nao exista** validacao positiva da chave nova sem custo - com a garantia vindo do `L1` pre-swap e o pos-swap sendo apenas ausencia de regressao? Se o owner exigir prova positiva pos-swap, isso implica **admitir task paga**, colidindo com o freeze de atribuicao e exigindo autorizacao e orcamento explicitos | **ABERTA** | owner |

Acrescento uma sexta, que a V5 tornou necessaria ao introduzir o `E3.0`:

| **Q6** | qual o **secret-id exato** e a **regiao** do segredo da chave de inferencia, e qual identidade tem permissao de `describe-secret`/`list-secret-version-ids`? O corpo propoe um nome; eu **nao adivinho nome de segredo** e a identidade desta sessao nao foi verificada | **ABERTA** | owner |

## V5.9 - o que a V5 **nao** altera

Preservo sem mudanca, porque atacei e nao encontrei defeito: a leitura da credencial **por chamada**
(sem cache, portanto **sem restart** para troca de conteudo - so troca de **caminho** exigiria
restart, ao contrario do `JWT_SECRET`, que tem `jwtSecretOnce sync.Once` em
`internal/auth/jwt.go:31`); a troca atomica por `rename(2)` no mesmo diretorio com `umask` restrito e
proibicao de `cp` sobre o destino e de symlink; o gate de fila com os **4** estados ativos
(`queued, dispatched, running, waiting_local_directory`, de
`migrations/109_agent_task_waiting_local_directory.up.sql:15`), com a justificativa especifica de que
uma task `running` tem CLI vivo que pode chamar o gateway a qualquer instante; e a distincao dos tres
segredos, incluindo o ponto de que rotacionar a chave do gateway **nao** desloga usuario nem derruba
WebSocket.

## V5.10 - nao-afirmacoes

- **Nao li nenhum valor de segredo. Nao fiz `stat` no arquivo de credencial** - nao sei modo, dono,
  tamanho nem se existe. Tudo que exijo vem do **codigo do consumidor**.
- Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA em `localhost:2773`, **zero `asm-exec`
  executado**, **zero chamada a provider ou AWS**. **Nao** executei `describe-secret` nem
  `list-secret-version-ids` - apenas os proponho como gate `E3.0`.
- Nao troquei arquivo, nao reiniciei unit, nao admiti/atribui/comentei task, nao mutei quadro, nao
  compilei, nao implementei.
- **Nao executei `L1`** nem qualquer requisicao a `/v1/models`.
- Nao calculei a porta de nenhum perfil real executando codigo; li a funcao e apliquei o algoritmo ao
  caso "perfil vazio". A afirmacao de que a unit do ORQ2 nao passa `--profile` vem de leitura anterior
  de `systemctl --user cat`, **nao repetida nesta rodada**.
- Nao verifiquei o corpo de `credential_file_source_unix.go` linha a linha; confirmei existencia e
  build tag, e li o `_windows.go` integralmente.
- Nao sei se o segredo proposto no corpo existe; ver **Q6**.
- **Nao respondi nenhuma das cinco perguntas** e acrescentei uma sexta em vez de presumir.
- Nao alterei assignee nem postei comentario em issue; nenhum card criado; freeze respeitado.

---

**PROPOSTA - requer revisao independente.** Ao revisor, ataque prioritariamente: (a) se `D0` cobre
alvo que use override de config por outro caminho que eu nao tenha encontrado; (b) se
`secret_reference_configured` realmente prova **configuracao** e nao **valor** - e se e atualizado
fora de `admitTask`; (c) se `describe-secret`/`list-secret-version-ids` sao aceitaveis sob a regra 1 da
skill; (d) se declarar `V4c` como "snapshot velho" e suficiente, ou se ele deveria ser **removido** do
runbook para nao induzir leitura errada.
