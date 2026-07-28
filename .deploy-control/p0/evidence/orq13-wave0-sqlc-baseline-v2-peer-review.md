# ORQ-13 - Peer review independente do runbook Wave 0 SQLC Baseline (V2)

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T15:32Z
- auditado: `.deploy-control/p0/evidence/orq13-wave0-sqlc-baseline-runbook-v2.md` (Antigravity,
  veredito proprio **PASS**)
- independencia: nao sou autor da V1 nem da V2, e nao emiti o BLOCK anterior desta Wave 0.
- modo: READ-ONLY. **Nao executei** `sqlc generate`, `go build`, `go vet`, `gofmt`, migration ou
  qualquer gate. Nenhum arquivo editado. Nenhuma mutacao de board, AWS, container ou unit.

---

## VEREDITO: **BLOCK**

As retificacoes conceituais da V2 estao **corretas** e eu as endosso: remover o grep de `snake_case`
como gate, declarar `pkg/db/generated/**` como excecao de lint de estilo, proibir edicao manual e
aceitar `max_seq` porque e saida determinista do gerador. Isso fecha o eixo conceitual do BLOCK
anterior.

O BLOCK e por **quatro defeitos executaveis**: **dois gates estao quebrados de forma que eu provei
empiricamente** (um sempre falha, o outro **muta arquivos**), o **worktree citado nao existe**, e o
runbook **ignora que o churn ja esta materializado** no worktree real - incluindo a migration `127`
untracked. Ha ainda uma **colisao de FILES_LOCKED com o ORQ-41**.

---

## 1. ✅ O que a V2 acertou

| item pedido na revisao | veredito |
|---|---|
| excecao de estilo para `generated/**` explicita | **PASS** - secao 1.2, declarada formalmente e com escopo correto (isenta lint de estilo, nao isenta compilacao) |
| nunca editar a mao | **PASS** - "PROIBICAO ABSOLUTA DE EDICAO MANUAL" na 1.2 e linha propria na matriz de aceite |
| `sqlc generate` duas vezes / idempotencia | **PASS conceitual** - Gate 1.1 + 1.2 e o desenho certo; o *criterio* esta furado (secao 2.3) |
| gofmt / vet / build | **PASS conceitual**, **FAIL na forma** (secao 2.2) |
| diff exclusivamente em generated | **PASS conceitual**, **FAIL na forma** (secao 2.1) |
| caches privados `0700` | **PASS parcial** - ver M1 |
| split por hunk e dono unico | **PASS parcial** - ver secao 4 |
| remocao do grep de nome de variavel | **PASS** - correto e bem justificado |

Registro que `sqlc` **existe** onde a V2 assume: `/home/ec2-user/.cache/sqlcbuild/sqlc` (e nao esta no
`PATH` por default, o que valida o `export PATH` da secao 3). `gofmt` tambem existe:
`/home/ec2-user/goroot/go/bin/gofmt`.

---

## 2. 🔴 BLOQUEADORES executaveis (provados empiricamente, sem executar os gates)

### 2.1 BLOQUEADOR 1 - o Gate 1.3 **sempre falha**, mesmo quando o diff esta 100% correto

Gate como escrito:
```bash
cd .../multica-auth-work/server
git diff --name-only | grep -v "^pkg/db/generated/" && { echo "ERRO: Diff fora de generated!"; exit 1; } || true
```
Executei `git diff --name-only` **de dentro de `multica-auth-work/server`** no worktree real do
ORQ-13, e a saida e relativa a **raiz do repositorio**, nao ao diretorio corrente:
```
multica-auth-work/server/internal/daemon/daemon.go
multica-auth-work/server/internal/daemon/types.go
multica-auth-work/server/internal/handler/daemon.go
multica-auth-work/server/pkg/db/generated/models.go
multica-auth-work/server/pkg/db/generated/task_message.sql.go
```
Logo **nenhuma** linha comeca com `pkg/db/generated/`. O `grep -v` casa **todas** as linhas, retorna
exit 0, e o gate dispara `exit 1` **inclusive quando o diff esta inteiramente dentro de
`generated/`**. E um gate que reprova o cenario correto.

Prova do contrario, com a flag que corrige:
```
$ git diff --name-only --relative
internal/daemon/daemon.go
internal/daemon/types.go
internal/handler/daemon.go
pkg/db/generated/models.go
pkg/db/generated/task_message.sql.go
```
**Correcao**: usar `git diff --name-only --relative`, ou ancorar em
`^multica-auth-work/server/pkg/db/generated/`. Recomendo `--relative`, porque nao amarra o gate ao
layout do monorepo.

