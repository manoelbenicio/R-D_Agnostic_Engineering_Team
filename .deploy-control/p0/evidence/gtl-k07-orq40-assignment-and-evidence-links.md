# GTL-K07 — Assignment do ORQ-40 (feito) e links de evidência (PARADO com zero mutações)

Executor/registrar: Codex56#A (`w7:p3`) · UTC 2026-07-27T13:03Z

## Parte A — ORQ-40 atribuído e verificado ✔

```text
GET /api/agents?workspace_id=20fce817-...   -> nome "Opus48-A": MATCHES 1
   agent_uuid = 2c042fdd-9a76-4da1-9b02-c2332a736a86      (exatamente um; nada inventado)

PUT /api/issues/d2001a24-af70-4367-8828-2225ed43ad84?workspace_id=...   -> HTTP=200
   body enviado: {"assignee_type":"agent","assignee_id":"2c042fdd-9a76-4da1-9b02-c2332a736a86"}
   (nenhum outro campo enviado; UpdateIssue é parcial — issue.go:2308-2352)

GET-back /api/issues/d2001a24-...?workspace_id=...
   id= d2001a24-af70-4367-8828-2225ed43ad84   identifier= ORQ-40   number= 40
   assignee_type= agent   assignee_id= 2c042fdd-9a76-4da1-9b02-c2332a736a86
   title= "Alinhar Codex CLI entre ORQ1 e ORQ2 (higiene de vers…"   status= todo   priority= low
   desc_len= 2286
```

Assignment provado. `title`, `status`, `priority` e `description` intactos (não foram enviados).
Nenhuma execução iniciada: o card segue `todo` e o alvo do card continua sendo trabalho futuro.

## Parte B — **STOP, zero mutações**. O endpoint aditivo existe, mas dispara agente.

Endpoint aditivo encontrado (existe, então esta não é a razão da parada):

```text
server/cmd/server/router.go:738   r.Post("/comments", h.CreateComment)
server/cmd/server/router.go:739   r.Get("/comments",  h.ListComments)
server/cmd/server/router.go:737   r.Post("/comments/trigger-preview", h.PreviewCommentTriggers)
handler/comment.go:771-777        CreateCommentRequest{Content, Type, ParentID, AttachmentIDs, SuppressAgentIDs}
```

Razão da parada — **comentário em issue com assignee de agente ENFILEIRA task, sem precisar de
`@menção`**:

```text
handler/comment.go:1122-1138  (enqueueCommentAgentTriggers)
    case commentTriggerSourceIssueAssignee:
        if trigger.Squad != nil { ... EnqueueTaskForSquadLeader(...) ; continue }
        if _, err := h.TaskService.EnqueueTaskForIssue(ctx, issue, triggerCommentID); err != nil { ... }
    case commentTriggerSourceMentionSquadLeader: ... EnqueueTaskForSquadLeader(...)
    case commentTriggerSourceMentionAgent:       ... EnqueueTaskForMention(...)
```

Os quatro cards-alvo **têm assignee de agente** (medido no K03):

| card | UUID | assignee_type / assignee_id |
|---|---|---|
| ORQ-12 | `8b1419f5-c9d4-466c-adff-cded98f91d29` | agent / `2c042fdd-9a76-4da1-9b02-c2332a736a86` |
| ORQ-13 | `2aefcb3d-97b3-4e70-a4b0-ae3729b7981d` | agent / `4dc3b1f5-451f-4f68-b71e-91e856fed286` |
| ORQ-14 | `19ab8cfe-0a95-4a79-af99-4ee384326021` | agent / `3db514db-810e-4393-817e-eb3707ce59ae` |
| ORQ-23 | `6230b5c0-c57a-4b88-a7df-84899057d0c8` | agent / `4dc3b1f5-451f-4f68-b71e-91e856fed286` |

Consequência: um comentário "inofensivo" de evidência em qualquer um dos quatro **iniciaria execução
de agente** — proibido explicitamente por este dispatch ("Do not start execution") — e três deles
(ORQ-12, ORQ-13, ORQ-23) estão entre as **nove issues preservadas sem rerun**. Existe mitigação no
contrato (`suppress_agent_ids`, filtrada em `comment.go:1099-1120`), mas usá-la exigiria eu enumerar
corretamente **todos** os gatilhos (assignee, squad leader e menções) e um único gatilho não suprimido
enfileiraria task numa issue preservada. Isso é precisamente o caso de "contrato ambíguo/risco" previsto
na condição de parada, portanto **não emiti nenhum POST de comentário**.

