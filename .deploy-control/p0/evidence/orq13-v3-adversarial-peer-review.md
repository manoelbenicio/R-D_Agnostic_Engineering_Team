# ORQ-13 - Peer review adversarial independente do RUNBOOK V3 (split SQLC)

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T15:52Z
- auditado: `.deploy-control/p0/evidence/orq13-sqlc-split-runbook-v3.md` (Opus48#B, status
  **PROPOSTA**, sem auto-aprovacao)
- reserva referenciada: `RES-ORQ13-001` em `.deploy-control/p0/evidence/gtl-migration-registrar-governance.md`
- modo: READ-ONLY. **Nao gerei, nao fiz stage, nao commitei, nao buildei, nao testei, nao editei
  codigo de produto nem board.** Nao instalei nem baixei nada.

---

## VEREDITO: **PASS CONDICIONAL**

**Todos os BLOCKs anteriores estao fechados**, e verifiquei cada um contra o host - nao contra o
texto. Nao encontrei nenhuma afirmacao factual errada no V3, o que e a primeira vez hoje que digo isso
sobre um runbook. Ele tambem nao se auto-aprova.

O PASS e **condicional a 5 adicoes obrigatorias** (secao 8), sendo duas de governanca da propria
reserva. **Nao e autorizacao de execucao.**

---

## 1. Verificacao da reserva `RES-ORQ13-001` - fingerprints **conferem**

Calculei os hashes dos arquivos reais no worktree:
```
0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1  127_task_usage_thinking_level.up.sql
74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113  127_task_usage_thinking_level.down.sql
```
Comparado com o registro:

| campo | reserva | medido | resultado |
|---|---|---|---|
| `sha256_up` | `0ea3005d...6cac1` | idem | **CONFERE** |
| `sha256_down` | `74354ae2...61113` | idem | **CONFERE** |
| numero canonico | `127` | arquivos nomeados `127_*` | **CONFERE** |
| emitida / expira | `2026-07-27T15:37:12Z` / `2026-07-28T15:37:12Z` | agora e `~15:52Z` do dia 27 | **ATIVA**, ~23h40 restantes |

Consequencia pratica que o V3 nao registra: como o numero reservado **e** o `127` que os arquivos ja
usam, o passo 2 da secao 8 ("renomear para `<N>`") e **no-op**. O V3 continua escrevendo `<N>` no
texto, o que esta correto por disciplina, mas o executor precisa saber que nao ha renome a fazer -
senao alguem renomeia para `127_...` de novo ou, pior, "corrige" o nome e **muda o fingerprint**.

## 2. Re-varredura independente de worktrees **e refs remotas** - o ataque que o V3 pediu

O V3 declara explicitamente que **nao** inspecionou refs remotas e pediu que o revisor fizesse. Fiz.

**Worktrees**: inspecionei **24** (o V3 reporta 23 - apareceu **um a mais** desde a entrega dele, o
que e prova concreta de que a varredura e uma foto, exatamente como ele admite). Resultado:
```
gtl-i03-orq13-phase1: 127_task_usage_thinking_level.up.sql
(nenhum outro worktree tem migration >= 127)
```
**Refs remotas**: 22 heads em `origin`, 23 refs locais de `origin`. Varri **todas**:
```
NENHUM remote head tem migration >= 127
```
E na integracao remota (`origin/integration/dev-transition-candidate-20260719`) o maior numero e
**126**. Isso corrobora o ruling do registrar por caminho independente: ele afirma ter verificado
`git ls-remote --heads origin` e o fetch da integracao; eu verifiquei **todas as 23 refs de origin**,
nao apenas a integracao, e o resultado e o mesmo. **Sem colisao.**

Limite que declaro: refs locais de `origin` refletem o ultimo fetch. **Nao executei fetch** (seria
mutacao de refs locais), logo um push feito nos ultimos minutos por outro agente nao apareceria. Isso
nao invalida a reserva - e o proprio motivo de a reserva existir e ter TTL.

