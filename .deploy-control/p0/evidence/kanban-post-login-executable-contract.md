# Contrato executavel pos-login do Kanban (READ-ONLY, mapeado do codigo)

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T17:14Z
- **Gate 0:** delta preflight enviado e **aceito** antes de comecar
- **Modo:** READ-ONLY. **Nenhuma chamada de API, nenhuma mutacao de board, nada executado.** Todos os
  comandos abaixo estao escritos para serem rodados por quem tiver autorizacao, com **IDs resolvidos em
  runtime** — nenhum ID literal inventado.
- **Fonte:** `cmd/server/router.go` e `internal/handler/issue.go` da arvore atual. Cada rota abaixo tem
  numero de linha.
- **Nao duplico** bootstrap nem CFN.

## Convencao usada em todos os comandos

```bash
# preenchidos pelo operador; nada aqui e segredo deste documento
: "${API:?ex.: http://127.0.0.1:18080}"
: "${TOKEN:?PAT mul_... ou JWT; NUNCA colar em evidencia}"
: "${WS_SLUG:?slug do workspace}"
H=(-H "Authorization: Bearer $TOKEN" -H "X-Workspace-Slug: $WS_SLUG" -H "Content-Type: application/json")
```
`X-Workspace-Slug` e obrigatorio: `CreateIssue` resolve o workspace por
`h.resolveWorkspaceID(r)` (`issue.go:2082`) e falha com `400` se nao resolver.

---

## 1. CRIAR PROJETO

Rota: **`POST /api/projects`** — `router.go:781` (`h.CreateProject`), grupo `/api/projects` em `:778`.
```bash
PROJECT_ID=$(curl -sS "${H[@]}" -X POST "$API/api/projects" \
  -d '{"name":"Contrato pos-login","description":"projeto de verificacao"}' \
  | jq -r '.id')
test -n "$PROJECT_ID" -a "$PROJECT_ID" != null || { echo "falha ao criar projeto"; exit 1; }
echo "PROJECT_ID=$PROJECT_ID"
```
**Exige prova runtime:** o conjunto exato de campos aceitos por `CreateProject` e a chave do `id` na
resposta. Eu **nao** li `CreateProject`; inferi o par `name`/`description` do padrao dos demais
handlers. **Marcado como incerto.**

## 2. CRIAR ISSUE **SEM ASSIGNEE**, COM ZERO TASK GARANTIDO

Rota: **`POST /api/issues`** — `router.go:729` (`h.CreateIssue`), grupo `/api/issues` em `:723`.

Payload real, de `CreateIssueRequest` (`issue.go:2043-2054`):
```
title (string, OBRIGATORIO)   description (*string)   status (string)   priority (string)
assignee_type (*string)       assignee_id (*string)   parent_issue_id (*string)
project_id (*string)          start_date (*string)     due_date (*string)
attachment_ids ([]string)
```
```bash
ISSUE_ID=$(curl -sS "${H[@]}" -X POST "$API/api/issues" \
  -d "$(jq -nc --arg p "$PROJECT_ID" '{title:"Contrato pos-login - sem assignee",
        description:"criada sem assignee para provar zero task",
        status:"todo", priority:"none", project_id:$p}')" \
  | jq -r '.id')
echo "ISSUE_ID=$ISSUE_ID"
```
### 2.1 Por que isto garante zero task — cadeia verificada no codigo
O enfileiramento acontece em **exatamente** dois pontos de `issue.go`, ambos **dentro** de
`if assigneeChanged`:
```go
// issue.go:2531-2536  (atualizacao de issue)
	if assigneeChanged {
		h.TaskService.CancelTasksForIssue(r.Context(), issue.ID)
		if h.shouldEnqueueAgentTask(r.Context(), issue) {
			h.TaskService.EnqueueTaskForIssue(r.Context(), issue)
		}
// issue.go:3032-3036  (o outro call site, mesmo padrao)
```
E o predicado (`issue.go:2663-2668` e `:2740-2751`):
```go
func (h *Handler) shouldEnqueueAgentTask(ctx context.Context, issue db.Issue) bool {
	if issue.Status == "backlog" { return false }
	return h.isAgentAssigneeReady(ctx, issue)
}

func (h *Handler) isAgentAssigneeReady(ctx context.Context, issue db.Issue) bool {
	if !issue.AssigneeType.Valid || issue.AssigneeType.String != "agent" || !issue.AssigneeID.Valid {
		return false
	}
	agent, err := h.Queries.GetAgent(ctx, issue.AssigneeID)
	if err != nil || !agent.RuntimeID.Valid || agent.ArchivedAt.Valid { return false }
	return true
}
```
Sem `assignee_type` e sem `assignee_id`, `isAgentAssigneeReady` retorna `false` no **primeiro** `if`,
logo **nao ha enfileiramento**. Tres barreiras independentes, e basta uma:
1. `assignee_type != "agent"` (ou ausente);
2. agente sem `runtime_id`;
3. agente arquivado.
E `status:"backlog"` e uma quarta barreira, anterior a todas.

