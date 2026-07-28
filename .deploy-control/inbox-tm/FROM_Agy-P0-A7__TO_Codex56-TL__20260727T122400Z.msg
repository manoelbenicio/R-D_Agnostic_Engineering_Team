[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Especificação Arquitetural V2 GTL-82 — Runtime Retention Lifecycle (gtl-runtime-retention-safe-lifecycle-v2.md)

1. EVIDÊNCIA DE DDL E CONSTRAINTS REALMENTE COMPROVADAS:
- Auditadas migrations 004, 032, 120 (UNIQUE workspace_id, daemon_id, provider e idx agent_runtime_workspace_daemon_profile_key).
- Confirmado que o SQL de Heartbeat/DaemonRegister não limpa archived_at; se não for verificado nas queries, deamons ignoram arquivamento e ressuscitam runtimes obsoletos.

2. ESPECIFICAÇÃO DE SCHEMA V2 E ÍNDICES PARCIAIS:
- Adição de archived_at TIMESTAMPTZ, archived_by UUID e archive_reason TEXT.
- Atualização das constraints de Uniqueness para índices parciais WHERE archived_at IS NULL, permitindo manter múltiplos registros históricos arquivados sem colisão no Postgres.

3. REGRAS DE NEGÓCIO E CONTRATOS DE API:
- Rejeição de Daemon: Heartbeat e Claim em runtime arquivado respondem HTTP 409 Conflict (code: "runtime_archived").
- Guard de Active-Task: reassign-and-archive exige active_tasks == 0; caso contrário responde 409 (code: "runtime_has_active_tasks").
- API Pública: DELETE /api/runtimes/{id} é convertido internamente em soft-archive. Deletes físicos restritos a scripts offline.
- Parâmetro Protegido: GET /api/runtimes?include_archived=true restrito a admin/billing para retenção histórica perpétua (task_usage 100% preservada).

4. BACKFILL E DOWN MIGRATION SEGURO:
- Reatribuição dos 8 agentes ativos nos 2 runtimes offline (kiro eb41a0c9 -> 6d0d721a; claude 588ebcba -> online/archived) com marcação de archived_at = NOW().
- DOWN Migration segura com remoção de linhas arquivadas antes do DROP COLUMN para evitar violações de UNIQUE constraint.

5. VEREDITO FINAL:
- STATUS: PASS (ESPECIFICAÇÃO V2 APROVADA).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-runtime-retention-safe-lifecycle-v2.md.
Operação 100% Read-Only. Zero mutações.
