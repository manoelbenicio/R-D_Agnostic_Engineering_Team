# GTL-60 — Deep SQL Audit do Kanban Integrity Monitor V5

- Auditor: Codex56#B (`w7:p4`)
- Data UTC: 2026-07-27
- Escopo: auditoria READ-ONLY de `gtl-kanban-integrity-monitor-v5.md` e `gtl-kanban-v5-peer-review.md`
- Veredito: **BLOCK**
- Não executado: SQL, conexão com banco, build, teste, mutação de código/board/deploy

## Resultado executivo

O `PASS` do peer review deve ser revogado. A V5 contém falsos positivos determinísticos nas
Anomalias 1, 2 e 3; pode emitir mais de uma linha/evento para a mesma issue na Anomalia 4; e
aceita `d.type='blocks'` sem existir consumidor no repositório que prove essa direção semântica.
A suíte de sete testes perdeu oito gates adversariais já pedidos, inclusive os contraexemplos
que detectariam os regressos.

| Item | Veredito | Evidência literal |
|---|---|---|
| Anomalia 3 usa a última task | **CONFIRMADO / BLOCK** | V5:67-73 e 77; requisito correto anterior: V3:244-259 e peer anterior:147-154 |
| `UNION ALL` pode duplicar pai + dependency | **CONFIRMADO / BLOCK** | V5:82-98; peer afirma “sem ... duplicatas” em peer-review:28 |
| `d.type='blocks'` tem direção não provada | **CONFIRMADO / BLOCK** | V5:93-98; schema só enumera valores em `001_init.up.sql`:89-94; não há consumer CRUD |
| Gates V3/adversariais sumiram | **CONFIRMADO / BLOCK** | V5:115-123 lista só sete happy paths; lista exigida em peer anterior:167-188 |

## 1. Anomalia 3 — `latest_task` produz falso positivo

Trecho literal V5 (`gtl-kanban-integrity-monitor-v5.md:67-77`):

```sql
LEFT JOIN LATERAL (
    SELECT status
    FROM agent_task_queue atq
    WHERE atq.issue_id = i.id
    ORDER BY created_at DESC
    LIMIT 1
) latest_task ON TRUE
...
AND (latest_task.status IS NULL OR latest_task.status != 'completed');
```

Contraexemplo SQL autocontido, apresentado para revisão e **não executado**:

```sql
WITH issue(id, status) AS (
  VALUES ('I', 'done')
),
agent_task_queue(issue_id, status, created_at, completed_at) AS (
  VALUES
    ('I', 'completed', TIMESTAMPTZ '2026-07-27 10:00Z',
                       TIMESTAMPTZ '2026-07-27 10:01Z'),
    ('I', 'queued',    TIMESTAMPTZ '2026-07-27 10:02Z', NULL)
),
latest_task AS (
  SELECT status
  FROM agent_task_queue
  WHERE issue_id = 'I'
  ORDER BY created_at DESC
  LIMIT 1
)
SELECT i.id
FROM issue i
LEFT JOIN latest_task ON TRUE
WHERE i.status = 'done'
  AND (latest_task.status IS NULL OR latest_task.status <> 'completed');
-- V5 retorna I: falso positivo; existe conclusão efetiva com completed_at.
```

Correção mínima:

```sql
AND NOT EXISTS (
  SELECT 1
  FROM agent_task_queue atq
  WHERE atq.issue_id = i.id
    AND atq.status = 'completed'
    AND atq.completed_at IS NOT NULL
)
```

Esta é exatamente a regra V3 (`gtl-kanban-integrity-monitor-v2.md:244-259`) e o gate mínimo é
`TestAnomaly3_CompletedThenRetryQueued_NoAnomaly`. A V5 não o lista.

## 2. Anomalia 4 — `UNION ALL` não deduplica

Trecho literal V5 (`gtl-kanban-integrity-monitor-v5.md:82-98`) seleciona primeiro o pai e depois,
via `UNION ALL`, a mesma issue alcançada por `issue_dependency`. `UNION ALL` preserva ambas as
linhas; `dependency_source` diferente impediria deduplicação mesmo com `UNION`.

Contraexemplo SQL autocontido, apresentado para revisão e **não executado**:

```sql
WITH issue(id, parent_issue_id, status) AS (
  VALUES
    ('C', 'P', 'blocked'),
    ('P', NULL, 'done')
),
issue_dependency(issue_id, depends_on_issue_id, type) AS (
  VALUES ('C', 'P', 'blocked_by')
)
SELECT i.id AS issue_id, 'parent_dependency' AS dependency_source,
       p.id AS dependency_id
FROM issue i
JOIN issue p ON i.parent_issue_id = p.id
WHERE i.status = 'blocked' AND p.status IN ('done', 'cancelled')
UNION ALL
SELECT i.id, 'issue_dependency', dep.id
FROM issue i
JOIN issue_dependency d ON d.issue_id = i.id
JOIN issue dep ON d.depends_on_issue_id = dep.id
WHERE i.status = 'blocked'
  AND d.type = 'blocked_by'
  AND dep.status IN ('done', 'cancelled');
-- Retorna (C,parent_dependency,P) e (C,issue_dependency,P).
```

Logo, a afirmação do peer review de que V5:89 usa `UNION ALL` “sem ... duplicatas”
(`gtl-kanban-v5-peer-review.md:28`) é falsa. Também há N linhas para N dependências resolvidas,
enquanto o requisito anterior era um evento por issue.

Correção mínima: normalizar primeiro o par `(issue_id, dependency_id)` sem a origem e depois
agrupar uma única linha por issue.

```sql
WITH resolved AS (
  SELECT i.id AS issue_id, p.id AS dependency_id
  FROM issue i
  JOIN issue p ON p.id = i.parent_issue_id
  WHERE i.workspace_id = $1
    AND i.status = 'blocked'
    AND p.status IN ('done', 'cancelled')
  UNION
  SELECT i.id, dep.id
  FROM issue i
  JOIN issue_dependency d ON d.issue_id = i.id
  JOIN issue dep ON dep.id = d.depends_on_issue_id
  WHERE i.workspace_id = $1
    AND i.status = 'blocked'
    AND d.type = 'blocked_by'
    AND dep.status IN ('done', 'cancelled')
)
SELECT issue_id, array_agg(dependency_id ORDER BY dependency_id) AS dependency_ids
FROM resolved
GROUP BY issue_id;
```

Gate mínimo:
`TestAnomaly4_ParentAlsoDependency_EmitsSingleIssueEvent`, além do já exigido
`TestAnomaly4_MultipleResolvedDependencies_EmitsSingleEvent`.

## 3. Direção semântica de `blocks`

Busca textual no código real encontrou somente:

- `server/migrations/001_init.up.sql:89-94`, que cria `issue_dependency` e apenas valida o enum;
- `server/pkg/db/generated/models.go:396-400`, modelo gerado sem comportamento.

Não existe query CRUD, handler, service, frontend ou CLI que leia/escreva `issue_dependency`,
`depends_on_issue_id` ou `blocked_by`. `git log -S` encontra apenas o commit que introduziu a
migration, não um consumer. Portanto o schema prova que `'blocks'` é valor válido, mas **não**
prova que tem a mesma direção de `'blocked_by'`.

O nome `depends_on_issue_id` estabelece a leitura segura da linha
`(issue_id=A, depends_on_issue_id=B)` como “A depende de B”; nessa direção, o tipo coerente para
detectar que A pode sair de `blocked` é `blocked_by`. Se `type='blocks'` descreve a relação a partir
de A, então A bloqueia B; B estar `done` não prova que A deixou de estar bloqueada. Incluir ambos
silenciosamente pode gerar falso positivo.

Isto também contradiz a correção mínima anterior, que dizia literalmente
`AND d.type = 'blocked_by'` (`gtl-kanban-v4-peer-review.md:85-95`).

Correção mínima: usar somente `d.type = 'blocked_by'`. Se o produto quiser armazenar relações
inversas com `blocks`, primeiro deve definir e testar a orientação no writer/reader; até lá, falhar
fechado.

## 4. Regressões adicionais de falso positivo

### 4.1 Anomalia 1 — atribuição recém-feita

V5:41-46 usa `LEFT JOIN` e `atq.id IS NULL`, sem janela de graça. O `UpdateIssue` real grava
`updated_at = now()` (`server/pkg/db/queries/issue.sql:85-100`), então uma issue antiga atribuída
agora será sinalizada no intervalo legítimo entre update e enqueue.

Correção mínima: restaurar `i.updated_at < now() - INTERVAL '15 seconds'` e testar
`TestAnomaly1_OldIssueAssignedNow_NoAnomalyDuringEnqueue`. Além disso, `atq.id IS NULL` tem o
falso-negativo simétrico: qualquer task terminal histórica suprime a anomalia para sempre; deve
ser `NOT EXISTS` de task ativa.

### 4.2 Anomalia 2 — `waiting_local_directory` é ativa

