# GTL-43 - Peer review independente: Kanban Integrity Monitor V3

- revisor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:52Z
- alvo: secao "CORRECAO V3" de `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v2.md` (linhas 188-325)
- referencia: `.deploy-control/p0/evidence/gtl-kanban-monitor-v2-peer-review.md` (meu BLOCK do GTL-32)
- modo: READ-ONLY. Nenhum codigo editado, nenhum build, deploy, restart, commit ou rerun.

---

## VEREDITO: **BLOCK**

5 dos meus 7 findings estao **resolvidos**. O BLOCK persiste por **1 finding nao resolvido**
(custo de query), **1 defeito NOVO introduzido pelo proprio V3** (o `COALESCE` dentro do
`jsonb_agg` mascara dependencias e duplica o pai) e **3 lacunas** (arquivo travado ausente, estado
de dedup volatil sob advisory lock, e regressao da suite de testes).

`pg_try_advisory_lock(4247)` **nao colide** - confirmado, ver secao 3.

---

## 1. Placar contra os meus 7 findings do GTL-32

| # | finding GTL-32 | V3 | evidencia |
|---|---|---|---|
| 1 | 2 caminhos de `arquivos_locked` inexistentes | **RESOLVIDO** | V3 usa `server/pkg/protocol/events.go` e `packages/views/issues/components/board-card.tsx` / `board-view.tsx`, ambos existentes. Substituicao de `kanban-card.tsx` reconhecida no texto. |
| 2 | graca de 10s no timestamp errado (`created_at`) | **RESOLVIDO** | agora `i.updated_at < now() - INTERVAL '15 seconds'`. Ver 2.1 para um efeito colateral positivo. |
| 3 | Anomalia 4 duplicava eventos por linha | **PARCIAL / NOVO DEFEITO** | `jsonb_agg` + `GROUP BY` resolve a duplicacao de LINHA, mas o `COALESCE` dentro do objeto introduz erro pior. Ver 2.2. |
| 4 | sem dedup/clear entre ticks | **RESOLVIDO EM DESENHO, com lacuna** | `ActiveAnomalies map[string]Set` + `EventKanbanIntegrityCleared`. Ver 2.3. |
| 5 | evento pode ser consumido pelos listeners de notificacao | **NAO ENDERECADO** | a secao V3 nao menciona isolamento; nada garante que `kanban:integrity_anomaly` fique fora de `notification_listeners.go:583`, `activity_listeners.go:51`, `autopilot_listeners.go:20`, `subscriber_listeners.go:48`. |
| 6 | custo de query / falta de escopo por workspace | **NAO RESOLVIDO** | ver 2.4 - o plano afirmado nao existe no schema. |
| 7 | ordenacao com o sweeper e instancia unica | **RESOLVIDO NO MECANISMO, incompleto no locked** | advisory 4247 correto; ver 2.5. |
| extra | Anomalia 3 com "ultima task" | **RESOLVIDO** | trocado para `NOT EXISTS (... status='completed' AND completed_at IS NOT NULL)`, que e exatamente o predicado que eu pedi. |
| extra | `backlog`/`in_review` sem declaracao | **RESOLVIDO** | secao 2 declara ambos fora de escopo, com justificativa. |

---

## 2. Detalhamento do que ainda barra

### 2.1 RESOLVIDO, e com um bonus que vale registrar
A troca para `updated_at` nao so corrige o falso positivo da issue de backlog atribuida agora:
ela interage bem com o patch de reconciliacao ja auditado no T2. A query
`ResetIssueToTodoIfNoActiveTask` faz `status='todo', updated_at = now()`, logo uma issue que o
sweeper acabou de reconciliar entra com `updated_at` fresco e ganha 15s de graca automaticamente.
Isso suprime justamente o falso positivo transitorio que eu apontei no finding 7. Efeito bom e
provavelmente nao intencional - vale um comentario no codigo para ninguem "otimizar" isso depois.

### 2.2 DEFEITO NOVO - o `COALESCE` dentro do `jsonb_agg` mascara dependencias e duplica o pai

```sql
jsonb_agg(jsonb_build_object(
    'id', COALESCE(parent.id, dep_issue.id), ...
FROM issue i
LEFT JOIN issue parent ON i.parent_issue_id = parent.id AND parent.status IN ('done','cancelled')
LEFT JOIN issue_dependency idep ON idep.issue_id = i.id AND idep.type = 'blocked_by'
LEFT JOIN issue dep_issue ON idep.depends_on_issue_id = dep_issue.id AND dep_issue.status IN ('done','cancelled')
```
Trace por caso, lembrando que `issue_dependency` nao tem UNIQUE (`001_init.up.sql:89-94`):

