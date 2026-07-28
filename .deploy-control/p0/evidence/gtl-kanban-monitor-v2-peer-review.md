# GTL-32 - Peer review independente: Kanban Integrity Monitor V2

- revisor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T11:47Z
- alvo: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v2.md` (Antigravity w8:p2, GTL-30)
- modo: READ-ONLY. Nenhum codigo editado, nenhum build, deploy, restart, commit ou rerun.
- reportado a: Codex56-TL / GENERAL-TECH-LEAD w5:pC

---

## VEREDITO: **BLOCK**

As quatro correcoes que o GTL-28 exigiu estao **corretas e confirmadas contra o schema**. O BLOCK
nao e por regressao: e por **7 defeitos novos ou remanescentes**, sendo 3 deles impeditivos para
entregar a um escritor unico (2 caminhos de arquivo que nao existem, e uma janela de graca que
protege o timestamp errado).

---

## 1. O que o V2 acertou - confirmado linha a linha

| correcao V2 | verificacao no schema | resultado |
|---|---|---|
| `issue.status = 'done'` | `001_init.up.sql:57-58 CHECK (status IN ('backlog','todo','in_progress','in_review','done','blocked','cancelled'))` | CONFIRMADO (`'completed'` de fato nao existe em `issue`) |
| `NOT EXISTS` em vez de `LEFT JOIN ... IS NULL` | semantica correta e independente de historico | CONFIRMADO |
| incluir `waiting_local_directory` | `109_agent_task_waiting_local_directory.up.sql:15 CHECK (status IN ('queued','dispatched','running','waiting_local_directory','completed','failed','cancelled'))` | CONFIRMADO |
| `issue_dependency` com `type='blocked_by'` | `001_init.up.sql:89-94`, colunas `issue_id`, `depends_on_issue_id`, `type CHECK (type IN ('blocks','blocked_by','related'))` | CONFIRMADO |
| zero rerun | nenhuma chamada a `EnqueueTaskForIssue`, `CreateRetryTask` ou `MaybeRetryFailedTask` no desenho | CONFIRMADO por inspecao do documento |

Outras premissas que conferi e que **passam**:
- `assignee_type IN ('agent','squad')`: `'squad'` e valido desde `084_squad.up.sql:31`, que substitui
  o CHECK original de `001_init.up.sql:61`. Nao e erro.
- `agent_task_queue.completed_at` existe: `001_init.up.sql:136`.
- `agent_task_queue.status='completed'` existe: `001_init.up.sql:132`.
- Indice que sustenta o `NOT EXISTS`: `035_task_queue_issue_id_index.up.sql:4
  idx_agent_task_queue_issue_id`. As anomalias 1 e 2 tem suporte de indice no lado da subconsulta.
- Deriva de citacao, sem impacto: o doc cita `001_init.up.sql:56` para o enum; o CHECK esta em 57-58.

---

## 2. BLOQUEADOR 1 - dois dos sete `arquivos_locked` NAO EXISTEM

A lista de arquivos travados e o contrato de nao colisao da frota. Dois itens apontam para caminhos
inexistentes, e um escritor obediente criaria arquivo em arvore errada.

| item do doc | realidade |
|---|---|
| `server/internal/protocol/events.go` | **NAO EXISTE**. `ls server/internal/protocol/events.go` -> No such file or directory. O caminho real e **`server/pkg/protocol/events.go`** (e e de lá que vem `protocol.EventIssueUpdated`, `protocol.EventTaskFailed` usados hoje). |
| `packages/views/issues/components/kanban-card.tsx` | **NAO EXISTE**. `find . -iname "*kanban*"` fora de `node_modules` retorna **ZERO** arquivos no repo inteiro. Os componentes reais do board sao `packages/views/issues/components/board-card.tsx`, `board-column.tsx` e `board-view.tsx`. |

Correcao exigida na lista: trocar por `server/pkg/protocol/events.go` e por `board-card.tsx` (o
badge no card e ali), confirmando antes se o badge deve nascer em `board-card.tsx` ou em
`board-view.tsx`.

## 3. BLOQUEADOR 2 - a janela de graca de 10s protege o timestamp ERRADO

```sql
AND i.created_at < now() - INTERVAL '10 seconds' -- Mitiga janela de corrida no enqueue inicial
```
A corrida que o doc quer mitigar (secao 5.1) e entre **atribuir a issue a um agente** e **enfileirar
a task**. Mas o filtro olha `created_at`, que e a criacao da ISSUE. Consequencia concreta e comum:
uma issue criada ontem, que o owner atribui a um agente agora, tem `created_at` de ontem, passa pela
graca instantaneamente e **e reportada como anomalia durante os milissegundos normais do enqueue**.
Ou seja o falso positivo que a clausula existe para evitar continua acontecendo em todo fluxo de
atribuicao manual de issue antiga - que e o fluxo normal de um board com backlog.

`issue` tem `updated_at` (`001_init.up.sql:71`) e existe `idx_issue_status ON issue(workspace_id, status)`.
Recomendo trocar por `i.updated_at < now() - INTERVAL '10 seconds'`, com a ressalva de que
`updated_at` muda por qualquer edicao, portanto e aproximacao. A forma correta de verdade seria um
timestamp de atribuicao, que **nao existe** hoje na tabela - se o integrador quiser exatidao, isso e
coluna nova, e ai vira mudanca de schema.

## 4. BLOQUEADOR 3 - duplicacao de eventos por linha na Anomalia 4

`issue_dependency` **nao tem UNIQUE** (`001_init.up.sql:89-94`: so PK em `id`). Logo uma issue pode
ter N linhas `blocked_by`. No SQL da Anomalia 4:
```sql
LEFT JOIN issue parent ON i.parent_issue_id = parent.id
LEFT JOIN issue_dependency idep ON idep.issue_id = i.id AND idep.type = 'blocked_by'
LEFT JOIN issue dep_issue ON idep.depends_on_issue_id = dep_issue.id
```
uma issue `blocked` com pai resolvido **e** duas dependencias resolvidas retorna **3 linhas**, e o
`COALESCE(parent.id, dep_issue.id)` mascara isso porque cada linha parece uma anomalia distinta.
Resultado: 3 eventos `kanban:integrity_anomaly` para a MESMA issue, e 3 badges. Correcao: agregar
por issue (`DISTINCT ON (i.id)` ou `GROUP BY` com `array_agg` das dependencias resolvidas), e o
payload passar a carregar lista em vez de um id.

## 5. Duplicacao de eventos ENTRE ticks - nao ha supressao alguma

O desenho publica no bus a cada tick de 30s (secao 4.1 e 5.3) e **nao tem estado, dedup, watermark
nem TTL de alerta**. Uma anomalia que persista - e as 9 issues preservadas persistem por decisao do
owner - gera **2 eventos por minuto, 2880 por dia, por issue**, indefinidamente, para cada cliente
WebSocket do workspace. Com 9 issues em anomalia isso e ~26 mil eventos/dia sem nenhuma informacao
nova.

Agrava: `protocol.EventIssueUpdated` ja e assinado por tres listeners de producao
(`cmd/server/activity_listeners.go:51`, `autopilot_listeners.go:20`,
`notification_listeners.go:583`, mais `subscriber_listeners.go:48`). Nao afirmo que o evento novo
sera assinado por eles - e evento novo. Mas o padrao do codigo e que evento no bus gera
`activity_log` e **notificacao**, entao quem implementar precisa garantir explicitamente que
`kanban:integrity_anomaly` NAO entre em nenhum desses listeners, ou o owner recebe milhares de
notificacoes. Exigencia minima: dedup por `(issue_id, anomaly_code)` com reemissao apenas na
transicao (aparecer/desaparecer), nao por tick.

## 6. Custo de query - a Anomalia 2 nao tem indice utilizavel

O unico indice de status em `issue` e `001_init.up.sql:170 CREATE INDEX idx_issue_status ON issue(workspace_id, status)`.
A coluna **lider e `workspace_id`**. Os quatro SQLs do V2 **nao filtram workspace**: a Anomalia 2 e
literalmente `WHERE i.status='in_progress' AND NOT EXISTS (...)`. Sem predicado em `workspace_id` o
planner nao usa esse indice como acesso; tende a **seq scan em `issue`** a cada 30 segundos, quatro
vezes por tick, crescendo linearmente com o board.

Duas saidas, ambas aceitaveis: (a) varrer **por workspace**, aproveitando o indice existente e
casando com o fato de o evento ser por workspace de qualquer forma; ou (b) criar indices parciais
(`WHERE status='in_progress'` etc.), o que e migration nova. Recomendo (a): zero migration e casa com
o modelo de sala WebSocket.
A Anomalia 3 e a mais caras das quatro: `LEFT JOIN LATERAL ... ORDER BY created_at DESC LIMIT 1` por
issue `done` - e uma subconsulta ordenada por issue, sobre o conjunto que **mais cresce com o tempo**
(issues concluidas). Precisa de limite explicito de janela, por exemplo so issues `done` com
`updated_at` nos ultimos N dias, senao o custo cresce para sempre.

## 7. Multiplas instancias - sem guarda

O monitor e wired em `cmd/server/main.go` (item 2 dos locked) como ticker de 30s. Nao ha eleicao de
lider nem lock: `grep -rn "pg_try_advisory_lock|advisory|leader" server/cmd/server/*.go` acha apenas
`main.go:382` (advisory lock 4246, de outra funcao SQL) e nomes de squad leader. Com 2 replicas do
backend, cada tick emite o dobro de eventos. Nao verifiquei se o deploy atual roda replica unica -
declaro como pergunta, nao como fato. Se for replica unica hoje, ainda assim recomendo `pg_try_advisory_lock`
no tick, porque e barato e evita que um `docker compose up --scale` futuro duplique alertas
silenciosamente.

## 8. Ordem versus o sweeper - a garantia da secao 5.3 nao existe

O doc afirma: *"O monitor roda no servidor a cada 30 segundos, **após** a rodada do
`runtime_sweeper.go`"*. Nao ha como garantir isso: `cmd/server/runtime_sweeper.go:21
sweepInterval = 30 * time.Second` e `:77 ticker := time.NewTicker(sweepInterval)`. Sao **dois tickers
independentes de mesmo periodo**, iniciados em momentos distintos; a defasagem entre eles e arbitraria
e pode ser de milissegundos, inclusive intercalando no meio do `HandleFailedTasks`. Nesse instante o
monitor ve issue `in_progress` cuja task acabou de falhar e cuja reconciliacao ainda nao rodou -
falso positivo transitorio, com evento e badge.

Saidas: (a) o monitor rodar **acoplado ao fim** do tick do sweeper (mesma goroutine, apos o sweep);
(b) periodo maior e desalinhado, por exemplo 45s ou 60s, e uma graca sobre `updated_at` da task
maior que o periodo do sweeper. Recomendo (a): elimina a corrida em vez de reduzir a probabilidade.
Como bonus, (a) tambem resolve o item 7, porque so uma instancia roda o sweeper... o que eu **nao
verifiquei** - se o sweeper nao tem guarda de instancia, o problema apenas se move.

## 9. Falsos positivos remanescentes na Anomalia 3

Dois caminhos concretos, ambos com o SQL como esta:
1. **retry em voo**: `ORDER BY created_at DESC LIMIT 1` elege a task MAIS NOVA. Se uma task
   `completed` fechou a issue e depois um retry/rerun foi enfileirado, `latest_task.status='queued'`
   e a issue `done` e reportada como anomalia, mesmo tendo conclusao real no historico. O predicado
   correto para "conclusao efetiva" e `EXISTS (task completed com completed_at NOT NULL)`, nao
   "a ultima task esta completed".
2. **conclusao humana legitima**: issue atribuida a agente cuja task falhou e que o owner concluiu
   manualmente e marcou `done`. O SQL a marca como anomalia para sempre. Precisa de decisao: ou
   aceitar como sinal informativo, ou excluir issues cuja transicao para `done` foi feita por
   `member` - dado que **nao esta na tabela `issue`**, exigiria olhar `activity_log`, o que muda o
   custo da query. Registro como decisao de produto, nao como bug puro.

Ainda: o enum tem `'backlog'` e `'in_review'` (`001_init.up.sql:57-58`) e nenhuma das 4 invariantes
os cobre. Nao e defeito, mas o doc deveria declarar que sao estados deliberadamente fora de escopo,
para nao parecerem esquecimento em auditoria futura.

---

## 10. TESTES - o que falta na suite proposta

A suite de 6 testes cobre o caminho felizes das 4 anomalias e a invariancia read-only. Faltam, e
cada um mapeia direto a um achado acima:

1. **`TestAnomaly1_OldIssueAssignedNow_NoAnomalyDuringEnqueue`** - issue com `created_at` de dias
   atras, atribuida agora, task enfileirada 100ms depois: NAO deve haver anomalia. Este teste
   **falha** com o SQL atual (bloqueador 2). E o teste que prova a correcao.
2. **`TestAnomaly4_MultipleResolvedDependencies_EmitsSingleEvent`** - pai `done` + 2 deps `done`:
   exatamente **1** evento, nao 3 (bloqueador 3).
3. **`TestMonitor_SameAnomalyAcrossTicks_EmitsOnce`** - 10 ticks com a mesma anomalia: 1 evento na
   aparicao, 0 nos demais, e 1 evento de clear quando resolver (secao 5).
4. **`TestAnomaly3_CompletedThenRetryQueued_NoAnomaly`** - task `completed` + retry `queued`
   posterior: NAO deve haver anomalia (secao 9.1).
5. **`TestMonitor_NoNotificationListenerSubscribes`** - garante que `kanban:integrity_anomaly` nao
   e consumido pelos listeners de notificacao/activity, senao o alerta vira spam (secao 5).
6. **`TestMonitor_SingleInstanceGuard`** - dois monitores no mesmo banco: apenas um emite por tick
   (secao 7).
7. **`TestMonitor_RunsAfterSweeper_NoTransientFalsePositive`** - task acabando de falhar, sweeper
   ainda nao reconciliou: monitor nao emite (secao 8).
8. **`TestQueries_UseWorkspaceScopedIndex`** - `EXPLAIN` das 4 queries nao contem `Seq Scan` em
   `issue` (secao 6). Teste de custo, com banco populado.

O `TestMonitor_StrictReadOnlyInvariance` proposto e bom e deve ficar, com um reforco: comparar
tambem `updated_at` das linhas, nao so contagem e valores - um `UPDATE` idempotente nao muda valor
mas muda `updated_at`, e passaria batido.

---

## 11. Resumo do BLOCK - o minimo para virar PASS

1. Corrigir os dois caminhos de `arquivos_locked` (`pkg/protocol/events.go`, `board-card.tsx`).
2. Trocar a graca de `created_at` para `updated_at`, ou declarar a necessidade de coluna de atribuicao.
3. Deduplicar a Anomalia 4 por issue.
4. Adicionar supressao de reemissao entre ticks e garantir que o evento nao alimenta notificacao.
5. Escopar as queries por workspace (ou aceitar migration de indice parcial) e limitar a janela da
   Anomalia 3.
6. Decidir instancia unica (advisory lock) e acoplar o monitor ao fim do tick do sweeper em vez de
   afirmar ordenacao entre tickers independentes.
7. Trocar o predicado da Anomalia 3 de "ultima task completed" para "existe task completed".

Nada disso invalida a direcao do V2: as 4 correcoes do GTL-28 estao certas e o principio de zero
rerun esta integro.

## 12. Perguntas que declaro em aberto, sem afirmar

- Se o backend roda em replica unica no deploy atual: nao verifiquei.
- Se o `runtime_sweeper` tem guarda de instancia unica: nao verifiquei.
- Se o badge deve nascer em `board-card.tsx` ou `board-view.tsx`: nao inspecionei os componentes.

## 13. Nada mutado

Nenhum arquivo de codigo editado; sem build, vet, test, deploy, restart, commit, push ou rerun. As 9
issues ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23 permanecem PRESERVADAS. Somente leitura de migrations,
fonte e arvore de arquivos. Os unicos arquivos criados sao este e o meu check-out.
