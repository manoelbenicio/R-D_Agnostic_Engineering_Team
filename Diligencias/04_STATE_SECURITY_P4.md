# P4 — Diligência: State / Security

## Objetivo
Garantir estado compartilhado em Postgres (sem SQLite), migrations reversíveis, redaction de segredos e
taxonomia de auditoria.

## REQ-IDs
REQ-10 (Postgres/secrets boundary) · REQ-11 (redaction testada) · REQ-12 (audit taxonomy).
Spec: `specs/qa-conformance` (evidence-gated) + invariantes do PROJECT.

## Pré-requisitos
- P0 verde (Postgres/Redis alcançáveis).

## Passos
- 4.1 Backend Postgres/Redis para gateway/ledger/approved-accounts. **SQLite proibido** para estado compartilhado.
- 4.2 Migrations **reversíveis** (up/down) versionadas; testar rollback de migration.
- 4.3 Redaction policy aplicada a logs/traces/errors/audit.
- 4.4 Taxonomia de audit: account selection, redeem attempt, fallback, continuation binding, context-rewrite decision.

## Verificação / evidência
- Smoke no-SQLite: nenhum uso de SQLite para estado compartilhado (grep + teste).
- Migration up→down→up reversível testada.
- Redaction smoke: injeta token/JWT/`ghp_…` fake e confirma **mascarado** em log/trace/audit.
  - (nota: hits conhecidos são stubs de teste — JWT do jwt.io, `ghp_ABCDEF…` exemplo — não são tokens vivos.)
- 5 audit events emitidos e verificados.

## Critério de GATE (DONE)
✅ no-SQLite verificado · ✅ migration reversível testada · ✅ redaction smoke verde com evidência · ✅ audit taxonomy presente.

## Riscos
- Vazamento de segredo em evidência — scrubbing obrigatório antes de commitar qualquer log.
