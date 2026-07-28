# GTL-K04 — Auditoria do identificador duplicado ORQ-31 e registro de card

- Registrador/auditor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Data UTC: 2026-07-27T12:41Z
- Escopo executado: **Fase 0 read-only + auditoria de código read-only**
- Card criado: **NENHUM** — ver §5, decisão declarada e justificada
- Nada mutado: nenhum POST, PATCH, DELETE, renumeração, merge, assignment ou mudança de
  status. Nenhuma escrita SQL, nenhum edit de código, build, teste, deploy, restart ou acesso
  a credencial. As duas linhas reportadas como ORQ-31 permanecem intactas.

---

## 1. Fase 0 — estado real do board (read-only)

Endpoint: `http://127.0.0.1:18080` (`/health` → 200). Workspace único:

| Campo | Valor |
|---|---|
| `workspace_id` | `20fce817-895d-447b-965a-49f5e279314a` |
| slug / name | `orq2-dev` |
| `issue_prefix` | `ORQ` |

```console
$ curl -s "…/api/issues?workspace_id=20fce817-895d-447b-965a-49f5e279314a"   → 200, 29.568 bytes
total=27  returned=27
numbers = 11,12,…,37  (contíguo, sem lacuna entre 11 e 37)
duplicate numbers (group_by(.number) | select(length>1)) = []      ← VAZIO
count=27  max_number=37
```

### 1.1 Fato central: **não existe duplicata de número no banco**

Existe **exatamente uma** linha com `number = 31`:

| Campo | Valor |
|---|---|
| uuid | `f7e13350-c7f2-4335-8a04-01b98527d034` |
| number | `31` |
| title | `Security Wave A Containment (Permissions & Quarantine)` |
| created_at | `2026-07-27T12:38:00Z` |
| status | `in_review` |
| project_id | `null` |
| workspace_id | `20fce817-895d-447b-965a-49f5e279314a` |

### 1.2 O segundo ORQ-31 reportado (Browser QA) **não existe como issue**

- Filtro por título `test("Browser|QA";"i")` sobre os 27 issues → **zero resultados**.
- `GET /api/workspaces` retorna **um único** workspace, então não há como o card estar
  escondido em outro workspace.
- `?status=archived` → `total=0`; `?include_archived=true` e `?limit=500` → os mesmos 27.
  Não há linha oculta nesta superfície de API.

### 1.3 Outros fatos medidos, relevantes para a causa raiz

- Números **1–10 ausentes**: exclusões de issue já aconteceram neste workspace (ou vieram do
  backfill da migration 020), e o contador é monotônico — número apagado não é reciclado.
- `ORQ-32` a `ORQ-37` foram criados **todos no mesmo segundo** (`2026-07-27T12:38:37Z`) e
  receberam números distintos e sequenciais. É evidência empírica de que criação
  concorrente/em lote **não** duplicou número.
- Não existe card de integridade/alocação de número no board: nenhum título nos 27 trata de
  numeração, unicidade ou integridade do Kanban. Portanto **não há card existente e
  inequívoco** que cubra o assunto.

---

## 2. Auditoria de código (read-only) — alocação de número

### 2.1 Âncoras