| caso | linhas apos join | `resolved_dependencies` produzido | correto? |
|---|---|---|---|
| pai resolvido, 0 deps | 1 | `[pai]` | sim |
| sem pai, 2 deps resolvidas | 2 | `[dep1, dep2]` | sim |
| **pai resolvido + 2 deps resolvidas** | 2 | **`[pai, pai]`** | **NAO** |
| **pai resolvido + 3 deps, 1 resolvida** | 3 | **`[pai, pai, pai]`** | **NAO** |

Como `parent.id` e non-null nas linhas todas, o `COALESCE` sempre escolhe o pai e as dependencias
**nunca aparecem**, alem de o pai ser repetido uma vez por linha de `idep`. O V3 trocou
"3 eventos" por "1 evento com array errado" - e agora o erro e silencioso, porque o array parece
plausivel. Correcao: agregar as duas fontes separadamente, por exemplo
`jsonb_agg(DISTINCT ...)` sobre uma subconsulta `UNION` de (pai resolvido) e (deps resolvidas), ou
dois `jsonb_agg` com `FILTER`. O `DISTINCT` sozinho nao resolve, porque o problema nao e repeticao,
e a escolha errada da fonte.

### 2.3 LACUNA - `ActiveAnomalies` em memoria nao sobrevive a troca do detentor do lock

O dedup e um mapa em memoria do processo, mas quem executa o tick e decidido por
`pg_try_advisory_lock(4247)`. Consequencias:
- restart do backend: o mapa nasce vazio e **todas** as anomalias sao reemitidas como novas. Aceitavel
  uma vez, mas precisa estar declarado.
- **troca de detentor entre replicas**: a replica B assume o lock com mapa vazio e reemite tudo, e a
  replica A - que tinha o estado - **nunca emite o `cleared`** correspondente. O frontend fica com
  badge orfao ate um novo tick coincidir. Isso e pior que o restart, porque nao ha sinal.
Minimo: declarar instancia unica como pre-condicao operacional, ou persistir o estado (tabela ou
`pg_notify` com watermark). Nao exijo tabela; exijo a decisao escrita.

Ponto POSITIVO que confirmei: `map[issue_id]AnomalyType` com um tipo por issue e suficiente, porque
as 4 anomalias sao ancoradas em `i.status` (`todo`, `in_progress`, `done`, `blocked`), que sao
mutuamente exclusivos no CHECK de `001_init.up.sql:57-58`. Nao ha issue com duas anomalias
simultaneas. O modelo de estado esta correto.

### 2.4 NAO RESOLVIDO - o plano de EXPLAIN afirmado nao existe no schema

O V3 afirma para a Anomalia 1: *"Plano EXPLAIN: Index Scan em `issue(status, assignee_type)`"*.
**Esse indice nao existe.** Os indices reais de `issue` sao:
```
001_init.up.sql:168 idx_issue_workspace       ON issue(workspace_id)
001_init.up.sql:169 idx_issue_assignee        ON issue(assignee_type, assignee_id)
001_init.up.sql:170 idx_issue_status          ON issue(workspace_id, status)
001_init.up.sql:171 idx_issue_parent          ON issue(parent_issue_id)
020_issue_number.up.sql:36 idx_issue_workspace_number ON issue(workspace_id, number)
034_projects.up.sql:20 idx_issue_project      ON issue(project_id)
032/036: GIN em title/description
```
Nao ha `(status, assignee_type)` e nao ha indice com `status` como coluna **lider** - em
`idx_issue_status` a lider e `workspace_id`. As quatro queries V3 continuam **sem predicado de
workspace**, e a Anomalia 2 e literalmente `WHERE i.status='in_progress' AND NOT EXISTS (...)`.
Resultado: seq scan em `issue` quatro vezes por tick, a cada 30s, crescendo com o board. O V3 nao
adicionou escopo por workspace nem migration de indice: **apenas escreveu um plano que o planner
nao pode executar**. Este e o finding 6 intacto, agora com uma afirmacao incorreta em cima.

Duas saidas, como antes: varrer por workspace (usa `idx_issue_status`, zero migration, e casa com o
fato de o evento ser publicado por sala de workspace de qualquer forma) ou criar indices parciais
(migration nova). E o EXPLAIN precisa ser **medido**, nao afirmado - por isso pedi o teste 7.8 no
GTL-32, que o V3 removeu.

