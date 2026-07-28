# GTL-67 — Peer Review Adversarial do Kanban Integrity Monitor V6

- Auditor: Codex56#B (`w7:p4`)
- Data UTC: 2026-07-27
- Artefato: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v6.md`
- Referência: `.deploy-control/p0/evidence/gtl-kanban-v5-deep-sql-audit.md` (GTL-60)
- Modo: READ-ONLY; nenhuma conexão a banco, mutação, build ou execução de teste
- Veredito: **BLOCK**

## Resumo

A V6 corrige corretamente os quatro defeitos SQL centrais do GTL-60. O design ainda não pode
receber PASS porque a garantia de dedup/clear depende de memória de processo, mas o advisory lock
é liberado a cada tick; a posse pode alternar entre réplicas e o estado desaparece em restart.
Também faltam a semântica de clear em falha parcial, hidratação inicial/reconnect do frontend,
tenant guards nas dependências, índice para `issue_dependency`, enumeração de workspaces, testes
com a semântica exata e arquivos realmente necessários no `FILES_LOCKED`.

| Requisito GTL-67 | Veredito | Evidência |
|---|---|---|
| Grace por `updated_at` | **PASS** | V6:40-54 |
| Quatro estados ativos, incluindo `waiting_local_directory` | **PASS** | V6:50-54 e 64-68 |
| Ausência de completed via `status='completed' AND completed_at IS NOT NULL` | **PASS** | V6:71-85 |
| CTE + `UNION` + parent/`blocked_by`/multi-dep dedup | **PASS lógica SQL; BLOCK tenant/index** | V6:87-115 |
| Workspace/index | **BLOCK** | falta guard em `p/dep`; falta índice em `issue_dependency`; não há enumeração global |
| Lock/query/unlock na mesma conexão | **BLOCK incompleto** | V6:123-142 não instancia `db.New(conn)` e unlock usa ctx cancelável |
| Chamada pós-sweeper | **PASS de posição, incompleto no wiring** | V6:178; ponto real é após `runtime_sweeper.go:88` |
| Dedup/clear e listener isolation | **BLOCK** | estado volátil, falha parcial e frontend/reconnect não definidos |
| Onze testes exatos | **BLOCK** | há 11 nomes, mas o teste 6 não prova supressão e o 7 não prova todos os listeners |
| `FILES_LOCKED` reais/completos | **BLOCK** | faltam migration/index, testes, tipos/event state/hidratação frontend |

## 1. Reavaliação das quatro queries

### 1.1 Anomalia 1 — PASS

Trecho V6:45-54:

```sql
WHERE i.workspace_id = $1
  AND i.status = 'todo'
  AND i.assignee_type IN ('agent', 'squad')
  AND i.assignee_id IS NOT NULL
  AND i.updated_at < now() - INTERVAL '15 seconds'
  AND NOT EXISTS (
    SELECT 1 FROM agent_task_queue atq
    WHERE atq.issue_id = i.id
      AND atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
  );