### 2.2 ATENCAO - assignee **nao** e o unico gatilho
`shouldEnqueueOnComment` (`issue.go:2680-2685`) enfileira por **comentario** em issue que **ja tem**
agente atribuido, e o comentario dispara "para qualquer status", incluindo `done`. Ha ainda
`shouldEnqueueSquadLeaderOnAssign` e o caminho de `@mention`
(`computeMentionedAgentCommentTriggers`).
**Consequencia para o contrato:** "issue sem assignee" garante zero task **na criacao**. Nao garante
zero task depois, se alguem atribuir ou comentar. A regra operacional segura e: **nao atribuir e nao
comentar** ate a etapa 4.

**Exige prova runtime:** que a resposta de `POST /api/issues` traga `id` nessa chave, e a contagem
`agent_task_queue = 0` apos a criacao (verificacao da etapa 6, item V1).

## 3. LISTAR AGENTES E MODELOS

### 3.1 Agentes
Rota: **`GET /api/agents`** — `router.go:866` (`h.ListAgents`), grupo em `:865`.
```bash
curl -sS "${H[@]}" "$API/api/agents" | jq -r '.[] | "\(.id)  \(.name)  runtime=\(.runtime_id)  archived=\(.archived_at)"'
AGENT_ID=$(curl -sS "${H[@]}" "$API/api/agents" \
  | jq -r 'map(select(.runtime_id != null and .archived_at == null)) | .[0].id')
echo "AGENT_ID=$AGENT_ID   # so agentes com runtime e nao arquivados sao elegiveis (isAgentAssigneeReady)"
```
O filtro **nao e cosmetico**: e exatamente o predicado de `isAgentAssigneeReady`. Escolher um agente
sem `runtime_id` faria a atribuicao **nao** disparar task, e o teste da etapa 4 daria falso negativo.

### 3.2 Runtimes e modelos — fluxo assincrono de duas etapas
Rotas: **`GET /api/runtimes`** (`router.go:928`), **`POST /api/runtimes/{id}/models`**
(`router.go:937`, `h.InitiateListModels`) e **`GET /api/runtimes/{id}/models/{requestId}`**
(`router.go:938`, `h.GetModelListRequest`).
```bash
RUNTIME_ID=$(curl -sS "${H[@]}" "$API/api/runtimes" | jq -r 'map(select(.status=="online")) | .[0].id')
REQ_ID=$(curl -sS "${H[@]}" -X POST "$API/api/runtimes/$RUNTIME_ID/models" | jq -r '.request_id // .id')
sleep 3
curl -sS "${H[@]}" "$API/api/runtimes/$RUNTIME_ID/models/$REQ_ID" | jq '{status, error, count:(.models|length)}'
```
Listar modelo **nao** e leitura sincrona: o backend cria um pedido, o **daemon** responde por
`POST /api/daemon/runtimes/{runtimeId}/models/{requestId}/result` (`router.go:523`) e so entao o `GET`
tem resultado. Orcamento do daemon: **40 s** (`internal/daemon/daemon.go:2058-2067`).

