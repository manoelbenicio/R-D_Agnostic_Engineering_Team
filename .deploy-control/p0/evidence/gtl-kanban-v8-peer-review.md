# GTL-80 — Peer review adversarial do Kanban Integrity Monitor V8

- Auditor: Codex56#B (`w7:p4`)
- Data UTC: 2026-07-27T12:21:45Z
- Artefato auditado: `.deploy-control/p0/evidence/gtl-kanban-integrity-monitor-v8.md`
- Baseline: `.deploy-control/p0/evidence/gtl-kanban-v7-peer-review.md` (GTL-77)
- Modo: READ-ONLY; nenhum código, migration, SQL, banco, build, teste, board ou serviço foi
  alterado/executado
- Veredito: **BLOCK**

## Resumo executivo

O V8 materializa DDL e SQL de upsert/reconcile/prune, escolhe uma transação por workspace, mantém o
placeholder de migration e posiciona corretamente a rota sob o middleware de membro. Entretanto,
ele ainda não é implementável com as garantias declaradas. Há quatro bloqueios independentes:

1. `conn.Hijack()` não destrói nem fecha a conexão. O pgx exige que o chamador feche a conexão
   retornada; o pseudocódigo descarta o retorno e pode deixar a sessão — e o advisory lock 4247 —
   vivos para sempre.
2. `revision` é por linha, mas a API/WS compara contra uma `workspace_revision` que não existe no
   schema nem nas queries. Revisões de anomalias diferentes não são comparáveis.
3. O upsert retorna inserts **e updates** via `(xmax = 0)`, exatamente o oposto do contrato GTL-77
   “`DO NOTHING RETURNING` somente novos”. Se filtrar `is_new_detection`, uma anomalia resolvida e
   reativada não emite evento; se publicar todas as linhas, toda varredura reemite.
4. O V8 afirma rollback/scan completo, mas não apresenta `reconcileWorkspaceTransaction`, não
   verifica seu retorno no loop, usa `ReadCommitted` e não materializa os quatro detectores em um
   snapshot único. A garantia descrita não é demonstrada por código.

O documento também não define polling periódico/outbox para a janela commit→crash, não traz SQL de
hidratação active/resolved, não traz SQL keyset, não lista testes exatos e sua lista de 20 locks
omite arquivos obrigatórios do API client, exports, store/revision/buffer, WS e sqlc/gates.

## Matriz GTL-80

| Requisito | Veredito | Evidência |
|---|---|---|
| DDL/SQL concreto e sintaxe | **PASS sintático parcial / BLOCK semântico** | V8:37-68,88-131 |
| Tx por workspace | **PASS de intenção / BLOCK executável** | V8:21,86,181-183; função omitida e erro ignorado |
| `scan_id` | **PASS como marcador / BLOCK de snapshot** | V8:55,99,106,122; `ReadCommitted` |
| Revision monotônica | **BLOCK** | V8:56,107,118 versus `workspace_revision` em V8:196-202 |
| Upsert retorna só novos | **BLOCK** | V8:103-110 retorna também conflitos e usa `xmax` |
| Reconcile após quatro detectores | **PASS de diagrama / BLOCK de prova** | V8:22-25; detectores/resolução não estão na função |
| Rollback em falha | **BLOCK de implementação** | promessa V8:13,23; nenhuma sequência Begin/Rollback/Commit real |
| Resolve durante scan | **BLOCK** | quatro statements em `ReadCommitted` podem observar estados diferentes |
| Commit→crash / polling | **BLOCK** | V8:198-202 só mount/reconnect; bus real é in-process |
| Hydration active/resolved | **BLOCK** | resposta citada sem query, filtro, order ou regra de substituição |
| Lock bool/err | **PASS parcial** | V8:146-151 e 158-165 inspecionam bool/err |
| Hijack seguro | **BLOCK crítico** | V8:162 descarta retorno; pgxpool/conn.go:70-83 exige `Close` |
| Paginação keyset | **BLOCK** | diagrama diz LIMIT/OFFSET; código chama query inexistente; SQL não foi dado |
| Membership route | **PASS** | V8:193-195 casa com router.go:582-603 e middleware |
| WS revision/order | **BLOCK** | revisão por linha comparada a revisão de workspace inexistente |
| Retention/down/cascade | **BLOCK parcial** | prune existe; constraints, índice, clear de cascade e down seguro faltam |
| Migration placeholder | **PASS** | V8:35-37,65; Z01 mantém 131 condicional |
| `FILES_LOCKED` 20 reais/completos | **BLOCK** | 20 entradas, 9 existentes/11 futuras, mas conjunto incompleto |
| Testes | **BLOCK** | V8 não lista casos/comandos; só três arquivos de teste |
| Waves | **PASS de ordem macro / BLOCK para iniciar** | V8:206-214; Wave 0 não tem gates executáveis |
| Claim READ-ONLY | **PASS com ressalva** | refere-se à autoria do design, não ao monitor mutável |