```

Isto fecha ambos os defeitos V5: o update real de assignee grava `updated_at=now()`
(`server/pkg/db/queries/issue.sql:85-100`) e tasks terminais antigas não escondem a ausência de
uma task ativa.

### 1.2 Anomalia 2 — PASS

V6:64-68 contém exatamente:

```sql
atq.status IN ('queued', 'dispatched', 'running', 'waiting_local_directory')
```

É coerente com a migration que define `waiting_local_directory` como estado ativo
(`server/migrations/109_agent_task_waiting_local_directory.up.sql:1-15`) e com a query produtiva
de cancelamento de tasks ativas (`server/pkg/db/queries/agent.sql:198-207`).

### 1.3 Anomalia 3 — PASS

V6:79-84 usa corretamente:

```sql
AND NOT EXISTS (
  SELECT 1 FROM agent_task_queue atq
  WHERE atq.issue_id = i.id
    AND atq.status = 'completed'
    AND atq.completed_at IS NOT NULL
)
```

Assim, uma conclusão antiga válida continua provando conclusão mesmo com retry `queued` criado
depois. O termo “EXISTS” no resumo deve ser lido como a condição cuja ausência é detectada; o SQL
correto para a anomalia é `NOT EXISTS`.

### 1.4 Anomalia 4 — PASS de cardinalidade; BLOCK de tenant/index

O CTE V6:90-114 projeta as mesmas quatro colunas nas duas fontes e usa `UNION`, não
`UNION ALL`. Se o pai também existir como `blocked_by`, as duas tuplas são idênticas e uma é
removida. Dependências distintas permanecem e o `GROUP BY issue_id, workspace_id, issue_title`
produz uma linha por issue com `jsonb_agg` ordenado. O filtro é somente
`d.type='blocked_by'`, como exigido.

Os contraexemplos do GTL-60 são refutados por inspeção relacional:

1. pai P + relação `C blocked_by P` vira uma única tupla `(C, workspace, title, P)`;
2. P, D1 e D2 resolvidos viram três IDs no mesmo aggregate, não três eventos;
3. uma linha `type='blocks'` não entra.

Há, porém, dois bloqueios novos:

1. **Tenant guard incompleto.** O schema permite `parent_issue_id` e
   `depends_on_issue_id` apontarem para qualquer workspace; são FKs apenas por UUID
   (`server/migrations/001_init.up.sql:65,89-94`). A query limita `i.workspace_id`, mas não exige
   `p.workspace_id=i.workspace_id` nem `dep.workspace_id=i.workspace_id`. Uma relação corrompida
   cross-tenant pode influenciar o badge e expor UUID alheio no aggregate.
2. **Não há índice em `issue_dependency`.** As migrations possuem
   `idx_issue_status(workspace_id,status)` (`001_init.up.sql:167-175`) e
   `idx_agent_task_queue_issue_id` (`035_task_queue_issue_id_index.up.sql:1-5`), mas nenhuma cria
   índice para `issue_dependency`. A segunda perna do CTE pode fazer scan integral dessa tabela.

Correção mínima:

```sql
JOIN issue p
  ON p.id = i.parent_issue_id
 AND p.workspace_id = i.workspace_id
...
JOIN issue dep
  ON dep.id = d.depends_on_issue_id
 AND dep.workspace_id = i.workspace_id
```

e migration versionada:

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_issue_dependency_issue_type_dep
  ON issue_dependency (issue_id, type, depends_on_issue_id);
```

O gate de plano deve usar fixture de escala e verificar os relations/indexes relevantes. Exigir
“zero Seq Scan” de forma global é falso-frágil: PostgreSQL escolhe legitimamente Seq Scan para
tabelas minúsculas.

## 2. Parse/execução segura

Os quatro blocos foram revisados estaticamente quanto a aridade de `UNION`, aliases, tipos,
agregação, parâmetros e terminação; não há erro sintático aparente. O host oferece somente o
cliente `psql`, sem parser PostgreSQL offline instalado. Conectar a uma instância existente não
seria um teste efêmero autorizado, e compilar um helper sobre `pg_query_go` violaria o escopo sem
build. Portanto nenhum SQL foi enviado a banco. Os contraexemplos foram avaliados pela álgebra das
tuplas acima, sem tocar em dados.

## 3. Workspace traversal ausente

Cada query exige `$1`, mas `sweepKanbanIntegrity(ctx,pool,bus)` em V6:124 não recebe workspace e o
pseudocódigo não enumera workspaces. O `ListWorkspaces` existente não serve ao worker global:
ele exige um `user_id` e retorna workspaces do membro (`server/pkg/db/queries/workspace.sql:1-8`).

Correção mínima: adicionar query interna explícita, na mesma conexão bloqueada, por exemplo
`SELECT id FROM workspace ORDER BY id`, e definir:

- política para workspace removido durante o tick;
- somente reconciliar um workspace depois que suas quatro queries terminarem com sucesso;
- limite/batching para não segurar conexão/lock indefinidamente.

Sem isso, “quatro queries por workspace” não é executável e o estado de workspaces omitidos pode
ser confundido com clear.

## 4. Lock: mesma handle não basta

V6:125-139 adquire um `*pgxpool.Conn` e faz try-lock/unlock nessa handle. Isto melhora a V5, mas
V6:141 deixa a parte decisiva como comentário:

```go
// Execução das 4 queries SQL V6 na MESMA conexão conn...
```

O wiring deve ser literal:

```go
lockedQueries := db.New(conn)
// ListWorkspaceIDs e as quatro Detect... usam lockedQueries, nunca queries global.
```

`db.New` aceita qualquer `DBTX`, inclusive a conexão adquirida
(`server/pkg/db/generated/db.go:14-25`).

Há ainda uma falha de cleanup: o defer de unlock usa o mesmo `ctx`. Se o serviço for cancelado,
`conn.Exec(ctx, pg_advisory_unlock...)` pode falhar; o erro é descartado e `Release()` pode devolver
ao pool uma sessão ainda dona do lock. Correção: usar transaction-scoped advisory lock dentro de
uma transação read-only, ou unlock com contexto de cleanup independente e destruir/hijack+close a
conexão se unlock falhar ou retornar false.

O teste deve provar:

1. try-lock, enumeração e quatro queries foram chamadas via a mesma `*pgxpool.Conn`;
2. a segunda conexão não adquire durante o tick;
3. após cancelamento no meio do tick, a segunda conexão consegue adquirir;
4. nenhum caminho usa o `*db.Queries` global.

## 5. Single-instance por tick não preserva dedup

`ActiveAnomalies map[...]` em V6:149 é local ao processo. O lock em V6:137-139 é liberado ao fim de
cada tick. Logo o desenho garante exclusão mútua momentânea, não uma instância estável.

Contraexemplo temporal:

```text
t1: réplica A adquire 4247; A.prev={} e current={K}; A emite anomaly(K); unlock.
t2: réplica B adquire 4247; B.prev={} e current={K}; B emite anomaly(K) de novo.
```

Contraexemplo de clear perdido:

```text
t1: A emite anomaly(K), guarda K localmente e libera o lock.
t2: K é resolvida; B adquire com B.prev={}; current={}; B não emite cleared(K).
t3+: se B continuar adquirindo, o badge recebido pelo cliente nunca é removido.
```

Restart reproduz ambos: o mapa volta vazio. Além disso, V6 não diz onde o monitor é construído.
Se for instanciado dentro do tick, até uma réplica única perde dedup a cada 30s.

Para manter a semântica delta `anomaly/cleared`, a correção robusta é estado compartilhado e
durável com chave única `(workspace_id,issue_id,anomaly_type)`; inserts/deletes retornam exatamente
os eventos novos/cleared em transação. Isto requer decisão explícita porque o monitor deixa de ser
zero-write absoluto, embora continue sem mutar `issue` ou task. Alternativa estritamente
read-only: abandonar deltas como fonte de verdade e publicar/servir snapshot completo versionado
que o frontend substitui atomicamente. Memória local + lock liberado por tick não atende.

## 6. Semântica de `cleared` incompleta

Mesmo após escolher estado compartilhado/snapshot, V6 deve definir:

- payload com `workspace_id`, `issue_id`, `anomaly_type` e geração/versão;
- clear por chave, não “remover todo alerta da issue”;
- se K1 resolve no mesmo tick em que K2 aparece, clear(K1) não pode apagar K2;
- erro em qualquer uma das quatro queries do workspace aborta a reconciliação desse workspace:
  ausência por erro **não** é resolução;
- ordem após commit/snapshot completo, para não publicar estado que depois faça rollback.

O teste V6 número 6 apenas diz que valida “a emissão do evento ... quando a anomalia é resolvida”
(V6:166). Ele não exige um único anomaly em dez ticks, zero duplicatas, clear único, falha parcial,
restart/failover nem duas anomaly types simultâneas.

## 7. Frontend e listeners

### Listener isolation — parcial

No código atual, notification/activity/autopilot assinam tipos específicos; nenhum conhece os
eventos futuros. `listeners.go:150-165` usa `SubscribeAll`, que é justamente o fanout WebSocket.
Logo a direção “realtime sim, writers secundários não” é correta.

O teste V6 número 7 só promete zero linhas em notification/activity. Ele também deve provar:

- zero task/autopilot dispatch;
- zero inbox/subscriber mutation;
- exatamente um fanout realtime;
- ausência de listener tipado nos três registradores.

### Badge não tem fonte inicial

