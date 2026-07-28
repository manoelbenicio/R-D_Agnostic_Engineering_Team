# ORQ-41 - Desacoplamento de metadados do Kanban e execucao paga (design READ-ONLY)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T13:22Z
- card citado: **ORQ-41**, UUID **`666f1ead-7fe9-4051-bab9-5d0a936c4701`**, `number: 41`,
  workspace `20fce817-895d-447b-965a-49f5e279314a` (`orq2-dev`), status `todo`, **sem assignee**
  (verificado por `GET /api/issues?workspace_slug=orq2-dev`, read-only)
- modo: READ-ONLY. Sem mutacao de codigo, board, API, schema, build, teste, deploy ou credencial.
  **Nenhuma implementacao.** Revisor independente solicitado.
- containment confirmado por medicao: **ORQ-39** `c03941bc-3bde-4de1-ab19-1ba93de0ad51` e
  **ORQ-40** `d2001a24-af70-4367-8828-2225ed43ad84` estao ambos `blocked` e com
  `assignee_type: null`. **Replay proibido**; nao reexecutei nada.

## 0. DIVERGENCIA DE ESCOPO A RESOLVER ANTES (nao inventei)
O titulo real do ORQ-41 e *"Add safe non-triggering documentation comments to assig..."*, que e mais
estreito que o objetivo deste dispatch (contrato de desacoplamento). Cito o ORQ-41 porque foi o card
indicado, mas registro a divergencia: ou o titulo precisa ser ampliado, ou este design pertence a um
card diferente. **Nao alterei o card** (freeze de assignee/comentario e regra de registrar unico).

---

## 1. Superficie medida - todo caminho que hoje enfileira task paga

`EnqueueTaskForIssue` / `EnqueueTaskForSquadLeader`, call sites reais (sem testes):

| # | arquivo:linha | gatilho de negocio | guarda existente |
|---|---|---|---|
| T1 | `internal/handler/issue.go:2535` | **UpdateIssue: assignee mudou** | `shouldEnqueueAgentTask` (issue.go:2663) |
| T2 | `internal/handler/issue.go:2541` | UpdateIssue: assignee = squad | `shouldEnqueueSquadLeaderOnAssign` (squad.go:946) |
| T3 | `internal/handler/issue.go:2559` | UpdateIssue: `backlog -> outro status` | `isAgentAssigneeReady` (issue.go:2740) + `isAgentRunningOnIssue` |
| T4 | `internal/handler/issue.go:2562` | idem, squad | `isSquadLeaderReady` (squad.go:958) |
| T5 | `internal/handler/issue.go:3035` | **UpdateIssues (batch): assignee mudou** | `shouldEnqueueAgentTask` |
| T6 | `internal/handler/issue.go:3038` | batch, squad | `shouldEnqueueSquadLeaderOnAssign` |
| T7 | `internal/handler/issue.go:3050` | batch, saida de backlog | `isAgentAssigneeReady` |
| T8 | `internal/handler/issue.go:3053` | batch, squad | `isSquadLeaderReady` |
| T9 | `internal/handler/comment.go:1136` | **comentario em issue atribuida, SEM mencao** | `commentTriggerSourceIssueAssignee` |
| T10 | `internal/handler/comment.go:1127` | comentario, assignee = squad | idem, `trigger.Squad != nil` |
| T11 | `internal/handler/comment.go:1140` | comentario com **mencao a lider de squad** | `commentTriggerSourceMentionSquadLeader` |
| T12 | `internal/handler/comment.go:1146` | comentario com **mencao a agente** | `commentTriggerSourceMentionAgent` |
| T13 | `internal/service/issue.go:388` (`maybeEnqueueOnAssign`, :383) | **criacao de issue JA com assignee** | `shouldEnqueueAgentTask` |
| T14 | `internal/handler/onboarding_shim.go:329` | onboarding cria issue com assistente | `shouldEnqueueAgentTask` |
| T15 | `internal/service/autopilot.go:275` | Autopilot dispatch (`create_issue`) | acesso a leader privado |
| T16 | `internal/service/autopilot.go:271` | Autopilot, lider de squad | idem |
| T17 | `internal/service/task.go:1675` (`CreateRetryTask` via `MaybeRetryFailedTask`:1631) | **auto-retry apos falha** | `commitledger.CheckOrAllow` (task.go:1666) |

Webhooks: `router.go:498` `POST /api/webhooks/autopilots/{token}` e `:501`
`POST /api/webhooks/github` chegam ao mesmo `EnqueueTaskForIssue` por T15/T16 e pelo caminho de
criacao/comentario - **nao ha caminho de enqueue exclusivo de webhook**; eles herdam a semantica de
T13/T15.

**Diagnostico**: existem **17 gatilhos** e **nenhum deles e uma acao explicita de "executar"**. Os
dois que o incidente expos, T1/T5 (assign) e T9/T10 (comentario comum), sao operacoes que qualquer
usuario le como metadado. O unico ponto com gate real de seguranca e T17, e ele guarda replay, nao
custo.