### Caminho seguro proposto para a Parte B (não executado)

1. `POST /api/issues/<uuid>/comments/trigger-preview` com o texto exato — é **preview**, existe para
   isso, e mostra quais agentes seriam disparados; deve ser rodado antes de qualquer comentário.
2. Só então `POST /comments` com `suppress_agent_ids` cobrindo **todos** os agentes retornados pelo
   preview, com re-verificação de que a fila permaneceu em zero:
   `SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','dispatched','running','waiting_local_directory')`
   antes e depois.
3. `GET /comments` para provar o comentário criado.
4. Alternativa sem risco de gatilho: manter o mapeamento apenas em evidência durável
   (`gtl-legacy-kanban-mapping-opus48b.md`, já entregue) e anexar ao card só quando o GTL autorizar a
   sequência preview+suppress.

Os quatro textos de evidência a anexar já estão prontos no K03 (§4), com verdicts e dependências:
ORQ-12 = GTL-03 (auditoria, sem implementação); ORQ-13 = GTL-25 design + GTL-I03/GTL-68 PASS condicional
a 3 critérios; ORQ-14 = GTL-17/GTL-26 PASS com retificação de premissa stale; ORQ-23 = GTL-37 **BLOCK**.

## ADENDO CRÍTICO (13:03Z) — minhas ações de registrar DISPARARAM EXECUÇÃO

Ao verificar a fila depois da Parte A, encontrei **2 tasks `running`**, ambas criadas exatamente nos
instantes das minhas duas ações de board:

```text
select id,status,issue_id,created_at,trigger_comment_id is not null from agent_task_queue
where status in ('queued','dispatched','running','waiting_local_directory') order by created_at;

f460ed12-d44d-4634-8032-ad6d6e05264e | running | c03941bc-… (ORQ-39) | 2026-07-27 12:53:27.431+00 | f
07cdc53b-e87c-42f1-8b90-593ee545c920 | running | d2001a24-… (ORQ-40) | 2026-07-27 13:00:10.635+00 | f
```

- 12:53:27 = meu `POST` do ORQ-39 **com assignee Opus48-B** (K05).
- 13:00:10 = meu `PUT` de assignee no ORQ-40 (K07-A).
- `trigger_comment_id` é NULL nas duas: **não** foi comentário. É o gatilho **on-assign**.

Causa no código, confirmada:

```text
handler/issue.go:2530-2536   // Reconcile task queue when assignee changes.
                             if assigneeChanged {
                                 h.TaskService.CancelTasksForIssue(...)
                                 if h.shouldEnqueueAgentTask(...) { h.TaskService.EnqueueTaskForIssue(...) }
                             }
handler/issue.go:3035,3050   mesmo enqueue no caminho de criação/transição
```

**Consequência:** atribuir agente a um card — na criação ou por `PUT` — **inicia execução paga**. Eu
tratei `POST`/`PUT` de assignee como operações puramente de registro; elas não são. As duas tasks estão
`running` e eu **não** as cancelei: cancelar é mutação de task não autorizada e pode deixar estado
parcial. Nenhuma das nove issues preservadas foi afetada (ORQ-39 e ORQ-40 são cards novos).

Isto reforça, por medição, a razão da parada da Parte B: se `assignee` sozinho já enfileira, um
comentário nos quatro cards com assignee de agente enfileiraria em issues preservadas.

**Decisão pedida ao GTL (não executo sem ruling):** (a) deixar as duas tasks concluírem, (b) cancelar
via serviço, ou (c) política nova de "registro sem assignee" — criar/atribuir apenas com `assignee`
vazio e atribuir só quando o trabalho for realmente autorizado.



- Parte A: um único `PUT`, apenas dois campos; nada mais alterado; assignment confirmado por GET-back.
- Parte B: **zero mutações** — nenhum `POST /comments`, nenhum `trigger-preview` chamado, nenhuma
  descrição/título/status/prioridade/assignee dos quatro cards tocado.
- Não criei card; não toquei ORQ-31, ORQ-38 nem ORQ-39; não instalei, atualizei, reiniciei nem editei repo.
- Não afirmo que `suppress_agent_ids` seja insuficiente: afirmo que sem o preview não posso provar a
  cobertura completa dos gatilhos, e o custo do erro é enfileirar task em issue preservada.