`board-card.tsx` recebe apenas `Issue` e não mantém assinatura/estado de anomalia
(`packages/views/issues/components/board-card.tsx:59-77`). Eventos ocorridos antes de abrir o
board, durante desconexão ou antes de reload são perdidos. V6 não define endpoint/snapshot para
hidratação ou refetch em reconnect.

Além disso, os novos tipos nem compilam no cliente sem:

- adicionar os dois valores a `WSEventType` (`packages/core/types/events.ts:10-82`);
- definir payloads em `WSEventPayloadMap` (`events.ts:380-466`);
- armazenar o set por issue em contexto/store/query cache e atualizá-lo no realtime;
- refazer snapshot no reconnect (`realtime/hooks.ts:22-32`).

Alterar apenas `board-card.tsx` levaria a uma assinatura por card e ainda não resolveria estado
inicial. O estado deve viver no nível do board/workspace.

## 8. Chamada pós-sweeper

O ponto correto existe: o loop real chama, em ordem, stale runtimes, stale tasks, expired queued e
GC (`server/cmd/server/runtime_sweeper.go:80-89`). A chamada do monitor deve entrar após
`gcRuntimes`, ainda na mesma iteração. `main.go:330-365` possui `pool` e inicia o sweeper, portanto
é correto incluí-lo para construir uma única instância e passar `pool/monitor`.

PASS de desenho condicional: o teste precisa observar a ordem real, não apenas duas funções
mockadas sem o loop.

## 9. Onze testes: contagem PASS, cobertura BLOCK

A V6 lista exatamente 11 nomes (V6:161-171), mas não restaura a semântica exata do GTL-60:

1. Teste 6 não afirma `1 anomaly + 0 reemissões em 10 ticks + 1 clear`;
2. teste 6 não cobre troca de réplica/restart ou falha parcial;
3. teste 7 não cobre autopilot/task/inbox nem fanout realtime único;
4. teste 8 não cobre ctx cancelado/unlock falho nem proíbe queries globais;
5. teste 10 não define fixture de escala e ignora a ausência de índice em `issue_dependency`;
6. nenhum teste cobre hidratação inicial/reconnect;
7. os testes 1–3 descrevem somente contraexemplos negativos; a suíte completa deve preservar
   também os happy paths que detectam uma anomalia real.

Correção mínima: manter os 11 gates, mas tornar explícitas as assertions acima e adicionar os
casos de estado compartilhado/snapshot escolhidos. Se “11” for limite rígido, agrupar os cenários
no mesmo teste é aceitável; removê-los não é.

## 10. `FILES_LOCKED` incompleto

Os sete caminhos V6:178-184 são reais como destinos: quatro já existem e três são novos
planejados. Eles não cobrem a implementação declarada.

Faltam, conforme a solução escolhida:

- migration up/down para índice de `issue_dependency` e, se adotado, estado durável;
- arquivos Go de teste para service, lock/order e integração SQL/EXPLAIN;
- `packages/core/types/events.ts`;
- store/query/hook realtime de anomalias no nível workspace;
- teste frontend do badge, clear por type, initial hydration e reconnect;
- handler/query/rota de snapshot inicial, se o estado for durável ou consultável;
- locale/ARIA do badge, se houver texto visível.

Sem esses locks, a implementação forçaria escrita fora do lote ou entregaria um badge que só
funciona para clientes conectados no instante do primeiro evento.

## Correção mínima para PASS

1. Manter os quatro SQLs V6 e adicionar tenant guards em `p/dep`.
2. Adicionar índice versionado de `issue_dependency` e gate EXPLAIN com fixture de escala.
3. Definir enumeração/batching de workspaces na mesma conexão.
4. Tornar literal `lockedQueries := db.New(conn)` e fechar o caminho de cancelamento do lock.
5. Escolher estado compartilhado durável **ou** snapshot versionado; remover a dependência de mapa
   local com lock liberado por tick.
6. Definir clear por key, fail-closed em erro parcial e hidratação/reconnect frontend.
7. Completar testes e `FILES_LOCKED` com os caminhos acima.

## Veredito final

**BLOCK.** A lógica SQL principal agora está correta, mas o monitor V6 ainda não garante a
integridade observável prometida em múltiplas réplicas, restart, erro parcial ou reconnect. O
`PASS` deve aguardar a correção do estado/delta, tenant/index, lock cleanup, testes e superfície
frontend.
