# 99_arquivados — Histórico do projeto (consumido, mantido)

> **Tudo aqui já foi utilizado.** É histórico do projeto, preservado para rastreabilidade.
> Não é a fonte de verdade ativa — a fase atual vive em `docs/rotation-parity-polyglot/`.

## Conteúdo

### `prod-readiness/` — planejamento anterior (fase Go, superseded)
- `PROJECT_PLAN.md` — plano consolidado da fase Go (rotação + auth + observability).
- `MASTER_PROD_READINESS.md` — master de prod-readiness (blocos B/H/O/F).
- `PARALLELIZATION_PLAN_PROD.md`, `PARALLELIZATION_PLAN.md` — planos de paralelização.
- `MASTER_AGENTIC_PLAN.md` — plano agêntico anterior.
- `STATUS.md` — status verificado da fase Go.
- `REALTIME_E2E_RUNBOOK.md`, `STAGING_DEPLOY_RUNBOOK.md` — runbooks da fase Go.

> A maior parte destes itens foi entregue (B1–B4, H1/H6/H7, O3) ou **superseded** pela
> arquitetura polyglot. O mapeamento item-a-item (o que virou runtime do prodex vs o que
> permanece control-plane Go) está em `docs/rotation-parity-polyglot/03_PLATFORM_PLAN_360.md` §2.

### `deploy-control-checkins/` — check-ins de streams concluídos
Registros de execução (sign-in/out) das fases Go anteriores (W-*, PR-*, RR-*, STG-*, RT-*).
Histórico de quem fez o quê e quando.

### `deploy-control-prompts/` — prompts de agente já consumidos (board antigo)
Prompts das waves anteriores que rodaram no board `.deploy-control/`.

## Observação
O "vendor bible" (`docs/project/BACKLOG-detection.md`) e as specs `docs/project/00..07`
**permanecem ativos** (referência viva), por isso NÃO foram movidos para cá.