Observacao adicional: o `|| true` no fim e inofensivo aqui (o `exit 1` acontece antes), mas e um
padrao perigoso num script de gate - se alguem trocar `exit 1` por `return 1`, o `|| true` engole a
falha. Sugiro `set -euo pipefail` e uma comparacao explicita em vez de `&& { } || true`.

### 2.2 BLOQUEADOR 2 - o Gate 1.4 **muta arquivos**, e contradiz o proprio runbook

Gate como escrito:
```bash
UNFORMATTED=$(/home/ec2-user/goroot/go/bin/go fmt -l pkg/db/generated)
```
`go help fmt`, saida literal na maquina:
```
usage: go fmt [-n] [-x] [packages]

Fmt runs the command 'gofmt -l -w' on the packages named
by the import paths. It prints the names of the files that are modified.
```
Ou seja `go fmt` **sempre** inclui `-w` e **reescreve** os arquivos. Consequencias:
1. o gate **edita `pkg/db/generated/**`**, violando a "PROIBICAO ABSOLUTA DE EDICAO MANUAL" da secao
   1.2 - nao e edicao humana, mas e mutacao fora do gerador, exatamente o que a V2 quer proibir;
2. como reescreve, a saida fica **vazia na segunda execucao**, entao o gate passa a dar falso verde e
   deixa de detectar codigo desformatado;
3. `-l` nao esta na usage documentada (`[-n] [-x] [packages]`), e `pkg/db/generated` e caminho de
   diretorio, nao padrao de pacote - o correto para `go` seria `./pkg/db/generated/...`.

**Correcao**: usar o `gofmt` puro, que existe no host e **nao** escreve sem `-w`:
```bash
UNFORMATTED=$(/home/ec2-user/goroot/go/bin/gofmt -l ./pkg/db/generated)
[ -z "$UNFORMATTED" ] || { echo "ERRO: nao formatado: $UNFORMATTED"; exit 1; }
```

### 2.3 BLOQUEADOR 3 - o worktree citado **nao existe**, e o criterio de idempotencia e inaplicavel no que existe

O runbook manda:
```bash
cd /home/ec2-user/workspace/worktrees/gtl-orq13-wave0/multica-auth-work/server
```
Medido: **esse diretorio nao existe.** `ls -d` retorna `No such file or directory`. O worktree real do
ORQ-13 e:
```
/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1   [agent/opus48-b/orq-13-thinking-level]
```
O bloco de gates portanto **nao roda**: o `cd` falha e, sem `set -e`, os comandos seguintes executam
no diretorio errado - potencialmente na arvore de integracao.

E o Gate 1.2 usa `git status --porcelain # DEVE ESTAR LIMPO`. No worktree real ele **nunca** estara
limpo, porque o estado atual medido e:
```
 M multica-auth-work/server/internal/daemon/daemon.go
 M multica-auth-work/server/internal/daemon/types.go
 M multica-auth-work/server/internal/handler/daemon.go
 M multica-auth-work/server/pkg/db/generated/models.go
 M multica-auth-work/server/pkg/db/generated/task_message.sql.go
 M multica-auth-work/server/pkg/db/generated/task_usage.sql.go
 M multica-auth-work/server/pkg/db/queries/task_usage.sql
?? multica-auth-work/server/internal/daemon/task_usage_thinking_level_test.go
?? multica-auth-work/server/internal/handler/task_usage_thinking_level_test.go
?? multica-auth-work/server/migrations/127_task_usage_thinking_level.down.sql
?? multica-auth-work/server/migrations/127_task_usage_thinking_level.up.sql
```
`git status --porcelain` como criterio de "idempotencia" e **estruturalmente errado** aqui: ele mistura
o churn da segunda geracao com todo o trabalho em andamento. O criterio correto isola a **segunda**
geracao:
```bash
sqlc generate --config sqlc.yaml
git add -A -- pkg/db/generated            # snapshot pos-1a geracao no index
sqlc generate --config sqlc.yaml
git diff --quiet -- pkg/db/generated || { echo "ERRO: sqlc nao e idempotente"; exit 1; }
```
Assim se compara **geracao 2 contra geracao 1**, e nao contra HEAD.

### 2.4 BLOQUEADOR 4 - a premissa de "isolar churn" ignora que o churn **ja esta materializado e misturado**

