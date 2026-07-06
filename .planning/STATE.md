# STATE — Milestone v2.0 (Rotation-Parity Polyglot)

> Estado vivo do milestone. Atualizado a cada avanço de fase.

## Posição atual
- **Milestone:** v2.0 "Fundação + Deploy Correto" — planejamento **CONCLUÍDO**, PLANs GSD gerados para TODAS as fases (2026-07-04).
- **Fases planejadas:** 15 PLAN.md files em 11 fases (P0–P9 + Meta), formato GSD completo.
- **Próxima fase:** **P0 — Fundação** (bloqueia tudo). PLANs prontos, execução NÃO iniciada.
- **Config GSD:** mode=yolo, profile=quality, agents=8, granularity=standard.

## Blocker crítico (raiz)
- **prodex BINÁRIO não existe** — source presente (`/tmp/prodex-audit-7750da9` @7750da9b) mas **não buildado** (Rust/cargo ausente). → P0/REQ-01. **Nada de deploy até isso.**

## Já pronto (verificado, reaproveitável)
- Multica Go server + integração `prodex.go`/`l2runtime` (código existe).
- Isolamento por conta no produto (execenv) — intacto.
- Postgres/Redis (docker) up. docker v29.
- Contrato/vendor-matrix/redaction-audit produzidos em sessão anterior (revalidar como evidência sob os novos REQs).

## Pendências de processo
- ~~`rotation-parity-polyglot` (OpenSpec) tinha 0 tasks e furos~~ → substituído por planejamento GSD documentado. ✅
- ~~Arquivar `rotation-router` (SUPERSEDED). [REQ-24]~~ → **DONE** (2026-07-04T23:48Z) — `openspec/changes/rotation-router/status.md` criado com status SUPERSEDED, referenciando ADR-001 e successor `rotation-parity-polyglot`. ✅
- ~~Reconciliar "deploy direto × QA exaustivo". [REQ-25]~~ → **RESOLVIDO** (ver abaixo). ✅

## Reconciliação: Deploy Direto × QA Exaustivo [REQ-25]

> **Não há contradição.** A sequência é:
>
> 1. **P6 — QA exaustivo EM CONTAINER** (não em PROD): C1–C6 por capability + replay + fail-closed + Smart Context shadow→canary→live + tripla CODEX_HOME×prodex×Herdr. **Tudo com evidência verde em container controlado.**
> 2. **P7 — Deploy direto em PROD** (sem staging/canary separado): kill-switch testado + rollback 1-cmd testado + logs scrubbed → **deploy em PROD apenas DEPOIS do QA verde.**
>
> **Decisão do owner:** "Sem fase de staging dedicada — ajusta-se em PROD." Isso significa que o QA é feito em container (P6), não que o QA é bypassado. O deploy direto é APÓS o QA, não em vez do QA.
>
> **Guard-rails em PROD:** Smart Context em shadow/canary configurável + kill switch + logs scrubbed + rollback documentado (nativos do prodex).

## Próximo passo
Iniciar **P0 (Fundação)** — provisionar/buildar o binário prodex e confirmar ambiente. Só então P1→P7.

**Wave 1 em andamento (2026-07-04):** P4 (state-security), P5 (vendor-matrix), P10 (meta) em execução paralela.

## Atualização 2026-07-04T21:45 — Correção de 11 gaps (docs vs GSD)
Comparação exaustiva entre docs explore/propose (23 docs em `docs/`, `openspec/`, `Diligencias/`) e o GSD identificou 11 gaps. TODOS corrigidos:
- **GAP-01** (Early Rotation warnbanner) → task 3.6 + REQ-40
- **GAP-02** (Observability stack) → tasks 7.4a–7.4e
- **GAP-03** (Smoke S1-S5) → tasks 6.0a–6.0e
- **GAP-04** (11 replay scenarios) → task 6.2 expandida
- **GAP-05** (L2 Contract detail) → task 1.1 com ref `docs/contracts/l2-runtime-contract.md`
- **GAP-06** (4 tabelas Postgres) → task 4.1 com migration 123
- **GAP-07** (Runbook 12 steps) → task 7.4 com ref `docs/deploy/prod-rollout-runbook.md`
- **GAP-08** (Schema JSON existente) → task 1.2 com ref `docs/contracts/runtime-events.schema.json`
- **GAP-09** (Herança B5/H2-H5/O1-O4) → tabela de mapeamento em tasks.md + REQUIREMENTS.md
- **GAP-10** (Gates G1-G10) → tabela de mapeamento abaixo
- **GAP-11** (POSIX FS check) → task 4.11 + REQ-41

**Total de tasks atualizado:** 80 (era 66; +14 novas). **REQs:** 41 (era 39; +REQ-40, +REQ-41).

## Mapeamento gates executivos (STATUS_EXECUTIVO → tasks GSD)

| Gate | Descrição | Task(s) GSD |
|------|-----------|-------------|
| G1 | Tripla CODEX_HOME × prodex × Herdr | 6.6 |
| G2 | Coordenação Herdr smoke | 6.7 |
| G3 | Roteador único provado | 1.3, 3.4 |
| G4 | Troca perfil fail-closed | 6.4 |
| G5 | Smart Context shadow→canary→live | 6.5 |
| G6 | Reset-claim matriz empírica | 9.1, 9.3 |
| G7 | Conformance por capability | 6.1 |
| G8 | Secrets redaction test | 4.3, 7.3 |
| G9 | Postgres/Redis sem SQLite + migrations | 4.1, 4.2 |
| G10 | Container verde + killswitch + rollback | 7.1, 7.2 |

## QA verificado (2026-07-04, container IPv4)
- BUILD: verde · VET: verde · TEST internal: **24/24 pacotes OK, 0 FAIL** (execenv 77.5%, metrics 64.3%, daemon 68.3%, rotation 64.7%, l2runtime 63.1%, events 100%...).
- Nota: falha anterior de 2 pacotes era rede IPv6, não código → resolvido com `--sysctl net.ipv6.conf.all.disable_ipv6=1`.
- Dashboard plan_dashboard: QA 49/49 (base 29 + SEV-0 20). Encoding-safe.
- Blockers SEV-0 abertos (produto): ISSUE-001 binário prodex (P0), ISSUE-005 91 uncommitted, ISSUE-006 gates QA sem evidência empírica.

## Registro de erros (RCA)
Todas as cagadas/furos desta sessão estão documentados em **`.planning/RCA-2026-07-04-001-orchestrator-errors.md`** (22 erros ERR-01..22 + causa-raiz sistêmica + controles de prevenção).