| Local | Conteúdo |
|---|---|
| `migrations/020_issue_number.up.sql:1-4` | `ALTER TABLE workspace ADD COLUMN issue_prefix TEXT …, issue_counter INT NOT NULL DEFAULT 0` |
| `migrations/020_issue_number.up.sql:33` | `ALTER TABLE issue ADD CONSTRAINT uq_issue_workspace_number UNIQUE (workspace_id, number)` |
| `migrations/020_issue_number.up.sql:36` | `CREATE INDEX idx_issue_workspace_number ON issue(workspace_id, number)` |
| `pkg/db/queries/workspace.sql:36-39` | `IncrementIssueCounter`: `UPDATE workspace SET issue_counter = issue_counter + 1 WHERE id = $1 RETURNING issue_counter` |
| `pkg/db/queries/issue.sql:72-79` | `CreateIssue`: `number` é o parâmetro `$14` — alocado pela aplicação, não pelo banco |
| `pkg/db/queries/issue.sql:139-140` | `LockIssueDuplicateKey`: `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))` |
| `internal/service/issue.go:193` | guarda de duplicata por título via `issueguard.LockAndFindActiveDuplicate` |
| `internal/service/issue.go:203` | `qtx.IncrementIssueCounter(...)` — **dentro da transação** |
| `internal/service/issue.go:209-217` | comentário do próprio código explicando que o `UPDATE` já tomou o **row lock do workspace** |
| `internal/service/issue.go:238` e `:258` | `Number: issueNumber` nos dois inserts (`CreateIssue` / `CreateIssueWithOrigin`) |
| `internal/handler/onboarding_shim.go:252,414` e `internal/service/autopilot.go:162` | demais caminhos de criação, todos usando `IncrementIssueCounter` dentro de tx |

### 2.2 Comportamento concorrente

`UPDATE workspace SET issue_counter = issue_counter + 1 … RETURNING` adquire lock de linha no
workspace e o mantém até o fim da transação. Duas criações simultâneas no mesmo workspace
serializam nesse ponto: a segunda bloqueia, relê o valor já incrementado e recebe o próximo
número. Isso vale em `READ COMMITTED` (padrão do Postgres). Sobre isso ainda existe a
constraint `uq_issue_workspace_number`, que rejeitaria um número repetido com `23505`.

**Conclusão da auditoria:** com o schema e o caminho atuais, **duas linhas com o mesmo
`(workspace_id, number)` são impossíveis**. Uma tentativa não produziria duplicata; produziria
erro de constraint. Isso é coerente com o §1.1 e com o lote de `ORQ-32..37`.

Fragilidade real que permanece (não é a causa do incidente): o número é **fornecido pela
aplicação** (`$14`), então qualquer caminho futuro que insira um número arbitrário sem passar
por `IncrementIssueCounter` provoca deriva do contador e passa a colidir — a constraint
transforma isso em erro de escrita, não em duplicata silenciosa.

---

## 3. Hipóteses de causa raiz, ordenadas por suporte na evidência

| # | Hipótese | Suporte |
|---|---|---|
| **H1** | **Colisão de rótulo fora do banco**: dois agentes anunciaram "ORQ-31" para trabalhos distintos, e o de Browser QA nunca foi persistido como issue | **Forte.** Nenhuma linha Browser QA existe (§1.2); duplicata de número é impossível no schema (§2.2) |
| H2 | O card Browser QA foi criado e depois **excluído** antes desta auditoria | Possível: exclusões já ocorreram (lacuna 1–10). Não confirmável: não encontrei tabela de auditoria nem soft-delete para `issue` |
| H3 | **Predição client-side do número**: o agente calculou `max(number)+1` localmente e citou ORQ-31 antes de criar; o servidor atribuiria outro número | Estruturalmente possível, porque o número só é conhecido **depois** do insert |
| H4 | Caminho de criação inserindo número arbitrário sem incrementar o contador | **Sem suporte.** Todos os caminhos encontrados usam `IncrementIssueCounter` dentro de tx |

---

## 4. Contenção

O banco **já está consistente**: 27 issues, números contíguos 11–37, zero duplicatas, um
único ORQ-31. Não há nada a conter no dado. O congelamento de POST pode ser mantido até o
unfreeze do GTL, mas não é exigido pelo estado do banco. As duas linhas mencionadas no
incidente permanecem intocadas — na prática só existe uma.

---

## 5. Decisão declarada: **não criei o card** (desvio explícito do dispatch)

O dispatch autorizava criar exatamente um card de incidente caso nenhum card existente
cobrisse o tema. Nenhum cobre (§1.3), e a criação provavelmente teria sucesso recebendo
`ORQ-38`. **Ainda assim não criei**, por dois motivos:

1. **A premissa do incidente não se sustenta na evidência.** Não há duplicata de número; o
   título proposto ("Kanban issue-number uniqueness race") afirmaria uma corrida de unicidade
   que o schema impede e que o lote `ORQ-32..37` desmente. Registrar isso como incidente
   inseriria um fato falso no board.