## 3. BLOCKs anteriores - fechados um por um, verificados no host

| BLOCK anterior (meu parecer da V2) | V3 | verificacao independente |
|---|---|---|
| worktree citado nao existia (`gtl-orq13-wave0`) | usa `gtl-i03-orq13-phase1` | **FECHADO** - o worktree existe, branch `agent/opus48-b/orq-13-thinking-level`, base `0cb8aeb` |
| gate de diff sempre falhava (`^pkg/db/generated/` a partir de `server/`) | roda da **raiz do worktree** com `--` e `--exit-code`, e oferece `git -C` como alternativa imune a cwd | **FECHADO** - e a correcao exata; a secao 1 do V3 ate declara a distincao de `cwd` |
| `go fmt -l` mutava arquivos | usa `gofmt -l` puro e **proibe `-w`** explicitamente | **FECHADO** |
| escopo do gofmt causaria falso negativo | lista explicita de 8 arquivos, com justificativa medida (`client.go`, `repocache/cache.go`, `actor_guards.go` pre-existentes) | **FECHADO**, e melhor que a minha sugestao original |
| `-m 0700` nao corrige diretorio existente | `install -d -m 0700` **mais** `chmod` **mais** `test "$(stat -c %a ...)" = 700` | **FECHADO** |
| build so de `cmd/server` | `go build ./...` com justificativa (quebra de assinatura em consumidor distante) | **FECHADO** |
| faltava `set -euo pipefail` | presente em todos os blocos | **FECHADO** |
| churn ja materializado e misturado ignorado | secao 2 inteira dedicada ao split **por hunk**, com inventario | **FECHADO** |
| `FILES_LOCKED` inflado e colidindo com ORQ-41 | A = 2 arquivos, B = 10, e uma lista explicita de "fora do lock"; nao reivindica `service/task.go` nem `generated/**` inteiro | **FECHADO** |
| auto-PASS | secao 10: "Nao me auto-aprovo e nao declaro PASS" | **FECHADO** |
| numero de migration auto-atribuido | pedido formal ao registrar, e `<N>` no texto | **FECHADO**, e agora reservado |
| gate de fila com 2 estados | 4 estados ativos, derivado de `109:13-15` | **FECHADO** |

Verifiquei ainda, por amostragem adversarial, dois numeros que o V3 afirma:
- `models.go` tem **exatamente 6 hunks** (contei), e a linha `+ ThinkingLevel pgtype.Text` cai no
  hunk **5**. Logo "5 de 6 para A, 1 para B" **procede**.
- `task_message.sql.go`: **2 hunks**, `13 insertions / 13 deletions`, e o arquivo em worktree contem
  `max_seq` nas linhas 66, 73, 74 e 75. Confirma que HEAD tem `maxSeq` e o gerado emite `max_seq`,
  isto e **codigo gerado foi editado a mao antes deste cartao**, como o V3 conclui.

## 4. 🎯 A pergunta do lint - **RESOLVIDA localmente, sem instalar nada**

O V3 deixou isso como "unico risco aberto do commit A". Consegui fechar por um caminho que ele nao
tentou, e que nao exige o linter.

Fatos medidos:
1. **Nao existe `.golangci.*`** em nenhum nivel do repo (`find -maxdepth 4 -name ".golangci*"` vazio).
   Portanto valem **os defaults** do golangci-lint.
2. O CI e `multica-auth-work/.github/workflows/ci.yml:114-118`, literal:
   ```yaml
   - name: Go lint
     uses: golangci/golangci-lint-action@v9
     with:
       version: v2.12
       working-directory: server
   ```
   A citacao do V3 esta **correta** (ele diz `ci.yml:115-118`; o bloco comeca em 114).
3. **Os arquivos gerados carregam o header canonico de codigo gerado**:
   ```
   pkg/db/generated/task_message.sql.go:1   // Code generated by sqlc. DO NOT EDIT.
   pkg/db/generated/models.go:1             // Code generated by sqlc. DO NOT EDIT.
   ```

