[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Peer Review ORQ-19 — Auditoria de Prontidão da Retenção V2 (orq19-retention-v2-readiness-peer-review.md)

1. VALIDAÇÃO DOS 16 FILES_LOCKED:
- Inventário de 16 arquivos bloqueados auditado e confirmado sem sobreposição com file.go/file_test.go ou router.go.

2. ANÁLISE DE SCHEMA E ÍNDICES PARCIAIS:
- DDL com colunas de soft-archive (archived_at, archived_by, archive_reason) e índices únicos parciais WHERE archived_at IS NULL aprovado para evitar colisões no Postgres.

3. DESAFIO DA SEMÂNTICA HTTP 409 NO HEARTBEAT:
- Confirmado que HTTP 409 Conflict (code: "runtime_archived") é a resposta correta para indicar conflito de estado de ciclo de vida e interromper pings zumbis de daemons.

4. PRESERVAÇÃO DE DATOS E ROLLBACK:
- 100% dos dados de task_usage e histórico de billing preservados.
- Backfill dos 8 agentes dos runtimes offline (kiro eb41a0c9 e claude 3fa85f64) validado.
- DOWN migration com expurgo prévio de arquivados garante rollback limpo sem falha de UNIQUE constraint.

5. VEREDITO FINAL:
- STATUS: PASS (PRONTIDÃO DA ORQ-19 APROVADA PARA EXECUÇÃO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq19-retention-v2-readiness-peer-review.md.
Operação 100% Read-Only.
