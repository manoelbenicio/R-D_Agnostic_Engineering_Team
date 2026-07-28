# ORQ-44 - Re-review independente da EMENDA OBRIGATORIA 1 (E1-E8)

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T16:00Z
- auditado: emenda `E1`-`E8` de `.deploy-control/p0/evidence/orq44-omniroute-inference-key-lifecycle.md`
  (Opus48#B, status **PROPOSTA**, sem auto-aprovacao)
- independencia: **nao sou o autor** (Opus48#B) **nem o primeiro revisor** deste cartao.
- skill carregada **integralmente** antes desta revisao:
  `.agents/skills/aws-secrets-manager/SKILL.md` v1 - overview, aviso *"best-effort defense, not a
  security boundary"*, as 3 regras MUST, sintaxe `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}`
  com defaults `SecretString`/`AWSCURRENT`, `Using asm-exec`, `How It Works` (scan de argumentos,
  ordem SMA -> MCP SigV4, `re.sub` single-pass), SigV4, prerequisitos, Common Patterns (incluindo
  *Configuration file templating*), hook `PreToolUse` do `aws-core` e troubleshooting.
- modo: READ-ONLY. **Nao li segredo, nao fiz `stat` do arquivo alvo**, zero
  `get-secret-value`/`batch-get-secret-value`, zero SMA em `localhost:2773`, **zero `asm-exec`**, zero
  chamada AWS ou provider, zero troca de arquivo, restart, mutacao de board, build ou implementacao.

---

## VEREDITO: **BLOCK**

A emenda melhora muito o corpo original e **acerta a direcao dos dois achados** que a motivaram. Mas
encontrei **tres defeitos verificados no codigo**:

1. **`D1` e `D2` sao inexequiveis** - nao existe flag nem variavel de ambiente de porta de health, e a
   porta **nunca** aparece na unit; e a emenda **nao viu** que existe uma derivacao **deterministica**
   real no codigo, que torna `D1`-`D3` desnecessarios na maior parte dos casos;
2. **`V4(b)` da E5 da falsa garantia** - o probe de readiness do gateway **so** roda dentro de
   `admitTask`; o que o `/health` expoe e um **snapshot da ultima admissao**, portanto **stale** apos o
   swap;
3. **`E3.1` depende de uma pre-condicao que a propria emenda lista como pergunta 4 nao resolvida**, e
   ainda assim a marca como **PREFERIDA** - inversao de precedencia.

E, como instruido, as **quatro perguntas ao provider/owner permanecem bloqueadores nao resolvidos**;
nao inventei resposta para nenhuma.

---

## 1. ✅ O que a emenda acertou (verificado, nao aceito por confianca)

| item | verificacao independente |
|---|---|
| **E1**: 19514 e **so default** | `config.go:58 DefaultHealthPort = 19514`; `:88 HealthPort int // ... (default: 19514)`; `:571-573 healthPort := DefaultHealthPort; if overrides.HealthPort > 0 {...}`; `health.go:63 addr := fmt.Sprintf("127.0.0.1:%d", d.cfg.HealthPort)`. **Todas as quatro citacoes conferem.** Fixar 19514 e de fato errado. |
| **E1** risco de falso PASS/BLOCK | procede, e o **proprio codigo reforca**: `health.go:102-105` avisa que *"The health port is bound before preflight for liveness/diagnostics, so callers must not treat a reachable endpoint as ready"*. Um `200` nao prova prontidao nem identidade. |
| **E2**: UNIX-only | `credential_file_source_windows.go` com `//go:build windows` retorna `platform_unsupported` em **`openCredentialFile`** e **`checkCredentialOwner`**, literal, incondicional. E existe o par `credential_file_source_unix.go` com `//go:build !windows`. **Confere.** A conclusao normativa "UNIX-only, parar em nao-Unix" esta correta. |
| **E7**: logging content-free | `credential_file_source.go` mantem `credentialFileError` com apenas token de razao; toda validacao proposta usa `stat` (metadado) ou codigo de saida de `grep -q`. Proibir `cat`/`head`/`tail`/`xxd`/`od` esta certo. **Nenhuma objecao.** |
| **E5**: trocar smoke task por probe | a **motivacao** esta certa e e importante: atribuir/comentar enfileira task paga, logo "task de smoke" era execucao disfarcada de validacao. O problema e a **substituicao escolhida** (secao 3). |
| **E4**: teto de uma listagem, proibir inferencia | correto e coerente: `/v1/models` e o endpoint de **Readiness** do proprio cliente (`brain_integration.go:327 Endpoints: gateway.EndpointSet{Liveness: "/api/health/ping", Readiness: "/v1/models"}`), ou seja o canario usa o mesmo endpoint que o daemon ja usaria - nao inventa trafego novo de natureza diferente. |
| nao auto-aprovacao | secao de status diz PROPOSTA e pede re-review. **Confere.** |

---

## 2. 🔴 BLOQUEADOR 1 - `D1` e `D2` nao existem; e a emenda perdeu a derivacao real

### 2.1 A derivacao deterministica que a emenda nao viu
`cmd/multica/cmd_daemon.go:159-172`, literal:
```go
// healthPortForProfile returns the health check port for the given profile.
// Default profile uses the standard port (19514). Named profiles get a
// deterministic offset derived from the profile name.
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
e o unico ponto que popula o override: `cmd_daemon.go:383 HealthPort: healthPortForProfile(profile)`.

Ou seja a porta e **funcao pura do nome do perfil**: `19514` para o perfil default, e
`19515 + (soma dos bytes do nome % 1000)` para perfil nomeado. Isso e uma derivacao **D0**, mais forte
e mais barata que qualquer uma das tres propostas, e a emenda nao a menciona.

### 2.2 Por que `D1` e `D2` sao inexequiveis
- **`D1` ("porta declarada explicitamente pelo operador para o perfil alvo")**: **nao existe knob**.
  `grep -rn "HealthPort" cmd/ internal/daemon/config.go` (sem testes) retorna apenas a constante, o
  campo, o campo de `Overrides`, a derivacao por perfil e a atribuicao final. **Nao ha flag
  `--health-port` e nao ha variavel de ambiente.** O operador nao pode declarar porta; ele pode, no
  maximo, escolher o **perfil**.
- **`D2` ("derivada da unit: ExecStart e Environment")**: a porta **nunca aparece na unit**, porque nao
  e parametro. Confirmei no unit real do ORQ2: o `ExecStart` e
  `... daemon start --foreground --no-auto-update --server-url http://127.0.0.1:18080 --daemon-id orq2-credential-runtime-v1 --device-name ORQ2 Credential Runtime`
  e o bloco `Environment=` traz `PATH` e os `MULTICA_*_PATH` - **nenhuma** porta. Logo `D2` **nunca**
  resolve, para nenhum perfil.

**Correcao exigida**: substituir `D1`/`D2` por:
```
D0 (canonico): derivar do PERFIL, aplicando healthPortForProfile(profile):
   perfil vazio -> 19514 ; perfil nomeado -> 19514 + 1 + (soma dos bytes do nome mod 1000).
   O perfil e o unico insumo, e ele SIM aparece na unit / no comando.
D3 (fallback/confirmacao): ss -ltnp correlacionado ao PID, como a emenda ja propoe.
```
E manter `I1`/`I2` como estao - eles sao a parte boa da E1 (secao 4).

Risco pratico de deixar como esta: um executor que siga `D1` para e escala sem necessidade; um que
siga `D2` conclui "nao consta na unit, logo e o default 19514" e cai exatamente no falso PASS que a
E1 quer evitar - em host multi-perfil.

## 3. 🔴 BLOQUEADOR 2 - `V4(b)` da E5 e **stale por construcao**, e da falsa garantia

A E5 substitui a task de smoke por: *"b) readiness do gateway reportando protocolo e modelo
selecionados sob politica estrita fail-closed"*. Verifiquei quem produz esse dado.

**O probe de readiness so existe dentro da admissao de task.** `NewReadinessChecker` tem **um** unico
chamador fora de teste:
```
internal/daemon/brain_integration.go:356  checker, err := gateway.NewReadinessChecker(client, registry, ...)
```
e a funcao que o contem e `admitTask`:
```
internal/daemon/brain_integration.go:269  func (r *agentBrainRuntime) admitTask(ctx context.Context, task Task, provider, legacyModel string) (*agentBrainTaskPlan, error) {
```
**O que o `/health` expoe nao e um probe, e um snapshot.** `internal/daemon/health.go:124-138`:
```go
	diagnostics := d.agentBrain.snapshot()
	...
	State: diagnostics.State, Readiness: string(diagnostics.Readiness),
	CLIKind: string(diagnostics.CLIKind), RouteModel: string(diagnostics.RouteModel),
	RouterOwner: string(diagnostics.RouterOwner), Protocol: string(diagnostics.Protocol),
```
`snapshot()` le `agentBrainDiagnostics`, que e preenchido **durante a admissao**. Portanto:

- **antes** de qualquer nova admissao, o bloco `agent_brain` do `/health` reflete a **ultima**
  admissao - isto e, a chave **antiga**;
- apos o swap, `V4(b)` retornaria protocolo e modelo "selecionados" com aparencia saudavel **sem ter
  tocado a chave nova**. E um **falso PASS estrutural**, nao um risco marginal;
- para reavaliar readiness de verdade e preciso `admitTask`, ou seja **enfileirar task paga** -
  exatamente o que a E5 existe para evitar.

O proprio codigo antecipa a confusao: `health.go:102-105` avisa que a porta e ligada **antes** do
preflight e que *"callers must not treat a reachable endpoint as ready"*.

**Correcao exigida** - `V4` reescrito com honestidade sobre o que cada sinal prova:
```
V4a  health na porta DERIVADA (D0, confirmada por D3) + identidade por I1/I2  -> 200
     prova: processo vivo e e o alvo. NAO prova nada sobre a credencial.
V4b  bloco agent_brain do /health lido ANTES e DEPOIS do swap, comparados
     prova: ausencia de REGRESSAO de estado. Declarar explicitamente que e SNAPSHOT da ultima
     admissao e que, sem nova admissao, NAO reflete a chave nova.
V4c  a unica prova real da chave nova e o canario L1, executado ANTES do swap (secao 5 do corpo).
     Depois do swap, sem task, NAO existe validacao positiva possivel sem custo.
```
E, como consequencia, elevar `L1` de "recomendado" a **pre-condicao dura do L2**: se o canario nao
rodar, o swap instala credencial nao verificada e a proxima requisicao real vira `401`.

## 4. 🔴 BLOQUEADOR 3 - `E3.1` marcada como PREFERIDA depende da pergunta 4, que a propria emenda diz nao resolvida

A E3.1 (rollback por handle `AWSPREVIOUS`, sem `.prev` em disco) e **melhor** que a E3.2 - concordo com
o raciocinio e com a preferencia por nao duplicar segredo em disco. Mas a emenda a declara
**PREFERIDA** e, tres secoes depois, na E6, registra como **pergunta 4 NAO RESOLVIDA**: *"a promocao
no Secrets Manager preserva `AWSPREVIOUS` ... ou sobrescreve o valor?"*

Isso e inversao de precedencia: **nao se pode preferir um caminho cuja pre-condicao e desconhecida**.
Se `AWSPREVIOUS` nao existir no momento do rollback, o executor descobre isso **no pior instante
possivel** - com a chave nova instalada e falhando.

**Correcao exigida**: tornar a escolha **condicional e verificada antes do L2**, por metadado, sem ler
valor:
```
E3.0 (novo gate, antes de L2): confirmar por METADADO que existe versao com estagio AWSPREVIOUS
     -> aws secretsmanager list-secret-version-ids --secret-id <id>   (NAO retorna valor)
     ou describe-secret, inspecionando VersionIdsToStages.
     se AWSPREVIOUS existe  -> caminho E3.1 (sem .prev)
     se NAO existe          -> caminho E3.2 (.prev com trap), e o `.prev` passa a ser OBRIGATORIO
     se nao for possivel verificar -> PARAR
```
Observo que `list-secret-version-ids` e `describe-secret` **nao** sao `get-secret-value` nem
`batch-get-secret-value`, portanto nao violam R1 da skill - retornam apenas metadado de versao e
estagio. Eu **nao** executei nenhum dos dois nesta revisao.

Nota adicional sobre a E3.2, que esta boa: o `trap ... EXIT INT TERM` cobrindo `.prev`, `.new` e
`.rollback`, mais o `test ! -e` final, e exatamente o que faltava. E a recusa em propor `shred` com a
justificativa de CoW/SSD e tecnicamente correta e honesta - `shred` daria falsa seguranca.

---

## 5. Ataques que **nao** encontraram defeito

- **Troca atomica**: `rename(2)` no mesmo diretorio, temporario com ponto inicial, `umask 077` antes de
  criar, proibicao de `cp` sobre o destino e de symlink. Coerente com o consumidor: `O_NOFOLLOW`,
  `Perm() & 0o077 != 0` rejeitado, dono = UID do processo, tamanho 8..4096. Nao achei furo.
- **Sem restart**: sustentado. O comentario de `credential_file_source.go` diz que le em **call time** e
  que o valor **nunca** e cacheado; nao ha `sync.Once` no caminho. O contraste com `JWT_SECRET`
  (`internal/auth/jwt.go:31`) esta correto, e a ressalva "trocar **caminho** exige restart, trocar
  **conteudo** nao" e precisa.
- **Gate de fila com 4 estados**: correto, derivado de `109:13-15`. E a justificativa especifica -
  task `running` tem CLI vivo que pode chamar o gateway a qualquer instante - e melhor que a formula
  generica usada nos outros runbooks.
- **Distincao dos tres segredos** (chave de inferencia, MCP do ORQ-37, `mdt_`): correta, e o ponto de
  que rotacionar a chave do gateway **nao** desloga usuario nem derruba WebSocket delimita bem o raio.
- **Custo do canario**: a natureza do endpoint (`/v1/models`, listagem) esta certa, e o teto de **uma**
  requisicao com timeout curto e sem loop e prudente. A honestidade sobre nao saber se listagem conta
  em cota e apropriada - e continua sendo a pergunta 3.

---

## 6. As quatro perguntas ao provider/owner - **TODAS SEGUEM BLOQUEADORAS**

Tratadas como nao resolvidas, conforme instruido. **Nao inventei resposta para nenhuma.**

| # | pergunta | estado | quem responde |
|---|---|---|---|
| **Q1** | O OmniRoute aceita **duas** chaves de inferencia ativas simultaneamente? Se sim, por quanto tempo e como se revoga a antiga? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q2** | A chave e por **conta** ou **global**? Rotacionar afeta as 4 contas Antigravity, 4 ClinePass e 2 OpenAI Codex, ou so o par do cliente? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q3** | Existe endpoint de introspeccao que valide a chave **sem consumir cota**, e a listagem do canario conta em alguma cota? | **NAO RESOLVIDA** | operador do OmniRoute |
| **Q4** | A promocao no Secrets Manager preserva `AWSPREVIOUS` (viabilizando E3.1) ou sobrescreve? | **NAO RESOLVIDA** | owner / operador de AWS |

Acrescento **uma quinta**, que decorre do Bloqueador 2 e nao pode ser respondida por agente:

| **Q5** | Aceita-se que, apos o swap, **nao exista** validacao positiva da chave nova sem custo - isto e, que a garantia venha do canario **pre**-swap (L1) e o pos-swap seja apenas ausencia de regressao? Se o owner exigir prova positiva pos-swap, isso implica **admitir uma task paga**, o que colide com o freeze de atribuicao e precisa de autorizacao explicita e orcamento. | **ABERTA** | owner |

## 7. Correcoes exatas para virar PASS

| # | correcao | onde |
|---|---|---|
| **C1** | substituir `D1`/`D2` (inexequiveis) por `D0` = `healthPortForProfile(profile)` - `19514` para perfil vazio, `19514 + 1 + (soma dos bytes mod 1000)` para nomeado - mantendo `D3` como confirmacao | E1.1 |
| **C2** | manter `I1`/`I2` (estao corretos) e citar `health.go:102-105` como razao de o `200` sozinho nao bastar | E1.2 |
| **C3** | reescrever `V4` em tres sinais com o que cada um prova (`V4a` identidade, `V4b` ausencia de regressao **declarada como snapshot stale**, `V4c` a prova real e o canario pre-swap) | E5 |
| **C4** | elevar `L1` a **pre-condicao dura** do `L2` | corpo, secao 5 |
| **C5** | inserir `E3.0`: verificar por **metadado** (`list-secret-version-ids`/`describe-secret`, que nao sao `get-secret-value`) se `AWSPREVIOUS` existe, **antes** do L2, e so entao escolher entre E3.1 e E3.2; sem verificacao -> PARAR | E3 |
| **C6** | remover a marcacao "PREFERIDA" de E3.1 enquanto Q4 estiver aberta; a preferencia passa a ser **condicional** ao resultado de `E3.0` | E3.1 |
| **C7** | registrar `Q5` na lista de perguntas bloqueantes | E6 |

Nenhuma dessas exige refazer o desenho. O nucleo - leitura por chamada, swap por `rename(2)`, sem
restart, logging content-free, UNIX-only - esta correto e eu o endosso.

## 8. Nao-afirmacoes desta revisao

- **Nao li segredo. Nao fiz `stat` do arquivo `/etc/agent-brain/secrets/omniroute-inference-key`** -
  nao sei seu modo, dono, tamanho nem se existe. Todas as exigencias que citei vem do **codigo do
  consumidor**.
- Zero `get-secret-value`/`batch-get-secret-value`, zero SMA em `localhost:2773`, **zero `asm-exec`
  executado**, zero chamada AWS ou provider. **Nao** executei `list-secret-version-ids` nem
  `describe-secret` - apenas os proponho como gate.
- Nao troquei arquivo, nao reiniciei nada, nao mutei board, nao compilei, nao implementei.
- Nao executei `healthPortForProfile` nem derivei a porta de nenhum perfil real; li a funcao.
- Nao confirmei `daemon_id` nem PID de socket nesta rodada; a verificacao do `ExecStart` do unit do
  ORQ2 e de leitura anterior (`systemctl --user cat`), e nao a repeti agora.
- Nao verifiquei o corpo de `credential_file_source_unix.go` linha a linha; confirmei a **existencia**
  e o build tag `//go:build !windows`, e li o arquivo **windows** integralmente.
- Nao sei se o segredo `prod/multica/omniroute-inference-key` existe; o nome e proposto pelo autor.
- Nao respondi nenhuma das quatro perguntas ao provider/owner, e adicionei uma quinta em vez de
  presumir.
- Nao alterei assignee nem postei comentario; nenhum card criado; freeze respeitado.
