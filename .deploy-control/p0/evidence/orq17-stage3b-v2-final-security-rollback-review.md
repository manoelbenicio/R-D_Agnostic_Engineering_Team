# ORQ-17 Stage3B V2 - revisao final independente de **seguranca e rollback** (READ-ONLY)

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T17:02Z
- artefato V2 revisado (arquivos de 16:42-16:49):
  `orq17-stage3b-v2-bootstrap-helper.go.txt`, `orq17-stage3b-v2-sealed-runner.sh.txt`,
  `orq17-stage3b-v2-bootstrap-helper_test.go.txt`, `orq17-stage3b-v2-helper-go.mod.txt`,
  `orq17-stage3b-v2-helper-go.sum.txt`, `orq17-stage3b-v2-source-closure.sha256`,
  `orq17-stage3b-v2-acceptance-matrix.md`
- **escopo, por instrucao**: **somente** riscos de (1) exposicao de credencial, (2) corrupcao de
  usuarios/memberships, (3) lockout. **Nao reabro** preferencias menores, estilo, nomeacao, ou os
  itens A2.x de pin de build - varios deles seguem abertos e **nao** sao objeto deste parecer.
- modo: **READ-ONLY**. Nao executei o helper, o runner, `asm-exec`, Docker/Compose, SQL, login, nem
  qualquer chamada AWS. Nao li segredo. Nao mutei quadro.

---

## VEREDITO: **BLOCK** - por **dois** defeitos de rollback/lockout. Exposicao de credencial e corrupcao: **PASS**.

O V2 e substancialmente bom nos dois eixos mais perigosos. O que reprova e estreito, mecanico e
corrigivel em poucas linhas:

- **L1** - o rollback e **condicionado a posse da senha**: perdida a senha, **nao existe** caminho de
  reversao nem de reprovisionamento;
- **L2** - existe um caminho que **reporta falha depois de um commit bem-sucedido**, sem compensar.

---

## 1. Exposicao de credencial - **PASS** (com 2 travas exigidas, 1 linha cada)

Verificado no codigo, nao aceito por confianca:

| vetor | resultado |
|---|---|
| valor em `argv` | **fechado**. O runner passa por **stdin**: `printf '%s\0%s\0' "$OWNER_EMAIL" "$OWNER_PASSWORD" \| compose`. Nenhum valor em flag. O helper le `os.Stdin`. |
| valor no ambiente do Compose/Docker | **fechado**. `env -u OWNER_EMAIL -u OWNER_PASSWORD docker compose …` remove as duas do filho **e** da interpolacao do Compose. |
| valor em log/stdout/stderr | **fechado**. O helper **nao tem nenhuma chamada de log**. Todos os erros sao `opError` com **literais fixos** (`E_*`); erro desconhecido colapsa em `E_INTERNAL`. Nenhum `opError` interpola email, senha ou erro do Postgres. O sucesso imprime linha fixa, sem identidade. |
| DSN em erro | **fechado**. `checkDBTuple` compara host/porta/banco/usuario e devolve `E_DATABASE_TUPLE` **sem imprimir o DSN**. |
| arquivos de captura | **fechado o suficiente**. `stdout/stderr/expected` em diretorio `0700`, arquivos `0600`, removidos por `trap` em `EXIT HUP INT TERM`. `mkdir -m 0700` **sem `-p`** falha se o diretorio existir - e o `trap` e instalado **depois** do `mkdir`, portanto uma falha ali nao dispara limpeza em diretorio alheio. Correto. |
| limites de entrada | **fechado**. `io.LimitReader(r,1024)`, email <=320, senha <=`handler.PasswordMaxBytes`, exigencia de **EOF** apos o 2o NUL (`E_TRAILING_INPUT`), e `clear()` dos buffers em todos os caminhos. |

### Trava exigida S1 - provar que `printf` e **builtin**
Se o `/bin/sh` da janela resolver `printf` para `/usr/bin/printf`, a senha vai para o **`argv` daquele
processo** e fica legivel em `/proc/<pid>/cmdline`. O runner nao verifica isso. Uma linha fecha:
```sh
case "$(command -V printf 2>/dev/null)" in *builtin*) ;; *) printf '%s\n' 'E_PRINTF_EXTERNAL'; exit 1;; esac
```
(usar `command -V` **antes** de qualquer expansao das variaveis). Sem essa trava, todo o desenho
"nunca em argv" depende de uma propriedade do shell que ninguem mediu.