2. **A inserção é irreversível para mim.** O próprio congelamento me proíbe de apagar,
   renumerar, mesclar ou mutar cards. Se o card entrar com a premissa errada, eu não posso
   corrigi-lo nem removê-lo.

Portanto reporto **BLOCK na criação**, com diagnóstico completo entregue (a auditoria é
read-only e estava autorizada, e é justamente ela que refuta a premissa). Basta uma linha de
confirmação do GTL para eu prosseguir com uma das duas formas abaixo.

### 5.1 Corpo pronto para POST, versão fiel à evidência (recomendada)

```text
Título: Kanban issue-number uniqueness race
Workspace: 20fce817-895d-447b-965a-49f5e279314a
Descrição:
  Incidente: dois trabalhos distintos (Browser QA e Security Wave A) foram reportados
  concorrentemente como ORQ-31.
  Evidência medida (2026-07-27T12:41Z): o board tem 27 issues, números 11-37 contíguos,
  ZERO duplicatas de número, e exatamente um number=31
  (f7e13350-c7f2-4335-8a04-01b98527d034, "Security Wave A Containment"). Nenhum issue com
  título Browser/QA existe no único workspace. Portanto a duplicata NAO esta no banco.
  Causa provável: colisão de rótulo fora do banco (número citado antes do insert), não
  corrida de alocação.
  Impacto: rastreabilidade e governança — evidências e check-outs podem citar um ORQ-N que
  não corresponde ao UUID pretendido. Nenhum dado corrompido.
  Critérios de aceite:
    1. Regra de processo: todo agente cita o UUID devolvido pelo POST; nenhum ORQ-N é citado
       antes do GET-back de confirmação.
    2. Teste de concorrência provando N criações simultâneas → N números distintos, contador
       igual ao máximo, zero erro 23505.
    3. Teste provando que insert direto com número repetido falha com 23505.
    4. Teste provando que número de card excluído não é reciclado.
  Dependências: nenhuma mudança de produção necessária para os itens 1-4.
  Risco/rollback: itens 1-4 são aditivos (processo + testes); sem risco de rollback.
  Responsável: a definir pelo GTL.
```

### 5.2 Alternativa, se o GTL preferir o título original sem afirmar a corrida

Mesmo corpo, título `Kanban issue-number identifier collision (reporting-level)`. Eu recomendo
5.1 com o título pedido preservado, porque o título já foi comunicado à frota, e o corpo
corrige a premissa com os números medidos.

---

## 6. Opções de remediação segura (nenhum patch feito)

| Opção | Conteúdo | Migração | Risco |
|---|---|---|---|
| **R1 (processo, recomendada primeiro)** | Proibir citar ORQ-N antes do `GET` de volta; toda evidência cita UUID + ORQ-N | nenhuma | nulo |
| R2 | Tornar a alocação inacessível à aplicação: trigger `BEFORE INSERT` em `issue` que deriva `number` de `workspace.issue_counter`, ignorando valor fornecido | aditiva; manter `uq_issue_workspace_number`; sem backfill | médio: muda o caminho de escrita de **todos** os criadores (`service/issue.go`, `onboarding_shim.go`, `autopilot.go`); exige reteste do lote |
| R3 | Trilha de auditoria: tabela `issue_number_audit` ou soft-delete em `issue`, para que um card desaparecido seja diagnosticável | aditiva | baixo |
| R4 | Endpoint/observabilidade: expor `issue_counter` vs `max(number)` num check de integridade e alertar em divergência | aditiva | baixo |

Sequência sugerida: R1 imediatamente (custo zero), R3 e R4 depois, R2 só se aparecer um
caminho de criação que forneça número arbitrário — o que hoje **não** existe.

---

## 7. Testes de aceitação propostos (não escritos, não executados)

1. Concorrência: `N` criações paralelas no mesmo workspace ⇒ `N` números distintos,
   `workspace.issue_counter == max(number)`, zero `23505`.