## 1. Validação estática do SQL concreto

### 1.1 DDL

As construções abaixo são sintaticamente válidas em PostgreSQL:

- `CREATE INDEX CONCURRENTLY IF NOT EXISTS ...` (V8:40-41);
- `CREATE TABLE IF NOT EXISTS ...` com duas FKs, `CHECK`, PK composta e JSONB default
  (V8:44-62);
- `DROP TABLE IF EXISTS ... CASCADE` e `DROP INDEX IF EXISTS ...` (V8:67-68);
- `INSERT ... SELECT unnest(...) ... ON CONFLICT DO UPDATE ... RETURNING` (V8:90-110);
- `UPDATE ... RETURNING` (V8:115-123);
- `DELETE ... INTERVAL '7 days'` (V8:128-131).

Esta é validação estática; não executei DDL ou DML nem conectei a banco. O runner real aceita
`CREATE INDEX CONCURRENTLY` porque deliberadamente não envolve migrations numa transação
(`server/cmd/migrate/main.go:188-200`).

### 1.2 DDL semanticamente incompleto

1. As FKs independentes em V8:45-46 ainda permitem
   `(workspace_id=W1, issue_id pertencente a W2)`. O PK da tabela `issue` é somente `id`
   (`server/migrations/001_init.up.sql:52-69`); não há FK composta que imponha o tenant.
2. Falta `CHECK` de lifecycle:

   ```sql
   CHECK (
     (status = 'active' AND resolved_at IS NULL)
     OR
     (status = 'resolved' AND resolved_at IS NOT NULL)
   )
   ```

   Hoje uma linha `resolved` com `resolved_at=NULL` nunca entra no prune.
3. Falta `CHECK (revision > 0)`.
4. `CREATE ... IF NOT EXISTS` pode aceitar silenciosamente tabela/índice de mesmo nome com definição
   incompatível. O gate deve comparar constraints e `pg_indexes.indexdef`.
5. O prune precisa de índice seletivo, por exemplo em `resolved_at WHERE status='resolved'`; o único
   índice novo é para `issue_dependency`.
6. `DROP TABLE ... CASCADE` no down pode remover objetos futuros que dependam da tabela. O down de
   gate deve falhar fechado sem `CASCADE`, salvo dependências enumeradas. `DROP INDEX` não é
   concurrent e pode bloquear writes em rollback live; rollback live continua expand-first, não
   deve executar down automaticamente.

### 1.3 Upsert não satisfaz “RETURNING só novos”

V8:103-110:

```sql
ON CONFLICT (...) DO UPDATE
...
RETURNING ..., (xmax = 0) AS is_new_detection;
```

Isto retorna uma linha para **todo insert e todo update**. `xmax` é detalhe MVCC interno e GTL-77
explicitamente exigiu uma origem de evento sem esse truque.

Contraexemplo:

```text
scan S1: K não existe -> insert, is_new=true, revision=1
scan S2: K segue ativa -> conflict update, is_new=false, revision=2
scan S3: K foi resolved e reaparece -> conflict update, is_new=false, revision=N
```

- publicar somente `is_new=true` perde a reativação de S3;
- publicar todo `RETURNING` duplica a anomalia em S2 e a cada 30 segundos;
- incrementar revision em todo “seen” cria tráfego/revisão sem mudança semântica.

Correção mínima: separar `INSERT ... ON CONFLICT DO NOTHING RETURNING` (primeiras detecções) de um
`UPDATE ... RETURNING` condicionado a `status='resolved' OR details IS DISTINCT FROM ...`
(reativações/mudanças). Um terceiro update idempotente deve apenas marcar `scan_id/last_seen_at`
das ativas inalteradas sem emitir e sem avançar revisão semântica.

### 1.4 Arrays paralelos

Os quatro `unnest` em V8:94-97 são aceitos, mas dependem de arrays de mesmo comprimento. Se um
chamador fornecer comprimentos diferentes, PostgreSQL preenche as colunas curtas com `NULL` e a
query falha por `NOT NULL`; rollback evita parcialidade, porém o contrato deveria usar um input
estruturado (`jsonb_to_recordset`/composite) ou validar comprimentos antes da query. Como a
transação é por workspace, `workspace_id` deve ser um único parâmetro autoritativo, não um array
fornecido por candidato.

