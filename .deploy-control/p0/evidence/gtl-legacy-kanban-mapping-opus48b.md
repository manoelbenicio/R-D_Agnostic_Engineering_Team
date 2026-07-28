# GTL-K03 — Mapeamento retroativo de Kanban para entregáveis legados (Opus48#B)

Executor: Codex56#A (`w7:p3`) · UTC 2026-07-27T12:38Z
Modo: **READ-ONLY sobre o board.** Nenhum `POST`, `PUT`, `PATCH`, delete, renumeração ou merge foi
emitido. Motivo: o freeze **GTL KANBAN FREEZE — DUPLICATE ORQ-31 INCIDENT** chegou durante esta tarefa
e proíbe criação de issue até a auditoria de unicidade; adotei também a postura conservadora de não
reescrever `description` de card durante um incidente de integridade, porque `PUT` substitui o campo
enviado e não há undo. Os payloads exatos ficam prontos abaixo para aplicação após o unfreeze.

## 1. Inventário do board no corte (medido, não relatado)

`GET /api/issues?workspace_id=20fce817-895d-447b-965a-49f5e279314a` (via ORQ1 loopback), duas leituras
(12:36:5xZ e 12:37:3xZ), resultado idêntico:

```text
TOTAL = 20 issues · MAX number = 30
ORQ-11 in_review | ORQ-12 todo | ORQ-13 todo | ORQ-14 blocked | ORQ-15 todo | ORQ-16 in_review
ORQ-17 todo | ORQ-18 todo | ORQ-19 blocked | ORQ-20 blocked | ORQ-21 todo | ORQ-22 done
ORQ-23 todo | ORQ-24 done | ORQ-25 done | ORQ-26 in_progress | ORQ-27..29 done | ORQ-30 in_review
match "browser"/"qa" no título: []      match "cli" no título: []
```

**Achado que alimenta a auditoria do incidente:** no corte não existia **nenhum** ORQ-31 e nenhum card
de Browser QA ou de CLI. A premissa do dispatch K03 ("Browser QA is already ORQ-31") **já era falsa às
12:36–12:37Z** neste workspace. Se dois UUIDs distintos foram depois reportados como ORQ-31, a colisão
é posterior a esse corte e compatível com **duas criações concorrentes** consumindo o mesmo `number`
— o que é exatamente o motivo pelo qual eu **não** criei o card de CLI (seria a terceira).

UUIDs dos cards-alvo (necessários para reuso "sem ambiguidade" exigido pelo freeze):

| ORQ | UUID | status | prioridade | assignee_type/id | desc_len |
|---|---|---|---|---|---|
| ORQ-12 | `8b1419f5-c9d4-466c-adff-cded98f91d29` | todo | urgent | agent / `2c042fdd-9a76-4da1-9b02-c2332a736a86` | 490 |
| ORQ-13 | `2aefcb3d-97b3-4e70-a4b0-ae3729b7981d` | todo | high | agent / `4dc3b1f5-451f-4f68-b71e-91e856fed286` | 500 |
| ORQ-14 | `19ab8cfe-0a95-4a79-af99-4ee384326021` | blocked | urgent | agent / `3db514db-810e-4393-817e-eb3707ce59ae` | 447 |
| ORQ-23 | `6230b5c0-c57a-4b88-a7df-84899057d0c8` | todo | high | agent / `4dc3b1f5-451f-4f68-b71e-91e856fed286` | 575 |

Todos os quatro **já têm assignee**; nenhum `PUT` de assignment é necessário para eles.

## 2. Mapeamento exato dos seis entregáveis

Regra aplicada: **um card por problema distinto**, não por arquivo de evidência; reuso só quando o
escopo de aceite do card cobre o problema; **nenhum card marcado como done a partir de evidência de
design**.

