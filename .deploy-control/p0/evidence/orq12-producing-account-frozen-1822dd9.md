# ORQ-12 - conta produtora **congelada no claim** (commit `1822dd9`)

- executor/owner: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T15:05Z
- worktree: `/home/ec2-user/workspace/worktrees/orq12-account-id`, branch
  `agent/opus48-a/orq12-task-usage-account-id`, base `c0e93a2`, **limpo**
- commit **novo** `1822dd9` sobre `e7c6a5c`; **sem amend**
- adota o handoff **ORQ-21 -> ORQ-12** (Codex56#B, `182b7b3`): conta aprovada resolvida no backend e
  **persistida** na fila; ORQ-12 apenas **copia**. Nao integrei lookup vivo como imutavel.
- **Sem push, PR, merge, quadro, AWS, segredo, Docker, prod ou alvo real.**

## (1) `COALESCE` invertido - e por que era um defeito real meu

```sql
account_id = COALESCE(task_usage.account_id, EXCLUDED.account_id)
```
O valor **existente vence**; so um `NULL` e preenchido. A ordem anterior
(`COALESCE(EXCLUDED..., task_usage...)`) permitia que um report posterior **reescrevesse** a historia
depois de uma rotacao - e o meu teste anterior afirmava o contrario **sem nunca ter rodado**. Registro
como defeito meu, nao como ajuste de estilo.

## (2) Congelamento **atomico no claim**, nao no primeiro report

`agent_task_queue` ganha `credential_account_id` (nulavel, FK `ON DELETE SET NULL`), congelado **dentro
do mesmo `UPDATE` atomico** de `ClaimAgentTask`:
```sql
SET status = 'dispatched',
    dispatched_at = now(),
    credential_account_id = COALESCE(
        agent_task_queue.credential_account_id,
        (SELECT asg.account_id FROM assignments asg
           JOIN approved_accounts ap ON ap.account_id = asg.account_id AND ap.allowed IS TRUE
          WHERE asg.agent_id = agent_task_queue.agent_id LIMIT 1))
```
- resolvido **server-side**, restrito a `approved_accounts.allowed`;
- `COALESCE` mantem valor ja congelado, logo **reclaim nao re-arquiva** a task;
- o upsert de usage **so copia**: `(SELECT q.credential_account_id FROM agent_task_queue q WHERE q.id = $1)`.
  Ele **nao** faz mais join em `assignments`;
- **nenhum ID vem do daemon**: `UpsertTaskUsageParams` continua sem campo de conta.

## (3) `sqlc.yaml` canonico **restaurado**; config descartavel para o staged

Canonico volta a ter **um** root (`migrations/`). O DDL staged e tipado por
`sqlc.staged-orq12.yaml`, marcado para **deletar na promocao**.

Consequencia **medida**, e ela e melhor do que eu esperava: um `sqlc generate` canonico **antes** da
promocao **falha alto**:
```
pkg/db/queries/task_usage.sql:35:5: column "account_id" of relation "task_usage" does not exist
```
Ou seja, o estado staged e **impossivel de passar despercebido** - em vez de silenciosamente destipar a
coluna. O revisor deve decidir se aceita esse erro duro como sinal, porque ele **quebra** um
`sqlc generate` de rotina ate a promocao.

## (4) Relatorio com escopo de workspace **no SQL** + consumidor autorizado

`task_usage` nao tem coluna de workspace, entao as duas queries passam por
`agent_task_queue -> agent` e filtram `a.workspace_id = @workspace_id`. Sem isso, somariam **todos os
tenants**.

Consumidor: **`GET /api/dashboard/usage/by-account`**, dentro do mesmo grupo autenticado e do mesmo
`h.workspaceMember` das outras rotas de `/api/dashboard`. O bucket nao-atribuivel vai como
`account_id: null` **com** `row_count`, e a atribuicao vai como **tres contadores**, nunca razao.

## (5) `assignments(account_id)` unico, com preflight e simetria de down

`assignments.agent_id` ja era PK, mas a direcao inversa era livre - dois agentes na mesma conta tornam o
gasto por conta nao atribuivel a um trabalhador. Um bloco `DO` **nomeia as contas duplicadas** e aborta
com mensagem propria **antes** de criar o indice (em vez do erro cru de indice unico), e o `down`
derruba o indice.

## Numero de migration

**Nao materializado.** Nome segue `NEXT_CANONICAL_task_usage_account_id.{up,down}.sql`. A expectativa de
**128** esta registrada **em prosa** no cabecalho, porque 127 e a mais alta materializada. So o
registrador transforma isso em numero.

## Testes - 9 casos

1. congelado no claim e copiado para o usage;
2. **reatribuicao A->B depois do claim e antes do primeiro report continua nomeando A** - o caso exato
   que a review bloqueou;
3. re-report **nao** reescreve conta gravada, e a correcao de tokens `(7,9)` chega;
4. tres casos de `NULL`: sem assignment, assignment **nao aprovado**, e aprovacao **revogada**;
5. linha legada `NULL` **nao** e retro-atribuida, mas **e** preenchivel depois que a fila congela;
6. **delete da conta preserva os tokens** `(42,84)` e anula os **dois** snapshots, sem cascade da fila;
7. tres reports colapsam em **exatamente uma** linha, e o ultimo vence nos tokens;
8. relatorio com bucket `NULL` visivel e contadores que somam;
9. **segundo workspace ve zero linhas** - escopo provado no SQL, nao so no handler.

**Dois defeitos latentes meus, corrigidos antes do commit:** o fixture de conta usava colunas que
**nao existem** em `123_rotation` (`provider`, `label`, `status='active'`), e o fixture do segundo
workspace usava `workspace.owner_id`, que **nao existe** em `001`. Ambos **compilavam** e falhariam
**so em CI**.

## Verificacao executada

```
go build ./...                          -> OK
go vet ./internal/handler ./cmd/server ./pkg/db/...  -> OK
sqlc generate (config staged, v1.31.1)  -> OK e reproduzivel
go test -c -o /dev/null ./internal/handler -> COMPILA
gofmt -l  nos arquivos que eu toquei    -> vazio
git diff --check                        -> limpo
```
Nota: `gofmt -l internal/handler/` lista **12** arquivos que **ja** eram nao-formatados em `c0e93a2`
(confirmei com `git stash`); nenhum deles e meu e eu **nao** os reformatei.

Efeito colateral esperado no codigo gerado: `agent_task_queue` ganhou coluna, entao todo `RETURNING *`
sobre ela mudou - `agent.sql.go`, `autopilot.sql.go`, `chat.sql.go` e `runtime.sql.go` foram
regenerados. Nenhuma query foi reescrita a mao.

## Nao-afirmacoes

- **Os testes com banco NAO foram executados**, e portanto **nao rodei** `up/down/up`, `-race` nem
  `-json`. Este host nao tem Postgres alcancavel (5432 fechado, `docker` ausente) e o `TestMain` do
  pacote faz `os.Exit(0)` antes do `m.Run`, o que o `go test` reporta como `ok` com **zero** testes.
  Afirmo **compilacao**, nao aprovacao - inclusive para o preflight do `DO` block, a FK
  `ON DELETE SET NULL`, o indice unico e o congelamento no claim.
- **Nao apliquei a migration em lugar nenhum**, nem em banco de teste.
- **Nao fiz rebase nem port sobre a stack corrigida do ORQ-21**: ela ainda nao esta disponivel para mim
  e o handoff diz que `182b7b3` **nao** persiste o ID. Minhas mudancas estao **isoladas** neste
  worktree, e o port fica pendente. Se o ORQ-21 escolher outro nome de coluna ou outra fonte de
  aprovacao, **o meu claim precisa ser reconciliado** - e esse e o principal risco de integracao aberto.
- O `LIMIT 1` no subselect do claim fica **deterministico** apenas apos o indice unico; e nao resolvi a
  questao de **tenant**: `approved_accounts` e por tenant, e eu filtro so por `allowed IS TRUE`. Se um
  mesmo `account_id` estiver aprovado em tenants diferentes, a escolha e arbitraria. **Isso precisa de
  decisao do ORQ-21/owner.**
- Nao medi performance do subselect no claim (dois lookups por indice) nem dos indices novos.
- Nao alterei assignee, nao postei comentario, nao criei card.