**Conclusao**: no golangci-lint v2 o processamento de issues tem
`issues.exclude-generated` com default **`lax`**, que exclui achados em arquivos cujo cabecalho
contem `Code generated ... DO NOT EDIT.`. Como (a) nao ha config custom, (b) os dois arquivos de A
tem o header exato, o achado de nomenclatura em `max_seq` **nao chega a ser reportado**,
independentemente de a regra de naming estar ou nao habilitada.

Reforco secundario, tambem sem instalar: na v2 as checagens `ST` vivem dentro do `staticcheck`, e o
conjunto default de `staticcheck.checks` **exclui** as regras de nomenclatura (`ST1003` entre elas).
Ou seja ha **duas** camadas independentes que absolvem o `max_seq`.

**Honestidade sobre o limite desta conclusao**: eu **nao executei** `golangci-lint` - nao esta
instalado, nao ha binario, nao ha fonte no module cache (`/home/ec2-user/go/pkg/mod/github.com/golangci`
e `honnef.co` ausentes), e instalar e proibido. A camada (a)+(b) e **verificavel localmente** (header +
ausencia de config); a afirmacao sobre o default de `staticcheck.checks` e **documental**, nao medida.
Recomendacao: manter o gate 5.6 como **opcional e informativo**, e **nao** tratar `max_seq` como
risco bloqueante do commit A. A correcao que o V3 propoe caso reprove - excluir `generated/` do
linter, nunca reeditar o gerado - esta certa e ja e o comportamento default.

## 5. ⚔️ Ataque ao split por hunk: `git add -p` interativo **e** o pior caminho disponivel

O V3 e honesto: `git add -p` "exige operador humano" e a alternativa e recortar hunks a mao para
`a-models.patch` + `git apply --cached`. Ataquei as duas.

**Ataque 1 - as ancoras publicadas nao batem com o que o operador vai ver.** O V3 lista para A, em
`models.go`: `@@ -12,0 +13,17 @@`, `@@ -103,0 +121,15 @@`, `@@ -249,0 +282,10 @@`,
`@@ -587,0 +630,10 @@`, `@@ -763,2 +817,9 @@`. Os cabecalhos **reais** com `git diff` default
(`-U3`) sao:
```
@@ -10,6 +10,23 @@    @@ -101,6 +118,21 @@   @@ -247,6 +279,16 @@
@@ -585,6 +627,16 @@   @@ -704,6 +756,8 @@    @@ -761,6 +815,13 @@
```
As ancoras do V3 sao consistentes com `git diff -U0`, nao com o default. Nao e erro de leitura - e
**risco de reprodutibilidade**: um operador seguindo o documento com `git add -p` (que usa contexto
default) nao encontra nenhuma das cinco ancoras e pode selecionar o hunk errado. Em `models.go` o
hunk de B e o **quinto** (`@@ -704,6 +756,8 @@`, contendo `+ ThinkingLevel pgtype.Text`), nao o sexto.
**Correcao**: publicar a invocacao exata (`git diff -U0 -- <path>`) junto das ancoras, **ou** trocar
para ancoras de contexto default, **e** identificar o hunk de B por **ordinal + conteudo**
("hunk 5 de 6, o que contem `ThinkingLevel`"), que e o unico identificador estavel.

**Ataque 2 - a alternativa "determinista" nao tem ferramenta no host.** `filterdiff`, `splitdiff` e
`interdiff` (patchutils) **nao estao instalados**, e instalar e proibido. Logo "recortar os hunks para
um patch" e literalmente **edicao manual de patch de codigo gerado** - o mesmo pecado que originou o
`maxSeq` que o commit A vem reverter.