**Nota de comentario "sem mencao"**: `comment.go:1174-1180` mostra que o assignee e adicionado como
trigger quando o comentario **nao** menciona outros e **nao** e resposta a thread de membro. Ou seja
o gatilho e o **default**, e a ausencia de mencao nao protege - confirma o fato medido do dispatch.

---

## 2. Requisito 1 - metadados por default, fail-closed

Contrato proposto:

> **Toda** operacao de metadado - `POST /api/issues`, `PATCH /api/issues/{id}`,
> `PATCH /api/issues` (batch) e `POST /api/issues/{id}/comments` - **nunca** enfileira task.
> Enfileirar exige uma acao distinta (secao 3). Campo ausente = **nao executar**.

Isso inverte o default atual em T1-T14. T15-T16 (Autopilot) e T17 (retry) recebem semantica propria
na secao 4.

---

## 3. Requisito 2 - a acao explicita, e a opcao menos destrutiva

### 3.1 Opcoes avaliadas
| opcao | forma | quebra clientes? | risco |
|---|---|---|---|
| **A** endpoint dedicado `POST /api/issues/{id}/runs` | acao nova, corpo com opt-in | **nao** quebra nada existente, mas todo cliente atual **perde** o disparo implicito | precisa UI/CLI novos para nao regredir fluxo do usuario |
| **B** campo opt-in `execute: {...}` em PATCH/POST/comment | reusa endpoints | compativel; omissao = nao executa | mistura metadado e execucao no mesmo contrato; auditoria menos limpa |
| **C** manter implicito com confirmacao no cliente | so UI | zero quebra de API | **rejeitada**: API continua disparando custo sem intencao; CLI/mobile/webhook seguem vulneraveis |

**RECOMENDO A + B como um par, com A canonico e B como ponte:**
- **A** e o contrato correto e auditavel: `POST /api/issues/{id}/runs` cria a execucao, e so ele.
- **B** existe **apenas** durante a migracao, como `?execute=true` / campo `execute`, para que
  clientes antigos possam ser adaptados sem big-bang. B nasce **deprecated** e com prazo.
- **C** nao entra: e a situacao atual maquiada.

### 3.2 Contrato de `POST /api/issues/{id}/runs`
```
POST /api/issues/{issue_id}/runs
Idempotency-Key: <uuid v4 do cliente>            # obrigatorio
{
  "agent_id":        "<uuid>",                    # explicito, nunca inferido do assignee
  "model":           "<route model id>",          # confirmado pelo cliente
  "reasoning_level": "<low|medium|high|...>",     # confirmado
  "account_ref":     "<referencia de conta, NAO credencial>",
  "cost_ack": { "estimated": true, "confirmed_by": "<user_id>" },
  "reason":          "<texto curto de auditoria>"
}
-> 201 { "task_id": "...", "issue_id": "...", "idempotency_key": "...", "state": "queued" }
-> 200 (mesmo corpo) quando a chave de idempotencia repete: NAO cria segunda task
-> 409 quando o guard de fila recusa (ja existe task ativa para (issue, agent))
-> 402/422 quando falta confirmacao de custo/modelo/conta
```
Guardas obrigatorias, todas fail-closed:
1. **autorizacao** - papel com permissao de executar, distinta de permissao de editar issue;
2. **idempotencia** - chave do cliente persistida (secao 5);
3. **confirmacao de modelo/reasoning/conta/custo** - ausente => `422`, nunca default silencioso;
4. **queue-state guard** - nao enfileira se ja existe task ativa para o par (issue, agent);
5. **campo omitido => nao executa**, em qualquer endpoint.

---

## 4. Requisito 3 - semantica explicita por caminho (sem bypass acidental)