**Exige prova runtime:** os nomes de campo `request_id` e `models`, e o tempo real ate o resultado.
Marcado como incerto — nao li `InitiateListModels`.

Nota de expectativa, medida por mim em outra auditoria: com `codex` e `claude` o catalogo e **estatico**
e popula sempre; com `kiro` e `antigravity` a descoberta chama o CLI e depende de login. **Nao afirmar
falha do endpoint** se esses dois vierem vazios.

## 4. UMA ATRIBUICAO CONTROLADA, COM GATILHO CONHECIDO

**Nao existe rota dedicada de "assign".** Confirmado: `grep -nE 'assign|Assignee' router.go` devolve
apenas `GET /api/assignee-frequency` (`:720`). A atribuicao e feita **atualizando a issue**, e o
gatilho e o campo `assigneeChanged`.

Preview obrigatorio **antes** de atribuir:
```bash
# P1 estado atual da issue
curl -sS "${H[@]}" "$API/api/issues/$ISSUE_ID" | jq '{id,status,assignee_type,assignee_id}'
# P2 elegibilidade do agente escolhido - reproduz isAgentAssigneeReady
curl -sS "${H[@]}" "$API/api/agents" | jq --arg a "$AGENT_ID" \
  'map(select(.id==$a))[0] | {id,name,runtime_id,archived_at,
     elegivel:((.runtime_id!=null) and (.archived_at==null))}'
# P3 fila ANTES (baseline obrigatorio)
#   ler agent_task_queue nos 4 estados ativos: queued, dispatched, running, waiting_local_directory
```
Atribuicao — **uma** chamada, e o efeito esperado e **exatamente uma** task:
```bash
curl -sS "${H[@]}" -X PUT "$API/api/issues/$ISSUE_ID" \
  -d "$(jq -nc --arg a "$AGENT_ID" '{assignee_type:"agent", assignee_id:$a}')" | jq '{id,assignee_type,assignee_id}'
```
Gatilho conhecido, de `issue.go:2531-2536`: `assigneeChanged` -> `CancelTasksForIssue` **primeiro**,
depois `EnqueueTaskForIssue` se o predicado passar. Ou seja o proprio backend cancela task anterior
antes de criar a nova — nao ha acumulo por reatribuicao.

**R-1 RESOLVIDO durante a redacao deste contrato.** O verbo real e **`PUT /api/issues/{id}`** ->
`h.UpdateIssue`, dentro do grupo `/{id}` (`router.go:733`); **nao existe `r.Patch`** para issue. E o
`PUT` **aceita corpo parcial**, porque todo campo de `UpdateIssueRequest` (`issue.go:2289-2306`) e
**ponteiro**: `Title *string`, `Status *string`, `AssigneeType *string`, `AssigneeID *string`, etc.
Campo ausente fica `nil` e nao e alterado. Portanto enviar somente os dois campos de assignee e
correto e **nao** apaga titulo, status nem projeto.

## 5. OBSERVAR STATUS, LOG E RESULTADO

```bash
# tasks do agente
curl -sS "${H[@]}" "$API/api/agents/$AGENT_ID/tasks" | jq -r '.[] | "\(.id) \(.status) \(.failure_reason) \(.created_at)"'
TASK_ID=$(curl -sS "${H[@]}" "$API/api/agents/$AGENT_ID/tasks" | jq -r 'sort_by(.created_at)|last|.id')
# mensagens/log da task
curl -sS "${H[@]}" "$API/api/tasks/$TASK_ID/messages" | jq -r '.[] | "\(.seq) \(.type) \(.tool)"'
```
Rotas, e as duas primeiras sao as **mais diretas** para este contrato porque partem da issue:
- **`GET /api/issues/{id}/active-task`** -> `h.GetActiveTaskForIssue` — responde "esta issue tem task
  ativa agora?", que e exatamente a pergunta de V1 e V3;
- **`GET /api/issues/{id}/task-runs`** -> `h.ListTasksByIssue` — historico de execucoes da issue;
- **`GET /api/agents/{id}/tasks`** (`router.go:879`, `h.ListAgentTasks`);
- **`GET /api/tasks/{taskId}/messages`** (`router.go:764`, `h.ListTaskMessagesByUser`).