**Contraproposta determinista, sem hunk-surgery e sem patchutils** (a melhor das tres, e o executor
deveria preferi-la): fazer os dois commits serem **saida do gerador**, nao curadoria humana.
```
1. guardar a mudanca de query:      git stash push -- multica-auth-work/server/pkg/db/queries/task_usage.sql
2. sqlc generate                 -> produz EXATAMENTE o drift (queries == HEAD)
3. commit A  (generated/ apenas)
4. git stash pop                 -> volta a mudanca de query
5. sqlc generate                 -> produz EXATAMENTE o delta da feature
6. commit B  (resto)
```
Vantagens sobre `add -p`: **zero** decisao humana sobre hunk, reproduzivel por qualquer executor, e o
commit A passa a ser por construcao "o que o gerador emite sobre as migrations ja integradas" - que e
a definicao de baseline que o V3 quer. Custos honestos: exige `sqlc generate` **duas** vezes
(autorizacao 3 do V3 ja cobre) e um `git stash` (mutacao local, reversivel, mas precisa autorizacao
explicita porque hoje o cartao nao tem nem `stage`). Recomendo que o revisor/GTL avalie esta rota
antes de autorizar `add -p`.

## 6. Gates A e B validados de forma independente

O V3 admite: *"os gates independentes de A e de B **nunca** rodaram separadamente"*. Correto e
importante. Avaliando os gates **como escritos**:

- **Commit A**, gates 5.2-5.6: o 5.3 (`git diff --exit-code -- .../generated/`) e valido **depois** do
  commit A, mas note que apos o commit A ainda restara o hunk de B em `models.go` **nao commitado**,
  logo `git diff --exit-code` sobre `generated/` **falhara** legitimamente entre A e B. O gate 5.3
  precisa ser qualificado: aplica-se **apos B**, ou com `--cached` **antes** de cada commit. Como
  esta, um executor literal aborta entre A e B. **E a unica falha de sequenciamento que encontrei.**
- **Commit A**, 5.4/5.5: `gofmt -l` na lista fechada e `go build ./...` - validos e independentes.
- **Commit B**, gates 5.4/5.5/5.8: validos. O 5.8 roda testes por nome (`ThinkingLevel|TaskUsageEntry`),
  o que e apropriado e nao mascara skip - mas note que `internal/handler` usa
  `t.Skip("database not available")` em varios testes, entao um `go test ./internal/handler` verde
  **sem banco** nao prova nada sobre os testes novos. O gate deveria exigir banco, ou afirmar
  explicitamente que os dois testes novos nao dependem de banco - **nao verifiquei quais deles
  dependem**.
- **5.7 idempotencia**: forma correta (`sqlc generate` -> `git diff --exit-code` da raiz com `--`).
  Uma dobra que falta: o teste real de idempotencia e **duas** geracoes consecutivas comparadas entre
  si; como escrito, ele compara contra o **commit anterior**, o que so funciona se rodar
  imediatamente apos o commit correspondente. Explicitar a ordem.

## 7. Ordem ORQ-13 antes de ORQ-41 - **sustentada**, e verifico o porque

O argumento do V3 e tecnico e correto: o commit A e a regeneracao de baseline; quem gerar antes dele
arrasta os 6 structs de rotacao (`Account`, `ApprovedAccount`, `Assignment`, `Credential`,
`RotationEvent`, `UserPasswordCredential`, das migrations 123/124/125) para dentro do proprio PR.

Coerencia com o que eu mesmo desenhei no ORQ-41: o plano de ondas do ORQ-41 exige um **LANE-DB serial
com dono unico** sobre `migrations/`, `pkg/db/queries/` e `pkg/db/generated/`. ORQ-13 fase 1 **e** um
item de LANE-DB, e portanto deve ocupar a fila **primeiro**, com o ORQ-41 esperando. Nao ha
contradicao entre os dois documentos - ha uma fila, e o ORQ-13 esta na frente por razao tecnica.