| caminho | semantica proposta | observacao |
|---|---|---|
| **assign / reassign** (T1,T2,T5,T6) | **metadado puro**. Nunca enfileira. `CancelTasksForIssue` (issue.go:2532/3033) **permanece**, porque cancelar e contencao, nao custo | e a correcao central do incidente |
| **saida de backlog** (T3,T4,T7,T8) | **metadado puro**. O "parking lot" deixa de disparar | hoje e o gatilho mais surpreendente: mudar status de coluna gasta dinheiro |
| **comentario comum em issue atribuida** (T9,T10) | **metadado puro** | fato medido do dispatch |
| **mencao explicita a agente** (T12) | **executa**, e a mencao **e** a acao explicita - mas exige `cost_ack` do autor e passa pelas mesmas 5 guardas | preservar mencao como gatilho e a unica forma de nao destruir o fluxo conversacional; a mencao e intencional por natureza |
| **mencao a lider de squad** (T11) | idem T12 | |
| **criacao com assignee** (T13) | **metadado puro**; criar com assignee nao executa | UI oferece "Criar e executar" como duas chamadas: `POST /issues` depois `POST /runs` |
| **onboarding** (T14) | **executa**, com opt-in **explicito no fluxo de onboarding** e `cost_ack` | e o unico lugar onde o produto quer primeira execucao guiada; documentar como excecao nomeada |
| **Autopilot** (T15,T16) | **executa**, porque o Autopilot **e** um automatizador que o usuario ligou deliberadamente. Exigir: `cost_ack` gravado na definicao do autopilot, e o gate de replay (hoje ausente ali - ver `gtl-ledger-v2-corrected-design.md`) | risco declarado: autopilot hoje **nao** consulta o replay gate |
| **Retry automatico** (T17) | **executa**, mantendo `commitledger.CheckOrAllow` (task.go:1666) e somando o queue-state guard | nao mexer no gate existente |
| **webhooks** (`router.go:498`,`:501`) | herdam T13/T15: se criam issue, **nao** executam; se disparam autopilot, seguem a regra do Autopilot | nao ha caminho proprio, e isso deve ficar escrito |

Regra guarda-chuva: **qualquer novo call site de `EnqueueTaskFor*` fora de `POST /runs`, Autopilot e
Retry e defeito**, e deve ser barrado por teste (secao 8.12).

---

## 5. Requisitos 4 - exactly-once transacional e auditoria imutavel

### 5.1 O que existe hoje
`migrations/037_fix_pending_task_unique_index.up.sql`:
```sql
CREATE UNIQUE INDEX idx_one_pending_task_per_issue_agent
    ON agent_task_queue (issue_id, agent_id)
    WHERE status IN ('queued', 'dispatched');
```
**Lacuna medida**: o predicado cobre so `queued` e `dispatched`. Nao cobre `running` nem
`waiting_local_directory` (adicionado em `109_agent_task_waiting_local_directory.up.sql:15`). Logo,
com uma task **em execucao**, um segundo enqueue **passa** pelo indice. Isso e a raiz do duplo clique
sobreviver hoje.

### 5.2 Necessidades de schema (**migration NEXT_CANONICAL**, numero nao previsto)
```sql
-- NEXT_CANONICAL_a: idempotencia de execucao
CREATE TABLE task_execution_request (
    idempotency_key  UUID        PRIMARY KEY,          -- fornecida pelo cliente
    issue_id         UUID        NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
    agent_id         UUID        NOT NULL,
    task_id          UUID        NULL REFERENCES agent_task_queue(id) ON DELETE SET NULL,
    requested_by     UUID        NOT NULL,
    trigger_kind     TEXT        NOT NULL CHECK (trigger_kind IN
                       ('explicit_run','mention','onboarding','autopilot','retry')),
    model            TEXT        NOT NULL,
    reasoning_level  TEXT        NULL,
    account_ref      TEXT        NULL,                 -- referencia, NUNCA credencial
    cost_ack         BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- NEXT_CANONICAL_b: fechar a lacuna do guard de fila
DROP INDEX IF EXISTS idx_one_pending_task_per_issue_agent;
CREATE UNIQUE INDEX idx_one_active_task_per_issue_agent
    ON agent_task_queue (issue_id, agent_id)
    WHERE status IN ('queued','dispatched','running','waiting_local_directory');

-- NEXT_CANONICAL_c: auditoria imutavel
CREATE TABLE task_trigger_audit (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        UUID        NULL REFERENCES agent_task_queue(id) ON DELETE SET NULL,
    issue_id       UUID        NOT NULL,
    actor_type     TEXT        NOT NULL,               -- member | agent | system
    actor_id       UUID        NOT NULL,
    trigger_kind   TEXT        NOT NULL,
    idempotency_key UUID       NULL,
    decision       TEXT        NOT NULL CHECK (decision IN ('enqueued','denied_guard','denied_auth','deduped')),
    reason         TEXT        NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- imutabilidade: sem UPDATE/DELETE concedidos ao role da aplicacao; append-only por convencao
-- de permissao, nao por trigger (trigger seria contornavel pelo mesmo role)
```
**RESSALVA sobre `NEXT_CANONICAL_b`**: ampliar o indice unico e **breaking** para qualquer fluxo que
hoje dependa de enfileirar durante `running`. Nao verifiquei se algum existe. Precisa levantamento
antes, e por isso proponho aplica-lo **depois** de a e c, atras da flag.

### 5.3 Transacionalidade
`POST /runs` executa numa unica transacao: insert em `task_execution_request` (PK garante
exactly-once sob clique concorrente) -> insert em `agent_task_queue` -> insert em
`task_trigger_audit`. Conflito na PK => `200` com o `task_id` ja existente, **sem** segunda task.
Conflito no indice unico de fila => `409` + linha de auditoria `denied_guard`. `TxStarter` ja existe
(`service/task.go:127` recebe `tx`), e `runInTx` (`task.go:1918`) e o utilitario a reusar.

