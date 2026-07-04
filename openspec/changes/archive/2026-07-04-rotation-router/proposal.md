# Proposal — Rotation Router (policy-driven, self-hosted)

> **⚠️ SUPERSEDED (2026-07-04) por `openspec/changes/rotation-parity-polyglot/` + `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`.**
> A **autoridade de runtime** deste router (seleção/rotação/fallback/loadbalance/proactive_reset em voo)
> foi **absorvida pelo `prodex`/Rust L2** (arquitetura polyglot; ver ADR-001). **Permanece válido como
> CONTROL PLANE Go:** Account Registry, approved-accounts por tenant (migration 124), observability e
> KPI Savings. O código de seleção runtime em Go (policy/fallback/loadbalance/proactive_reset) fica como
> referência/legado; a decisão de request em voo passa ao L2. Nada aqui deve ser tratado como runtime ativo.


## Why
Nossa rotação de contas hoje é **ingênua**: drena a conta prioridade-1 até esgotar, depois
vai pra 2. Falta a mecânica madura de roteamento (fallback com retry/backoff, seleção por
política, consistência de sessão, observabilidade por dimensão, governança).

Essa mecânica **já existe pronta e validada em produção** no Requesty (gateway de LLM que
faz fallback / load-balancing / latency routing / policies nomeadas / spend console). A
decisão do dono: **NÃO assinar/pagar o Requesty** (seria custo metered novo, contra o moat).
Em vez disso, **copiar a metodologia** (design público + console próprio como referência) e
implementá-la **self-hosted na nossa camada de ASSINATURA**, onde o Requesty estruturalmente
não atua (ele é metered; não conhece janela 5h/créditos/resets).

## What Changes
- Introduzir um **Rotation Router policy-driven**: seleção de conta guiada por uma
  `RotationPolicy` nomeada (fallback / load-balancing / latency), com retry+backoff.
- **Taxonomia de policies por tipo de trabalho** (general / heavy / cheap / review).
- **Account Registry** com metadados + governança (approved accounts por tenant).
- **Observabilidade por dimensão** (custo/volume/tokens/latência × conta/vendor/task/
  workspace) + KPI **Savings** ($ economizado vs metered — quantifica o moat).
- **Melhorias exclusivas** (teto acima do Requesty): rotação **proativa** (antes da falha),
  **claim-de-reset** antes de rotacionar, **load-balance por saúde de janela**.

## Scope
- REUSA a fundação já construída: `rotation/*` (detector, service, pool, store_pg,
  auth_authenticator, proactive, usage, warnbanner, probe_codex, token_lifecycle), execenv,
  credential_metrics, observability stack.
- NÃO adiciona dependência/custo de terceiro. Requesty = referência de design, não vendor.
- NÃO altera o contrato existente (`contract.go`) sem necessidade; estende.

## Non-Goals
- Não vira gateway metered. Não roteia produção por Requesty. Não paga por token.
- Não substitui Multica (orquestração L4) — o router vive na camada de conta (L2).

## Impact
- Rotação deixa de ser priority-drain e passa a policy-driven, configurável sem redeploy.
- Base fundamentada em prática de mercado (Requesty), com teto proativo que a supera.
