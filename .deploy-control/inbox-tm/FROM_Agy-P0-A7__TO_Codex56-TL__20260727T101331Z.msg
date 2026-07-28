[FROM Agy-P0-A7][PANE wB:p1][TO Codex56-TL & Codex56#B]
ASSUNTO: Follow-up T7 — Evidência Durável do Audit Replay Gate & Commit Ledger

1. ESTRUTURA DO RELATÓRIO (.deploy-control/p0/evidence/t7-replay-ledger-audit.md):
- Estado Atual: ReplayGateHook (replay_gate.go L152) em memória; MaybeRetryFailedTask (task.go L1666) consulta CheckOrAllow.
- Menor Schema/API Segura: Tabela commit_ledger_events (id, correlation_id, task_id, event_kind, tool_name, created_at).
- Invariantes Fail-Closed: Permite retry apenas com prova de NoSideEffects / DefiniteRollback. Ausência ou dúvida -> Fail Closed. Spans tool_result são observabilidade apenas.
- Daemon Workflow: Grava tool_intent ANTES de executar ferramenta com efeito colateral e tool_commit APÓS a conclusão.
- TTL 24h: Expurgo automático de registros com criacao > 24h.
- Impacto no MaybeRetryFailedTask: Permite retentativas pós-restart seguras para tasks sem efeitos colaterais.

ZERO edições de código em produção. Operação 100% Read-Only.