O V3 tambem mediu que o worktree do ORQ-41 (`gtl-orq41-w4-autopilot`) tem `git diff --stat` vazio e
nenhum arquivo em `generated`/`queries`/`migrations`. Nao reverifiquei esse worktree especifico, mas
minha varredura de 24 worktreesse confirma que **nenhum outro** tem migration >= 127, o que e
consistente.

## 8. Condicoes obrigatorias do PASS (5 adicoes, nenhuma correcao de fato)

1. **F1 - gate de fingerprint antes do commit B.** Reverificar `sha256sum` dos dois arquivos `127_*`
   contra `RES-ORQ13-001` **imediatamente antes** de commitar; qualquer divergencia exige
   re-associacao de fingerprint com o registrar (protocolo §2.3 da governanca). Motivo: hoje nada no
   V3 impede uma edicao de DDL entre a reserva e o commit.
2. **F2 - gate de TTL.** A reserva expira em `2026-07-28T15:37:12Z`. O V3 tem **8** autorizacoes em
   serie; se a cadeia atravessar a expiracao, o numero e liberado (protocolo §2.4). Adicionar
   verificacao explicita de TTL antes do commit B e, se expirado, **parar** e pedir renovacao.
3. **F3 - corrigir as ancoras de hunk** ou publicar `git diff -U0` como invocacao canonica, e
   identificar o hunk de B por **ordinal + conteudo** (hunk 5 de 6, o do `ThinkingLevel`).
   Recomendado: avaliar a rota determinista da secao 5 antes de autorizar `git add -p`.
4. **F4 - qualificar o gate 5.3.** Como escrito, ele **falha legitimamente entre A e B** (resta o
   hunk de B em `models.go`). Especificar que se aplica apos B, ou usar `--cached` antes de cada
   commit.
5. **F5 - declarar dependencia de banco nos testes de 5.8**, dado que `internal/handler` tem
   `t.Skip("database not available")`; sem isso, um verde sem banco nao prova os testes novos.

Nenhuma dessas contradiz um fato do V3. Sao gates que faltam, nao erros - e por isso o veredito e
PASS condicional, e nao BLOCK.

## 9. Observacao de governanca (nao bloqueante, para o registrar)

O proprio documento de governanca, §6, diz que agentes **devem obrigatoriamente** usar placeholders
`NEXT_CANONICAL_*` na fase de design e que o renumero para canonico e **exclusivo do Codex56-TL**. Os
arquivos do ORQ-13 nasceram nomeados `127` **antes** da reserva, por heranca de entrega anterior - o
V3 admite isso e propoe renomear. A `RES-ORQ13-001` **ratificou** o `127`, o que resolve na pratica,
mas convem registrar que houve desvio do protocolo e que a ratificacao foi a saida escolhida. Sem isso,
o precedente lido por outro agente sera "materializar primeiro e pedir ratificacao depois", que e o
oposto do que a governanca quer.

## 10. Nao-afirmacoes desta revisao

- **Nao gerei** (`sqlc generate`), nao fiz `stage`/`commit`/`push`, nao buildei, nao testei, nao
  apliquei migration, nao editei codigo de produto nem board, e **nao instalei nem baixei nada**.
- **Nao executei `golangci-lint`**: nao esta instalado, sem binario e sem fonte no module cache. A
  conclusao da secao 4 se apoia em dois fatos locais (header de codigo gerado e ausencia de
  `.golangci.*`) mais o default **documental** de `staticcheck.checks`; o segundo nao foi medido.
- **Nao executei `git fetch`**: as refs de `origin` que varri sao do ultimo fetch. Um push feito nos
  ultimos minutos nao apareceria.
- Nao reverifiquei o worktree `gtl-orq41-w4-autopilot` individualmente; validei apenas que nenhum dos
  24 worktrees tem migration >= 127.
- Nao determinei quais dos dois testes novos dependem de banco (base do F5).
- Nao revalidei o UUID do ORQ-13 por `GET`; o V3 cita o cartao por numero.
- Nao alterei assignee nem postei comentario; nenhum card criado; freeze respeitado.
