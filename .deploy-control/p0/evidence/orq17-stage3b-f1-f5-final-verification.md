# ORQ-17 Stage3B - verificacao final dos bloqueadores **F1-F5** (READ-ONLY)

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T17:12Z
- escopo autorizado (GATE 0 ACCEPT): **somente** os bloqueadores finais F1-F5, com as ferramentas
  presentes e SSH read-only ao ORQ1. **Nao** reabri A2.x, estilo, testes, pins de build nem qualquer
  item das tarefas pausadas ORQ-42/44/13.
- artefatos revisados e seus hashes, **medidos por mim** (nao recebi hash do General-TL; usei o meu):
```
fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82  orq17-stage3b-v2-bootstrap-helper.go.txt
c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42  orq17-stage3b-v2-sealed-runner.sh.txt
92f92b5a58e6efd763452dbd79eeb096d94591cd6334989fcf6b3155dc51dc97  orq17-stage3b-v2-bootstrap-helper_test.go.txt
dce169561dfaeb6d3751fadf1faef07b153b59cecb690f1f7aed020d9f76a2ba  orq17-stage3b-v2-owner-credential-provisioning-runbook.md
```
- modo: **READ-ONLY**. Nao executei helper, runner, Docker/Compose, SQL, `curl`, login, `asm-exec` nem
  chamada AWS. Nao usei o SSH ao ORQ1 nesta rodada (nao foi necessario). Sem mutacao de AWS, segredo,
  DB ou quadro.

---

## VEREDITO: **PASS** - os cinco bloqueadores estao fechados. 4 condicoes registradas, nenhuma bloqueante.

Os dois BLOCK do meu parecer anterior (L1 rollback preso a senha, L2 falha reportada apos commit) estao
resolvidos, e em dois pontos a solucao e **melhor** do que eu pedi. As novas superficies introduzidas
por F4 (`http-body`, `state`, cookie jar) **nao** criam exposicao de credencial nem caminho de
corrupcao - verifiquei cada uma.

## F1 - `rollback-breakglass` sem `Login`, com guardas mantidas - **FECHADO**

`rollbackBreakglass` existe e preserva **todas** as guardas: `assertFKCatalog`, `verifyGlobal(1,1,0)`,
`targetReferenceCounts` (que inclui `member`, cuja FK e `CASCADE`), DELETE exigindo `RowsAffected()==1`
em cada tabela, `verifyGlobal(0,0,0)` antes do commit, e locks via `beginMutation(rollback=true)`.
Acrescentou um controle que eu **nao** havia pedido e que e melhor: a identidade e **pinada**
(`expectedOwnerEmail`), e o DELETE so ocorre se o e-mail da unica linha casar - portanto o breakglass
nao pode apagar um usuario arbitrario.

Autorizacao: exige `ORQ17_BREAKGLASS_RECEIPT` **e** `ORQ17_BREAKGLASS_RECEIPT_SHA256`, verificados por
`check_receipt` (prefixo de caminho, modo `600`, UID do dono, igualdade de SHA-256). O runbook amarra o
recibo a ORQ1, ao login fixo, as contagens correntes e aos hashes exatos da V2.

## F2 - prova de custodia da senha antes do provision - **FECHADO**

O runner exige `ORQ17_CUSTODY_RECEIPT` + `ORQ17_CUSTODY_RECEIPT_SHA256` **antes** de qualquer mutacao,
e o runbook (`:37-40`) exige que o recibo declare `CUSTODY_PROVEN=1 LOGIN=<owner>` afirmando que a
senha real permanece recuperavel pelo owner. Falha fecha com `E_CUSTODY_PROOF`.

## F3 - estado terminal inequivoco quando o commit ja ocorreu - **FECHADO, e acima do pedido**

`execute` agora distingue explicitamente:
```
E_PROVISION_COMMITTED_GUARD_FAILED_COMPENSATED
E_PROVISION_COMMITTED_GUARD_FAILED_STATE_UNKNOWN
E_ROLLBACK_COMMITTED_GUARD_FAILED
```
E foi alem: `compensateWithFreshGuard` **readquire uma guarda nova** antes de compensar - correto e
necessario, porque o `pgx` fecha a transacao quando o `Commit` falha, deixando a antiga inutilizavel.
Sem essa reaquisicao, a compensacao rodaria **sem** o congelamento de fila/`member`.

No runner, `read_committed_state` usa o novo modo `state` (que **nao** recebe credencial: e chamado com
`</dev/null`) e traduz o resultado em tokens precisos:
`E_PROVISION_FAILED_STATE_EMPTY`, `..._ROLLED_BACK_AFTER_COUNTS`,
`E_PROVISION_FAILURE_STATE_REQUIRES_OWNER_REVIEW`,
`E_ROLLBACK_FAILURE_COUNTS_CAPTURED_REQUIRES_OWNER_REVIEW`. O operador nunca mais confunde "nada
aconteceu" com "commitou e a janela nao fechou". O runbook (`:251`) proibe `DELETE` manual de forma
explicita.

## F4 - gate de login HTTP do owner - **FECHADO**, e o desenho do transporte esta correto