### Trava exigida S2 - proibir core dump
`password := string(passwordBytes)` cria uma **string Go imutavel**; o `defer` que faz `password = ""`
apenas solta a referencia - os bytes permanecem no heap ate o GC. Isso e aceitavel para um processo
efemero, **desde que** nao haja despejo de memoria. Uma linha no runner:
```sh
ulimit -c 0
```
Sem isso, um `SIGSEGV`/panic com core habilitado grava a senha em disco fora do controle do `trap`.

### Residual que **precisa** de aceite explicito (nao e defeito novo, e uma precondicao irrealista)
A matriz declara em **A1.5** que `/proc/<pid>/environ` do shell filho e legivel por **mesmo-UID e
root**, e exige "janela selada e **nenhum outro processo do UID do operador**". Observo que, no host
real, essa precondicao e **falsa por construcao** enquanto a frota de agentes roda como `ec2-user`.
E a janela de exposicao do V2 e **longa**: as variaveis ficam no ambiente do shell durante **todo** o
`docker compose run` (criacao do container + execucao), nao por milissegundos. Portanto A1.5 nao pode
ser marcada como satisfeita por prosa: exige **aceite escrito do owner** com a janela declarada, ou
uma janela em que nenhum outro processo `ec2-user` esteja ativo. Isso e decisao do owner, nao minha.

## 2. Corrupcao de usuarios / memberships - **PASS**, e com defesa acima do exigido

Ataquei especificamente cascata e escrita concorrente. **Nao encontrei caminho de corrupcao.**

1. **Provision so opera em banco virgem.** Dentro da transacao Serializable, `verifyGlobal(tx,0,0,0)`
   exige `user=0`, `user_password_credential=0`, `member=0`; qualquer linha preexistente aborta com
   `E_NOT_EMPTY`. Logo o V2 **nao pode** alterar usuario existente - o pior caso e recusa.
2. **Catalogo de FK conferido por igualdade exata de conjunto.** `loadUserFKs` extrai as FKs que
   apontam para `public.user(id)` com a acao `ON DELETE` decodificada de `confdeltype`, e `compareFKs`
   exige **igualdade ordenada** com os 16 pares esperados. Uma tabela nova referenciando `user` faz o
   passo falhar (`E_FK_CATALOG`) em vez de cascatear silenciosamente. Ha ainda o gate de **forma**:
   `count(*)` total versus `count(*) FILTER (array_length(conkey,1)=1)`, que reprova (`E_FK_SHAPE`) se
   surgir FK multi-coluna que a consulta nao representa. Isso e exatamente a defesa certa contra o
   risco de cascata.
3. **Nenhum DELETE sem contagem previa de referencias.** `targetReferenceCounts` percorre **todas** as
   16 FKs (pulando so `user_password_credential`, que e o alvo legitimo) e exige `count = 0`, inclusive
   para **`member`** - cuja FK e `ON DELETE CASCADE`. Ou seja, **nenhuma membership pode ser apagada
   por cascata**: se existir uma, o rollback recusa com `E_ROLLBACK_REFERENCED`.
4. **`member` congelado durante toda a janela.** A transacao-guarda toma
   `LOCK TABLE agent_task_queue, member IN SHARE MODE`, que conflita com `RowExclusive` de
   `INSERT/UPDATE/DELETE`, e **so** e liberada no `guard.Commit` final.
5. **`member=0` verificado tres vezes ou mais**: na guarda (T1), antes da criacao
   (`verifyGlobal(0,0,0)`), antes do commit (`verifyGlobal(1,1,0)`) e **depois** do commit pela guarda.
   Satisfaz A5.2.
6. **Gate de fila correto**: tabela `agent_task_queue` e os **4** estados ativos
   (`queued, dispatched, running, waiting_local_directory`), medidos duas vezes com 3 s de intervalo.
   Confere com a `CHECK` da migration 109.