`active-task` e `task-runs` permitem checar V1 e V3 **sem** acesso ao banco, o que e preferivel para
quem so tem token de API.

Os estados possiveis vem da constraint de `migrations/109:13-15`: **ativos**
`queued`, `dispatched`, `running`, `waiting_local_directory`; **terminais** `completed`, `failed`,
`cancelled`. Campos uteis de `agent_task_queue`: `status`, `failure_reason`, `error`, `result`,
`attempt`, `max_attempts`, `wait_reason`.

**Nao** usar as rotas `/api/daemon/tasks/...` (`router.go:527-543`): sao do **daemon**, exigem auth de
daemon e nao sao o caminho do operador.

## 6. VERIFICACOES OBRIGATORIAS

| # | verificacao | como | aceite |
|---|---|---|---|
| V1 | criar issue sem assignee **nao** enfileira | contar `agent_task_queue` nos 4 estados ativos antes e depois do passo 2 | **delta 0** |
| V2 | agente escolhido e elegivel | preview P2 | `elegivel: true` |
| V3 | a atribuicao cria **exatamente uma** task | contar antes e depois do passo 4 | **delta 1** |
| V4 | a task e da issue certa | `issue_id` da task == `ISSUE_ID` | igual |
| V5 | nenhuma task orfa de outra issue foi afetada | comparar a lista completa antes/depois | so a nova linha muda |

Predicado canonico da fila, derivado de `migrations/109:13-15`:
```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```

## 7. ROLLBACK / CANCEL SE O DISPARO FOR INDEVIDO

Tres rotas reais, em ordem de precisao:
```bash
# R1 cancelar UMA task especifica (preferido)
curl -sS "${H[@]}" -X POST "$API/api/tasks/$TASK_ID/cancel"            # router.go:971  CancelTaskByUser
# R2 cancelar as tasks de UMA issue
curl -sS "${H[@]}" -X POST "$API/api/issues/$ISSUE_ID/tasks/$TASK_ID/cancel"   # router.go:745  CancelTask
# R3 cancelar TODAS as tasks de um agente — martelo, usar so em contencao
curl -sS "${H[@]}" -X POST "$API/api/agents/$AGENT_ID/cancel-tasks"     # router.go:878  CancelAgentTasks
```
Reversao da causa, nao apenas do efeito — **remover o assignee** para nao re-disparar:
```bash
curl -sS "${H[@]}" -X PUT "$API/api/issues/$ISSUE_ID" -d '{"assignee_type":null,"assignee_id":null}'
```
Isso volta a acionar `assigneeChanged`, que chama `CancelTasksForIssue` e **nao** enfileira, porque o
predicado falha sem assignee. Confirmar depois que a fila voltou ao baseline de V1.

**Exige prova runtime:** que `assignee_type:null` seja aceito — o campo e `*string`, logo `null` e
representavel, mas nao confirmei se o handler distingue "ausente" de "null explicito". **Confirmar antes.**

Ordem de contencao recomendada: **R1** (uma task) -> remover assignee -> **R2** se sobrou -> **R3** so
se houver disparo em cascata. Nunca comecar por R3.

---

## 8. CHECKLIST CURTO PARA O KIRO