O gate roda **depois** do provision e **antes** de declarar sucesso, exigindo
`login_code == 200` em `POST /auth/login` **e** `me_code == 200` em `GET /api/me`, com `http_stderr`
vazio. Em qualquer falha ele le o estado commitado e, se for `(1,1,0)`, tenta o rollback credenciado -
o que **resolve a tensao com L1** que eu havia apontado, porque `OWNER_*` ainda estao no ambiente
(o `unset` final so ocorre apos o gate).

**Ataquei o transporte da senha nesta nova superficie e ele esta correto:**
- `http-body` roda o helper **no host**, nao via Compose, e sua **stdout vai direto para o stdin do
  `curl`** por pipe: `"$ORQ17_HELPER_BIN" http-body | curl … --data-binary @-`. A senha **nao** vai a
  arquivo e **nao** vai a `argv`;
- a substituicao de comando captura **somente** `%{http_code}`, porque `--output /dev/null`;
- `http-body` **nao e um oraculo de divulgacao**: ele nao le segredo de lugar nenhum, apenas
  re-emite o que recebeu por stdin. Quem pudesse executa-lo ja teria de possuir a credencial;
- o `env -u OWNER_EMAIL -u OWNER_PASSWORD` e mantido tambem nesta invocacao.

## F5 - as duas travas de uma linha - **FECHADAS**

```sh
case "$(command -V printf 2>/dev/null)" in *builtin*) ;; *) printf '%s\n' 'E_PRINTF_EXTERNAL'; exit 1 ;; esac
ulimit -c 0
```
Ambas nas **linhas 5-9**, isto e **antes** de qualquer expansao de `OWNER_*`. Fecha o vazamento por
`argv` de um `printf` externo e impede core dump com a senha no heap.

## Condicoes registradas (nenhuma bloqueante)

| # | condicao | por que importa |
|---|---|---|
| **C1** | os recibos de custodia e de breakglass sao controle **de governanca, nao criptografico**: o hash esperado chega por **variavel de ambiente do proprio operador**, logo um recibo auto-assinado e possivel. So e forte se o hash vier do owner por canal proprio | autorizacao de F1/F2 |
| **C2** | o gate F4 **emite uma sessao real de owner**: cookie `multica_auth` com JWT de **30 dias** (`defaultAuthTokenTTL`), gravado em jar `0600` e removido em todos os caminhos - mas o JWT e **stateless e nao revogavel**. O owner precisa aceitar que um token de 30 dias existiu brevemente em disco; opcionalmente chamar `/auth/logout` por higiene de cookie (sem efeito no servidor) | exposicao residual |
| **C3** | `check_receipt` encadeia `test` sem `&&`; hoje **falha fechado** porque o valor de retorno e o do `sha256sum` comparado (que falha se o arquivo nao existe), mas depende de composicao. Encadear com `&&` torna a intencao explicita | robustez do gate de autorizacao |
| **C4** | `expectedOwnerEmail` esta **fixo no fonte** do helper. E um controle positivo (impede provisionar/apagar outra identidade), mas trocar o owner exige **recompilar** e refazer o pin de hash | operacional |

Observacao adicional, sem acao: `checkDBTuple` agora tambem exige `cc.TLSConfig == nil`, ou seja
**proibe** TLS na conexao ao Postgres. Como o trafego e container-para-container na rede privada do
Compose e o objetivo e determinismo da tupla, nao vejo risco - mas registro que a escolha e explicita.

## O que continua valendo do meu parecer anterior

- Exposicao de credencial e corrupcao de usuarios/memberships seguem **PASS**, com todos os controles
  que ja havia verificado (banco virgem exigido, catalogo de FK por igualdade exata + gate de forma,
  `targetReferenceCounts` cobrindo `member` com `CASCADE`, `member` congelado em `SHARE` na janela
  inteira, `member=0` verificado 4 vezes, ausencia de qualquer chamada de log no helper).
- **A A5.1 literal continua inexequivel** e a exclusao de `member` do lock da transacao de mutacao
  continua **correta** - repito aqui para que ninguem gere BLOCK cruzado por leitura literal.

## Nao-afirmacoes

- **Nao executei nada**: nem helper, runner, Docker/Compose, SQL, `curl`, login, `asm-exec`, nem
  chamada AWS. Nao usei o SSH ao ORQ1 nesta rodada. Nao mutei AWS, segredo, DB ou quadro.
- Revisei o **texto** dos `.txt`; **nao compilei** o helper, **nao** verifiquei o `source-closure`, e
  **nao** confirmei que o `.txt` corresponde ao binario que seria montado na janela.
- **Nao recebi hash do General-TL** para conferir; registrei os hashes que eu mesmo medi. Se o hash
  oficial divergir dos quatro acima, esta revisao se refere a outra revisao dos arquivos e precisa ser
  repetida.
- Nao avaliei o arquivo de teste (fora do escopo concedido) e portanto **nao** afirmo ausencia de
  falso-verde nos testes; nao rodei `go build`/`go vet` (as ferramentas existem, mas o escopo aceito e
  revisao dos F1-F5).
- Nao li `handler.ValidatePassword`, `PasswordMaxBytes` nem o store nesta rodada; assumi contratos pelo
  uso, como na revisao anterior.
- Nao verifiquei o schema real do Postgres do ORQ1: avaliei o catalogo de FK como **mecanismo**.
- Nao avaliei os pins de build A2.x nem nada das tarefas pausadas ORQ-42/44/13, conforme a decisao do
  GATE 0.
- Nao alterei assignee, nao postei comentario, nao criei card.