| # | Entregável (GTL) | Arquivo de evidência | Veredito atual | ORQ-N | Decisão |
|---|---|---|---|---|---|
| 1 | GTL-03 — auditoria de implementação de custo | `gtl-cost-implementation-audit.md` (11:30) + base `cost-accounting-design.md` | auditoria concluída; design confrontado com HEAD (achados por afirmação) | **ORQ-12** | **REUSO** — o card é "Contabilizar custo por conta em `task_usage`", exatamente o objeto auditado. Sem novo card |
| 2 | GTL-17 — observabilidade do schema de usage | `gtl-usage-schema-observability.md` + peer review `gtl-usage-observability-peer-review.md` (GTL-26, 11:36) | **PASS (aprovado com retificação de premissa stale)** — é sonda/design, não implementação | **ORQ-14** | **REUSO** — card "Contabilizar tokens reais em AGY, Codex e Kiro", hoje `blocked`. Anexar como design aprovado; **não** mover status |
| 3 | GTL-25 — design de preço versionado por provider/model/tier | `gtl-versioned-tier-pricing-design.md` (11:38) + `gtl-versioned-pricing-peer-review.md` | DESIGN; nenhum valor monetário proposto (preço é decisão do owner) | **ORQ-13** | **REUSO** — card "Calcular preço por tier de reasoning". Design ≠ implementação |
| 4 | GTL-37 — peer review do cutover ORQ-23 | `gtl-orq23-cutover-peer-review.md` (11:44) + `gtl-orq23-durable-cutover-plan.md` + `gtl-orq23-cutover-v2-peer-review.md` | **BLOCK** | **ORQ-23** | **REUSO** — card "Concluir contabilização da Fase 3 e gate de rollback 4.5". O BLOCK deve constar como bloqueio do próprio card |
| 5 | GTL-I03 — ORQ-13 fase 1 (`thinking_level` em `task_usage` + wiring) | `gtl-i03-orq13-phase1-implementation.md` (12:03) + peer review `gtl-orq13-phase1-peer-review.md` (GTL-68, 12:05) + `gtl-orq13-salvage-map.md` | **PASS de código e testes unitários com SPLIT PLAN, condicional a 3 critérios** (ruling Z01 de número, baseline sqlc separado, prova em Postgres real) | **ORQ-13** (mesmo card do item 3, como fase 1) | **REUSO, sem card novo** — é pré-requisito do mesmo problema (preço por tier exige tier persistido). **Não marcar done**: patch vive em worktree, sem commit/migration aplicada |
| 6 | GTL-74 — peer review adversarial do alinhamento de CLI V2 | `gtl-cli-alignment-v2-peer-review.md` (GTL-74, 12:20) sobre `gtl-cli-version-alignment-v2.md` (GTL-71, 12:09), com base GTL-50 e GTL-44 | plano V2 + review adversarial; operação de frota não executada | **NENHUM CARD EXISTENTE** | **CRIAÇÃO PENDENTE — BLOQUEADA PELO FREEZE.** Card proposto pronto em §3; não emiti `POST` |

Resumo: **5 reusos (ORQ-12, ORQ-13 ×2 fases, ORQ-14, ORQ-23) + 1 card pendente de criação**. Zero
duplicação criada. **ORQ-31 não tocado** (e, no meu corte, inexistente).

## 3. Card proposto (único) — a criar após o unfreeze

```text
title:       Alinhar versoes de CLI da frota (codex/claude) entre ORQ1 e ORQ2
priority:    high
status:      todo            # nunca "done": nada foi instalado
assignee:    Codex56-TL (GENERAL-TECH-LEAD) como integrador; owner autoriza a janela
evidence:    .deploy-control/p0/evidence/gtl-cli-version-alignment-v2.md            (GTL-71, plano V2)
             .deploy-control/p0/evidence/gtl-cli-alignment-v2-peer-review.md        (GTL-74, review adversarial)
             .deploy-control/p0/evidence/gtl-cli-alignment-second-peer-review.md    (GTL-50, medicoes)
             .deploy-control/p0/evidence/gtl-official-release-notes-watch.md        (GTL-44, fontes oficiais)
problema:    frota heterogenea — ORQ2 codex-cli 0.145.0 + claude 2.1.215; ORQ1 codex-cli 0.144.6 +
             claude 2.1.218; ultima oficial codex 0.145.0 (2026-07-21) e claude 2.1.220
impacto:     agentes que deveriam ser equivalentes divergem; ORQ2 sem as correcoes de 2.1.218
             (hooks de frontmatter exigindo workspace trust e metering de gateway)
aceite:      (1) versao-alvo fixada por CLI e registrada; (2) ORQ1 e ORQ2 na mesma versao com
             `--version` como evidencia; (3) CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH fixado antes de
             2.1.219+; (4) exec-policy e fork de thread testados antes de subir codex no ORQ1;
             (5) canary com rollback exato por versao
dependencias: fila zero (`agent_task_queue` sem queued/dispatched/running/waiting_local_directory);
             janela autorizada pelo owner (instalacao/upgrade e STOP-AND-WAIT)
riscos/rollback: upgrade simultaneo derruba a frota; rollback e pin de versao anterior por CLI
```