O runbook trata a Wave 0 como algo a produzir. No worktree real, `pkg/db/generated/models.go`,
`task_message.sql.go` e `task_usage.sql.go` **ja estao modificados**, junto de
`queries/task_usage.sql`, `handler/daemon.go`, `daemon/daemon.go` e `daemon/types.go`, e mais quatro
arquivos untracked, incluindo a migration **`127_task_usage_thinking_level`**.

Isso muda a natureza do trabalho: **nao e "gerar e commitar"**, e **separar por hunk um estado
ja misturado**. Em particular `pkg/db/generated/task_message.sql.go` esta modificado, e
`task_message` **nao** e alvo do ORQ-13 - o que e consistente com a tese da Wave 0 (existe drift de
baseline anterior), mas o runbook precisa dizer isso e provar a origem de cada hunk, em vez de
assumir arvore limpa. Sem isso, o "Commit 1" pode arrastar hunk de feature, e o "Commit 2" pode
arrastar hunk de baseline - exatamente o que a Wave 0 existe para impedir.

**Correcao**: a secao 3 precisa comecar por um passo 0 de **inventario do estado atual**
(`git status --porcelain` + `git diff --stat -- pkg/db/generated`), e a estrategia de split precisa
ser `git add -p` por hunk com criterio declarado de atribuicao, nao um `git add` de diretorio.

---

## 3. Reconciliacao com a `127` materializada e com o registrador central

Fatos medidos:
- `127_task_usage_thinking_level.{up,down}.sql` existem no worktree do ORQ-13 e estao **UNTRACKED**
  (`git ls-files --error-unmatch` falha).
- Na branch de integracao o maior numero e **126**, portanto a `127` **ainda nao esta reservada** de
  forma visivel para ninguem alem de quem abriu esse worktree.
- Ha pelo menos duas outras frentes com worktree aberto que provavelmente precisarao de numero:
  **ORQ-12** (`account_id` em `task_usage`) e **ORQ-21** (`accounts`/`approved_accounts`/`assignments`).

Consequencia para este runbook: a secao 2 diz que o Commit 2 leva "Migration `NEXT_CANONICAL`", o que
esta **correto e alinhado** ao principio de nao reivindicar digito. Mas o worktree **ja materializou
um digito concreto (127)**, e essas duas coisas estao em conflito silencioso. Duas saidas, e a escolha
e do registrador central, nao minha:
- **(i)** o registrador **ratifica** a `127` para o ORQ-13, e o runbook substitui `NEXT_CANONICAL` pelo
  numero ratificado, registrando a ratificacao na evidencia; ou
- **(ii)** a `127` e **renumerada** se outra frente tiver prioridade, e como o arquivo esta untracked o
  custo e baixo - `git mv` nao e nem necessario.

**Bloqueio de processo que declaro abertamente**: eu **nao sei quem e o registrador central de
migrations**, e essa mesma lacuna ja apareceu no ORQ-41. Enquanto ela existir, quatro cartoes
(ORQ-12, ORQ-13, ORQ-21 e ORQ-41) estao criando numero em worktrees isolados sem coordenacao visivel,
e o primeiro merge ganha por acidente. Isso e pre-requisito, nao detalhe.

---

## 4. Colisao de `FILES_LOCKED` com o ORQ-41

`FILES_LOCKED` desta Wave 0 (secao 5) declara, entre outros:
```
multica-auth-work/server/pkg/db/generated/*
multica-auth-work/server/internal/service/task.go
multica-auth-work/server/internal/handler/daemon.go
```
Colisoes reais:
1. **`pkg/db/generated/*`** e reivindicado pelo **LANE-DB do ORQ-41** como lane exclusiva serial, com
   dono unico por toda a duracao. Dois cartoes nao podem deter o mesmo diretorio de codigo gerado -
   uma execucao de `sqlc generate` de um sobrescreve a do outro.
2. **`internal/service/task.go`** e reivindicado pela **W2 do ORQ-41**.
3. `generated/*` com **um** asterisco nao cobre subdiretorios; se houver qualquer nivel abaixo, fica
   fora do lock. Usar `**`.
4. `internal/handler/daemon.go` e `internal/service/task.go` sao artefatos do **Commit 2 (Wave 1)**,
   nao da Wave 0. Declara-los no `FILES_LOCKED` da Wave 0 amplia o lock alem do necessario e bloqueia
   outras frentes sem motivo.

**Correcao**: o `FILES_LOCKED` da Wave 0 deve conter **somente** `sqlc.yaml` e
`pkg/db/generated/**`, e a coordenacao com o LANE-DB do ORQ-41 precisa de ruling do GTL sobre **quem
detem `generated/**` primeiro**. Wave 1 declara seu proprio lock, separado.