---

## 6. Requisito 5 - UI

`packages/views/issues/components/` (medido: `board-card.tsx`, `board-column.tsx`, `board-view.tsx`,
`issue-actions-*.tsx`). Separacao proposta:
- **Assign** - seletor de assignee, sem qualquer efeito de execucao, e sem spinner de "iniciando".
- **Comment** - botao de comentar puro. Se o texto contiver mencao a agente, a UI mostra **antes** do
  envio um aviso "isto vai executar o agente X, modelo Y, custo estimado" com confirmacao explicita.
- **Run now** - botao distinto, com dialogo obrigatorio: agente, modelo, reasoning, conta e
  confirmacao de custo. E o unico controle que chama `POST /runs`.
Estado desabilitado quando ja existe task ativa, refletindo o queue-state guard, para o usuario nao
descobrir o guard por um `409`.
`FILES_LOCKED` sugerido para a UI: `board-card.tsx`, `issue-actions-dropdown.tsx`,
`issue-actions-context-menu.tsx`, `use-issue-actions.ts` - um dono unico.

---

## 7. Requisito 6 - compatibilidade e migracao por flag

Flag: `MULTICA_EXECUTION_TRIGGER_DECOUPLED` (nome sugerido; ha precedente de flags de env em
`config.go`).

| fase | flag | comportamento | quem adapta |
|---|---|---|---|
| 0 | off | igual a hoje, **mais** auditoria (`NEXT_CANONICAL_c`) e `POST /runs` ja disponivel | ninguem quebra; ja da visibilidade de quem dispara |
| 1 | `warn` | metadado ainda enfileira, mas resposta inclui `deprecation` e log/auditoria marca `implicit_trigger` | web, CLI e mobile migram para `/runs` |
| 2 | on | metadado **nao** enfileira; `execute=true` (opcao B) ainda aceito, marcado deprecated | clientes antigos sobrevivem |
| 3 | on + B removido | apenas `/runs`, mencao, onboarding, autopilot e retry executam | fim |

Compatibilidade dura: **campo omitido => nao executa**, em todas as fases >= 2. CLI (`cmd/multica/`)
e clientes moveis que nao conhecerem `/runs` perdem o disparo implicito - por isso a fase 1 existe, e
por isso a metrica de saida da fase 1 e "zero `implicit_trigger` na auditoria por N dias".

---

## 8. Requisito 8 - testes exigidos (nenhum executado aqui)

1. `TestCreateIssueWithAssignee_NoEnqueue` - T13 vira metadado.
2. `TestReassign_NoEnqueue_ButCancelsActive` - T1/T5 nao enfileiram e `CancelTasksForIssue` continua.
3. `TestBacklogPromotion_NoEnqueue` - T3/T4/T7/T8.
4. `TestOrdinaryComment_NoEnqueue` - T9/T10, incluindo o caso de `comment.go:1174-1180`.
5. `TestMentionAgent_Enqueues_WithCostAck` e `TestMentionAgent_MissingCostAck_422` - T12.
6. `TestMentionSquadLeader_Enqueues` - T11.
7. `TestExplicitRun_201_AndAuditRow` - `/runs` felizes + linha imutavel de auditoria.
8. `TestExplicitRun_DuplicateIdempotencyKey_200_SingleTask` - exactly-once.
9. `TestExplicitRun_ConcurrentClicks_OneTask` - N goroutines, mesma chave: 1 task, N-1 `deduped`.
10. `TestExplicitRun_ActiveTaskExists_409` - queue-state guard, inclusive com task em `running`
    (falha hoje, por 5.1).
11. `TestRetry_StillGatedByCommitLedger` - T17 intacto.
12. `TestAutopilot_Enqueues_WithRecordedAck` - T15/T16, e o gap de replay gate documentado.
13. `TestFailureRollback_NoOrphanTask` - erro depois do insert de fila desfaz tudo na transacao.
14. `TestMetadataOps_ZeroCost` - bateria de assign/comment/status que assere **zero** linhas novas em
    `agent_task_queue`.
15. `TestNoNewEnqueueCallSites` - teste estrutural (grep/AST) falhando se `EnqueueTaskFor*` aparecer
    fora dos call sites permitidos. E o que impede a regressao voltar.

---

## 9. Gates de PASS-readiness e rollback

- **P1** este design revisado por revisor independente, com o mapa de 17 gatilhos conferido.
- **P2** levantamento de dependencia do `NEXT_CANONICAL_b` (existe fluxo que enfileira durante
  `running`?). Se existir, redesenhar antes.
- **P3** decisao do owner sobre as tres excecoes que **continuam** executando: mencao, onboarding,
  autopilot. Se o owner quiser zero execucao implicita, mencao e onboarding tambem viram `/runs`.
- **P4** resolver a divergencia de titulo do ORQ-41 (secao 0).
- **P5** somente depois: implementacao por fases da secao 7, cada fase com peer review.