```
[ ] 0. exportar API, TOKEN, WS_SLUG. Token NUNCA em evidencia, log ou chat
[ ] 1. (RESOLVIDO no contrato) verbo e PUT /api/issues/{id}, corpo PARCIAL aceito (campos ponteiro)
[ ] 2. baseline da fila nos 4 estados ativos  -> anotar N0
[ ] 3. POST /api/projects            -> PROJECT_ID
[ ] 4. POST /api/issues SEM assignee -> ISSUE_ID          (status != backlog)
[ ] 5. fila de novo  -> exigir N == N0   (V1: delta 0)
[ ] 6. GET /api/agents -> escolher AGENT_ID com runtime_id != null e archived_at == null (V2)
[ ] 7. GET /api/runtimes ; POST /api/runtimes/{id}/models ; GET .../models/{requestId}
       -> nao tratar catalogo vazio de kiro/antigravity como falha de endpoint
[ ] 8. preview P1 + P2 + P3 antes de atribuir
[ ] 9. UMA atribuicao: PUT /api/issues/{ISSUE_ID} com {assignee_type:"agent", assignee_id:AGENT_ID}
[ ] 10. fila -> exigir N == N0 + 1 (V3) e issue_id da task == ISSUE_ID (V4, V5)
[ ] 11. observar: GET /api/issues/{ISSUE_ID}/active-task , GET /api/issues/{ISSUE_ID}/task-runs ,
        GET /api/agents/{AGENT_ID}/tasks , GET /api/tasks/{TASK_ID}/messages
[ ] 12. se indevido: R1 cancel da task -> remover assignee -> R2 -> R3 so em cascata
[ ] 13. anexar evidencia: contagens N0/N1/N2, IDs resolvidos, status final. SEM token, SEM segredo
```
Regra que atravessa o checklist: **atribuir enfileira task paga.** O passo 9 e o unico que gasta
orcamento, e por isso ele vem depois de dois gates de contagem e de um preview de elegibilidade.

## 9. O QUE EXIGE PROVA RUNTIME — LISTA CONSOLIDADA

| # | item | risco se nao confirmar |
|---|---|---|
| ~~R-1~~ | **RESOLVIDO sem runtime**: e `PUT /api/issues/{id}` (`h.UpdateIssue`), sem `r.Patch`, e o corpo e parcial porque `UpdateIssueRequest` usa ponteiros (`issue.go:2289-2306`) | — |
| R-2 | campos aceitos por `CreateProject` e chave do `id` na resposta | passo 3 falha |
| R-3 | chave do `id` na resposta de `CreateIssue` | passo 4 falha |
| R-4 | nomes `request_id` e `models` no fluxo de modelos, e latencia real | passo 7 confunde pendente com erro |
| R-5 | `assignee_type:null` aceito para desatribuir | rollback da causa nao funciona |
| R-6 | forma de resposta de `ListAgentTasks` e de `ListTaskMessagesByUser` | passo 11 nao parseia |
| R-7 | **V1 empiricamente**: criar sem assignee realmente nao enfileira | e a garantia central do contrato; a cadeia de codigo esta verificada, a **execucao** nao |
| R-8 | se `@mention` em descricao de issue pode disparar na **criacao** | uma issue "sem assignee" com @mention no corpo poderia enfileirar |

R-7 e R-8 sao os dois que eu recomendo provar **primeiro**, porque atacam a garantia de zero task.

## 10. NAO-AFIRMACOES
- **Nenhuma chamada de API foi feita. Nenhuma mutacao de board. Nada executado.** Nao criei projeto,
  issue, atribuicao, comentario ou task; nao cancelei nada.
- Nao li `CreateProject`, `InitiateListModels`, `ListAgentTasks`, `ListTaskMessagesByUser`,
  `GetActiveTaskForIssue` nem `ListTasksByIssue`: os payloads desses seis sao **inferidos do padrao** e
  estao marcados em R-2, R-4 e R-6.
- R-1 foi **fechado por leitura** ainda durante a redacao: `PUT` confirmado e `r.Patch` inexistente
  para issue. Corrigi o contrato de `PATCH` para `PUT` em dois lugares e no checklist.
- A garantia de zero task da secao 2.1 vem de **leitura de codigo** (`issue.go:2531-2536`,
  `:3032-3036`, `:2663-2668`, `:2740-2751`), nao de execucao. R-7.
- Nao mapeei os caminhos de squad leader nem de `@mention` alem de constatar que existem e que sao
  gatilhos **adicionais**. Ver 2.2 e R-8.
- Nao usei nem li token, PAT ou credencial. As variaveis `TOKEN` etc. sao placeholders do operador.
- Nao dupliquei bootstrap nem CFN, conforme instruido.
- Nao resolvi nenhum ID: todos os comandos resolvem em runtime, por design.