---

## CONTENÇÃO EXECUTADA (ORQ-41, autorizada pelo GTL) — 13:07–13:08Z

### Pré-cancelamento, read-only (13:07:40Z)

```text
TASK f460ed12-… | running   | issue c03941bc (ORQ-39) | created 12:53:27.431+00 | started=t
TASK 07cdc53b-… | completed | issue d2001a24 (ORQ-40) | created 13:00:10.635+00 | started=t
mensagens por tipo (contagem, sem ler conteúdo):
  f460ed12: tool_use 54 · tool_result 2 · thinking 99 · text 7   (total 175)
  07cdc53b: tool_use 30 · tool_result 0 · thinking 51 · text 11  (total  92)
ACTIVE_TOTAL global = 1
```

`07cdc53b` (ORQ-40) **já estava `completed`, terminal** — nenhum cancel foi emitido para ela. Só a task
de ORQ-39 estava viva.

### Cancelamento — uma emissão efetiva

```text
POST /api/tasks/f460ed12-…/cancel                          -> HTTP 400 {"error":"workspace_id or workspace_slug is required"}  (sem efeito)
POST /api/tasks/f460ed12-…/cancel?workspace_id=20fce817-…   -> HTTP 200

GET-back (13:08:18Z):  f460ed12-… = cancelled  ·  07cdc53b-… = completed
ACTIVE_GLOBAL = 0 · ACTIVE_ORQ39_40 = 0
MSG_PRESERVED: f460ed12 = 175 · 07cdc53b = 92   (nada apagado)
```

### Limpeza de assignee — contrato verificado, não adivinhado

`handler/issue.go:2368-2385`: `if _, ok := rawFields["assignee_type"]; ok { … } else { pgtype.Text{Valid:false} }
// explicit null = unassign`, e o mesmo para `assignee_id` (`pgtype.UUID{Valid:false}`).

```text
PUT /api/issues/c03941bc-…?workspace_id=…  {"assignee_type":null,"assignee_id":null} -> HTTP 200
PUT /api/issues/d2001a24-…?workspace_id=…  {"assignee_type":null,"assignee_id":null} -> HTTP 200

GET-back:
  ORQ-39  assignee_type=None assignee_id=None  status=in_progress  priority=high  desc_len=2182
  ORQ-40  assignee_type=None assignee_id=None  status=blocked      priority=low   desc_len=2286
fila após (13:08:49Z): GLOBAL=0 · ORQ39_40 ativo=0 · histórico dos dois: cancelled 1 + completed 1
```

Ambos comprovadamente **sem assignee**; `title` e `priority` preservados.

### Caveats de efeito parcial

1. As duas tasks **produziram trabalho antes da contenção**: 267 mensagens (175 + 92), das quais 84
   `tool_use`. `tool_use` prova ação executada, não conclusão — efeitos colaterais desses runs no
   ambiente **não são conhecidos daqui** e não devem ser presumidos nulos. Nada apagado.
2. O `status` dos dois cards mudou **durante a execução, não pelo meu PUT**: ORQ-39 `in_progress`,
   ORQ-40 `blocked`. Enviei apenas os dois campos de assignee. Não revertí status: reverter seria
   mutação não autorizada e apagaria evidência do incidente.
3. `07cdc53b` **completou** antes da ordem de contenção; o custo dessa execução já foi incorrido.
4. `desc_len` do ORQ-40 segue 2286, idêntico ao medido no K07 antes do PUT; para o ORQ-39 não havia
   medição anterior, então não afirmo alteração.
5. Nenhum comentário, delete, edição de código, escrita SQL direta, deploy ou restart. Nenhuma das
   nove issues preservadas tocada. ORQ-39/40 **não** foram rerodadas.

---

## RECONCILIAÇÃO DE STATUS (ORQ-41, autorizada) — 13:13Z

### Verificação de contrato antes de mutar (por que status-only é seguro aqui)

`handler/issue.go:2555-2564` — o único enqueue no caminho de mudança de status exige **três** condições
simultâneas:

```go
if statusChanged && !assigneeChanged &&
    prevIssue.Status == "backlog" && issue.Status != "done" && issue.Status != "cancelled" &&
    !h.isAgentRunningOnIssue(r, actorType, issue) {
        if h.isAgentAssigneeReady(r.Context(), issue) { h.TaskService.EnqueueTaskForIssue(...) }
        if h.isSquadLeaderReady(r.Context(), issue)   { h.enqueueSquadLeaderTask(...) }
}
```

Aplicado ao ORQ-39: `prevIssue.Status` era **`in_progress`**, não `backlog` ⇒ o gate não abre; e mesmo se
abrisse, `isAgentAssigneeReady` falharia porque o card está **sem assignee**. Também confirmei que o
enqueue das linhas ~3035/3050 pertence a `BatchUpdateIssues` (espelha o mesmo gate) e não ao `PUT`
individual. Portanto um `PUT` contendo **apenas** `status` não enfileira.

### Execução

```text
pré  (GET, workspace_id):  ORQ-39 status=in_progress assignee=None priority=high desc_len=2182 title_len=55 project=None
PUT  /api/issues/c03941bc-…?workspace_id=…   body {"status":"blocked"}   -> HTTP 200
pós  (GET, workspace_id):  ORQ-39 status=blocked  assignee_type=None assignee_id=None
                           priority=high  desc_len=2182  title_len=55  project=None
```

`title`, `description`, `priority`, `assignee` e `project` idênticos antes/depois (nenhum foi enviado).

### Verificação de fila e do ORQ-40 (13:13:56Z)

```text
ORQ39_ACTIVE  = 0
GLOBAL_ACTIVE = 0
ORQ-40        = blocked · unassigned=t     (não tocado neste passo)
```

Nenhum comentário, nenhuma atribuição, nenhum rerun, nenhuma escrita SQL direta, nenhuma edição de
código, nenhum deploy ou restart. Os dois cards do incidente ficam `blocked` e sem assignee, com a
evidência das tasks (cancelled + completed, 267 mensagens) preservada.

---

## RECONCILIAÇÃO DE TÍTULO DO ORQ-41 (autorizada) — 13:21Z

### Verificação de contrato antes de mutar

Um `PUT` contendo **apenas** `title` não pode enfileirar nem cancelar, porque os três gatilhos do
handler dependem de `assigneeChanged` ou `statusChanged`, e ambos são falsos quando os campos não são
enviados:

```go
issue.go:2493  assigneeChanged := (req.AssigneeType != nil || req.AssigneeID != nil) && …
issue.go:2495  statusChanged   := req.Status != nil && prevIssue.Status != issue.Status
issue.go:2530  if assigneeChanged { CancelTasksForIssue; if shouldEnqueueAgentTask { EnqueueTaskForIssue } }
issue.go:2555  if statusChanged && !assigneeChanged && prevIssue.Status == "backlog" … { Enqueue… }
issue.go:2569  if statusChanged && issue.Status == "cancelled" { CancelTasksForIssue }
issue.go:2577  if statusChanged { notifyParentOfChildDone(…) }      // nem comentário de sistema dispara
```

Com body `{"title": …}`: `req.AssigneeType`/`req.AssigneeID`/`req.Status` são `nil` ⇒ `assigneeChanged`
e `statusChanged` falsos ⇒ **nenhum enqueue, nenhum cancel, nenhum comentário de sistema**. Contrato
claro, sem ambiguidade.

### Execução e prova

```text
pré  (GET, workspace_id): ORQ-41 | title="Add safe non-triggering documentation comments to assigned i…"
                          status=todo | priority=high | assignee=None | desc_len=2209 | number=41
PUT  /api/issues/666f1ead-…?workspace_id=…  body {"title":"Decouple Kanban metadata from paid task execution"} -> HTTP 200
pós  (GET, workspace_id): ORQ-41 | uuid=666f1ead-7fe9-4051-bab9-5d0a936c4701 | number=41
                          title="Decouple Kanban metadata from paid task execution"
                          status=todo | priority=high | assignee_type=None | assignee_id=None | desc_len=2209
fila (13:21:13Z):         GLOBAL_ACTIVE = 0 · ORQ41_ACTIVE = 0
```

Estáveis por medição: **UUID** `666f1ead-…`, **number** 41, **status** `todo`, **priority** `high`,
**assignee** vazio e **description** com os mesmos 2209 bytes. Nenhum comentário, atribuição, criação
de card, mudança de status/descrição ou patch de código.