### Ponto que um revisor apressado marcaria como violacao de **A5.1** - e nao e
A matriz A5.1 exige `member` travado **junto** com `user`/`user_password_credential` no mesmo
`SHARE ROW EXCLUSIVE`. O V2 **exclui** `member` do lock da transacao de mutacao, com comentario
explicito. **O V2 esta certo e a A5.1, como literal, e inexequivel**: a guarda ja detem `SHARE` sobre
`member` em **outra conexao**, e `SHARE ROW EXCLUSIVE` **conflita** com `SHARE` - pedir esse lock faria
a transacao de mutacao bloquear contra a propria guarda e morrer em `lock_timeout` (`E_TX_LOCK`). A
**intencao** de A5.1 (nenhuma membership criada ou alterada na janela) e atendida por um mecanismo
**mais forte**: o `SHARE` cobre a janela inteira, incluindo a verificacao pos-commit. Registro isso
para impedir um BLOCK por leitura literal.

## 3. Lockout e rollback - **BLOCK**

### L1 - o rollback e **condicionado a posse da senha**; sem senha, nao ha reversao **nem** reprovisionamento
`rollbackProvision` exige, **antes** de qualquer DELETE:
```go
identity, err := handler.NewPasswordAuthProvider(store).Login(ctx, email, password)
if err != nil || identity.UserID != user.ID { return opError("E_ROLLBACK_VERIFY") }
```
Como controle contra apagar a linha errada, isso e bom. O problema e o estado resultante:

- se a senha for perdida, digitada diferente, ou o segredo rotacionado entre provision e rollback, o
  `rollback` **recusa para sempre** (`E_ROLLBACK_VERIFY`);
- e o `provision` tambem recusa para sempre, porque o banco nao esta mais vazio (`E_NOT_EMPTY`);
- resultado: **conta de owner inutilizavel, sem caminho de reversao nem de recriacao** pelo
  ferramental aprovado. A saida seria `DELETE` manual - **fora** de todas as guardas de FK, lock e
  contagem que este helper existe para impor. Ou seja: o desenho empurra o operador para a acao mais
  perigosa exatamente no pior momento.

**Correcao exigida (pequena):** (a) exigir **prova de custodia da senha** antes do provision (owner
confirma que o valor esta guardado no gerenciador de segredo, nao no terminal); e (b) um modo
`rollback-breakglass` que dispense o `Login`, exija **autorizacao escrita do owner** e **mantenha
todas** as outras guardas - catalogo de FK, `targetReferenceCounts`, locks, `member=0` e as contagens
finais `(0,0,0)`. Sem (b), a reversibilidade que o Stage3B promete nao existe em um cenario plausivel.

### L2 - `E_GUARD_COMMIT`: falha reportada **depois** de um commit bem-sucedido, sem compensacao
Em `execute`, a ordem e: `provision(...)` **commita** a transacao de mutacao -> verificacao pos-commit
pela guarda -> `guard.Commit`. Se **este ultimo** falhar:
```go
if err := guard.Commit(ctx); err != nil { return opError("E_GUARD_COMMIT") }
```
o helper sai com **codigo 1**, o runner imprime `E_COMPOSE_OR_HELPER`, e **o usuario e a credencial
existem e estao commitados**. Diferente do caminho de pos-verificacao - que chama `compensate` e
distingue `E_POST_VERIFY_ROLLED_BACK` de `E_POST_VERIFY_ROLLBACK_FAILED` - aqui **nao ha compensacao
nem token distintivo**.

Consequencia operacional: o operador ve falha, conclui "nao provisionou" e tenta de novo; o segundo
provision falha com `E_NOT_EMPTY`, o que parece contradicao e convida a intervencao manual no banco -
de novo, fora das guardas. O mesmo vale para o rollback: `verifyGlobal(guard,0,0,0)` roda **depois** do
`Commit` do rollback; se ele falhar, o DELETE ja esta commitado e o resultado e um `E_*` de falha sobre
um estado ja alterado.

