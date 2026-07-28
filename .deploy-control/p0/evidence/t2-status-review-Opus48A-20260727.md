# T2 STATUS REVIEW - ResetIssueToTodoIfNoActiveTask + reconcileFailedIssue

- revisor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T10:15Z
- escopo: SOMENTE issue.sql, issue.sql.go gerado, internal/service/task.go e testes correlatos
- modo: leitura. Nada editado, nenhum deploy, restart ou rerun.

## F1 ALTA - a atomicidade NAO fecha a corrida: sobrou um terceiro caminho nao atomico

`cmd/server/runtime_sweeper.go:307-317` NAO foi migrado e continua fazendo check-then-act em
tres idas ao banco:

```go
307 		if t.IssueID.Valid {
308 			if issue, err := queries.GetIssue(ctx, t.IssueID); err == nil {
309 				workspaceID = util.UUIDToString(issue.WorkspaceID)
310 				issueKey := util.UUIDToString(t.IssueID)
311 				if issue.Status == "in_progress" && !processedIssues[issueKey] {
312 					processedIssues[issueKey] = true
313 					if hasActive, herr := queries.HasActiveTaskForIssue(ctx, t.IssueID); herr == nil && !hasActive {
314 						queries.UpdateIssueStatus(ctx, db.UpdateIssueStatusParams{ID: t.IssueID, Status: "todo", WorkspaceID: issue.WorkspaceID})
315 					}
316 				}
317 			}
318 		}
```
`UpdateIssueStatus` nao tem guarda de task ativa nem de status. Entre a leitura da 308, a
checagem da 313 e a escrita da 314 cabe: um retry filho ser enfileirado, outra task reivindicar
a issue, ou o usuario/agente escolher outro status - e a 314 sobrescreve para `todo`. E o
retorno da 314 e DESCARTADO por completo: sem `err`, sem log. Enquanto essa linha existir, o
ganho de `ResetIssueToTodoIfNoActiveTask` e local ao `TaskService` e a classe de bug do ORQ-22
segue alcancavel pelo processo do sweeper. RECOMENDACAO: trocar 313-315 pela query atomica e
tratar o erro.

## F2 MEDIA - lacuna de evento de UI nova, introduzida pelo patch

`FailTask` agora MUTA o status da issue (task.go:1546) mas continua publicando apenas evento de
TASK: `task.go:1594 s.broadcastTaskEvent(ctx, protocol.EventTaskFailed, task)`. Nenhum dos dois
caminhos emite evento com escopo de ISSUE na transicao `in_progress -> todo`.

Isso e regressao de comportamento observavel, nao de codigo: ANTES o `FailTask` nao tocava o
status da issue - o proprio comentario removido dizia "Issue status is NOT changed here" - logo
nao havia transicao para anunciar. AGORA existe transicao sem evento correspondente. Se o cliente
web nao refizer fetch de `/api/issues` ao receber `task:failed`, o card fica exibindo
`in_progress` embora o banco esteja em `todo`, ou seja o sintoma do ORQ-22 vira um fantasma de
UI em vez de um bug de estado. PRECISA VERIFICAR no frontend se `task:failed` dispara refetch de
issues; nao verifiquei, esta fora do meu escopo de leitura.

## F3 SEGURA, verificada - remover `retriedIssues` NAO abriu buraco

A supressao cruzada por issue dentro do lote foi removida: antes, se QUALQUER task do lote
gerasse retry para a issue, o reset era suprimido para as OUTRAS tasks da mesma issue; agora
`reconcileFailedIssue` recebe apenas `child != nil` da task corrente. Verifiquei que a guarda SQL
subsume o caso: `pkg/db/queries/agent.sql`, `CreateRetryTask`, insere o filho com status
literal `'queued'`, e a guarda testa
`t.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')`. Logo o `NOT EXISTS`
ja bloqueia o reset quando existe retry recem-criado. O parametro `retried bool` passou a ser
defesa redundante, nao carga - o que e bom, nao um problema.

## F4 BAIXA - dedup por issue (`processedIssues`) removido de `HandleFailedTasks`

N tasks falhas da mesma issue no mesmo lote agora geram N `GetIssue` + N tentativas de UPDATE.
Sem perda de correcao: a partir da segunda, `i.status = 'in_progress'` nao casa, o `:one`
devolve zero linhas e o codigo trata explicitamente
`err != nil && !errors.Is(err, pgx.ErrNoRows)`, engolindo o ErrNoRows esperado. Custo e apenas
round-trips extras por tick. Aceitavel; se quiserem, o dedup pode voltar como micro-otimizacao,
nunca como correcao.

## F5 SEM DIVERGENCIA SQLC - checado

- `pkg/db/queries/issue.sql` usa `@id` e `@workspace_id`; o gerado usa `$1` e `$2` com
  `ResetIssueToTodoIfNoActiveTaskParams{ID, WorkspaceID}` na MESMA ordem.
- `RETURNING i.*` foi expandido para 24 colunas na mesma ordem da expansao pre-existente de
  `updateIssueStatus` (`id, workspace_id, title, description, status, priority, assignee_type,
  assignee_id, creator_type, creator_id, parent_issue_id, acceptance_criteria, context_refs,
  position, due_date, created_at, updated_at, number, project_id, origin_type, origin_id,
  first_executed_at, start_date, metadata`), e a ordem de `row.Scan` acompanha.
- O prefixo `i.` aparece so nessa query, o que e o esperado para UPDATE com alias, nao sinal de
  edicao manual.
Conclusao: saida coerente com `sqlc generate`. Nao encontrei divergencia.

## F6 POSITIVO - nao ha TOCTOU dentro de `reconcileFailedIssue`

O `GetIssue` da linha 1893 serve apenas para obter `workspaceID`; NAO e usado como guarda de
`status`. Toda a decisao vive no `WHERE` da query. Se o `GetIssue` falhar, a funcao retorna ""
e nao tenta reset - fail-closed. Correto.

## Nada mutado

Nenhum arquivo editado, nenhum deploy, restart, rerun, commit, push, reset, clean ou
checkout --. Somente leitura de fonte e de `git diff` no worktree.
