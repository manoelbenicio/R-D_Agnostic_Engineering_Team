# P2 — Diligência: prodex fork-map / invariantes

## Objetivo
Mapear a arquitetura do prodex (crates) e isolar as fronteiras do futuro fork, preservando os
invariantes de runtime. Fase de **análise** (alvo do marco de fork; não bloqueia F0).

## REQ-IDs
REQ-09 (Smart Context via prodex; fork-map). Spec: `specs/prodex-runtime-provisioning` (contexto de source).

## Pré-requisitos
- P0 verde (source estável).

## Passos
- 2.1 Mapear crates do workspace (`/tmp/prodex-audit-7750da9/crates/`), ex.: `prodex-core`,
  `prodex-context`, `prodex-provider-core`, `prodex-runtime-anthropic|gemini`, `prodex-presidio`,
  `prodex-shared-types`, `prodex-runtime-doctor`, `prodex-terminal-ui`.
- 2.2 Isolar runtime proxy/gateway/Smart Context/state/redeem; propor **fork boundary**.
- 2.3 Documentar invariantes a preservar:
  - **hard affinity** (`previous_response_id`, turn-state, `session_id`);
  - **rotate-before-commit** (nunca rotacionar mid-stream após início do output);
  - **profile auth isolation** (`$PRODEX_HOME/profiles/<name>`).

## Verificação / evidência
- `docs/prodex/prodex-fork-map.md` com crates mapeados e invariantes rastreados.
- Gaps de SBOM/conformance anotados (reconhecidos nos docs do prodex).

## Critério de GATE (DONE)
✅ fork-map revisado · ✅ invariantes rastreados aos crates · ✅ gaps listados para o marco de fork.

## Nota
Conformance de provider é parcialmente split no prodex (reconhecido) → promover contratos ao core **no fork** (marco futuro), não agora.
