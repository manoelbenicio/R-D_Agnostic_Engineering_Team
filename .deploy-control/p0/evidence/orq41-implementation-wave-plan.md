# ORQ-41 - Plano de ondas de implementacao com propriedade exclusiva (READ-ONLY)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T13:44Z
- card: **ORQ-41**, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`, number 41 (par verificado, read-only)
- base: `orq41-execution-trigger-decoupling-design.md`, secao **V2 (PASS)** - 21 itens de inventario
- modo: READ-ONLY, **plano**. Nenhuma implementacao, build, teste, deploy, mutacao de board, API,
  schema ou credencial. Nao alterei assignee nem postei comentario (freeze vigente).

---

## 1. Invariantes de propriedade (regra de nao colisao)

1. **Uma onda, um dono, um conjunto de arquivos disjunto.** Nenhum arquivo aparece em duas ondas.
2. **`migrations/` + `pkg/db/queries/` + `pkg/db/generated/` sao UMA lane exclusiva (W1).** Nenhuma
   outra onda toca esses tres diretorios, por dois motivos medidos: `sqlc.yaml` usa
   `schema: "migrations/"` e `queries: "pkg/db/queries/"` com saida em `pkg/db/generated`, logo a
   ordem migration -> query -> `sqlc generate` e obrigatoria; e codigo gerado em dois worktrees
   produz conflito puro, sem valor de review.
3. **`pkg/db/generated/*` nunca e editado a mao.** E saida de `sqlc generate`, executado apenas pelo
   dono de W1.
4. Toda onda entrega **atras da flag** `MULTICA_EXECUTION_TRIGGER_DECOUPLED`, cujo **default e `off`**
   ate W6.
5. Nenhuma onda pode adicionar call site de `EnqueueTaskFor*` ou `CreateRetryTask`. O teste
   estrutural de W5 e o guardiao.

---

## 2. Ondas, donos e FILES_LOCKED

### W1 - Schema, queries e codigo gerado (LANE EXCLUSIVA)
**Depende de**: nada. **Bloqueia**: W2, W3, W5.
**FILES_LOCKED**
```
server/migrations/NEXT_CANONICAL_a_task_execution_request.{up,down}.sql
server/migrations/NEXT_CANONICAL_c_task_trigger_audit.{up,down}.sql
server/migrations/NEXT_CANONICAL_d_issue_workflow_authorization.{up,down}.sql
server/pkg/db/queries/task_execution_request.sql
server/pkg/db/queries/task_trigger_audit.sql
server/pkg/db/queries/issue_workflow_authorization.sql
server/pkg/db/generated/**            (SAIDA de sqlc generate, nunca editada a mao)
```
**Fora de W1, deliberadamente**: `NEXT_CANONICAL_b` (ampliar o indice unico para 4 estados) - vai
sozinho em **W7**, porque e o unico item **breaking** e o unico com rollback nao trivial.
**Gate de saida**: `sqlc generate` limpo; `go build ./...` verde; migration `up` e `down` aplicadas e
revertidas em banco descartavel; **zero** mudanca fora dos caminhos acima.

### W2 - Refactor de gatilho no backend (o coracao)
**Depende de**: W1. **Bloqueia**: W3 (contrato), W5 (asserts).
**FILES_LOCKED**
```
server/internal/handler/issue.go              # C1 2535, C2 2559, C3 3035, C4 3050 -> metadata-only
server/internal/handler/comment.go            # C6 1127, C7 1136, C8 1140, C9 1147 -> metadata-only
server/internal/handler/issue_child_done.go   # C10 302, C11 357 + fan-out 259/266/268
server/internal/handler/squad.go              # C5 1007 (call site real do helper)
server/internal/handler/onboarding_shim.go    # C12 329
server/internal/service/issue.go              # C13 388, C14 466
server/internal/service/autopilot.go          # C15 271, C16 275 + gate de replay
server/internal/service/task.go               # runInTx/TxStarter, enqueue transacional, audit
server/internal/service/execution_trigger.go  # NOVO: unico ponto autorizado de enqueue
```
Nao toca `commitledger/` (ver W4) nem `router.go` (W3).
**Gate de saida**: `go build`/`go vet` verdes; com a flag `off` o comportamento e **byte-identico** ao
de hoje; com `warn` a auditoria registra `implicit_trigger` sem mudar comportamento.

### W3 - API, RBAC e contrato de cliente
**Depende de**: W1 (tabelas), W2 (funcao de enqueue). **Bloqueia**: W5.
**FILES_LOCKED**
```
server/cmd/server/router.go                   # rota POST /api/issues/{id}/runs
server/internal/handler/issue_run.go          # NOVO: handler de /runs, idempotencia, 201/200/409/422
server/internal/handler/cancel_task.go        # NOVO: cancel_active_task explicito + RBAC + audit
server/internal/middleware/rbac_execute.go    # NOVO: permissao de executar e de cancelar
```
**Gate de saida**: contrato conferido contra a secao V2.6; `Idempotency-Key` UUIDv4 obrigatoria;
campo omitido => nao executa; **nenhuma** rota existente com semantica alterada enquanto flag `off`.

### W4 - Dependencia de ledger/HMAC (lane separada, pode correr em paralelo a W2/W3)
**Depende de**: nada tecnicamente; **bloqueia o gate do Autopilot** de W2.
**FILES_LOCKED**
```
server/internal/daemon/config.go                    # popular CommitLedgerHMACSecret (hoje NUNCA atribuido)
server/internal/daemon/commitledger/replay_gate.go  # interface ReplayGateChecker (ver gtl-ledger-v2)
server/internal/service/replay_gate_db.go           # NOVO: checker duravel
```
**Fato que amarra as duas frentes**: `config.go:115` declara `CommitLedgerHMACSecret` e
`daemon.go:4145` le, mas **nao existe atribuicao em lugar algum** - logo todo ledger nasce
`NewFailClosed` com `EverSaturated=true`. Consequencia para W2: se o gate do Autopilot entrar antes de
W4, o Autopilot **para** (fail-closed correto, efeito indesejado).
**Gate de saida**: secret populado e verificado; checker duravel com fail-closed em ausente, ambiguo e
expirado; **decisao escrita do owner** sobre o Autopilot parar no intervalo.

### W5 - Testes adversariais (dono independente, NAO o autor de W2/W3)
**Depende de**: W1, W2, W3. **Bloqueia**: W6.
**FILES_LOCKED**
```
server/internal/handler/issue_metadata_zero_cost_test.go     # 9 testes de custo zero
server/internal/handler/issue_child_done_trigger_test.go     # 5 regressoes de child-done
server/internal/service/autopilot_gate_test.go               # 5 de gate, fail-closed
server/internal/handler/issue_run_test.go                    # /runs, idempotencia, 409, RBAC
server/internal/handler/cancel_task_test.go                  # 4 de cancelamento + rollback
server/internal/service/enqueue_callsite_guard_test.go       # estrutural, lista branca C1-C17 + I1-I4
```
Regra de independencia: quem escreveu W2/W3 **nao** escreve W5. Os 27 testes da secao V2.8 sao o
contrato de aceite, incluindo os dois que **devem falhar antes** da correcao
(`TestExplicitRun_ActiveRunningTask_409` e `TestChildDone_NoWorkflowAuth_ZeroCost`).
**Gate de saida**: `go test ./... -count=1` verde com flag `on`; e os dois testes acima **vermelhos**
quando a flag esta `off`, provando que medem a mudanca real.

### W6 - Frontend
**Depende de**: W3 (contrato estavel). **Nao bloqueia** ninguem.
**FILES_LOCKED**
```
packages/views/issues/components/board-card.tsx
packages/views/issues/actions/issue-actions-dropdown.tsx
packages/views/issues/actions/issue-actions-context-menu.tsx
packages/views/issues/actions/use-issue-actions.ts
packages/views/issues/components/run-now-dialog.tsx          # NOVO
```
Separacao visual obrigatoria: **Assign**, **Comment** e **Run now** distintos; dialogo de Run now com
agente, modelo, reasoning, conta e confirmacao de custo; botao desabilitado quando ha task ativa, para
o usuario nao descobrir o guard por um `409`.
**Gate de saida**: testes de UI existentes verdes (`vitest` do pacote `views`); nenhuma chamada de
execucao fora de `/runs`.

### W7 - Indice de 4 estados (SOZINHO, por ultimo)
**Depende de**: W1, W2, W3, W5 verdes e P2 respondido.
**FILES_LOCKED**
```
server/migrations/NEXT_CANONICAL_b_active_task_unique_index.{up,down}.sql
server/pkg/db/generated/**            (regenerado por W1-owner; W7 e executada pelo MESMO dono de W1)
```
Preserva paralelismo de squad: chave `(issue_id, agent_id)`, **nao** `(issue_id)` - exatamente a
intencao registrada no comentario de `037_fix_pending_task_unique_index.up.sql`. A unica mudanca e o
predicado de status de 2 para 4 estados.
**Gate de saida**: `TestSquadParallelism_DifferentAgents_SameIssue_BothAllowed` verde e
`TestExplicitRun_ActiveRunningTask_409` verde.

---

## 3. Ordem de integracao e paralelismo real

```
W1 ──┬─> W2 ──┬─> W3 ──┬─> W5 ──> W6(flag on) ──> W7
     │        │        └─> W6 (pode comecar com contrato congelado)
     └────────┴─> (nada mais depende de W1 alem destes)
W4 ── paralela a W2/W3, mas obrigatoria ANTES de ligar o gate do Autopilot
```
- Paralelizavel de verdade: **W1 e W4** no inicio; depois **W5 e W6** apos W3.
- Serial obrigatorio: W1 -> W2 -> W3; e W7 sempre por ultimo e sozinho.
- **Auditoria forense em curso**: W2 toca os mesmos arquivos que a forense esta lendo
  (`issue.go`, `comment.go`, `issue_child_done.go`, `autopilot.go`). Por isso W2 **nao comeca** antes
  de a forense declarar leitura concluida nesses quatro arquivos, ou entao roda em **worktree
  isolado** sem tocar o checkout que a forense inspeciona. Recomendo worktree isolado: nao serializa a
  frota e nao contamina a evidencia.

---

## 4. Defaults de feature flag por onda

| onda | `MULTICA_EXECUTION_TRIGGER_DECOUPLED` | efeito observavel |
|---|---|---|
| W1 | `off` | tabelas criadas e inertes |
| W2 | `off` | comportamento identico a hoje; auditoria comeca a gravar |
| W3 | `off` | `/runs` existe e funciona, mas nada implicito mudou |
| W4 | `off` | secret populado; gate disponivel, ainda nao ligado no Autopilot |
| W5 | `off` para a suite legada, `on` para a suite nova | prova os dois lados |
| W6 | `warn` em staging | UI nova visivel; implicito ainda dispara, marcado |
| W7 | `on` | metadado nao enfileira; indice de 4 estados ativo |

Invariante em toda fase >= `on`: **campo omitido => nao executa**.

---

## 5. Rollback por onda

| onda | rollback | custo |
|---|---|---|
| W1 | `down` das tres migrations; tabelas aditivas e inertes | baixo |
| W2 | flag para `off`; codigo novo fica dormente | **segundos**, sem deploy novo se a flag for lida em runtime |
| W3 | flag `off`; rota `/runs` continua existindo e inofensiva | baixo |
| W4 | remover o secret volta ao estado atual (todo ledger fail-closed) | baixo, mas **para o Autopilot** |
| W5 | reverter arquivos de teste | nulo |
| W6 | reverter build de frontend | medio, exige rebuild |
| W7 | recriar o indice antigo de 2 estados | **unico nao trivial** |

Regra: **W7 nunca entra na mesma janela de outra onda**, para o rollback ser atribuivel.

---

## 6. Gates que precedem TODA a implementacao

- **G-a** os **cinco rulings de produto** da secao V2.7 (mencao, child-done, reassign, onboarding,
  Autopilot fail-closed). Sem eles, W2 nao tem contrato.
- **G-b** **P2**: existe fluxo legitimo que enfileira durante `running`? Bloqueia W7.
- **G-c** divergencia de titulo do **ORQ-41** (o card diz "Add safe non-triggering documentation
  comments to assig...", mais estreito que este escopo). Ruling do GTL.
- **G-d** decisao sobre o Autopilot parar no intervalo entre o gate e W4.
- **G-e** autorizacao escrita do owner para cada onda que muta (todas, menos este plano).
- **G-f** confirmacao de que a forense liberou os quatro arquivos de W2, ou aprovacao do worktree
  isolado.

---

## 7. Perguntas em aberto (declaradas, nao afirmadas)

- se o modelo de papeis atual permite separar `executar` e `cancelar` de `editar issue`: nao verifiquei.
- nomes exatos dos arquivos NOVOS que propus: sao sugestoes, nao fatos; o dono de cada onda pode
  renomear desde que o conjunto siga disjunto.
- numeros de migration: mantidos como `NEXT_CANONICAL_*` por regra; nao previ numero.
- estado atual da auditoria forense nos quatro arquivos de W2: nao consultei.

## 8. Nada mutado

Nenhum codigo, board, API, schema, build, teste, deploy ou credencial tocado. Nenhuma implementacao.
Nao sou registrar: nenhum card criado. Nao reexecutei ORQ-39, ORQ-40 nem issue preservada.

---
---

# SECAO V2 (correcao do BLOCK GTL-41 WAVES) - ORQ-41

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T14:30Z
- card: **ORQ-41**, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`, number 41
- **SUPERSEDE as secoes 2, 4, 5 e 6 da V1** deste documento. As demais permanecem.
- modo: READ-ONLY, apenas este documento. Nenhum codigo, board, migration, build, teste ou deploy.
- **PARADA para revisao independente** ao fim (secao V2.9).

## V2.1 Ownership unico de `generated` - por SEQUENCIA, nao por onda

Defeito da V1: `pkg/db/generated/**` aparecia em W1 **e** em W7. Isso e ownership duplo, mesmo com o
"mesmo dono", porque as duas ondas rodam em janelas diferentes e cada `sqlc generate` reescreve o
diretorio inteiro.

Regra corrigida, **sequencial e serializada**:

> `migrations/`, `pkg/db/queries/` e `pkg/db/generated/` formam **um unico lane serial chamado
> `LANE-DB`**, com **um dono para toda a duracao do ORQ-41**. Nenhuma outra onda toca esses tres
> diretorios. Dentro do LANE-DB, as entregas sao **sequenciais e numeradas** (S1, S2, S3), nunca
> concorrentes, e cada uma termina com `sqlc generate` + `go build ./...` verdes antes da proxima
> comecar.

| passo do LANE-DB | conteudo | quando |
|---|---|---|
| **S1** | migrations 127, 128, 129 (aditivas) + 3 arquivos de query + `sqlc generate` | antes de W2 |
| **S2** | (nada de schema) - apenas re-`sqlc generate` se W2/W3 adicionarem query nova | sob demanda, ainda pelo dono do LANE-DB |
| **S3** | migration 130 (indice de 4 estados) + `sqlc generate` | depois de W5 verde |

`pkg/db/generated/**` pertence **exclusivamente ao LANE-DB, em todos os passos**. Nenhuma onda de
codigo abre PR que toque esse diretorio; se precisar de query nova, **pede** ao dono do LANE-DB.

## V2.2 Reserva de numeros de migration - alinhada ao Z01, faixa **127-131**

Medido: maior migration atual e **`126_runtime_profile_protocol_family_native_runtimes.up.sql`**.
Logo a faixa livre comeca em 127, e a reserva do Z01 (127-131) e consistente. Alocacao fixa, para dois
agentes nao colidirem no mesmo numero:

| numero | placeholder V1 | conteudo | onda |
|---|---|---|---|
| **127** | `NEXT_CANONICAL_a` | `task_execution_request` (idempotencia) | LANE-DB S1 |
| **128** | `NEXT_CANONICAL_c` | `task_trigger_audit` (auditoria append-only) | LANE-DB S1 |
| **129** | `NEXT_CANONICAL_d` | `issue_workflow_authorization` (autorizacao child-done) | LANE-DB S1 |
| **130** | `NEXT_CANONICAL_b` | indice unico de **4 estados ativos** (BREAKING) | LANE-DB S3 |
| **131** | - | `task_ledger_summary` (dependencia do ledger duravel, W4) | LANE-DB S1 ou S2 |

Cada numero tem par `.up.sql` e `.down.sql`. A reserva e **exclusiva do ORQ-41**: qualquer outra
frente que precise de migration usa 132+, e isso precisa ser anunciado pelo GTL para nao haver
corrida por numero.

## V2.3 W4 corrigida - escopo completo do ledger, sem sobreposicao com W2

Defeito da V1: W4 listava apenas 3 arquivos e **nao** incluia migration, query, `generated`, o hook no
`task.go` nem a rota do daemon - e ao mesmo tempo W2 reivindicava `service/task.go`, criando
**sobreposicao real** com W4.

Resolucao: `service/task.go` pertence a **W2**, e W4 **nao** o edita. O ponto de contato passa a ser
uma **interface declarada em W2** e implementada em W4.

| item | onda | arquivo |
|---|---|---|
| migration 131 `task_ledger_summary` + query + `generated` | **LANE-DB** | `migrations/131_*`, `pkg/db/queries/task_ledger_summary.sql`, `pkg/db/generated/**` |
| interface `ReplayGateChecker` + `CheckOrAllow(interface)` | **W4** | `internal/daemon/commitledger/replay_gate.go` |
| checker duravel | **W4** | `internal/service/replay_gate_db.go` (NOVO) |
| popular `CommitLedgerHMACSecret` | **W4** | `internal/daemon/config.go` |
| gravacao do summary nos 3 pontos | **W4** | `internal/daemon/daemon.go` (linhas 4357, 4399, 4199+4200) |
| cliente do PUT de summary | **W4** | `internal/daemon/client.go` |
| **rota** `PUT /api/daemon/tasks/{id}/ledger-summary` | **W4** | `cmd/server/router.go` **conflito com W3** -> ver abaixo |
| campo do tipo `ReplayGateChecker` em `TaskService` e chamada do gate no autopilot | **W2** | `internal/service/task.go`, `internal/service/autopilot.go` |

**Conflito residual resolvido**: `cmd/server/router.go` e reivindicado por W3 (rota `/runs`) **e** por
W4 (rota de ledger-summary). Como um arquivo nao pode ter dois donos, **`router.go` pertence a W3**, e
W4 entrega a rota como **patch de registro** aplicado pelo dono de W3 na mesma PR - ou, alternativa
preferivel, W4 e agendada **depois** de W3 e recebe o arquivo. Escolha do GTL; o plano nao pode deixar
o arquivo com dois donos.

## V2.4 `CancelTasksForIssue` - **6 call sites**, nao 2

Defeito da V1: mencionava `2532` e `3033`. Medido, ha **seis** call sites, com semanticas diferentes,
e a V2 do design (reassign nao cancela) so afeta **dois** deles:

| # | arquivo:linha | contexto medido | veredito V2 |
|---|---|---|---|
| K1 | `issue.go:2532` | dentro de `if assigneeChanged` (update single) | **REMOVER** - reassign deixa de cancelar |
| K2 | `issue.go:3033` | idem, batch | **REMOVER** |
| K3 | `issue.go:2570` | `if statusChanged && issue.Status == "cancelled"` - comentario 2566-2568: *"cancellation is a user-initiated terminal action that should stop execution"* | **MANTER** - e acao terminal explicita do usuario |
| K4 | `issue.go:3059` | idem, batch | **MANTER** |
| K5 | `issue.go:2760` | `DeleteIssue`, antes de `FailAutopilotRunsByIssue` e do CASCADE | **MANTER** - apagar issue com task viva deixaria orfa |
| K6 | `issue.go:3115` | delete em batch, idem | **MANTER** |

Definicao em `internal/service/task.go:839` (doc em `:830`). Existe tambem uma variante por agente
(`task.go:855`), **fora do escopo** desta onda - registrada para nao ser confundida.

Consequencia para o plano: W2 mexe em **K1 e K2 apenas**; K3-K6 sao explicitamente **preservados**, e
o teste `TestCancel_...` de W5 precisa provar os dois lados - que reassign nao cancela **e** que
cancelar/deletar issue continua cancelando.

## V2.5 A flag **nao existe** - e a definicao de "off = byte-identical"

Medido: `grep -rn "MULTICA_EXECUTION_TRIGGER_DECOUPLED" --include=*.go .` -> **ZERO**. A flag e
**nova** e precisa ser criada, com dono unico:

| item | onda | arquivo |
|---|---|---|
| leitura da flag + parser de 3 valores (`off`/`warn`/`on`) | **W2** | `internal/service/execution_trigger.go` (NOVO) |
| default | **W2** | `off`, e **fail-closed no sentido de nao mudar comportamento**: valor invalido ou ausente => `off` |

**Definicao operacional de `off = byte-identical`**, para o gate de W2 ser verificavel e nao uma
promessa:
1. com `off`, **nenhum** dos 21 call sites do inventario muda de decisao: as mesmas guardas
   (`shouldEnqueueAgentTask`, `isAgentAssigneeReady`, `shouldEnqueueSquadLeaderOnAssign`,
   `isSquadLeaderReady`, triggers de comentario, `HasPendingTaskForIssueAndAgent`) sao consultadas na
   mesma ordem e com os mesmos argumentos;
2. com `off`, `CancelTasksForIssue` continua sendo chamado em **todos os 6** sites, inclusive K1/K2;
3. com `off`, o **unico** efeito novo permitido e **escrita em `task_trigger_audit`** - portanto
   "byte-identical" e definido como: **identico em toda escrita EXCETO a tabela de auditoria**;
4. verificacao: a suite legada completa passa com `off` **sem alteracao de nenhum arquivo de teste
   existente**. Se algum teste legado precisar mudar, a premissa de `off` foi violada.

## V2.6 Matriz exata de **32** testes e o `TestMain` sem falso-verde

### V2.6.1 O risco de falso-verde e real e medido
`internal/handler/handler_test.go:38` define `func TestMain(m *testing.M)`, e o pacote usa
`t.Skip("database not available")` em varios pontos (por exemplo `agent_access_test.go:133, 166, 210,
234, 280`). Ou seja: **sem banco, os testes SKIPam e a suite fica verde**. Uma matriz de
custo-zero que dependa de banco e que faca `t.Skip` na sua ausencia **nao prova nada** - e exatamente
o falso-verde que o BLOCK aponta.

Regra obrigatoria para as 32: **nenhum** dos testes desta matriz pode usar `t.Skip`. Onde precisar de
banco, `TestMain` deve **falhar** (`log.Fatal`) se o banco nao estiver disponivel, e a suite de
ORQ-41 roda com tag propria (por exemplo `-run 'TestORQ41'`) num alvo de CI que **exige** banco. Os
testes de contagem de linhas nao podem ser satisfeitos por skip.

### V2.6.2 Matriz (32)
| # | teste | onda | precisa DB | prova |
|---|---|---|---|---|
| 1 | `TestORQ41_Assign_ZeroCost` | W5 | sim | C1/C3 |
| 2 | `TestORQ41_Reassign_ZeroCost` | W5 | sim | C1/C3 |
| 3 | `TestORQ41_Reassign_DoesNotCancel` | W5 | sim | **K1/K2 removidos** |
| 4 | `TestORQ41_CancelStatus_StillCancels` | W5 | sim | **K3/K4 preservados** |
| 5 | `TestORQ41_DeleteIssue_StillCancels` | W5 | sim | **K5/K6 preservados** |
| 6 | `TestORQ41_BacklogPromotion_ZeroCost` | W5 | sim | C2/C4 |
| 7 | `TestORQ41_BatchUpdate_ZeroCost` | W5 | sim | C3/C4 |
| 8 | `TestORQ41_CreateWithAgentAssignee_ZeroCost` | W5 | sim | C13 |
| 9 | `TestORQ41_CreateWithSquadAssignee_ZeroCost` | W5 | sim | **C14** |
| 10 | `TestORQ41_SquadHelperCallSite_ZeroCost` | W5 | sim | **C5 (squad.go:1007)** |
| 11 | `TestORQ41_OrdinaryComment_ZeroCost` | W5 | sim | C7 |
| 12 | `TestORQ41_MentionAgent_ZeroCost` | W5 | sim | C9, invertido |
| 13 | `TestORQ41_MentionSquadLeader_ZeroCost` | W5 | sim | C8 |
| 14 | `TestORQ41_CommentSquadAssignee_ZeroCost` | W5 | sim | C6 |
| 15 | `TestORQ41_Onboarding_RespectsRuling` | W5 | sim | C12 |
| 16 | `TestORQ41_ChildDone_NoAuth_ZeroCost` | W5 | sim | **C10/C11** |
| 17 | `TestORQ41_ChildDone_AuthNoCostAck_ZeroCost` | W5 | sim | migration 129 |
| 18 | `TestORQ41_ChildDone_Authorized_EnqueuesOnce` | W5 | sim | 129 |
| 19 | `TestORQ41_ChildDone_WebhookRedelivery_Idempotent` | W5 | sim | **I3 github.go:1331** |
| 20 | `TestORQ41_ChildDone_SingleAndBatch_Identical` | W5 | sim | I1/I2 |
| 21 | `TestORQ41_Autopilot_CallsGate` | W5 | nao | C15/C16 |
| 22 | `TestORQ41_Autopilot_MissingState_FailsClosed` | W5 | nao | W4 |
| 23 | `TestORQ41_Autopilot_Ambiguous_FailsClosed` | W5 | nao | W4 |
| 24 | `TestORQ41_Autopilot_Expired_FailsClosed` | W5 | nao | W4 |
| 25 | `TestORQ41_Autopilot_NoPriorTask_Allowed` | W5 | nao | correlacao |
| 26 | `TestORQ41_Run_201_AndImmutableAudit` | W5 | sim | 127+128 |
| 27 | `TestORQ41_Run_DuplicateKey_200_SingleTask` | W5 | sim | 127 |
| 28 | `TestORQ41_Run_ConcurrentClicks_OneTask` | W5 | sim | 127 |
| 29 | `TestORQ41_Run_ActiveRunningTask_409` | W5 | sim | **130; VERMELHO antes de S3** |
| 30 | `TestORQ41_SquadParallelism_DifferentAgents_BothAllowed` | W5 | sim | 130 nao quebrou squad |
| 31 | `TestORQ41_Run_MissingCostAck_422` + `NonUUIDv4Key_422` + `WrongRBAC_403` | W5 | sim | W3 |
| 32 | `TestORQ41_NoUnauthorizedEnqueueCallSites` | W5 | **nao** | estrutural, lista branca C1-C17 + I1-I4 + K1-K6 |

Dois testes **devem estar vermelhos** antes da correcao correspondente: **#29** (antes da migration
130) e **#16** (antes de W2). Se passarem antes, a matriz esta medindo a coisa errada.
O #32 nao precisa de banco e por isso e o **unico** que pode rodar em qualquer CI - deve ser
obrigatorio no pre-commit.

## V2.7 Rollback de W7/S3 - **sem recriar o indice errado**

Defeito da V1: dizia "recriar o indice antigo", ambiguo, e o BLOCK aponta com razao. Existem **dois**
indices historicos, e recriar o errado reintroduz um bug ja corrigido:

- `idx_one_pending_task_per_issue` - **antigo e INCORRETO**, por issue apenas. O proprio
  `037_fix_pending_task_unique_index.up.sql` o derruba e explica: *"the old index only allowed one
  pending task per issue across ALL agents. This caused different agents' pending tasks to block each
  other."* **NUNCA recriar.**
- `idx_one_pending_task_per_issue_agent` - **o vigente**, criado por 037:
```sql
CREATE UNIQUE INDEX idx_one_pending_task_per_issue_agent
    ON agent_task_queue (issue_id, agent_id)
    WHERE status IN ('queued', 'dispatched');
```

Portanto `130_*.down.sql` deve, **literalmente**: derrubar `idx_one_active_task_per_issue_agent` e
recriar **exatamente** a definicao acima, com os dois estados e a chave `(issue_id, agent_id)`. E o
`.down.sql` precisa ser **testado** (V2.8), nao apenas escrito.

## V2.8 Harness executavel de migration

Existe e e o caminho oficial: `cmd/migrate/main.go:106-112`, `Usage: go run ./cmd/migrate <up|down>`.
Gate de cada passo do LANE-DB, em banco **descartavel** (nunca no Postgres do ORQ1):

```bash
# 1. subir
go run ./cmd/migrate up
# 2. conferir objetos criados (metadado, sem dados)
#    \d+ task_execution_request / task_trigger_audit / issue_workflow_authorization
#    \di idx_one_active_task_per_issue_agent
# 3. descer
go run ./cmd/migrate down
# 4. conferir que o indice VIGENTE de 037 voltou com os 2 estados e a chave (issue_id, agent_id)
# 5. subir de novo (idempotencia do par up/down)
go run ./cmd/migrate up
```
Nenhum passo do LANE-DB e aceito sem esse ciclo up->down->up verde. `cmd/migrate` usa advisory lock
4246 (`cmd/migrate/main.go:21`), o que serializa migrations concorrentes - mais uma razao para o
LANE-DB ser serial.

## V2.9 Diff desta correcao e hash

Alteracoes da V2 sobre a V1, para o revisor conferir sem reler tudo:

| # | secao V1 | mudanca |
|---|---|---|
| D1 | 2 (W1/W7) | `generated` deixa de ter dono por onda e passa a LANE-DB serial S1/S2/S3 |
| D2 | 2 | placeholders `NEXT_CANONICAL_a..d` mapeados para **127, 128, 129, 130** e novo **131** |
| D3 | 2 (W4) | W4 ganha migration 131, query, `generated`(via LANE-DB), `daemon.go`, `client.go` e a rota; `service/task.go` fica em W2 com interface; conflito de `router.go` explicitado |
| D4 | 2 (W2) | `CancelTasksForIssue` corrigido de 2 para **6** sites, com veredito por site (K1/K2 remover, K3-K6 manter) |
| D5 | 4 | flag declarada **inexistente**; criacao atribuida a W2; `off` definido operacionalmente em 4 clausulas verificaveis |
| D6 | 2 (W5) | matriz de 27 -> **32** testes, com coluna de DB, prova por call site e dois testes que devem falhar antes |
| D7 | 5 (W7) | rollback especifica **qual** indice recriar e proibe explicitamente o `idx_one_pending_task_per_issue` |
| D8 | nova | harness executavel `go run ./cmd/migrate up/down` com ciclo obrigatorio up->down->up |
| D9 | nova | regra anti-falso-verde: proibido `t.Skip` na matriz; `TestMain` falha sem banco |

Hash de integridade desta secao V2 (SHA-256 do arquivo apos a insercao) registrado no check-out
correspondente, para o revisor confirmar que leu a mesma versao.

## V2.10 PARADA para revisao independente

**Nao prossigo.** Nada aqui foi implementado; W1/LANE-DB nao comeca antes de:
- revisao independente desta V2 (mesmo revisor do GTL-41 WAVES, de preferencia);
- os **cinco rulings** de produto da V2.7 do design;
- confirmacao do GTL da reserva **127-131** contra o Z01 e contra outras frentes;
- decisao sobre o dono de `cmd/server/router.go` (W3 ou W4 depois de W3).

## V2.11 Nada mutado

Apenas este documento foi atualizado. Nenhum codigo, board, API, schema, migration, build, teste,
deploy ou credencial. Nao alterei assignee nem postei comentario. Nenhum card criado.

---
---

# SECAO V3 - CORRECAO DEFINITIVA DA RESERVA DE MIGRATIONS - ORQ-41

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T15:12Z
- card: **ORQ-41**, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`, number 41
- **SUPERSEDE a secao V2.2 integralmente** e emenda V2.1, V2.3 e V2.8. O resto da V2 permanece.
- modo: READ-ONLY, apenas este documento. Nenhum codigo, board, migration, build, teste ou deploy.

## V3.1 EU ESTAVA ERRADO NA V2 - e tenho a prova de colisao

A V2 reivindicou a faixa **127-131** para o ORQ-41. **Retirado.** A reivindicacao ja estava colidida
no momento em que a escrevi, e eu nao teria descoberto olhando so a branch de integracao.

Evidencia medida agora, read-only:

```
git worktree list  ->  10+ worktrees ativos, entre eles:
  agent/opus48-a/orq-12-task-usage-account-id
  agent/codex-b/orq-21
  agent/opus48-b/orq-13-thinking-level
  agent/codex-b/orq-18-runtime-delete-ui
  agent/codex-a/orq-23
  agent/kiro-opus5/orq-26-contract-fix
  agent/agy-a8/orq17-auth-regression
```
E no worktree do **ORQ-13** a migration **127 JA EXISTE**:
```
/home/ec2-user/workspace/worktrees/gtl-i03-orq13-phase1/.../migrations/
  126_runtime_profile_protocol_family_native_runtimes.up.sql
  127_task_usage_thinking_level.down.sql      <-- JA CRIADA por outra frente
  127_task_usage_thinking_level.up.sql
```
Enquanto isso, na branch de integracao o maior numero continua **126** - e foi exatamente essa leitura
parcial que me levou ao erro. **A branch de integracao nao e fonte de verdade para alocacao de
numero**, porque toda frente cria a migration no proprio worktree e o numero so vira visivel no merge.

Alem do ORQ-13, ha pelo menos duas frentes que quase certamente vao precisar de numero e ainda nao
criaram: **ORQ-12** (adicionar `account_id` a `task_usage`) e **ORQ-21** (popular `accounts`,
`approved_accounts` e `assignments`). Ambas com worktree aberto e migrations paradas em 126.

Conclusao: **nenhuma onda do ORQ-41 reivindica numero de migration.** Reivindicar numero em plano e
uma corrida que o ultimo a mergear perde.

## V3.2 Placeholders subordinados a reserva central

O plano passa a usar **apenas** placeholders simbolicos, sem numero, e o numero e atribuido por um
**registrador central de migrations** no momento da criacao do arquivo - nunca no plano.

| placeholder canonico | conteudo | onde e usado neste plano |
|---|---|---|
| `MIG_EXEC_REQUEST` | `task_execution_request` (idempotencia de execucao) | LANE-DB, passo 1 |
| `MIG_TRIGGER_AUDIT` | `task_trigger_audit` (auditoria append-only) | LANE-DB, passo 1 |
| `MIG_WORKFLOW_AUTH` | `issue_workflow_authorization` (autorizacao child-done) | LANE-DB, passo 1 |
| `MIG_LEDGER_SUMMARY` | `task_ledger_summary` (dependencia do ledger duravel) | LANE-DB, passo 1 ou 2 |
| `MIG_ACTIVE_TASK_INDEX` | indice unico de 4 estados ativos (BREAKING) | LANE-DB, passo final |

Regras de subordinacao:
1. **O plano nunca escreve digito.** Todo artefato, teste, runbook e PR referencia o placeholder.
2. **O numero e pedido ao registrador central** imediatamente antes de criar o arquivo, e a resposta e
   registrada na evidencia do ORQ-41 **depois** de atribuida - nao antes.
3. **A verificacao de proximo numero livre varre TODOS os worktrees**, nao apenas a integracao:
   ```bash
   for w in $(git worktree list --porcelain | awk '/^worktree /{print $2}'); do
     ls "$w"/multica-auth-work/server/migrations/*.up.sql 2>/dev/null
   done | sed 's/.*\///' | cut -d_ -f1 | sort -n | tail -1
   ```
   Este comando e o **unico** metodo aceitavel de descobrir o maior numero em uso. Ele encontra o
   `127` do ORQ-13, que a leitura da integracao nao encontra.
4. **Se o numero atribuido colidir no merge**, a renumeracao e do ORQ-41 - somos os ultimos a chegar,
   e nao pedimos exclusividade de faixa.
5. Nenhuma faixa e reservada para ORQ-41. A V2 pedia 127-131; **isso esta cancelado**.

## V3.3 LANE-DB unico, sem sub-lanes concorrentes

Emenda a V2.1: some a nomenclatura S1/S2/S3 como se fossem entregas paralelizaveis. O LANE-DB e **um
lane, um dono, uma fila**, e seus passos sao apenas posicoes na fila:

> **LANE-DB**: dono unico para toda a duracao do ORQ-41. Detem exclusivamente
> `multica-auth-work/server/migrations/`, `multica-auth-work/server/pkg/db/queries/` e
> `multica-auth-work/server/pkg/db/generated/`. Executa em **fila serial**: pede numero ao registrador,
> cria o par `.up.sql`/`.down.sql`, escreve a query, roda `sqlc generate`, roda o gate executavel de
> V3.5, e so entao pega o proximo item. Nenhuma outra onda abre PR que toque esses tres diretorios; se
> precisar de query, **pede** ao dono do LANE-DB.

Ordem da fila:
1. `MIG_EXEC_REQUEST`, `MIG_TRIGGER_AUDIT`, `MIG_WORKFLOW_AUTH` (aditivas) - liberam W2 e W3.
2. `MIG_LEDGER_SUMMARY` - libera W4.
3. `MIG_ACTIVE_TASK_INDEX` - **por ultimo**, apos W5 verde, e sozinho na janela.

`pkg/db/generated/**` pertence ao LANE-DB em **todos** os passos, sem excecao. Isso encerra o
ownership duplo que a V1 tinha e que a V2 corrigiu pela metade ao falar de "mesmo dono em janelas
diferentes".

## V3.4 Dono unico de `cmd/server/router.go`: **W3**

A V2 deixou a decisao aberta, o que e ownership ambiguo - o defeito que o plano existe para evitar.
Decidido no plano, sujeito a veto do GTL:

> **`cmd/server/router.go` pertence a W3, e somente a W3.**
> W4 **nao** edita `router.go`. A rota `PUT /api/daemon/tasks/{id}/ledger-summary` e **especificada**
> por W4 e **registrada** por W3, na PR de W3. Se W4 ficar pronta antes, ela espera: a rota nao e
> caminho critico do checker duravel, porque o daemon pode gravar o summary somente apos a rota
> existir, e ate lá o gate opera com o registry em memoria, falhando fechado.

Consequencia de agenda: **W3 precisa conhecer o contrato da rota de W4 antes de fechar**. Ou seja W4
entrega o **contrato** (payload e semantica) como artefato de documento antes de W3 codificar, e a
implementacao do handler de ledger-summary fica em arquivo proprio de W4
(`internal/handler/daemon_ledger_summary.go`), com apenas **uma linha de registro** em `router.go`
pertencendo a W3.

## V3.5 Gates executaveis, com comando e criterio de aprovacao

Cada gate abaixo e um comando real, com saida esperada. Nenhum gate e "verificar que esta correto".

### G-DB (por item da fila do LANE-DB, em banco DESCARTAVEL)
```bash
# 0. descobrir o maior numero em uso em TODOS os worktrees (nunca so na integracao)
for w in $(git worktree list --porcelain | awk '/^worktree /{print $2}'); do
  ls "$w"/multica-auth-work/server/migrations/*.up.sql 2>/dev/null
done | sed 's/.*\///' | cut -d_ -f1 | sort -n | tail -1
# criterio: o numero pedido ao registrador e > este valor

# 1. ciclo up / down / up
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate up
# criterio: exit 0 nas tres; nenhuma exige intervencao manual

# 2. sqlc coerente
sqlc generate && git diff --stat -- pkg/db/generated
# criterio: diff contido EXCLUSIVAMENTE em pkg/db/generated

# 3. compila
go build ./...
# criterio: exit 0
```

### G-W2 (`off = byte-identical`, definicao de V2.5 tornada executavel)
```bash
MULTICA_EXECUTION_TRIGGER_DECOUPLED=off go test ./... -count=1
# criterio: verde SEM alterar nenhum arquivo _test.go pre-existente
git diff --stat -- '*_test.go'
# criterio: vazio para testes pre-existentes; apenas arquivos NOVOS de W5 aparecem
```

### G-W5 (anti-falso-verde, medido em V2.6.1)
```bash
grep -rn "t.Skip" <arquivos da matriz ORQ-41>
# criterio: ZERO ocorrencias

# a matriz roda exigindo banco; sem banco deve FALHAR, nao passar
go test ./internal/handler ./internal/service -run 'TestORQ41' -count=1
# criterio com banco: verde nos 32
# criterio sem banco: FALHA (nao skip) - provando que nao ha falso-verde
```

### G-W5-negativo (os dois testes que devem estar vermelhos antes)
```bash
go test ./... -run 'TestORQ41_Run_ActiveRunningTask_409' -count=1   # antes de MIG_ACTIVE_TASK_INDEX
go test ./... -run 'TestORQ41_ChildDone_NoAuth_ZeroCost' -count=1   # antes de W2
# criterio: AMBOS falham. Se passarem, a matriz mede a coisa errada.
```

### G-ROLLBACK (`MIG_ACTIVE_TASK_INDEX`, o unico nao trivial)
```bash
go run ./cmd/migrate down
psql -c "\di idx_one_pending_task_per_issue_agent"
psql -c "SELECT indexdef FROM pg_indexes WHERE indexname='idx_one_pending_task_per_issue_agent'"
# criterio: existe, com chave (issue_id, agent_id) e WHERE status IN ('queued','dispatched')
psql -c "\di idx_one_pending_task_per_issue"
# criterio: NAO existe. Recriar o indice por-issue reintroduz o bug que 037 corrigiu.
```

## V3.6 Diff V2 -> V3 e hash

| # | mudanca | motivo |
|---|---|---|
| E1 | **reserva 127-131 CANCELADA** | colisao provada: ORQ-13 ja tem `127_task_usage_thinking_level` no worktree `gtl-i03-orq13-phase1` |
| E2 | numeros substituidos por 5 placeholders simbolicos subordinados a registrador central | plano nao escreve digito |
| E3 | metodo de descoberta do proximo numero varre **todos os worktrees** | a integracao mostra 126 e mente por omissao |
| E4 | ORQ-41 aceita **renumerar** em caso de colisao no merge | somos os ultimos a chegar |
| E5 | S1/S2/S3 viram posicoes de uma **fila serial unica** | evita leitura de sub-lanes paralelas |
| E6 | `router.go` decidido: **W3**, com handler de ledger em arquivo proprio de W4 | ownership ambiguo era o defeito |
| E7 | 6 gates transformados em comandos com criterio | "verificar que esta correto" nao e gate |

Hash de integridade do arquivo apos esta insercao: registrado no check-out desta entrega.

## V3.7 Parada e peer review

**Nao prossigo.** Peco peer review independente desta V3, e o LANE-DB nao comeca antes de:
- peer review aprovado;
- os **cinco rulings** de produto (V2.7 do design);
- **confirmacao de quem e o registrador central de migrations** - hoje nao sei se existe; se nao
  existir, isso e um pre-requisito do ORQ-41 e afeta ORQ-12, ORQ-13 e ORQ-21 igualmente;
- veto ou aceite do GTL sobre `router.go` pertencer a W3.

## V3.8 Nada mutado

Apenas este documento. Nenhum codigo, board, API, schema, migration, build, teste, deploy ou
credencial. Nao alterei assignee nem postei comentario. Nenhum card criado. Leitura de worktrees foi
somente `git worktree list` e `ls` de diretorio de migrations.