Rollback por fase: a flag volta para o valor anterior, **sem** reverter migration - as tabelas de
`NEXT_CANONICAL_a` e `_c` sao aditivas e inertes com a flag off. O unico item nao trivialmente
reversivel e `NEXT_CANONICAL_b` (indice), cujo rollback e recriar o indice antigo; por isso ele fica
por ultimo e sozinho.

## 10. Perguntas em aberto (declaradas, nao afirmadas)

- se algum fluxo legitimo enfileira durante `running` (bloqueia `NEXT_CANONICAL_b`): nao levantei.
- se o CLI/mobile tem caminho proprio de enqueue alem dos 17: varri `EnqueueTaskFor*`, mas nao varri
  chamadas indiretas por outro nome.
- custo estimado por execucao: nao existe estimador hoje; `cost_ack` como desenhei confirma
  **intencao**, nao valor. Estimativa real depende de preco por tier, que e o gap da ORQ-13.
- permissao dedicada de "executar": nao verifiquei se o modelo de papeis atual permite separa-la de
  "editar issue".

## 11. Nada mutado

Nenhum codigo, board, API, schema, build, teste, deploy ou credencial tocado. Somente leitura de
fonte e `GET` na API. Nao alterei assignee nem postei comentario (freeze vigente). Nao sou o
registrar: nenhum card criado. **Nao reexecutei ORQ-39, ORQ-40 nem qualquer issue preservada.**

**Revisor independente solicitado antes de qualquer implementacao.**

---
---

