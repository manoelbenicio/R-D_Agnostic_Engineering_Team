# GTL-77 — Peer Review Adversarial do Kanban Integrity Monitor V7

- Auditor: Codex56#B (`w7:p4`)
- Data UTC: 2026-07-27
- Artefato: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v7.md`
- Referência: `.deploy-control/p0/evidence/gtl-kanban-v6-peer-review.md` (GTL-67)
- Modo: READ-ONLY; nenhum SQL, migration, build, teste, board ou serviço executado
- Veredito: **BLOCK**

## Resumo executivo

A V7 escolhe a arquitetura correta ao substituir o mapa volátil por estado PostgreSQL, adicionar
tenant guards, índice, endpoint de hidratação e tipos frontend. Porém o documento ainda não define
o protocolo que produz as garantias declaradas: não há SQL de upsert/reconcile/clear, transação por
workspace, scan token, paginação, política de delete/retention, autorização da API, revisionamento
API/WS, outbox ou lista completa de arquivos/testes.

| Requisito GTL-77 | Veredito | Evidência |
|---|---|---|
| Unique `(workspace,issue,type)` | **PASS estrutural** | PK em V7:48-56 |
| Upsert monotônico/idempotente | **BLOCK** | nenhum SQL de upsert; `last_seen_at` existe mas não tem regra |
| Clear somente após scan completo | **PASS de intenção / BLOCK de atomicidade** | V7:21-25,29-32; sem transação/reconcile SQL |
| Transação/lock/queries mesma conn | **BLOCK** | `db.New(conn)` existe, mas não há `BeginTx`; unlock ignora falha |
| Failover/restart | **BLOCK até protocolo atômico** | tabela ajuda, mas eventos e reconcile ainda podem se perder |
| Delete/cascade/retention | **BLOCK parcial** | cascades existem; semantics de evento/retention/down ausentes |
| Workspaces paginados | **BLOCK** | V7:175 usa `ListAllWorkspaceIDs`, não pagina |
| Parent/dep tenant guard | **PASS** | V7:120 e 131 |
| Índice e EXPLAIN | **PASS de índice / BLOCK de gate** | índice correto em V7:43-45; fixture/plano não definidos |
| API auth/hydration | **BLOCK** | endpoint sem middleware/route/query/version |
| WS ordering/frontend reconnect | **BLOCK** | sem revision, buffer, outbox ou tratamento de race GET×WS |
| Listener isolation | **BLOCK de cobertura** | direção correta, teste ainda omite autopilot/task/inbox/fanout |
| Placeholder migration/Z01 | **PASS** | não materializa número; deve continuar placeholder |
| `FILES_LOCKED` completo | **BLOCK** | faltam down, router, API client/query, store, board-view e testes |

## 1. Schema durável

### 1.1 Chave única — PASS

V7:48-56 declara:

```sql
PRIMARY KEY (workspace_id, issue_id, anomaly_type)
```

Isto satisfaz literalmente a unicidade pedida e oferece o índice necessário para listar por
`workspace_id`; por isso `idx_kanban_integrity_anomaly_ws` em V7:58-59 é redundante com o prefixo
da PK e só acrescenta custo de escrita.

### 1.2 Integridade ainda incompleta — BLOCK

O schema aceita:

1. `anomaly_type` arbitrário; um typo cria uma segunda “classe” que nunca será limpa pelo código
   correto;
2. `workspace_id=W1` com `issue_id` pertencente a W2, pois as duas FKs são independentes;
3. `last_seen_at < first_detected_at`;
4. overwrite acidental de `first_detected_at` por um upsert mal escrito.

Correção mínima:

```sql
CHECK (anomaly_type IN (
  'assigned_todo_no_active_task',
  'in_progress_no_active_task',
  'done_no_completed_task',
  'blocked_with_resolved_dependency'
)),
CHECK (last_seen_at >= first_detected_at)
```

e garantir tenant consistency por FK composta `(issue_id,workspace_id)` para uma unique
correspondente em `issue`, ou, se o custo do índice redundante não for aceito, por todas as escritas
serem `INSERT ... SELECT` de `issue` com ambos os IDs e a API sempre fazer `JOIN issue` com
workspace guard. A primeira opção é enforcement de schema; a segunda depende de código e deve ter
teste adversarial cross-tenant.

O estado também precisa de `last_scan_id UUID NOT NULL` ou equivalente para distinguir “não visto
neste scan completo” de “não processado por falha”.

## 2. Upsert monotônico/idempotente não foi desenhado

A V7 cria `first_detected_at` e `last_seen_at`, mas não apresenta nenhuma query para inserir,
tocar, listar ou remover anomalias. A PK sozinha evita duas linhas; não define:

- evento somente no primeiro insert;
- `first_detected_at` imutável;
- `last_seen_at = GREATEST(old,new)`;
- detalhes atualizados sem reemitir;
- retry do mesmo scan sem novo evento;
- clear emitido somente pela sessão que realmente deletou a linha.

Um protocolo mínimo set-based por workspace deve, dentro de uma única transação:

1. gerar `scan_id` e `observed_at` uma vez;
2. materializar as quatro queries em um conjunto `current`;
3. `INSERT ... ON CONFLICT DO NOTHING RETURNING` para obter somente novas;
4. atualizar existentes com `last_seen_at=GREATEST(...)`, `last_scan_id=scan_id` e details;
5. `DELETE ... WHERE workspace_id=$1 AND last_scan_id<>scan_id RETURNING` para obter somente clears;
6. commit;
7. publicar eventos derivados dos `RETURNING` somente após commit.

O passo 3 não deve depender de `xmax=0`, detalhe interno frágil do PostgreSQL. Inserts e updates
separados, ou uma staging CTE com `DO NOTHING RETURNING`, tornam a origem do evento explícita.

Sem essas queries, os testes V7:234-237 não têm contrato verificável.

## 3. Scan completo, transação e lock

### 3.1 Mesma conexão — parcial

V7:151-181 agora possui `conn := pool.Acquire`, `lockedQueries := db.New(conn)`, try-lock,
enumeração e reconcile na mesma handle. Isso fecha o erro básico da V6.

Não há, porém, `BeginTx`. As quatro leituras, upserts e deletes seriam autocommits separados. Uma
falha na quarta query pode deixar inserts das três anteriores; uma falha depois de deletes pode
deixar clears persistidos sem evento. A frase “falha cancela reconciliação” não produz rollback.

Correção mínima: uma transação por workspace na conexão já bloqueada:

```go
tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
workspaceQ := db.New(tx)
// quatro queries + reconcile set-based
// tx.Rollback(ctx) em qualquer erro
// tx.Commit(ctx)
// só então bus.Publish(...)
```

Uma transação global para todos os workspaces seria longa e ampliaria contenção; transaction per
workspace preserva atomicidade local e permite que um workspace falho não bloqueie os demais.

### 3.2 Unlock ainda pode vazar lock

V7:167-172 usa contexto não cancelado, melhoria correta. Mas descarta erro e o booleano retornado
por `pg_advisory_unlock`. Se rede/servidor falhar, `conn.Release()` pode devolver ao pool uma sessão
ainda bloqueada. Advisory locks de sessão são reentrantes: reutilizar a mesma sessão e chamar
`pg_try_advisory_lock` novamente aumenta a contagem, tornando o vazamento persistente.

O defer deve fazer `QueryRow(...).Scan(&unlocked)`. Em `err != nil || !unlocked`, deve retirar a
conexão do pool (`Hijack`) e fechá-la; não fazer `Release`. Alternativa mais segura: uma transação
global com `pg_try_advisory_xact_lock`, mas isso conflita com a recomendação de transactions
curtas por workspace. Se mantido o session lock global, seu cleanup precisa ser comprovado sob
cancelamento e falha real.

## 4. Races obrigatórias

### 4.1 Duas réplicas

Com lock adquirido e liberado corretamente, apenas uma reconcilia. A PK e
`INSERT DO NOTHING RETURNING`/`DELETE RETURNING` devem ser a segunda defesa: mesmo se duas sessões
chegarem ao reconcile por bug, somente uma emite new/clear. V7 não mostra essas queries, logo o
teste de handover ainda é aspiracional.

### 4.2 Query parcial

Contraexemplo atual:

```text
Q1 detecta K1 -> estado persistido;
Q2/Q3 executam;
Q4 falha;
função retorna;
K1 ficou gravada apesar de o workspace não ter scan completo.
```

Somente uma transação por workspace e publicação pós-commit fecham isso. Em erro, preservar o
snapshot anterior e emitir zero eventos.

### 4.3 Anomalia resolve durante o scan

Sem transação, Q1 pode observar “todo sem task” e, milissegundos depois, a task ser criada antes de
Q4/clear; o monitor persiste e publica um estado que já não existe. Com `REPEATABLE READ`, o
resultado representa um snapshot consistente no início da transação, mas ainda pode ficar
defasado até o próximo tick. O design deve declarar esse SLA (máximo um intervalo) ou revalidar
candidatos antes do commit com locks adequados. Não pode prometer ausência absoluta de evento
transiente sem escolher uma dessas semânticas.

### 4.4 Issue apagada durante o scan

Os FKs `ON DELETE CASCADE` são corretos: apagar issue/workspace remove estado ativo. Mas há dois
casos não definidos:

- delete após detection e antes do insert causa FK error; a transação do workspace deve rollback
  e emitir zero;
- delete após commit e antes do WS event remove a linha por cascade, mas um evento anomaly stale
  ainda pode ser publicado.

Não é necessário emitir `cleared` para issue apagada se `issue:deleted` remove card e store, mas
isto deve ser contrato e teste. O store deve purgar anomalies no evento `issue:deleted`.

## 5. Delete, cascade e retention

V7 passa na mecânica de cascade (`workspace` e `issue` em V7:49-50), mas não declara política:

- resolvida é deletada imediatamente segundo o teste V7:235;
- não existe retenção histórica;
- `last_seen_at` não possui TTL nem job;
- tipos descontinuados após upgrade podem ficar eternamente se não participarem do scan.

Para uma tabela de **estado ativo**, delete imediato pode ser a decisão correta; deve ser escrito
explicitamente: “sem retention histórica; cascades não emitem clear; tipos removidos entram em
cleanup versionado”. Se auditoria de 24h for requisito, precisa tabela/outbox histórica separada,
não TTL na tabela ativa. A V7 não pode alegar retention até essa decisão constar.

A migration também precisa de placeholder `.down.sql`. Ele deve remover tabela/índice somente em
DB efêmero/gate; rollback de aplicação live deve seguir a política expand-first do Z01 e não
derrubar estado automaticamente.

## 6. Enumeração de workspaces

V7:174-181 chama `ListAllWorkspaceIDs` e carrega toda a lista. Isso não satisfaz “paginada” e pode
segurar a conexão/session lock por tempo ilimitado.

Correção mínima keyset:

```sql
SELECT id
FROM workspace
WHERE id > sqlc.arg('after_id')::uuid
ORDER BY id
LIMIT sqlc.arg('limit_rows');
```

O cursor inicial precisa de variante nullable ou UUID mínimo documentado. Novos UUIDs inseridos
“atrás” do cursor podem esperar o tick seguinte; isto é aceitável se declarado. Cada página deve
ter limite, timeout e métricas. Workspace deletado entre enumeração e reconcile é rollback/no-op,
nunca clear inferido por ausência de página.

## 7. SQL de detecção, tenant guards e índices

Os quatro SQLs V7 preservam as correções V6. A Anomalia 4 agora contém literalmente:

```sql
p.workspace_id = i.workspace_id
dep.workspace_id = i.workspace_id
d.type = 'blocked_by'
UNION
```

Veredito: **PASS** para lógica de detecção e dedup relacional.

O índice proposto `(issue_id,type,depends_on_issue_id)` é compatível com a perna relacional. O
runner real não envolve migrations numa transação justamente por suportar
`CREATE INDEX CONCURRENTLY` (`server/cmd/migrate/main.go:188-200`), então a sintaxe é compatível.

O gate EXPLAIN continua subespecificado. Deve:

- usar PostgreSQL efêmero migrado e `ANALYZE`;
- carregar cardinalidade suficiente em `issue`, `agent_task_queue` e `issue_dependency`;
- verificar os relations/indexes relevantes, não proibir todo `Seq Scan` em tabela pequena;
- provar que `idx_issue_status`, `idx_agent_task_queue_issue_id` e
  `idx_issue_dependency_issue_type_dep` aparecem nos cenários seletivos;
- registrar `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)` sem dados sensíveis.

`CREATE ... IF NOT EXISTS` pode esconder objeto com mesmo nome e definição errada. O gate deve
comparar `pg_indexes.indexdef` e constraints exatas; idealmente a migration canônica usa create
fail-closed quando não há necessidade de compatibilidade com estado prévio.

## 8. API: auth e hydration incompletos

V7:192-206 define apenas caminho e JSON. A rota real precisa ser adicionada sob
`/api/workspaces/{id}` com
`RequireWorkspaceMemberFromURL(queries,"id")`, seguindo
`server/cmd/server/router.go:582-603`. O middleware faz auth, membership, task-token binding e
tenant context (`server/internal/middleware/workspace.go:171-245`).

O handler deve consultar por `workspace_id` vindo do contexto/URL, nunca aceitar workspace do
body/query, retornar ordem determinística e `Cache-Control: private, no-store` ou ETag/revision
coerente. Testes obrigatórios: 401 sem identidade, 404/403 sem membership, cross-workspace vazio,
member autorizado 200 e issue FK inconsistente não vazada.

`FILES_LOCKED` omite `server/cmd/server/router.go` e o teste do handler.

## 9. Ordering API × WebSocket e crash window

V7:208-210 diz “fetch na montagem e reconnect”, mas não define ordem. Duas races:

```text
GET snapshot começa -> anomaly event chega -> GET antigo termina e sobrescreve o evento.
GET termina -> antes de subscribe, clear event ocorre -> cliente mantém badge stale.
```

Também existe crash window:

```text
DB commit insere/deleta estado -> processo morre antes de bus.Publish -> cliente conectado
naquela janela nunca recebe delta e não reconecta, embora API já esteja correta.
```

O `events.Bus` atual é síncrono e in-process (`server/internal/events/bus.go:27-31,57-87`), não um
outbox durável.

Contrato mínimo:

1. snapshot API retorna `workspace_revision` junto com anomalies;
2. anomaly/cleared carregam a mesma revision monotônica e a chave completa;
3. frontend subscreve primeiro, bufferiza, hidrata snapshot e aplica em ordem apenas eventos com
   revision maior;
4. reconnect sempre refaz snapshot;
5. commit+event usa outbox transacional, ou o frontend faz polling/refetch periódico para fechar a
   janela commit→crash. Sem uma dessas opções, durable DB não implica UI convergente.

Uma linha de estado por workspace com revision transacional é mais segura que timestamps. Clear
deve conter `workspace_id`, `issue_id`, `anomaly_type`, `revision`; clear de K1 nunca apaga K2.

## 10. Frontend e listener isolation

É correta a direção de montar o hook no board, não uma assinatura por card. Porém
`board-view.tsx` é o consumidor literal (`packages/views/issues/components/board-view.tsx:111+`)
e não consta no lock.

Também faltam:

- API/query key em `packages/core/issues/queries.ts` ou módulo próprio;
- client HTTP em `packages/core/api/client.ts`;
- store workspace-scoped para snapshot/revision/buffer;
- export de `packages/core/types/kanban.ts` em `packages/core/types/index.ts`;
- testes do hook, store, reconnect/order e `board-card`;
- tratamento de `issue:deleted` no store;
- locale/ARIA do badge.

Listener isolation permanece incompleto. Hoje os writers escutam tipos específicos e
`SubscribeAll` faz o fanout realtime, o que é correto. O teste V7:240 ainda menciona apenas
notification/activity. Deve provar zero inbox/subscriber/task/autopilot writes e exatamente um
fanout realtime. “notification_listeners/activity_listeners/autopilot_listeners” são arquivos,
não tabelas.

## 11. Placeholder Z01 e `FILES_LOCKED`

O placeholder é **PASS**: esta revisão não atribui nem materializa número. O controle Z01 reserva
uma lane serial de migrations/sqlc e atualmente cita Kanban condicionalmente
(`gtl-z01-master-integration-control.md:161-179,205-228`). A instrução GTL-77 prevalece: manter
`NEXT_CANONICAL_MIGRATION` até o General-TL liberar a lane e o HEAD integrado confirmar o número.

A lista V7:215-228 cobre parte de migration/query/generated/server/API/core/UI, mas não cobre:

1. placeholder migration `.down.sql`;
2. `server/cmd/server/router.go`;
3. testes separados de handler, migration/up-down, SQL/EXPLAIN, sweeper/lock;
4. API client/query key;
5. store/revision/buffer;
6. `packages/core/types/index.ts`;
7. `packages/views/issues/components/board-view.tsx`;
8. testes hook/store/UI/reconnect;
9. eventual outbox schema/query/service/dispatcher e teste, se essa opção for escolhida.

Logo não satisfaz a exigência explícita de migration/query/generated/server/API/core/store/UI/tests.

## 12. Gates faltantes

Os dez testes V7 são bons nomes iniciais, mas faltam assertions/casos:

- duas réplicas concorrentes com insert/delete `RETURNING` e um evento;
- issue/workspace delete em três pontos do scan;
- anomaly resolve durante scan com SLA escolhido;
- uma das quatro queries falha depois de outras três;
- upsert repetido preserva first, avança last monotonicamente e não reemite;
- scan_id repetido é idempotente;
- unlock retorna erro/false e a conexão não volta ao pool;
- paginação com múltiplas páginas, insert/delete entre páginas;
- API auth/cross-tenant;
- GET×WS ordering, reconnect, crash/poll/outbox;
- K1 cleared e K2 active na mesma issue;
- listener isolation completo;
- migration up/down/up e definição exata dos índices/constraints;
- positive/negative paths das quatro anomalias preservados.

## 13. Implementation waves recomendadas

### Wave 0 — Sequenciamento e freeze

- Manter migrations como placeholders.
- General-TL confirma lane Z01, número canônico e locks globais de migrations/generated.
- Congelar contrato de anomaly types, retention e estratégia outbox versus polling.
- Gate: `FILES_LOCKED` completo e nenhuma sobreposição.

### Wave 1 — Schema + queries

- Migration up/down: tabela/constraints/index/scan state e, se escolhido, revision/outbox.
- Queries: paginação, quatro detectors, set reconcile, list hydration.
- `sqlc generate` uma única vez na lane exclusiva.
- Gates: ephemeral PostgreSQL up/down/up, segunda geração diff zero, cross-tenant, idempotência,
  concurrency e EXPLAIN de escala.

### Wave 2 — Service + sweeper

- Session lock seguro, transactions por workspace, scan completo e eventos pós-commit/outbox.
- Acoplamento após `gcRuntimes`.
- Gates: duas réplicas, restart/handover, query parcial, cancel/unlock failure, issue delete e
  resolution-during-scan.

### Wave 3 — API autenticada

- Route member-scoped, handler, snapshot revisionado e resposta determinística.
- Gates: 401/403/404/200, cross-tenant, deleted issue/workspace, pagination/size.

### Wave 4 — Core/store/frontend

- Tipos e exports, API query, store workspace-scoped, subscribe-buffer-hydrate, reconnect,
  issue-delete purge e badge.
- Gates: GET×WS inversions, K1/K2, reconnect, missed event recovery, ARIA/UI.

### Wave 5 — Integração

- Listener isolation completo, full affected suites, typecheck/lint/build, migration fingerprint,
  review independente no mesmo hash.
- Nenhum deploy/restart/board mutation faz parte destas waves sem gate separado do owner.

## Veredito final

**BLOCK.** V7 é um avanço arquitetural material e preserva corretamente migration placeholder,
SQL detectors, tenant guards e índice. Ainda não define atomicidade, monotonicidade, retention,
paginação nem convergência API/WS necessárias para chamar o estado de durável. Implementação só
deve começar após corrigir o design e fechar Wave 0.

