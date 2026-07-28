# ORQ-12: predicados da ADR aplicados (`41ae01b` + `d9f0ae2`)

- executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T16:20Z
- fonte autoritativa: `adr-orq21-orq12-producing-account-snapshot.md`, sha256
  **`f34e6dcc69112292c782294b857df419790f1a2ff1ef87f53150529bcc49139b`**, mais as rulings R3 do ORQ-21
- commits **novos** sobre `1822dd9`, **sem amend**: `41ae01b` (predicados + matriz de recusa) e
  `d9f0ae2` (correcao de compilacao). Worktree limpo.
- **Nenhum arquivo do worktree do ORQ-21 foi tocado.** Sem push, PR, merge, quadro, AWS, segredo, prod.

## Modelo confirmado pela ADR

Uma linha de `agent_task_queue` e **uma tentativa**; o retry cria linha nova e **congela de novo**
(`CreateRetryTask`). Logo nao e preciso tabela de tentativas, e o `task_usage` chaveado por task fica
correto por construcao. Mantive esse modelo e escrevi um teste dedicado para ele.

## Predicados aplicados dentro do `UPDATE` atomico do claim

| predicado | regra aplicada |
|---|---|
| tenant | `accounts.tenant_id` **e** `approved_accounts.tenant_id` iguais ao `agent.workspace_id`. `tenant_id` **e** o UUID do workspace; outra semantica **falha fechado**. Sem `COALESCE`, sem palpite |
| provider | fonte canonica e o runtime **da task** (`runtime_id -> agent_runtime.provider`), **nao** `runtime_config`, com `agy -> antigravity` antes de comparar com `accounts.vendor` |
| status | somente `available` ou `leased`; exclusividade duravel vem do `agent_id` PK + indice unico staged em `assignments(account_id)` |
| worktype | exatamente `GENERAL`; **`NULL` rejeitado**, sem default implicito e sem backfill |
| ambiguidade | **`LIMIT` removido**. Duas linhas fazem o claim **falhar**, em vez de escolher |

`sqlc` **validou o subselect correlacionado contra o schema** - e prova de que as colunas e a correlacao
com `agent_task_queue` sao legais, nao so de que compila em Go.

## Matriz de recusa - um caso por predicado (16 funcoes de teste no arquivo)

tenant divergente na **conta**; tenant divergente na **aprovacao**; **vendor** divergente do provider do
runtime; `exhausted`, `cooldown`, `degraded` recusados **e** `leased` aceito; `HEAVY`, `CHEAP`, `REVIEW`
e **`NULL`** recusados; `allowed = false` recusado; e o caso **positivo** do alias (`agy` no runtime
contra vendor `antigravity`) - que e exatamente o que quebra primeiro se o `CASE` em SQL divergir do
canonicalizador Go.

Mais o teste do item 6 da ADR: o agente rotaciona **entre tentativas** e cada linha de usage conserva
**sua propria** conta produtora.

Os fixtures foram reescritos para **satisfazer** os predicados em vez de presumi-los: o vendor e **lido
do runtime da task** em vez de hardcoded, o tenant e o workspace de teste, e o escopo e explicito para
que um `NULL` possa ser gravado e provado como recusado.

## Falha minha nesta rodada, e a correcao

O `41ae01b` **nao compilava**: eu declarei `strPtr` que o pacote ja tem em `handler_test.go:3954`.
Causa: rodei `git commit` **no mesmo script** da verificacao, sem gatear nela, entao um `go vet`
vermelho nao impediu o commit. Corrigido em `d9f0ae2`, **commit novo, sem amend**. O `HEAD` agora passa
`go vet` e `go test -c`.

## Verificacao executada

```
sqlc generate (config staged)  -> OK, schema-validado e deterministico (2 execucoes iguais)
go vet ./internal/handler ./pkg/db/...  -> OK
go build ./...                 -> OK
go test -c -o /dev/null ./internal/handler -> COMPILA
gofmt -l  nos meus arquivos    -> vazio
git diff --check               -> limpo
```

## Gate combinado - o que **ainda nao** foi provado

A ADR exige, e eu **nao** posso executar aqui: atomicidade do claim **sob concorrencia**, preservacao no
reclaim, re-snapshot no retry, reatribuicao pos-claim, mismatches de tenant/provider/worktype, contas
revogadas/indisponiveis/ambiguas, exclusividade sob concorrencia, legado `NULL`, delete de conta,
isolamento de workspace, idempotencia de usage, **zero falso-verde** de pacote, e `up/down/up`.

Fico com a preparacao pronta e **aguardo** o commit/review do R3 para o gate combinado, conforme
acordado com o Codex56#B.

## Nao-afirmacoes

- **Os testes com banco continuam sem execucao.** Sem Postgres alcancavel aqui, e o `TestMain` do pacote
  sai antes do `m.Run` - o `go test` reportaria `ok` com **zero** testes. Afirmo **compilacao**, nao
  aprovacao, para **todos** os 16 testes e para os 5 predicados.
- Nao rodei `-race`, `-json` nem `up/down/up`.
- **Divida tecnica que continua aberta**: o `CASE` `agy -> antigravity` **duplica** o mapeamento que vive
  em `internal/daemon/brain/compatibility.go:116-133`, e `agent_runtime.provider` e `TEXT` **sem
  `CHECK`**. Escrevi o `CASE` conforme a ruling, mas **nao** implementei o teste de paridade contra o
  helper Go, e continuo preferindo normalizar **na escrita** (P2). Enquanto isso nao for decidido, um
  alias novo faria o claim parar de congelar **em silencio**.
- Nao medi quantas linhas existentes ficam nao-executaveis sob as rulings; pela ADR isso e **esperado**,
  mas o tamanho dessa fila deveria ser medido antes da janela.
- Nao apliquei a migration em lugar nenhum; o numero segue **nao materializado** (`NEXT_CANONICAL_`).
- Nao fiz rebase sobre o R3 - ele ainda nao esta disponivel para mim.