# SECAO V2 (GTL-D41-V2) - CORRECAO QUE SUPERSEDE A V1 - ORQ-41

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T13:32Z
- card: **ORQ-41**, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`, number 41 (par verificado, read-only)
- fecha: BLOCK do GTL-R41
- **SUPERSEDE as secoes 1, 4 e 8 da V1.** O restante da V1 permanece, com as emendas abaixo.
- modo: READ-ONLY. Sem mutacao de codigo, board, API, schema, build, teste, deploy ou credencial.
  **Nenhuma implementacao.** Re-review pelo **mesmo revisor** solicitada.

## V2.1 INVENTARIO CANONICO CORRIGIDO - 21 call sites, nao 17

A V1 tinha **4 lacunas**. Enumeracao completa por `grep -rn "EnqueueTaskFor[A-Za-z]*("` sem testes e
sem a definicao dos metodos:

| # | arquivo:linha | metodo | gatilho | novo na V2? |
|---|---|---|---|---|
| C1 | `handler/issue.go:2535` | `EnqueueTaskForIssue` | update: assignee mudou | |
| C2 | `handler/issue.go:2559` | `EnqueueTaskForIssue` | update: saida de backlog | |
| C3 | `handler/issue.go:3035` | `EnqueueTaskForIssue` | batch: assignee mudou | |
| C4 | `handler/issue.go:3050` | `EnqueueTaskForIssue` | batch: saida de backlog | |
| C5 | `handler/squad.go:1007` | `EnqueueTaskForSquadLeader` | **helper `enqueueSquadLeaderTask`**, alvo real de issue.go 2541/2562/3038/3053 | **SIM** - a V1 citava os `if`, nao o call site |
| C6 | `handler/comment.go:1127` | `EnqueueTaskForSquadLeader` | comentario, assignee = squad | |
| C7 | `handler/comment.go:1136` | `EnqueueTaskForIssue` | comentario comum em issue atribuida | |
| C8 | `handler/comment.go:1140` | `EnqueueTaskForSquadLeader` | mencao a lider | |
| C9 | `handler/comment.go:1147` | `EnqueueTaskForMention` | mencao a agente (V1 dizia 1146) | |
| **C10** | **`handler/issue_child_done.go:302`** | **`EnqueueTaskForMention`** | **conclusao de issue filha -> task no assignee do PAI** | **SIM - LACUNA** |
| **C11** | **`handler/issue_child_done.go:357`** | **`EnqueueTaskForSquadLeader`** | **conclusao de filha -> task no lider do squad do PAI** | **SIM - LACUNA** |
| C12 | `handler/onboarding_shim.go:329` | `EnqueueTaskForIssue` | onboarding | |
| C13 | `service/issue.go:388` | `EnqueueTaskForIssue` | criacao com assignee agente | |
| **C14** | **`service/issue.go:466`** | **`EnqueueTaskForSquadLeader`** | **criacao com assignee squad** | **SIM - LACUNA** |
| C15 | `service/autopilot.go:271` | `EnqueueTaskForSquadLeader` | autopilot, squad | |
| C16 | `service/autopilot.go:275` | `EnqueueTaskForIssue` | autopilot, agente | |
| C17 | `service/task.go:1675` | `CreateRetryTask` | auto-retry | |

Sao **17 call sites diretos** (C1-C17) mais os **4 caminhos indiretos** de entrada do child-done,
que sao gatilhos de negocio distintos e precisam de nome proprio no inventario:

| # | entrada indireta | arquivo:linha | observacao |
|---|---|---|---|
| I1 | `notifyParentOfChildDone` chamado no **update single** | `handler/issue.go:2579` | filha concluida por PATCH |
| I2 | idem no **batch** | `handler/issue.go:3065` | filha concluida em lote |
| I3 | idem por **webhook GitHub** | `handler/github.go:1331` | `actorType="system"`, **sem usuario humano na cadeia** |
| I4 | `dispatchParentAssigneeTrigger` -> `triggerChildDoneAgent` (`:266`) / `triggerChildDoneSquad` (`:268`) | `handler/issue_child_done.go:259,266,268` | fan-out interno para C10/C11 |

**Por que isto e grave e por que a V1 falhou**: C10/C11 disparam trabalho pago no **PAI** por causa
de uma mudanca de status na **FILHA**. Nenhum usuario tocou o pai. E por I3 isso pode nascer de um
webhook do GitHub, com `actorType="system"` - ou seja **sem ator humano** para confirmar custo. A
guarda existente e apenas `HasPendingTaskForIssueAndAgent` (`:294`, `:349`), que e o mesmo predicado
fraco do indice de `037`: cobre pendente, nao `running`.

Correcao de contrato: **conclusao de filha NAO gera trabalho pago por default.** Passa a exigir uma
**autorizacao de workflow persistida** no pai (secao V2.2), mais `cost_ack` e `Idempotency-Key`.

## V2.2 Autorizacao de workflow persistida para child-completion

Sem inventar `user_session`-style: reusar a tabela de idempotencia da V1 e adicionar **um** registro
de politica por issue-pai.

```sql
-- NEXT_CANONICAL_d: autorizacao explicita de encadeamento pai/filha
CREATE TABLE issue_workflow_authorization (
    issue_id        UUID        PRIMARY KEY REFERENCES issue(id) ON DELETE CASCADE,
    on_child_done   BOOLEAN     NOT NULL DEFAULT FALSE,   -- fail-closed
    max_chained     INT         NOT NULL DEFAULT 0,       -- 0 = ilimitado apenas se on_child_done
    cost_ack        BOOLEAN     NOT NULL DEFAULT FALSE,
    authorized_by   UUID        NOT NULL,
    authorized_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
Regra: C10/C11 so enfileiram se existir linha do **pai** com `on_child_done = TRUE` **e**
`cost_ack = TRUE`. Ausencia de linha => nao executa (fail-closed). A `Idempotency-Key` do
encadeamento e derivada deterministicamente de `(parent_id, child_id, child_done_at)` para que
reentrega de webhook (I3) ou duplo PATCH nao gere duas tasks - e essa e a chave que vai para
`task_execution_request` com `trigger_kind = 'child_done'`.

`trigger_kind` da V1 passa a: `explicit_run | onboarding | autopilot | retry | child_done`.
**`mention` sai da lista** - ver V2.3.

## V2.3 CORRECAO - mencao de texto **nao** e mais execucao implicita

A V1 recomendava preservar mencao como acao explicita. **Retirado.** Motivo aceito: mencao e texto
livre num campo de comentario; nao carrega `cost_ack`, nao carrega `Idempotency-Key`, nao tem
confirmacao de modelo/conta, e e trivialmente produzida por copiar-e-colar, por agente, por template
ou por reentrega de webhook. Tratar texto como autorizacao de gasto e exatamente a classe de defeito
do ORQ-41.

Contrato V2: **comentario e mencao sao metadados, sempre.** C6, C7, C8 e C9 tornam-se metadata-only.
Quem quiser executar apos comentar usa a acao explicita (`POST /runs`) ou o opt-in com `cost_ack` +
`Idempotency-Key` no mesmo request. A UI pode oferecer, no compositor de comentario, um checkbox
"executar o agente mencionado" que **anexa** os campos de execucao ao request - o gatilho e o campo,
nunca o texto.

Consequencia de produto a declarar ao owner: o fluxo conversacional "menciono e o agente responde"
**deixa de existir por default**. Isso e uma perda de conveniencia deliberada, trocada por nao gastar
dinheiro sem intencao. **Ruling do owner necessario** (V2.7).

## V2.4 CORRECAO - reatribuicao e metadado e **nao** cancela por default

A V1 mantinha `CancelTasksForIssue` (`issue.go:2532` e `:3033`) como "contencao, nao custo". Retirado:
cancelar e **destrutivo** - mata trabalho em andamento, descarta o que a task ja produziu e interage
com o commit-ledger, que registra `tool_use` possivelmente ja commitado. Um cancelamento implicito
por trocar assignee e a face espelhada do enqueue implicito.

Contrato V2:
- reatribuir **nao cancela nada**;
- cancelar exige `cancel_active_task=true` **explicito** no request, com:
  1. **RBAC** proprio de cancelar (distinto de editar e de executar);
  2. **confirmacao** obrigatoria na UI, nomeando a task e o agente afetados;
  3. **linha de auditoria** em `task_trigger_audit` com `decision='cancelled'`;
  4. **idempotencia** pela mesma `Idempotency-Key`, para duplo clique nao cancelar duas vezes;
  5. **rollback em falha**: cancelamento e mudanca de assignee na **mesma transacao** - se o cancel
     falhar, o reassign nao persiste, e vice-versa.
- `decision` de `task_trigger_audit` passa a incluir `'cancelled'`.

Efeito colateral positivo: hoje trocar assignee cancela e reenfileira (C1 + cancel). Com V2, trocar
assignee e barato e silencioso, e o operador decide separadamente o que fazer com a task viva.

## V2.5 CORRECAO - Autopilot passa pelo replay gate duravel, fail-closed

A V1 apenas registrava que o Autopilot nao consulta o gate. V2 exige que consulte.

Estado medido: `service/autopilot.go:271` e `:275` enfileiram sem nenhuma chamada a
`commitledger.CheckOrAllow`; o unico consumidor do gate e `service/task.go:1666`, no retry.

Contrato V2, antes de C15/C16:
```
correlationID := <task anterior da MESMA issue+agent, a mais recente terminal>
if err := commitledger.CheckOrAllow(<checker duravel>, correlationID); err != nil { NAO enfileira }
```
Definicao de correlacao anterior, para nao ficar ambigua: a task **mais recente em estado terminal**
(`completed`/`failed`/`cancelled`) para o par `(issue_id, agent_id)`. Se **nao existir** task
anterior, nao ha efeito colateral previo e o gate **permite** - esse e o unico caso de permissao por
ausencia, e ele e seguro porque significa "primeira execucao".

Fail-closed exigido nos tres casos: estado **ausente** para uma task anterior que existe, estado
**ambiguo** (`ever_ambiguous`) e estado **expirado** (fora da janela de 24h) => **bloqueia**. Isso
depende do checker duravel desenhado em `gtl-ledger-v2-corrected-design.md`; enquanto ele nao
existir, o Autopilot deve usar o registry em memoria e **falhar fechado** quando o registry nao
conhecer a correlacao - o que na pratica desliga o autopilot apos restart, e isso precisa ser dito ao
owner em vez de descoberto em producao.

Sem bypass: o teste estrutural de V2.8 passa a exigir que **todo** call site de `EnqueueTaskFor*`
fora de `POST /runs` esteja precedido, no mesmo bloco, por uma chamada ao gate.

## V2.6 Preservado da V1, sem alteracao

`POST /api/issues/{id}/runs` canonico; `Idempotency-Key` **UUIDv4** obrigatoria; **RBAC de executar
separado** de editar (e agora tambem de cancelar); transacao unica
`task_execution_request` + `agent_task_queue` + `task_trigger_audit`; auditoria **imutavel por
permissao de role**, nao por trigger; `account_ref` como **referencia**, nunca credencial; e as fases
de feature flag.

Emenda ao guard de fila (`NEXT_CANONICAL_b`), para nao quebrar paralelismo de squad:
```sql
CREATE UNIQUE INDEX idx_one_active_task_per_issue_agent
    ON agent_task_queue (issue_id, agent_id)
    WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```
A chave e `(issue_id, agent_id)`, **nao** `(issue_id)`. Portanto **agentes diferentes do mesmo squad
continuam paralelos na mesma issue** - o indice so impede o mesmo agente duas vezes. Isso preserva a
intencao original de `037_fix_pending_task_unique_index.up.sql`, cujo proprio comentario diz que o
indice antigo por issue "caused different agents' pending tasks to block each other". A unica mudanca
e ampliar o predicado de status de 2 para 4 estados.

## V2.7 Rulings de produto que o owner precisa dar

1. **Mencao deixa de executar** (V2.3): confirma a perda do fluxo conversacional por default?
2. **Child-done deixa de encadear** (V2.2): confirma que sub-tarefas seriais passam a exigir
   `on_child_done=TRUE` por issue-pai? Isso muda o comportamento documentado de cadeia serial citado
   em `issue.go:2545-2554`.
3. **Reassign deixa de cancelar** (V2.4): confirma que trocar assignee deixa a task viva rodando?
4. **Onboarding continua executando?** E a unica excecao que sobrou junto de autopilot e retry.
5. **Autopilot fail-closed apos restart** (V2.5): aceita que o autopilot pare enquanto o ledger
   duravel nao existir?

Sem esses cinco rulings, o design nao vira implementacao.

## V2.8 Testes - substitui a secao 8 da V1

Metadata zero-custo (assercao: **zero** linhas novas em `agent_task_queue` e zero em
`task_execution_request`):
1. `TestAssign_ZeroCost` - atribuir agente.
2. `TestReassign_ZeroCost_AndDoesNotCancel` - reatribuir nao enfileira **e nao cancela**.
3. `TestBacklogPromotion_ZeroCost` - `backlog -> todo`.
4. `TestBatchUpdate_ZeroCost` - batch com assignee e status.
5. `TestCreateWithAgentAssignee_ZeroCost` - C13.
6. `TestCreateWithSquadAssignee_ZeroCost` - **C14**, lacuna da V1.
7. `TestOrdinaryComment_ZeroCost` - C7.
8. `TestMentionAgent_ZeroCost` - **C9, invertido pela V2.3**.
9. `TestMentionSquadLeader_ZeroCost` - C8.

Regressoes de child-done:
10. `TestChildDone_NoWorkflowAuth_ZeroCost` - **C10/C11 sem `on_child_done`**: nada enfileirado.
11. `TestChildDone_AuthWithoutCostAck_ZeroCost` - `on_child_done=TRUE`, `cost_ack=FALSE` => nada.
12. `TestChildDone_Authorized_EnqueuesOnce` - autorizado enfileira **uma** task.
13. `TestChildDone_WebhookRedelivery_Idempotent` - **I3** duas vezes com o mesmo
    `(parent, child, child_done_at)` => 1 task, 1 `deduped`.
14. `TestChildDone_BatchAndSinglePathsIdentical` - I1 e I2 com o mesmo veredito.

Regressoes de autopilot:
15. `TestAutopilot_CallsReplayGate` - falha se o gate nao for consultado.
16. `TestAutopilot_GateMissingState_FailsClosed`.
17. `TestAutopilot_GateAmbiguous_FailsClosed`.
18. `TestAutopilot_GateExpired_FailsClosed`.
19. `TestAutopilot_NoPriorTask_Allowed` - unico caso de permissao por ausencia.

Execucao explicita e concorrencia:
20. `TestExplicitRun_201_AndImmutableAudit`.
21. `TestExplicitRun_DuplicateKey_200_SingleTask`.
22. `TestExplicitRun_ConcurrentClicks_OneTask`.
23. `TestExplicitRun_ActiveRunningTask_409` - falha hoje (predicado de 2 estados).
24. `TestSquadParallelism_DifferentAgents_SameIssue_BothAllowed` - **prova que o indice ampliado nao
    quebrou o paralelismo de squad**.
25. `TestExplicitRun_MissingCostAck_422`, `TestExplicitRun_NonUUIDv4Key_422`,
    `TestExplicitRun_WrongRBAC_403`.

Cancelamento:
26. `TestCancel_RequiresExplicitFlag`, `TestCancel_RequiresRBAC`,
    `TestCancel_Idempotent_DoubleClickCancelsOnce`,
    `TestCancel_FailureRollsBackReassign` - transacao unica de V2.4.

Estrutural:
27. `TestNoUnauthorizedEnqueueCallSites` - lista branca literal de C1-C17 mais os indiretos I1-I4;
    qualquer novo call site de `EnqueueTaskFor*` ou `CreateRetryTask` **falha o teste**, e todo call
    site fora de `/runs` precisa de chamada ao gate no mesmo bloco.

## V2.9 Gates de PASS e rollback (emenda a secao 9 da V1)

- **P1** re-review pelo **mesmo revisor** que emitiu o GTL-R41 BLOCK, conferindo os 21 itens de V2.1.
- **P2** levantar se algum fluxo legitimo enfileira durante `running` (bloqueia `NEXT_CANONICAL_b`).
- **P3** os **cinco** rulings de V2.7.
- **P4** divergencia de titulo do ORQ-41 (secao 0 da V1) ainda aberta.
- **P5** dependencia declarada: V2.5 depende do checker duravel de
  `gtl-ledger-v2-corrected-design.md`, que por sua vez depende de popular
  `CommitLedgerHMACSecret` - hoje **nunca atribuido**, o que faz todo ledger nascer fail-closed.
  Encadeamento: sem o secret, o Autopilot com gate **para**. Isso e feature, nao bug, mas tem de ser
  escolhido.
- **P6** implementacao por fases da secao 7 da V1, review por fase.

Rollback: flag volta ao valor anterior; migrations `_a`, `_c` e `_d` sao **aditivas e inertes** com a
flag off; `_b` (indice) e o unico com rollback nao trivial, recriar o antigo - vem por ultimo e
sozinho.

## V2.10 Nada mutado

Nenhum codigo, board, API, schema, build, teste, deploy ou credencial tocado. Nenhuma implementacao.
Nao alterei assignee nem postei comentario (freeze vigente). Nao sou registrar: nenhum card criado.
Nao reexecutei ORQ-39, ORQ-40 nem qualquer issue preservada. Somente leitura de fonte e `GET` na API.

**Re-review pelo mesmo revisor solicitada. ORQ-41 citado com par UUID+numero verificado.**