## 2. Transação, scan e races

### 2.1 Função transacional ausente

V8:86 diz “dentro da transação”, mas o único Go mostrado chama:

```go
reconcileWorkspaceTransaction(ctx, conn, bus, ws.ID)
```

em V8:182 sem receber/verificar `error`. Não há `BeginTx`, `defer Rollback`, quatro chamadas,
upserts, reconcile, coleta de deltas, `Commit` ou publish pós-commit concretos. Portanto não é
possível provar:

- rollback total se detector 2, 3 ou 4 falhar;
- reconcile somente depois de quatro resultados completos;
- zero publish se commit falhar;
- publish somente dos `RETURNING` deste commit;
- retry idempotente do mesmo scan.

O contrato mínimo precisa retornar `error`, e o loop deve registrar/continuar sem confundir erro com
workspace limpo.

### 2.2 `ReadCommitted` não produz snapshot dos quatro detectores

O diagrama V8:21 escolhe `ReadCommitted`. Em PostgreSQL, cada statement recebe um snapshot novo.

Contraexemplo:

```text
Q1 observa issue I como todo/sem task.
Outra transação cria task ativa e muda I para in_progress.
Q2 observa o estado novo.
Q3/Q4 completam.
Upsert persiste a observação antiga de Q1 e publica após commit.
```

A transação faz rollback em erro, mas não transforma quatro leituras em um snapshot consistente.
Use `RepeatableRead` para semântica “snapshot do início do workspace”, declarando SLA de até um tick,
ou revalide candidatos antes do commit com regra explícita. O V8 não fecha resolve-during-scan.

### 2.3 Reconcile apenas após scan completo

`scan_id != $2` em V8:120-123 é uma boa guarda quando:

1. um único `scan_id` é gerado por tentativa;
2. os quatro detectores terminam;
3. todos os candidatos são marcados;
4. só então o reconcile executa na mesma transação.

Mas o documento não mostra esse fluxo. Também não define o caso “zero candidatos”: os arrays vazios
devem ser uma query válida/no-op antes de resolver todas as linhas anteriores.

### 2.4 Issue/workspace apagado e cascade

As FKs `ON DELETE CASCADE` removem a linha durável, mas não geram evento `cleared`. O frontend real
já recebe `issue:deleted`; o contrato deve dizer que esse evento purga todas as anomalias da issue e
testar essa ordem. Se o delete ocorrer:

- antes do insert dentro do scan: o insert pode falhar em FK e deve rollback;
- após commit e antes do publish: o evento “active” pode ser publicado para linha já apagada;
- após snapshot e sem WS: só polling periódico converge.

V8 não define esses casos.

## 3. Revision race: bloqueio estrutural

O schema tem `revision` **em cada linha** (V8:56). A resposta promete
`workspace_revision` (V8:196) e o cliente aceita evento somente quando
`event.revision > snapshot.workspace_revision` (V8:202), mas nenhuma tabela, sequência ou query
produz uma revisão global por workspace.

Contraexemplo determinístico:

```text
K1 foi vista muitas vezes: row revision=42.
K2 foi criada recentemente: row revision=1.
GET retorna workspace_revision=42 (se for MAX).
K2 é resolvida: clear de K2 recebe row revision=2.
Cliente testa 2 > 42, descarta o clear e mantém K2 ativa.
```

Também não existe revisão quando o último registro é podado/cascadeado. A correção é uma linha
workspace-scoped de estado/revision incrementada uma vez por transação que produz mudança
semântica, ou um outbox/sequence com ordenação total por workspace. Active e clear devem carregar
essa revisão global, nunca a revisão isolada da anomalia.

## 4. Lock, unlock e conexão

### 4.1 Aquisição — parcial

V8:146-151 verifica `err` e `acquired`. Isso é melhor que V7. Porém, em erro de `Scan`, a query pode
ter sido executada; devolver a conexão ao pool sem tentar unlock/close é uma incerteza fail-open.
O caminho `err != nil` deve fechar/destruir a sessão, não simplesmente `Release`.

### 4.2 `Hijack` está incorreto e vaza o lock

V8:160-163:

```go
if unlockErr != nil || !unlocked {
    conn.Hijack()
    return
}
```

O pgx instalado é explícito em
`github.com/jackc/pgx/v5@v5.9.2/pgxpool/conn.go:70-83`:

```text
Hijack assumes ownership of the connection from the pool.
Caller is responsible for closing the connection.
```

`Hijack()` apenas retorna `*pgx.Conn`; não destrói a sessão. Descartar esse retorno remove a conexão
do pool mas mantém socket/sessão/lock vivos. A forma segura deve capturar e fechar:

```go
raw := conn.Hijack()
closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := raw.Close(closeCtx); err != nil { ... }
```

Se nem o close confirmar, a sessão deve ser tratada como incidente observável. Testes precisam
provar: unlock `true`, unlock `false`, erro/cancelamento, close do hijacked conn e nenhuma reentrada
do lock na mesma sessão.

## 5. Paginação não está demonstrada como keyset

Há uma contradição literal:

- diagrama V8:20 diz `ORDER BY id LIMIT/OFFSET`;
- comentário V8:168 diz keyset;
- código V8:173-176 passa `AfterID` e `LimitRows`;
- nenhuma query `ListWorkspacesPaged` é mostrada ou existe no repo.

O repo real só tem `ListWorkspaces` por membro, ordenado por `created_at`
(`server/pkg/db/queries/workspace.sql:1-8`). Para ser keyset, o SQL futuro precisa conter
literalmente:

```sql
SELECT id
FROM workspace
WHERE (sqlc.narg('after_id')::uuid IS NULL OR id > sqlc.narg('after_id'))
ORDER BY id
LIMIT sqlc.arg('limit_rows');
```

`Limit` sozinho não é keyset e `OFFSET` reprova. O cursor inicial `pgtype.UUID{Valid:false}` precisa
ser aceito como NULL. O teste deve cobrir múltiplas páginas, UUID adicionado atrás/à frente do cursor,
workspace apagado entre página e reconcile e erro de página. V8:177 transforma erro em `break`
silencioso; deve logar/meter erro separadamente de EOF.

## 6. API, hydration, WS e commit-crash

### 6.1 Membership route — PASS

O caminho V8:193-195 encaixa no grupo real:

- auth global em `server/cmd/server/router.go:546-549`;
- `/api/workspaces/{id}` em `router.go:582-585`;
- `RequireWorkspaceMemberFromURL(queries,"id")` em `router.go:587-603`;
- middleware valida identidade, UUID, task-token binding e membership em
  `server/internal/middleware/workspace.go:171-245`.

A rota deve ser registrada dentro desse grupo member-level.

### 6.2 Hydration active/resolved — BLOCK

Não há SQL de listagem nem handler concreto. A API não define:

- se `anomalies` inclui somente `status='active'` ou também resolved;
- ordem determinística;
- como um snapshot substitui/purga chaves locais ausentes;
- como expõe revisões globais;
- limite/tamanho;
- comportamento após delete/cascade/prune.

Para o board, o snapshot autoritativo deve retornar as ativas, ordenadas pela chave completa, e sua
aplicação deve substituir o conjunto workspace-scoped. Histórico resolved pode ser endpoint
separado; misturá-lo no mesmo array sem filtro gera badges falsos.

### 6.3 GET×WS ordering — BLOCK

V8:199-202 diz subscribe → GET → aplicar evento se revision maior, mas não define buffer. Se evento
chegar antes do GET terminar e for aplicado imediatamente, um snapshot antigo pode sobrescrevê-lo.
Se for guardado, faltam estrutura, arquivo, limite, cleanup e testes. O `WSClient` real mantém
handlers e callbacks de reconnect, mas não possui revision/buffer
(`packages/core/api/ws-client.ts:27-43,93-137,150-162`).

### 6.4 Commit→crash — BLOCK

O bus real é síncrono e in-process (`server/internal/events/bus.go:27-31,57-87`). Se o processo
commitar e morrer antes de `Publish`, o V8 só faz novo GET em mount/reconnect. Um cliente conectado
pode ficar stale indefinidamente.

Correções aceitáveis:

- outbox transacional + dispatcher idempotente; ou
- polling/refetch periódico bounded, além de mount/reconnect.

V8 não escolhe nenhum. “WS best-effort” não fecha convergência sem poll periódico.

## 7. Retention, down e lifecycle

O prune de sete dias é uma decisão explícita e melhor que V7, mas está incompleto:

- não diz onde/quando roda nem se usa o lock;
- não possui índice para o predicado;
- não há teste de boundary exatamente 7 dias;
- não há regra para tipos descontinuados;
- cascade remove active/resolved sem clear;
- down usa `CASCADE`, com blast radius futuro;
- não há regra de rollback live versus up/down em DB efêmero.