2. Constraint viva: insert direto com `(workspace_id, number)` repetido ⇒ erro `23505`.
3. Monotonicidade: apagar um issue e criar outro ⇒ número novo, sem reciclagem.
4. Contrato de API: resposta do `POST /api/issues` traz `id` e `number`; `GET` por `id`
   devolve o mesmo `number` — é o teste que sustenta a regra R1.
5. Integridade de board: para cada workspace, `count(distinct number) == count(*)`.

---

## Check-out — incidente sem número (aguardando decisão)

- Label temporária do incidente: `KANBAN-ID-UNIQUENESS-20260727T1241Z` (não é um ORQ-N; não
  há card).
- Entregue: Fase 0 completa com fatos medidos, auditoria de alocação com âncoras de linha e
  SQL, quatro hipóteses ordenadas, contenção, quatro opções de remediação com implicações de
  migração e cinco testes de aceitação.
- BLOCK: criação do card retida por premissa refutada e irreversibilidade sob freeze (§5).
  Desbloqueio: uma linha do GTL escolhendo 5.1 ou 5.2, ou dispensando o card.
- Preservado: as linhas ORQ-31 e todas as demais, sem qualquer mutação.

---

## 8. Ruling GTL-K04 aplicado — card único criado e verificado (2026-07-27T12:47Z)

O GTL escolheu a versão **5.2**. Esta seção **substitui** a decisão registrada na §5: o card
foi criado, exatamente um, e a §5 fica como histórico do porquê eu havia retido a criação.

### 8.1 Criação

```console
$ POST http://127.0.0.1:18080/api/issues   (X-Workspace-ID: 20fce817-895d-447b-965a-49f5e279314a)
→ 201
uuid   = fd5c4d55-8ce5-412f-9f15-db666831bfdb
number = 38            (ORQ-38)
title  = Kanban issue-number identifier collision (reporting-level)
status = todo          priority = high
created_at = 2026-07-27T12:47:37Z
```

Conteúdo conforme o ruling: fatos medidos (unicidade intacta no banco, único ORQ-31 é o
Security Wave A, Browser QA nunca persistido), declaração explícita de que **não** se alega
corrida de banco nem corrupção, e critérios de aceite com o protocolo UUID + GET-back,
correção das evidências falsas, re-registro do Browser QA por outro agente e proposta de
auditoria/soft-delete rastreada em card separado apenas se necessária. Owner/reviewer de
governança: General-Tech-Lead (Codex56-TL, w5:pC); `assignee` formal deixado em branco de
propósito, porque atribuir exigiria um identificador de assignee que eu não vou inventar e o
freeze me proíbe de mutar cards.

### 8.2 Prova de unicidade imediata

```console
$ GET /api/issues/fd5c4d55-8ce5-412f-9f15-db666831bfdb?workspace_id=…   → 200
uuid=fd5c4d55-8ce5-412f-9f15-db666831bfdb  number=38  status=todo
title=Kanban issue-number identifier collision (reporting-level)

$ GET /api/issues?workspace_id=…
total=28   rows=28   distinct_numbers=28
group_by(number) | select(length>1) = []        ← nenhuma duplicata
rows com number=38 → 1   (uuid fd5c4d55-8ce5-412f-9f15-db666831bfdb)
rows com number=31 → 1   (uuid f7e13350-c7f2-4335-8a04-01b98527d034,
                          "Security Wave A Containment", status in_review — inalterado)
```

Nenhum mismatch, portanto nenhum retry foi necessário e nenhum foi feito.

### 8.3 Limites respeitados

- **Um único** POST em toda a operação. Nenhum outro create.
- Browser QA **não** foi criado por mim (o ruling designa outro agente).
- Zero mutação em cards existentes: nenhum PATCH, DELETE, renumeração, merge, assignment ou
  mudança de status. ORQ-31 permanece exatamente como estava.
- Nenhuma escrita SQL direta, nenhum edit de código, build, teste, deploy, restart ou acesso
  a credencial.