**Correcao exigida (pequena):** (a) tokens distintos e inequivocos para "mutacao **commitada**, janela
nao encerrada" - por exemplo `E_GUARD_COMMIT_AFTER_COMMIT` - de modo que o operador **nunca** confunda
com "nada aconteceu"; e (b) um passo obrigatorio no runbook: **em qualquer falha, ler as contagens
`(user, credential, member)` antes de qualquer segunda acao**, e proibir explicitamente `DELETE`
manual. Idealmente (c) chamar `compensate` tambem neste caminho, com a mesma distincao de
"compensado" vs "compensacao falhou".

### L3 - `HTTP_LOGIN=NOT_CALLED` e honesto, mas **nao existe gate** que prove que o owner consegue entrar
O helper prova o credencial pelo **provider real** dentro da transacao
(`handler.NewPasswordAuthProvider(store).Login`), e o `HTTP_LOGIN=NOT_CALLED` no stdout e uma
declaracao correta e louvavel. Mas ela nao fecha o risco de lockout: pelas medicoes do proprio ORQ-17,
`/auth/login|google|logout` sao rotas de **backend** enquanto `/auth/callback` e a unica `/auth/*` do
Next.js - ha colisao de prefixo. Um `Login` de provider bem-sucedido **nao** prova login pelo
navegador (cookie `multica_auth`, CSRF, assinatura JWT, origem).

**Correcao exigida:** um gate de **login HTTP pelo owner** apos o Stage3B, executado **antes** de
declarar sucesso e **antes** de descartar a senha - e a janela de rollback deve permanecer aberta ate
esse gate passar. Note a tensao com **L1**: se a senha for descartada cedo, perde-se o rollback; o
runbook precisa ordenar isso explicitamente.

## 4. O que **nao** e objeto deste parecer

Nao avaliei, por instrucao: pins de build e fechamento de fonte (A2.x), `gofmt`/`go vet`, versao do Go,
igualdade de hash do binario, `-mod=readonly`, estilo, nomes de token, o teste unitario
(`_test.go.txt`), nem preferencias de formato. **Nada** neste parecer deve ser lido como aprovacao
desses itens - varios continuam abertos em outras revisoes.

## 5. Para virar PASS (5 itens, todos pequenos)

| # | correcao | eixo |
|---|---|---|
| **F1** | modo `rollback-breakglass` sem `Login`, com autorizacao escrita do owner e **todas** as demais guardas mantidas | L1 |
| **F2** | prova de **custodia da senha** exigida antes do provision | L1 |
| **F3** | token distinto para "mutacao commitada, janela nao encerrada" + passo obrigatorio de ler contagens antes de qualquer segunda acao + proibicao explicita de `DELETE` manual | L2 |
| **F4** | gate de **login HTTP do owner** antes de declarar sucesso e antes de descartar a senha, com a janela de rollback aberta ate ele passar | L3 |
| **F5** | duas linhas no runner: assercao de `printf` **builtin** e `ulimit -c 0` | exposicao |

## 6. Nao-afirmacoes

- **Nao executei nada**: nem helper, nem runner, nem `asm-exec`, nem Docker/Compose, nem SQL, nem
  login, nem chamada AWS. Nao li segredo, nao fiz `stat` de arquivo de credencial, nao mutei quadro.
- Analisei o **texto** dos artefatos `.txt`; **nao** compilei o helper, **nao** verifiquei os hashes do
  `source-closure`, e **nao** confirmei que o `.txt` corresponde ao que seria compilado na janela.
- Nao verifiquei o schema real do banco: a corretude do catalogo de FK e avaliada como **mecanismo**
  (igualdade exata + gate de forma), nao como lista conferida contra o Postgres do ORQ1.
- Nao li `handler.PasswordMaxBytes`, `handler.ValidatePassword` nem
  `NewPostgresPasswordCredentialStore` nesta revisao; assumi seus contratos pelos nomes e pelo uso.
  Se o owner quiser, isso pode ser fechado em uma segunda passada.
- Nao avaliei o arquivo de teste e portanto **nao** afirmo ausencia de falso-verde nos testes.
- A colisao de prefixo `/auth/*` citada em L3 vem das minhas medicoes anteriores do ORQ-17, **nao**
  remedidas nesta rodada.
- Nao alterei assignee, nao postei comentario, nao criei card.