V5:58 considera apenas `queued`, `dispatched`, `running`. A migration real
`109_agent_task_waiting_local_directory.up.sql:1-15` declara
`waiting_local_directory` como estado em que o daemon aguarda o lock e ainda executará a task.
As queries de produção também o incluem como ativo, por exemplo
`server/pkg/db/queries/agent.sql:206`.

Correção mínima: incluir `'waiting_local_directory'` no `IN` e manter
`TestAnomaly2_InProgressWaitingLocalDirectory_NoAnomaly`.

## 5. Suite V5: gates perdidos

A lista V5 (`gtl-kanban-integrity-monitor-v5.md:115-123`) tem sete nomes, mas não cobre:

1. `TestAnomaly1_OldIssueAssignedNow_NoAnomalyDuringEnqueue`;
2. `TestAnomaly2_InProgressWaitingLocalDirectory_NoAnomaly`;
3. `TestAnomaly3_CompletedThenRetryQueued_NoAnomaly`;
4. `TestAnomaly4_ParentAlsoDependency_EmitsSingleIssueEvent`;
5. `TestAnomaly4_MultipleResolvedDependencies_EmitsSingleEvent`;
6. `TestMonitor_SameAnomalyAcrossTicks_EmitsOnce` e emissão única de clear;
7. `TestMonitor_NoNotificationOrActivityListenerSubscribes`;
8. um teste de lock que prove `try-lock`, consultas e `unlock` no mesmo
   `*pgxpool.Conn`, não apenas concorrência;
9. `TestMonitor_RunsAfterSweeper_NoTransientFalsePositive`;
10. `TestQueries_UseWorkspaceScopedIndex`, com `EXPLAIN` em banco populado e rejeição de
    `Seq Scan` em `issue`.

Os itens 1, 3, 5, 6, 7, 9 e 10 já estavam explicitamente exigidos em
`gtl-kanban-monitor-v2-peer-review.md:167-188`. A lifecycle de dedup/clear estava especificada
em `gtl-kanban-integrity-monitor-v2.md:282-290`; não existe na V5.

Sobre listener isolation: hoje notification/activity registram listeners tipados
(`notification_listeners.go:544-583`; `activity_listeners.go:19-51`), mas o bus também tem
`SubscribeAll` (`server/cmd/server/listeners.go:150-165`; `internal/events/bus.go:49-58`).
Assim, o evento novo deve chegar ao realtime global, mas não aos listeners que escrevem inbox ou
activity. A frase V5:111 “sem listeners secundários” não é prova; o gate precisa instrumentar
esses registradores e demonstrar zero escrita/dispatch secundário.

Sobre lock: V5:108-109 é apenas declaração de desenho e `TestAdvisoryLockConcurrency` não obriga
que unlock e consultas usem a mesma sessão. O teste deve adquirir duas conexões dedicadas,
demonstrar que a segunda não obtém 4247 enquanto a primeira executa, liberar 4247 pela própria
primeira conexão e então demonstrar aquisição pela segunda.

## 6. Correção mínima e ordem de gates

1. SQL: restaurar a graça/`NOT EXISTS` ativo da Anomalia 1; incluir
   `waiting_local_directory` na 2; usar `NOT EXISTS completed + completed_at` na 3; na 4 usar
   apenas `blocked_by`, deduplicar `(issue_id, dependency_id)` e retornar uma linha por issue.
2. Serviço: restaurar estado entre ticks e evento clear; um evento por
   `(workspace_id, issue_id, anomaly_type)`.
3. Concorrência: conexão dedicada única para lock, quatro queries e unlock; tick após o sweeper.
4. Isolamento: realtime global permitido, notification/activity/autopilot proibidos de consumir.
5. Gates: executar todos os dez testes acima, `StrictReadOnly` incluindo `updated_at`, e os quatro
   `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)` somente em banco de teste autorizado/populado.

Critério de PASS: todos os contraexemplos retornam zero anomalias; a Anomalia 4 emite exatamente
um evento; dez ticks iguais emitem um evento inicial e nenhum duplicado; a resolução emite um
clear; a segunda instância não consulta/emite enquanto não possui o lock; nenhum listener
secundário grava; nenhum plano faz `Seq Scan` de `issue` no fixture de escala.

## Conclusão

**BLOCK.** O design V5 não está pronto para implementação. O peer review validou nomes de schema,
mas não semântica temporal, cardinalidade, direção da relação nem cobertura adversarial. Nenhuma
correção exige mudança de produto: são restaurações mínimas de requisitos já aceitos na V3 e em
seu peer review.