O estado active/resolved persiste em restart/failover no banco, mas o protocolo de eventos e revisão
ainda não.

## 8. `FILES_LOCKED`: 20 entradas, conjunto não completo

Verificação no checkout atual:

- **9 existem**: `models.go`, `runtime_sweeper.go`, `main.go`, `router.go`, `events.go`,
  `packages/core/types/events.ts`, `packages/core/api/schema.ts`, `board-view.tsx`,
  `board-card.tsx`;
- **11 ainda não existem** e seriam novos: duas migrations placeholder, query/gerado Kanban,
  service+test, handler+test, type Kanban, hook e board-view test.

Novos arquivos são aceitáveis como locks, então “não existe ainda” não é defeito por si só. O
defeito é a omissão de consumidores obrigatórios:

1. `packages/core/types/index.ts` para exportar `kanban.ts`;
2. `packages/core/api/client.ts` para o GET;
3. módulo/query key do TanStack Query;
4. store workspace-scoped de snapshot/revision/buffer e seus testes;
5. `packages/core/api/ws-client.ts` ou módulo que ligue reconnect;
6. teste de hook/reconnect/ordering;
7. locale/ARIA do badge;
8. migration/SQL/EXPLAIN/up-down tests;
9. sweeper/lock integration tests;
10. eventual schema/query/service de workspace revision/outbox;
11. qualquer gerado adicional produzido pelo sqlc deve ser fingerprintado; a lane
    `server/pkg/db/generated/**` é global exclusiva no Z01.

Logo a lista “factual completa” V8:219-240 não é completa.

## 9. Testes e waves

V8 não lista os “11 testes exatos” pedidos desde GTL-67/77. Ele nomeia apenas:

- `server/internal/service/kanban_integrity_test.go`;
- `server/internal/handler/kanban_integrity_test.go`;
- `packages/views/issues/components/__tests__/board-view.test.tsx`.

Não há casos, fixtures, comandos ou gates para:

- positive/negative dos quatro detectores;
- query 4 multi-dep/parent dedup e tenant guard;
- upsert idempotente, reactivation e first/last timestamps;
- resolução K1 mantendo K2 ativa;
- falha em cada detector e rollback;
- `ReadCommitted`/resolve during scan;
- duas réplicas;
- lock false/error/hijack close;
- paginação multi-page;
- active-only hydration/auth/cross-tenant;
- GET×WS inversions, revision race, reconnect e commit-crash polling;
- issue/workspace delete e retention;
- listener isolation completo;
- migration up/down/up e EXPLAIN.

As waves V8:206-214 têm ordem macro aceitável, mas nenhuma wave pode começar enquanto Wave 0 não
fixar o contrato de revision/outbox-or-poll, tenant enforcement, lifecycle e locks. Como o veredito
é BLOCK, **não há pacote Wave 1 implementável para liberar**.

## 10. Claim READ-ONLY

Não há contradição se “READ-ONLY” em V8:1,7,242 descreve apenas a produção do documento: o autor
declara que não executou mudanças. O monitor proposto é intencionalmente mutável
(`INSERT/UPDATE/DELETE` em V8:90-131) e não deve ser chamado de runtime read-only em documentação,
permissões ou observabilidade. A próxima versão deve escrever “design produzido em modo
read-only; implementação runtime muta somente a tabela de integridade”.

## 11. Correções mínimas para V9

1. Criar estado de revisão global por workspace ou outbox ordenado; remover comparação entre
   revisões de linhas distintas.
2. Separar insert-new, reactivate/change e mark-seen; eliminar `xmax` e evento por scan inalterado.
3. Mostrar SQL completo dos quatro detectores, query keyset e query active hydration.
4. Mostrar `reconcileWorkspaceTransaction` completo com `RepeatableRead`, rollback, commit e publish
   pós-commit; tratar seu erro no loop.
5. Corrigir `Hijack` capturando e fechando `*pgx.Conn`; fail-closed também no erro de aquisição.
6. Escolher outbox ou polling periódico para commit→crash e desenhar buffer GET×WS.
7. Fechar tenant FK/guard, lifecycle CHECKs, retention index e down sem cascade implícito.
8. Expandir locks e enumerar testes/gates com nomes e comandos.

## Veredito final

**BLOCK.** O V8 melhora materialmente o V7, mas ainda contém uma falha operacional crítica no
cleanup do advisory lock e uma falha de modelo no revisionamento. SQL/tx/API/WS/testes não sustentam
as garantias declaradas. Migration permanece placeholder; nenhuma Wave 1 está liberada.