### 2.5 LACUNA - falta `runtime_sweeper.go` na lista de arquivos travados

O V3 diz: *"Em `cmd/server/main.go`, a chamada `RunKanbanIntegrityMonitorTick(ctx)` e agendada para
rodar imediatamente apos a conclusao do `runtime_sweeper`"*. Isso e contraditorio com o codigo: o
sweeper roda em goroutine com ticker proprio (`cmd/server/runtime_sweeper.go:21 sweepInterval = 30 * time.Second`
e `:77 ticker := time.NewTicker(sweepInterval)`). `main.go` nao tem como sequenciar algo que
acontece dentro daquele loop sem que o sweeper exponha um hook/callback ou chame o monitor no fim de
cada tick. Ou seja a mudanca **e em `runtime_sweeper.go`**, e esse arquivo nao esta nos 7
`arquivos_locked` - um escritor seguindo a lista nao teria permissao de tocar onde a mudanca precisa
ocorrer. Corrigir a lista para 8 itens, ou reformular como "monitor invocado pelo sweeper".

### 2.6 NAO ENDERECADO - isolamento do evento em relacao a notificacao

Nada na secao V3 trata disso. O padrao do repo e que evento no bus alimente `activity_log` e
notificacao: `cmd/server/activity_listeners.go:51`, `autopilot_listeners.go:20`,
`notification_listeners.go:583`, `subscriber_listeners.go:48` assinam `EventIssueUpdated`. Nao
afirmo que os dois eventos novos serao assinados - sao novos. Exijo apenas que o desenho **declare**
que `kanban:integrity_anomaly` e `kanban:integrity_anomaly_cleared` nao entram em nenhum listener de
notificacao/activity, e que exista teste fixando isso. Com dedup por transicao o volume caiu muito,
mas um `cleared`/`anomaly` alternando em issue instavel ainda geraria notificacao repetida.

---

## 3. `pg_try_advisory_lock(4247)` - NAO COLIDE, confirmado

Varri o repo inteiro por ids de advisory lock:
```
cmd/backfill_codex_usage_cache/main.go:26  const rollupAdvisoryLockID = 4246
cmd/backfill_task_usage_hourly/main.go:107 SELECT pg_advisory_lock(4246)
cmd/migrate/main.go:21                     advisory lock 4246
cmd/server/main.go:382                     advisory lock 4246
internal/scheduler/jobs_task_usage.go:22,60  advisory lock 4246
internal/taskusagebackfill/backfill.go:19,128 advisory lock 4246
```
**4246 e o unico id em uso** em todo o repo, e sempre para a mesma familia (rollup de `task_usage`,
migrate e scheduler). `4245` e `4247` nao aparecem em nenhum lugar. Logo **4247 esta livre** e a
escolha esta correta.

Duas ressalvas de qualidade, nao bloqueantes:
1. Nao existe registro central de ids. 4246 esta hardcoded em 6 arquivos e como `const` em apenas um
   (`cmd/backfill_codex_usage_cache/main.go:26`). Recomendo que 4247 nasca como `const` unica e
   exportada, para o proximo agente nao precisar fazer o grep que eu fiz.
2. `pg_try_advisory_lock` e **session-level**: exige liberar com `pg_advisory_unlock(4247)` no fim do
   tick e, no pool, garantir que lock e unlock ocorram na MESMA conexao. Se o tick usar `pgxpool`
   sem conexao dedicada, o unlock pode cair em outra conexao e o lock fica preso ate a sessao morrer,
   travando o monitor para sempre. O precedente correto esta em
   `cmd/backfill_task_usage_hourly/main.go:107-114`, que usa `lockConn` dedicada e `defer` de
   unlock - o V3 deve seguir esse padrao explicitamente. Alternativa mais segura:
   `pg_try_advisory_xact_lock` dentro de transacao, liberado automaticamente no commit.

---

## 4. Testes - a suite V3 REGREDIU frente ao GTL-32

O GTL-32 exigiu 8 testes. O V3 lista 7, mas **5 dos meus 8 desapareceram** e nao foram substituidos:

| teste exigido no GTL-32 | V3 |
|---|---|
| `TestAnomaly1_OldIssueAssignedNow_NoAnomalyDuringEnqueue` | **AUSENTE** - e o unico teste que prova a correcao da graca (finding 2) |
| `TestAnomaly4_MultipleResolvedDependencies_EmitsSingleEvent` | **AUSENTE** - o V3 so tem `BlockedWithDoneParent_JSONAgg`, que e o caso que NAO expoe o bug de 2.2 |
| `TestMonitor_SameAnomalyAcrossTicks_EmitsOnce` | **AUSENTE** - o V3 so testa o `cleared`, nao a supressao por tick |
| `TestAnomaly3_CompletedThenRetryQueued_NoAnomaly` | AUSENTE, porem agora **desnecessario**: o `NOT EXISTS` de 3.3 elimina a classe |
| `TestMonitor_NoNotificationListenerSubscribes` | **AUSENTE** (finding 5) |
| `TestMonitor_SingleInstanceGuard` | coberto por `TestMonitor_PostSweeperOrderAndAdvisoryLock` |
| `TestMonitor_RunsAfterSweeper_NoTransientFalsePositive` | coberto pelo mesmo teste acima |
| `TestQueries_UseWorkspaceScopedIndex` (EXPLAIN sem Seq Scan) | **AUSENTE** - exatamente o teste que refutaria a afirmacao de 2.4 |

Testes que exijo adicionar, alem dos 7 do V3:
1. `TestAnomaly4_ParentAndDependenciesResolved_ArrayContainsAll` - pai `done` + 2 deps `done`: o
   array tem **3 entradas distintas**, nao `[pai, pai]`. Este teste **falha** com o SQL de 3.4 e e a
   prova do defeito 2.2.
2. `TestAnomaly1_OldIssueAssignedNow_NoAnomalyDuringEnqueue` - `updated_at` de agora, task 100ms
   depois: zero anomalia.
3. `TestMonitor_SameAnomalyAcrossTicks_EmitsOnce` - 10 ticks, 1 evento.
4. `TestQueries_NoSeqScanOnIssue` - `EXPLAIN` das 4 queries com banco populado.
5. `TestMonitor_AdvisoryLockReleasedOnSameConn` - lock e unlock na mesma conexao; segundo tick
   consegue adquirir novamente (a armadilha de 3.2).
6. `TestMonitor_LockHandoverDoesNotOrphanBadge` - detentor troca com mapa vazio: comportamento
   declarado e determinístico (2.3).
7. `TestMonitor_NoNotificationListenerSubscribes` (2.6).
E manter o reforco que pedi: `TestMonitor_StrictReadOnlyInvariance` deve comparar tambem `updated_at`
das linhas, nao apenas contagem e valores.

---

## 5. Minimo para PASS

1. Corrigir o `jsonb_agg` da Anomalia 4 para agregar pai e dependencias como fontes distintas
   (defeito 2.2) - **este e o unico defeito de correcao funcional, e e novo do V3**.
2. Escopar as 4 queries por `workspace_id` (ou entregar migration de indice) e **medir** o EXPLAIN,
   removendo a afirmacao sobre `issue(status, assignee_type)`, que nao existe (2.4).
3. Incluir `cmd/server/runtime_sweeper.go` nos `arquivos_locked`, ou reformular o acoplamento (2.5).
4. Declarar o padrao de conexao dedicada + unlock para o advisory 4247, ou usar
   `pg_try_advisory_xact_lock` (3.2).
5. Declarar o comportamento de dedup na troca de detentor/restart (2.3).
6. Declarar o isolamento dos dois eventos novos frente aos listeners de notificacao (2.6).
7. Recolocar os 4 testes removidos e adicionar os 3 novos (secao 4).

Nada disso e refundacao: o V3 arrumou paths, graca, escopo, predicado da Anomalia 3, ciclo de vida
com `cleared` e escolheu um advisory lock livre e correto. O que falta e um bug de agregacao, uma
afirmacao de performance sem base e as declaracoes operacionais.

## 6. Perguntas que declaro em aberto, sem afirmar

- se o backend roda em replica unica hoje: nao verifiquei (afeta 2.3).
- se o `runtime_sweeper` tem guarda de instancia: nao verifiquei; com 4247 no monitor, o monitor
  passa a ter guarda propria, o que torna a pergunta menos critica.
- se o badge pertence a `board-card.tsx` ou `board-view.tsx`: nao inspecionei os componentes; o V3
  cita os dois na secao 1 e apenas `board-card.tsx` na lista de locked - vale alinhar.

## 7. Nada mutado

Nenhum arquivo de codigo editado; sem build, vet, test, deploy, restart, commit, push ou rerun. As 9
issues ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23 permanecem PRESERVADAS. Somente leitura de migrations,
fonte e arvore. Os unicos arquivos criados sao este e o meu check-out.
