# P5 — Diligência: Vendor Capability Matrix

## Objetivo
Matriz de capability por provider a partir de fonte primária, com decisão explícita sobre OpenCode
(arquivado) e as capabilities `not_validated` (disabled-by-default).

## REQ-IDs
REQ-07 (matriz) · REQ-08 (decisão OpenCode). Base: `docs/vendors/owner-acceptance-request.md`.

## Pré-requisitos
- P0 verde.

## Passos
- 5.1 Matriz por vendor (Codex/Kiro/Antigravity/Cline/OpenCode): `launch_mode`, `auth_mode`,
  `quota_mode`, `rotation_mode`, `continuation_mode`, `smart_context_mode`, `reset_claim_mode`.
  Classificar `verified | inferred | not_validated` (fonte primária checada).
- 5.2 **DECISÃO OpenCode** (projeto ARQUIVADO, sucessor Crush): `disabled` / descopar / migrar p/ Crush.
  Documentar a escolha e a razão.
- 5.3 owner-acceptance dos `not_validated` → **disabled-by-default** até validação empírica.

## Fatos já levantados (sessão anterior)
- **Smart Context não é nativo de nenhum vendor** — existe só via prodex (fato arquitetural).
- `reset_claim` é Codex-specific, implementado pelo prodex (`redeem`), sem doc oficial linkável.
- 8 células `not_validated` + 2 borderline aguardando decisão do dono (accept-disabled recomendado; #7 Smart Context = gate; #6 OpenCode = arquivado).

## Verificação / evidência
- Links de fonte checados (200 vs 404) por célula.
- Decisão OpenCode registrada.

## Critério de GATE (DONE)
✅ matriz com fontes checadas · ✅ decisão OpenCode registrada · ✅ not_validated marcados disabled-by-default.

## Nota
Habilitar capability só `verified` ou explicitamente aceita pelo dono; resto **disabled**. Nunca ligar capability não-verificada.