---

## 5. Achados menores

- **M1 - `-m 0700` nao corrige diretorio pre-existente.** `mkdir -p -m 0700 "$TMPDIR" "$GOCACHE"`
  aplica o modo apenas aos diretorios **criados**. Se `$HOME/.private-tmp` ja existir com modo mais
  frouxo, ele permanece frouxo e o gate passa. Adicionar `chmod 0700 "$TMPDIR" "$GOCACHE"`
  incondicional, e verificar com `stat -c%a`.
- **M2 - falta `set -euo pipefail`** no bloco de gates. Sem isso, `cd` que falha nao interrompe, e o
  `$(...)` do Gate 1.4 mascara erro do binario.
- **M3 - Gate 1.6 compila so `./cmd/server/...`.** O churn em `pkg/db/generated` afeta tambem o
  daemon e a CLI (`cmd/multica`). Um `go build ./...` custa pouco mais e cobre o raio real.
- **M4 - `go vet ./pkg/db/generated/...`** e util, mas vet em codigo gerado costuma ser silencioso;
  o valor real esta em `go build ./...`. Manter, sem tratar como sinal forte.
- **M5 - veredito proprio `PASS`.** A secao 6 declara PASS no proprio artefato. Pelo mesmo criterio
  aplicado a outros runbooks hoje, autor nao se auto-aprova; o status correto e "PROPOSTA, aguardando
  review".

---

## 6. Bloqueadores exatos para virar PASS

1. **B1** corrigir o Gate 1.3 com `--relative` (ou ancora de caminho completo) - hoje **reprova o
   cenario correto**, provado empiricamente.
2. **B2** trocar `go fmt -l` por `gofmt -l` - hoje o gate **reescreve** `pkg/db/generated/**`,
   contrariando a proibicao da secao 1.2 e produzindo falso verde na segunda execucao.
3. **B3** corrigir o caminho do worktree (`gtl-i03-orq13-phase1`, nao `gtl-orq13-wave0`) e trocar o
   criterio de idempotencia de `git status --porcelain` limpo por comparacao **geracao 2 vs geracao 1**
   restrita a `pkg/db/generated`.
4. **B4** adicionar passo 0 de inventario do estado ja materializado e definir o split **por hunk**
   com criterio de atribuicao, incluindo a origem do churn em `task_message.sql.go`.
5. **B5** reduzir o `FILES_LOCKED` da Wave 0 a `sqlc.yaml` + `pkg/db/generated/**`, e obter ruling do
   GTL sobre a colisao de `generated/**` e `service/task.go` com o ORQ-41.
6. **B6** reconciliar a `127` untracked com o registrador central: ratificar ou renumerar, e registrar
   a decisao. Enquanto nao houver registrador nomeado, isso e bloqueio de processo.
7. **B7** `chmod 0700` incondicional + verificacao, `set -euo pipefail`, e `go build ./...` em vez de
   apenas `./cmd/server/...`.
8. **B8** trocar o veredito proprio de PASS por "proposta aguardando review".

Nenhum desses exige refundar o runbook. As retificacoes conceituais da V2 estao certas; o que falha e
a **execucao dos gates**, e falha de um jeito que passaria despercebido: um gate que sempre reprova e
outro que muta arquivos e depois fica verde.

## 7. Nao-afirmacoes

- **Nao executei** nenhum gate: sem `sqlc generate`, sem `go build`, sem `go vet`, sem `gofmt`, sem
  migration. As unicas leituras foram `git status --porcelain`, `git diff --name-only` (com e sem
  `--relative`), `git ls-files --error-unmatch`, `git worktree list`, `ls`, `command -v` e
  `go help fmt`.
- Nao editei nenhum arquivo, nao commitei, nao fiz stage e nao toquei o worktree do ORQ-13 - as
  leituras de `git` sao nao-mutantes.
- Nao sei se a `127` e o numero correto para o ORQ-13: e decisao do registrador central, que eu nao
  identifiquei.
- Nao verifiquei o conteudo dos hunks de `pkg/db/generated/*` para atribuir cada um a Wave 0 ou 1 -
  isso e o trabalho do executor, e o motivo do B4.
- Nao revalidei o UUID do cartao ORQ-13 por `GET`; o runbook cita a issue por numero, sem UUID.
- Nao alterei assignee nem postei comentario; nenhum card criado.