## 4. Payloads `PUT` prontos (não aplicados)

Semântica verificada em `server/internal/handler/issue.go:2308-2352`: `UpdateIssue` lê o body cru e
**só altera campo explicitamente presente** (`if req.X != nil`), pré-preenchendo nullable com o valor
atual. Portanto anexar evidência é seguro **desde que** `description` enviada seja
*descrição atual + bloco anexado* — por isso o passo de apply exige um `GET` imediatamente antes.

```text
PUT /api/issues/8b1419f5-c9d4-466c-adff-cded98f91d29   # ORQ-12
  {"description": "<atual>\n\n## Evidência retroativa (GTL-K03)\n- GTL-03 auditoria de custo: .deploy-control/p0/evidence/gtl-cost-implementation-audit.md\n- base: cost-accounting-design.md\n- veredito: auditoria concluída, implementação NÃO feita"}
PUT /api/issues/2aefcb3d-97b3-4e70-a4b0-ae3729b7981d   # ORQ-13 (fases 1 e preço)
  {"description": "<atual>\n\n## Evidência retroativa (GTL-K03)\n- fase 1 (thinking_level): gtl-i03-orq13-phase1-implementation.md; peer review GTL-68 = PASS condicional a 3 critérios\n- design de preço: gtl-versioned-tier-pricing-design.md (nenhum valor monetário)\n- pendências: ruling de número de migration, baseline sqlc separado, prova em Postgres real"}
PUT /api/issues/19ab8cfe-0a95-4a79-af99-4ee384326021   # ORQ-14
  {"description": "<atual>\n\n## Evidência retroativa (GTL-K03)\n- GTL-17 observabilidade de usage + peer review GTL-26 = PASS com retificação de premissa stale\n- natureza: sonda/design; medição real de tokens ainda não executada"}
PUT /api/issues/6230b5c0-c57a-4b88-a7df-84899057d0c8   # ORQ-23
  {"description": "<atual>\n\n## Evidência retroativa (GTL-K03)\n- GTL-37 peer review do cutover = BLOCK; gtl-orq23-durable-cutover-plan.md; gtl-orq23-cutover-v2-peer-review.md\n- bloqueio: rollback T2 de um comando não ensaiado"}
```

Nenhum `status`, `priority` ou `assignee_*` é enviado: os quatro já têm assignee e **nada é marcado
done**. Os payloads são idempotentes se o bloco `## Evidência retroativa (GTL-K03)` for verificado
antes de anexar.

## 5. Wave A de segurança e o mapeamento K02

A evidência da Wave A (`gtl-security-remediation-wave-a-execution.md`) **não foi alterada** por esta
tarefa. Não localizei no board card de segurança/Wave A no meu corte (`match "browser"/"qa"` e
`"cli"` vazios; sem ORQ-31), portanto **não posso citar um K02 mapping confirmado**. Registro a
pendência: quando a auditoria de unicidade definir qual UUID é Browser QA e qual é Security Wave A, a
Wave A passa a citar o ORQ-N correto por adendo próprio, não por edição retroativa do arquivo.

## 6. Não-alegações

- Nenhuma mutação de board: zero `POST`/`PUT`/`PATCH`/delete/renumeração/merge. `ORQ-31` não tocado.
- Nenhuma mutação de repo, código, teste, build, instalação, deploy, restart, credencial ou permissão
  nesta tarefa.
- Não marquei nada como done e não movi status: os vereditos anexados são de design/auditoria.
- Não afirmo que ORQ-31 não exista **agora**: afirmo que às 12:36–12:37Z, em duas leituras, o máximo
  era 30 e não havia card de Browser QA, Security ou CLI neste workspace.
- Não li valor de segredo; nenhuma evidência citada expõe valor.
- Não inspecionei o histórico de comentários dos cards (apenas `/api/issues`): se houver menção às
  evidências em comentários, o reuso continua válido, mas a checagem de duplicidade por comentário
  fica pendente para o agente de busca designado.
