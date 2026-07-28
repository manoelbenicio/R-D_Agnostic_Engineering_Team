# GTL-K05 — Criação do card único de Browser QA (registrar designado)

Registrar: Codex56#A (`w7:p3`) · UTC 2026-07-27T12:55Z · autorização: GTL-K05 (um card, uma vez)

## 1. Pré-checagem de duplicata (antes do POST)

```text
GET /api/issues?workspace_id=20fce817-895d-447b-965a-49f5e279314a
TOTAL 28 · MAX number 38
title contém browser|playwright|qa|supply -> HITS 0
```

Nenhum card de Browser QA existia. Também confirmei, por UUID, que os alvos do K03 seguem estáveis:
`8b1419f5…`→ORQ-12, `2aefcb3d…`→ORQ-13, `19ab8cfe…`→ORQ-14, `6230b5c0…`→ORQ-23 (todos OK).

## 2. Assignee: UUID exato descoberto, não inventado

```text
GET /api/agents?workspace_id=...  (10 agentes)
Opus48-B = 4069a041-9c68-416a-b0cf-52226c076c6c
```

Só atribuí porque o UUID é descobrível e exato. Nome no board é `Opus48-B`.

## 3. POST — exatamente uma vez

```text
POST /api/issues?workspace_id=20fce817-895d-447b-965a-49f5e279314a   -> HTTP=201
id         = c03941bc-3bde-4de1-ab19-1ba93de0ad51
identifier = ORQ-39
number     = 39
status     = todo   priority = high
assignee   = agent 4069a041-9c68-416a-b0cf-52226c076c6c (Opus48-B)
```

Nenhum número foi previsto antes do 201. Nenhum segundo POST foi emitido.

## 4. GET imediato por UUID — confirmação do par UUID+number

Primeira tentativa (`GET /api/issues/<uuid>` **sem** `workspace_id`) devolveu corpo com campos vazios.
**Não** retentei o POST; repeti apenas a leitura, com `workspace_id`, e confirmei:

```text
GET /api/issues/c03941bc-3bde-4de1-ab19-1ba93de0ad51?workspace_id=...
{"id":"c03941bc-3bde-4de1-ab19-1ba93de0ad51","workspace_id":"20fce817-...","number":39,
 "identifier":"ORQ-39","title":"Ephemeral Browser QA & Playwright Supply-Chain Pipeline", ...}
```

Observação de API para o incidente ORQ-38: **`GET /api/issues/{id}` só devolve o objeto quando
`workspace_id` é enviado**; sem ele, retorna corpo com campos nulos e não um 404. Um verificador que
use a forma sem `workspace_id` conclui erradamente que o card não existe — é exatamente o tipo de
leitura que produz relato de identidade divergente.

## 5. Prova de unicidade no workspace (pós-criação)

```text
TOTAL 29 · MAX 39
DUPLICATE numbers:     {}      (zero)
DUPLICATE identifiers: {}      (zero)
MATCH_BY_UUID: [('ORQ-39', 39, 'Ephemeral Browser QA & Playwright Supply-Chain Pip', '4069a041-…')]
title browser|playwright|supply: [('ORQ-39','c03941bc')]   -> 1 único card, zero duplicata
```

## 6. Conteúdo registrado no card

Invariante **NO PRODUCTION DATA**; ativação do workflow da raiz como **gate de governança**
(STOP-AND-WAIT do owner); **pins imutáveis** por SHA/digest; stack **efêmera** pgvector+backend+frontend;
**7 specs existentes + 6 testes de browser** (clipe/upload, drag-and-drop, feedback, avatar, comentário,
quick-create); **teto de custo/tempo** com abort; **cleanup** verificado; **6 critérios de aceite**;
dependências (ORQ-26 `e966922d-…`, ruling de CI da raiz, fila zero); **estado atual BLOCK até peer
review V2 PASS**; riscos e rollback.

Vínculo com ORQ-26 feito **por referência na descrição**, não por `parent_issue_id`: alterar hierarquia
não foi autorizado e teria efeito no board além do card criado.

## 7. Não-alegações

- Um único `POST`; nenhum retry; nenhum outro card criado. **Card de CLI não criado.**
- `ORQ-31` e `ORQ-38` não tocados; nenhum status/descrição de card preexistente alterado.
- Nenhuma evidência anexada a card (o K05 proíbe por enquanto); nenhum arquivo de repo editado.
- Não previ ORQ-N: o número saiu do 201 e foi reconfirmado por leitura.
- Não afirmo que o pipeline exista ou passe: o card nasce `todo` e **BLOCK** até o peer V2 PASS.
